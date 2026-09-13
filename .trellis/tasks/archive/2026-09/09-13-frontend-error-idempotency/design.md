# 技术设计 (design.md)

## 1. 总体结构

两个正交能力，互不依赖，可独立实施与回滚：

```
[Sentry 报错捕获]                      [防重与幂等]
 web/config/config.ts (define 注入)     web/src/requestErrorConfig.ts (inflight 守卫)
 web/src/app.tsx (条件初始化)            server order schema/migration (幂等键)
 web/src/requestErrorConfig.ts (捕获)    server biz order Create/UpdateDraft (意图比对)
                                        web 订单提交点 (键生成)
```

## 2. Sentry 轻量接入

- **依赖**：`pnpm --dir web add @sentry/react`（唯一新增依赖，禁止顺带加其他包）。
- **注入**：`web/config/config.ts` 的 `define` 增加 `'process.env.SENTRY_DSN': process.env.SENTRY_DSN ?? ''`；`.env.example` 增加空的 `SENTRY_DSN=` 示例（真实 DSN 不入库）。
- **初始化**（`web/src/app.tsx` 模块顶层，客户端执行一次）：
  - `process.env.SENTRY_DSN` 为空 → 完全跳过 `Sentry.init`（未配置路径零网络行为）；
  - 配置时：`dsn`、`release: process.env.COMMIT_HASH`、`environment: process.env.UMI_ENV === 'prod' ? 'production' : process.env.UMI_ENV || 'development'`；不设置 `tracesSampleRate`/`replaysSessionSampleRate`；不开启 `sendDefaultPii`。
- **捕获点**：
  - 全局 JS 异常与 unhandledrejection：Sentry 默认 handler，无需额外代码；
  - 请求错误：`requestErrorConfig.ts` `errorHandler` 末尾追加 `Sentry.captureException`，带 tag（`kind: 'request'`、`http_status`）；401 跳登录（预期流）不上报；
  - 不加自定义 React ErrorBoundary（全局 handler 已覆盖 render 异常，保持轻量）。
- **脱敏**：依赖 Sentry 默认脱敏（不开启 PII、不附加请求体）；`beforeSend` 中显式剔除可能出现的 `Authorization`/`Cookie` 值作双保险。

## 3. 请求层防重守卫

- 位置：`requestErrorConfig.ts` 的 `requestInterceptors`（与 errorConfig 同文件，单一挂载点）。
- 规则：方法 ∈ {POST, PUT, DELETE} 且「method + URL + 序列化请求体」与某个 in-flight 请求完全一致时，第二个请求**不发出发起**，直接抛业务错误（message：「操作正在提交中，请勿重复提交」，复用 ErrorEnvelope 形状走既有提示链）。
- 生命周期：请求完成（成功或失败）即移除 in-flight 记录，**不做完成后的时间窗缓存**——合法的再次提交不受阻；快速双击（首次仍在途）被拦截。
- 例外：`options.skipErrorHandler` 的调用方仍受守卫（防重是正确性底线而非错误提示策略），但守卫抛错统一走既有错误信封。
- 明确不做：按用户/会话区分（登录用户串扰场景不存在同体会话并发）、请求体哈希持久化、可配置白名单。

## 4. 订单幂等（镜像建账模式）

- **Schema/迁移**：order 表加 `idempotency_key varchar(128)`；`(organization_id, idempotency_key)` 唯一索引。存量开发数据迁移中用 `gen_random_uuid()` 回填（建账字段为 NotEmpty，不允许空串）。
- **Create**（严格镜像 `finance_bill.go` Create 的幂等段）：
  - 请求 `idempotencyKey` 规范化（trim，非必填但传入即生效；缺省时由服务端生成还是保持可选 → 与建账一致按可选处理，传入才启用幂等；前端总是生成传入）；
  - 事务内 `GetByIdempotencyKey` 查既有订单：存在且意图一致（关键载体字段相同）→ 返回既有订单；存在但意图不同 → 409 冲突（复用建账的幂等冲突错误与文案口径）；
  - 幂等键计算/意图比对在费用锁定之后、写入之前，位置对齐范本。
- **UpdateDraft**：`idempotency_key` 为**可变**列（每次成功更新写入本次键）。语义：同键 + 同 expectedVersion 的重放 → 返回当前草稿（无副作用）；键不同或版本不匹配 → 走既有乐观锁 409。不在草稿上保留历史键（只存最新）。
- **前端**：订单创建 Modal 与草稿编辑路径在「每次提交意图」时经 `web/src/utils/uuid` 的 `generateUUID` 生成键并随请求上报；提交成功或 Modal 重置后重新生成；守卫拦截的重试沿用同键。
- **错误码**：复用建账幂等冲突的既有错误码/文案口径，不新增 proto 错误码；order proto 的 Create/UpdateDraft 请求体各加 `idempotency_key` 字段（openapi/生成客户端同步重生成）。

## 5. 兼容与回滚

- Sentry：删掉 app.tsx 初始化与依赖即完全回滚；未配置 DSN 的部署升级后行为不变。
- 防重守卫：单文件逻辑，回滚即删；对既有页面无接口变化。
- 订单幂等：新增列为可选键（前端总是传，API 直调可不传，行为与现状一致）；迁移回填不影响存量单据。

## 6. 已定取舍

- 守卫放请求层而非逐页 hook：一处生效全覆盖、无页面遗漏面；代价是无法按业务语义区分「故意重复提交」（评估为可接受，写操作同体连发几乎必然是误操作）。
- UpdateDraft 采用「可变最新键」而非全历史比对：草稿重放的失败模式收敛为「再走一次乐观锁 409 后刷新」，不引入键历史存储。
- 不做 sourcemap 上传：轻量优先，堆栈按压缩名呈报（后续需要再立项）。
