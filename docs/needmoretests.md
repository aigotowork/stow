# 需要更多测试的领域

## 测试现状总结

截至 2024-12-14，测试覆盖情况：

### ✅ 测试良好的模块
- **internal/blob** - 16 tests (去重、哈希、文件管理)
- **internal/codec** - 16 tests (序列化、反序列化、嵌套结构体)
- **internal/core** - 14 tests (JSONL 编解码、反向读取)
- **internal/fsutil** - 10 tests (原子写入、安全操作)
- **internal/index** - 17 tests (Key 清洗、缓存、TTL)
- **tests/concurrent_test.go** - 3 tests (并发读写)
- **tests/compact_async_test.go** - 4 tests (异步压缩)
- **tests/nested_struct_test.go** - 6 tests (嵌套结构体)

### ⚠️ 需要加强的领域

---

## 1. 错误处理与边界条件

### 1.1 文件系统错误

**当前状态**: 缺少系统级错误处理测试

**需要测试的场景**:
```go
// 磁盘空间不足
func TestPutWithDiskFull(t *testing.T) {
    // 模拟磁盘满，验证错误处理
    // 预期：返回 ErrDiskFull，不留部分写入文件
}

// 权限不足
func TestPutWithPermissionDenied(t *testing.T) {
    // 创建只读目录
    // 预期：返回 ErrPermissionDenied
}

// 文件损坏
func TestGetWithCorruptedJSONL(t *testing.T) {
    // 手工创建损坏的 JSONL 文件
    // 预期：跳过损坏行，返回有效数据或 NotFound
}

// 文件锁冲突（多进程场景）
func TestConcurrentAccessAcrossProcesses(t *testing.T) {
    // 两个进程同时写入同一 Key
    // 预期：正确串行化，数据一致
}
```

**优先级**: 🟡 P1
**影响**: 生产环境错误恢复

---

### 1.2 边界值测试

**当前状态**: 缺少极限值测试

**需要测试的场景**:
```go
// 超大 Key
func TestPutWithVeryLongKey(t *testing.T) {
    key := strings.Repeat("a", 10000) // 10KB Key
    // 预期：Key 清洗后仍可用，或返回错误
}

// 空值
func TestPutEmptyValue(t *testing.T) {
    ns.Put("key", map[string]interface{}{})
    ns.Put("key2", "")
    ns.Put("key3", nil)
    // 预期：正确存储和恢复
}

// 最大文件大小
func TestBlobAtMaxFileSize(t *testing.T) {
    data := make([]byte, config.MaxFileSize)
    // 预期：成功存储
}

func TestBlobExceedsMaxFileSize(t *testing.T) {
    data := make([]byte, config.MaxFileSize+1)
    // 预期：返回 ErrFileTooLarge
}

// 大量版本历史
func TestKeyWith10000Versions(t *testing.T) {
    for i := 0; i < 10000; i++ {
        ns.Put("key", i)
    }
    // 验证：Get 性能、Compact 正确性、内存占用
}
```

**优先级**: 🟢 P2
**影响**: 极限场景稳定性

---

## 2. 并发与竞态条件

### 2.1 高并发压力测试

**当前状态**: 有基础并发测试，缺少压力测试

**需要测试的场景**:
```go
// 100 并发写入
func TestHighConcurrencyWrites(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            ns.Put(fmt.Sprintf("key%d", id), data)
        }(i)
    }
    wg.Wait()
    // 验证：所有数据正确写入，无数据丢失
}

// 读写混合高并发
func TestMixedReadWriteLoad(t *testing.T) {
    // 50 readers + 50 writers 同时运行 10 秒
    // 验证：无 panic，数据一致性
}

// 压缩期间的并发操作
func TestCompactDuringHighLoad(t *testing.T) {
    // 高并发读写的同时执行 Compact
    // 验证：Compact 不影响读写，数据正确
}

// Cache 惊群效应
func TestCacheThunderingHerd(t *testing.T) {
    // 1000 个 goroutine 同时访问刚过期的 Key
    // 验证：不会同时打到磁盘，性能稳定
}
```

**优先级**: 🟡 P1
**影响**: 高负载场景性能

---

### 2.2 Race Detector 覆盖

**当前状态**: 部分测试通过 race detector

**需要测试的场景**:
```bash
# 所有测试都应该通过 race detector
go test -race ./...

# 长时间运行的 race 测试
go test -race -run TestConcurrent -timeout 5m
```

**待验证模块**:
- Store 的 namespace 缓存操作
- BlobManager 的 hashIndex 并发访问
- Cache 的 TTL 计算和过期清理
- KeyMapper 的并发读写

**优先级**: 🔴 P0
**影响**: 数据竞态可能导致数据损坏

---

## 3. 性能与基准测试

### 3.1 基准测试

**当前状态**: 无基准测试

