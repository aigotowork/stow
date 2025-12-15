# Core 模块设计

## 职责定位

**Core 模块是 JSONL 数据层的核心**，负责：
1. JSONL 格式的编码/解码
2. 元数据（Meta）管理
3. 记录（Record）结构定义
4. 文件读写操作

## 核心数据结构

### Meta - 元数据

```go
type Meta struct {
    Key       string    `json:"k"`      // 原始 Key
    Version   int       `json:"v"`      // 版本号（从 1 开始）
    Operation string    `json:"op"`     // "put" | "delete"
    Timestamp time.Time `json:"ts"`     // 时间戳
}
```

**设计要点：**
1. **字段缩写** - 使用单字符字段名（`k`, `v`, `op`, `ts`）减少存储空间
2. **版本号** - 单调递增，用于历史追踪
3. **操作类型** - 支持 Put 和 Delete，实现软删除
4. **时间戳** - RFC3339 格式，便于人工阅读

### Record - 记录

```go
type Record struct {
    Meta *Meta                  `json:"_meta"`  // 元数据
    Data map[string]interface{} `json:"data"`   // 用户数据
}
```

**JSONL 格式示例：**
```json
{"_meta":{"k":"server","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{"host":"localhost","port":8080}}
{"_meta":{"k":"server","v":2,"op":"put","ts":"2024-01-01T01:00:00Z"},"data":{"host":"0.0.0.0","port":8080}}
{"_meta":{"k":"server","v":3,"op":"delete","ts":"2024-01-01T02:00:00Z"},"data":null}
```

**为什么用 `_meta` 前缀？**
- 避免与用户数据字段冲突
- 视觉区分系统字段和用户字段
- 便于工具解析（grep `_meta`）

## 编码器（Encoder）

### 职责

将 Record 序列化为 JSONL 格式。

### 实现

```go
func (e *Encoder) Encode(record *Record) ([]byte, error) {
    // 1. JSON 序列化
    data, err := json.Marshal(record)

    // 2. 添加换行符（JSONL 要求）
    data = append(data, '\n')

    return data, nil
}
```

**关键点：**
- ✅ 紧凑格式（不使用缩进）
- ✅ 每行必须以 `\n` 结尾
- ✅ 无 BOM（避免兼容性问题）

### 追加写入

```go
func AppendRecord(filePath string, record *Record) error {
    // 1. 打开文件（追加模式）
    f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

    // 2. 编码
    data, err := encoder.Encode(record)

    // 3. 写入
    _, err = f.Write(data)

    // 4. Sync（确保落盘）
    err = f.Sync()

    return nil
}
```

**为什么需要 Sync？**
- 确保数据真正写入磁盘
- 避免断电丢失数据
- 权衡：性能下降 ~10x，但数据安全

## 解码器（Decoder）

### 核心功能

1. **逐行解码** - 支持大文件
2. **容错处理** - 跳过格式错误的行
3. **反向读取** - 实现 Last Write Wins

### 读取所有记录

```go
func (d *Decoder) ReadAll(filePath string) ([]*Record, error) {
    f, _ := os.Open(filePath)
    scanner := bufio.NewScanner(f)

    var records []*Record
    for scanner.Scan() {
        record, err := d.Decode(scanner.Bytes())
        if err != nil {
            // 容错：跳过错误行
            continue
        }
        records = append(records, record)
    }

    return records, nil
}
```

**容错设计：**
```
正常行：{"_meta":{...}, "data":{...}}  ✅ 解析成功
损坏行：{"_meta":{..., "data":}        ❌ 跳过，继续
空行：                                 ❌ 跳过
注释：# This is a comment             ❌ 跳过
```

### 反向读取最后有效记录

**最关键的方法**，实现 Last Write Wins 语义：

```go
func (d *Decoder) ReadLastValid(filePath string) (*Record, error) {
    f, _ := os.Open(filePath)

    // 1. 读取所有行到内存
    var lines [][]byte
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := make([]byte, len(scanner.Bytes()))
        copy(line, scanner.Bytes())
        lines = append(lines, line)
    }

    // 2. 从末尾向前遍历
    for i := len(lines) - 1; i >= 0; i-- {
        record, err := d.Decode(lines[i])
        if err != nil {
            continue  // 跳过错误行
        }

        // 3. 遇到 delete 操作，返回 nil（已删除）
        if record.Meta.IsDelete() {
            return nil, nil
        }

        // 4. 遇到 put 操作，返回（最新值）
        if record.Meta.IsPut() {
            return record, nil
        }
    }

    return nil, nil  // 文件为空
}
```

**为什么反向读取？**

场景示例：
```jsonl
{"_meta":{"k":"key1","v":1,"op":"put"},"data":{"value":1}}     # 历史
{"_meta":{"k":"key1","v":2,"op":"put"},"data":{"value":2}}     # 历史
{"_meta":{"k":"key1","v":3,"op":"put"},"data":{"value":3}}     # 历史
...
{"_meta":{"k":"key1","v":100,"op":"put"},"data":{"value":100}} # 最新 ← 从这里开始
```

正向读取需要解析所有 100 行，反向读取只需解析 1 行。

**性能对比：**
- 正向：O(n)，n = 总记录数
- 反向：O(k)，k = 通常为 1，最坏为 n

