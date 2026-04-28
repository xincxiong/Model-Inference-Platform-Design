<p align="center">
  <img src="assets/logo.png" alt="Model Inference Platform — 多模型汇聚至云端推理端点" width="128" height="128" />
</p>

<h1 align="center">Model Inference Platform</h1>

<p align="center"><strong>Phase 1–3</strong> · 模型推理云平台 · v0.3.0</p>

模型推理云平台：Phase 1 MVP + Phase 2 专属端点与微调 + Phase 3 多引擎支持、HAMi GPU 虚拟化、Volcano 调度与编排层、GPU 虚拟化方案（MIG/vGPU/国产算力）、跨域算力管理设计、控制面/数据面分离。

**Logo 含义**：深蓝圆角底座象征稳定云底座；上方三节点汇聚到中心端点，表示多模型路由与统一推理出口；顶部箭头暗示请求流入与低延迟响应。青蓝配色与控制台强调色一致。README 使用 **PNG** 以保证在 GitHub 上稳定显示。

## Quick Start

```bash
# 启动所有服务 (PostgreSQL + Redis + Inference Server + Management Server + Frontend + Prometheus + Grafana)
docker-compose up --build

# 访问
# - Frontend Console:    http://localhost:3000
# - Inference Server:    http://localhost:8080
# - Management Server:   http://localhost:8081
# - Prometheus:          http://localhost:9090
# - Grafana:             http://localhost:3001 (admin/admin)
```

## 项目结构

```
├── assets/           品牌资源（logo.png）
├── backend/          Go API 服务 (Gin)
│   ├── cmd/
│   │   ├── inference-server/    推理数据面（/v1/chat/completions 等）
│   │   ├── management-server/   控制面（/v0/dedicated_endpoints 等）
│   │   └── server/              单体模式（同时启动两个服务）
│   ├── internal/
│   │   ├── engine/              多推理引擎（vLLM/SGLang/Mock/Custom）
│   │   ├── hami/                HAMi GPU 虚拟化调度
│   │   ├── volcano/             Volcano 批量作业调度（Gang/Queue/Preemption）
│   │   ├── handler/             HTTP 处理器（17 个）
│   │   ├── modelrouter/         模型路由与解析
│   │   ├── router/              路由注册（分离为两个函数）
│   │   ├── store/               数据访问层（PostgreSQL + Redis）
│   │   └── ...
│   └── deploy/
│       ├── kubernetes/          K8s 部署清单（11 个 YAML）
│       ├── hami/                HAMi Helm Chart 配置
│       └── volcano/             Volcano Helm Chart + Job 示例 + 安装脚本
├── frontend/         Next.js 15 控制台
├── monitoring/       Prometheus + Grafana 配置
├── docs/             迁移指南
└── docker-compose.yml
```

## API Endpoints

### Inference (OpenAI Compatible) — Inference Server (port 8080)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | Chat Completions (SSE streaming) |
| `/v1/completions` | POST | Text Completions (FIM 代码补全) |
| `/v1/responses` | POST | Responses API (Items model, store, previous_response_id) |
| `/v1/responses/:id` | GET | Get stored response |
| `/v1/responses/:id/input_items` | GET | Get response input items |
| `/v1/embeddings` | POST | Text embeddings (向量嵌入) |
| `/v1/rerank` | POST | Document reranking (重排序) |
| `/v1/images/generations` | POST | Image generation (图像生成) |
| `/v1/videos/generations` | POST | Video generation (视频生成) |
| `/v1/audio/transcriptions` | POST | Speech-to-text (语音转文本) |
| `/v1/audio/speech` | POST | Text-to-speech (文本转语音) |
| `/v1/models` | GET | List models (OpenAI 兼容格式) |
| `/v1/fine_tuning/jobs` | POST | 创建微调任务（MVP：排队 → 运行 → 成功，产出合成 `fine_tuned_model` id） |
| `/v1/fine_tuning/jobs` | GET | 列出当前用户的微调任务 |
| `/v1/fine_tuning/jobs/:id` | GET | 查询单个任务 |
| `/v1/fine_tuning/jobs/:id/cancel` | POST | 取消排队中/运行中的任务 |

**专属推理路由**：创建端点后，将 `routing_key`（形如 `ep_ab12cd34:deepseek-ai/DeepSeek-V4`）作为 `/v1/chat/completions` 等接口的 `model` 字段；平台会校验端点归属与 `running` 状态，并路由到专属引擎键（当前与共享池相同为 Mock 引擎）。

