# Stow KV Storage - 核心操作

**概念**: Store 管理多个 Namespace，每个 Namespace 是独立的 KV 存储空间（JSONL 格式）。如果为文件时，

**三个核心方法**:

```go
// 初始化
store := stow.MustOpen("/path/to/data")
ns, err := store.GetNamespace("namespace_name")

// 存储 - 支持任意可序列化类型
ns.Put(key string, value interface{}) error

// 读取 - 反序列化到 target
ns.Get(key string, target interface{}) error

// 删除 - 软删除
ns.Delete(key string) error
```

**示例**:
```go
err := ns.Put("config", map[string]interface{}{"port": 8080})

// 读取
var config map[string]interface{}
err := ns.Get("config", &config)

// 删除
err := ns.Delete("config")
```

**返回值**: 所有方法在键不存在时返回 `ErrNotFound`。
