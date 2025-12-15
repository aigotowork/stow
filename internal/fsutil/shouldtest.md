# FSUtil 模块测试改进建议

## 当前状态
- **覆盖率**: 56.8%
- **问题**: 所有测试集中在一个文件，许多边界情况和错误处理未测试

## 测试文件重构建议

### 建议的测试文件结构

```
internal/fsutil/
├── atomic.go           -> atomic_test.go
├── safe.go             -> safe_test.go
├── walk.go             -> walk_test.go
├── permissions_test.go (新增 - 权限相关测试)
└── integration_test.go (新增 - 集成测试)
```

---

## 1. Atomic 操作测试 (atomic_test.go)

### 当前覆盖情况
- ⚠️ AtomicWriteFile: 45.8% (错误分支未覆盖)
- ✅ SafeRename: 100%
- ⚠️ syncDir: 80%

### 需要添加的测试场景

#### AtomicWriteFile 测试
```go
TestAtomicWriteFileBasic
  - 正常写入
  - 覆盖现有文件
  - 创建新文件

TestAtomicWriteFilePermissions
  - 指定文件权限 (0644, 0600, 0755)
  - 权限继承
  - 权限验证

TestAtomicWriteFileErrors
  - 目录不存在
  - 权限不足
  - 磁盘空间不足 (模拟)
  - 临时文件创建失败
  - 写入过程中断
  - Rename 失败
  - Sync 失败

TestAtomicWriteFileConcurrency
  - 并发写入同一文件
  - 并发写入不同文件
  - 写入冲突处理

TestAtomicWriteFileEdgeCases
  - 空内容
  - 大文件 (> 100MB)
  - 特殊字符文件名
  - 符号链接目标
```

#### SafeRename 测试
```go
TestSafeRenameEdgeCases
  - 源文件不存在
  - 目标已存在
  - 跨文件系统 rename
  - 符号链接处理

TestSafeRenameConcurrency
  - 并发 rename 操作
  - Rename 竞争条件
```

#### SyncDir 测试
```go
TestSyncDirErrors
  - 目录不存在
  - 权限不足
  - 不是目录
  - Sync 失败处理

TestSyncDirTypes
  - 普通目录
  - 空目录
  - 嵌套目录
```

---

## 2. Safe 操作测试 (safe_test.go)

### 当前覆盖情况
- ⚠️ EnsureDir: 70% (部分错误分支未覆盖)
- ✅ FileExists, DirExists: 100%
- ❌ FileSize: 0%
- ⚠️ RemoveAll: 75%
- ❌ CleanPath, AbsPath: 0%

### 需要添加的测试场景

#### EnsureDir 测试
```go
TestEnsureDirBasic
  - 创建单级目录
  - 创建多级目录
  - 目录已存在
  - 权限设置

TestEnsureDirErrors
  - 路径是文件而非目录
  - 权限不足
  - 父目录不可写
  - 路径过长
  - 特殊字符路径

TestEnsureDirConcurrency
  - 并发创建同一目录
  - 并发创建父子目录
```

#### FileSize 测试
```go
TestFileSizeBasic
  - 空文件 (0 字节)
  - 小文件 (< 1KB)
  - 中等文件 (1-100MB)
  - 大文件 (> 1GB)

TestFileSizeErrors
  - 文件不存在
  - 路径是目录
  - 权限不足
  - 符号链接处理
```

#### CleanPath 和 AbsPath 测试
```go
TestCleanPath
  - 相对路径 (./path, ../path)
  - 绝对路径
  - 多余斜杠 (//path, path//)
  - . 和 .. 处理
  - 空路径
  - 根路径

TestAbsPath
  - 相对路径转绝对路径
  - 绝对路径保持不变
  - ~/ 展开 (如果支持)
  - 符号链接解析
  - 错误处理 (不存在的路径)
```

#### RemoveAll 测试
```go
TestRemoveAllEdgeCases
  - 空目录
  - 嵌套目录
  - 包含文件的目录
  - 符号链接
  - 只读文件/目录
  - 不存在的路径

TestRemoveAllErrors
  - 权限不足
  - 正在使用的文件
  - 部分删除失败
```

---

## 3. Walk 操作测试 (walk_test.go)

### 当前覆盖情况
- ✅ Walk: 100%
- ⚠️ ListFiles: 87.5%
- ⚠️ ListDirs: 87.5%
- ❌ WalkFilesWithExt: 0%
- ⚠️ FindFiles: 80%
- ⚠️ DirSize: 87.5%
- ❌ CountFiles: 0%
- ✅ IsHidden: 100%
- ❌ FilterHidden: 0%

### 需要添加的测试场景

#### WalkFilesWithExt 测试
```go
TestWalkFilesWithExt
  - 单个扩展名
  - 多个扩展名
  - 大小写敏感/不敏感
  - 无扩展名文件
  - 多级目录遍历

TestWalkFilesWithExtEdgeCases
  - 空目录
  - 没有匹配文件
  - 扩展名格式 (".txt" vs "txt")
  - 特殊扩展名 (".tar.gz")
```

#### CountFiles 测试
```go
TestCountFiles
  - 空目录
  - 单级目录
  - 多级目录
  - 包含子目录
  - 符号链接处理

TestCountFilesFiltered
  - 按扩展名过滤
  - 按模式过滤
  - 排除隐藏文件
```

#### FilterHidden 测试
```go
TestFilterHidden
  - Unix 隐藏文件 (.file)
  - Windows 隐藏文件
  - 隐藏目录
  - 混合场景

TestFilterHiddenEdgeCases
  - 空列表
  - 全部隐藏
  - 无隐藏文件
  - .和 .. 处理
```