### 控制面 `/v0`（需 Bearer API Key）

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v0/dedicated_endpoints/templates` | GET | 可作为专属端点基座的模型（text-to-text / vision） |
| `/v0/dedicated_endpoints` | GET / POST | 列表 / 创建专属端点 |
| `/v0/dedicated_endpoints/:id` | PATCH / DELETE | 更新（名称、副本、状态等）/ 删除 |

### Console API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/models` | GET | Model list (含 speed/quality/features) |
| `/api/api-keys` | GET/POST | List / Create API keys |
| `/api/api-keys/:id` | DELETE | Delete API key |
| `/api/usage` | GET | Usage summary + daily breakdown |
| `/api/billing/redeem` | POST | Redeem promo code |

## 支持的模型类型

| 类型 | 说明 | 示例模型 |
|------|------|---------|
| text-to-text | 对话补全、代码生成 | DeepSeek-V4, Qwen3.5, GLM-5, Llama 4, Kimi K2.5, 豆包 2.0 |
| vision | 图文理解、视觉问答 | Qwen2.5-VL, InternVL3 |
| embedding | 文本向量化、语义检索 | BGE-M3, Jina Embeddings v3 |
| rerank | 文档重排序 | BGE Reranker v2, Qwen3 Reranker |
| text-to-image | 图像生成 | FLUX.1, SDXL |
| text-to-video | 视频生成 | CogVideoX |
| speech | 语音合成与识别 | CosyVoice 2, SenseVoice |

## 技术栈

- **Backend**: Go 1.23, Gin, pgx, go-redis
- **Frontend**: Next.js 15, React 19, TailwindCSS
- **Database**: PostgreSQL 16, Redis 7
- **Monitoring**: Prometheus, Grafana
- **Engine**: vLLM / SGLang / Mock / Custom
- **Scheduling**: HAMi (GPU 虚拟化) + Volcano (批量作业调度) + K8s 原生组件
- **GPU 虚拟化**: NVIDIA MIG (H100/A100) + HAMi vGPU + 国产算力虚拟化 (昇腾 CANN/海光 ROCm/寒武纪 Neuware)

## 本地开发 (不用 Docker)

```bash
# 1. 启动依赖
docker-compose up postgres redis -d

# 2. 启动后端
cd backend && go run ./cmd/server

# 3. 启动前端
cd frontend && npm install && npm run dev
```

## 预置数据

- 默认用户: `admin@example.com`
- Promo Code: `WELCOME50` (充值 $50)
- 18 个预置模型，覆盖 7 大类型 (Text/Vision/Embedding/Rerank/Image/Video/Speech)

## 功能特性

**Phase 1**

- ✅ 双格式 API: Chat Completions + Responses API
- ✅ Responses API 服务端状态管理 (store + previous_response_id)
- ✅ 7 种模型类型支持 (含 Mock 推理引擎)
- ✅ API Key 认证 + SHA256 + Redis 缓存
- ✅ Service Tier 限流 (auto / default / flex)
- ✅ Web Console: 模型列表(带类型筛选) + Playground + API Key 管理 + 用量统计
- ✅ 按 Token 计量 + 余额扣减 + Promo Code
- ✅ Prometheus + Grafana 监控
- ✅ OpenAI SDK 兼容迁移指南

**Phase 2**

- ✅ 专属端点：`dedicated_endpoints` 表 + `/v0/dedicated_endpoints*` + `routing_key` 推理校验
- ✅ 微调任务：`fine_tuning_jobs` 表 + `/v1/fine_tuning/jobs`（异步占位流水线）
- ✅ 控制台：专属端点页、模型微调页

**Phase 3**

