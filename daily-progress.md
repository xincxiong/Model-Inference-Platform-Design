## 每日开发进展

- 记录口径：每日收尾时按日期追加，简述当日可交付变更，方便事后追溯。

### 2026-04-30（下午/晚间）
- **前端新增功能后端支持验证**
  - 验证范围：可观测性、数据实验室、成本分析、模型对比四大模块
  - 生成详细验证报告：`docs/前端新增功能-后端支持验证报告.md`
  
- **后端支持现状总览**
  - 可观测性：30% 支持（Prometheus 端点已配置，缺少推理指标收集、日志查询、GPU 监控）
  - 数据实验室：70% 支持（Datasets API 完整，缺少推理日志存储、Lance 格式支持）
  - 成本分析：40% 支持（基础用量统计 API 已就绪，缺少预算管理、费用预测、优化建议）
  - 模型对比：10% 支持（几乎全部缺失，缺少基准数据、对比 API）
  - 平均支持度：37.5%

- **已支持功能清单 ✅**
  - Prometheus 指标导出：`GET /metrics`（数据面/管理面均已配置）
  - 基础用量统计：`GET /api/usage`（余额、总花费、每日分解、按模型分解）
  - Datasets API：完整 CRUD + 导出 + SQL 查询（`/v1/datasets`）
  - 健康检查：`GET /health/*`（Live/Ready/熔断器状态）
  - Promo Code：`POST /api/billing/redeem`（推广码兑换）

- **待开发功能清单 ❌（按优先级排序）**

  **P0 - 立即开发（预计 10.5 天）**
  1. 推理指标收集中间件（2 天）
     - 在推理请求时自动收集 TTFT、TPS、错误率、Token 数
     - 创建 `internal/middleware/metrics.go`
     - 集成到 ChatCompletionsHandler 和 CompletionsHandler
  
  2. 推理日志存储（1.5 天）
     - 设计 `inference_logs` 表（user_id/model/timestamp/input_tokens/output_tokens/ttft_ms/tps/total_latency_ms/status_code/input_text/output_text/metadata）
     - 创建数据库迁移脚本
     - 实现日志写入逻辑
  
  3. 可观测性查询 API（2 天）
     - 支持时间范围、模型过滤、分页查询指标
     - API: `GET /api/observability/logs?model=xxx&start_time=xxx&end_time=xxx&page=1&limit=50`
  
  4. 推理日志查询 API（1.5 天）
     - 支持按模型/时间/项目/API Key 过滤 + 分页
     - API: `GET /api/datalab/logs`
  
  5. 预算 CRUD API（2 天）
     - 设计 `budgets` 表（user_id/name/limit_amount/spent_amount/period/alert_thresholds）
     - 实现创建/查询/更新/删除
     - API: `GET/POST/PATCH/DELETE /api/budgets`
  
  6. 模型基准数据表（1 天）
     - 设计 `model_benchmarks` 表（model_name/provider/mmlu/humaneval/gsm8k/math/ceval/avg_ttft_ms/avg_tps/context_window/input_price_per_m/output_price_per_m）
     - 录入主流模型评测数据（DeepSeek-V3/Qwen3.5-72B/GLM-5/Llama-4-70B/Kimi-K2.5/豆包 2.0）
  
  7. 模型对比 API（1.5 天）
     - 获取多模型对比数据（基准/性能/成本）
     - API: `GET /api/compare?models=ds,qw,ll`

  **P1 - 本周内完成（预计 8 天）**
  8. GPU 节点状态查询（2 天）
     - 集成 K8s API 查询 GPU 节点利用率和显存
     - API: `GET /api/observability/gpu-nodes`
  
  9. 端点副本状态查询（1 天）
     - 查询端点运行状态和副本数
     - API: `GET /api/observability/endpoints`
  
  10. Lance/LanceDB 集成（3 天）
      - 使用 Lance 格式存储推理日志
      - 支持向量混合查询
      - 实现数据集版本管理
  
  11. 费用预测服务（2 天）
      - 基于历史数据的简单预测算法
      - 支持基准/乐观/悲观三种场景
      - API: `GET /api/costs/forecast?months=6`
  
  12. 第三方评测数据集成（2 天）
      - 自动同步 OpenCompass/HELM 等评测数据
      - 定期更新模型基准数据

  **P2 - 下周完成（预计 7 天）**
  13. Prometheus 自定义指标（1 天）
      - 暴露 TTFT/TPS/GPU 利用率等自定义指标
  
  14. 状态码分布统计（0.5 天）
      - 按 HTTP 状态码分组统计请求数
  
  15. 数据标注功能（2 天）
      - 支持对推理日志进行标注和筛选
  
  16. 预算阈值告警（2 天）
      - 定时检查预算使用率
      - 触发邮件/Webhook 通知
  
  17. 成本优化建议（2 天）
      - 模型替换建议算法
      - 语义缓存收益估算
      - 批量任务迁移建议
  
  18. 专属端点成本分析（1.5 天）
      - 分析高频率调用是否适合专属端点

