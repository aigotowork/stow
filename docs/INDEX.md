# Stow 设计文档索引

## 快速导航

### 🏠 总览
- **[README](README.md)** - 项目概述、核心理念、设计决策
- **[架构设计](architecture.md)** - 分层架构、数据流、并发模型

### 📦 模块设计
- **[Core](modules/core.md)** - JSONL 编解码、版本管理、反向读取
- **[Blob](modules/blob.md)** - 大文件管理、流式读写、哈希计算
- **[Index](modules/index.md)** - Key 映射、缓存、TTL + Jitter
- **[Codec](modules/codec.md)** - 序列化、Struct Tag、Blob 检测
- **[FSUtil](modules/fsutil.md)** - 原子写入、文件系统工具

### 🔍 设计反思
- **[Could Be Better](couldbebetter.md)** - 14 个改进建议及优先级

---

## 文档统计

| 文档 | 行数 | 主题 |
|------|------|------|
| README | ~400 | 项目定位、核心价值、设计理念 |
| architecture | ~550 | 分层架构、数据流、性能分析 |
| couldbebetter | ~650 | 14 个设计问题及改进方案 |
| core | ~350 | JSONL、反向读取、容错 |
| blob | ~450 | 流式读写、去重、GC |
| index | ~400 | Key 清洗、冲突处理、缓存 |
| codec | ~450 | 序列化、Tag 解析、Blob 检测 |
| fsutil | ~250 | 原子写入、跨平台抽象 |
| **总计** | **~3500 行** | **完整设计文档体系** |

---

## 阅读路径

### 路径 1：快速了解（15 分钟）
1. [README](README.md) - 核心概念
2. [architecture](architecture.md) 前半部分 - 分层架构
3. 浏览各模块文档的"职责定位"部分

### 路径 2：深入理解（1 小时）
1. [README](README.md) - 完整阅读
2. [architecture](architecture.md) - 完整阅读
3. 选择感兴趣的模块详细阅读
4. [couldbebetter](couldbebetter.md) - 了解设计权衡

### 路径 3：贡献代码（2 小时）
1. 完整阅读所有文档
2. 重点关注 [couldbebetter](couldbebetter.md)
3. 选择一个改进项实施
4. 参考对应模块文档理解当前实现

---

## 关键概念索引

### 核心概念
- **JSONL 格式** → [core](modules/core.md#核心数据结构)
- **Blob 引用** → [blob](modules/blob.md#核心数据结构)
- **Key 清洗** → [index](modules/index.md#key-清洗sanitize)
- **Struct Tag** → [codec](modules/codec.md#struct-tag-解析)
- **原子写入** → [fsutil](modules/fsutil.md#原子写入atomicwritefile)

### 设计决策
- **为什么用 JSONL？** → [README](README.md#1-为什么使用-jsonl-而非-json)
- **为什么反向读取？** → [README](README.md#2-为什么反向读取文件)
- **为什么分离 Blob？** → [README](README.md#3-为什么分离-blob-存储)
- **为什么 TTL + Jitter？** → [README](README.md#5-为什么使用-ttl--jitter-缓存)

### 性能特征
- **时间复杂度** → [architecture](architecture.md#时间复杂度总结)
- **吞吐量估算** → [architecture](architecture.md#吞吐量估算)
- **并发模型** → [architecture](architecture.md#并发模型)

### 改进建议
- **高优先级（P0）** → [couldbebetter](couldbebetter.md#按优先级排序的改进)
- **中优先级（P1）** → [couldbebetter](couldbebetter.md#按优先级排序的改进)
- **低优先级（P2）** → [couldbebetter](couldbebetter.md#按优先级排序的改进)

---

## 文档维护

### 更新频率
- 重大架构变更：更新 [architecture](architecture.md)
- 新增模块：创建对应 `modules/xxx.md`
- 发现设计问题：追加到 [couldbebetter](couldbebetter.md)
- API 变更：更新 [README](README.md) 快速开始部分

### 文档风格
- ✅ 用语精炼，直击要点
- ✅ 代码示例清晰
- ✅ 说明"为什么"而非"是什么"
- ✅ 权衡明确，不回避问题

---

## 相关资源

- **源代码：** [../](../)
- **测试文档：** [../TEST_SUMMARY.md](../TEST_SUMMARY.md)
- **示例代码：** [../examples/](../examples/)
- **设计文档：** [../design.md](../design.md)（原始需求）
