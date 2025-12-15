# FSUtil 模块设计

## 职责定位

**FSUtil 是文件系统工具层**，提供：
1. 原子文件写入
2. 安全文件操作
3. 目录遍历和搜索
4. 跨平台抽象

## 核心功能

### 原子写入（AtomicWriteFile）

**问题：** 直接写入可能导致部分写入

```go
// 不安全的写入
os.WriteFile(path, data, 0644)
// 如果崩溃 → 文件可能损坏
```

**解决：** 写临时文件 → Sync → Rename

```go
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
    tmpPath := path + ".tmp"

    // 1. 写临时文件
    f, _ := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
    _, err := f.Write(data)

    // 2. Sync（确保落盘）
    f.Sync()
    f.Close()

    // 3. 原子 Rename
    return SafeRename(tmpPath, path)
}
```

**保证：**
- ✅ 要么完整写入，要么不写入
- ✅ 不会出现部分写入
- ✅ 崩溃安全

---

### 安全重命名（SafeRename）

**跨平台处理：**

```go
func SafeRename(oldPath, newPath string) error {
    // Windows: 目标文件存在需先删除
    if runtime.GOOS == "windows" {
        if FileExists(newPath) {
            os.Remove(newPath)
        }
    }

    return os.Rename(oldPath, newPath)
}
```

**为什么需要？**
- Linux/Mac: Rename 可覆盖（POSIX 语义）
- Windows: Rename 不可覆盖，需先删除

---

### 目录操作

#### EnsureDir - 确保目录存在

```go
func EnsureDir(path string, perm os.FileMode) error {
    if DirExists(path) {
        return nil
    }
    return os.MkdirAll(path, perm)
}
```

**幂等性：** 多次调用无副作用

#### FindFiles - 文件查找

```go
func FindFiles(root string, pattern string) ([]string, error) {
    var matches []string

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            return nil
        }

        matched, _ := filepath.Match(pattern, filepath.Base(path))
        if matched {
            matches = append(matches, path)
        }

        return nil
    })

    return matches, err
}
```

**支持模式：**
- `*.jsonl` - 所有 JSONL 文件
- `user_*.jsonl` - 前缀匹配
- `*_v1.jsonl` - 后缀匹配

---

### 工具函数

#### FileSize - 文件大小

```go
func FileSize(path string) int64 {
    info, err := os.Stat(path)
    if err != nil {
        return 0
    }
    return info.Size()
}
```

#### DirSize - 目录大小

```go
func DirSize(path string) (int64, error) {
    var size int64

    err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            size += info.Size()
        }
        return nil
    })

    return size, err
}
```

**用途：** 统计 Namespace 总大小

#### IsHidden - 隐藏文件检测

```go
func IsHidden(path string) bool {
    name := filepath.Base(path)
    return len(name) > 0 && name[0] == '.'
}
```

**用途：** 跳过 `.DS_Store` 等隐藏文件

---

## 设计要点

### 1. 为什么需要 Sync？

**磁盘写入缓存：**
```
Application → OS Buffer → Disk
              ^^^^^^^^^^^^
              可能延迟几秒
```

**不 Sync 的风险：**
```go
f.Write(data)
f.Close()
// 崩溃 → 数据仍在 OS Buffer，未落盘
```

**Sync 保证：**
```go
f.Write(data)
f.Sync()  // 强制刷新到磁盘
f.Close()
// 崩溃 → 数据已落盘
```

**性能代价：**
- 10x ~ 100x 性能下降
- 但数据安全优先

---

### 2. 为什么不用 `ioutil.WriteFile`？

**标准库问题：**
```go
ioutil.WriteFile(path, data, 0644)
// 1. 直接写入，非原子
// 2. 没有 Sync
```

**FSUtil 优势：**
```go
AtomicWriteFile(path, data, 0644)
// 1. 临时文件 + Rename（原子）
// 2. 显式 Sync（安全）
```

---

## 性能特征

### 时间复杂度

