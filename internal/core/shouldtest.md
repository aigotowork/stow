# Core 模块测试改进建议

## 当前状态
- **覆盖率**: 69.6%
- **问题**: 部分核心功能和错误处理未测试

## 测试文件重构建议

### 建议的测试文件结构

```
internal/core/
├── decoder.go          -> decoder_test.go
├── encoder.go          -> encoder_test.go
├── record.go           -> record_test.go
├── meta.go             -> meta_test.go
├── format_test.go      (新增 - 格式相关测试)
└── integration_test.go (新增 - 集成测试)
```

---

## 1. Decoder 测试 (decoder_test.go)

### 当前覆盖情况
- ✅ NewDecoder: 100%
- ⚠️ Decode: 66.7%
- ❌ DecodeString: 0%
- ⚠️ ReadAll: 82.4%
- ✅ ReadLastValid: 100%
- ⚠️ ReadLastValidReverse: 76.2%
- ⚠️ ReadVersion: 71.4%
- ⚠️ CountLines: 81.8%
- ⚠️ GetLatestVersion: 83.3%
- ⚠️ AppendRecord: 69.2%
- ❌ ReadLastNRecords: 0%
- ❌ ReadLines: 0%

### 需要添加的测试场景

#### 基础解码测试
```go
TestDecodeString
  - JSON 字符串解码
  - 空字符串
  - 无效 JSON
  - 大字符串
  - 特殊字符

TestDecodeMultipleFormats
  - JSON
  - 二进制
  - 自定义格式
```

#### ReadLastNRecords 测试
```go
TestReadLastNRecords
  - 读取最后 N 条记录
  - N = 0
  - N = 1
  - N > 实际记录数
  - 负数 N

TestReadLastNRecordsPerformance
  - 大文件高效读取
  - 避免完整文件扫描
```

#### ReadLines 测试
```go
TestReadLines
  - 读取所有行
  - 空文件
  - 单行文件
  - 大文件分页读取

TestReadLinesWithOffset
  - offset + limit
  - 越界处理
```

#### 错误处理测试
```go
TestDecoderErrors
  - 格式错误
  - 损坏的数据
  - 不完整的记录
  - 版本不匹配

TestDecoderCorruptedData
  - 部分损坏
  - 完全损坏
  - 恢复可能
```

#### 性能测试
```go
TestDecoderLargeFile
  - 大文件解码
  - 内存使用
  - 流式处理

TestDecoderConcurrent
  - 并发读取
  - 线程安全
```

---

## 2. Encoder 测试 (encoder_test.go)

### 当前覆盖情况
- ✅ NewEncoder: 100%
- ⚠️ Encode: 66.7%
- ❌ EncodeToString: 0%

### 需要添加的测试场景

#### 基础编码测试
```go
TestEncodeToString
  - 对象编码为字符串
  - 空对象
  - 复杂对象
  - 特殊字符处理

TestEncodeFormats
  - JSON 格式
  - 二进制格式
  - 压缩格式
  - 自定义格式
```

#### 编码选项测试
```go
TestEncodeOptions
  - 缩进选项
  - 压缩选项
  - 格式化选项

TestEncodeWithMetadata
  - 包含元数据
  - 版本信息
  - 时间戳
```

#### 编码错误测试
```go
TestEncodeErrors
  - 不可序列化类型
  - 循环引用
  - 大对象处理
  - 写入失败

TestEncodeEdgeCases
  - nil 值
  - 空结构
  - 嵌套深度限制
```

---

## 3. Record 测试 (record_test.go)

### 当前覆盖情况
- ❌ NewRecord: 0%
- ✅ NewPutRecord: 100%
- ✅ NewDeleteRecord: 100%
- ✅ IsValid: 100%

### 需要添加的测试场景

#### NewRecord 测试
```go
TestNewRecord
  - 创建基础记录
  - 不同操作类型
  - 带元数据

TestNewRecordValidation
  - 必需字段验证
  - 字段类型验证
  - 约束检查
```

#### Record 操作测试
```go
TestRecordSerialization
  - 序列化/反序列化
  - 格式一致性
  - 版本兼容性

TestRecordComparison
  - 记录相等性
  - 字段比较
  - Hash 比较
```

#### Record 类型测试
```go
TestRecordTypes
  - Put 记录
  - Delete 记录
  - Update 记录 (如果有)
  - 自定义类型

TestRecordMetadata
  - 时间戳
  - 版本号
  - 作者信息
  - 自定义元数据
```

---

## 4. Meta 测试 (meta_test.go)

### 当前覆盖情况
- ✅ NewMeta: 100%
- ✅ IsPut: 100%
- ✅ IsDelete: 100%

### 需要添加的测试场景

#### Meta 创建测试
```go
TestMetaCreation
  - 默认值
  - 自定义值
  - 验证规则

TestMetaValidation
  - 操作类型验证
  - 版本验证
  - 时间戳验证
```

#### Meta 操作测试
```go
TestMetaOperations
  - 获取/设置字段
  - 元数据更新
  - 元数据克隆

TestMetaSerialization
  - JSON 序列化
  - 二进制序列化
  - 压缩格式
```

---

## 5. Format 测试 (format_test.go)

### 新增格式相关测试

