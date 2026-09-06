# Agent 训推平台 调研报告

**调研日期**：2026-06-29
**报告范围**：覆盖 Agent（智能体）**训练**与**推理**两侧的端到端平台
**信息截止**：2026-01（基于公开资料，部分产品功能以 2025 年发布版本为准）

---

## 一、调研范围与定义

"Agent 训推平台"指同时具备以下两侧能力的平台：

- **训练侧**：支持对 Agent 能力进行微调（SFT / DPO / RLHF / Agentic RL），能采集多轮轨迹、工具调用结果作为训练数据
- **推理侧**：提供生产级 Agent 运行时——多步工具调用、沙箱化代码执行、记忆/状态管理、长会话持久化、可观测性

**不包含**（这些是相关但独立赛道）：
- 纯模型训练平台（Hugging Face、Axolotl、LLaMA-Factory）
- 纯 Agent 开发框架（LangGraph、CrewAI、AutoGen、Agno）
- 纯推理服务平台（Fireworks、Together）

---

## 二、能力维度矩阵

下表从 6 个维度评估主流玩家。✅ = 完整支持，◐ = 部分支持/有方案但需集成，○ = 弱/路线图上，— = 不支持

| 平台 | SFT | RL/DPO | Agentic RL | 工具/Function | 代码沙箱 | Computer/Browser Use | MCP 托管 | 长会话(Threads/Runs) | 轨迹→训练闭环 |
|---|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| OpenAI | ✅ | ✅(RLHF) | ○ | ✅ | ✅ | ✅(Operator/CUA) | ○ | ✅(Responses/AgentKit) | ◐(生产轨迹可蒸馏) |
| Anthropic | — | — | — | ✅ | ✅ | ✅(Computer Use) | ✅(首创) | ◐(API 风格) | — |
| Google Vertex | ✅ | ◐ | ◐ | ✅ | ✅ | ✅(ADK) | ◐ | ✅(Agent Engine) | ◐ |
| AWS Bedrock AgentCore | ◐ | ◐ | — | ✅ | ✅ | ◐ | ◐ | ✅ | — |
| Azure AI Foundry | ✅ | ◐ | ◐ | ✅ | ✅ | ◐ | ◐ | ✅ | ◐ |
| 阿里云 百炼 + AgentScope | ✅ | ◐ | ◐ | ✅ | ✅ | ◐ | ○ | ✅ | ◐ |
| 字节 扣子 (Coze) | — | — | — | ✅ | ✅ | ◐ | ○ | ✅(对话流) | — |
| 腾讯 TokenHub | — | — | — | ◐ | ◐ | ○ | ○ | ◐ | — |
| LangChain LangGraph | — | — | — | ✅ | ◐ | ◐ | ✅ | ✅(自建) | ◐(LangSmith 采集) |
| CrewAI | — | — | — | ✅ | ◐ | ◐ | ◐ | ✅ | — |
| Hugging Face smolagents | ◐ | — | — | ✅ | ✅ | ◐ | ◐ | ✅(本地) | ◐ |
| Prime Intellect | — | ✅(分布式 RL) | ✅ | — | — | — | — | — | ✅(主业) |
| Anyscale (Ray) | ◐ | ✅(RLlib) | ◐ | — | — | — | — | — | ✅(主业) |
| Together AI | ✅ | ◐ | — | ◐ | ◐ | — | — | — | ◐ |
| Fireworks AI | ✅ | ◐ | — | ✅(FireFunction) | ◐ | — | — | — | — |
| Manus | — | — | — | ✅ | ✅ | ✅ | ○ | ✅ | — |
| **我方平台（路线图目标）** | ✅ | ✅ | ✅(Megatron+SGLang) | ◐ | ○ | — | ○ | ○(路线图) | ◐(Data Lab) |

---

## 三、玩家分类与画像

### 3.1 闭源前沿实验室（Tier 1）

#### OpenAI — AgentKit / Responses API / Operator
- **定位**：一体化 Agent 开发与部署平台（2025 年 OpenAI DevDay 推出 AgentKit 正式 GA）
- **训练侧**：
  - Supervised Fine-Tuning 支持 GPT-4o / GPT-4.1 / o1 全系
  - Reinforcement Fine-Tuning（RFT）支持 o 系列推理模型
  - "Production traces → training data" 工具链（`traces` API），可对生产 Agent 运行做 SFT
