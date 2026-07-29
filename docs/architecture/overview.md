# ModelFleet 架构概览

## 1. 系统目标

ModelFleet 为 Kubernetes 集群中的模型推理服务提供统一的声明式控制面。

用户创建 ModelService 资源描述模型服务，不需要直接管理底层 Deployment、Service、版本流量和扩缩容资源。

## 2. 控制面

控制面由 Go 编写的 Kubernetes Operator 构成。

第一阶段控制链路：

    ModelService
        ↓
    ModelServiceReconciler
        ↓
    计算期望状态
        ↓
    创建或更新 Deployment
        ↓
    创建或更新 Service
        ↓
    更新 ModelService Status

后续控制面还将管理：

- ModelRevision
- 灰度发布状态
- 自动扩缩容资源
- 发布分析
- 自动回滚
- 部署和升级报告

## 3. 数据面

数据面负责实际执行推理请求。

计划支持三类运行时：

1. tiny Transformers Runtime
2. KVCache-Serve
3. vLLM

不同运行时后续通过统一的 RuntimeAdapter 接口接入控制面。

## 4. 流量面

ModelFleet Gateway 将作为统一推理入口，负责：

- 模型路由
- stable 和 canary 版本选择
- 权重流量分配
- 请求排队
- 并发限制
- 限流
- 超时控制
- 请求指标

Gateway 不在最小控制闭环阶段实现。

## 5. 观测面

ModelFleet 将采集：

- Operator 控制循环指标
- Gateway 请求和队列指标
- 推理运行时指标

指标进入 Prometheus，并通过 Grafana 展示。

后续发布分析器将基于错误率、TTFT 和可用性 SLO 决定升级或回滚。

## 6. 第一版最小闭环

    ModelService CR
            ↓
    Go Controller
            ↓
    Deployment
            ↓
    Service
            ↓
    tiny-runtime
            ↓
    ModelService Status

第一版暂时不实现 Gateway、多版本、vLLM、灰度和自动扩缩容。
