# Stow 架构设计

## 架构概览

Stow 采用**分层架构**设计，从底层文件系统到顶层 API，每层职责清晰。

```
┌─────────────────────────────────────────────┐
│           用户应用 (User Application)        │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         公共 API (Public API)               │
│  Store, Namespace, PutOption, IFileData     │
└──────────────┬──────────────────────────────┘
               │
     ┌─────────┴─────────┐
     │                   │
┌────▼─────┐      ┌─────▼────┐
│  Store   │      │Namespace │  ← 核心引擎
└────┬─────┘      └─────┬────┘
     │                  │
     │    ┌─────────────┴──────────────┐
     │    │             │               │
     │ ┌──▼───┐    ┌───▼────┐    ┌────▼────┐
     │ │Index │    │ Codec  │    │  Blob   │
     │ │Mapper│    │Marshal │    │ Manager │
     │ └──┬───┘    └───┬────┘    └────┬────┘
     │    │            │               │
     │    │    ┌───────┴───────┐       │
     │    │    │               │       │
     │ ┌──▼────▼──┐     ┌─────▼───┐   │
     │ │  Core    │     │  Blob   │   │
     │ │ Encoder  │     │ Writer  │   │
     │ │ Decoder  │     │ Reader  │   │
     │ └────┬─────┘     └─────┬───┘   │
     │      │                 │       │
     │      └─────────┬───────┘       │
     │                │               │
     │         ┌──────▼───────┐       │
     │         │   FSUtil     │       │
     │         │  (文件工具)   │       │
     │         └──────┬───────┘       │
     │                │               │
     └────────────────┼───────────────┘
                      │
         ┌────────────▼─────────────┐
         │   操作系统文件系统         │
         │  (OS Filesystem)         │
         └──────────────────────────┘
```

## 分层说明

### L0: 文件系统层 (FSUtil)

**职责：** 封装文件系统操作

**核心功能：**
- 原子文件写入（Sync + Rename）
- 安全文件操作（权限检查）
- 目录遍历和搜索
- 文件大小计算

**依赖：** 无（底层）

**为什么需要？**
- 抽象平台差异（Windows/Linux/Mac）
- 提供原子性保证
- 统一错误处理

---

### L1: 数据层 (Core + Blob)

#### Core - JSONL 引擎

**职责：** JSONL 格式的编解码

**核心功能：**
- Record 编码/解码
- 反向读取（Last Write Wins）
- 版本管理
- 追加写入

**依赖：** FSUtil

**为什么设计成独立模块？**
- JSONL 是核心数据格式，需要独立测试
- 可能被其他组件复用（未来）
- 职责单一，易于维护

#### Blob - 大文件管理

**职责：** 独立存储大文件

**核心功能：**
- 流式读写（避免 OOM）
- SHA256 哈希计算
- 文件命名和索引
- 垃圾回收

**依赖：** FSUtil

**为什么分离？**
- 避免 JSONL 膨胀
- 支持流式处理
- 内容寻址（未来去重）

---

### L2: 转换层 (Index + Codec)

#### Index - 索引和缓存

**职责：** 加速 Key 查找

**核心功能：**
- Key → 文件名映射
- Key 清洗（非法字符）
- 冲突检测和处理
- TTL 缓存（防惊群）

**依赖：** Core, FSUtil

**为什么需要？**
- Key 可能包含非法字符
- 避免每次 Get 都扫描目录
- 缓存减少磁盘 I/O

#### Codec - 序列化

**职责：** Go 对象 ↔ map 转换

**核心功能：**
- Struct Tag 解析
- 自动 Blob 检测
- Marshal/Unmarshal
- 类型转换

**依赖：** Blob

**为什么需要？**
- Core 只处理 map，需要转换层
- Blob 检测需要反射分析
- 提供 Struct Tag 灵活性

---

### L3: 引擎层 (Namespace)

**职责：** 核心 KV 引擎

**核心功能：**
- Put/Get/Delete 操作
- 版本历史追踪
- Compact 和 GC
- 并发控制（RWMutex）

**依赖：** Index, Codec, Core, Blob

**为什么是核心？**
- 整合所有底层模块
- 实现 KV 语义
- 提供用户 API

---

### L4: 管理层 (Store)

**职责：** 多 Namespace 管理

**核心功能：**
- Namespace 创建/删除
- Namespace 缓存
- 延迟加载

**依赖：** Namespace

**为什么需要？**
- 隔离不同数据集
- 统一管理入口
- 懒加载优化

---

## 数据流

### Put 操作完整流程

