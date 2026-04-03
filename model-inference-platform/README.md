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
| `/v1/responses` | POST | Responses API (Items model) |
| `/v1/responses/:id` | GET | Get stored response |
| `/v1/models` | GET | List models |

### Console API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/models` | GET | Model list (detailed) |
| `/api/api-keys` | GET/POST | List / Create API keys |
| `/api/api-keys/:id` | DELETE | Delete API key |
| `/api/usage` | GET | Usage summary + daily breakdown |
| `/api/billing/redeem` | POST | Redeem promo code |

## 技术栈

- **Backend**: Go 1.23, Gin, pgx, go-redis
- **Frontend**: Next.js 15, React 19, TailwindCSS
- **Database**: PostgreSQL 16, Redis 7
- **Monitoring**: Prometheus, Grafana
- **Engine**: Mock (可替换为 vLLM)

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
- 8 个预置模型 (DeepSeek / Qwen / GLM / Llama / BGE / FLUX)
