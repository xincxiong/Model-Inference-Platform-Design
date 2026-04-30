# Draft: System Architecture Diagram Generation

## Source Document
- File: `模型推理云平台-产品方案.md`
- Target: Section 5.1 总体架构分层 (7-layer architecture)
- Output: `model-inference-platform-architecture.svg` + `model-inference-platform-architecture.png`

## Architecture Layers (from doc)

```
Layer 1: 接入层 (Access Layer)
  - Web Console (React/Next.js)
  - Responses API + Chat Completions
  - SDK (Py/JS/Go)
  - CLI 工具 (Go)
  - 第三方集成 (LangChain/LiteLLM etc.)

Layer 2: 网关与路由层 (Gateway & Routing)
  - API Gateway (Kong/Envoy)
  - Responses Adapter (Items ↔ Messages)
  - 认证鉴权 (JWT/API Key)
  - 速率限制 (RPM/TPM)
  - 智能路由 (KV-aware Router)

Layer 3: 控制面 (Control Plane)
  Row 1: 用户/团队服务, 计费服务, 端点管理, 模型注册中心, 负载均衡器, A/B测试, 模型评测
  Row 2: Prompt模板, 预算告警, 数据质量检查, 语义缓存, 模型版本管理, 微调调度, 成员管理(RBAC)

Layer 4: 数据面 (Data Plane) — GPU集群
  - MultiEngineRouter (vLLM/SGLang/Custom/Mock)
  - 专属端点 Dedicated Endpoints (HAMi GPU隔离)
  - 批量推理 + 微调训练集群
  - Agentic RL训练平台 (Megatron-LM + SGLang)

Layer 5: 调度与编排层 (Orchestration)
  - Volcano Scheduler (Gang Scheduling, Queue, Preemption)
  - HAMi Device Plugin (GPU Memory Isolation, 多厂商适配)
  - K8s 原生组件 (HPA/Deployment/Service/PVC)

Layer 6: 存储与数据层 (Storage & Data)
  - PostgreSQL (元数据)
  - Redis (缓存)
  - S3 (对象存储)
  - ClickHouse (分析)
  - Kafka (事件流)
  - LanceDB (AI数据/向量)

Layer 7: 可观测性 (Observability)
  - Prometheus + VictoriaMetrics
  - Grafana 仪表盘
  - OpenTelemetry + Jaeger
  - Loki + 日志聚合
```

## Arrow Flow Colors
- Blue `#2563eb`: Inference request flow (L1→L2→L3→L4)
- Green `#16a34a`: Resource scheduling (L4→L5→L6)
- Purple `#9333ea`: Data/async flow (L6→L7, internal L4→L6)
- Orange `#ea580c`: GPU cluster internal communication
- Gray `#6b7280`: Monitoring metrics

## Style: Flat Icon (Style 1)
- Background: `#ffffff`
- Layer containers: `#f8fafc` fill, `#e2e8f0` stroke, dashed borders
- Node boxes: `#ffffff` fill, `#d1d5db` stroke, rx=8px
- Font: Helvetica Neue, 14px labels, 12px sub-labels

## ViewBox
- `0 0 960 1400` (7 layers + title + legend + key features)

## Key Features to Highlight
1. 控制面/数据面分离
2. 多引擎路由 (vLLM/SGLang/Custom/Mock)
3. 跨域算力管理 (Volcano + HAMi)
4. AI原生数据底座 (LanceDB + Lance)
5. 全链路闭环
6. 可观测性全覆盖

## Generation Command
```bash
python3 gen_arch.py
# Output: model-inference-platform-architecture.svg
# Then: rsvg-convert -w 1920 model-inference-platform-architecture.svg -o model-inference-platform-architecture.png
```

## Script Location
Output script to: `/Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/output/gen_arch.py`
Then execute from that directory.
