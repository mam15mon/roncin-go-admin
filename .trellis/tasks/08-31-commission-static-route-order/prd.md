# 修复提成静态路由遮蔽

## Goal

修复提成员工和候选人静态 GET 路由被 `/commissions/{id}` 动态路由抢先匹配的
问题，并增加真实 HTTP Router 分发回归测试，防止后续调整 Proto RPC 顺序时复发。

## Requirements

- 调整 `SettlementService` 中提成查询 RPC 的声明顺序，使以下静态 GET 路由均在
  `/api/v1/finance/commissions/{id}` 之前注册：
  - `/api/v1/finance/commissions/export`
  - `/api/v1/finance/commissions/employees`
  - `/api/v1/finance/commissions/candidates`
- 只修改 `.proto` 真相源并通过生成命令更新 HTTP/gRPC/描述符等生成物，禁止手工
  修改 `*_http.pb.go`。
- 在 `internal/server` 增加真实 Kratos HTTP Router 分发回归测试，分别请求三条
  静态路径并确认实际命中的 operation/处理器不是 `GetCommission`。
- 修复后补测试，不采用 TDD；测试应在未来有人把动态路由重新移到静态路由之前时
  稳定失败。
- 保持 `ExportCommissions` 的路径、权限和业务行为不变，不修改任何提成筛选、
  金额、审计或前端功能。

## Acceptance Criteria

- [ ] `export`、`employees`、`candidates` 三条静态 GET 路由均正确分发，不会把静态
      片段解析为 `{id}`。
- [ ] `/api/v1/finance/commissions/{id}` 的正常详情路由仍可分发到 `GetCommission`。
- [ ] 生成代码由 Proto 重建且重跑生成命令无新增差异。
- [ ] 新增 Router 回归测试、`go -C server test ./internal/server/...`、
      `go -C server test ./...` 与 `go -C server vet ./...` 全部通过。

## Notes

- 本任务是阶段 2 复核发现的既有 P2，与提成导出实现无关，必须独立提交并独立
  归档，不夹带到阶段 3。
- 这是边界明确的轻量缺陷任务，采用 PRD-only 规划。
