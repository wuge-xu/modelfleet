# ADR-0001：项目边界与技术基线

- 状态：Accepted
- 日期：2026-07-29

## 背景

现有 KVCache-Serve 负责单个模型实例的推理、缓存和性能指标，但缺少多模型服务生命周期管理能力。

ModelFleet 用于补充模型服务控制面，统一管理模型部署、版本、流量、扩缩容、监控和回滚。

## 决定

### 1. 控制面与数据面分离

ModelFleet 是控制面。

tiny-runtime、KVCache-Serve 和 vLLM 属于数据面。

ModelFleet 不重新实现模型推理引擎。

### 2. ModelService 是主要声明式 API

用户通过 ModelService CR 描述模型、版本、运行时、资源和发布策略。

Controller 根据 ModelService 创建和维护 Kubernetes 资源。

### 3. 每个模型版本使用独立资源

每个模型版本拥有独立的 Deployment 和 Service。

不在同一个 Deployment 中混合多个模型版本。

### 4. 优先完成最小控制闭环

第一版只实现：

    ModelService
        ↓
    Deployment
        ↓
    Service
        ↓
    Status

多版本、Gateway、灰度、扩缩容和自动回滚在后续阶段实现。

### 5. 本地运行环境

本地 Kubernetes 使用现有 K3s 集群。

K3s containerd 负责集群工作负载运行。

Docker Desktop 用于镜像构建。

### 6. Go 工具链

项目使用 Go 1.26.0。

Kubebuilder 版本为 4.15.0。

Kubernetes 和 controller-runtime 依赖由 Kubebuilder 项目骨架管理。

### 7. API 命名

Go Module：

    github.com/wuge-xu/modelfleet

Kubebuilder Domain：

    modelfleet.io

API Group：

    serving.modelfleet.io

主要资源：

    ModelService

## 结果

项目拥有清晰的职责边界，可以从最小可运行版本逐步扩展，而不是一次拼接完整平台。