- **数据库设计建议**
  ```sql
  -- 推理日志表
  CREATE TABLE inference_logs (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL,
      model VARCHAR(255) NOT NULL,
      endpoint_id UUID,
      timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
      input_tokens INT NOT NULL,
      output_tokens INT NOT NULL,
      ttft_ms FLOAT,
      tps FLOAT,
      total_latency_ms FLOAT,
      status_code INT NOT NULL,
      input_text TEXT,
      output_text TEXT,
      metadata JSONB
  );
  
  -- 预算表
  CREATE TABLE budgets (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL,
      name VARCHAR(255) NOT NULL,
      limit_amount FLOAT NOT NULL,
      spent_amount FLOAT DEFAULT 0,
      period VARCHAR(50) NOT NULL,
      alert_thresholds INT[] DEFAULT '{50,80,100}',
      created_at TIMESTAMP DEFAULT NOW(),
      updated_at TIMESTAMP DEFAULT NOW()
  );
  
  -- 模型基准评测表
  CREATE TABLE model_benchmarks (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      model_name VARCHAR(255) NOT NULL UNIQUE,
      provider VARCHAR(255),
      mmlu FLOAT,
      humaneval FLOAT,
      gsm8k FLOAT,
      math FLOAT,
      ceval FLOAT,
      avg_ttft_ms FLOAT,
      avg_tps FLOAT,
      context_window VARCHAR(50),
      input_price_per_m FLOAT,
      output_price_per_m FLOAT,
      updated_at TIMESTAMP DEFAULT NOW()
  );
  ```

- **下一步行动**
  1. 立即启动 P0 功能开发（推理日志存储 + 指标收集 + 预算 API）
  2. 前端改造：后端 API 就绪后，将 Mock 数据替换为真实 API 调用
  3. 更新产品文档至 v2.6（已完成）
  4. 详细验证报告见：`docs/前端新增功能-后端支持验证报告.md`

- **Phase 3 UI 审查（03-UI-REVIEW.md）**
  - 审查范围：`/observability`、`/datalab`、`/costs`、`/compare` 四个新增页面
  - 总体评分：**15/24**（62.5 分）
  
  **六大维度评分**：
  | 维度 | 得分 | 核心问题 |
  |------|------|---------|
  | 文案 | 3/4 | 中文标签一致，但部分内联样式缺少 aria-labels |
  | 视觉 | 3/4 | 卡片/表格层级清晰，但移动端侧边栏独占、内容区空白 |
  | 颜色 | 3/4 | CSS 变量使用一致，但数据模型有 12 个硬编码颜色 |
  | 排版 | 2/4 | 使用 `text-[10px]` 和 `text-[11px]` 任意值，超出设计系统规范 |
  | 间距 | 2/4 | Tailwind 比例一致，但图表 SVG 硬编码 `viewBox="0 0 800"` 限制响应式 |
  | 体验设计 | 2/4 | **无加载/错误/空状态** — 所有页面都假设有数据 |

  **🔴 三大优先级修复**：
  1. **缺少加载/错误/空状态**（P0）
     - 添加加载骨架屏
     - 添加空状态插图和引导文本
     - 添加错误边界和重试按钮
  
  2. **图表硬编码 800px viewBox，移动端崩溃**（P0）
     - 动态 viewBox 或百分比宽度 + `preserveAspectRatio`
     - 修复 `min-w-[500px]` 导致的横向滚动
  
  3. **任意字体大小违反排版规范**（P1）
     - `text-[10px]` 和 `text-[11px]` 映射到设计令牌
     - 标准化为 `text-xs` 或扩展 Tailwind 配置

  **其他关键问题**：
  - 删除操作无确认对话框（`datalab/page.tsx`）
  - 编辑按钮无功能（`costs/page.tsx`）
  - 图表无 tooltip/hover 交互
  - 移动端响应式布局问题（侧边栏固定 240px）
  
  **详细审查报告**：`03-UI-REVIEW.md`

