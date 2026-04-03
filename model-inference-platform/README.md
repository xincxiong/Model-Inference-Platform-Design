# Model Inference Platform — Phase 1 MVP

模型推理云平台 MVP 实现，包含完整的后端 API 服务、前端控制台和本地开发环境。

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

## 功能特性 (Phase 1)

- ✅ 双格式 API: Chat Completions + Responses API
- ✅ Responses API 服务端状态管理 (store + previous_response_id)
- ✅ 7 种模型类型支持 (含 Mock 推理引擎)
- ✅ API Key 认证 + SHA256 + Redis 缓存
- ✅ Service Tier 限流 (auto / default / flex)
- ✅ Web Console: 模型列表(带类型筛选) + Playground + API Key 管理 + 用量统计
- ✅ 按 Token 计量 + 余额扣减 + Promo Code
- ✅ Prometheus + Grafana 监控
- ✅ OpenAI SDK 兼容迁移指南
