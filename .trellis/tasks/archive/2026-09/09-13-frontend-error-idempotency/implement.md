# 实施清单 (implement.md)

> 按阶段推进；Sentry（Phase 1-2）与防重+幂等（Phase 3-5）相互独立，可分别验证。

## Phase 1: Sentry 依赖与环境注入
- [ ] **Step 1.1**: `pnpm --dir web add @sentry/react`（唯一新增依赖）；
- [ ] **Step 1.2**: `web/config/config.ts` define 增加 `process.env.SENTRY_DSN`；`.env.example` 增加空示例；
- [ ] **Step 1.3**: `web/src/app.tsx` 条件初始化（DSN 空则跳过；release=COMMIT_HASH；无 tracing/Replay/PII）。

## Phase 2: 请求错误捕获
- [ ] **Step 2.1**: `web/src/requestErrorConfig.ts` errorHandler 追加 captureException（401 跳登录不上报；tag: kind=http_status）；
- [ ] **Step 2.2**: `beforeSend` 双保险剔除 Authorization/Cookie；
- [ ] **Step 2.3**: 定向测试：DSN 未配置时不初始化（可 mock 断言）、配置后错误被捕获（Sentry 官方测试模式或注入测试 transport）。

## Phase 3: 请求层防重守卫
- [ ] **Step 3.1**: `requestErrorConfig.ts` requestInterceptors 实现 POST/PUT/DELETE 同 method+URL+体 inflight 去重（拒绝并走既有中文提示链）；
- [ ] **Step 3.2**: 定向测试：inflight 双发只到后端一次、完成后立即可重发、GET 不受影响、skipErrorHandler 仍受守卫。

## Phase 4: 订单幂等
- [ ] **Step 4.1**: order Ent schema 加 `idempotency_key`（Create 用 NotEmpty(128) Immutable + (organization_id, idempotency_key) 唯一索引；UpdateDraft 用可变最新键——具体形态按 design.md 第 4 节，若两操作共用一列需在 schema 层面分列或确认可变语义后固定）；`go -C server generate`；迁移含存量回填（gen_random_uuid）；
- [ ] **Step 4.2**: proto：CreateOrderRequest/UpdateOrderDraftRequest 加 `idempotency_key`；`make -C server api` + `pnpm run generate:web-client`；
- [ ] **Step 4.3**: biz：Create 镜像 finance_bill 幂等段（GetByIdempotencyKey + 意图比对 + 409）；UpdateDraft 同键同版本重放返回当前草稿；
- [ ] **Step 4.4**: service 透传；前端订单创建/草稿编辑提交点接入 generateUUID 键（成功/重置后重新生成）；
- [ ] **Step 4.5**: 测试：biz 单测（同意图重放/冲突/无键行为不变）+ 真实库集成（并发/重放）+ 前端提交点测试。

## Phase 5: 收尾验证
- [ ] **Step 5.1**: `go -C server build ./... && go -C server vet ./... && go -C server test ./internal/...`；真实库集成（注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`，取 .env.local）；
- [ ] **Step 5.2**: `pnpm --dir web tsc`；biome 定向检查改动文件；定向 vitest（requestErrorConfig、订单提交点、RoleFormModal 无关不跑）；
- [ ] **Step 5.3**: 生成物幂等（重跑 make api / generate:web-client 无 diff）；`gofmt -l`；`git diff --check`；
- [ ] **Step 5.4**: 手工冒烟清单记录：未配 DSN 启动无 Sentry 网络请求；双击提交按钮后端仅一次调用。

## 风险与回滚点
- 每阶段独立可提交；Sentry 回滚 = 删初始化与依赖；守卫回滚 = 删单文件逻辑；订单幂等回滚 = 前端停止传键（后端可选键，行为回到现状）。