#### ListFiles 和 ListDirs 增强测试
```go
TestListFilesErrors
  - 目录不存在
  - 权限不足
  - 不是目录

TestListDirsRecursive
  - 递归列出子目录
  - 目录深度限制
  - 循环符号链接处理

TestListFilesPatterns
  - Glob 模式匹配
  - 正则表达式匹配
  - 多个模式
```

#### FindFiles 完善测试
```go
TestFindFilesPatterns
  - 简单模式 (*.txt)
  - 复杂模式 (**/test_*.go)
  - 多个模式
  - 排除模式

TestFindFilesDepth
  - 深度限制
  - 递归搜索
  - 跟随符号链接

TestFindFilesPerformance
  - 大目录树 (1000+ 文件)
  - 深层嵌套 (10+ 层)
```

#### DirSize 完善测试
```go
TestDirSizeEdgeCases
  - 空目录 (应该是 0)
  - 符号链接大小
  - 稀疏文件
  - 硬链接处理

TestDirSizeErrors
  - 权限不足的子目录
  - 计算过程中文件被删除
```

---

## 4. 权限测试 (permissions_test.go)

### 新增权限相关测试

```go
TestPermissionsPreservation
  - 创建文件保持权限
  - 复制文件保持权限
  - Rename 保持权限

TestPermissionsChange
  - chmod 操作
  - chown 操作 (如果支持)
  - 权限继承

TestPermissionsDenied
  - 读取权限不足
  - 写入权限不足
  - 执行权限不足
  - 遍历权限不足

TestUmask
  - 默认 umask 行为
  - 自定义 umask
```

---

## 5. 集成测试 (integration_test.go)

### 完整流程测试

```go
TestAtomicFileOperations
  - 原子写入 -> 读取验证
  - 原子写入 -> 重命名 -> 验证
  - 并发原子写入互不干扰

TestDirectoryManagement
  - 创建目录树
  - 填充文件
  - 遍历和验证
  - 清理

TestFileSystemResilience
  - 磁盘满模拟
  - 权限错误恢复
  - 部分写入回滚

TestSymlinksHandling
  - 创建符号链接
  - 跟随符号链接
  - 检测循环链接
  - 断开的链接处理
```

### 并发测试

```go
TestConcurrentFileOperations
  - 并发读写不同文件
  - 并发读同一文件
  - 并发写同一文件 (应该安全)

TestConcurrentDirectoryOps
  - 并发创建目录
  - 并发删除目录
  - 并发遍历
```

---

## 特殊场景测试

### 跨平台测试
```go
TestCrossPlatformPaths
  - Windows 路径 (C:\path)
  - Unix 路径 (/path)
  - UNC 路径 (\\server\share)
  - 路径分隔符转换

TestCrossPlatformLineEndings
  - LF (\n)
  - CRLF (\r\n)
  - CR (\r)
```

### 特殊文件系统测试
```go
TestSpecialFileSystems
  - tmpfs/ramfs 行为
  - 网络文件系统
  - 只读文件系统
  - Case-insensitive 文件系统
```

---

## 优先级建议

### 🔴 高优先级 (立即添加)
1. FileSize 测试 (当前 0%)
2. CleanPath 和 AbsPath 测试 (当前 0%)
3. WalkFilesWithExt, CountFiles, FilterHidden 测试 (当前 0%)
4. AtomicWriteFile 错误处理测试 (提升至 90%+)

### 🟡 中优先级 (第二阶段)
1. EnsureDir 错误场景
2. RemoveAll 边界测试
3. 权限相关测试
4. 并发测试

### 🟢 低优先级 (优化阶段)
1. 性能测试
2. 跨平台测试
3. 特殊文件系统测试

---

## 预期效果

实施上述测试改进后：
- **覆盖率目标**: 56.8% → **85%+**
- **测试文件数**: 1 → 5
- **测试用例数**: ~20 → **70+**

---

## 实施建议

### 第一阶段：重构和补充基础测试
1. 拆分测试文件
2. 添加未覆盖功能的测试
3. 补充错误处理测试

### 第二阶段：增强测试
1. 添加并发测试
2. 添加权限测试
3. 添加集成测试

### 第三阶段：跨平台测试
1. Windows 特定测试
2. Unix 特定测试
3. 跨平台兼容性测试

---

## 测试最佳实践

### 使用测试辅助函数
```go
// 创建临时测试目录
func setupTestDir(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    return dir
}

// 创建测试文件
func createTestFile(t *testing.T, dir, name string, content []byte) string {
    t.Helper()
    path := filepath.Join(dir, name)
    err := os.WriteFile(path, content, 0644)
    require.NoError(t, err)
    return path
}

// 验证文件内容
func assertFileContent(t *testing.T, path string, expected []byte) {
    t.Helper()
    actual, err := os.ReadFile(path)
    require.NoError(t, err)
    assert.Equal(t, expected, actual)
}
```

### Table-Driven Tests
```go
func TestCleanPath(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"relative", "./path", "path"},
        {"absolute", "/path", "/path"},
        {"double slash", "//path", "/path"},
        {"parent", "../path", "../path"},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CleanPath(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 错误断言
```go
func TestFileOperationErrors(t *testing.T) {
    // 使用明确的错误检查
    err := someOperation()
    require.Error(t, err)
    assert.Contains(t, err.Error(), "expected message")

    // 或使用类型断言
    var pathErr *os.PathError
    assert.ErrorAs(t, err, &pathErr)
}
```
