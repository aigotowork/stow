# Codec 模块测试改进建议

## 当前状态
- **覆盖率**: 60.6% (从 52.8% 提升)
- **测试用例数**: 43 个
- **问题**: 所有测试集中在一个文件 (1700+ 行)，部分核心功能未测试

## 测试文件重构建议

### 建议的测试文件结构

```
internal/codec/
├── tag.go              -> tag_test.go
├── marshal.go          -> marshal_test.go
├── unmarshal.go        -> unmarshal_test.go
├── reflect.go          -> reflect_test.go
├── blob_test.go        (新增 - Blob 相关测试)
└── integration_test.go (新增 - 集成测试)
```

---

## 1. Tag 测试 (tag_test.go)

### 当前覆盖情况
- ✅ 已测试: ParseStowTag (95%)
- ❌ 未测试: HasStowTag, IsEmpty, ShouldStoreAsBlob (0%)

### 需要添加的测试场景

#### 基础功能测试
```go
// TestHasStowTag - 测试是否有 stow 标签
TestHasStowTag
  - 有标签的字段
  - 没有标签的字段
  - 空标签
  - 只有 json 标签

// TestTagIsEmpty - 测试标签是否为空
TestTagIsEmpty
  - 空 TagInfo
  - 部分字段为空
  - 所有字段都有值

// TestShouldStoreAsBlob - 测试是否应存储为 blob
TestShouldStoreAsBlob
  - 有 "file" 标签
  - 大于阈值的 []byte
  - io.Reader 类型
  - 普通类型
```

#### 标签解析测试
```go
TestParseStowTagEdgeCases
  - 无效格式
  - 重复属性
  - 空白字符
  - 特殊字符在值中
  - Unicode 字符
```

---

## 2. Marshal 测试 (marshal_test.go)

### 当前覆盖情况
- ✅ 已测试: Marshal, MarshalBytes, MarshalReader, MarshalSimple
- ⚠️ 部分测试: StoreBytesAsBlob

### 需要添加的测试场景

#### Struct Marshal 测试
```go
TestMarshalComplexStruct
  - 嵌套结构
  - 循环引用检测
  - 接口字段
  - 匿名字段

TestMarshalWithBlobFields
  - 多个 blob 字段
  - Blob 字段命名冲突
  - name_field 引用
  - 自定义 MIME 类型
```

#### Map Marshal 测试
```go
TestMarshalMapWithBlobs
  - map 值为 []byte
  - map 值为 io.Reader
  - map 值为 struct (包含 blob)
  - 嵌套 map

TestMarshalMapKeyTypes
  - string 键
  - 非 string 键错误处理
```

#### Slice Marshal 测试
```go
TestMarshalSliceWithBlobs
  - [][]byte
  - []io.Reader
  - []struct (包含 blob)
```

#### 错误处理测试
```go
TestMarshalErrors
  - nil 指针
  - 不支持的类型
  - Blob 存储失败
  - 循环引用
```

---

## 3. Unmarshal 测试 (unmarshal_test.go)

### 当前覆盖情况
- ✅ 已测试: Unmarshal, UnmarshalSimple (基础场景)
- ❌ 未测试: unmarshalToMap (0%), loadBlobAsFileData (0%)
- ⚠️ 部分测试: loadBlobIntoField (50%)

### 需要添加的测试场景

#### Map 目标测试 (unmarshalToMap)
```go
TestUnmarshalToMapWithBlobs
  - map 值包含 blob 引用
  - 加载 blob 到 map
  - Blob 加载失败处理
  - 混合 blob 和普通值

TestUnmarshalToMapEdgeCases
  - nil map
  - 空 map
  - 嵌套 map with blobs
```

#### Blob 加载测试
```go
TestLoadBlobAsFileData
  - 加载为 IFileData 接口
  - 文件句柄管理
  - 延迟加载
  - 大文件处理

TestLoadBlobIntoFieldTypes
  - []byte 目标
  - IFileData 接口目标
  - 不支持的类型
  - nil 字段
```

#### Interface 字段测试
```go
TestUnmarshalToInterface
  - interface{} 字段
  - 具体接口类型
  - 接口指针
```

#### 错误恢复测试
```go
TestUnmarshalWithPartialFailure
  - 部分 blob 缺失
  - 部分字段类型不匹配
  - 日志记录验证
```

---

## 4. Reflect 测试 (reflect_test.go)

### 当前覆盖情况
- ✅ 已测试: setFieldValue (92.9%), IsSimpleType
- ⚠️ 部分测试: ToMap (58.3%), FromMap (40%)

### 需要添加的测试场景

#### ToMap 完整测试
```go
TestToMapWithTimeTypes
  - time.Time
  - *time.Time
  - 自定义时间类型

TestToMapWithPointers
  - 指向基础类型的指针
  - 指向 struct 的指针
  - 多级指针
  - nil 指针

TestToMapWithTags
  - json 标签
  - omitempty
  - 标签优先级
```

#### FromMap 完整测试
```go
TestFromMapTypeConversions
  - 数值类型转换（int -> int64, float64 -> int）
  - 字符串转数值
  - 接口赋值
  - 类型不兼容处理

TestFromMapWithPointers
  - nil 指针初始化
  - 指针字段赋值
  - 指针链
```

