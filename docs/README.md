# Stow 设计文档

## 项目定位

**Stow** 是一个介于纯 JSON 文件和 SQLite 数据库之间的嵌入式 KV 存储引擎。

### 核心价值主张

1. **透明性** - 使用人类可读的 JSONL 格式，可直接查看和编辑
2. **可编辑性** - 支持外部修改，通过缓存刷新机制同步
3. **媒体友好** - 智能 Blob 路由，大文件独立存储不占内存
4. **版本追踪** - 追加写入保留完整历史，支持时间旅行
5. **零依赖** - 纯 Go 实现，无需外部数据库

### 适用场景

**适合：**
- 配置管理（需要人工审查）
- 小型应用数据存储（< 10000 条记录）
- 多媒体内容管理（图片、视频元数据 + 文件）
- 需要版本历史的数据
- 需要跨进程编辑的数据

**不适合：**
- 高并发写入（> 1000 TPS）
- 复杂查询（无索引、全表扫描）
- 大规模数据（> 100MB 单 namespace）
- 需要事务的场景

## 核心设计理念

### 1. 简单至上 (Simplicity First)

```
复杂度层次：纯文件 < Stow < SQLite < PostgreSQL
```

Stow 的设计目标是"刚刚好够用"的复杂度：
- 比纯文件多一点：结构化 + 版本管理 + Blob 路由
- 比数据库少很多：无 SQL、无索引、无事务

### 2. 透明可见 (Transparency)

所有数据以 JSONL 格式存储，用户可以：
- 用文本编辑器查看
- 用 grep/awk 搜索
- 用 git 跟踪变化
- 手工修复损坏数据

### 3. 追加优先 (Append-Only)

永不修改已有数据，只追加新记录：
- 简化并发控制（无 WAL）
- 保留完整历史
- 故障恢复简单（最多丢失最后一行）

### 4. 延迟加载 (Lazy Loading)

仅在需要时加载数据：
- 启动快（不预加载）
- 内存占用小
- 支持大文件（流式读取）

## 架构概览

```
┌─────────────────────────────────────────────────┐
│                   Store                         │
│  (命名空间管理器)                                │
└────────────┬────────────────────────────────────┘
             │
             ├─── Namespace 1 (config/)
             ├─── Namespace 2 (users/)
             └─── Namespace 3 (media/)
                        │
        ┌───────────────┴────────────────┐
        │        Namespace               │
        │   (隔离的存储空间)              │
        └───────────┬────────────────────┘
                    │
        ┌───────────┼────────────┐
        │           │            │
    ┌───▼───┐  ┌───▼───┐  ┌────▼─────┐
    │ Index │  │ Cache │  │   Blob   │
    │(映射) │  │(加速) │  │ Manager  │
    └───┬───┘  └───────┘  └────┬─────┘
        │                       │
    ┌───▼────────────────┐  ┌──▼──────┐
    │   JSONL Files      │  │ _blobs/ │
    │ (key1.jsonl)       │  │ (files) │
    │ (key2.jsonl)       │  │         │
    └────────────────────┘  └─────────┘
```

## 核心模块关系

### 数据流：Put 操作

```
User Data
    │
    ▼
┌─────────────┐
│  Codec      │  识别大字段 → Blob
│ (Marshal)   │  小字段 → 内联
└──────┬──────┘
       │
       ├──────────────┐
       │              │
   内联数据        大字段
       │              │
       │          ┌───▼────┐
       │          │  Blob  │
       │          │Manager │
       │          └───┬────┘
       │              │
       │          保存文件
       │              │
       │          返回引用
       │              │
       ▼              ▼
   ┌──────────────────────┐
   │    JSONL Record      │
   │  {data: {...},       │
   │   field: $blob}      │
   └──────────┬───────────┘
              │
          追加到文件
              │
   ┌──────────▼───────────┐
   │    key1.jsonl        │
   │  line 1: v1          │
   │  line 2: v2 (append) │
   └──────────────────────┘
```

### 数据流：Get 操作