- **推理侧**：
  - **Responses API**：合并 Assistants + Chat Completions，原生支持多轮工具调用、内置工具（web/file/computer）
  - **AgentKit**：可视化画布编排 + Connector Registry（500+ 集成）
  - **Operator / Computer-Using Agent (CUA)**：浏览器 + 桌面双模操控
  - **Deep Research**：内置研究 Agent，异步 + 报告生成
- **差异化**：最完整闭环 + 工具生态最丰富 + 最强 reasoning 模型（o-series）
- **短板**：闭源、不支持私有部署、不开源工具链

#### Anthropic — Claude + MCP + Computer Use
- **定位**：前沿模型 + 首创 MCP 协议 + Computer Use 范式
- **训练侧**：**不公开提供 fine-tuning API**（仅企业客户 POC 通道）—— 这是与 OpenAI 最显著的差异
- **推理侧**：
  - **MCP（Model Context Protocol）**：开放标准，2024 年发布后成为行业事实标准；2025 年 OpenAI / Google 先后宣布支持
  - **Computer Use**：Claude 3.5/4 Sonnet 可直接操作桌面（截图→动作循环）
  - **Skills**：可复用的工具包（类似 MCP server 包装）
  - **Agent SDK**：开源 Python/TS SDK，循环调用 Claude + 工具
- **差异化**：MCP 标准制定者 + 最强长上下文（200K-1M）+ Computer Use 首创
- **短板**：无 fine-tuning → 不能为客户定制 Agent 行为

#### Google — Agent Development Kit (ADK) + Agent Engine + A2A
- **定位**：Gemini 模型 + ADK 开源框架 + Vertex AI 托管运行时
- **训练侧**：
  - Vertex AI Model Garden 提供 Gemini fine-tuning（TPU 加速）
  - RLHF via Vertex AI Pipelines
  - "Tuning Studio" 实验管理
- **推理侧**：
  - **Agent Development Kit (ADK)**：开源 Python/JS 框架，与 Gemini 深度集成
  - **Agent Engine**：托管运行时，Sessions / Memory / Artifacts 抽象
  - **Agent2Agent (A2A) Protocol**：跨厂商 Agent 互操作协议（与 Anthropic MCP 互补）
  - **Computer Use**：Gemini 2.5 CUA 模型
- **差异化**：ADK 开源 + A2A 协议 + TPU 算力 + Workspace 深度集成
- **短板**：企业级落地慢、生态分散（GCP 强绑定）

### 3.2 云厂商（Tier 2）

#### AWS Bedrock AgentCore
- **定位**（2025 年 re:Invent 发布 GA）：企业级 Agent 运行时
- **核心组件**：
  - **Runtime**：会话隔离执行环境（microVM 级隔离）
  - **Memory**：短期 + 长期记忆
  - **Identity**：跨账户/跨服务的 Agent 身份
  - **Gateway**：工具/MCP 统一接入
  - **Code Interpreter**：安全代码沙箱
  - **Browser Tool**：远程浏览器
  - **Observability**：CloudWatch + LangSmith 集成
- **训练侧**：**不自研训练**，靠合作伙伴（SageMaker + 自定义）
- **差异化**：企业合规 + AWS 生态 + microVM 隔离（vs 容器级）
- **短板**：训练侧薄弱 + 集成复杂度高

#### Azure AI Foundry Agent Service
- **定位**：Microsoft 一站式 Agent 平台，与 Azure 生态深度集成
- **核心组件**：
  - Foundry SDK + Agent Service
  - Connected Agents（Agent 互联）
  - Azure Functions / Logic Apps 作为工具
  - 与 Copilot Studio / M365 Copilot 双向桥接
- **训练侧**：
  - Azure OpenAI Fine-tuning（GPT-4o / o1）
  - 与 Phi 系列开源模型微调
- **差异化**：M365 / Teams / Outlook 集成 + AutoGen / Semantic Kernel 开源
- **短板**：价格贵 + 跨云体验割裂

#### 阿里云 百炼 + AgentScope
- **定位**：阿里通义大模型一站式平台
- **训练侧**：
  - Qwen / Qwen-VL / Qwen-Coder 全系 SFT + RLHF
  - PAI 平台提供 RL 训练（基于 Megatron）
- **推理侧**：
  - **百炼应用**：可视化 Agent 编排（Coze 风格）
  - **AgentScope**（开源）：多 Agent 框架，支持分布式部署
  - **ModelStudio**：模型广场 + Playground
