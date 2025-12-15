# Stow 项目测试改进总览

## 执行摘要

本文档提供了 Stow 项目所有内部模块的测试改进建议总览。通过系统化的测试重构和补充，我们的目标是将整体测试覆盖率从当前的 **58.9%** 提升到 **80%+**。

---

## 当前状态总览

| 模块 | 当前覆盖率 | 目标覆盖率 | 优先级 | 预估工作量 |
|------|-----------|-----------|--------|----------|
| **index** | 40.3% | 75%+ | 🔴 高 | 5-7 天 |
| **codec** | 60.6% | 80%+ | 🟡 中 | 4-6 天 |
| **fsutil** | 56.8% | 85%+ | 🟡 中 | 4-5 天 |
| **core** | 69.6% | 85%+ | 🟢 低 | 3-4 天 |
| **blob** | 73.9% | 90%+ | 🟢 低 | 3-4 天 |
| **总计** | **58.9%** | **83%+** | - | **19-26 天** |

---

## 测试文件重构路线图

### 原则
1. **单一职责**: 每个测试文件只测试一个源文件
2. **适度大小**: 测试文件保持在 200-500 行
3. **清晰命名**: 遵循 `<源文件>_test.go` 命名约定
4. **独立运行**: 测试之间不应有依赖

### 重构前后对比

#### Index 模块
```
重构前:
internal/index/
└── index_test.go (1 个文件, ~500 行)

重构后:
internal/index/
├── cache_test.go
├── mapper_test.go
├── scanner_test.go
├── sanitize_test.go
└── integration_test.go
(5 个文件, 平均 ~300 行)
```

#### Codec 模块
```
重构前:
internal/codec/
└── codec_test.go (1 个文件, 1700+ 行)

重构后:
internal/codec/
├── tag_test.go
├── marshal_test.go
├── unmarshal_test.go
├── reflect_test.go
├── blob_test.go
└── integration_test.go
(6 个文件, 平均 ~300 行)
```

#### FSUtil 模块
```
重构前:
internal/fsutil/
└── fsutil_test.go (1 个文件, ~400 行)

重构后:
internal/fsutil/
├── atomic_test.go
├── safe_test.go
├── walk_test.go
├── permissions_test.go
└── integration_test.go
(5 个文件, 平均 ~250 行)
```

#### Blob 模块
```
重构前:
internal/blob/
└── (测试文件名待确认)

重构后:
internal/blob/
├── file_test.go
├── hash_test.go
├── manager_test.go
├── reference_test.go
├── writer_test.go
└── integration_test.go
(6 个文件, 平均 ~250 行)
```

#### Core 模块
```
重构前:
internal/core/
└── (测试文件名待确认)

重构后:
internal/core/
├── decoder_test.go
├── encoder_test.go
├── record_test.go
├── meta_test.go
├── format_test.go
└── integration_test.go
(6 个文件, 平均 ~250 行)
```

---

## 实施路线图

### 阶段 1: 高优先级模块 (第 1-2 周)

#### Week 1: Index 模块重构
- **Day 1-2**: 拆分现有测试文件
- **Day 3-4**: 补充 Cache 和 Mapper 测试
- **Day 5**: Scanner 和 Sanitize 测试

**里程碑**: Index 覆盖率 40.3% → 70%+

#### Week 2: Codec 和 FSUtil 模块
- **Day 1-2**: Codec 文件拆分和基础测试补充
- **Day 3-4**: FSUtil 文件拆分和测试补充
- **Day 5**: 两个模块的集成测试

**里程碑**:
- Codec 覆盖率 60.6% → 75%+
- FSUtil 覆盖率 56.8% → 80%+

### 阶段 2: 中低优先级模块 (第 3 周)

#### Week 3: Blob 和 Core 模块
- **Day 1-2**: Blob 测试重构和补充
- **Day 3-4**: Core 测试重构和补充
- **Day 5**: 集成测试和文档更新

**里程碑**:
- Blob 覆盖率 73.9% → 90%+
- Core 覆盖率 69.6% → 85%+

### 阶段 3: 完善和优化 (第 4 周)

#### Week 4: 高级测试和优化
- **Day 1-2**: 所有模块并发测试
- **Day 3**: 性能基准测试
- **Day 4**: 集成测试完善
- **Day 5**: 文档和报告

**最终目标**: 整体覆盖率 83%+

---

## 测试分类标准

### 1. 单元测试 (Unit Tests)
- **范围**: 单个函数或方法
- **特点**: 快速、独立、可重复
- **命名**: `Test<Function><Scenario>`
- **示例**: `TestCacheGet`, `TestMapperSanitizeKey`

### 2. 集成测试 (Integration Tests)
- **范围**: 多个组件协作
- **特点**: 测试组件间交互
- **文件**: `integration_test.go`
- **示例**: `TestCacheMapperIntegration`

### 3. 边界测试 (Edge Case Tests)
- **范围**: 边界条件和特殊情况
- **特点**: 覆盖异常情况
- **命名**: `Test<Function>EdgeCases`
- **示例**: `TestCacheTTLEdgeCases`

### 4. 错误测试 (Error Tests)
- **范围**: 错误处理和恢复
- **特点**: 验证错误场景
- **命名**: `Test<Function>Errors`
- **示例**: `TestAtomicWriteFileErrors`

