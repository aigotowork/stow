# Codec 模块设计

## 职责定位

**Codec 模块是序列化/反序列化层**，负责：
1. 将 Go 结构体转换为 map[string]interface{}
2. 识别需要存为 Blob 的字段
3. 调用 Blob Manager 存储大文件
4. 反序列化时还原 Blob 数据
5. 支持 Struct Tag 自定义行为

## 核心挑战

### 挑战 1：如何识别 Blob 字段？

**用户数据：**
```go
type User struct {
    Name   string
    Avatar []byte  // 图片，应该存为 Blob
    Bio    string
}
```

**问题：** 如何判断 `Avatar` 应该存为 Blob？

**解决方案：** 多层策略
1. **类型判断** - `[]byte` 或 `io.Reader`
2. **大小判断** - 超过阈值（默认 1KB）
3. **Struct Tag** - `stow:"file"`
4. **选项覆盖** - `WithForceFile()`

### 挑战 2：如何获取 Blob 文件名？

**场景：**
```go
type User struct {
    Name       string
    AvatarData []byte     // 应该存为 avatar.jpg
    AvatarName string     // 文件名在这里
}
```

**解决：** Struct Tag 引用其他字段
```go
type User struct {
    Name       string
    AvatarData []byte `stow:"file,name_field:AvatarName"`
    AvatarName string
}
```

### 挑战 3：如何处理 Blob 缺失？

**场景：** 反序列化时 Blob 文件已被删除

**策略：**
- ❌ 不返回错误（避免阻塞整个 Get）
- ✅ 返回零值（nil）
- ✅ 记录警告日志

## Struct Tag 解析

### Tag 格式

```
stow:"option1,option2:value,option3:value"
```

### 支持的选项

| 选项 | 说明 | 示例 |
|------|------|------|
| `file` | 强制存为 Blob | `stow:"file"` |
| `name:xxx` | 指定文件名 | `stow:"file,name:avatar.jpg"` |
| `name_field:Xxx` | 从其他字段获取文件名 | `stow:"file,name_field:FileName"` |
| `mime:xxx` | 指定 MIME 类型 | `stow:"file,mime:image/jpeg"` |

### 解析实现

```go
type TagInfo struct {
    IsFile    bool
    Name      string
    NameField string
    MimeType  string
}

func ParseStowTag(tag string) TagInfo {
    info := TagInfo{}

    parts := strings.Split(tag, ",")
    for _, part := range parts {
        part = strings.TrimSpace(part)

        if part == "file" {
            info.IsFile = true
            continue
        }

        if strings.Contains(part, ":") {
            kv := strings.SplitN(part, ":", 2)
            key, value := kv[0], kv[1]

            switch key {
            case "name":
                info.Name = value
            case "name_field":
                info.NameField = value
            case "mime":
                info.MimeType = value
            }
        }
    }

    return info
}
```

### Tag 使用示例

```go
type Document struct {
    Title    string
    Content  []byte `stow:"file,name:doc.pdf,mime:application/pdf"`
}

type Photo struct {
    Caption  string
    Image    []byte `stow:"file,name_field:FileName"`
    FileName string
}
```

## Marshaler - 序列化器

### 工作流程

```
Go Struct
    ↓
ToMap() → map[string]interface{}
    ↓
遍历字段
    ↓
  Blob?
   ↓   ↓
  Yes  No
   ↓    ↓
 Store 保留
   ↓    ↓
 返回  返回
 引用  原值
    ↓
最终 map
```

### 核心实现

```go
func (m *Marshaler) Marshal(value interface{}, opts MarshalOptions) (
    map[string]interface{},
    []*blob.Reference,
    error,
) {
    // 1. 转换为 map
    data, _ := ToMap(value)

    var blobRefs []*blob.Reference

    // 2. 遍历字段
    for key, fieldValue := range data {
        // 3. 判断是否需要存为 Blob
        shouldStore, blobData := m.shouldStoreAsBlob(fieldValue, opts)
        if !shouldStore {
            continue
        }

        // 4. 存储 Blob
        ref, _ := m.blobManager.Store(blobData, opts.FileName, opts.MimeType)

        // 5. 替换为引用
        data[key] = ref.ToMap()
        blobRefs = append(blobRefs, ref)
    }

    return data, blobRefs, nil
}
```

### Blob 检测逻辑

