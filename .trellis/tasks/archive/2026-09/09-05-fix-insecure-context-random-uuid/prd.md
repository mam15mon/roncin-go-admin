# 修复非安全上下文下 crypto.randomUUID 缺失导致的前端请求与登录报错

## Goal

在非 localhost 的 HTTP 访问场景下（非安全上下文，例如局域网 IP `http://10.180.10.50:8001`），现代浏览器禁用 `crypto.randomUUID`。通过提供全局安全的 UUID 生成与降级机制并统一前端调用，确保在各种内网或公网 HTTP 开发测试环境下均能正常请求与登录。

## Requirements

1. **核心工具方法**：
   - 在 `web/src/utils/uuid.ts` 封装并导出统一安全的 UUID v4 生成函数 `generateUUID()`。
   - 优先使用原生 `crypto.randomUUID`（安全上下文、HTTPS、localhost）。
   - 若 `crypto.randomUUID` 不存在，降级使用 `crypto.getRandomValues` 生成符合 RFC 4122 v4 标准的 UUID。
   - 若极端环境下 Web Crypto API 完全不可用，降级使用基于 `Math.random` 的兜底实现。

2. **拦截器与请求标头修复**：
   - 修改 `web/src/requestErrorConfig.ts` 中的请求拦截器，使用 `generateUUID()` 生成 `X-Request-ID`，彻底解决登录与普通 API 请求抛出 `crypto.randomUUID is not a function` 的问题。

3. **业务模块统一迁移**：
   - 将现有零散直接调用 `crypto.randomUUID` 或 `globalThis.crypto.randomUUID` 的业务组件统一替换为 `generateUUID()`：
     - `web/src/pages/orders/components/detail/OrderLockControl.tsx`
     - `web/src/pages/finance/bills/components/BillCreationWorkbench.tsx`
     - `web/src/pages/finance/cashflows/index.tsx`
     - `web/src/pages/finance/commissions/components/CommissionAdjustmentModal.tsx`
     - `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx`
     - `web/src/pages/finance/invoices/index.tsx`
     - `web/src/pages/finance/verifications/VerificationWorkbench.tsx`
     - `web/src/pages/orders/fees.tsx`
     - `web/src/pages/orders/order-fee-panel.tsx`
     - `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx`
     - `web/src/pages/settings/components/ExchangeRateImportModal.tsx`

## Acceptance Criteria

- [x] `web/src/utils/uuid.ts` 实现健壮且具备单元测试覆盖。
- [x] `web/src/requestErrorConfig.ts` 与所有 11 处业务引用均已切换至统一工具方法。
- [x] 执行 `pnpm --dir web test` 和 `pnpm --dir web lint`（及 `tsc`）无错误通过。
- [x] 在非安全上下文（如内网 IP HTTP 访问）下登录或发起请求时不再报错。
