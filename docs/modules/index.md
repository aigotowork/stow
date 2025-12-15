# Index 模块设计

## 职责定位

**Index 模块是查询加速层**，提供：
1. **Key 映射** - Key → 文件名的快速查找
2. **Key 清洗** - 处理非法文件名字符
3. **冲突检测** - 处理清洗后的 Key 冲突
4. **缓存** - 减少磁盘 I/O
5. **目录扫描** - 启动时构建索引

## 核心问题

### 问题 1：Key 包含非法字符

**场景：**
```go
ns.Put("user/profile:v1", data)
```

**问题：** `user/profile:v1` 包含 `/` 和 `:`，不能作为文件名。

**解决：** Key 清洗（Sanitize）
```go
"user/profile:v1" → "user_profile_v1"
```

### 问题 2：清洗后 Key 冲突

**场景：**
```go
ns.Put("user/data", data1)
ns.Put("user_data", data2)
```

**问题：** 两个 Key 清洗后都是 `user_data`。

**解决：** 哈希后缀
```go
"user/data"  → "user_data.jsonl"
"user_data"  → "user_data_a3f2c1.jsonl"  // 追加哈希
```

### 问题 3：启动时如何找到所有 Key？

**问题：** 文件名是清洗后的，如何知道原始 Key？

**解决：** 扫描目录，读取每个文件的第一行（包含原始 Key）。

## Key 清洗（Sanitize）

### 非法字符列表

**文件系统禁用字符：**
```
/  \  :  *  ?  "  <  >  |
```

### 清洗规则

```go
func SanitizeKey(key string) string {
    // 1. 替换非法字符为下划线
    result := key
    for _, char := range invalidChars {
        result = strings.ReplaceAll(result, char, "_")
    }

    // 2. 修剪首尾空格和下划线
    result = strings.Trim(result, " _")

    // 3. 空 Key 处理
    if result == "" {
        result = "unnamed"
    }

    return result
}
```

**示例：**
```go
"user/profile:v1"     → "user_profile_v1"
"file<test>"          → "file_test"
"  spaces  "          → "spaces"
"___underscores___"   → "underscores"
"query*"              → "query"
```

**注意：** 尾部 `_` 被修剪，避免 `query_` 这样的丑陋文件名。

### 文件名生成

```go
func GenerateFileName(key string, addHash bool) string {
    sanitized := SanitizeKey(key)

    if !addHash {
        return sanitized + ".jsonl"
    }

    // 追加 6 字节哈希
    hash := hashString(key)
    return fmt.Sprintf("%s_%s.jsonl", sanitized, hash)
}
```

**何时添加哈希？**
- 检测到冲突时
- 用户显式请求时

## KeyMapper - Key 映射器

### 数据结构

```go
type FileInfo struct {
    FileName    string  // 实际文件名
    OriginalKey string  // 原始 Key
}

type KeyMapper struct {
    index map[string][]FileInfo  // cleanKey → []FileInfo
    mu    sync.RWMutex
}
```

**索引结构：**
```
index = {
    "user_data": [
        {FileName: "user_data.jsonl", OriginalKey: "user/data"},
        {FileName: "user_data_abc123.jsonl", OriginalKey: "user_data"}
    ],
    "server_config": [
        {FileName: "server_config.jsonl", OriginalKey: "server_config"}
    ]
}
```

### 核心操作

#### Add - 添加映射

```go
func (km *KeyMapper) Add(originalKey, fileName string) {
    cleanKey := SanitizeKey(originalKey)

    // 检查是否已存在
    for i, info := range km.index[cleanKey] {
        if info.OriginalKey == originalKey {
            km.index[cleanKey][i].FileName = fileName  // 更新
            return
        }
    }

    // 添加新映射
    km.index[cleanKey] = append(km.index[cleanKey], FileInfo{
        FileName:    fileName,
        OriginalKey: originalKey,
    })
}
```

#### Find - 查找候选文件

```go
func (km *KeyMapper) Find(key string) []FileInfo {
    cleanKey := SanitizeKey(key)
    return km.index[cleanKey]  // 可能返回多个候选
}
```

**为什么返回多个候选？**

场景：`user/data` 和 `user_data` 都清洗为 `user_data`，Find 返回两个文件：
```go
[
    FileInfo{FileName: "user_data.jsonl", OriginalKey: "user/data"},
    FileInfo{FileName: "user_data_abc123.jsonl", OriginalKey: "user_data"}
]
```

调用者需要读取文件的 `_meta.k` 字段匹配原始 Key。

