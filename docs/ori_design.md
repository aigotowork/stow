# Stow - 嵌入式透明文件 KV 存储引擎
## 完整产品需求与技术设计文档 v2.0

---

## 📋 目录

1. [项目概述](#1-项目概述)
2. [设计哲学与核心原则](#2-设计哲学与核心原则)
3. [功能需求规范](#3-功能需求规范)
4. [架构设计](#4-架构设计)
5. [数据模型设计](#5-数据模型设计)
6. [接口设计规范](#6-接口设计规范)
7. [实现细节约束](#7-实现细节约束)
8. [项目结构设计](#8-项目结构设计)
9. [使用场景与示例](#9-使用场景与示例)
10. [验收标准](#10-验收标准)

---

## 1. 项目概述

### 1.1 项目定位

**Stow** 是一个用 Golang 实现的嵌入式 KV 存储引擎，定位于"纯文本配置文件"与"SQLite/嵌入式数据库"之间的存储解决方案。

### 1.2 核心痛点

| 存储方式 | 优点 | 缺点 |
|---------|------|------|
| **SQLite** | 完整的数据库特性、高性能 | 数据不透明、无法直接编辑、二进制格式 |
| **纯 JSON 文件** | 人类可读、可直接编辑 | 缺乏原子写入、无历史记录、大文件性能差、不适合二进制数据 |
| **Stow** | **透明可读 + 数据库特性 + 多媒体友好** | - |

### 1.3 目标用户

- 需要可视化配置文件的应用程序
- 需要存储混合数据（文本 + 二进制）的工具
- 需要版本历史但不想引入完整数据库的项目
- 需要用户能直接编辑数据文件的场景

---

## 2. 设计哲学与核心原则

### 2.1 透明性 (Transparency)

**原则**：数据即文件，利用文件系统作为天然索引

- ✅ 所有数据以 JSONL 格式存储，可用任何文本编辑器查看
- ✅ 目录结构直观，Key 映射为文件名
- ✅ 元数据（`_meta`）与业务数据（`data`）分离但在同一文件中
- ✅ 二进制文件独立存储，通过引用关联

### 2.2 可编辑性 (Editability)

**原则**：允许用户在程序关闭时手动修改数据

- ✅ 程序运行时允许外部编辑，缓存失效时自动重新加载
- ✅ 提供 `Refresh()` 接口供用户主动触发重载
- ✅ 不校验时间戳顺序，以文件内容为准
- ✅ 容错设计：跳过格式错误的行，向前查找有效记录

### 2.3 多媒体友好 (Media-friendly)

**原则**：针对音频、图片等二进制数据优化

- ✅ 智能 Blob 路由：小数据 Base64 内联，大数据独立文件
- ✅ 支持流式读写，避免大文件 OOM
- ✅ 支持自定义文件名，方便用户直接替换资源
- ✅ 自动管理 Blob 引用，支持垃圾回收

### 2.4 简单性 (Simplicity)

**原则**：单进程独占，不处理复杂的分布式场景

- ✅ 不保证跨进程的原子性（可选文件锁防止多开）
- ✅ 不实现批量事务
- ✅ 不支持跨 Namespace 操作
- ✅ 使用悲观锁保证进程内并发安全

---

## 3. 功能需求规范

### 3.1 基础 KV 能力

#### 3.1.1 写入操作 (Put)

**功能描述**：存储键值对数据

**接口签名**：
```go
Put(key string, value interface{}, opts ...PutOption) error
MustPut(key string, value interface{}, opts ...PutOption)
```

**支持的数据类型**：
- 基本类型：`string`, `int`, `bool`, `float64` 等
- 结构体：任意 struct，支持嵌套
- 字节数组：`[]byte`
- 流数据：`io.Reader`
- 映射：`map[string]interface{}`

**可选参数**：
- `WithForceFile()`：强制存为文件（即使小于阈值）
- `WithFileName(name string)`：指定文件名
- `WithMimeType(mime string)`：指定 MIME 类型

**行为约束**：
- Append-only 写入，不修改历史记录
- 自动递增版本号
- 写入成功后更新内存索引
- 根据配置自动触发压缩

#### 3.1.2 读取操作 (Get)

**功能描述**：读取指定 Key 的最新数据

**接口签名**：
```go
Get(key string, target interface{}) error
MustGet(key string, target interface{})
GetRaw(key string) (RawItem, error)
```

**行为约束**：
- Last Write Wins：返回最后一条有效记录
- 优先从缓存读取（如果未过期）
- 自动处理 Blob 引用：
  - 目标字段为 `[]byte`：读取文件内容到内存
  - 目标字段为 `IFileData`：返回流句柄（不加载内容）
- Blob 文件不存在时：打印 Warn 日志，字段设为零值

#### 3.1.3 删除操作 (Delete)

**功能描述**：软删除指定 Key

**接口签名**：
```go
Delete(key string) error
MustDelete(key string)
```

**行为约束**：
- 追加 `op: "delete"` 记录，不物理删除文件
- 删除后 `Get` 返回 `ErrNotFound`
- 关联的 Blob 文件不立即删除，等待 GC

#### 3.1.4 存在性检查 (Exists)

**功能描述**：检查 Key 是否存在

**接口签名**：
```go
Exists(key string) bool
```

#### 3.1.5 列表操作 (List)

**功能描述**：列出当前 Namespace 下所有有效的 Key

**接口签名**：
```go
List() ([]string, error)
```

**行为约束**：
- 扫描目录下所有 `.jsonl` 文件
- 过滤掉已删除的 Key
- 返回原始 Key（非文件名）

---

### 3.2 高级数据处理

#### 3.2.1 历史版本管理

**功能描述**：查询和访问 Key 的历史版本

**接口签名**：
```go
GetHistory(key string) ([]Version, error)
GetVersion(key string, version int, target interface{}) error
```

**Version 结构**：
```go
type Version struct {
    Version   int
    Timestamp time.Time
    Operation string  // "put" | "delete"
    Size      int64
}
```

**行为约束**：
- 历史版本按时间倒序排列
- 历史版本的 Blob 文件保留，直到 Compact 后确认不再引用

#### 3.2.2 智能 Blob 路由

**触发条件**（满足任一即存为文件）：
1. 数据类型为 `io.Reader`
2. `[]byte` 大小超过阈值（默认 4KB）
3. Struct Tag 包含 `stow:"file"`
4. 调用时传入 `WithForceFile()` 选项

**文件命名规则**：
1. **指定名称**：`{name}_{hash}.{ext}`
   - 通过 `WithFileName()` 指定
   - 或 Tag 中 `name:xxx` 指定
   - 或 Tag 中 `name_field:FieldName` 引用其他字段
2. **无指定名称**：`{hash}.bin`
   - 使用 SHA256 哈希前 16 位

**存储位置**：`{namespace}/_blobs/`

**引用结构**（在 JSONL 中）：
```json
{
  "$blob": true,
  "loc": "_blobs/avatar_a1b2c3d4.jpg",
  "hash": "a1b2c3d4e5f6...",
  "size": 102400,
  "mime": "image/jpeg",
  "name": "avatar.jpg"
}
```

**查询效率优化**：
- Namespace 启动时扫描 `_blobs/` 目录
- 建立"纯净文件名 → 带哈希文件名"的内存映射
- 示例：`avatar.jpg` → `["avatar_abc123.jpg", "avatar_def456.jpg"]`

#### 3.2.3 Key 清洗与冲突处理

**清洗规则**：
- 移除非法字符：`/ \ : * ? " < > |`
- 替换为下划线 `_`
- 示例：`user/data:v1` → `user_data_v1`

**冲突处理**：
- 清洗后文件名相同时，追加哈希后缀
- 示例：
  - `user/data:v1` → `user_data_v1_a1b2c3.jsonl`
  - `user_data:v1` → `user_data_v1_d4e5f6.jsonl`

**原始 Key 保存**：
- 在 `_meta.k` 字段存储原始 Key
- 读取时通过遍历候选文件，匹配 `_meta.k` 确定正确文件

**索引缓存**：
- Namespace 启动时遍历目录，建立索引
- 结构：`cleanKey → [{fileName, originalKey}]`
- `Get` 时先查索引，再匹配原始 Key

#### 3.2.4 懒加载 (Lazy Loading)

**原则**：初始化时不加载数据到内存

**实现**：
- Namespace 启动时仅扫描文件名，建立索引
- `Get` 调用时才打开并解析对应文件
- 缓存机制：解析后根据 TTL 缓存结果

#### 3.2.5 流式文件处理

**IFileData 接口**：
```go
type IFileData interface {
    io.ReadCloser
    Name() string
    Size() int64
    MimeType() string
    Path() string
    Hash() string
}
```

**行为约束**：
- 不将文件内容全部加载到内存
- 返回实现了 `io.ReadCloser` 的句柄
- 用户负责调用 `Close()` 释放资源
- 支持多次 `Read()` 调用（流式读取）

---

### 3.3 维护操作

#### 3.3.1 压缩 (Compact)

**功能描述**：合并 JSONL 文件，减少历史记录占用空间

**接口签名**：
```go
Compact(keys ...string) error
CompactAll() error
```

**触发策略**（在 Namespace 配置中指定）：
1. **按行数触发**：文件超过 N 行（默认 20 行）
2. **按文件大小触发**：文件超过 M 字节
3. **按时间触发**：定期后台任务
4. **手动触发**：仅通过接口调用

**压缩策略**：
- 保留最后 N 条记录（默认 3 条）
- 删除更早的历史版本
- 标记不再引用的 Blob 文件（供 GC 清理）

**原子性保证**：
1. 写入临时文件 `{key}.jsonl.tmp`
2. Sync 到磁盘
3. 原子 Rename 替换原文件
4. 删除临时文件

**自动压缩**：
- 配置 `AutoCompact: true` 时，每次 `Put` 后检查
- 满足触发条件则自动执行压缩

#### 3.3.2 垃圾回收 (GC)

**功能描述**：清理未被引用的 Blob 文件

**接口签名**：
```go
GC() (GCResult, error)
```

**GCResult 结构**：
```go
type GCResult struct {
    RemovedBlobs  int
    ReclaimedSize int64
    Duration      time.Duration
}
```

**执行流程**：
1. 扫描所有 `.jsonl` 文件，收集所有 Blob 引用
2. 扫描 `_blobs/` 目录，找出未被引用的文件
3. 删除孤立文件
4. 返回清理统计

**触发方式**：
- 仅手动调用，不自动执行
- 建议在 Compact 后调用

#### 3.3.3 缓存刷新 (Refresh)

**功能描述**：重新加载数据，检测外部修改

**接口签名**：
```go
Refresh(keys ...string) error
RefreshAll() error
```

**行为约束**：
- 清除指定 Key 的缓存
- 下次 `Get` 时重新读取文件
- 支持在程序运行时检测用户手动编辑

---

### 3.4 配置管理

#### 3.4.1 Namespace 配置

**接口签名**：
```go
GetConfig() NamespaceConfig
SetConfig(config NamespaceConfig) error
```

**配置项**：

| 配置项 | 类型 | 默认值 | 说明 |
|-------|------|--------|------|
| `BlobThreshold` | `int64` | 4KB | Blob 阈值，超过此大小存为文件 |
| `MaxFileSize` | `int64` | 100MB | 单个文件最大大小限制 |
| `BlobChunkSize` | `int64` | 64KB | 写入 Blob 时的分块大小 |
| `CacheTTL` | `time.Duration` | 5 分钟 | 缓存过期时间 |
| `CacheTTLJitter` | `float64` | 0.2 | 缓存 TTL 随机偏移（±20%） |
| `DisableCache` | `bool` | false | 禁用缓存 |
| `CompactStrategy` | `CompactStrategy` | LineCount | 压缩策略 |
| `CompactThreshold` | `int` | 20 | 触发压缩的阈值 |
| `CompactKeepRecords` | `int` | 3 | 压缩后保留的历史记录数 |
| `AutoCompact` | `bool` | true | 是否自动压缩 |
| `LockTimeout` | `time.Duration` | 30 秒 | 锁超时时间 |

**配置持久化**：
- 配置存储在 `{namespace}/_config.json`
- 首次创建时写入默认配置
- `SetConfig` 时更新文件
- 部分配置（如缓存相关）立即生效，部分需要重启

#### 3.4.2 链式配置接口

**功能描述**：支持 Fluent API 风格的配置

**接口签名**：
```go
WithLogger(logger Logger) Namespace
WithBlobThreshold(bytes int64) Namespace
WithMaxFileSize(bytes int64) Namespace
```

**示例**：
```go
ns.WithLogger(myLogger).
   WithBlobThreshold(8 * 1024).
   Put("key", value)
```

---

### 3.5 日志与监控

#### 3.5.1 Logger 接口

**功能描述**：允许用户自定义日志输出

**接口定义**：
```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}
```

**使用场景**：
- Blob 文件不存在时打印 Warn
- 压缩、GC 操作的进度日志
- 文件 IO 错误的 Error 日志
- 调试模式下的详细操作日志

**设置方式**：
```go
ns.WithLogger(customLogger)
```

#### 3.5.2 统计信息

**接口签名**：
```go
Stats() (NamespaceStats, error)
```

**NamespaceStats 结构**：
```go
type NamespaceStats struct {
    KeyCount       int
    BlobCount      int
    TotalSize      int64
    BlobSize       int64
    LastCompactAt  time.Time
    LastGCAt       time.Time
}
```

---

## 4. 架构设计

### 4.1 分层架构

```
┌─────────────────────────────────────────┐
│      Application Layer (用户代码)        │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│   Access Layer (TypedBox/DynamicBox)    │  ← 业务访问层
│   - Struct Tag 解析                      │
│   - 类型转换                             │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│      Namespace Layer (核心逻辑层)        │  ← 核心引擎
│   - KV 操作                              │
│   - 索引管理                             │
│   - 缓存控制                             │
│   - 并发控制                             │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       Core Layer (底层能力层)            │  ← 基础设施
│   - JSONL 编解码                         │
│   - Blob 管理                            │
│   - 文件 IO                              │
│   - 压缩/GC                              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│         File System (存储层)             │
└─────────────────────────────────────────┘
```

### 4.2 模块职责划分

| 模块 | 职责 | 不负责 |
|------|------|--------|
| **Store** | Namespace 生命周期管理 | 具体数据操作 |
| **Namespace** | KV 操作、索引、缓存、并发控制 | 序列化细节 |
| **Codec** | 序列化/反序列化、Tag 解析 | 文件 IO |
| **Blob Manager** | Blob 文件读写、引用管理 | 业务逻辑 |
| **Index Cache** | 文件名映射、缓存管理 | 数据解析 |
| **Compactor** | 压缩策略执行 | 业务决策 |

---

## 5. 数据模型设计

### 5.1 目录结构

```
/BaseDir/
├── namespace_A/
│   ├── _config.json                 # Namespace 配置
│   ├── app_config.jsonl             # Key: "app_config"
│   ├── user_data_v1_a1b2c3.jsonl    # Key: "user/data:v1" (冲突后加哈希)
│   ├── settings.jsonl
│   └── _blobs/                      # Blob 文件池
│       ├── avatar_abc123.jpg        # 指定名称的文件
│       ├── avatar_def456.jpg        # 同名但不同版本
│       ├── e5f6g7h8.bin             # 无名称的文件（纯哈希）
│       └── resume_xyz789.pdf
│
├── namespace_B/
│   ├── _config.json
│   ├── ...
│   └── _blobs/
│
└── .lock                            # 进程锁文件（可选）
```

### 5.2 JSONL 文件格式

**单行记录结构**：
```json
{
  "_meta": {
    "k": "user/data:v1",
    "v": 3,
    "op": "put",
    "ts": "2025-12-14T18:09:00Z"
  },
  "data": {
    "name": "Alice",
    "age": 30,
    "avatar": {
      "$blob": true,
      "loc": "_blobs/avatar_abc123.jpg",
      "hash": "abc123...",
      "size": 102400,
      "mime": "image/jpeg",
      "name": "avatar.jpg"
    }
  }
}
```

**删除记录**：
```json
{
  "_meta": {
    "k": "user/data:v1",
    "v": 4,
    "op": "delete",
    "ts": "2025-12-14T18:10:00Z"
  },
  "data": null
}
```

### 5.3 配置文件格式

**_config.json**：
```json
{
  "blob_threshold": 4096,
  "max_file_size": 104857600,
  "blob_chunk_size": 65536,
  "cache_ttl": "5m",
  "cache_ttl_jitter": 0.2,
  "disable_cache": false,
  "compact_strategy": "line_count",
  "compact_threshold": 20,
  "compact_keep_records": 3,
  "auto_compact": true,
  "lock_timeout": "30s"
}
```

---

## 6. 接口设计规范

### 6.1 Store 接口

```go
type Store interface {
    // Namespace 管理
    CreateNamespace(name string, config NamespaceConfig) (Namespace, error)
    GetNamespace(name string) (Namespace, error)
    MustGetNamespace(name string) Namespace
    ListNamespaces() ([]string, error)
    DeleteNamespace(name string) error
    
    // 生命周期
    Close() error
}

// 构造函数
func Open(basePath string, opts ...StoreOption) (Store, error)
func MustOpen(basePath string, opts ...StoreOption) Store
```

### 6.2 Namespace 接口

```go
type Namespace interface {
    // ========== 基础 KV ==========
    Put(key string, value interface{}, opts ...PutOption) error
    MustPut(key string, value interface{}, opts ...PutOption)
    Get(key string, target interface{}) error
    MustGet(key string, target interface{})
    GetRaw(key string) (RawItem, error)
    Delete(key string) error
    MustDelete(key string)
    Exists(key string) bool
    List() ([]string, error)
    
    // ========== 历史版本 ==========
    GetHistory(key string) ([]Version, error)
    GetVersion(key string, version int, target interface{}) error
    
    // ========== 维护 ==========
    Compact(keys ...string) error
    CompactAll() error
    GC() (GCResult, error)
    Refresh(keys ...string) error
    RefreshAll() error
    
    // ========== 配置 ==========
    GetConfig() NamespaceConfig
    SetConfig(config NamespaceConfig) error
    
    // ========== 链式调用 ==========
    WithLogger(logger Logger) Namespace
    WithBlobThreshold(bytes int64) Namespace
    WithMaxFileSize(bytes int64) Namespace
    
    // ========== 元信息 ==========
    Name() string
    Path() string
    Stats() (NamespaceStats, error)
}
```

### 6.3 数据类型接口

```go
// IFileData 文件数据接口
type IFileData interface {
    io.ReadCloser
    Name() string
    Size() int64
    MimeType() string
    Path() string
    Hash() string
}

// RawItem 原始数据项
type RawItem interface {
    Meta() MetaInfo
    DecodeInto(target interface{}) error
    RawData() map[string]interface{}
}

// Logger 日志接口
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}
```

### 6.4 配置与选项

```go
// Namespace 配置
type NamespaceConfig struct {
    BlobThreshold      int64
    MaxFileSize        int64
    BlobChunkSize      int64
    CacheTTL           time.Duration
    CacheTTLJitter     float64
    DisableCache       bool
    CompactStrategy    CompactStrategy
    CompactThreshold   int
    CompactKeepRecords int
    AutoCompact        bool
    LockTimeout        time.Duration
}

// Store 选项
type StoreOption func(*storeOptions)
func WithStoreLogger(logger Logger) StoreOption

// Put 选项
type PutOption func(*putOptions)
func WithForceFile() PutOption
func WithFileName(name string) PutOption
func WithMimeType(mime string) PutOption
```

### 6.5 Struct Tag 规范

**支持的 Tag 格式**：

```go
type Example struct {
    // 基础用法：标记为文件类型
    Avatar []byte `stow:"file"`
    
    // 指定文件名
    Cover []byte `stow:"file,name:cover.jpg"`
    
    // 引用其他字段作为文件名
    Resume IFileData `stow:"file,name_field:ResumeName"`
    ResumeName string
    
    // 指定 MIME 类型
    Video []byte `stow:"file,mime:video/mp4"`
    
    // 组合使用
    Photo []byte `stow:"file,name:photo.png,mime:image/png"`
}
```

**Tag 解析优先级**：
1. 函数调用时的 `PutOption`（最高优先级）
2. Struct Tag 中的配置
3. 自动推断（根据数据大小和类型）

**不支持的场景**（忽略 Tag）：
- 非 `[]byte` 或 `io.Reader` 类型
- 嵌套结构体
- 数组/切片类型（`[][]byte`）

---

## 7. 实现细节约束

### 7.1 序列化流程 (Marshal)

**Put 操作的完整流程**：

```
1. 加锁 (Namespace 级别写锁)
   ↓
2. 反射分析 value 类型
   ├─ 解析 Struct Tag
   └─ 识别需要存为 Blob 的字段
   ↓
3. 处理 Blob 数据
   ├─ 判断条件：
   │  ├─ io.Reader?
   │  ├─ []byte > threshold?
   │  ├─ Tag 包含 "file"?
   │  └─ opts.forceFile?
   ├─ 生成文件名：
   │  ├─ 有指定名称? → {name}_{hash}.{ext}
   │  └─ 无指定名称? → {hash}.bin
   ├─ 分块写入 _blobs/
   │  ├─ 每块 BlobChunkSize (64KB)
   │  ├─ 检查 MaxFileSize 限制
   │  └─ 写入失败则清理已写入部分
   └─ 计算 SHA256 哈希
   ↓
4. 构建 Record
   ├─ 替换 Blob 字段为 BlobReference
   ├─ 填充 _meta：
   │  ├─ k: 原始 Key
   │  ├─ v: 版本号++
   │  ├─ op: "put"
   │  └─ ts: 当前时间
   └─ 序列化为 JSON
   ↓
5. Append 到 .jsonl 文件
   ├─ 打开文件 (O_APPEND|O_CREATE|O_WRONLY)
   ├─ 写入 JSON + "\n"
   ├─ Sync 到磁盘
   └─ 关闭文件
   ↓
6. 更新索引缓存
   ├─ 更新文件名映射
   ├─ 更新 Blob 名称映射
   └─ 设置缓存过期时间 (TTL + jitter)
   ↓
7. 检查自动压缩
   ├─ AutoCompact 开启?
   ├─ 满足触发条件?
   └─ 异步执行 Compact (不阻塞返回)
   ↓
8. 解锁
   ↓
9. 返回 nil 或 error
```

### 7.2 反序列化流程 (Unmarshal)

**Get 操作的完整流程**：

```
1. 加读锁 (Namespace 级别读锁)
   ↓
2. 检查缓存
   ├─ 命中且未过期? → 跳到步骤 6
   └─ 未命中或过期? → 继续
   ↓
3. 查找文件名
   ├─ cleanKey = Sanitize(key)
   ├─ 从索引获取候选文件列表
   ├─ 遍历候选文件：
   │  ├─ 读取第一行 JSON
   │  ├─ 解析 _meta.k
   │  └─ 匹配原始 Key? → 找到目标文件
   └─ 未找到? → 返回 ErrNotFound
   ↓
4. 读取 JSONL 文件
   ├─ 打开文件
   ├─ 从最后一行开始向前读取
   ├─ 跳过格式错误的行 (JSON 解析失败)
   ├─ 找到第一条 op="put" 的记录
   └─ 如果都是 op="delete" → 返回 ErrNotFound
   ↓
5. 解析 Record
   ├─ 解析 JSON 到 map[string]interface{}
   ├─ 提取 data 字段
   └─ 遍历字段，识别 BlobReference
   ↓
6. 处理 Blob 引用
   ├─ 发现 BlobReference?
   │  ├─ 检查文件是否存在
   │  ├─ 不存在? → 打印 Warn 日志，字段设为零值
   │  ├─ target 字段类型是 []byte?
   │  │  └─ ReadAll 文件内容到内存
   │  └─ target 字段类型是 IFileData?
   │     └─ 创建 FileDataHandle (不读取内容)
   └─ 继续下一个字段
   ↓
7. 反射填充 target
   ├─ 根据 target 类型进行类型转换
   └─ 赋值到 target 指针
   ↓
8. 更新缓存
   ├─ 计算过期时间: TTL * (1 ± jitter)
   └─ 存入缓存
   ↓
9. 解锁
   ↓
10. 返回 nil 或 error
```

### 7.3 Compact 流程

```
1. 加写锁
   ↓
2. 读取完整 JSONL 文件
   ├─ 解析所有记录
   ├─ 跳过格式错误的行
   └─ 按版本号排序
   ↓
3. 应用压缩策略
   ├─ 保留最后 N 条记录 (CompactKeepRecords)
   ├─ 收集要删除的记录
   └─ 标记不再引用的 Blob 文件
   ↓
4. 写入临时文件
   ├─ 创建 {key}.jsonl.tmp
   ├─ 写入保留的记录
   ├─ Sync 到磁盘
   └─ 关闭文件
   ↓
5. 原子替换
   ├─ Rename {key}.jsonl.tmp → {key}.jsonl
   └─ 删除临时文件 (如果 Rename 失败)
   ↓
6. 更新元数据
   ├─ 记录 LastCompactAt
   └─ 更新缓存
   ↓
7. 解锁
   ↓
8. 返回 nil 或 error
```

### 7.4 GC 流程

```
1. 加写锁
   ↓
2. 扫描所有 JSONL 文件
   ├─ 解析每个文件
   ├─ 提取所有 BlobReference
   └─ 构建引用集合: Set<blobPath>
   ↓
3. 扫描 _blobs/ 目录
   ├─ 遍历所有文件
   ├─ 检查是否在引用集合中
   └─ 收集孤立文件列表
   ↓
4. 删除孤立文件
   ├─ 遍历孤立文件列表
   ├─ 删除文件
   ├─ 累计删除数量和大小
   └─ 记录错误 (继续处理其他文件)
   ↓
5. 更新元数据
   ├─ 记录 LastGCAt
   └─ 更新统计信息
   ↓
6. 解锁
   ↓
7. 返回 GCResult
```

### 7.5 并发控制

**锁粒度**：Namespace 级别

**锁类型**：`sync.RWMutex`

**锁策略**：
- **读操作**（Get, Exists, List）：`RLock()`
- **写操作**（Put, Delete）：`Lock()`
- **维护操作**（Compact, GC）：`Lock()`

**悲观锁语义**：
- 写操作期间，阻塞所有读写
- 写完成后才释放锁，保证"读自己写"一致性

**不支持的场景**：
- 跨 Namespace 的原子操作
- 批量事务
- 跨进程的并发控制（可选文件锁防止多开）

### 7.6 错误处理

**错误类型定义**：

```go
var (
    ErrNotFound          = errors.New("key not found")
    ErrKeyConflict       = errors.New("key conflict after sanitization")
    ErrFileTooLarge      = errors.New("file exceeds MaxFileSize limit")
    ErrDiskFull          = errors.New("disk space insufficient")
    ErrPermissionDenied  = errors.New("permission denied")
    ErrInvalidConfig     = errors.New("invalid configuration")
    ErrNamespaceNotFound = errors.New("namespace not found")
    ErrNamespaceExists   = errors.New("namespace already exists")
    ErrCorruptedData     = errors.New("data corrupted")
    ErrLockTimeout       = errors.New("lock acquisition timeout")
)
```

**错误处理策略**：

| 场景 | 处理方式 |
|------|---------|
| 磁盘空间不足 | 返回 `ErrDiskFull`，不清理已写入数据 |
| 文件权限不足 | 返回 `ErrPermissionDenied` |
| JSONL 某行格式错误 | 跳过该行，继续读取下一行 |
| 最后一行格式错误 | 向前查找有效记录 |
| Blob 文件不存在 | 打印 Warn 日志，字段设为零值 |
| 写入超过 MaxFileSize | 返回 `ErrFileTooLarge`，清理已写入部分 |
| 用户忘记 Close IFileData | 依赖 GC finalizer 自动清理（可选） |

### 7.7 缓存失效策略

**TTL 计算**：
```
actualTTL = CacheTTL * (1 + random(-jitter, +jitter))
```

**示例**：
- `CacheTTL = 5min`, `jitter = 0.2`
- 实际 TTL 范围：`4min ~ 6min`

**失效触发**：
1. 时间到期自动失效
2. `Refresh()` 手动清除
3. `Put/Delete` 操作后更新

**禁用缓存**：
- `DisableCache = true` 时，每次 Get 都从文件读取

---

## 8. 项目结构设计

### 8.1 完整目录树

```
stow/
├── README.md                    # 项目说明
├── LICENSE                      # 开源协议
├── go.mod
├── go.sum
├── .gitignore
│
├── doc/                         # 文档
│   ├── design.md                # 设计文档
│   ├── api.md                   # API 文档
│   └── examples.md              # 示例文档
│
├── stow.go                      # 主入口，Store 接口定义
├── store.go                     # Store 实现
├── namespace.go                 # Namespace 核心实现
├── namespace_config.go          # Namespace 配置
├── types.go                     # 公共类型定义
├── filedata.go                  # IFileData 接口与实现
├── logger.go                    # Logger 接口
├── errors.go                    # 错误定义
├── options.go                   # 选项模式
│
├── internal/                    # 内部实现（不对外暴露）
│   │
│   ├── core/                    # 核心数据结构
│   │   ├── record.go            # JSONL Record 结构
│   │   ├── meta.go              # MetaInfo 结构
│   │   ├── encoder.go           # JSONL 编码器
│   │   └── decoder.go           # JSONL 解码器
│   │
│   ├── blob/                    # Blob 管理
│   │   ├── manager.go           # Blob 文件管理器
│   │   ├── reference.go         # BlobReference 结构
│   │   ├── filedata.go          # FileDataHandle 实现
│   │   ├── hash.go              # 文件哈希计算
│   │   └── writer.go            # 分块写入器
│   │
│   ├── index/                   # 索引与缓存
│   │   ├── cache.go             # 内存缓存实现
│   │   ├── sanitize.go          # Key 清洗逻辑
│   │   ├── mapper.go            # 文件名映射器
│   │   └── scanner.go           # 目录扫描器
│   │
│   ├── codec/                   # 序列化
│   │   ├── marshal.go           # 序列化逻辑
│   │   ├── unmarshal.go         # 反序列化逻辑
│   │   ├── reflect.go           # 反射工具
│   │   └── tag.go               # Struct Tag 解析
│   │
│   ├── compact/                 # 压缩与 GC
│   │   ├── compactor.go         # 压缩器实现
│   │   ├── strategy.go          # 压缩策略
│   │   ├── gc.go                # 垃圾回收器
│   │   └── scheduler.go         # 自动压缩调度器
│   │
│   └── fsutil/                  # 文件系统工具
│       ├── atomic.go            # 原子文件写入
│       ├── lock.go              # 文件锁实现
│       ├── walk.go              # 目录遍历
│       └── safe.go              # 安全文件操作
│
├── examples/                    # 示例代码
│   ├── basic/
│   │   └── main.go              # 基础 KV 示例
│   ├── media/
│   │   └── main.go              # 多媒体文件示例
│   ├── config/
│   │   └── main.go              # 配置管理示例
│   └── advanced/
│       └── main.go              # 高级特性示例
│
└── tests/                       # 测试
    ├── store_test.go            # Store 测试
    ├── namespace_test.go        # Namespace 测试
    ├── blob_test.go             # Blob 测试
    ├── codec_test.go            # 序列化测试
    ├── compact_test.go          # 压缩测试
    ├── gc_test.go               # GC 测试
    ├── concurrent_test.go       # 并发测试
    ├── integration_test.go      # 集成测试
    └── benchmark_test.go        # 性能测试
```

### 8.2 模块依赖关系

```
┌─────────────────────────────────────────┐
│              stow.go (入口)              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│              store.go                    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           namespace.go                   │
│  ┌─────────────────────────────────┐    │
│  │  internal/codec   (序列化)      │    │
│  │  internal/index   (索引缓存)    │    │
│  │  internal/blob    (Blob 管理)   │    │
│  │  internal/compact (压缩/GC)     │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           internal/core                  │
│           internal/fsutil                │
└─────────────────────────────────────────┘
```

### 8.3 包职责说明

| 包路径 | 职责 | 对外暴露 |
|--------|------|---------|
| `stow` | 主包，定义所有公共接口 | ✅ |
| `internal/core` | JSONL 编解码、Record 结构 | ❌ |
| `internal/blob` | Blob 文件读写、引用管理 | ❌ |
| `internal/index` | 索引缓存、文件名映射 | ❌ |
| `internal/codec` | 序列化、Tag 解析 | ❌ |
| `internal/compact` | 压缩、GC 实现 | ❌ |
| `internal/fsutil` | 文件系统工具 | ❌ |

---

## 9. 使用场景与示例

### 9.1 场景一：应用配置管理

**需求**：存储应用配置，允许用户手动编辑

```go
type AppConfig struct {
    AppName    string
    Version    string
    Features   map[string]bool
    MaxWorkers int
}

func main() {
    store := stow.MustOpen("/data/myapp")
    ns, _ := store.CreateNamespace("config", stow.DefaultNamespaceConfig())
    
    // 首次写入
    cfg := AppConfig{
        AppName:    "MyApp",
        Version:    "1.0.0",
        Features:   map[string]bool{"beta": true},
        MaxWorkers: 10,
    }
    ns.MustPut("app", cfg)
    
    // 用户手动编辑 /data/myapp/config/app.jsonl 后
    ns.Refresh("app")
    
    // 重新读取
    var loaded AppConfig
    ns.MustGet("app", &loaded)
    fmt.Println(loaded.MaxWorkers) // 用户修改后的值
}
```

### 9.2 场景二：多媒体资源管理

**需求**：存储用户头像、简历等文件

```go
type User struct {
    Name       string
    Email      string
    Avatar     []byte         `stow:"file,name:avatar.jpg,mime:image/jpeg"`
    Resume     stow.IFileData `stow:"file,name_field:ResumeName"`
    ResumeName string
}

func main() {
    store := stow.MustOpen("/data/users")
    ns := store.MustGetNamespace("profiles").
        WithBlobThreshold(8 * 1024)
    
    // 上传用户资料
    avatarData, _ := os.ReadFile("avatar.jpg")
    resumeFile, _ := os.Open("resume.pdf")
    defer resumeFile.Close()
    
    user := User{
        Name:       "Alice",
        Email:      "alice@example.com",
        Avatar:     avatarData,
        Resume:     resumeFile,
        ResumeName: "alice_resume.pdf",
    }
    ns.MustPut("alice", user)
    
    // 读取用户
    var loaded User
    ns.MustGet("alice", &loaded)
    
    // 流式读取简历
    defer loaded.Resume.Close()
    io.Copy(os.Stdout, loaded.Resume)
}
```

### 9.3 场景三：版本历史追踪

**需求**：查看配置的修改历史

```go
func main() {
    store := stow.MustOpen("/data/myapp")
    ns := store.MustGetNamespace("config")
    
    // 多次修改
    ns.MustPut("server", map[string]interface{}{"port": 8080})
    time.Sleep(time.Second)
    ns.MustPut("server", map[string]interface{}{"port": 8081})
    time.Sleep(time.Second)
    ns.MustPut("server", map[string]interface{}{"port": 8082})
    
    // 查看历史
    history, _ := ns.GetHistory("server")
    for _, v := range history {
        fmt.Printf("Version %d at %s: %s\n", 
            v.Version, v.Timestamp, v.Operation)
    }
    
    // 读取特定版本
    var oldConfig map[string]interface{}
    ns.GetVersion("server", 1, &oldConfig)
    fmt.Println(oldConfig["port"]) // 8080
}
```

### 9.4 场景四：定期维护

**需求**：定期压缩和清理

```go
func main() {
    store := stow.MustOpen("/data/myapp")
    ns := store.MustGetNamespace("logs").
        WithLogger(&MyLogger{})
    
    // 定期任务
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        // 压缩所有 Key
        if err := ns.CompactAll(); err != nil {
            log.Printf("Compact failed: %v", err)
        }
        
        // 垃圾回收
        result, err := ns.GC()
        if err != nil {
            log.Printf("GC failed: %v", err)
        } else {
            log.Printf("GC: removed %d blobs, reclaimed %d bytes",
                result.RemovedBlobs, result.ReclaimedSize)
        }
    }
}
```

---

## 10. 验收标准

### 10.1 功能验收

| 功能 | 验收标准 |
|------|---------|
| **基础 KV** | ✅ Put/Get/Delete 正常工作<br>✅ 支持所有声明的数据类型<br>✅ List 返回正确的 Key 列表 |
| **文件可见性** | ✅ 存入数据后，能在文件系统找到对应 `.jsonl` 文件<br>✅ 文件内容可用文本编辑器打开<br>✅ `_meta` 和 `data` 字段完整 |
| **Blob 分离** | ✅ 大于阈值的数据存为独立文件<br>✅ `_blobs/` 目录下能找到文件<br>✅ JSONL 中只有引用结构 |
| **类型还原** | ✅ `io.Reader` 存入后，能通过 `IFileData` 读出<br>✅ `[]byte` 存入后，能还原为 `[]byte`<br>✅ Struct 存入后，能正确反序列化 |
| **手动编辑** | ✅ 手动修改 JSONL 最后一行<br>✅ 调用 `Refresh()` 或等待缓存过期<br>✅ `Get` 能读到修改后的值 |
| **命名空间隔离** | ✅ 不同 Namespace 的数据在不同目录<br>✅ 配置独立<br>✅ 互不干扰 |
| **历史版本** | ✅ `GetHistory()` 返回所有版本<br>✅ `GetVersion()` 能读取指定版本<br>✅ 历史 Blob 文件保留 |
| **压缩** | ✅ 手动压缩正常工作<br>✅ 自动压缩按配置触发<br>✅ 压缩后文件变小，历史记录减少 |
| **GC** | ✅ 能识别未引用的 Blob<br>✅ 正确删除孤立文件<br>✅ 返回准确的统计信息 |
| **并发安全** | ✅ 多 Goroutine 读写不 panic<br>✅ 数据不损坏<br>✅ 悲观锁保证一致性 |

### 10.2 性能验收

| 指标 | 目标 |
|------|------|
| **小数据写入** | < 1ms (不含 Sync) |
| **小数据读取** | < 0.5ms (缓存命中) |
| **大文件写入** | 流式处理，不 OOM |
| **大文件读取** | 流式处理，不 OOM |
| **List 操作** | < 10ms (1000 个 Key) |
| **Compact** | < 100ms (20 行记录) |
| **GC** | < 1s (1000 个 Blob) |

### 10.3 健壮性验收

| 场景 | 预期行为 |
|------|---------|
| **磁盘满** | 返回 `ErrDiskFull`，不损坏数据 |
| **权限不足** | 返回 `ErrPermissionDenied` |
| **JSONL 损坏** | 跳过错误行，读取有效记录 |
| **Blob 丢失** | 打印 Warn，字段设为零值 |
| **超大文件** | 返回 `ErrFileTooLarge`，清理部分写入 |
| **并发写入** | 悲观锁保证顺序执行 |
| **进程崩溃** | 重启后数据完整（已 Sync 的部分） |

### 10.4 文档验收

| 文档 | 要求 |
|------|------|
| **README** | ✅ 项目介绍<br>✅ 快速开始<br>✅ 安装说明 |
| **API 文档** | ✅ 所有公共接口有注释<br>✅ 参数说明完整<br>✅ 示例代码 |
| **设计文档** | ✅ 架构图<br>✅ 数据模型<br>✅ 实现细节 |
| **示例代码** | ✅ 覆盖主要场景<br>✅ 可直接运行 |

---

## 附录 A：术语表

| 术语 | 定义 |
|------|------|
| **Store** | Stow 的主入口，管理多个 Namespace |
| **Namespace** | 逻辑隔离的存储空间，对应一个目录 |
| **Key** | 数据的唯一标识符 |
| **JSONL** | Newline Delimited JSON，每行一个 JSON 对象 |
| **Blob** | 二进制大对象，存储为独立文件 |
| **Append-only** | 只追加不修改的写入模式 |
| **Last Write Wins** | 最后一次写入的值为有效值 |
| **Compact** | 压缩操作，合并历史记录 |
| **GC** | 垃圾回收，清理未引用的 Blob |
| **Lazy Loading** | 延迟加载，按需读取数据 |
| **IFileData** | 文件数据接口，支持流式读取 |

---

## 附录 B：配置示例

### 默认配置

```go
stow.DefaultNamespaceConfig()
```

等价于：

```go
NamespaceConfig{
    BlobThreshold:      4 * 1024,        // 4KB
    MaxFileSize:        100 * 1024 * 1024, // 100MB
    BlobChunkSize:      64 * 1024,       // 64KB
    CacheTTL:           5 * time.Minute,
    CacheTTLJitter:     0.2,             // ±20%
    DisableCache:       false,
    CompactStrategy:    CompactStrategyLineCount,
    CompactThreshold:   20,              // 20 行
    CompactKeepRecords: 3,               // 保留 3 条
    AutoCompact:        true,
    LockTimeout:        30 * time.Second,
}
```

### 高性能配置（牺牲历史记录）

```go
NamespaceConfig{
    BlobThreshold:      16 * 1024,       // 16KB
    CacheTTL:           10 * time.Minute,
    CompactThreshold:   10,
    CompactKeepRecords: 1,               // 只保留最新
    AutoCompact:        true,
}
```

### 低内存配置（禁用缓存）

```go
NamespaceConfig{
    DisableCache:       true,
    CompactThreshold:   5,
    AutoCompact:        true,
}
```

---

## 附录 C：错误处理最佳实践

### 1. 使用 Must* 系列函数

```go
// 适用于初始化阶段，错误应该 panic
store := stow.MustOpen("/data")
ns := store.MustGetNamespace("config")
ns.MustPut("key", value)
```

### 2. 使用普通函数

```go
// 适用于运行时，需要优雅处理错误
if err := ns.Put("key", value); err != nil {
    if errors.Is(err, stow.ErrDiskFull) {
        // 磁盘满，触发清理
        ns.GC()
    } else {
        log.Printf("Put failed: %v", err)
    }
}
```

### 3. 检查特定错误

```go
if err := ns.Get("key", &target); err != nil {
    if errors.Is(err, stow.ErrNotFound) {
        // Key 不存在，使用默认值
        target = defaultValue
    } else {
        return err
    }
}
```