```go
func (m *Marshaler) shouldStoreAsBlob(value interface{}, opts MarshalOptions) (bool, interface{}) {
    // 1. 已经是 Blob 引用？
    if m, ok := value.(map[string]interface{}); ok {
        if blob.IsBlobReference(m) {
            return false, nil
        }
    }

    // 2. io.Reader 类型？
    if reader, ok := value.(io.Reader); ok {
        return true, reader
    }

    // 3. []byte 类型？
    if bytes, ok := value.([]byte); ok {
        // 检查：强制文件 OR 大小超阈值
        if opts.ForceFile || int64(len(bytes)) > opts.BlobThreshold {
            return true, bytes
        }
    }

    return false, nil
}
```

**优先级：**
```
WithForceFile() > Struct Tag > 大小阈值
```

### ToMap - 结构体转换

```go
func ToMap(value interface{}) (map[string]interface{}, error) {
    val := reflect.ValueOf(value)

    // 解引用指针
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    // 已经是 map？
    if val.Kind() == reflect.Map {
        result := make(map[string]interface{})
        iter := val.MapRange()
        for iter.Next() {
            keyStr := iter.Key().Interface().(string)
            result[keyStr] = iter.Value().Interface()
        }
        return result, nil
    }

    // 转换 struct
    if val.Kind() != reflect.Struct {
        return nil, errors.New("value must be struct or map")
    }

    result := make(map[string]interface{})
    typ := val.Type()

    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        fieldType := typ.Field(i)

        // 跳过未导出字段
        if !field.CanInterface() {
            continue
        }

        // 获取字段名（支持 json tag）
        fieldName := fieldType.Name
        if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
            fieldName = strings.Split(jsonTag, ",")[0]
        }

        result[fieldName] = field.Interface()
    }

    return result, nil
}
```

**支持的类型：**
- ✅ Struct
- ✅ Map (map[string]interface{})
- ✅ 指针（自动解引用）
- ❌ Slice / Array（需要包装成 struct）

## Unmarshaler - 反序列化器

### 工作流程

```
map[string]interface{}
    ↓
遍历字段
    ↓
 Blob 引用?
   ↓     ↓
  Yes    No
   ↓     ↓
 Load   直接
 Blob   赋值
   ↓     ↓
 赋值   ↓
   ↓     ↓
 填充 Go Struct
```

### 核心实现

```go
func (u *Unmarshaler) Unmarshal(data map[string]interface{}, target interface{}) error {
    val := reflect.ValueOf(target)

    // 必须是指针
    if val.Kind() != reflect.Ptr {
        return errors.New("target must be pointer")
    }

    elem := val.Elem()

    // 遍历字段
    for key, fieldValue := range data {
        field := elem.FieldByName(key)
        if !field.IsValid() || !field.CanSet() {
            continue
        }

        // 检查是否是 Blob 引用
        if m, ok := fieldValue.(map[string]interface{}); ok {
            if blob.IsBlobReference(m) {
                // 加载 Blob
                u.loadBlob(m, field)
                continue
            }
        }

        // 直接赋值
        field.Set(reflect.ValueOf(fieldValue))
    }

    return nil
}
```

### Blob 加载逻辑

```go
func (u *Unmarshaler) loadBlob(refMap map[string]interface{}, field reflect.Value) error {
    ref, _ := blob.FromMap(refMap)

    // 检查 Blob 是否存在
    if !u.blobManager.Exists(ref) {
        log.Warn("blob missing", ref.Location)
        // 设置零值
        field.Set(reflect.Zero(field.Type()))
        return nil
    }

    // 根据目标类型加载
    switch field.Type() {
    case reflect.TypeOf([]byte{}):
        // 加载为 []byte
        data, _ := u.blobManager.LoadBytes(ref)
        field.SetBytes(data)

    case reflect.TypeOf((*IFileData)(nil)).Elem():
        // 加载为 IFileData（延迟加载）
        fileData, _ := u.blobManager.Load(ref)
        field.Set(reflect.ValueOf(fileData))

    default:
        // 不支持的类型
        return errors.New("unsupported blob target type")
    }

    return nil
}
```

**支持的 Blob 目标类型：**
- `[]byte` - 完整加载到内存
- `IFileData` - 返回文件句柄（流式读取）

## 设计决策

### 1. 为什么 Marshal 返回 map 而非 []byte？

