-- 客户同一角色可配置多名人员，订单需分别固化每个人员的提成归属。
DROP INDEX "ordercommissionattribution_order_role";

CREATE UNIQUE INDEX "ordercommissionattribution_order_employee_role"
  ON "order_commission_attributions"("order_id","employee_id","personnel_role");

-- 第一版历史回填每个角色只保留一人，此处补齐同角色的其他客户人员。
INSERT INTO "order_commission_attributions"(
  "id","created_at","updated_at","organization_id","order_id","customer_id",
  "source_assignment_id","employee_id","employee_name","personnel_role","attributed_at"
)
SELECT gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,"o"."organization_id","o"."id","o"."customer_id",
       "pa"."id","pa"."user_id","u"."display_name","pa"."role","o"."created_at"
FROM "orders" AS "o"
JOIN "partner_assignments" AS "pa"
  ON "pa"."partner_id"="o"."customer_id" AND "pa"."organization_id"="o"."organization_id"
JOIN "users" AS "u" ON "u"."id"="pa"."user_id"
WHERE "pa"."role" IN('SALES','OPERATOR','CUSTOMER_SERVICE')
ON CONFLICT ("order_id","employee_id","personnel_role") DO NOTHING;
