# 提成 CNY 快照与归属日期（阶段 1）

## Goal

`FinanceCommission` 新增不可变的 CNY 快照与归属日期字段；生成草稿事务内解析
并固化；预览返回折算依据。对应计划 §10 阶段 1。

## Requirements

- 新增字段（全部 immutable，见计划 §5）：`commission_date`、
  `cny_exchange_rate numeric(18,8)`、`cny_exchange_rate_source`
  （BASE_CURRENCY/DERIVED）、`cny_exchange_rate_date`、
  `cny_exchange_rate_setting_id`（可空）、`cny_commission_amount numeric(28,8)`。
- 索引：`organization_id + commission_date`。
- `commission_date = verification.verification_date`，生成时快照；重生成从本次
  核销单重新派生，不沿用已取消提成的旧快照。
- 折算规则严格按计划 §4.2.1：本位币为 CNY 时 rate=1/source=BASE_CURRENCY/
  setting_id=null；非 CNY 时倒数派生 rate=round(1/source,8)/source=DERIVED，
  decimal 计算禁浮点。
- 汇率缺失/冲突/日期无效：预览与创建均显式报错，不落库（§4.2.2）；创建不依赖
  预览已调用，事务内重新解析。
- `PreviewCommission` 响应增加将使用的 CNY 折算率、汇率日期与来源；
  `ListCommissions` / `GetCommission` 返回新字段与动态 CNY 调整/有效金额
  （§4.2.3，调整继承主单汇率，不落库）。
- 存量数据按 §4.2.7 前提：无历史记录则非空无默认值，本地库重建获得最终结构。

## Acceptance Criteria

- [ ] 本位币 CNY：汇率恒 1、来源 BASE_CURRENCY、setting ID 空。
- [ ] 非 CNY 本币：倒数折算、来源 DERIVED、setting ID 回填，倒数精度断言。
- [ ] 汇率缺失/日期无效时预览与创建均报错，不落库。
- [ ] 预览后修改汇率配置，创建按事务内重新解析结果写入。
- [ ] `commission_date` 等于核销日期；同核销单重生成按新快照。
- [ ] CNY 调整/有效金额与调整状态机一致（草稿、取消不计入）。
- [ ] 既有生成、确认、支付、调整链路不回归。
- [ ] 契约变更走 proto 先行，生成物同提交；`go -C server test ./...` 通过。

## Notes

- 依赖：阶段 0 口径全部冻结（见父任务 prd 前置条件）。
- 提交信息：`feat: 增加提成 CNY 快照与归属日期`。
- 实现前需补 `design.md`（字段/折算/错误路径设计）与 `implement.md`。