```
1. 用户调用
   ns.Put("user", userStruct, WithForceFile())

2. Namespace.Put
   ↓ 加写锁
   ↓ 应用 PutOption

3. Codec.Marshal
   ↓ 反射分析 userStruct
   ↓ 检测 Blob 字段（Avatar []byte）
   ↓ 调用 Blob.Store(avatarData)

4. Blob.Store
   ↓ 计算 SHA256 哈希
   ↓ 生成文件名: avatar_a3f2c1.jpg
   ↓ 分块写入 _blobs/avatar_a3f2c1.jpg
   ↓ 返回 BlobReference

5. Codec.Marshal (继续)
   ↓ 替换 Avatar 字段为 BlobReference
   ↓ 返回 map[string]interface{}

6. Namespace.Put (继续)
   ↓ 获取下一个版本号（v2）
   ↓ 创建 Record{_meta, data}

7. Core.AppendRecord
   ↓ 编码为 JSON
   ↓ 追加到 user.jsonl
   ↓ Sync 到磁盘

8. Namespace.Put (继续)
   ↓ 更新 KeyMapper 索引
   ↓ 更新 Cache
   ↓ 解锁
   ↓ 返回 nil (success)
```

**时序图：**
```
User     Namespace    Codec      Blob     Core     Disk
 │           │          │         │        │        │
 ├─Put()────>│          │         │        │        │
 │           ├─Marshal()>│        │        │        │
 │           │          ├─Store()>│        │        │
 │           │          │         ├─Write────────>  │
 │           │          │<────────┤        │        │
 │           │<─────────┤         │        │        │
 │           ├─Append()──────────>│        │        │
 │           │          │         │        ├─Write─>│
 │           │<──────────────────────────  │        │
 │<──────────┤          │         │        │        │
```

### Get 操作完整流程

```
1. 用户调用
   var user User
   ns.Get("user", &user)

2. Namespace.Get
   ↓ 加读锁
   ↓ 检查 Cache（未命中）

3. Index.Find
   ↓ SanitizeKey("user") = "user"
   ↓ 查找 KeyMapper
   ↓ 返回 [user.jsonl]

4. Core.ReadLastValid
   ↓ 打开 user.jsonl
   ↓ 反向读取
   ↓ 找到最后一条 op="put"
   ↓ 解码 Record
   ↓ 返回 map{_meta, data}

5. Codec.Unmarshal
   ↓ 遍历 data 字段
   ↓ 发现 Avatar 是 BlobReference
   ↓ 调用 Blob.Load(ref)

6. Blob.Load
   ↓ 检查文件存在
   ↓ 返回 IFileData（不读取内容）

7. Codec.Unmarshal (继续)
   ↓ 根据目标类型：
   ↓   []byte → LoadBytes()
   ↓   IFileData → 返回句柄
   ↓ 反射填充 &user

8. Namespace.Get (继续)
   ↓ 更新 Cache
   ↓ 解锁
   ↓ 返回 nil (success)
```

---

## 并发模型

### 锁策略

```go
type namespace struct {
    mu sync.RWMutex  // Namespace 级别锁
}

// 读操作（可并发）
Get, Exists, List, GetHistory
    → RLock()

// 写操作（独占）
Put, Delete, Compact, GC
    → Lock()
```

**锁粒度：**
- 当前：Namespace 级别
- 未来改进：Key 级别（见 couldbebetter.md）

### 并发场景分析

**场景 1：多读并发** ✅
```go
go ns.Get("key1", &v1)
go ns.Get("key2", &v2)
go ns.Get("key3", &v3)
// 全部并发执行，无阻塞
```

**场景 2：读写并发** ⚠️
```go
go ns.Get("key1", &v1)  // 读锁
go ns.Put("key2", v2)   // 等待读锁释放
// 写操作阻塞所有读操作
```

**场景 3：多写并发** ❌
```go
go ns.Put("key1", v1)  // 写锁
go ns.Put("key2", v2)  // 等待
go ns.Put("key3", v3)  // 等待
// 完全串行化
```

### 改进方向

**Key 级锁（未来）：**
```go
type namespace struct {
    keyLocks sync.Map  // key → *sync.Mutex
}

func (ns *namespace) Put(key string, ...) {
    lock := ns.getKeyLock(key)
    lock.Lock()
    defer lock.Unlock()
    // 只锁当前 Key
}
```

**优势：**
- 不同 Key 可并发写入
- 性能提升 10x+

---

## 故障处理

### 写入故障

**问题：** 写入过程中断电/崩溃

**保护机制：**

1. **原子写入（FSUtil）**
   ```
   Write → Sync → Rename
   ```
   - 最多丢失一次写入
   - 不会出现部分写入

2. **追加写入（Core）**
   ```
   只追加，不修改已有数据
   ```
   - 旧数据不受影响
   - 最多损坏最后一行

3. **容错解码（Core）**
   ```go
   // 跳过错误行
   for scanner.Scan() {
       record, err := Decode(line)
       if err != nil {
           continue  // 跳过
       }
   }
   ```
   - 部分损坏不影响整体

### Blob 文件丢失

**问题：** 用户手动删除 Blob