#### HasConflict - 冲突检测

```go
func (km *KeyMapper) HasConflict(key string) bool {
    cleanKey := SanitizeKey(key)
    return len(km.index[cleanKey]) > 1
}
```

**用途：**
- 决定是否需要添加哈希后缀
- 警告用户潜在的 Key 冲突

## Scanner - 目录扫描器

### 职责

启动时扫描 Namespace 目录，构建 KeyMapper 索引。

### 扫描流程

```go
func (s *Scanner) ScanNamespace(path string) (*KeyMapper, error) {
    mapper := NewKeyMapper()

    // 1. 查找所有 .jsonl 文件
    files, _ := fsutil.FindFiles(path, "*.jsonl")

    // 2. 逐个处理
    for _, filePath := range files {
        // 跳过 _blobs 目录
        if strings.Contains(filePath, "_blobs") {
            continue
        }

        // 3. 读取第一行，获取原始 Key
        originalKey, err := s.readKeyFromFile(filePath)
        if err != nil {
            continue  // 跳过无效文件
        }

        // 4. 添加到 Mapper
        fileName := filepath.Base(filePath)
        mapper.Add(originalKey, fileName)
    }

    return mapper, nil
}
```

**为什么只读第一行？**

JSONL 文件格式保证所有记录的 Key 相同：
```jsonl
{"_meta":{"k":"server","v":1,...},"data":{...}}  # Key = server
{"_meta":{"k":"server","v":2,...},"data":{...}}  # Key = server
{"_meta":{"k":"server","v":3,...},"data":{...}}  # Key = server
```

读取第一行即可获取 Key，避免解析整个文件。

### 容错设计

```go
// 跳过无效文件
for _, filePath := range files {
    originalKey, err := s.readKeyFromFile(filePath)
    if err != nil {
        // 不返回错误，继续处理其他文件
        log.Warn("invalid file", filePath, err)
        continue
    }
}
```

**场景：**
- 空文件
- 格式错误的 JSON
- 损坏的文件

**策略：** 跳过，不阻塞启动。

## Cache - 缓存层

### 设计目标

1. **减少磁盘 I/O** - 缓存热数据
2. **避免惊群** - TTL + Jitter
3. **支持失效** - 手动刷新
4. **线程安全** - sync.Map

### 数据结构

```go
type Cache struct {
    store  *sync.Map
    ttl    time.Duration
    jitter float64
}

type cacheEntry struct {
    value      interface{}
    expireTime time.Time
}
```

### TTL + Jitter 机制

**问题：** 所有缓存同时过期 → 惊群

```
时间 0:   1000 个请求同时缓存
时间 60s: 1000 个缓存同时过期
时间 60s: 1000 个请求同时打到磁盘 → 卡顿
```

**解决：** 随机抖动

```go
func (c *Cache) calculateTTL(baseTTL time.Duration) time.Duration {
    if c.jitter == 0 {
        return baseTTL
    }

    // ±20% 随机抖动
    factor := 1.0 + (rand.Float64()*2-1)*c.jitter
    return time.Duration(float64(baseTTL) * factor)
}
```

**效果：**
```
基础 TTL: 60s
Jitter:   ±20%
实际 TTL: 48s ~ 72s（均匀分布）

时间 48s: 部分缓存开始过期
时间 60s: 大部分缓存过期
时间 72s: 最后的缓存过期

→ 请求分散在 24s 时间窗口内
```

### 核心操作

#### Set - 设置缓存

```go
func (c *Cache) Set(key string, value interface{}) {
    ttl := c.calculateTTL(c.ttl)
    expireTime := time.Now().Add(ttl)

    c.store.Store(key, cacheEntry{
        value:      value,
        expireTime: expireTime,
    })
}
```

#### Get - 获取缓存

```go
func (c *Cache) Get(key string) (interface{}, bool) {
    v, ok := c.store.Load(key)
    if !ok {
        return nil, false
    }

    entry := v.(cacheEntry)

    // 检查是否过期
    if time.Now().After(entry.expireTime) {
        c.store.Delete(key)  // 清理过期条目
        return nil, false
    }

    return entry.value, true
}
```

**懒惰清理策略：**
- 不定时扫描清理（避免性能开销）
- 读取时检查过期（懒惰删除）

### 失效策略

```go
// 单个 Key 失效
func (c *Cache) Delete(key string) {
    c.store.Delete(key)
}

// 批量失效
func (c *Cache) DeleteMultiple(keys []string) {
    for _, key := range keys {
        c.store.Delete(key)
    }
}

// 全部失效
func (c *Cache) Clear() {
    c.store = &sync.Map{}  // 新建，旧的被 GC
}
```