### 5. 并发测试 (Concurrency Tests)
- **范围**: 多线程/协程场景
- **特点**: 检测竞争条件
- **命名**: `Test<Function>Concurrent`
- **示例**: `TestCacheConcurrentAccess`

### 6. 性能测试 (Benchmark Tests)
- **范围**: 性能度量
- **特点**: 量化性能指标
- **命名**: `Benchmark<Function>`
- **示例**: `BenchmarkCacheGet`

---

## 测试质量标准

### 覆盖率目标

#### 代码覆盖率
- **整体目标**: 80%+
- **核心功能**: 95%+
- **错误处理**: 70%+
- **工具函数**: 85%+

#### 场景覆盖
- ✅ 正常流程: 100%
- ✅ 错误处理: 80%+
- ✅ 边界条件: 70%+
- ✅ 并发场景: 50%+

### 测试质量指标

#### 可读性
- [ ] 清晰的测试命名
- [ ] 适当的注释
- [ ] 使用 table-driven tests
- [ ] 辅助函数复用

#### 可维护性
- [ ] 测试独立运行
- [ ] 无外部依赖
- [ ] 使用 t.TempDir()
- [ ] 适当的 cleanup

#### 完整性
- [ ] 正常路径测试
- [ ] 错误路径测试
- [ ] 边界测试
- [ ] 并发测试

---

## 测试最佳实践

### 1. 测试结构

```go
func TestFunctionName(t *testing.T) {
    // 1. 准备 (Arrange)
    input := setupTestData()
    expected := expectedResult()

    // 2. 执行 (Act)
    actual := FunctionToTest(input)

    // 3. 断言 (Assert)
    if actual != expected {
        t.Errorf("got %v, want %v", actual, expected)
    }

    // 4. 清理 (Cleanup) - 如果需要
    cleanup()
}
```

### 2. Table-Driven Tests

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "result",
            wantErr: false,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 3. 测试辅助函数

```go
// 使用 t.Helper() 标记辅助函数
func setupTestEnvironment(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    // 设置逻辑...
    return dir
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if !reflect.DeepEqual(got, want) {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

### 4. 并发测试

```go
func TestConcurrentAccess(t *testing.T) {
    const goroutines = 100
    cache := NewCache()

    var wg sync.WaitGroup
    wg.Add(goroutines)

    for i := 0; i < goroutines; i++ {
        go func(id int) {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", id)
            cache.Set(key, id)
            got := cache.Get(key)
            if got != id {
                t.Errorf("got %v, want %v", got, id)
            }
        }(i)
    }

    wg.Wait()
}
```

### 5. Benchmark 测试

```go
func BenchmarkFunction(b *testing.B) {
    setup := setupBenchmark()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Function(setup)
    }
}

// 不同大小的基准测试
func BenchmarkFunctionSizes(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := makeData(size)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                Function(data)
            }
        })
    }
}
```

---

## 工具和资源

### 测试工具
- `go test` - 标准测试工具
- `go test -cover` - 覆盖率分析
- `go test -race` - 竞争检测
- `go test -bench` - 性能测试

### 推荐库
- `github.com/stretchr/testify` - 断言和 mock
- `github.com/google/go-cmp` - 深度比较
- `golang.org/x/sync/errgroup` - 并发测试

### 覆盖率工具
```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率详情
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

---

## 持续改进

### 定期审查
- **每周**: 检查新增代码的测试覆盖率
- **每月**: 审查测试质量和可维护性
- **每季度**: 评估测试策略和调整优先级

### 质量门禁
1. 新增代码必须有测试
2. 测试覆盖率不能降低
3. 所有测试必须通过
4. 无竞争条件

### 文档维护
- 更新测试文档
- 记录测试模式
- 分享最佳实践
- 培训团队成员

---

## 预期收益

### 代码质量
- ✅ 减少 bug 数量 (预计 50%+)
- ✅ 提高代码可靠性
- ✅ 增强重构信心
- ✅ 改善代码设计

### 开发效率
- ✅ 快速发现问题
- ✅ 安全重构
- ✅ 减少调试时间
- ✅ 提高开发速度

### 团队协作
- ✅ 清晰的代码契约
- ✅ 更好的文档
- ✅ 知识共享
- ✅ 提高信心

---

## 结论

通过系统化的测试改进，我们将：

1. **提升覆盖率**: 58.9% → 83%+
2. **改善代码质量**: 减少 bug，提高可靠性
3. **优化代码结构**: 更好的模块化和可维护性
4. **建立测试文化**: 形成良好的测试习惯

**投入**: 19-26 人天
**产出**: 高质量、可靠的代码库

---

## 附录

### 各模块详细文档
- [Index 模块测试建议](internal/index/shouldtest.md)
- [Codec 模块测试建议](internal/codec/shouldtest.md)
- [FSUtil 模块测试建议](internal/fsutil/shouldtest.md)
- [Blob 模块测试建议](internal/blob/shouldtest.md)
- [Core 模块测试建议](internal/core/shouldtest.md)

### 参考资源
- [Go Testing Best Practices](https://golang.org/doc/code.html#Testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Effective Go - Testing](https://golang.org/doc/effective_go#testing)