```go
TestRecordFormat
  - 格式规范验证
  - 格式版本
  - 向后兼容性

TestRecordFormatMigration
  - 旧格式迁移
  - 版本升级
  - 降级处理

TestRecordFormatValidation
  - 格式检查
  - 损坏检测
  - 自动修复
```

### 格式兼容性测试

```go
TestFormatCompatibility
  - 跨版本读写
  - 格式演进
  - 破坏性变更处理

TestFormatEdgeCases
  - 最小记录
  - 最大记录
  - 边界值
```

---

## 6. 集成测试 (integration_test.go)

### 完整流程测试

```go
TestEncodeDecodeRoundTrip
  - Encode -> Decode -> 验证
  - 不同数据类型
  - 复杂结构
  - 元数据保持

TestRecordPersistence
  - 写入记录
  - 读取验证
  - 更新记录
  - 删除记录

TestVersionEvolution
  - 多版本记录
  - 版本回溯
  - GetLatestVersion
  - ReadVersion

TestConcurrentAccess
  - 并发读
  - 并发写
  - 读写混合
  - 数据一致性
```

### 错误恢复测试

```go
TestRecordRecovery
  - 损坏记录恢复
  - 部分读取
  - 跳过坏记录
  - 完整性检查

TestTransactionalBehavior
  - 原子写入
  - 回滚机制
  - 一致性保证
```

### 性能测试

```go
BenchmarkEncode
  - 小对象
  - 大对象
  - 复杂结构

BenchmarkDecode
  - 不同大小数据
  - 不同格式

BenchmarkReadOperations
  - ReadAll
  - ReadLastValid
  - ReadLastNRecords

BenchmarkAppendRecord
  - 顺序追加
  - 大量追加
```

---

## 优先级建议

### 🔴 高优先级 (立即添加)
1. DecodeString 测试 (0% 覆盖)
2. EncodeToString 测试 (0% 覆盖)
3. ReadLastNRecords 测试 (0% 覆盖)
4. ReadLines 测试 (0% 覆盖)
5. NewRecord 测试 (0% 覆盖)

### 🟡 中优先级 (第二阶段)
1. Decode 错误处理 (提升至 90%+)
2. Encode 边界测试
3. 格式相关测试

### 🟢 低优先级 (优化阶段)
1. 并发测试
2. 性能基准测试
3. 压力测试

---

## 预期效果

实施上述测试改进后：
- **覆盖率目标**: 69.6% → **85%+**
- **测试文件数**: 1 → 6
- **测试用例数**: ~20 → **60+**

---

## 特殊测试场景

### 数据完整性测试
```go
TestDataIntegrity
  - Hash 验证
  - Checksum 验证
  - 内容验证

TestDataCorruption
  - 单比特错误
  - 多比特错误
  - 结构损坏
  - 恢复策略
```

### 格式演进测试
```go
TestFormatEvolution
  - 版本 1 -> 版本 2
  - 添加字段
  - 删除字段
  - 重命名字段
  - 类型变更

TestBackwardCompatibility
  - 新代码读旧数据
  - 旧代码读新数据 (如果支持)
  - 优雅降级
```

### 大规模数据测试
```go
TestLargeDataset
  - 百万级记录
  - GB 级文件
  - 性能指标
  - 内存使用

TestStreamProcessing
  - 流式读取
  - 流式写入
  - 增量处理
```

---

## 实施建议

### 第一阶段：补充基础测试
1. 添加所有 0% 覆盖的函数测试
2. 确保所有公开 API 有基础测试
3. 预计时间：2-3 天

### 第二阶段：增强错误处理
1. 添加错误场景测试
2. 边界条件测试
3. 预计时间：1-2 天

### 第三阶段：集成和性能测试
1. 添加集成测试
2. 添加性能基准测试
3. 预计时间：2-3 天

---

## 测试最佳实践

### 使用 Table-Driven Tests
```go
func TestDecodeString(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    interface{}
        wantErr bool
    }{
        {
            name:    "valid json",
            input:   `{"key":"value"}`,
            want:    map[string]interface{}{"key": "value"},
            wantErr: false,
        },
        {
            name:    "empty string",
            input:   "",
            want:    nil,
            wantErr: true,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            decoder := NewDecoder()
            var got interface{}
            err := decoder.DecodeString(tt.input, &got)

            if (err != nil) != tt.wantErr {
                t.Errorf("DecodeString() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("DecodeString() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 测试辅助函数
```go
// 创建测试记录
func makeTestRecord(t *testing.T, op string, data interface{}) *Record {
    t.Helper()
    // ...
}

// 验证记录
func assertRecordEqual(t *testing.T, got, want *Record) {
    t.Helper()
    // ...
}

// 创建临时文件
func createTempFile(t *testing.T, content []byte) string {
    t.Helper()
    // ...
}
```

### Benchmark 示例
```go
func BenchmarkEncode(b *testing.B) {
    encoder := NewEncoder()
    data := makeComplexData()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = encoder.Encode(data)
    }
}

func BenchmarkEncodeSizes(b *testing.B) {
    sizes := []int{100, 1000, 10000, 100000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            encoder := NewEncoder()
            data := makeDataOfSize(size)

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = encoder.Encode(data)
            }
        })
    }
}
```
