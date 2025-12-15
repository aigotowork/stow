# Blob 模块测试改进建议

## 当前状态
- **覆盖率**: 73.9% (良好)
- **问题**: 部分边界情况和错误处理未测试

## 测试文件重构建议

### 建议的测试文件结构

```
internal/blob/
├── file.go             -> file_test.go
├── hash.go             -> hash_test.go
├── manager.go          -> manager_test.go
├── reference.go        -> reference_test.go
├── writer.go           -> writer_test.go
└── integration_test.go (新增)
```

---

## 1. File 测试 (file_test.go)

### 当前覆盖情况
- ✅ NewFileData: 100%
- ⚠️ Read: 85.7%
- ⚠️ Close: 80%
- ✅ Name, Size, MimeType, Hash: 100%
- ❌ Path: 0%

### 需要添加的测试场景

#### 基础功能测试
```go
TestFileDataPath
  - 获取文件路径
  - 相对路径vs绝对路径

TestFileDataReadPartial
  - 读取部分内容
  - 多次读取
  - EOF 处理

TestFileDataReadErrors
  - 文件被删除
  - 权限变更
  - 读取中断
```

#### 资源管理测试
```go
TestFileDataCloseMultiple
  - 多次 Close
  - Close 后 Read

TestFileDataConcurrentRead
  - 并发读取同一 FileData
  - 读写竞争
```

---

## 2. Hash 测试 (hash_test.go)

### 当前覆盖情况
- ⚠️ ComputeSHA256: 80%
- ✅ ComputeSHA256FromBytes: 100%
- ✅ HashPrefix, ShortHash: 100%

### 需要添加的测试场景

#### Hash 计算测试
```go
TestComputeSHA256Errors
  - 文件不存在
  - 权限不足
  - 读取中断
  - 大文件 hash

TestComputeSHA256Consistency
  - 相同内容相同 hash
  - 不同内容不同 hash
  - 空文件 hash
```

#### Hash 工具函数测试
```go
TestHashPrefixLength
  - 不同长度前缀
  - 空 hash
  - 无效 hash

TestShortHashUniqueness
  - 冲突检测
  - 唯一性验证
```

---

## 3. Manager 测试 (manager_test.go)

### 当前覆盖情况
- ⚠️ NewManager: 66.7%
- ⚠️ Store: 78.4%
- ⚠️ Load: 71.4%
- ⚠️ LoadBytes: 80%
- ⚠️ Exists: 75%
- ⚠️ Delete: 86.7%
- ⚠️ ListAll: 77.8%
- ❌ TotalSize: 0%
- ⚠️ Count: 75%
- ⚠️ buildIndex: 28.6%

### 需要添加的测试场景

#### Manager 创建测试
```go
TestNewManagerValidation
  - 无效目录
  - 权限不足
  - 目录不存在自动创建
  - 参数验证 (负数阈值等)

TestNewManagerIndexBuild
  - 空目录
  - 有现有 blob
  - 损坏的 blob
  - 孤儿文件处理
```

#### Store 测试增强
```go
TestStoreWithDifferentSources
  - []byte 存储
  - io.Reader 存储
  - 大文件流式存储
  - 空内容

TestStoreErrors
  - 磁盘空间不足
  - 权限不足
  - Hash 冲突
  - 存储中断
  - 回滚验证

TestStoreDedplication
  - 相同内容多次存储
  - Hash 去重验证
```

#### Load 测试增强
```go
TestLoadErrors
  - Blob 不存在
  - Blob 损坏
  - Hash 不匹配
  - 权限不足

TestLoadConcurrency
  - 并发加载同一 blob
  - 并发加载不同 blob
```

#### TotalSize 测试
```go
TestManagerTotalSize
  - 空 manager
  - 单个 blob
  - 多个 blob
  - 大小计算准确性

TestTotalSizeAfterOperations
  - Store 后大小增加
  - Delete 后大小减少
  - 实时更新验证
```

#### BuildIndex 测试
```go
TestBuildIndexComplete
  - 正常 blob 文件
  - 损坏的 blob
  - 无效文件名
  - 孤儿文件
  - 部分索引重建

TestBuildIndexPerformance
  - 大量 blob 索引
  - 深层目录结构
```

---

## 4. Reference 测试 (reference_test.go)

