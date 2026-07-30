# ModelFleet

ModelFleet 是一个基于 Kubernetes 和 Go Operator 构建的模型服务管控平台。

它负责多个模型服务的声明式部署、版本管理、流量路由、扩缩容、灰度发布、监控和回滚。

## 项目定位

ModelFleet 属于模型服务控制面。

推理运行时属于数据面，包括：

- tiny Transformers Runtime
- KVCache-Serve
- vLLM

ModelFleet 不重新实现模型推理引擎，而是管理推理运行时在 Kubernetes 中的生命周期。

## 核心链路

    ModelService CR
            ↓
    ModelFleet Controller
            ↓
    Deployment / Service / Status
            ↓
    ModelFleet Gateway
            ↓
    tiny-runtime / KVCache-Serve / vLLM

## 计划能力

- ModelService CRD
- Go Operator
- 模型和运行时配置
- 健康检查
- 模型版本管理
- 多版本部署
- 90/10 灰度流量
- 手动和自动回滚
- 请求路由
- 排队和限流
- 自动扩缩容
- Prometheus 指标
- SLO 发布分析
- 故障注入
- 部署、升级和回滚报告

## 当前阶段

阶段 1：ModelService v1alpha1 API 契约。

当前已经完成 CRD、API 类型、默认值、字段校验、Status 子资源和 EnvTest 验证。

## 技术栈

- Go 1.26
- Kubebuilder 4.15
- controller-runtime
- Kubernetes / K3s
- Docker
- Prometheus
- Grafana

## 本地验证

    go test ./...

## 开发原则

优先完成可以运行、可以测试、可以展示的最小闭环，再逐步增加多版本、流量治理、扩缩容和自动回滚能力。