```
Key
 │
 ▼
┌────────┐    命中     ┌────────┐
│ Cache  │──────────→ │ 返回  │
└────┬───┘            └────────┘
     │ 未命中
     ▼
┌────────┐
│ Index  │  找到文件名
│(Mapper)│
└────┬───┘
     │
     ▼
┌──────────────┐
│ JSONL Decoder│  反向读取最后一条有效记录
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Record    │
│ {data: ...}  │
└──────┬───────┘
       │
       ▼
  有 Blob 引用?
       │
   ┌───┴───┐
   否      是
   │       │
   │   ┌───▼────┐
   │   │  Blob  │  加载文件
   │   │Manager │
   │   └───┬────┘
   │       │
   └───┬───┘
       ▼
┌──────────────┐
│  Unmarshal   │  填充目标结构体
└──────┬───────┘
       │
       ▼
   返回用户
```

## 模块职责

| 模块 | 职责 | 文件 |
|------|------|------|
| **Store** | 管理多个 Namespace | [store.go](../store.go) |
| **Namespace** | 核心 KV 引擎 | [namespace.go](../namespace.go) |
| **Index** | Key→文件映射、缓存 | [internal/index/](modules/index.md) |
| **Core** | JSONL 编解码 | [internal/core/](modules/core.md) |
| **Blob** | 大文件管理 | [internal/blob/](modules/blob.md) |
| **Codec** | 序列化/反序列化 | [internal/codec/](modules/codec.md) |
| **FSUtil** | 文件系统工具 | [internal/fsutil/](modules/fsutil.md) |

## 关键设计决策

### 1. 为什么使用 JSONL 而非 JSON？

**JSONL 优势：**
- ✅ 追加写入高效（无需重写整个文件）
- ✅ 逐行解析，内存占用小
- ✅ 容错：坏行不影响其他行
- ✅ 流式处理友好

**JSON 劣势：**
- ❌ 每次修改需重写整个文件
- ❌ 必须全部加载到内存
- ❌ 一处损坏全文件不可用

### 2. 为什么反向读取文件？

实现 **Last Write Wins** 语义：
```go
// 从文件末尾向前读，找到第一条有效的 put 记录
for i := len(lines) - 1; i >= 0; i-- {
    if record.Op == "put" {
        return record  // 最新的值
    }
}
```

**好处：**
- 避免解析完整历史
- O(1) 获取最新值（通常最后几行）
- 支持删除操作（遇到 delete 立即返回 NotFound）

### 3. 为什么分离 Blob 存储？

**问题：** 如果把图片 base64 编码存 JSONL，会导致：
- 文件膨胀 33%（base64 开销）
- 内存占用大（无法流式读取）
- 查询慢（每次读取大量无用数据）

**解决：** Blob 单独存储，JSONL 只存引用：
```json
{
  "_meta": {...},
  "data": {
    "title": "Photo",
    "image": {
      "$blob": true,
      "loc": "_blobs/image_a3f2c1.jpg",
      "hash": "a3f2c1...",
      "size": 2048000
    }
  }
}
```

### 4. 为什么需要 Key 清洗？

**问题：** 用户输入的 Key 可能包含非法文件名字符：
```
"user/profile:v1"  → 在 Windows 上非法
"file<test>"       → < > 是保留字符
```

**解决：** 清洗 + 冲突检测：
```
"user/profile:v1" → "user_profile_v1.jsonl"
"user_profile:v1" → "user_profile_v1_abc123.jsonl"  (冲突，加哈希)
```

### 5. 为什么使用 TTL + Jitter 缓存？

**问题：** 所有缓存同时过期 → 惊群效应：
```
时间 0:   1000 个请求同时缓存
时间 60s: 1000 个缓存同时过期
时间 60s: 1000 个请求同时打到磁盘 → 卡顿
```

**解决：** TTL 加随机抖动：
```go
actualTTL = baseTTL * (1 + random(-0.2, +0.2))
// 60s ± 20% = 48s~72s，分散过期时间
```

## 性能特征

### 时间复杂度

| 操作 | 时间复杂度 | 说明 |
|------|------------|------|
| Put | O(1) | 追加写入 |
| Get (缓存命中) | O(1) | 内存查询 |
| Get (缓存未命中) | O(k) | k = 文件尾部扫描行数 |
| Delete | O(1) | 追加 delete 记录 |
| List | O(n) | 遍历所有文件 |
| Compact | O(m) | m = 保留记录数 |