### 当前覆盖情况
- ✅ NewReference: 100%
- ✅ IsValid: 100%
- ✅ IsBlobReference: 100%
- ⚠️ FromMap: 87.5%
- ✅ ToMap: 100%

### 需要添加的测试场景

#### FromMap 边界测试
```go
TestFromMapInvalid
  - 缺少必需字段
  - 类型不匹配
  - 无效值

TestFromMapEdgeCases
  - 额外字段
  - 空值处理
  - nil map
```

#### Reference 验证测试
```go
TestReferenceValidation
  - 有效引用
  - 无效 hash
  - 无效 location
  - 负数 size
  - 空名称

TestReferenceComparison
  - 相等性比较
  - Hash 比较
  - Location 比较
```

---

## 5. Writer 测试 (writer_test.go)

### 当前覆盖情况
- ⚠️ NewWriter: 75%
- ⚠️ Write: 87.5%
- ⚠️ WriteFrom: 81.8%
- ⚠️ Close: 62.5%
- ✅ Abort: 100%
- ❌ Written: 0%

### 需要添加的测试场景

#### Writer 基础测试
```go
TestWriterWritten
  - 跟踪写入字节数
  - 多次写入累计
  - Abort 后重置

TestWriterMultipleWrites
  - 分多次写入
  - 大块写入
  - 小块写入
```

#### Writer 错误处理
```go
TestWriterCloseErrors
  - Close 前中断
  - Hash 计算失败
  - Rename 失败
  - 清理临时文件

TestWriterWriteErrors
  - 磁盘满
  - 权限不足
  - 写入中断
```

#### Writer 资源管理
```go
TestWriterAbort
  - Abort 清理临时文件
  - Abort 后不能写入
  - 多次 Abort

TestWriterCloseIdempotent
  - 多次 Close
  - Close 后写入失败
```

#### Writer 并发测试
```go
TestWriterConcurrent
  - 并发写入不同文件
  - Writer 不应并发使用 (验证)
```

---

## 6. 集成测试 (integration_test.go)

### 完整流程测试

```go
TestBlobLifecycle
  - Create -> Store -> Load -> Verify -> Delete
  - 多个 blob 管理
  - 引用计数 (如果实现)

TestBlobManagerRestart
  - Store blob
  - 重启 manager
  - Load blob
  - 索引一致性

TestBlobConcurrentOperations
  - 并发 Store
  - 并发 Load
  - 并发 Delete
  - 混合操作

TestBlobGarbageCollection
  - 识别孤儿 blob
  - 清理未引用 blob
  - 安全删除检查

TestBlobCorruptionRecovery
  - 检测损坏 blob
  - Hash 不匹配处理
  - 自动修复 (如果支持)

TestBlobMigration
  - 不同格式迁移
  - 版本升级
  - 数据完整性验证
```

### 性能测试

```go
BenchmarkBlobStore
  - 不同大小文件
  - 批量存储

BenchmarkBlobLoad
  - 冷加载
  - 热加载

BenchmarkBlobListAll
  - 不同数量级 blob

BenchmarkBlobIndexBuild
  - 重建索引性能
```

---

## 优先级建议

### 🔴 高优先级
1. TotalSize 测试 (0% 覆盖)
2. Written 测试 (0% 覆盖)
3. Path 测试 (0% 覆盖)
4. BuildIndex 完整测试 (28.6% 覆盖)

### 🟡 中优先级
1. Manager 错误处理测试
2. Writer Close 错误测试
3. 并发测试

### 🟢 低优先级
1. 性能基准测试
2. 压力测试
3. 迁移测试

---

## 预期效果

实施上述测试改进后：
- **覆盖率目标**: 73.9% → **90%+**
- **测试文件数**: 1 → 6
- **测试用例数**: ~30 → **80+**

---

## 特殊测试场景

### 文件系统故障模拟
```go
TestBlobDiskFailure
  - 磁盘满模拟
  - 写入失败恢复
  - 事务回滚

TestBlobFileSystemErrors
  - 权限错误
  - I/O 错误
  - 文件锁定
```

### 数据完整性测试
```go
TestBlobIntegrity
  - Hash 验证
  - 内容验证
  - Metadata 一致性

TestBlobCorruption
  - 位翻转检测
  - 部分写入检测
  - 自动恢复
```

### 清理和维护测试
```go
TestBlobMaintenance
  - 清理临时文件
  - 压缩存储
  - 重建索引
  - 验证完整性
```
