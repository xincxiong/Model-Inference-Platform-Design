# 开发路线图（2026-Q1 ~ Q2）

**更新时间**：2026-09-06
**总跨度**：~5 个月（13-17 周）
**策略**：聚焦"训推一体化 + 国产 + 私有"差异化

---

## 紧急收尾（本周 · 1 天）

| ID | 项目 | 工作量 | 价值 |
|---|---|---:|---|
| E1 | 清理 `compute_pools.used_gpu` stale 列（已改用 live aggregate 计算） | 2h | 防止再被误读 |
| E2 | 修 `playground` / `deployments` 的 pre-existing TS 错（`streamChat` / `JSX.Element`） | 半天 | 让 prod build 真正可过 |

---

## Q1 战略窗口（4-6 周）

| ID | 项目 | 工作量 | 优先级 | 战略价值 |
|---|---|---:|:---:|---|
| Q1-A | **MCP 托管 P0**：MCP server 元数据表 + 一键部署 + 网关代理 + 调用计费 | 2.5 周 | 🥇🔴 | 行业 2026 H1 标配，错过即落后 |
| Q1-B | **轨迹→训练闭环**：finetuning runs 自动落 Data Lab，标注→SFT/RL 一键转换 | 2-3 周 | 🥈⭐ | **独有护城河** — 训推一体化的最后一公里 |
| Q1-C | **endpoints 接入算力池**：专属端点改为从池实例化，统一心智 | 1 周 | 🥉🟠 | 产品一致性收尾 |

**不并行做这 3 个**。MCP 行业必做先；闭环做护城河；endpoints 收尾。

---

## Q2 技术深耕（6-8 周）

| ID | 项目 | 工作量 | 价值 |
|---|---|---:|---|
| Q2-A | **Agent 评测 harness**：τ-bench / SWE-bench Verified 子集 + 自定义业务 eval | 2 周 | 公信力 + 自我体检工具 |
| Q2-B | **国产 GPU 一体机**：HAMi 对接昇腾 / 天数智芯 + 政企信创打包 | 3-4 周 | 信创采购窗口期（12 个月内） |
| Q2-C | **Volcano 真集群集成 + KEDA HPA 联通**：从 simulation 切真调度 | 2-3 周 | 上线前必须 |
| Q2-D | 拆分大页面（endpoints 56KB / datalab / deployments / models） | 1 周/页 | 可维护性 |

**Q2-D 可以穿插到任何时候**，不需要专门排期（每页 1 周独立任务）。

---

## 决策点

- **目标用户**？（C 端 / B 端 / 政企）影响 Q3-Q4 优先级
- **是否开源核心**？MCP / 闭环 / 一体机都可以先闭源做商业护城河
- **团队规模**？决定 Q1 是聚焦 1 项还是并行 2 项

---

## 三条战略路径（待定）

### 路径 A — "agent 时代的算力订阅平台" ⭐ 推荐
```
Q1-A (MCP) → Q1-B (闭环) → Q2-B (国产) → Q3 合规
```
走训推一体化 + 信创 + 私有，跟同行不撞。

### 路径 B — "最优 AI Gateway"
```
Q1-C (endpoints) → 限流升级 → OIDC → Computer Use → A2A
```
跟随 OpenAI/LiteLLM，稳定但同质化。

### 路径 C — "Agent 训练基础设施"（学术向）
```
Q2-C (Volcano) → 多模态 RL → Agent Arena leaderboard
```
类似 Prime Intellect，长期布局。

---

## 进度追踪

- [ ] E1 清 stale 列
- [ ] E2 修 TS 错
- [ ] Q1-A MCP 托管 P0
- [ ] Q1-B 轨迹→训练闭环
- [ ] Q1-C endpoints 接入算力池
- [ ] Q2-A Agent 评测 harness
- [ ] Q2-B 国产 GPU 一体机
- [ ] Q2-C Volcano 真集群集成
- [ ] Q2-D 大页面拆分（持续）
