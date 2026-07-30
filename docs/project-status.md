# ModelFleet 项目状态

## 当前阶段

阶段 1：ModelService v1alpha1 API 契约已完成，下一步实现最小 Reconcile 控制链路。

## 已完成

- 确定项目正式名称和职责边界
- 创建独立 GitHub 公共仓库
- 初始化 Kubebuilder Go Operator
- 锁定 Go 1.26 工具链
- 建立架构文档和 ADR
- 创建 ModelService CRD
- 创建 ModelService Controller 骨架
- 定义 ModelSpec
- 定义 RuntimeSpec
- 定义 ModelServiceSpec
- 定义 ModelServiceStatus
- 启用 Status 子资源
- 配置 kubectl 打印列和 msvc Short Name
- 配置 Runtime 枚举校验
- 配置副本数和端口范围校验
- 配置副本数和端口默认值
- 完成 EnvTest 测试环境
- 在真实 K3s API Server 验证 CRD 契约

## 当前 API

ModelServiceSpec：

- model.name
- model.version
- model.uri
- runtime.type
- runtime.image
- runtime.args
- replicas
- port
- resources

ModelServiceStatus：

- observedGeneration
- phase
- deploymentName
- serviceName
- readyReplicas
- availableReplicas
- conditions

## 当前架构决定

- ModelFleet 负责模型服务控制面
- KVCache-Serve、tiny-runtime 和 vLLM 属于数据面
- ModelService 是主要声明式 API
- 每个模型版本使用独立 Deployment 和 Service
- 第一版先实现单模型、单版本控制闭环
- Runtime 类型限制为 transformers、kvcache-serve 和 vllm
- replicas 默认值为 1，范围为 1 到 100
- port 默认值为 8000，范围为 1 到 65535
- 后续自研 ModelFleet Gateway
- 先实现手动回滚，再实现 SLO 自动回滚
- 本地 Kubernetes 使用 K3s

## 下一步

实现最小 ModelService Reconcile 链路：

    ModelService
        ↓
    查询或创建 Deployment
        ↓
    设置 OwnerReference
        ↓
    查询或创建 Service
        ↓
    更新 ModelService Status

## 未完成事项

- Deployment 构建逻辑
- Service 构建逻辑
- OwnerReference
- Status Conditions
- tiny-runtime
- Controller 单元与集成测试
- Controller Docker 镜像
- K3s Controller 部署
- Gateway
- 多版本发布
- 灰度与回滚
- 自动扩缩容
- Prometheus 和 Grafana
- SLO 发布分析
- 故障演练
- 发布报告