- **明日开发评估计划（2026-05-01）**
  1. 评估 UI 审查问题的修复工作量
  2. 确定 P0 后端功能开发优先级
  3. 制定前后端联调计划
  4. 评估 Lance/LanceDB 集成方案

### 2026-04-30（上午）
- **竞品调研：火山方舟（Volcengine Ark）**
  - 全面调研火山方舟平台产品，覆盖产品概述、核心功能、架构设计、产品优势等
  - 核心功能模块：模型推理服务（在线推理/批量推理/模型单元）、模型精调与定制（SFT/DPO/RL）、多模态能力（文本/图片/视频/3D）、智能体与工具调用（Function Calling/MCP/知识库）、高级特性（Context API/流式输出/结构化输出/Prompt 工程）、RAG 解决方案
  - 产品架构：五层逻辑架构（接入层→调度与优化层→推理与计算层→精调与数据层→安全与治理层）
  - 产品优势：字节内部实践沉淀（豆包/抖音/飞书亿级用户场景验证）、全生命周期覆盖、自研推理加速引擎（vLLM 深度优化/KV Cache 优化/动态批处理）、无缝集成火山引擎生态
  - 生成详细调研报告：`docs/火山方舟平台产品调研报告.md`
- **竞品调研：腾讯云大模型服务平台 TokenHub**
  - 全面调研腾讯云 TokenHub 平台，覆盖产品概述、核心功能、模型列表、计费模式等
  - 核心功能模块：模型广场、体验中心、AI 创作、在线推理、模型监控（TTFT/TPOT/RPM）、用量统计、API Key 管理、Token Plan、Coding Plan
  - 支持的模型：腾讯混元（Hy3/HY 2.0/Hunyuan-role）、DeepSeek（V4/v3.2/v3.1/r1/v3）、智谱 GLM（5.1/5V-Turbo/5）、Kimi（K2.6/K2.5）、MiniMax（M2.7/M2.5）、优图视频生成、混元图像/3D 生成
  - 核心能力对比：深度思考、联网搜索、结构化输出、Function Calling、Cache 缓存
  - 计费模式：按量计费（Token/张/秒）+ 订阅套餐（Token Plan/Coding Plan）+ 新人免费体验包
  - 生成详细调研报告：`docs/腾讯云大模型服务平台TokenHub产品调研报告.md`
- **MRD 文档更新**
  - 更新 `docs/MRD-市场需求文档.md` 竞品分析章节（3.2 国产竞品）
  - 新增火山方舟竞品分析：定位、优势（字节实践/全生命周期/Context API/智能路由/MCP 协议/自研加速引擎）、劣势（绑定火山引擎生态/数据需上传云端）、我们的差异化（厂商中立 + 私有化部署 + Lance 数据版本管理 + 混合架构支持）
  - 新增腾讯云 TokenHub 竞品分析：定位、优势（模型聚合丰富/OpenAI 协议兼容/TTFT/TPOT 监控/API Key 精细化权限/体验中心）、劣势（无精调能力/无智能体生态/无私有化部署）、我们的差异化（全链路闭环 + 智能体构建 + 私有化部署 + 国产化深度适配）

