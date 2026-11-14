# 阶段 E：结算账户快建完成记录

日期：2026-11-14。

## 行为与边界

- `BillGroupCard` 通过同目录 `SettlementAccountSelect` 提供账户下拉底部「新增结算账户」，复用公共 `QuickCreateModal` 与既有 `partnerServiceCreatePartnerAccount`，不新增后端接口。
- 入口同时复用 `canCreatePartnerAccounts`、`canOperateOrganization`，并读取官方 `ConfigProvider.useConfig().componentDisabled`：当前公司不匹配时不查询、不提交；父建账表单冻结时关闭快建并禁止任何在途回填。
- 结算对象、币种、用途从当前叶子只读展示与写入。应收仅写应收用途及对应默认标志，应付同理；创建账户启用，名称、户名、银行、账号必填，SWIFT 选填。
- 账户候选由 React Query 管理，查询键包含工作台会话、公司、组键、结算对象、方向、币种和本叶子刷新序号。成功只更新本叶子账户并重新查询该叶子，保留账期、备注、预计开票配置以及其他叶子账户。
- 创建响应为空或业务拒绝时保留输入，不回填；成功响应缺业务 ID 明确报错。关闭重开清理旧输入；完整身份变化重建叶子，`useAsyncGuard` 拒绝旧创建结果，Query 请求取消拒绝旧候选回填。
- 与阶段 D 的结果待确认冻结衔接，不改 Workbench/结果表的批次提交语义。

## 改动文件

- `web/src/features/finance/bill-creation/BillGroupCard.tsx`：移出原手写账户候选请求链，消费新组件。
- `web/src/features/finance/bill-creation/SettlementAccountSelect.tsx`：叶子账户查询与快建。
- `web/src/features/finance/bill-creation/SettlementAccountSelect.test.tsx`：9 个账户行为用例。
- `web/src/features/finance/bill-creation/BillGroupCard.test.tsx`：原 4 个展示用例改用应用上下文包装。
- `.trellis/spec/web/frontend/capability-navigation.md`：补充建账能力中的账户快建契约。

## 验证

- `pnpm --dir web exec vitest run src/features/finance/bill-creation/SettlementAccountSelect.test.tsx src/features/finance/bill-creation/BillGroupCard.test.tsx`：2 文件、13 用例通过，文件用例均未超过 20 秒。
- `pnpm --dir web tsc`：通过。首次检查与阶段 D 客户端生成器清理输出目录相撞，待生成完成后重跑通过，不采信首次缺生成物的错误结果。
- 修改文件 `pnpm --dir web exec biome check`：通过。
- `antd lint` 修改产品文件的弃用检查：无问题。
- `pnpm run check:architecture`：18 个检查用例通过，产品依赖扫描无违规。
- `git diff --check`：通过。

未做真实浏览器视觉验收；本阶段没有新增 Schema、权限键或 API 契约。批量工作台旧账户请求断言增加 AbortSignal 第二参数的适配由阶段 D 代理负责。