- **差异化**：国内合规 + 国产 GPU 适配 + Qwen 生态
- **短板**：Agent 编排能力弱于专业平台

#### 字节 扣子 (Coze)
- **定位**：C 端/中小企业 Agent 工厂（"Agent 界的 App Store"）
- **训练侧**：**不提供模型微调**，专注于推理 + 工作流编排
- **推理侧**：
  - 可视化工作流（拖拽节点）
  - 1000+ 官方插件（搜索、图像、数据库…）
  - Bot 商店 + 知识库 + 长记忆
  - 多渠道发布（豆包 / 飞书 / 微信）
- **差异化**：零代码 + 插件生态 + 字节系流量入口
- **短板**：不能定制模型 + 复杂 Agent 能力上限低

### 3.3 AI 原生（Tier 3）

#### LangChain / LangGraph
- **定位**：开源 Agent 编排事实标准
- **核心组件**：
  - **LangGraph**：状态机式 Agent 框架（low-level + 高可控）
  - **LangSmith**：Agent 轨迹/可观测性/评估平台
  - **LangChain**：高层 SDK 封装
- **训练侧**：**不直接提供**，但 LangSmith 采集的轨迹可导出做 SFT
- **推理侧**：
  - LangGraph Platform（2025 GA）：托管运行时、长期记忆、Human-in-the-Loop
  - MCP 客户端（早期支持）
- **差异化**：生态最大 + LangSmith 评估能力 + 开源
- **短板**：易用性 / 学习曲线 + 不提供模型

#### CrewAI
- **定位**：多 Agent 协作框架（"Role-playing" 心智）
- **训练侧**：无
- **推理侧**：
  - Crew / Agent / Task / Tool 抽象
  - Flow：事件驱动工作流（2025 推出）
  - Enterprise：私有部署、审计、RBAC
- **差异化**：多 Agent 心智简单直观 + Enterprise 合规
- **短板**：复杂场景需 LangGraph 兜底 + 不开源 runtime

#### Hugging Face smolagents + AutoTrain
- **定位**：开源 Agent + 模型训练一体化
- **训练侧**：
  - AutoTrain：SFT/DPO/RLHF 全支持
  - TRL/PEFT 集成
- **推理侧**：
  - **smolagents**：极简 Agent 框架（<1000 行）
  - **Hf Inference Endpoints**：私有部署
  - **Spaces**：社区 Demo 托管
- **差异化**：开放权重 + 训练+部署一体化 + 社区生态最大
- **短板**：企业级 Agent 编排弱 + 运行时隔离不足

### 3.4 RL/Agent 训练专项（Tier 4）

#### Prime Intellect
- **定位**：分布式 Agent RL 训练（"去中心化 RL" 范式）
- **核心**：
  - INTELLECT-2 模型：用全球分散算力训练
  - **Synth Labs** 子品牌：合成轨迹数据工厂
  - 重点是训练 sidecasting，推理靠 vLLM/TGI
- **差异化**：去中心化训练 + 合成数据 + RLAIF
- **短板**：B2B 商业化早期

#### Anyscale (Ray)
- **定位**：Ray 生态，AI/RL 训练基础设施
- **核心**：
  - **RLlib**：分布式 RL 库（10+ 算法，PPO/GRPO/IMPALA…）
  - **Anyscale Platform**：托管 Ray 集群
  - 与 vLLM / Hugging Face 集成做 Agent RL
- **差异化**：生产级 RL 基础设施 + Ray 生态
- **短板**：不直接做 Agent 编排、需用户自建

### 3.5 推理 + 轻量微调（Tier 5）

#### Together AI
- **SFT/RL**：Together Fine-tuning（开源模型 LoRA/QLoRA/全参）
- **推理**：Together Inference（vLLM 后端，多模型）
- **Agent 能力**：函数调用支持 + Agent 工具栈（Together Agentic Stack）
- **差异化**：开源模型 + 训练 + 推理一体
- **短板**：闭源工具不丰富

#### Fireworks AI
- **SFT/RL**：FireFunction V2 微调 + RLHF 支持
- **推理**：超低延迟 inference（自研优化）
- **Agent 能力**：FireFunction V2 是开源 function calling SOTA 模型
- **差异化**：function calling 专项 + 极低延迟
- **短板**：生态小于 Together