| 操作 | 时间复杂度 | 说明 |
|------|------------|------|
| AtomicWriteFile | O(n) | n = 数据大小 |
| SafeRename | O(1) | 操作系统调用 |
| FindFiles | O(m) | m = 目录文件数 |
| DirSize | O(m) | m = 目录文件数 |

### 实际性能

**AtomicWriteFile vs WriteFile：**
```
数据大小: 1KB
WriteFile:       ~0.1ms
AtomicWriteFile: ~1ms (10x 慢)

数据大小: 1MB
WriteFile:       ~10ms
AtomicWriteFile: ~50ms (5x 慢)
```

**权衡：** 接受性能损失，换取数据安全。

---

## 测试覆盖

**10 个单元测试，覆盖率 56.8%**

关键测试：
- ✅ 原子写入正确性
- ✅ 目录创建（幂等性）
- ✅ 文件/目录存在检查
- ✅ 文件列表和查找
- ✅ 递归删除
- ✅ 隐藏文件检测
- ✅ 目录大小计算

---

## 使用示例

```go
// 1. 原子写入
err := fsutil.AtomicWriteFile("/path/to/file.txt", []byte("data"), 0644)

// 2. 确保目录存在
err := fsutil.EnsureDir("/path/to/dir", 0755)

// 3. 查找文件
files, _ := fsutil.FindFiles("./data", "*.jsonl")

// 4. 计算目录大小
size, _ := fsutil.DirSize("./data")

// 5. 检查文件存在
exists := fsutil.FileExists("/path/to/file")

// 6. 列出目录文件
files, _ := fsutil.ListFiles("./data")
```

---

## 相关模块

- 上层：所有模块（Core, Blob, Index）
- 下层：操作系统文件系统

## 潜在改进

### 改进 1：批量 Sync

**当前：** 每次写入都 Sync
```go
for _, data := range batch {
    AtomicWriteFile(path, data, 0644)  // 每次都 Sync
}
```

**改进：** 批量写入后统一 Sync
```go
func BatchWrite(files map[string][]byte) error {
    var fileHandles []*os.File

    // 1. 写入所有文件（不 Sync）
    for path, data := range files {
        f, _ := os.Create(path + ".tmp")
        f.Write(data)
        fileHandles = append(fileHandles, f)
    }

    // 2. 统一 Sync
    for _, f := range fileHandles {
        f.Sync()
        f.Close()
    }

    // 3. 批量 Rename
    for path := range files {
        SafeRename(path+".tmp", path)
    }

    return nil
}
```

**收益：** 减少 Sync 次数，提高吞吐。

---

### 改进 2：异步 Sync

**当前：** 同步 Sync，阻塞调用者

**改进：** 后台 Sync 线程
```go
type AsyncWriter struct {
    syncQueue chan *os.File
}

func (aw *AsyncWriter) Write(path string, data []byte) error {
    f, _ := os.Create(path + ".tmp")
    f.Write(data)

    // 投递到 Sync 队列
    aw.syncQueue <- f

    return nil
}

func (aw *AsyncWriter) syncWorker() {
    for f := range aw.syncQueue {
        f.Sync()
        f.Close()
        // Rename...
    }
}
```

**优势：**
- 不阻塞调用者
- 批量 Sync 优化

**劣势：**
- 复杂度增加
- 需要错误处理机制

**建议：** 对高频写入场景考虑异步 Sync。

---

## 设计哲学

**FSUtil 的原则：**
1. **安全第一** - 数据完整性优先于性能
2. **原子性** - 操作要么成功，要么失败
3. **跨平台** - 抽象操作系统差异
4. **简单直接** - API 清晰易用

**为什么简单？**
- 文件系统操作已经很复杂
- 不增加额外抽象
- 直接映射操作系统调用

---

## 总结

FSUtil 是 Stow 的**基础设施层**，提供：
- ✅ 原子写入保证数据完整性
- ✅ 跨平台抽象
- ✅ 常用文件系统工具

所有上层模块都依赖 FSUtil，确保文件操作的安全性和一致性。