### 空间复杂度

```
总空间 = JSONL 文件 + Blob 文件 + 索引 + 缓存

JSONL   ≈ 记录数 × 平均记录大小 × (1 + 历史版本数)
Blob    = 大文件总和
索引    ≈ 记录数 × 64 bytes
缓存    ≈ 活跃记录数 × 记录大小
```

## 并发模型

### 读写锁策略

```go
type namespace struct {
    mu sync.RWMutex  // Namespace 级别锁
}

// 读操作：允许并发
func (ns *namespace) Get() {
    ns.mu.RLock()
    defer ns.mu.RUnlock()
    // ...
}

// 写操作：独占
func (ns *namespace) Put() {
    ns.mu.Lock()
    defer ns.mu.Unlock()
    // ...
}
```

**特点：**
- ✅ 多读不阻塞
- ✅ 写时阻塞所有读写
- ⚠️ 粒度：Namespace 级别（非 Key 级别）

**权衡：**
- 简单，不易出错
- 不支持 Key 级并发写（未来可优化）

## 故障恢复

### 写入失败处理

```go
1. 写临时文件     (*.tmp)
2. Sync 到磁盘
3. 原子 Rename    (*.tmp → *.jsonl)
```

**保证：**
- 要么完整写入，要么不写入
- 不会出现部分写入的文件

### 文件损坏处理

**JSONL 容错设计：**
```go
// 解码时跳过错误行
for scanner.Scan() {
    record, err := Decode(line)
    if err != nil {
        continue  // 跳过，不返回错误
    }
}
```

**Blob 缺失处理：**
```go
// Unmarshal 时，Blob 缺失返回零值
if !blobManager.Exists(ref) {
    log.Warn("blob missing", ref)
    // 返回 nil，不返回错误
}
```

## 限制与约束

### 已知限制

1. **单机限制**
   - 无网络访问
   - 无分布式锁

2. **性能限制**
   - 写入：~1000 TPS（单 Namespace）
   - 读取（缓存）：~100K TPS
   - 读取（磁盘）：~1K TPS

3. **查询限制**
   - 无索引（O(n) List）
   - 无复杂查询（无 WHERE/JOIN）
   - 无聚合（无 COUNT/SUM）

4. **大小限制**
   - 单 Key 大小：< MaxFileSize (默认 10MB)
   - 单 Namespace：< 10000 Keys 推荐
   - 单文件历史：< 10000 版本

### 配置建议

```go
// 默认配置（适合大多数场景）
DefaultNamespaceConfig{
    BlobThreshold:      1KB,    // 小于 1KB 内联
    MaxFileSize:        10MB,   // 单文件最大 10MB
    CacheTTL:           5min,   // 缓存 5 分钟
    CompactThreshold:   100,    // 100 条记录触发压缩
    CompactKeepRecords: 10,     // 保留最近 10 版本
}

// 高频写入场景
HighWriteConfig{
    AutoCompact:     false,  // 禁用自动压缩
    DisableCache:    true,   // 禁用缓存避免一致性问题
}

// 只读场景
ReadOnlyConfig{
    CacheTTL:         1hour, // 长缓存时间
    DisableCache:    false,
}
```

## 后续阅读

- [架构详解](architecture.md) - 深入架构设计
- [模块设计](modules/) - 各模块详细设计
  - [Core - JSONL 编解码](modules/core.md)
  - [Blob - 文件管理](modules/blob.md)
  - [Index - 索引缓存](modules/index.md)
  - [Codec - 序列化](modules/codec.md)
  - [FSUtil - 文件工具](modules/fsutil.md)
- [设计反思](couldbebetter.md) - 可改进之处

## 快速开始示例

```go
// 1. 打开 Store
store := stow.MustOpen("./data")
defer store.Close()

// 2. 获取 Namespace
ns := store.MustGetNamespace("config")

// 3. 存储数据
ns.MustPut("server", map[string]interface{}{
    "host": "localhost",
    "port": 8080,
})

// 4. 读取数据
var config map[string]interface{}
ns.MustGet("server", &config)

// 5. 历史版本
history, _ := ns.GetHistory("server")

// 6. 维护
ns.Compact("server")  // 压缩历史
ns.GC()               // 清理 Blob
```
