# 技术设计

## commission
所有权：biz/finance_commission_rules.go、data/finance_commission_rules.go、finance_commission_write.go、finance_commission_application*.go 中门禁及定向测试。复用有效费用谓词，避免漏掉月申/整批入口。创建和更新分离日期准入；仓储锁内保留生效参数限制。更新未来启用方案先获取完整未取消分配员工，排序锁 Membership，再锁方案和复核成员集合，按方案与每段分配的交集检查冲突并排除当前方案，禁止锁序倒置。

## bill
所有权：data/finance_bill*.go、finance_netting.go、fee_correction_commission.go、order_fee_supplement_lock.go 中必要公共锁辅助及测试。复用现有经济投影和系统调整机制，不新增状态或数据库列。建账/更新/取消需在首个订单锁前展开关联提成订单闭包，一次排序加锁并重验，保持订单→账单→费用→经济调整既有权威序。单批多账单完成后统一重算避免中间态；同一事务写审计和调整。对冲确认在账单锁内重取当前行快照重算双端本位币，不接受草稿旧值。单张创建比较费用事实与账单派生金额各自来源，不再跨日期比较本币快照。

## rate
所有权：biz/finance_cashflow.go、order_fee.go、finance_invoice.go 及必要共用汇率校验和测试；web/src/pages/finance/cashflows/** 权限入口和测试。在实际解析本位币后校验手工同币种恒等，非 1 明确领域错误，不静默改为 1。发票从单据 direction 取应收/应付，保持权限/业务日和已有快照语义。前端只改 access 选择，不增加权限码或改变服务端授权。

三组共享现有业务契约；父任务同步领域规范并独立审核金额、失败原子性和锁序，子任务测试分别验收后再做整体门禁。无 API 或数据库列结构变更。

## R3 模型约束修正
对冲头的双端本币/汇差与分摊本币在 Ent 中标为 Immutable，生成 Update API 无对应 setter。为支持草稿确认时按锁内最新账单刷新快照，移除这些金额字段的 Immutable 并通过正式生成器更新 Ent；数据库列形不变，无 SQL 迁移。已确认状态仍不允许编辑，重复确认不得重算已生效金额。
