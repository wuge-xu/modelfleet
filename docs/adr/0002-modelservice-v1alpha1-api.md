# ADR-0002：ModelService v1alpha1 API

- 状态：Accepted
- 日期：2026-07-30

## 背景

ModelService 是 ModelFleet 的主要声明式 API。

第一版 API 必须支持单模型、单版本和单运行时部署，同时为后续版本管理、灰度发布和扩缩容保留演进空间。

## 决定

### Spec

第一版 ModelServiceSpec 包含：

- model.name
- model.version
- model.uri
- runtime.type
- runtime.image
- runtime.args
- replicas
- port
- resources

Runtime 类型限制为：

- transformers
- kvcache-serve
- vllm

replicas 使用指针类型，以区分用户未填写与用户显式配置，并由 CRD 默认值设置为 1。

### Status

第一版 ModelServiceStatus 包含：

- observedGeneration
- phase
- deploymentName
- serviceName
- readyReplicas
- availableReplicas
- conditions

observedGeneration 用于表明 Controller 已处理到哪个 spec generation。

Conditions 用于表达 Ready、Progressing 和 Degraded 等可组合状态。

### 延后能力

第一版暂不在 API 中加入：

- 多版本 Revision
- stable 和 canary
- 流量权重
- 自动扩缩容
- SLO
- 自动回滚
- Gateway 路由

这些能力在最小控制闭环验证后逐步加入。

## 结果

ModelService v1alpha1 保持范围可控，同时能够描述第一版推理服务 Deployment 和 Service 所需的全部输入。
