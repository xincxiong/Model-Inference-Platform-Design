<p align="center">
  <img src="assets/logo.svg" alt="Model Inference Platform — 多模型汇聚至云端推理端点" width="132" height="132" />
</p>

<h1 align="center">Model Inference Platform</h1>

<p align="center"><strong>Phase 1–2</strong> · 模型推理云平台</p>

模型推理云平台：Phase 1 MVP + Phase 2 专属端点与微调（管理面 `/v0`、OpenAI 风格微调任务 API）。

**Logo 含义**：深蓝圆角底座象征稳定云底座；上方三节点汇聚到中心端点，表示多模型路由与统一推理出口；顶部箭头暗示请求流入与低延迟响应。青蓝渐变与控制台强调色一致。

## Quick Start

```bash
# 启动所有服务 (PostgreSQL + Redis + Backend + Frontend + Prometheus + Grafana)
docker-compose up --build

# 访问
# - Frontend Console: http://localhost:3000
# - Backend API:      http://localhost:8080
# - Prometheus:       http://localhost:9090
# - Grafana:          http://localhost:3001 (admin/admin)
```

## 项目结构

```
├── assets/           品牌资源（如 logo.svg）
├── backend/          Go API 服务 (Gin)
├── frontend/         Next.js 15 控制台
├── monitoring/       Prometheus + Grafana 配置
├── docs/             迁移指南
└── docker-compose.yml
```

## API Endpoints

### Inference (OpenAI Compatible)

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
- **Engine**: Mock (可替换为 vLLM / SGLang)

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
