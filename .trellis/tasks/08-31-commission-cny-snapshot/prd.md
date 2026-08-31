# 提成 CNY 快照与归属日期（阶段 1）

## Goal

`FinanceCommission` 新增不可变的 CNY 快照与归属日期字段；生成草稿事务内解析
并固化；预览返回折算依据。对应计划 §10 阶段 1。

## Confirmed Facts

- 当前处于开发阶段且没有历史提成数据；新字段直接使用非空约束且不设置业务默认值。
- 开发数据库通过重建获得最终结构，不实现历史回填或兼容读取。
- 复用现有共享事务、`Data.client(ctx)` 与汇率查询 `ForShare`，不新增事务框架。

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
- 创建用例通过 `biz.Transactor.WithinTransaction` 编排生成上下文读取、汇率解析
  和提成写入；相关仓储读取使用 `Data.client(ctx)`，完整响应在提交后重读。
- `PreviewCommission` 响应增加将使用的 CNY 折算率、汇率日期与来源；
  `ListCommissions` / `GetCommission` 返回新字段与动态 CNY 调整/有效金额
  （§4.2.3，调整继承主单汇率，不落库）。
- `CommissionFilter` 与 `ListCommissions` 增加 `commission_date_from/to`；日期为
  包含边界的 `YYYY-MM-DD`，允许单边，双边必须 `from <= to`。
- 默认排序为 `commission_date DESC, created_at DESC, id DESC`；列表与阶段 2
  导出复用同一个数据层筛选谓词实现。
- CNY 调整金额按汇总后的本位币调整金额乘主单汇率并 round(8)，有效 CNY 金额为
  原始 CNY 金额加动态 CNY 调整金额。

## Acceptance Criteria

- [ ] 本位币 CNY：汇率恒 1、来源 BASE_CURRENCY、setting ID 空。
- [ ] 非 CNY 本币：倒数折算、来源 DERIVED、setting ID 回填，倒数精度断言。
- [ ] 汇率缺失/日期无效时预览与创建均报错，不落库。
- [ ] 预览后修改汇率配置，创建按事务内重新解析结果写入。
- [ ] `commission_date` 等于核销日期；同核销单重生成按新快照。
- [ ] CNY 调整/有效金额与调整状态机一致（草稿、取消不计入）。
- [ ] 日期单边、双边、同日、非法格式和 `from > to` 均有验证。
- [ ] 列表使用 `commission_date DESC, created_at DESC, id DESC` 稳定排序。
- [ ] 汇率解析、提成保存或审计任一步失败均完整回滚。
- [ ] 既有生成、确认、支付、调整链路不回归。
- [ ] 契约变更走 proto 先行，生成物同提交；`go -C server test ./...` 通过。

## Out of Scope

- 历史数据回填、零值默认、自动兼容和实时折算。
- 明细行 CNY、逐笔调整 CNY 快照、导出接口和前端页面。

## Notes

- 依赖：阶段 0 口径已冻结（见父任务 prd 前置条件）。
- 提交信息：`feat: 增加提成 CNY 快照与归属日期`。
- `design.md` 与 `implement.md` 已补齐；启动前必须通过上下文校验和最终规划批准。
