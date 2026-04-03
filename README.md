# Model Inference Cloud Platform

> 厂商中立的一站式模型推理云平台 — 全链路闭环 + AI 原生数据底座 + 国产化适配。

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
| [docs/index.html](docs/index.html) | GitHub Pages 对外展示页（含竞品对比与定位分析） |
| [model-inference-platform/](model-inference-platform/) | Phase 1 MVP 代码实现（Go 后端 + Next.js 前端） |

## 方案概览

**部署模式**：Serverless 共享推理 · 专属端点（Dedicated Endpoints）· 国产 GPU 异构混合部署

**功能模块**（13 个）：

> 推理引擎 · Playground · 专属端点 · 模型微调 · 数据实验室 · 批量推理 · 可观测性 · 团队管理 · 计费系统 · 第三方集成 · 迁移指南 · Cookbook · CLI 工具

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
Phase 1  MVP — 核心推理能力（vLLM 集群 · 双格式 API · Web Console · 基础计费）
    ↓
Phase 2  增强 — 专属端点与微调（Dedicated Endpoints · LoRA/Full FT · Batch API · 自动伸缩）
    ↓
Phase 3  生态 — 可观测性与集成（Data Lab · CLI · 第三方集成 · PD 分离 · MoE 并行）
    ↓
Phase 4  企业 — 安全合规（SSO/RBAC · ZDR · SOC 2/GDPR · VPC · Custom Models）
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
