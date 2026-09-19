# 规范财务台账顶层骨架与公司筛选器

## Goal

消除财务各页面在 `<FinanceLedgerTemplate>` 外部手写裸 `<Card>` 的反模式，在模板内原生支持顶层筛选工具栏插槽，并恢复 `PageContainer` 标题与副标题的规范展示。

## Requirements

1. **改造 `FinanceLedgerTemplate`**：
   - 增加 `extraHeader` 或 `topBar` 插槽（放置所属公司选择器、全局标签筛选等）；
   - 在 `FinanceLedgerTemplate` 内部的 `PageContainer` 中正常消费并展示 `pageTitle` 与 `pageSubTitle`（或吸顶页头规范），不再强行 `header={{ title: undefined }}` 清空。
2. **重构各财务页面顶层结构**：
   - `web/src/pages/finance/bills/index.tsx`
   - `web/src/pages/finance/cashflows/index.tsx`
   - `web/src/pages/finance/invoices/index.tsx`
   - `web/src/pages/finance/verifications/index.tsx`
   - `web/src/pages/finance/fees/index.tsx`
   - 移除外挂在 `FinanceLedgerTemplate` 外部的裸 `<Card>`，统一通过模板插槽传入。
3. **保持状态与行为零变更**：
   - 公司切换、标签联动、表格刷新逻辑原样保留。

## Acceptance Criteria

- [ ] 财务 5 个台账页面的最外层统一为 `<FinanceLedgerTemplate>` 包装的 `<PageContainer>`，无外挂裸 Card。
- [ ] 页面顶部拥有统一规范的标题与副标题，视觉上与订单管理、系统管理保持一致。
- [ ] 现有财务台账相关单测全量通过。
- [ ] `pnpm --dir web tsc` 零错误。
