# 应收账期与信用额度消费实施计划 (implement.md)

> 启动前置（Phase 0 检查，任一未满足不得开始）：
> 1. `09-11-casual-partner-loop` 已交付归档（`is_casual` 字段、契约、选择器散客标注与散客账期底线可用）；
> 2. 09-10 加固任务已收尾，未核销 / 对冲分摊口径修复已合入；
> 3. 工作区在途财务 / 伙伴 UI 改动已落地；
> 4. design.md 第 0 节两个待定决策已定稿。

## 实施阶段与步骤

### Phase 0: 前置核验
- [ ] **Step 0.1**: 核对上述四项前置条件；确认 `billUnsettledPredicate()` / `unverified_base_amount` 的最终口径与挂载点；
- [ ] **Step 0.2**: 定稿「默认账期回填位置」与「账单编辑场景预警」两个设计决策，回写 design.md。

### Phase 1: 数据模型与契约改造（Schema, Proto, OpenAPI 生成）
- [ ] **Step 1.1**: 在 Ent Schema 中添加字段：
  - `server/internal/data/ent/schema/partner_settlement_rule.go` 增加 `payment_terms_days` (int, optional, nillable)；
  - `server/internal/data/ent/schema/finance_custom_setting.go` 增加 `credit_limit_selection_allowed` (bool, default: true)；
- [ ] **Step 1.2**: 运行 `go -C server generate` 生成 Ent 代码并创建迁移 SQL；验证本地迁移；
- [ ] **Step 1.3**: 更新 Proto 契约：
  - `server/api/partner/v1/partner.proto`: PartnerSettlementRule, PartnerSettlementRuleInput 增加 `payment_terms_days`；
  - `server/api/finance/v1/settlement.proto`:
    - BillBatchPreviewGroup 增加信用字段 `credit_limit_amount`, `credit_currency`, `current_unsettled_amount`, `is_credit_exceeded`（`is_casual` 与 `default_payment_terms_days` 已由散客任务交付）；
    - FinanceSettlementPartyOption 增加 `credit_exceeded`；
    - 新增 CreditLimitControlPolicy 消息与 Get/UpdateCreditLimitControlPolicy RPC；
  - `server/api/finance/v1/error_reason.proto`: 增加 `PARTNER_CREDIT_LIMIT_EXCEEDED` 错误码；
- [ ] **Step 1.4**: 运行契约生成流水线：
  - `make -C server api`
  - `pnpm run generate:web-client`
  - 检查服务端与前端生成文件，保证无多余无关修改。

### Phase 2: 服务端业务逻辑改造与单元测试（biz & data）
- [ ] **Step 2.1**: 自定义设置信用额度管控策略：
  - `server/internal/biz/finance_custom_setting.go` 新增 CreditLimitControlPolicy 实体与用例；
  - `server/internal/data/finance_custom_setting.go` 持久化存取与并发乐观锁；
  - `server/internal/service/settlement_custom_setting.go` 暴露 HTTP/gRPC 服务接口；
  - 编写单测 `server/internal/biz/finance_custom_setting_test.go`。
- [ ] **Step 2.2**: 结算规则默认账期维护：
  - `server/internal/biz/partner_settlement_rule.go` 结构体和校验支持 `PaymentTermsDays`；
  - `server/internal/data/partner_settlement_rule.go` 持久化转换支持；
  - 完善 `server/internal/biz/partner_settlement_rule_test.go`。
- [ ] **Step 2.3**: 客户折本币超额判定与直接干预拦截：
  - `server/internal/data/finance_bill.go` 增加 `GetPartnerUnsettledReceivableBaseAmount`（**复用 09-10 修复后的未结清 / 核销分摊口径，不另写余额算法**）；
  - `server/internal/biz/finance_bill.go` 的 `PreviewBatch` 实现正式客户账期带出与超额比对（散客默认 0 已由散客任务交付）；
  - `server/internal/biz/order.go` 与 `order_fee.go` 在直接干预模式下校验拦截超额客户（尽力而为口径，不加串行化锁）；
  - 完善 `server/internal/biz/finance_bill_test.go`、`order_test.go`、`order_fee_test.go`。
- [ ] **Step 2.4**: 运行后端定向测试验证：
  - `go -C server test ./internal/biz -run 'SettlementRule|CreditLimit|Cashflow|Bill'`。

### Phase 3: 前端交互改造与组件验证（web）
- [ ] **Step 3.1**: 自定义设置面板扩展信用额度管控卡片：
  - `web/src/pages/settings/components/CustomSettingsPanel.tsx` 新增「往来单位信用额度管控策略」卡片与 Switch 开关；
  - 更新 `CustomSettingsPanel.test.tsx`。
- [ ] **Step 3.2**: 选择器超额置灰禁用：
  - `web/src/utils/options.ts` 暴露 `creditExceeded`；
  - 订单及相关选择器配置 `optionRender` 展示 `[已超信用额度]` Tag，超额且干预时置灰禁用；
  - 更新单测。
- [ ] **Step 3.3**: 建账工作台账期带出与预警展示：
  - `web/src/pages/finance/bills/components/BillCreationWorkbench.tsx` & `BillGroupCard.tsx`：
    - 应收账期默认回填（按 Phase 0 定稿的服务端兜底 / 前端预填方案）；
    - 散客修改账期为 > 0 天时，即时高亮黄色预警 Alert；
    - 信用额度超限时，即时高亮黄色超额预警 Alert；
    - 两个预警均为软提示，不阻断提交；
  - 更新对应组件单测。
- [ ] **Step 3.4**: 伙伴主档结算规则维护：
  - `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx` & `SettlementSection.tsx`：默认账期天数录入。

### Phase 4: 综合门禁与质量验收
- [ ] **Step 4.1**: 运行受影响前端文件的 Vitest 定向测试与 Biome 检查；
- [ ] **Step 4.2**: 运行 `pnpm --dir web tsc` 验证前端无类型报错；
- [ ] **Step 4.3**: 运行全套后端测试 `go -C server test ./...` 确保无回归（本任务触及订单 / 费用写入路径，属高风险跨层变更，按门禁要求执行一次完整验证）；
- [ ] **Step 4.4**: 审查变更差异 `git diff --check`，遵循 Conventional Commits 提交。

## 滚回点

- Phase 1 完成后（纯 Schema + 契约 + 生成物）为一个可独立审阅的提交组；
- Phase 2 中 Step 2.1/2.2（设置 + 账期字段）与 Step 2.3（超额判定 + 干预拦截）可分两个提交组，后者失败不影响前者；
- 干预拦截（order/order_fee 校验）出问题时，组织把开关切回「仅提醒」即为线上软降级，无需回滚代码。