#### ExtractBlobFields 测试
```go
TestExtractBlobFieldsThreshold
  - 不同阈值测试
  - 边界值（阈值-1, 阈值, 阈值+1）

TestExtractBlobFieldsTypes
  - []byte 字段
  - io.Reader 字段
  - 带标签字段
  - 嵌套结构中的 blob
```

---

## 5. Blob 集成测试 (blob_test.go)

### 新增 Blob 相关测试

```go
TestBlobReferenceRoundTrip
  - Marshal with blob -> Unmarshal
  - 验证 blob 内容一致性
  - 验证 blob 元数据

TestBlobWithDifferentSizes
  - 小文件 (< 1KB)
  - 中等文件 (1MB)
  - 大文件 (> 10MB)
  - 空文件

TestBlobMimeTypeDetection
  - 显式指定 MIME type
  - 自动检测 MIME type
  - 未知类型处理

TestBlobNameGeneration
  - 使用 name 标签
  - 使用 name_field
  - 自动生成名称
  - 名称冲突处理

TestBlobCleanup
  - 成功后清理临时 blob
  - 失败后清理临时 blob
  - 孤儿 blob 检测
```

---

## 6. 集成测试 (integration_test.go)

### 完整流程测试

```go
TestCodecEndToEnd
  - 复杂结构 Marshal -> Unmarshal
  - 包含多个 blob 的结构
  - 嵌套结构
  - 混合类型

TestCodecWithRealBlobs
  - 真实文件作为 blob
  - 图片、文档、数据文件
  - 大文件处理

TestCodecErrorRecovery
  - Blob 损坏恢复
  - 部分数据丢失
  - 版本不兼容
```

### 并发测试

```go
TestConcurrentMarshal
  - 并发 marshal 多个对象
  - 共享 BlobManager

TestConcurrentUnmarshal
  - 并发 unmarshal
  - 并发 blob 加载

TestConcurrentMarshalUnmarshal
  - 同时进行 marshal 和 unmarshal
```

---

## 测试数据组织建议

### 创建测试辅助文件

```
internal/codec/
├── testdata/
│   ├── fixtures.go        # 测试 fixture 定义
│   ├── samples/           # 示例数据文件
│   │   ├── sample.txt
│   │   ├── sample.jpg
│   │   └── sample.pdf
│   └── testhelpers.go     # 测试辅助函数
```

### Fixtures 示例

```go
// testdata/fixtures.go
package testdata

type ComplexStruct struct {
    Name     string
    Data     []byte
    Metadata map[string]interface{}
    Tags     []string
    Nested   *NestedStruct
}

type NestedStruct struct {
    ID      string
    Content []byte `stow:"file,name:nested.bin"`
}

// 预定义测试数据
var (
    SimpleStruct = &ComplexStruct{
        Name: "test",
        Data: []byte("test data"),
    }

    ComplexStructWithBlobs = &ComplexStruct{
        // ... 复杂数据
    }
)
```

---

## 优先级建议

### 🔴 高优先级 (立即添加)
1. unmarshalToMap 测试 (当前 0% 覆盖)
2. loadBlobAsFileData 测试 (当前 0% 覆盖)
3. Tag 工具函数测试 (HasStowTag, IsEmpty, ShouldStoreAsBlob)

### 🟡 中优先级 (第二阶段)
1. 完善 ToMap 和 FromMap 测试
2. Blob 集成测试
3. 错误处理和边界测试

### 🟢 低优先级 (优化阶段)
1. 并发测试
2. 性能测试
3. 压力测试

---

## 预期效果

实施上述测试改进后：
- **覆盖率目标**: 60.6% → **80%+**
- **测试文件数**: 1 → 6
- **测试用例数**: 43 → **100+**
- **单文件行数**: 1700+ → 200-400 每个文件

---

## 实施步骤

### 第一阶段：重构现有测试 (1-2 天)
1. 将 codec_test.go 拆分为多个文件
2. 按功能模块组织测试
3. 提取公共辅助函数

### 第二阶段：补充核心测试 (2-3 天)
1. 添加未覆盖功能的测试
2. 重点关注 unmarshalToMap 和 blob 相关功能
3. 确保所有公开 API 都有测试

### 第三阶段：完善测试 (1-2 天)
1. 添加边界测试
2. 添加错误处理测试
3. 添加集成测试

### 第四阶段：优化 (可选)
1. 添加并发测试
2. 添加性能基准测试
3. 添加模糊测试

---

## 测试质量指标

### 覆盖率目标
- **整体覆盖率**: 80%+
- **核心功能覆盖率**: 95%+
- **错误处理覆盖率**: 70%+

### 测试质量
- 每个测试函数职责单一
- 测试命名清晰（遵循 Test<Function><Scenario> 模式）
- 充分的边界测试
- 良好的错误消息

### 可维护性
- 测试文件大小适中 (200-500 行)
- 复用测试辅助函数
- 使用 table-driven tests
- 良好的文档注释