**方案 A：** 返回 []byte（JSON 字节）
```go
func Marshal(value interface{}) ([]byte, error)
```

**方案 B：** 返回 map ✅
```go
func Marshal(value interface{}) (map[string]interface{}, error)
```

**理由：**
- Blob 替换需要在 map 层面操作
- 避免 JSON 编码 → 解码 → 编码的冗余
- 分层清晰：Codec → Core（Core 负责 JSON 编码）

### 2. 为什么 Blob 缺失不返回错误？

**场景：** 用户手动删除了 Blob 文件

**方案 A：** 返回错误
```go
err := ns.Get("user", &user)
// → error: blob missing
```

**问题：**
- ❌ 阻塞整个 Get 操作
- ❌ 其他字段也无法读取

**方案 B：** 返回零值 + 警告 ✅
```go
err := ns.Get("user", &user)
// → nil
// user.Avatar = nil（零值）
// log.Warn("blob missing: avatar.jpg")
```

**优势：**
- ✅ 不阻塞
- ✅ 部分数据可用
- ✅ 日志中有警告

### 3. 为什么不支持嵌套 Struct？

**当前限制：**
```go
type User struct {
    Name    string
    Profile Profile  // ❌ 不支持
}
```

**原因：**
- 复杂度高（递归序列化）
- 实际需求少（扁平化设计更清晰）
- 可通过 JSON 字段变通：
  ```go
  type User struct {
      Name    string
      Profile string `json:"profile"`  // JSON 字符串
  }
  ```

## 选项模式

### MarshalOptions

```go
type MarshalOptions struct {
    BlobThreshold int64  // Blob 阈值
    ForceFile     bool   // 强制存为文件
    FileName      string // 文件名
    MimeType      string // MIME 类型
}
```

### PutOption 函数

```go
// 强制存为文件
func WithForceFile() PutOption

// 指定文件名
func WithFileName(name string) PutOption

// 指定 MIME 类型
func WithMimeType(mime string) PutOption
```

**使用示例：**
```go
ns.Put("doc", docData,
    stow.WithForceFile(),
    stow.WithFileName("document.pdf"),
    stow.WithMimeType("application/pdf"),
)
```

## 性能特征

### 时间复杂度

| 操作 | 时间复杂度 | 说明 |
|------|------------|------|
| ToMap | O(n) | n = 字段数 |
| Marshal | O(n + b) | b = Blob 字段数 |
| Unmarshal | O(n) | n = 字段数 |

### 内存占用

```
Marshal:   O(字段数) - 临时 map
Unmarshal: O(1) - 就地修改
```

## 测试覆盖

**16 个单元测试，覆盖率 45.0%**

关键测试：
- ✅ Struct Tag 解析（7 种组合）
- ✅ Marshal 简单结构体
- ✅ Marshal 大/小字节数组
- ✅ Marshal 强制文件
- ✅ Unmarshal 简单结构体
- ✅ Marshal/Unmarshal 往返
- ✅ MarshalReader
- ✅ Blob 缺失容错
- ✅ StoreBytesAsBlob

## 实际使用示例

```go
// 1. 创建 Marshaler/Unmarshaler
blobManager, _ := blob.NewManager("./data/_blobs", 10*1024*1024, 64*1024)
marshaler := codec.NewMarshaler(blobManager)
unmarshaler := codec.NewUnmarshaler(blobManager)

// 2. 序列化
type User struct {
    Name   string
    Avatar []byte `stow:"file,name:avatar.jpg,mime:image/jpeg"`
}

user := User{
    Name:   "Alice",
    Avatar: avatarBytes,
}

data, blobRefs, _ := marshaler.Marshal(user, codec.MarshalOptions{
    BlobThreshold: 1024,
})

// 3. 反序列化
var retrieved User
unmarshaler.Unmarshal(data, &retrieved)

// 4. 使用选项
marshaler.MarshalBytes(data, codec.MarshalOptions{
    ForceFile: true,
    FileName:  "data.bin",
    MimeType:  "application/octet-stream",
})
```

## 相关模块

- 上层：[Namespace](../namespace.go) - 调用 Codec 序列化
- 下层：[Blob](blob.md) - Codec 调用 Blob 存储
- 平级：[Core](core.md) - Codec 返回 map 给 Core 编码

## 潜在改进

参见 [couldbebetter.md](../couldbebetter.md#codec-模块)