### 2026-04-29（下午/晚间）
- **S3 对象存储集成**
  - 新增 `config.S3` 配置结构体（S3_ENABLED/ENDPOINT/BUCKET/REGION/ACCESS_KEY/SECRET_KEY/S3_PATH_STYLE）
  - 新增 `internal/storage/s3store.go`：基于 MinIO/S3 兼容客户端，支持上传/下载/删除/健康检查/预签名 URL
  - `Store` 结构注入可选 S3 客户端接口；三端（`cmd/server` / `cmd/management-server` / `cmd/inference-server`）统一初始化，失败自动回退 DB 存储
  - `FilesHandler` 上传/下载/删除优先走 S3，失败回退 DB，下载自动回退并获取文件名
  - 编译验证通过（`go build ./...` 零错误）
- **语义缓存（Semantic Cache）**
  - 新增 `config.SemanticCache` 配置（SEM_CACHE_ENABLED/LANCEDB_URI/SEM_CACHE_EMBED_MODEL/SEM_CACHE_THRESHOLD/SEM_CACHE_TOPK）
  - 新增 `internal/semcache/cache.go`：纯 Go 内存实现，LRU 淘汰（默认 10000 条），余弦相似度匹配
  - `Store` 注入 `*semcache.Cache`；三端统一初始化，失败记录警告并禁用
  - 集成到 `ChatCompletionsHandler` 和 `CompletionsHandler`：非流式请求先计算 prompt 向量，命中缓存直接返回（附带相似度元数据），未命中则正常推理后写入缓存
  - 新增 `internal/handler/semantic_cache.go`：嵌入函数构建器（调用 embedding 模型）+ 聊天消息拼接
  - 编译验证通过（`go build ./...` 零错误）
- **Video/Speech Handler**
  - 新增 `internal/model/models.go` 扩展：`TranscriptionRequest/Response`、`SpeechRequest`、`VideoGenerationRequest/Response`
  - 新增 `internal/handler/audio.go`：`/v1/audio/transcriptions`（语音转文本，multipart 文件上传）、`/v1/audio/speech`（文本转语音，返回音频流）
  - 新增 `internal/handler/video.go`：`/v1/videos/generations`（视频生成）
  - `Engine` 接口扩展：新增 `VideoGeneration`/`Transcription`/`Speech` 方法
  - MockEngine/vLLM/SGLang/MultiEngineRouter 全部实现对应方法（vLLM/SGLang 返回不支持错误，Mock 返回模拟数据）
  - 注册到推理路由：`/v1/videos/generations`、`/v1/audio/transcriptions`、`/v1/audio/speech`
  - 编译验证通过（`go build ./...` 零错误）
- **多集群 Volcano 调度与跨域算力管理**
  - 新增 `internal/volcano/multi_cluster.go`：`MultiClusterClient` 支持多 Kubernetes 集群注册、Volcano 客户端管理
  - 集群选择策略：按区域/可用区过滤 → 选择可用 GPU 最多的集群 → 主集群兜底
  - 资源使用跟踪：`ClusterUsage` 结构（TotalGPUs/AvailableGPUs/TotalCPU/AvailableCPU/TotalMemory/AvailableMemory/JobCount）
  - 周期性刷新：`RefreshClusterUsage` 轮询各集群 Node 资源状态，`StartUsageRefresh` 后台定时更新
  - 跨集群提交：`SubmitJobAcrossClusters` 自动选择最佳集群并提交 VolcanoJob，更新使用统计
  - `Client` 结构体扩展：新增 `clusterName`/`region`/`zone` 字段
  - 编译验证通过（`go build ./...` 零错误）
- **文档更新**
  - `README.md`：新增 Video/Speech API 端点、语义缓存环境变量、S3 配置说明、多集群调度功能记录

### 2026-04-29（上午）
- **项目整体更新（基于产品方案 v2.5）**
  - 更新 `model-inference-platform/ALIGNMENT.md`：产品方案 vs 代码实现对齐分析报告，总体对齐度 ~72%
  - 更新 `model-inference-platform/README.md`：加入 GPU 虚拟化方案、跨域算力管理设计、技术栈更新
  - 更新 `frontend/src/components/Sidebar.tsx`：版本信息更新至 v0.3.0 · GPU 虚拟化 & 跨域算力
  - 详细差距分析：Video/Speech Handler 缺失、Lance/LanceDB 未集成、S3 未集成、跨域算力管理代码未实现、NVIDIA MIG 未实现、国产算力虚拟化仅枚举定义
  - 更新优先级清单：高优先级（Video/Speech Handler、语义缓存、S3 存储、可观测性、智能路由）；中优先级（MIG、国产算力虚拟化、SSO/RBAC、多集群调度）；低优先级（PD 分离、MoE 并行、模型市场）
