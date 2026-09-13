# 前端报错捕获与关键操作防重防护 PRD

## Goal

1. 前端接入轻量 Sentry 报错捕获：DSN 环境变量门控、未配置时零开销降级、不引性能追踪/回放等重特性。
2. 关键业务写操作建立统一防重复提交防护；服务端幂等按资损风险补齐缺口（订单创建/草稿更新）。

## 背景与已确认事实

- 请求层：Umi Max 统一错误处理已存在（`web/src/requestErrorConfig.ts` 的 errorThrower/errorHandler，业务错误信封含 code/message/reason/traceId），生成客户端全部走 umi request——是请求错误捕获与防重守卫的单一挂载点。
- 框架：`@umijs/max` + React 19 + antd 6；无任何现役全局错误捕获（无 @sentry 依赖）；`web/config/config.ts` 用 `define` 注入 `process.env.*`（已有 COMMIT_HASH 先例）。
- 服务端幂等现状：finance/settlement（建账/核销/对冲）、exchange_rate、partner、sea_document、background_task 已有 idempotency；前端现金流/发票/核销/佣金页面已发送幂等键；建账幂等范本在 `server/internal/biz/finance_bill.go`（`GetByIdempotencyKey` + 幂等意图比对，`finance_bill.go:314/1125/1156`；schema `finance_bill.go:23` NotEmpty(128)+组织内唯一索引 `:82`）。
- 幂等缺口：order.proto 无幂等；钉钉注册确认有一次性 Token + 状态机天然防重；管理类配置操作低频可逆。
- 防重现状：各页面手工 loading/confirmLoading，无全局提交守卫；项目不使用 useRequest/useMutation。

## 用户决策（2026-09-13 定稿）

- **幂等范围**：仅订单创建/草稿更新补服务端幂等（对齐建账既有模式）；钉钉确认与管理类操作不补；财务域既有幂等直接复用。
- **防重形态**：请求层统一 inflight 防重守卫作为全局机制 + 页面既有 loading 状态保留。

## Requirements

- **R1 Sentry 轻量接入**：新增 @sentry/react（唯一新增依赖）；`SENTRY_DSN` 经 config.ts `define` 注入，未配置时初始化整体跳过（行为与现状一致）；捕获范围 = 全局 JS 异常 + unhandledrejection + errorHandler 中的请求错误（401 跳登录的预期流不上报，其余带 status tag 上报）；`beforeSend` 保证事件不含令牌/Cookie/请求体；release 复用 COMMIT_HASH；不做 tracing/Replay/sourcemap 上传；`sendDefaultPii` 保持关闭。
- **R2 请求层防重守卫**：对 POST/PUT/DELETE 的「同 method+URL+请求体」inflight 请求，第二次触发直接拒绝并提示「操作正在提交中，请勿重复提交」，不产生第二次后端调用；GET 与 `skipErrorHandler` 路径不受影响；完成后不设时间窗缓存（合法重提不受阻）。
- **R3 订单幂等**：镜像建账模式——schema 加 `idempotency_key`（组织内唯一索引），`Create` 同键同意图重放返回原单、同键不同意图返回冲突；`UpdateDraft` 用可变幂等键：同键 + 同 expectedVersion 重放返回当前草稿，版本不匹配仍走既有乐观锁 409；前端订单创建/更新在每次提交意图时生成 UUID 键（复用 `web/src/utils/uuid` 的 generateUUID），成功后重置。

## Acceptance Criteria

- [ ] 未配置 SENTRY_DSN：前端网络行为与现状完全一致（无 Sentry 请求、无 console 噪音）；配置后 JS 异常、Promise 拒绝、请求错误均上报，release=COMMIT_HASH，事件不含令牌/Cookie/请求体/完整敏感报文。
- [ ] 任一写操作 inflight 期间重复提交（同 method+URL+体）被守卫拦截并给出中文提示，后端只收到一次调用；GET 查询不受影响。
- [ ] 订单创建：同幂等键重放返回同一订单；同键不同内容返回 409；草稿更新：同键同版本重放返回当前草稿，不产生重复变更。
- [ ] 新增依赖仅 @sentry/react（pnpm）；全量门禁（web 647+ / server 全量 / govulncheck）绿；生成物幂等；gofmt 干净。

## Out of Scope

- Sentry 性能追踪（tracing）、Session Replay、sourcemap 上传、告警规则配置；自定义 React ErrorBoundary 组件（全局 handler 已覆盖）。
- 钉钉确认、管理类配置操作的服务端幂等。
- 后端 OTEL 遥测改动。
