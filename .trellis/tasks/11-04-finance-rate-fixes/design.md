# 技术设计

所有权：biz/finance_cashflow.go、order_fee.go、finance_invoice.go 及必要共用汇率校验和测试；web/src/pages/finance/cashflows/** 权限入口和测试。在实际解析本位币后校验手工同币种恒等，非 1 明确领域错误，不静默改为 1。发票从单据 direction 取应收/应付，保持权限/业务日和已有快照语义。前端只改 access 选择，不增加权限码或改变服务端授权。

复用现有领域错误、共享事务与版本号；不改变已付提成事实或历史单据快照，不自动修改已存数据。回滚使用本组提交的反向提交。
