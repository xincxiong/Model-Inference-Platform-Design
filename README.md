<p align="center">
  <img src="model-inference-platform/assets/logo.png" alt="Model Inference Cloud Platform" width="120" height="120" />
</p>

<h1 align="center">Model Inference Cloud Platform</h1>

<p align="center">厂商中立的一站式模型推理云平台 — 全链路闭环 + AI 原生数据底座 + 国产化适配。</p>

[![GitHub Pages](https://img.shields.io/badge/GitHub%20Pages-Live%20Demo-blue?logo=github)](https://xincxiong.github.io/Model-Inference-Platform-Design/)

## 核心竞争力

| 优势 | 说明 |
|------|------|
| **全链路闭环** | 推理 → Data Lab 数据采集 → SQL + 向量混合查询 → 微调训练 → 模型部署，消除跨工具数据搬运 |
| **AI 原生数据底座** | Lance v4.0 列式格式（100x 快于 Parquet）+ LanceDB 向量数据库（<1ms 语义缓存 + 混合搜索），Git-style 数据集版本管理 |
| **国产化算力适配** | 华为昇腾 910B/950 · 海光 K100 · 寒武纪 590，HAMi 异构中间件统一抽象，API 透明切换 |
| **厂商中立** | 不绑定模型（DeepSeek/Qwen/GLM/Kimi/Llama 7+ 厂商）、不绑定云，集成 30+ 第三方框架 |
| **Responses API** | 下一代推理接口，服务端状态管理 + 链式引用，多轮对话 Token 消耗降低 40-80% |
| **推理优化深度** | FlashAttention-4 · 3-bit KV 压缩 · EAGLE-3 投机 · PD 分离 · Dynamo 编排 · LanceDB 语义缓存 |
| **企业级安全** | RBAC + SSO + ZDR 零数据留存 + SOC 2 / GDPR / HIPAA，支付宝/微信支付 |

## 文档

| 文件 | 说明 |
|------|------|
| [模型推理云平台-产品方案.md](模型推理云平台-产品方案.md) | 完整产品方案文档（Markdown，1500+ 行） |
| [model-inference-platform-design.html](model-inference-platform-design.html) | 可视化产品架构设计（HTML，浏览器打开） |
| [model-inference-platform-architecture.svg](model-inference-platform-architecture.svg) | 系统架构图（SVG，7层架构可视化） |
| [docs/index.html](docs/index.html) | GitHub Pages 对外展示页（含竞品对比与定位分析） |
| [model-inference-platform/](model-inference-platform/) | Phase 1 & Phase 2 代码实现（Go 后端 + Next.js 前端） |

## 方案概览

**部署模式**：Serverless 共享推理 · 专属端点（Dedicated Endpoints）· 国产 GPU 异构混合部署

**功能模块**（15 个）：

> 推理引擎 · Playground · 专属端点 · **模型部署** · 模型微调 · 数据实验室 · 批量推理 · 可观测性 · **成员管理** · 团队管理 · 计费系统 · 第三方集成 · 迁移指南 · Cookbook · CLI 工具

**支持模型类型**（7 类）：

> Text-to-Text · Vision · Embedding · Rerank · Text-to-Image · Text-to-Video · Speech

**系统架构**：云原生微服务，7 层分层（接入层 → 网关路由 → 控制面 → 数据面 → 调度编排 → 存储数据 → 可观测性）

**技术选型**：

| 层级 | 核心技术 |
|------|---------|
| 推理引擎 | vLLM v0.18+ · SGLang · NVIDIA Dynamo · FlashAttention-4 · EAGLE-3 |
| 国产 GPU | vLLM-Ascend · LMDeploy · HAMi · 昇腾/海光/寒武纪 |
| 后端 | Go (Gin) · Kong/Envoy · Temporal · Kafka |
| 前端 | React 19 · Next.js 15 · TailwindCSS · Monaco Editor |
| 存储 | PostgreSQL 16 · Redis 7 · ClickHouse · S3 · Lance/LanceDB |
| 基础设施 | Kubernetes · GPU Operator · KEDA · Volcano |
| 可观测性 | Prometheus · Grafana · OpenTelemetry · Loki |

## 实施路线图

```
Phase 1  ✅ MVP — 核心推理能力（vLLM 集群 · 双格式 API · Web Console · 基础计费）
    ↓
Phase 2  ✅ 增强 — 专属端点与微调（Dedicated Endpoints · LoRA/QLoRA/Full FT · RL 后训练(GRPO/PPO/DPO等) · Batch API · 数据集管理 · 模型部署(Deployments) · 成员管理(Members)）
    ↓
Phase 3  🚀 进行中 — 生态 — 可观测性与集成（HAMi GPU 虚拟化 ✅ · Kubernetes 部署层 ✅ · Data Lab · CLI · 第三方集成 · PD 分离 · MoE 并行）
    ↓
Phase 4  企业 — 安全合规（SSO/RBAC · ZDR · SOC 2/GDPR · VPC · Custom Models）
```

## HAMi GPU 虚拟化集成（Phase 3）

[HAMi](https://github.com/Project-HAMi/HAMi)（CNCF Sandbox）是本平台调度与编排层的核心组件，实现单物理 GPU 的虚拟化切分与硬隔离。

### 核心能力

| 能力 | 说明 |
|------|------|
| **GPU 内存硬隔离** | `nvidia.com/gpumem`（MiB），容器超用即 OOM，杜绝 noisy neighbor |
| **算力配额** | `nvidia.com/gpucores`（0–100%），保障推理 SLA |
| **调度策略** | Binpack（共享集群，最大化利用率）/ Spread（专属端点，高可用）/ Topology-aware（MoE 专家并行，NVLink 优化）|
| **多厂商** | NVIDIA / 华为昇腾 / 寒武纪 MLU / 海光 DCU，统一 API 透明切换 |

### 快速部署

```bash
# 前提：kubectl 已配置集群管理员权限，Helm 3.x，Kubernetes 1.25+
cd model-inference-platform/deploy/hami
./install.sh

# 仅安装 HAMi（跳过 GPU Operator，适合已有 driver 的节点）
./install.sh --skip-gpu-operator

# 验证 GPU 资源
kubectl get nodes -o custom-columns=\
"NAME:.metadata.name,GPU:.status.capacity.nvidia\.com/gpu,GPU-MEM:.status.capacity.nvidia\.com/gpumem"
```

### 专属端点 GPU 资源声明

创建专属端点时可指定 HAMi GPU 资源（POST `/v0/dedicated_endpoints`）：

```json
{
  "name": "DeepSeek V3 专属推理",
  "model_name": "deepseek-ai/DeepSeek-V3",
  "gpu_type": "NVIDIA A100",
  "gpu_count": 1,
  "gpu_memory_mib": 20480,
  "gpu_cores": 60,
  "scheduler_policy": "binpack",
  "topology_aware": false,
  "hard_isolation": true,
  "min_replicas": 1,
  "max_replicas": 4
}
```

### 本地开发（无 Kubernetes）

无需 Kubernetes 即可运行，HAMi 自动降级为 Stub 模式：

```bash
# HAMI_ENABLED 不设置或设为 false，所有调度调用立即成功（stub）
cd model-inference-platform
docker compose up -d
```

## 推理引擎配置

平台支持多种推理引擎，用户可根据需求自由选择：

| 引擎 | 特点 | 适用场景 |
|------|------|---------|
| **vLLM** | PagedAttention + Continuous Batching，吞吐最高 | 共享推理集群、高并发场景 |
| **SGLang** | RadixAttention 自动前缀缓存，多轮对话更快 | Agent 多轮对话、长上下文场景 |
| **Mock** | 本地开发模拟，无需真实引擎 | 开发测试、CI/CD |
| **Custom** | 用户自定义引擎地址（LMDeploy/TensorRT-LLM 等） | 特殊模型、私有部署 |

### 环境变量配置

```bash
# 主引擎类型（vllm | sglang | mock）
INFERENCE_ENGINE=vllm

# vLLM endpoint（OpenAI-compatible API）
VLLM_ENDPOINT=http://localhost:8000

# SGLang endpoint（OpenAI-compatible API）
SGLANG_ENDPOINT=http://localhost:30000

# 单模型自定义引擎（格式：CUSTOM_ENGINE_<MODEL_ID>=<URL>）
CUSTOM_ENGINE_deepseek-ai/DeepSeek-R1=http://deepseek-r1-vllm:8000
```

### 启动真实引擎

```bash
# 启动 vLLM（DeepSeek V3）
docker run -d --gpus all -p 8000:8000 \
  vllm/vllm-openai:latest \
  --model deepseek-ai/DeepSeek-V3 \
  --tensor-parallel-size 4

# 启动 SGLang（Qwen3 72B）
docker run -d --gpus all -p 30000:30000 \
  lmsysorg/sglang:latest \
  --model Qwen/Qwen3-72B \
  --port 30000
```

### 数据库模型引擎配置

在 `models` 表中可设置每个模型的默认引擎：

```sql
UPDATE models SET engine_type = 'vllm' WHERE id = 'deepseek-ai/DeepSeek-V3';
UPDATE models SET engine_type = 'sglang' WHERE id = 'Qwen/Qwen3-72B';
UPDATE models SET engine_type = 'custom', engine_addr = 'http://my-custom-engine:8000'
  WHERE id = 'my-private-model';
```

## Kubernetes 部署文件结构

```
deploy/
├── kubernetes/
│   ├── namespace.yaml          # Namespace + RBAC
│   ├── configmap.yaml          # 平台配置
│   ├── secrets.yaml            # 密钥模板（需替换 base64 值）
│   ├── postgres-statefulset.yaml
│   ├── redis-statefulset.yaml
│   ├── backend-deployment.yaml # 含 HAMi env 注入
│   ├── frontend-deployment.yaml
│   ├── ingress.yaml            # Nginx Ingress + HPA
│   └── vllm-worker.yaml        # vLLM Worker + HAMi GPU 资源声明示例
└── hami/
    ├── values.yaml             # HAMi Helm Chart 配置
    ├── gpu-operator-values.yaml # NVIDIA GPU Operator 配置
    └── install.sh              # 一键安装脚本（7 步）
```

## 竞品对比

| 平台 | 推理 | 微调 | Data Lab | 专属端点 | CLI | 国产 GPU | 厂商中立 |
|------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **★ 本平台** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Together AI | ✅ | ✅ | — | ✅ | 部分 | — | ✅ |
| Fireworks AI | ✅ | ✅ | — | ✅ | — | — | ✅ |
| 硅基流动 | ✅ | ✅ | — | ✅ | — | 有限 | ✅ |
| 火山引擎方舟 | ✅ | ✅ | — | ✅ | — | ✅ | 绑定字节 |
| 阿里云百炼 | ✅ | ✅ | — | ✅ | — | ✅ | 绑定阿里 |

## License

MIT