#### Manus（独立公司，2025 年爆款）
- **定位**：通用 Agent 产品（"能自己做 PPT / 写代码 / 订机票"）
- **训练侧**：**未公开**，推测基于 Claude/GPT API + 自研 orchestration
- **推理侧**：
  - 多步规划 + 浏览器/工具调用
  - 云端沙箱执行
- **差异化**：产品体验 + 中文市场
- **短板**：商业模式争议 + 平台能力闭源

---

## 四、关键技术与产品趋势

### 4.1 从"模型即产品"到"Agent 即产品"
- 2024 年主流：卖 GPT-4 / Claude API
- 2025-2026：卖 Agent 运行时（OpenAI AgentKit、AWS AgentCore、Vertex Agent Engine）
- 训练目标从"会回答"转向"会执行多步任务"

### 4.2 Agent Harness / Runtime 成为新基础设施
- 类似数据库之于 Web 应用：Agent 平台提供
  - **会话隔离**（microVM / gVisor / Firecracker）
  - **工具沙箱**（代码执行、文件、网络）
  - **状态管理**（Memory、Artifacts、Threads）
  - **可观测性**（step-level trace、token usage、cost attribution）
- 三大云都已推出：AgentCore（AWS）、Agent Engine（Google）、Foundry（Azure）

### 4.3 训练数据范式转变
- 旧：人工标注 SFT 数据
- 新：
  - **生产轨迹回灌**：用户真实 Agent run → 过滤 → SFT（OpenAI traces、LangSmith 都支持）
  - **合成轨迹**：用强模型生成多步任务轨迹（Prime Intellect Synth Labs）
  - **环境生成**：构建可重放的环境（Terminal-Bench、AgentBench、SWE-bench）
- Agentic RL 训练对**轨迹质量 + 环境可重放性**要求远高于传统 RL

### 4.4 MCP 成为工具/Agent 互联事实标准
- 2024-11 Anthropic 发布 MCP
- 2025：OpenAI Agents SDK、Google ADK、AWS AgentCore 全部原生支持
- 2026：MCP server 数量爆发，**MCP server 托管/市场** 成为新赛道（参考 Cloudflare MCP Server、Smithery）

### 4.5 工具调用 + 结构化输出深度优化
- 主流推理框架全部内置 XGrammar / Outlines 约束解码
- 并行工具调用（parallel_tool_calls）成为标准
- 工具结果流式回传（SSE）优化首字延迟

### 4.6 评测范式转变
- 旧：MMLU/HumanEval 静态基准
- 新：
  - **轨迹级评估**（τ-bench、SWE-bench Verified、AgentBench）
  - **环境驱动评估**（Terminal-Bench、GAIA）
  - **多轮 SFT 评估**（多步任务成功率而非单轮准确率）

### 4.7 国产化与私有化
- 信创要求推动国产 GPU + 私有部署 Agent 平台
- 阿里百炼、华为盘古、智谱、DeepSeek、Moonshot 都加码 Agent 套件
- 对应能力：国产卡 + 私有集群 + 离线模型

---

## 五、对我方平台的差距与机会

### 5.1 能力对照（基于产品方案 §4.4.3 / §4.10.3 / Phase 4 路线图）

| 维度 | 我方现状 | 关键差距 |
|---|---|---|
| **Agentic RL 训练** | ✅ 路线图已有（slime：Megatron+SGLang），6 个 rollout 场景含 `agent_single` / `agent_multi` | 实现深度——目前是架构图，需要交付：沙箱化的 tool exec + browser env + 多步轨迹采集 |
| **SFT/RL 全栈** | ✅ 已实现 | — |
| **MCP 托管** | ○ 路线图 | 应在 Phase 3/4 之间插入——MCP 已是行业标准，缺位会被对手甩开 |
| **Threads/Runs API** | ○ 路线图 | 与 OpenAI Responses API / AgentKit 兼容，**对标必要性高** |
| **工具沙箱（代码执行）** | ○ 缺失 | RL rollout 必备，落地优先级 ≥ MCP |
| **Browser Use** | — 缺失 | 影响 rollouts `agent_single` 实际可用性 |
| **轨迹存储 / Replay** | ◐ Data Lab 半支持 | 需要结构化 TrajectoryStore（messages + tool_calls + tool_results + rewards） |
| **Agent 评测** | ◐ benchmark 包存在 | 缺 τ-bench / SWE-bench / Terminal-Bench 这类 agent 评测 harness |
| **MCP server 托管** | — 缺失 | 行业新标配（Cloudflare、Smithery 已做） |

