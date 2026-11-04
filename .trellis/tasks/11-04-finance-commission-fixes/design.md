# 技术设计

所有权：biz/finance_commission_rules.go、data/finance_commission_rules.go、finance_commission_write.go、finance_commission_application*.go 中门禁及定向测试。复用有效费用谓词，避免漏掉月申/整批入口。创建和更新分离日期准入；仓储锁内保留生效参数限制。更新未来启用方案先获取完整未取消分配员工，排序锁 Membership，再锁方案和复核成员集合，按方案与每段分配的交集检查冲突并排除当前方案，禁止锁序倒置。

复用现有领域错误、共享事务与版本号；不改变已付提成事实或历史单据快照，不自动修改已存数据。回滚使用本组提交的反向提交。