**处理：**
```go
// Unmarshal 时检查
if !blobManager.Exists(ref) {
    log.Warn("blob missing", ref.Location)
    field.Set(reflect.Zero(field.Type()))  // 零值
    return nil  // 不返回错误
}
```

**策略：**
- ⚠️ 警告日志
- ✅ 其他字段正常加载
- ✅ 不阻塞 Get 操作

### 索引损坏

**问题：** KeyMapper 索引与实际文件不一致

**恢复：**
```go
// Refresh 操作重建索引
func (ns *namespace) RefreshAll() error {
    scanner := index.NewScanner()
    newMapper, _ := scanner.ScanNamespace(ns.path)
    ns.keyMapper = newMapper  // 替换
    ns.cache.Clear()          // 清空缓存
    return nil
}
```

---

## 性能特征

### 时间复杂度总结

| 操作 | 时间复杂度 | 瓶颈 |
|------|------------|------|
| **Put** | O(1) | 磁盘 Sync |
| **Get (缓存命中)** | O(1) | 内存 |
| **Get (缓存未命中)** | O(k) | k = 尾部行数 |
| **Delete** | O(1) | 磁盘 Sync |
| **List** | O(n) | n = 文件数 |
| **GetHistory** | O(m) | m = 总版本数 |
| **Compact** | O(m) | m = 保留版本数 |
| **GC** | O(n + b) | n = 记录数, b = Blob 数 |

### 吞吐量估算

**测试环境：** M1 MacBook Pro, SSD

```
Put (小数据):      ~500 ops/s   (受 Sync 限制)
Get (缓存命中):    ~100K ops/s  (内存)
Get (缓存未命中):  ~1K ops/s    (磁盘)
Delete:            ~500 ops/s   (受 Sync 限制)
```

**优化方向：**
1. 批量 Sync → 提高写入吞吐
2. 预读索引 → 减少首次访问延迟
3. Blob 哈希索引 → 加速去重

---

## 扩展性分析

### 单 Namespace 限制

**推荐配置：**
```
- Keys:      < 10,000 条
- 单 Key 历史:  < 1,000 版本
- Blob 数量:   < 10,000 个
- 总数据量:    < 1GB
```

**扩展方案：**

**方案 1：** 多 Namespace 分片
```go
// 按用户 ID 分片
userID := 12345
nsName := fmt.Sprintf("users_%d", userID%100)
ns := store.GetNamespace(nsName)
```

**方案 2：** 定期归档
```go
// 将旧数据移到归档 Namespace
archiveNS := store.GetNamespace("archive_2024")
for _, key := range oldKeys {
    data, _ := activeNS.Get(key)
    archiveNS.Put(key, data)
    activeNS.Delete(key)
}
```

### 多进程访问

**当前限制：** 仅支持单进程

**未来支持：**
1. 文件锁（flock）
2. 进程间缓存失效
3. Watch 文件变化

---

## 设计原则

### 1. 简单优先 (Simplicity First)

**体现：**
- 无 SQL，无查询语言
- 无索引，全表扫描
- 无事务，单操作原子性

### 2. 透明可见 (Transparency)

**体现：**
- JSONL 人类可读
- Blob 独立文件
- 可用 grep/awk 搜索

### 3. 延迟加载 (Lazy Loading)

**体现：**
- Namespace 按需创建
- 索引首次使用时构建
- Blob 返回句柄而非内容

### 4. 容错优先 (Fault Tolerance)

**体现：**
- 解码跳过错误行
- Blob 缺失不返回错误
- 部分损坏不影响整体

---

## 与其他存储对比

| 特性 | Stow | SQLite | JSON 文件 | Redis |
|------|------|--------|-----------|-------|
| **类型** | KV | SQL | 文本 | 内存 KV |
| **可读性** | ✅ 高 | ❌ 二进制 | ✅ 高 | ❌ 内存 |
| **查询** | ❌ 无 | ✅ SQL | ❌ 无 | ⚠️ 有限 |
| **版本历史** | ✅ 内置 | ❌ 需设计 | ❌ 无 | ❌ 无 |
| **大文件** | ✅ 流式 | ⚠️ Blob | ❌ Base64 | ❌ 不适合 |
| **并发** | ⚠️ 读多写少 | ✅ 事务 | ❌ 文件锁 | ✅ 高并发 |
| **适用场景** | 配置/媒体 | 通用 | 简单数据 | 缓存 |

---

## 总结

**Stow 的定位：**

```
纯文件 ← → Stow ← → SQLite ← → PostgreSQL
 简单         中等           复杂
```

**何时选择 Stow：**
- ✅ 需要人工审查数据
- ✅ 有大文件存储需求
- ✅ 需要版本历史
- ✅ 读多写少
- ❌ 不需要复杂查询
- ❌ 不需要高并发写入

**未来方向：**
- Key 级并发锁
- Blob 内容去重
- WAL 支持
- 多进程访问

详见 [设计反思](couldbebetter.md)