**需要的 Benchmark**:
```go
// 写入性能
func BenchmarkPutSmall(b *testing.B) {
    // 小数据（< 1KB）
}

func BenchmarkPutLarge(b *testing.B) {
    // 大数据（> 1MB，存为 Blob）
}

// 读取性能
func BenchmarkGetCacheHit(b *testing.B) {
    // 缓存命中
}

func BenchmarkGetCacheMiss(b *testing.B) {
    // 缓存未命中，磁盘读取
}

// 并发性能
func BenchmarkConcurrentPut(b *testing.B) {
    // 不同 Key 并发写入
}

// 索引性能
func BenchmarkList1000Keys(b *testing.B) {
    // 列出 1000 个 Key
}

// Compact 性能
func BenchmarkCompact100Versions(b *testing.B) {
    // 压缩 100 个版本
}

// GC 性能
func BenchmarkGC1000Blobs(b *testing.B) {
    // 回收 1000 个 Blob
}
```

**优先级**: 🟢 P2
**影响**: 性能基线，优化指导

---

### 3.2 内存分析

**当前状态**: 无内存分析

**需要测试的场景**:
```bash
# 内存分配分析
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof

# 关注点：
# - Get 操作的内存分配
# - 反向读取的内存占用
# - Cache 的内存增长
# - Blob 流式读取的内存稳定性
```

**优先级**: 🟢 P2
**影响**: 内存占用优化

---

## 4. 集成与端到端测试

### 4.1 完整业务流程

**当前状态**: 有基础集成测试，缺少复杂场景

**需要测试的场景**:
```go
// 多 Namespace 隔离
func TestMultiNamespaceIsolation(t *testing.T) {
    ns1 := store.GetNamespace("ns1")
    ns2 := store.GetNamespace("ns2")

    ns1.Put("key", "value1")
    ns2.Put("key", "value2")

    // 验证：两个 namespace 互不影响
}

// 历史版本回滚
func TestVersionRollback(t *testing.T) {
    // 存储多个版本
    // 恢复到历史版本
    // 验证数据正确
}

// Blob GC 正确性
func TestBlobGCWithSharedBlobs(t *testing.T) {
    // 多个 Key 共享同一 Blob（去重）
    // 删除其中一个 Key
    // GC 后验证：Blob 仍存在（被其他 Key 引用）
}

// 崩溃恢复
func TestCrashRecovery(t *testing.T) {
    // 写入到一半强制关闭
    // 重新打开 Store
    // 验证：数据一致，无损坏
}
```

**优先级**: 🟡 P1
**影响**: 端到端功能正确性

---

### 4.2 外部编辑兼容性

**当前状态**: 缺少外部编辑测试

**需要测试的场景**:
```go
// 手工编辑 JSONL 文件
func TestManualEditJSONL(t *testing.T) {
    // 外部修改 JSONL 文件
    ns.Refresh("key")
    // 验证：读取到最新修改
}

// 删除 Blob 文件
func TestMissingBlobFile(t *testing.T) {
    // 手工删除 Blob 文件
    ns.Get("key", &target)
    // 预期：Warn 日志，字段为零值，不返回错误
}

// 格式错误的行
func TestMalformedJSONLLine(t *testing.T) {
    // 插入非 JSON 行
    // 预期：跳过错误行，继续读取
}
```

**优先级**: 🟢 P2
**影响**: 可编辑性保证

---

## 5. 模块特定测试

### 5.1 Blob 模块

**需要补充的测试**:
```go
// Blob 去重边界情况
func TestBlobDeduplicationWithConcurrentWrites(t *testing.T) {
    // 多个 goroutine 同时写入相同内容
    // 验证：只创建一个 Blob 文件
}

// 大文件流式读取
func TestStreamLargeBlob(t *testing.T) {
    // 创建 100MB Blob
    // 流式读取，不一次性加载到内存
    // 验证：内存占用 < 10MB
}

// Blob 索引重建
func TestBuildIndexAfterBlobChange(t *testing.T) {
    // 手工添加/删除 Blob 文件
    // 重新打开 Store
    // 验证：索引正确重建
}
```

**优先级**: 🟢 P2

---

### 5.2 Index 模块

**需要补充的测试**:
```go
// Key 冲突处理
func TestKeyConflictWithHash(t *testing.T) {
    // 两个 Key 清洗后相同
    // 验证：自动添加哈希后缀
}

// Cache 过期精度
func TestCacheTTLAccuracy(t *testing.T) {
    // 设置 1 秒 TTL
    // 验证：1 秒后确实过期
}

// 大量 Key 的索引性能
func TestIndexWith10000Keys(t *testing.T) {
    // 创建 10000 个 Key
    // 验证：查找性能 < 1ms
}
```

**优先级**: 🟢 P2

---

### 5.3 Codec 模块

**需要补充的测试**:
```go
// 复杂嵌套结构
func TestDeeplyNestedStruct(t *testing.T) {
    // 10 层嵌套
    // 验证：正确序列化/反序列化
}

// 循环引用检测
func TestCircularReference(t *testing.T) {
    type Node struct {
        Next *Node
    }
    n1 := &Node{}
    n1.Next = n1 // 循环引用
    // 预期：返回错误或栈溢出保护
}

// Struct Tag 优先级
func TestTagPriority(t *testing.T) {
    // 同时设置 WithForceInline + Tag + 大小超阈值
    // 验证：WithForceInline 优先级最高
}
```

