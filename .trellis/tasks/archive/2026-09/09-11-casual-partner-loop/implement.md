# 散客闭环：单次合作往来单位实施计划 (implement.md)

> 启动前置：工作区在途的伙伴 / 财务 UI 改动落地后再开始，避免互踩。

## 实施阶段与步骤

### Phase 1: 数据模型与契约改造（Schema, Proto, OpenAPI 生成）
- [x] **Step 1.1**: 在 Ent Schema 中添加字段：
  - `server/internal/data/ent/schema/partner.go` 增加 `is_casual` (bool, default: false)；
- [x] **Step 1.2**: 运行 `go -C server generate` 生成 Ent 代码并创建迁移 SQL；验证本地迁移；
- [x] **Step 1.3**: 更新 Proto 契约：
  - `server/api/partner/v1/partner.proto`: Partner, CreatePartnerRequest, UpdatePartnerRequest, ListPartnersRequest 增加 `is_casual`（字段编号以当前空闲位为准）；
  - `server/api/finance/v1/settlement.proto`: FinanceSettlementPartyOption 增加 `is_casual`；BillBatchPreviewGroup 增加 `is_casual` 与 `optional int32 default_payment_terms_days`（散客场景恒为 0）；
  - `server/api/finance/v1/error_reason.proto`: 增加 `FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED` 错误码；
- [x] **Step 1.4**: 运行契约生成流水线：
  - `make -C server api`
  - `pnpm run generate:web-client`
  - 检查服务端与前端生成文件，保证无多余无关修改。

### Phase 2: 服务端业务逻辑改造与单元测试（biz & data）
- [x] **Step 2.1**: 伙伴散客标识与过滤：
  - `server/internal/biz/partner.go` 接入 `IsCasual`；批量导入显式保持 `IsCasual = false`；
  - `server/internal/data/partner.go` 存取转换与查询谓词支持；
  - 完善 `server/internal/biz/partner_test.go`。
- [x] **Step 2.2**: 散客供应商出款账户必填约束：
  - `server/internal/biz/finance_cashflow.go` 在 `Create` 中增加散客且 `Payable` 时对方收款账户非空校验；
  - `server/internal/data/finance_cashflow.go` 扩展 `ResolveParty` 返回 `is_casual`；
  - 完善 `server/internal/biz/finance_cashflow_test.go`。
- [x] **Step 2.3**: 散客客户账期底线（症状级硬编码）：
  - `server/internal/biz/finance_bill.go` 的 `PreviewBatch`：对方伙伴 `is_casual` 的应收账单组返回 `is_casual = true` 与 `default_payment_terms_days = 0`，不读结算规则；正式客户组不返回默认值；
  - 入账接口不加任何账期拦截；
  - 完善 `server/internal/biz/finance_bill_test.go`（散客默认 0、正式客户无默认值回归）。
- [x] **Step 2.4**: 运行后端定向测试验证：
  - `go -C server test ./internal/biz -run 'Partner|Cashflow|Bill'`。

### Phase 3: 前端交互改造与组件验证（web）
- [x] **Step 3.1**: 快捷建档弹窗支持散客并统一角色契约：
  - `web/src/pages/orders/components/PartnerQuickAddSelect.tsx` 增加 `[x] 单次合作` 复选框（默认 true）；
  - `web/src/pages/orders/components/fees/QuickAddPartnerModal.tsx` 增加 `[x] 单次合作` 复选框（默认 true）；客商类型由 `mode="multiple"` 多选改为单选，新增 `defaultRole` prop 并由调用方（`order-fee-panel.tsx`、`fees.tsx`）按费用方向预选（应收→客户、应付→供应商；方向未选默认客户）；
  - 更新对应组件单测（含角色单选与方向预选用例）。
- [x] **Step 3.2**: 选择器散客 Tag：
  - `web/src/utils/options.ts` 暴露 `isCasual`；
  - 订单及相关选择器配置 `optionRender` 展示 `[散客]` Tag；
  - 更新单测。
- [x] **Step 3.3**: 现金流表单散客付款必填校验：
  - `web/src/pages/finance/cashflows/index.tsx`：向散客供应商出款时对方账户动态标星并执行必填校验。
- [x] **Step 3.4**: 建账工作台散客账期底线：
  - `web/src/pages/finance/bills/components/BillGroupCard.tsx`（及相关建账卡片）：预览组携带 `is_casual` 时账期默认回填 `default_payment_terms_days`（散客恒 0）并联动到期日；散客改大账期时即时高亮黄色预警，不阻断提交；正式客户组行为与现状一致；
  - 更新对应组件单测。
- [x] **Step 3.5**: 伙伴主档与列表维护：
  - `web/src/pages/partners/index.tsx`：合作类型筛选与散客 Tag；
  - `web/src/pages/partners/partner-detail.tsx` & `BasicInfoSection.tsx`：散客 Switch 维护。

### Phase 4: 综合门禁与质量验收
- [x] **Step 4.1**: 运行受影响前端文件的 Vitest 定向测试与 Biome 检查；
- [x] **Step 4.2**: 运行 `pnpm --dir web tsc` 验证前端无类型报错；
- [x] **Step 4.3**: 运行后端相关包测试，确保无回归；
- [x] **Step 4.4**: 审查变更差异 `git diff --check`，遵循 Conventional Commits 提交。

## 滚回点

- Phase 1 完成后（纯 Schema + 契约 + 生成物）为一个可独立审阅的提交组；
- Phase 2 + 3 完成后为功能完整提交组；任一阶段失败可 `git revert` 对应提交组，无数据回滚需求（新字段带默认值，向后平滑）。
