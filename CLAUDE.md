# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **vendor-neutral model inference cloud platform** (模型推理云平台) — a full-stack Go + Next.js application providing:
- OpenAI-compatible inference APIs (Chat, Completions, Responses, Embeddings, Rerank, Images, Video, Speech)
- Dedicated endpoints with GPU virtualization (HAMi)
- Model fine-tuning (SFT + RL post-training) with Volcano scheduling
- Data lab, batch inference, team management, and billing

The project is organized in **5 phases**. Current status: Phase 1-2 complete, Phase 3 in progress (~75% overall implementation).

## Repository Structure

This repo contains **both** product design documents **and** the implementation codebase.

```
(obsidian-vault root)/
├── README.md                           # Top-level project overview + roadmap
├── 模型推理云平台-产品方案.md            # Full product spec (1500+ lines)
├── model-inference-platform-design.html # Visual architecture diagram
├── docs/                               # Additional docs (MRD, PRD, tech spec)
│
└── model-inference-platform/           # <-- ACTUAL CODEBASE (work here)
    ├── backend/                        # Go 1.23 + Gin API server
    │   ├── cmd/
    │   │   ├── inference-server/       # Data plane (port 8080)
    │   │   ├── management-server/      # Control plane (port 8081)
    │   │   └── server/                 # Monolithic mode (both servers)
    │   ├── internal/                   # 23 packages (see below)
    │   ├── migrations/                 # DB schema migrations
    │   └── Dockerfile
    ├── frontend/                       # Next.js 15 + React 19 + TailwindCSS
    │   └── src/app/                    # 11 page routes
    ├── deploy/                         # k8s manifests, HAMi, Volcano
    ├── monitoring/                     # Prometheus + Grafana config
    └── docker-compose.yml              # Local dev stack
```

## Key Commands

### Local Development (Docker)
```bash
cd model-inference-platform
docker-compose up --build          # Full stack (postgres, redis, backend, frontend, prometheus, grafana)
docker-compose up postgres redis -d # Only dependencies, then run backend/frontend locally
```

### Local Development (No Docker)
```bash
# Backend
cd model-inference-platform/backend
go run ./cmd/server                 # Monolithic (both inference + management)

# Frontend
cd model-inference-platform/frontend
npm install && npm run dev
```

### Backend Build
```bash
cd model-inference-platform/backend
go build ./cmd/inference-server    # Build inference server binary
go build ./cmd/management-server   # Build management server binary
```

### Frontend Build
```bash
cd model-inference-platform/frontend
npm run build                       # Production build
npm run start                       # Production server
```

## Backend Architecture

**Go 1.23 + Gin** with a clean layered architecture. Three binary entry points share the same `internal/` packages.

### Internal Packages (23 packages)

| Package | Purpose |
|---------|---------|
| `engine/` | Multi-engine router (vLLM, SGLang, Mock, Custom) |
| `handler/` | 17 HTTP handlers (chat, completions, responses, embeddings, rerank, images, models, fine-tuning, dedicated endpoints, datasets, batches, files, members, billing, etc.) |
| `store/` | PostgreSQL + Redis data access layer |
| `model/` | Domain models (7 model types, fine-tuning config) |
| `modelrouter/` | Model routing and resolution |
| `hami/` | HAMi GPU virtualization scheduler (binpack/spread/topology-aware) |
| `volcano/` | Volcano batch scheduling (Gang/Queue/Preemption/TTL/multi-cluster) |
| `workerpool/` | Worker pool with 4 load balancing strategies + health scoring |
| `loadbalancer/` | 5 load balancing strategies (least-conn/round-robin/consistent-hash/weighted/multi-region) |
| `autoscaler/` | Dedicated endpoint auto-scaling (QPS/queue depth/GPU utilization) |
| `semcache/` | Semantic cache (vector cosine similarity) |
| `storage/` | S3 object storage (MinIO compatible) |
| `budget/` | Cost budget alerts (multi-threshold, multi-channel) |
| `circuitbreaker/` | Circuit breaker for engine failover |
| `datavalidator/` | Fine-tuning data quality checks (ChatML/Instruction format) |
| `prompttemplate/` | Prompt template management (system/team/user scope, versioning) |
| `abtest/` | A/B testing framework (traffic split, statistical significance) |
| `benchmark/` | Model evaluation (MMLU/HumanEval/GSM8K/MATH/C-Eval) |
| `queue/` | Job queue management |
| `health/` | Health check endpoints |
| `middleware/` | Retry middleware, streaming timeout control |
| `errors/` | Standardized error codes (OpenAI-compatible) |
| `config/` | Viper-based configuration |

### API Routes

- **Inference** (`/v1/*` on port 8080): Chat Completions, Completions, Responses, Embeddings, Rerank, Images, Videos, Audio, Models, Fine-tuning
- **Control Plane** (`/v0/*` on port 8081): Dedicated Endpoints CRUD
- **Console** (`/api/*`): Models, API Keys, Usage, Billing

## Frontend Architecture

**Next.js 15 App Router** with TypeScript, TailwindCSS, Zustand for state management, Recharts for visualization, Lucide for icons.

### Pages (11 routes)
- `/` — Home
- `/models` — Model catalog (type filtering)
- `/playground` — Interactive model testing
- `/api-keys` — API key management
- `/usage` — Usage statistics + billing
- `/endpoints` — Dedicated endpoint management (56.5 KB)
- `/finetuning` — Fine-tuning UI (SFT + RL dual mode, 71.6 KB)
- `/batches` — Batch inference jobs
- `/datasets` — Dataset management
- `/deployments` — Model deployment management (63.6 KB)
- `/members` — Team member management

## Infrastructure

- **Database**: PostgreSQL 16 (single migration: `001_init.sql`)
- **Cache**: Redis 7
- **Monitoring**: Prometheus + Grafana (dashboards in `monitoring/grafana/`)
- **Kubernetes**: Full deployment manifests in `deploy/kubernetes/` (11 YAMLs)
- **HAMi**: GPU virtualization (`deploy/hami/`)
- **Volcano**: Batch job scheduling (`deploy/volcano/`)
- **KEDA**: Auto-scaling based on queue depth/GPU utilization

## Environment Variables

### Backend
| Variable | Default | Description |
|----------|---------|-------------|
| `INFERENCE_ENGINE` | `mock` | `vllm`, `sglang`, or `mock` |
| `VLLM_ENDPOINT` | `http://localhost:8000` | vLLM server URL |
| `SGLANG_ENDPOINT` | `http://localhost:30000` | SGLang server URL |
| `HAMI_ENABLED` | `false` | Enable real HAMi scheduling (requires K8s) |
| `VOLCANO_ENABLED` | `false` | Enable real Volcano scheduling (requires K8s) |
| `S3_ENABLED` | `false` | Enable S3 object storage |
| `SEM_CACHE_ENABLED` | `false` | Enable semantic cache |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |

### Frontend
| Variable | Description |
|----------|-------------|
| `NEXT_PUBLIC_API_URL` | Backend API URL |

## Workflow Notes

- **Git push**: Never push automatically — always show the user what will be pushed (`git log origin/main..HEAD --oneline`) and wait for explicit confirmation (see `.cursor/rules/git-push-policy.mdc`)
- **Branch**: Currently on `feature/phase3-inference-enhancements`
- **Preset data**: Default user `admin@example.com`, promo code `WELCOME50` ($50 credit), 18 pre-seeded models across 7 types
- **Product docs**: Full product specs exist in Chinese at the repo root and in `docs/` — reference these for feature requirements and implementation gaps
