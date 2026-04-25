## 每日开发进展

- 记录口径：每日收尾时按日期追加，简述当日可交付变更，方便事后追溯。

### 2026-04-26
- **HAMi 基础集成（Phase 3 核心里程碑）**
  - 新增 `backend/internal/hami/` 包（`types.go` / `client.go` / `scheduler.go`），实现 HAMi Scheduler HTTP 客户端，支持 Binpack / Spread / Topology-aware 三种调度策略，本地开发自动降级为 Stub 模式（`HAMI_ENABLED=false`）
  - 更新 `internal/model/dedicated.go`：新增 `GPUMemoryMiB` / `GPUCores` / `SchedulerPolicy` / `TopologyAware` / `HardIsolation` / `ScheduledNode` / `PhysicalGPUID` 七个 HAMi 字段
  - 更新 `internal/handler/dedicated_endpoints.go`：Create 时触发 HAMi 调度决策，Patch 支持 GPU 资源字段更新，scanEndpoint 适配新列
  - 更新 `internal/store/postgres.go`：`dedicated_endpoints` 表通过幂等 `ALTER TABLE` 新增 7 个 HAMi 字段，兼容已有数据库
  - 更新 `internal/router/router.go` + `cmd/server/main.go`：注入 HAMi Scheduler，wire 到 DedicatedEndpointsHandler
  - 编译验证通过（`go build ./...` 零错误）
- **Kubernetes 部署层（新建 `deploy/` 目录）**
  - `deploy/kubernetes/`：namespace.yaml / configmap.yaml / secrets.yaml / postgres-statefulset.yaml / redis-statefulset.yaml / backend-deployment.yaml / frontend-deployment.yaml / ingress.yaml / vllm-worker.yaml（含 HAMi GPU 资源声明示例：`nvidia.com/gpumem: 20480` + `nvidia.com/gpucores: 60`）
  - `deploy/hami/`：values.yaml（HAMi Helm Chart 配置，含三种调度策略） / gpu-operator-values.yaml（NVIDIA GPU Operator，devicePlugin.enabled=false 交由 HAMi 接管） / install.sh（7 步一键安装脚本：cert-manager → GPU Operator → HAMi → 平台 K8s manifest）
- **多推理引擎支持（vLLM + SGLang + Custom）**
  - 新增 `backend/internal/engine/vllm.go`：vLLM OpenAI-compatible API 客户端，支持 SSE 流式解析
  - 新增 `backend/internal/engine/sglang.go`：SGLang OpenAI-compatible API 客户端，支持 RadixAttention + JSON mode
  - 新增 `backend/internal/engine/router.go`：多引擎路由器，支持环境变量配置（`INFERENCE_ENGINE` / `VLLM_ENDPOINT` / `SGLANG_ENDPOINT` / `CUSTOM_ENGINE_<MODEL_ID>`）+ 数据库模型引擎类型（`models.engine_type` / `models.engine_addr`）
  - 更新 `cmd/server/main.go`：使用 `MultiEngineRouter` 替代 `MockEngine`，默认引擎类型由环境变量控制
  - 更新 `internal/store/postgres.go`：`models` 表新增 `engine_type` / `engine_addr` 两字段
  - 更新 `deploy/kubernetes/configmap.yaml` + `docker-compose.yml`：添加引擎配置环境变量
  - 更新 `README.md`：新增推理引擎配置章节，说明 vLLM/SGLang/Custom/Mock 四种引擎使用方式
- **架构进展**：调度与编排层从 0% → 30%（HAMi 集成框架就绪），推理引擎层从 Mock → 多引擎支持（vLLM/SGLang/Custom）

### 2026-04-17
- **后端**: 更新 `cmd/server/main.go`、`internal/router/router.go`、`internal/store/postgres.go`、`internal/handler/billing.go`，完善模型/账单相关的路由与存储逻辑，支撑模型版本与用量聚合能力。
- **前端**: 重构 `deployments`、`endpoints`、`finetuning`、`usage` 页面，增加部署/端点列表与表单细节、用量趋势与成本分布可视化，整体 UI/交互更完整。
- **文档**: 同步产品方案与 HTML 设计稿至 2026-04-17 版本，校正微调、用量统计、模型版本管理等描述。