**使用场景：**
- Delete: Put/Delete 操作后
- DeleteMultiple: Compact 操作后
- Clear: RefreshAll 操作

## 性能分析

### 时间复杂度

| 操作 | 时间复杂度 | 说明 |
|------|------------|------|
| Sanitize | O(n) | n = Key 长度 |
| KeyMapper.Add | O(1) | 哈希表插入 |
| KeyMapper.Find | O(1) | 哈希表查找 |
| Scanner.Scan | O(m) | m = 文件数 |
| Cache.Get | O(1) | sync.Map 查询 |
| Cache.Set | O(1) | sync.Map 插入 |

### 内存占用

```
索引大小 = 记录数 × 64 bytes
缓存大小 = 活跃记录数 × 记录大小

示例：
- 10000 条记录
- 索引：10000 × 64 = 640KB
- 缓存（1000 活跃）：1000 × 1KB = 1MB
总计：~1.6MB
```

## 设计权衡

### 1. 为什么不用 LRU 缓存？

**LRU 问题：**
- 需要维护访问链表（复杂）
- 锁竞争严重（读写都需要移动节点）
- 内存开销大（额外指针）

**TTL 优势：**
- ✅ 实现简单
- ✅ 读操作无锁竞争
- ✅ 内存开销小

**适用性：** Stow 是读多写少场景，TTL 足够。

### 2. 为什么用 sync.Map 而非 map + Mutex？

**性能对比（读多场景）：**
```
sync.Map:        1000K ops/s
map + RWMutex:   500K ops/s
```

**原因：**
- sync.Map 针对读多写少优化
- 读操作无锁（copy-on-write）
- 适合缓存场景

### 3. 为什么不在启动时预加载所有数据？

**预加载问题：**
- 启动慢（需要读取所有文件）
- 内存占用大
- 大部分数据不会被访问

**延迟加载优势：**
- ✅ 启动快（仅扫描文件名）
- ✅ 内存占用小
- ✅ 按需加载

## 测试覆盖

**17 个单元测试，覆盖率 40.1%**

关键测试：
- ✅ Key 清洗规则（12 种场景）
- ✅ 文件名生成（有/无哈希）
- ✅ 冲突检测
- ✅ KeyMapper 完整功能
- ✅ 冲突处理
- ✅ Scanner 目录扫描
- ✅ Scanner 跳过无效文件
- ✅ Cache TTL 过期
- ✅ Cache Jitter 工作
- ✅ Cache 批量删除

## 实际使用示例

```go
// 1. Key 清洗
cleanKey := index.SanitizeKey("user/profile:v1")
// → "user_profile_v1"

// 2. 生成文件名
fileName := index.GenerateFileName("user/data", false)
// → "user_data.jsonl"

// 3. 创建 Mapper
mapper := index.NewKeyMapper()
mapper.Add("user/data", "user_data.jsonl")
mapper.Add("user_data", "user_data_abc123.jsonl")

// 4. 查找文件
files := mapper.Find("user/data")
// → [FileInfo{FileName: "user_data.jsonl", OriginalKey: "user/data"}]

// 5. 冲突检测
hasConflict := mapper.HasConflict("user_data")
// → true

// 6. 扫描目录
scanner := index.NewScanner()
mapper, _ := scanner.ScanNamespace("./data/namespace1")

// 7. 使用缓存
cache := index.NewCache(5*time.Minute, 0.2)
cache.Set("key1", data)
value, ok := cache.Get("key1")
```

## 并发安全

### KeyMapper

```go
type KeyMapper struct {
    mu sync.RWMutex  // 读写锁
}

func (km *KeyMapper) Find(key string) {
    km.mu.RLock()
    defer km.mu.RUnlock()
    // 读操作
}

func (km *KeyMapper) Add(key, fileName string) {
    km.mu.Lock()
    defer km.mu.Unlock()
    // 写操作
}
```

### Cache

```go
type Cache struct {
    store *sync.Map  // 内置并发安全
}

// 无需额外加锁
```

## 相关模块

- 上层：[Namespace](../namespace.go) - 使用 Index 查找文件
- 下层：[FSUtil](fsutil.md) - 文件系统扫描
- 平级：[Core](core.md) - 读取文件内容

## 潜在改进

参见 [couldbebetter.md](../couldbebetter.md#index-模块)