### 5.2 三个差异化机会点

#### 机会 1：Agent 训推一体化（业内稀缺）
- **现状**：OpenAI / Anthropic 强训弱推/强推弱训；LangChain / CrewAI 强推弱训；Anyscale 强训弱推
- **我方机会**：用 slime + Data Lab 闭环做"训练→轨迹采集→再训练"的飞轮
- **落地动作**：
  - 把 `/v1/threads` + `/v1/runs` 的生产轨迹自动入库 Data Lab
  - 标注后直接喂 SFT/RL pipeline
  - 这正是 Together AI、Fireworks 都没做闭环的原因（他们没 Data Lab）

#### 机会 2：国产 GPU + 私有部署 + Agent 一体机
- **现状**：信创 + 金融/政企客户需要全栈国产化
- **我方机会**：HAMi 已经在做国产卡适配（昇腾/天数），加 Agent 平台后打包成"国产 Agent 一体机"
- **落地动作**：
  - 路线图里"Agent 平台"和"国产 GPU"两条线合并推进
  - 客户拿到一台昇腾服务器 + Agent 平台 = 开箱即用

#### 机会 3：MCP server 托管（市场空白）
- **现状**：MCP server 托管分散在 Smithery、Cloudflare、Glama 等独立平台
- **我方机会**：把 MCP server 托管 + 算力池 + 安全审计打包，**Agent 调用外部工具时走平台中介**（可控 + 可计费 + 可审计）
- **落地动作**：
  - `/v1/mcp/servers` CRUD + 一键部署
  - 调用走平台网关，**按调用计费**（新收入流）
  - 对标 Cloudflare MCP Server 但加上计费/审计维度

### 5.3 风险与短板

| 风险 | 影响 | 缓解 |
|---|---|---|
| OpenAI AgentKit 闭源壁垒 | 集成难度 | 走 OpenAI 兼容 API + LangGraph 兼容层 |
| 国产卡生态成熟度 | Agent 性能受限 | 路线图里"推理引擎深度优化"和"国产化"合并排期 |
| MCP 标准还在演进 | 早期投入可能改方向 | 跟随 Anthropic/OpenAI 主线，不做激进扩展 |
| RL 训练人才稀缺 | 落地慢 | 优先做 SFT + 轨迹闭环，Agentic RL 放 P2 |

---

## 六、建议路线图（基于调研结论）

### P0（3 个月内，必须做）
1. **MCP 托管**：`/v1/mcp/servers` CRUD + 网关代理 + 调用计费
2. **轨迹采集器**：所有 `/v1/runs` 输出自动落 LanceDB，结构含 messages/tool_calls/rewards
3. **工具沙箱 v1**：基于 gVisor 的代码执行环境，绑 RL rollout

### P1（6 个月内，应该做）
1. **Threads/Runs API**：兼容 OpenAI 标准，让生态工具可迁移
2. **Agent 评测 harness**：τ-bench / SWE-bench 子集 + 自定义业务 eval
3. **Data Lab → RL pipeline**：标注完成的轨迹一键转 SFT 数据格式

### P2（12 个月内，可以做）
1. **Computer/Browser Use**：内置 Browserbase 自托管版本，作为 rollout env
2. **国产 Agent 一体机**：与 HAMi 国产卡适配合并交付
3. **Multi-Agent 编排**：参考 LangGraph 低代码编排

### P3（观望）
- A2A 协议兼容（Google 主导，标准未定）
- 去中心化 RL 训练（Prime Intellect 路线，商业模式未验证）

---

## 七、附录：参考来源

- OpenAI DevDay 2025 Keynote（AgentKit 正式 GA）
- Anthropic MCP 协议规范 v1.0
- Google Cloud Next 2025（Agent Engine + A2A）
- AWS re:Invent 2025（Bedrock AgentCore）
- 阿里云百炼 + AgentScope 2025 路线图
- 字节扣子 2025 商业化白皮书
- LangChain State of AI Agents 2025 报告
- Together AI 2025 年报
- Prime Intellect INTELLECT-2 技术报告

> 本报告信息截止 2026-01，建议每季度刷新一次；重点关注 MCP、Agent Harness、Agentic RL 训练框架的演进。