- ✅ HAMi GPU 虚拟化：`internal/hami/` 包，Binpack/Spread/Topology-aware 三种调度策略，HAMI_ENABLED=false 自动 Stub 降级
- ✅ Kubernetes 部署层：`deploy/kubernetes/`（11 个 YAML）+ `deploy/hami/`（Helm values + 安装脚本）
- ✅ 多推理引擎支持：`internal/engine/` 包（vllm.go/sglang.go/router.go），MultiEngineRouter 替代 MockEngine
- ✅ 控制面/数据面分离：`cmd/inference-server/`（8080）+ `cmd/management-server/`（8081），独立扩缩容与故障隔离
- ✅ Volcano 调度与编排层：`internal/volcano/` 客户端，支持 Gang Scheduling/Queue 隔离/PriorityClass/TTL 清理；`deploy/volcano/`（Helm values + Job 示例 + 安装脚本）；环境变量配置（VOLCANO_ENABLED/VOLCANO_NAMESPACE/VOLCANO_QUEUE/VOLCANO_JOB_IMAGE/VOLCANO_SCHEDULER_POLICY/VOLCANO_PRIORITY_CLASS/VOLCANO_TTL_SECONDS/VOLCANO_MIN_AVAILABLE）
- ✅ GPU 虚拟化方案：产品方案 3.3.1 章节完整设计（NVIDIA MIG/HAMi vGPU/昇腾 NPU/海光 DCU/寒武纪 MLU 虚拟化）
- ✅ 跨域算力管理设计：产品方案 4.3'.8 章节完整设计（全局算力资源池/跨域队列调度/算力利用率优化/监控仪表盘）
- ✅ 前端架构增强：错误边界、Zustand 全局状态、乐观更新、错误处理工具、通知容器
- ✅ KEDA 自动伸缩：基于队列深度/GPU 利用率自动扩缩容 Worker Pod
- ✅ Prometheus 监控：Prometheus + Grafana 监控栈部署配置

**Phase 1-2 补齐功能**

- ✅ Worker Pool 增强：4 种负载均衡策略（最小负载/轮询/一致性哈希/随机）、健康评分系统、多区域支持、自动健康检查
- ✅ 统一错误码标准化：`internal/errors/` 包，兼容 OpenAI API 规范的错误码体系
- ✅ 请求重试中间件：`internal/middleware/retry.go`，指数退避 + 抖动，可配置重试策略
- ✅ 流式超时控制：`internal/middleware/streaming_timeout.go`，TTFT/Token 间超时/总超时控制
- ✅ 专属端点自动扩缩容：`internal/autoscaler/` 包，支持 QPS/队列深度/GPU 利用率/复合指标四种策略
- ✅ 成本预算告警：`internal/budget/` 包，多级阈值（50%/80%/100%）、多渠道通知、告警历史记录
- ✅ 微调数据质量检查：`internal/datavalidator/` 包，ChatML/Completion/Instruction 格式验证、角色不平衡检测、质量评分
- ✅ Prompt 模板管理：`internal/prompttemplate/` 包，系统/团队/用户三级作用域、变量插值、版本管理
- ✅ 负载均衡器增强：`internal/loadbalancer/` 包，最少连接/轮询/一致性哈希/加权/多区域五种策略
- ✅ A/B 测试框架：`internal/abtest/` 包，流量分流、统计显著性计算、实时切换
- ✅ 模型评测 Benchmark：`internal/benchmark/` 包，MMLU/HumanEval/GSM8K/MATH/C-Eval 标准评测集、微调前后对比
- ✅ S3 对象存储：文件上传/下载优先走 S3（`internal/storage/s3store.go`），支持 MinIO/S3 兼容，环境变量：
  - `S3_ENABLED` (true/false)
  - `S3_ENDPOINT` / `S3_BUCKET` / `S3_REGION`
  - `S3_ACCESS_KEY` / `S3_SECRET_KEY`
  - `S3_PATH_STYLE` (true for MinIO)
- ✅ 语义缓存（Semantic Cache）：`internal/semcache/` 包，基于向量余弦相似度命中缓存响应，减少重复推理成本，环境变量：
  - `SEM_CACHE_ENABLED` (true/false)
  - `LANCEDB_URI` (预留 LanceDB 持久化路径)
  - `SEM_CACHE_EMBED_MODEL` (嵌入模型 ID，如 `text-embedding-3-small`)
  - `SEM_CACHE_THRESHOLD` (相似度阈值，默认 0.85)
  - `SEM_CACHE_TOPK` (TopK 搜索，默认 3)
- ✅ Video/Speech Handler：`/v1/videos/generations`（视频生成）、`/v1/audio/transcriptions`（语音转文本）、`/v1/audio/speech`（文本转语音），已接入模型路由与计费
- ✅ 多集群 Volcano 调度：`internal/volcano/multi_cluster.go`，支持跨集群资源选择、GPU/CPU/内存使用跟踪、自动刷新、主集群故障转移


## 文档

| 文件 | 说明 |
|------|------|
| [ALIGNMENT.md](ALIGNMENT.md) | 产品方案 vs 代码实现 对齐分析报告 |
| [docs/migration-guide.md](docs/migration-guide.md) | 从 OpenAI 迁移到本平台的指南 |
| [模型推理云平台-产品方案.md](../模型推理云平台-产品方案.md) | 完整产品方案文档（v2.5） |
