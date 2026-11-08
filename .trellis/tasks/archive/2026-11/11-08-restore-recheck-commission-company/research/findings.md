# 现场核验与根因

## 恢复问题现状

只读数据库：SE2026110800002为ACTIVE/version8，旧Link01a1bbd3-1906-7d3c-b37f-212871a37440仍ENDED/version3，补链新Link01a1bc0e-f08b-7968-a98a-82635a7f0b9c为ACTIVE/version1，均指向CODEXHISTORY1108。终止事件仍仅原三条，没有新的TERMINATED→ACTIVE事件。
正式HTTP接口：合法登录并切换北京公司后，GET订单详情和sea-documents均200且返回原主单。当前没有复发证据；本次不重复退关/恢复，不再次补链。

## 调整公司名称

正式API：CM2026110800003所属公司=融迅（北京）供应链管理有限公司；详情.adjustments中ADJ001/ADJ002均CANCELLED、有organizationId但无organizationName。独立GET commission-adjustments对相同记录返回完整公司名。

- web/src/pages/finance/commissions/index.tsx:221/274：确认及取消直接读取record.organizationName，缺省展示所属公司未标识。
- server/internal/data/finance_commission_ledger.go:48/94/132/158：列表、导出、GetByKey、GetScoped预载父单公司，但WithAdjustments只排序，未预载调整的Organization。
- server/internal/data/finance_commission_adjustment.go:432：adjustmentToBiz仅从调整.Edges.Organization映射名称；独立列表/详情会WithOrganization，故名称正确。
- server/internal/data/finance_commission_mapping.go:26：内嵌调整复用上述转换，未预载导致空值；service settlement_commission.go已有OrganizationName字段传递，无需新DTO或手改生成代码。

金额逻辑本次用户已验证：+1确认后有效额7.67，取消回6.67；冲减10被拒。当前ADJ001/ADJ002均已取消；本次只读核验，不重放财务动作。
