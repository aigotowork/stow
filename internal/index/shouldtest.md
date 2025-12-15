# Index 模块测试改进建议

## 当前状态
- **覆盖率**: 40.3% (最低)
- **问题**: 所有测试集中在一个文件中，许多关键功能未测试

## 测试文件重构建议

### 建议的测试文件结构

```
internal/index/
├── cache.go           -> cache_test.go
├── mapper.go          -> mapper_test.go
├── scanner.go         -> scanner_test.go
├── sanitize.go        -> sanitize_test.go
└── integration_test.go (新增)
```

---

## 1. Cache 测试 (cache_test.go)

### 当前覆盖情况
- ✅ 已测试: Get, Set, SetWithTTL, Delete, Clear, IsExpired
- ❌ 未测试: Exists, Count, Keys, CleanupExpired, Stats, HitRate, StartCleanupWorker

### 需要添加的测试场景

#### 基础功能测试
- `TestCacheExists` - 测试键存在性检查
- `TestCacheCount` - 测试缓存项计数
- `TestCacheKeys` - 测试获取所有键

#### 过期清理测试
- `TestCacheCleanupExpired` - 测试手动清理过期项
- `TestCacheAutoCleanup` - 测试自动清理功能
- `TestCacheStartCleanupWorker` - 测试清理工作协程

#### 统计信息测试
- `TestCacheStats` - 测试缓存统计信息
- `TestCacheHitRate` - 测试命中率计算
- `TestCacheStatsAfterOperations` - 测试各种操作后的统计

#### 边界测试
- `TestCacheConcurrentAccess` - 测试并发访问
- `TestCacheLargeDataset` - 测试大量数据
- `TestCacheTTLEdgeCases` - 测试 TTL 边界情况
  - TTL = 0
  - TTL = 负数
  - TTL 过期瞬间的访问

---

## 2. Mapper 测试 (mapper_test.go)

### 当前覆盖情况
- ✅ 已测试: Add, Find, Remove, ListAll, Count, Clear, HasConflict, SanitizeKey
- ❌ 未测试: FindExact, RemoveByFileName, GetConflicts, Stats, String

### 需要添加的测试场景

#### 基础功能测试
- `TestMapperFindExact` - 测试精确查找
- `TestMapperRemoveByFileName` - 测试按文件名删除
- `TestMapperGetConflicts` - 测试获取冲突列表
- `TestMapperStats` - 测试统计信息
- `TestMapperString` - 测试字符串表示

#### 冲突处理测试
- `TestMapperConflictDetection` - 测试冲突检测
- `TestMapperConflictResolution` - 测试冲突解决
- `TestMapperMultipleConflicts` - 测试多个冲突

#### 键清理测试
- `TestMapperSanitizeSpecialChars` - 测试特殊字符清理
- `TestMapperSanitizeUnicode` - 测试 Unicode 处理
- `TestMapperSanitizeLongKeys` - 测试长键处理

#### 工具函数测试
- `TestExtractKeyFromFileName` - 测试从文件名提取键
- `TestIsHexString` - 测试十六进制字符串检测
- `TestKeyConflict` - 测试键冲突判断
- `TestNeedsHashSuffix` - 测试是否需要哈希后缀
- `TestIsValidKey` - 测试键有效性验证
- `TestCleanPath` - 测试路径清理

---

## 3. Scanner 测试 (scanner_test.go)

### 当前覆盖情况
- ✅ 已测试: NewScanner, ScanNamespace, readKeyFromFile
- ❌ 未测试: ScanAndValidate, CountFiles, ListKeys

### 需要添加的测试场景

#### 基础功能测试
- `TestScannerScanAndValidate` - 测试扫描和验证
- `TestScannerCountFiles` - 测试文件计数
- `TestScannerListKeys` - 测试列出所有键

#### 错误处理测试
- `TestScannerInvalidFiles` - 测试处理无效文件
- `TestScannerCorruptedData` - 测试处理损坏数据
- `TestScannerMissingFiles` - 测试处理缺失文件
- `TestScannerPermissionDenied` - 测试权限错误

#### 性能测试
- `TestScannerLargeNamespace` - 测试扫描大命名空间
- `TestScannerDeepDirectory` - 测试深层目录结构

---

## 4. Sanitize 测试 (sanitize_test.go)

### 需要添加的测试场景

#### 字符清理测试
- `TestSanitizeBasicChars` - 测试基本字符清理
- `TestSanitizeSpecialChars` - 测试特殊字符（/, \, :, *, ?, ", <, >, |）
- `TestSanitizeUnicode` - 测试 Unicode 字符
- `TestSanitizeEmoji` - 测试 Emoji 处理
- `TestSanitizeWhitespace` - 测试空白字符

#### 路径清理测试
- `TestSanitizePathTraversal` - 测试路径遍历攻击（../）
- `TestSanitizeAbsolutePath` - 测试绝对路径
- `TestSanitizeMultipleSlashes` - 测试多个斜杠

#### 边界测试
- `TestSanitizeEmptyString` - 测试空字符串
- `TestSanitizeLongString` - 测试超长字符串
- `TestSanitizeOnlySpecialChars` - 测试只有特殊字符

---

## 5. 集成测试 (integration_test.go)

### 新增集成测试场景

#### 完整流程测试
- `TestIndexFullLifecycle` - 测试完整生命周期
  1. 创建索引
  2. 添加多个键
  3. 查询和验证
  4. 更新缓存
  5. 扫描和验证
  6. 清理和删除

#### 多组件协作测试
- `TestCacheMapperIntegration` - 测试缓存和映射器协作
- `TestMapperScannerIntegration` - 测试映射器和扫描器协作
- `TestCacheScannerSync` - 测试缓存和扫描器同步

#### 并发测试
- `TestConcurrentCacheAccess` - 测试并发缓存访问
- `TestConcurrentMapperOperations` - 测试并发映射操作
- `TestConcurrentScanAndUpdate` - 测试并发扫描和更新

---

## 优先级建议

### 高优先级 (立即添加)
1. ✅ Cache 基础功能测试 (Exists, Count, Keys)
2. ✅ Mapper 工具函数测试 (FindExact, RemoveByFileName)
3. ✅ Scanner 完整测试 (ScanAndValidate, CountFiles, ListKeys)

### 中优先级 (第二阶段)
1. ⚠️ Cache 过期和清理测试
2. ⚠️ Mapper 冲突处理测试
3. ⚠️ Scanner 错误处理测试

### 低优先级 (优化阶段)
1. 📝 性能测试和基准测试
2. 📝 并发和压力测试
3. 📝 集成测试

---

## 预期效果

实施上述测试改进后：
- **覆盖率目标**: 40.3% → **75%+**
- **测试文件数**: 1 → 5
- **测试用例数**: ~20 → **60+**
- **代码质量**: 显著提升错误检测能力

---

## 实施建议

1. **第一步**: 重构现有测试
   - 将 index_test.go 拆分为多个文件
   - 保持现有测试不变

2. **第二步**: 补充基础测试
   - 添加未覆盖的基础功能测试
   - 确保所有公开函数都有测试

3. **第三步**: 添加边界测试
   - 测试边界条件
   - 测试错误处理

4. **第四步**: 添加集成测试
   - 测试组件间协作
   - 测试并发场景
