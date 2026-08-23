-- 订单费用录入表；金额、数量和汇率全部使用 PostgreSQL numeric。
CREATE TABLE "order_fees" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "direction" character varying NOT NULL,
  "fee_code" character varying(30) NOT NULL,
  "fee_name" character varying(80) NOT NULL,
  "billing_unit" character varying(32) NOT NULL,
  "quantity" numeric(18,4) NOT NULL,
  "unit_price" numeric(18,4) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "exchange_rate" numeric(18,6) NOT NULL,
  "expense_date" character varying(10) NOT NULL,
  "note" character varying(500) NULL,
  "order_id" uuid NOT NULL,
  "settlement_party_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_fees_direction_check"
    CHECK ("direction" IN ('RECEIVABLE', 'PAYABLE')),
  CONSTRAINT "order_fees_quantity_positive" CHECK ("quantity" > 0),
  CONSTRAINT "order_fees_unit_price_positive" CHECK ("unit_price" > 0),
  CONSTRAINT "order_fees_total_amount_positive" CHECK ("total_amount" > 0),
  CONSTRAINT "order_fees_exchange_rate_positive" CHECK ("exchange_rate" > 0),
  CONSTRAINT "order_fees_orders_fees"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_fees_partners_order_fees"
    FOREIGN KEY ("settlement_party_id") REFERENCES "partners" ("id") ON DELETE NO ACTION
);

CREATE INDEX "orderfee_updated_at"
  ON "order_fees" ("updated_at");

CREATE INDEX "orderfee_order_id_direction_created_at"
  ON "order_fees" ("order_id", "direction", "created_at");

CREATE INDEX "orderfee_settlement_party_id_direction_currency"
  ON "order_fees" ("settlement_party_id", "direction", "currency");

-- 为四类订单写入费用动作权限；已有非管理员角色由管理员按职责显式授权。
CREATE TEMP TABLE order_fee_business_types ("code" varchar NOT NULL, "name" varchar NOT NULL) ON COMMIT DROP;
INSERT INTO order_fee_business_types ("code", "name") VALUES
  ('se', '海运出口（SE）'), ('si', '海运进口（SI）'), ('ae', '空运出口（AE）'), ('ai', '空运进口（AI）');

CREATE TEMP TABLE order_fee_permission_definitions ("operation" varchar NOT NULL, "name" varchar NOT NULL, "description" varchar NOT NULL) ON COMMIT DROP;
INSERT INTO order_fee_permission_definitions ("operation", "name", "description") VALUES
  ('fee.read', '查看费用', '查看订单应收应付费用'),
  ('fee.create', '录入费用', '录入订单应收应付费用'),
  ('fee.update', '编辑费用', '修改订单应收应付费用'),
  ('fee.delete', '删除费用', '删除订单应收应付费用');

INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description")
SELECT md5(format('business.order.%s.%s', business_type."code", definition."operation"))::uuid,
       CURRENT_TIMESTAMP,
       CURRENT_TIMESTAMP,
       format('business.order.%s.%s', business_type."code", definition."operation"),
       format('%s %s', business_type."name", definition."name"),
       format('订单管理 · %s · 费用', business_type."name"),
       definition."description"
FROM order_fee_business_types AS business_type
CROSS JOIN order_fee_permission_definitions AS definition
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT administrator_role."id", permission."id"
FROM "roles" AS administrator_role
JOIN "permissions" AS permission ON permission."key" LIKE 'business.order.%.fee.%'
WHERE administrator_role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
