## 每日开发进展

- 记录口径：每日收尾时按日期追加，简述当日可交付变更，方便事后追溯。

### 2026-04-17
- **后端**: 更新 `cmd/server/main.go`、`internal/router/router.go`、`internal/store/postgres.go`、`internal/handler/billing.go`，完善模型/账单相关的路由与存储逻辑，支撑模型版本与用量聚合能力。
- **前端**: 重构 `deployments`、`endpoints`、`finetuning`、`usage` 页面，增加部署/端点列表与表单细节、用量趋势与成本分布可视化，整体 UI/交互更完整。
- **文档**: 同步产品方案与 HTML 设计稿至 2026-04-17 版本，校正微调、用量统计、模型版本管理等描述。
