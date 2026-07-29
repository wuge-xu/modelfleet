# ModelFleet 项目状态

## 当前阶段

阶段 0：项目基线与环境初始化。

## 已完成

- 确定项目正式名称
- 确定控制面与数据面边界
- 确定最小控制闭环
- 创建独立 Git 仓库
- 初始化 Kubebuilder Go Operator
- 配置 Go Module
- 验证基础代码可以编译和测试
- 使用 Go 1.26
- 建立架构与 ADR 文档目录

## 当前架构决定

- ModelFleet 负责模型服务控制面
- KVCache-Serve 和 vLLM 属于数据面
- ModelService 是主要声明式 API
- 每个模型版本使用独立 Deployment 和 Service
- 先实现单版本控制闭环
- 后续自研 ModelFleet Gateway
- 先实现手动回滚，再实现 SLO 自动回滚
- 本地 Kubernetes 使用 K3s

## 下一阶段

设计 ModelService v1alpha1 API。

第一版字段范围：

- 模型名称
- 模型版本
- 模型地址
- Runtime 类型
- Runtime 镜像
- 副本数
- 服务端口
- CPU 和内存资源
- Deployment 和 Service 状态

## 未完成事项

- ModelService CRD
- Controller Reconcile
- tiny-runtime
- Deployment 和 Service 创建
- Status Conditions
- 单元测试和 EnvTest
- Docker 镜像
- K3s 部署
- Gateway
- 多版本发布
- 灰度与回滚
- 自动扩缩容
- Prometheus 和 Grafana
- 故障演练
- 发布报告