- **GPU 虚拟化方案产品方案（Phase 3 核心设计）**
  - 新增产品方案 3.3.1 章节：GPU 虚拟化方案（MIG/vGPU/国产算力虚拟化）
  - NVIDIA MIG：H100/H200/A100 硬件级强隔离，单卡最多切分 7 个独立实例，性能损耗 <5%
  - HAMi vGPU：软件层虚拟化，GPU 内存硬隔离 + 算力配额，volcano-vgpu-device-plugin 插件实现
  - 昇腾 NPU 虚拟化：CANN 抽象实现 AI Core 隔离 + HBM 内存配额
  - 海光 DCU 虚拟化：ROCm 兼容层实现计算单元隔离 + 显存配额
  - 寒武纪 MLU 虚拟化：Neuware 抽象实现 AI 引擎隔离 + 显存配额
  - 虚拟化方案选择策略：高性能推理用 MIG，共享集群用 vGPU，国产合规用 NPU 虚拟化，跨域混合用 HAMi 统一抽象
  - 更新基础设施与编排章节技术选型表，细化 GPU 虚拟化方案
  - 更新架构分层图，加入 GPU 虚拟化层详细结构
- **跨域算力管理产品方案（Phase 3/4 核心设计）**
  - 新增产品方案 4.3'.8 章节：跨域算力管理与统一调度
  - 全局算力资源池设计：全局算力视图/跨域部署选择/智能路由/故障转移/成本优化
  - 跨域队列调度设计：Volcano 联邦调度/全局队列资源池/跨域任务迁移/优先级抢占/负载均衡/多集群 AI 负载感知
  - 算力利用率优化：GPU 利用率 >85%/跨域负载均衡 <10% 差异/成本优化 20-40% 降低/冷启动 <30s
  - 跨域部署配置示例：JSON 格式展示多区域部署策略（regions/failover/cost_optimization）
  - 全局算力监控仪表盘：资源总览/跨域流量分布/成本分析/告警规则/容量规划
  - 更新 Phase 3/4/5 实施路线图，加入跨域算力管理相关交付物
- **调度与编排层重构（Phase 3 核心架构优化）**
  - 明确设计原则：HAMi + Volcano 联合方案为主，K8s 原生组件为辅，避免引入不必要第三方组件
  - 更新 `internal/config/config.go`：新增 `VolcanoConfig` 结构体（Enabled/Namespace/Queue/JobImage/SchedulerPolicy/PriorityClass/TTLSecondsAfterFinish/MinAvailableOverride）
  - 更新 `internal/volcano/client.go`：新增 `JobOptions` 结构体，支持 MinAvailable/TTL/PriorityClass/SchedulerPolicy 可配置
  - 更新 `internal/handler/fine_tuning.go`：接收 Volcano 配置，落盘调度元数据到 rollout_config，提交 VolcanoJob 时应用配置化参数
  - 更新 `cmd/server/main.go` 与 `cmd/management-server/main.go`：使用配置化 Volcano 客户端替代环境变量直读
  - 更新 `internal/router/router.go`：传递 Volcano 配置到 FineTuningHandler
  - 编译验证通过（`go build ./...` 零错误）
- **产品文档刷新**
  - 更新 `模型推理云平台-产品方案.md` 至 v2.5（2026-04-29）
  - 微调训练数据流 (5.4) 完整重写，加入 HAMi+Volcano 联合调度全流程
  - 基础设施与编排 (6.5) 详细描述联合调度架构，移除不必要组件（KEDA/Istio/Terraform 等）
  - 新增 4.4.3 Agentic RL 平台原生支持（Megatron-LM + SGLang + Data Buffer + Volcano/HAMi）
  - Phase 4 新增多集群 Volcano 调度与 Agentic RL 平台增强
  - Phase 5 新增算力利用率优化、智能成本路由、多集群调度与百万级并发架构
  - 持续演进方向增补 Volcano 多集群调度、GPU 碎片整理、算力利用率预测
  - 演进优先级矩阵调整：成本优化与调度智能化从 🟢 低 → 🔴 高优先级