### 读取特定版本

```go
func (d *Decoder) ReadVersion(filePath string, version int) (*Record, error) {
    records, _ := d.ReadAll(filePath)

    // 查找指定版本
    for _, record := range records {
        if record.Meta.Version == version {
            return record, nil
        }
    }

    return nil, ErrVersionNotFound
}
```

### 获取最新版本号

```go
func (d *Decoder) GetLatestVersion(filePath string) (int, error) {
    records, _ := d.ReadAll(filePath)

    if len(records) == 0 {
        return 0, nil
    }

    // 最后一条记录的版本号
    return records[len(records)-1].Meta.Version, nil
}
```

## 性能优化

### 1. 行数统计（CountLines）

**快速路径** - 无需解析 JSON：

```go
func CountLines(filePath string) (int, error) {
    f, _ := os.Open(filePath)
    scanner := bufio.NewScanner(f)

    count := 0
    for scanner.Scan() {
        count++
    }

    return count, nil
}
```

**用途：**
- 判断是否需要压缩（行数 > 阈值）
- 统计历史版本数

### 2. 最后 N 条记录

**压缩时使用** - 只保留最近的记录：

```go
func (d *Decoder) ReadLastNRecords(filePath string, n int) ([]*Record, error) {
    records, _ := d.ReadAll(filePath)

    if len(records) <= n {
        return records, nil
    }

    // 返回最后 n 条
    return records[len(records)-n:], nil
}
```

## 错误处理策略

### 容错层级

```
Level 1: 单行错误     → 跳过，继续（log.Debug）
Level 2: 文件不存在   → 返回 ErrNotFound
Level 3: 磁盘满       → 返回 ErrDiskFull
Level 4: 权限拒绝     → 返回 ErrPermissionDenied
```

### 损坏数据恢复

**场景 1：中间行损坏**
```jsonl
{"_meta":{...},"data":{...}}  ✅ 正常
{broken line}                  ❌ 跳过
{"_meta":{...},"data":{...}}  ✅ 正常（不受影响）
```

**场景 2：最后一行损坏**
```jsonl
{"_meta":{...},"data":{...}}  ✅ 正常
{"_meta":{...},"data":{..     ❌ 不完整（断电）
```
解决：ReadLastValid 会跳过不完整行，读取倒数第二行。

## 设计权衡

### 1. 为什么不用二进制格式？

| 格式 | 优势 | 劣势 |
|------|------|------|
| **JSONL** | 人类可读、可编辑、可 grep | 体积大、解析慢 |
| **MessagePack** | 体积小、解析快 | 不可读、不可编辑 |
| **Protobuf** | 体积最小、最快 | 需要 schema |

**结论：** Stow 优先透明性，接受性能损失。

### 2. 为什么版本号从 1 开始？

**方案 A：从 0 开始**
- 第一版是 v0，不直观
- `if version == 0` 容易误判为"无版本"

**方案 B：从 1 开始** ✅
- 第一版是 v1，符合直觉
- `if version <= 0` 可以统一判断无效

### 3. 为什么软删除而非物理删除？

**物理删除问题：**
```jsonl
{"_meta":{"k":"key1","v":1,"op":"put"},"data":{...}}
{"_meta":{"k":"key1","v":2,"op":"put"},"data":{...}}
{"_meta":{"k":"key1","v":3,"op":"put"},"data":{...}}
# 删除 → 需要重写整个文件，移除所有记录
```

**软删除优势：**
```jsonl
{"_meta":{"k":"key1","v":1,"op":"put"},"data":{...}}
{"_meta":{"k":"key1","v":2,"op":"delete"},"data":null}  # 追加即可
```
- O(1) 删除
- 保留删除历史
- 可恢复（读取旧版本）

**清理：** 通过 Compact 操作物理删除。

## 测试覆盖

**14 个单元测试，覆盖率 69.1%**

关键测试：
- ✅ Meta 创建和操作检查
- ✅ Record 验证（nil meta、空 key、无效版本）
- ✅ Encoder/Decoder 正确性
- ✅ ReadLastValid - 反向读取
- ✅ ReadLastValidWithDelete - 删除记录处理
- ✅ AppendRecord - 追加写入
- ✅ 容错：跳过错误行

## 实际使用示例

```go
// 创建编码器和解码器
encoder := core.NewEncoder()
decoder := core.NewDecoder()

// 1. 创建记录
record := core.NewPutRecord("mykey", 1, map[string]interface{}{
    "name": "test",
    "age":  30,
})

// 2. 追加到文件
err := core.AppendRecord("/path/to/mykey.jsonl", record)

// 3. 读取最新值
latest, err := decoder.ReadLastValid("/path/to/mykey.jsonl")

// 4. 读取历史
history, err := decoder.ReadAll("/path/to/mykey.jsonl")

// 5. 读取特定版本
v2, err := decoder.ReadVersion("/path/to/mykey.jsonl", 2)
```

## 相关模块

- 上层：[Namespace](../namespace.go) - 调用 Core 进行读写
- 下层：[FSUtil](fsutil.md) - 提供文件系统操作
- 平级：[Codec](codec.md) - 处理数据序列化后调用 Core

## 潜在改进

参见 [couldbebetter.md](../couldbebetter.md#core-模块)
