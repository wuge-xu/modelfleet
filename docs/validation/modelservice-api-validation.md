# ModelService v1alpha1 API 验证报告

- 验证日期：2026-07-30
- Kubernetes：K3s v1.35.5+k3s1
- CRD：modelservices.serving.modelfleet.io
- API Version：serving.modelfleet.io/v1alpha1

## 验证目标

验证 ModelService CRD 的默认值、字段校验、Status 子资源和 kubectl 展示配置。

## 验证结果

| 验证项 | 预期结果 | 实际结果 |
|---|---|---|
| Short Name | msvc | 通过 |
| Status 子资源 | 启用 | 通过 |
| 默认 replicas | 1 | 通过 |
| 默认 port | 8000 | 通过 |
| Runtime 枚举 | 仅允许 transformers、kvcache-serve、vllm | 通过 |
| replicas 最小值 | 不允许小于 1 | 通过 |
| port 最大值 | 不允许大于 65535 | 通过 |
| Printer Columns | Model、Version、Runtime、Ready、Phase、Age | 通过 |

## 结论

ModelService v1alpha1 的 OpenAPI Schema 已正确安装到 Kubernetes API Server。

默认值和字段验证均由 API Server 执行，不依赖 ModelFleet Controller 是否运行。

非法 ModelService 会在写入 etcd 之前被拒绝。
