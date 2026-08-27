-- 固化订单创建时客户档案中的业务、操作和客服归属，后续客户换人不影响历史订单提成。
CREATE TABLE "order_commission_attributions"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "order_id" uuid NOT NULL REFERENCES "orders"("id"),
  "customer_id" uuid NOT NULL REFERENCES "partners"("id"),
  "source_assignment_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL REFERENCES "users"("id"),
  "employee_name" varchar(100) NOT NULL,
  "personnel_role" varchar NOT NULL,
  "attributed_at" timestamptz NOT NULL,
  CONSTRAINT "order_commission_attribution_role_check" CHECK("personnel_role" IN('SALES','OPERATOR','CUSTOMER_SERVICE'))
);

CREATE UNIQUE INDEX "ordercommissionattribution_order_role"
  ON "order_commission_attributions"("order_id","personnel_role");
CREATE INDEX "ordercommissionattribution_org_employee_role"
  ON "order_commission_attributions"("organization_id","employee_id","personnel_role");
CREATE INDEX "ordercommissionattribution_customer_role"
  ON "order_commission_attributions"("customer_id","personnel_role");

-- 开发期历史订单以迁移时仍有效的客户人员补建快照；新订单一律在创建事务内固化。
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
ON CONFLICT ("order_id","personnel_role") DO NOTHING;
