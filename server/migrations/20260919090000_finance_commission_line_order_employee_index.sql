-- 提成行明细查询索引：订单列表提成摘要（EMPLOYEE 视图）按「页面订单 + 员工」
-- 精确定位 finance_commission_lines。新索引与 financecommissionline_order_id
-- 同样以 order_id 前导，叠加 employee_id 后，统计信息缺失时计划不再退化为
-- employee_id 位图扫描命中组织全量提成行、再与父提成单做无哈希 Join Filter
-- 的形态。仅新增索引，不改动任何数据。

CREATE INDEX "financecommissionline_order_id_employee_id"
  ON "finance_commission_lines" ("order_id", "employee_id");