**优先级**: 🟢 P2

---

## 6. 文档与示例测试

### 6.1 README 示例

**当前状态**: 缺少示例代码测试

**需要的测试**:
```go
// TestREADMEExamples 验证 README 中的所有示例代码
func TestREADMEExamples(t *testing.T) {
    // 复制 README 中的示例代码
    // 验证：可以正常运行
}

// TestExampleCode 验证 examples/ 目录中的示例
func TestExampleCode(t *testing.T) {
    // 运行 examples/basic/main.go
    // 验证：无错误
}
```

**优先级**: 🟢 P2
**影响**: 文档准确性

---

## 7. 回归测试

### 7.1 已知 Bug 防护

**当前状态**: 无回归测试套件

**建议**:
```go
// TestRegressions 包含所有已修复 Bug 的测试用例
func TestRegressionGCCorrectness(t *testing.T) {
    // Bug: GC 误删还在使用的 Blob
    // 场景：更新 Key 后，旧 Blob 被错误标记为未引用
    // 修复：只收集最新记录的 blob 引用
}

func TestRegressionCacheJitter(t *testing.T) {
    // Bug: Cache 可能过早过期
    // 场景：TTL × (1 - 0.2) 导致过早失效
    // 修复：单向 jitter
}

func TestRegressionScalarValue(t *testing.T) {
    // Bug: 无法存储标量值
    // 场景：ns.Put("key", "string") 失败
    // 修复：标量值包装
}
```

**优先级**: 🟡 P1
**影响**: 防止 Bug 复现

---

## 8. 测试基础设施

### 8.1 测试工具

**需要的工具**:
```go
// 测试辅助函数
func createTestStore(t *testing.T) *Store {
    tmpDir := t.TempDir()
    return MustOpen(tmpDir)
}

func createLargeBlob(size int64) []byte {
    // 生成大文件用于测试
}

func simulateDiskFull() {
    // 模拟磁盘满
}

func simulateSlowIO() {
    // 模拟慢磁盘
}

// Chaos 测试框架
func runChaosTest(t *testing.T, operations []Operation) {
    // 随机注入错误（磁盘满、网络断开、进程崩溃）
    // 验证系统稳定性
}
```

**优先级**: 🟢 P2

---

### 8.2 CI/CD 集成

**当前状态**: 缺少 CI 配置

**需要的 CI 任务**:
```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Run tests
        run: go test ./...

      - name: Run race detector
        run: go test -race ./...

      - name: Run benchmarks
        run: go test -bench=. -benchmem

      - name: Coverage report
        run: go test -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

**优先级**: 🟡 P1
**影响**: 自动化测试

---

## 9. 测试覆盖率目标

### 当前覆盖率
- 未测量（需要 `go test -cover`）

### 目标覆盖率
| 模块 | 当前 | 目标 | 关键未覆盖 |
|------|------|------|-----------|
| internal/blob | ??? | 85% | 错误处理路径 |
| internal/codec | ??? | 85% | 边界值 |
| internal/core | ??? | 90% | 文件损坏处理 |
| internal/index | ??? | 85% | Cache 过期 |
| namespace | ??? | 80% | 并发竞态 |
| store | ??? | 75% | 多 namespace |

---

## 10. 优先级总结

### 🔴 P0 - 立即实施
1. **Race Detector 全覆盖** - 确保无竞态条件
2. **错误处理测试** - 磁盘满、权限不足、文件损坏

### 🟡 P1 - 短期（1-2 周）
3. **并发压力测试** - 高负载场景验证
4. **回归测试套件** - 防止 Bug 复现
5. **端到端集成测试** - 完整业务流程
6. **CI/CD 集成** - 自动化测试

### 🟢 P2 - 中期（1 个月）
7. **性能基准测试** - 建立性能基线
8. **边界值测试** - 极限场景稳定性
9. **模块特定测试** - 各模块深度测试
10. **文档示例测试** - 文档准确性

---

## 实施建议

### 第一阶段（本周）
- ✅ 运行 `go test -race ./...` 并修复所有 race condition
- ✅ 添加磁盘满、权限不足的错误处理测试
- ✅ 创建回归测试文件 `tests/regression_test.go`

### 第二阶段（下周）
- 添加高并发压力测试
- 实现 CI/CD 配置
- 补充端到端集成测试

### 第三阶段（本月）
- 添加完整的性能基准测试
- 测量并提升代码覆盖率到 80%+
- 补充边界值和模块特定测试

---

## 总结

当前测试已覆盖核心功能，但以下领域需要加强：

1. **错误处理** - 系统级错误恢复
2. **并发压力** - 高负载场景验证
3. **性能基准** - 建立性能基线
4. **回归防护** - 防止已修复 Bug 复现
5. **CI/CD** - 自动化测试流程

**建议优先完成 P0 和 P1 任务，确保生产环境稳定性。**