- **项目文档同步更新**
  - 更新 `model-inference-platform/README.md`：加入 Volcano 调度与编排层、Phase 3 完整交付记录、技术栈更新
  - 更新 `README.md`：技术选型加入 HAMi + Volcano + K8s 原生组件，系统架构描述更新
  - 创建 `model-inference-platform/ALIGNMENT.md`：产品方案 vs 代码实现 对齐分析报告（总体对齐度 ~75%）
- **架构收益**：
  - 调度简化：HAMi 负责硬件抽象，Volcano 负责队列/优先级/Gang 调度，K8s 原生处理常规工作负载
  - 组件精简：移除 KEDA/Istio/Terraform/Harbor 等不必要组件，降低运维复杂度
  - 算力利用率：目标 GPU 利用率 >85%，支持碎片整理与紧凑调度

### 2026-04-28
- **控制面/数据面分离（Phase 3 核心架构重构）**
  - 新增 `backend/cmd/inference-server/main.go`：推理数据面服务，处理 `/v1/chat/completions`、`/v1/responses`、`/v1/embeddings`、`/v1/rerank`、`/v1/images/generations` 等推理端点，默认端口 8080
  - 新增 `backend/cmd/management-server/main.go`：控制面服务，处理 `/v0/dedicated_endpoints`、`/v1/fine_tuning`、`/v1/batches`、`/v1/datasets`、`/v1/deployments`、`/api/*` 等管理端点，默认端口 8081
  - 重构 `backend/internal/router/router.go`：拆分为 `SetupInferenceRouter()` 和 `SetupManagementRouter()` 两个独立函数，职责清晰分离
  - 更新 `backend/internal/config/config.go`：新增 `ManagementPort` 配置项（默认 8081）
  - 更新 `backend/cmd/server/main.go`：单体模式同时启动两个服务（推理 + 管理），便于本地开发和测试
  - 更新 `docker-compose.yml`：添加 `management-server` 服务，独立端口 8081
  - 更新 `backend/Dockerfile`：支持构建两个二进制文件（inference-server + management-server）
  - 编译验证通过（`go build ./...` 零错误）
- **Volcano 微调调度（Phase 3）**
  - 新增 `internal/volcano/` 客户端，读取 `VOLCANO_ENABLED`/`VOLCANO_NAMESPACE`/`VOLCANO_QUEUE`/`VOLCANO_JOB_IMAGE`，禁用时模拟运行
  - `fine_tuning` Handler 使用 Volcano JobPhase 常量轮询，支持可配置队列/命名空间，并在失败/取消时回写状态
  - 新增 `deploy/volcano/`（install.sh / values.yaml / volcano-job-examples.yaml）与 README 部署说明，支持 Helm 安装 + Gang 队列示例
- **架构收益**：
  - 故障隔离：管理接口 OOM/panic 不影响推理服务
  - 独立扩缩容：可单独设置推理/管理服务的副本数
  - 资源优化：独立连接池配置，避免管理操作耗尽推理资源
  - 安全边界：可限制管理接口访问，降低暴露面
- **产品文档更新**：
  - 更新 `模型推理云平台-产品方案.md` 至 v2.4（2026-04-28）
  - 新增控制面/数据面分离、前端增强、KEDA、Prometheus 等 Phase 3 交付物
  - 修正前端技术栈描述（移除未使用的 shadcn/ui、TanStack Query、ECharts）
  - 更新侧边栏版本信息（Phase 2 → Phase 3）
  - 更新 README.md：添加双服务架构说明、K8s 部署清单、技术栈更新
- **项目全面分析**：
  - 后端：13 个内部包，17 个 Handler，完整的多引擎路由和 HAMi GPU 虚拟化
  - 前端：11 个页面，4 个工具库（errorHandler、optimisticUpdate、Zustand、api）
  - 部署：14 个 YAML 文件（11 个 K8s + 3 个 HAMi）
  - 架构：控制面/数据面分离、多推理引擎、HAMi GPU 虚拟化、KEDA 自动伸缩

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
