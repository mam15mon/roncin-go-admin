-- 海运出口船公司改用 ShippingLine 行业主数据；不猜测映射既有业务数据。
-- 所有前置检查必须在 DDL 前完成，迁移执行器会在单一事务中原子回滚。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' OR "carrier_id" IS NOT NULL LIMIT 1) THEN
    RAISE EXCEPTION '海运船公司身份迁移已停止：orders 存在待治理的承运人引用';
  END IF;
  IF EXISTS (SELECT 1 FROM "sea_transport_executions" LIMIT 1) THEN
    RAISE EXCEPTION '海运船公司身份迁移已停止：sea_transport_executions 存在历史数据';
  END IF;
  IF EXISTS (SELECT 1 FROM "sea_master_bills" LIMIT 1) THEN
    RAISE EXCEPTION '海运船公司身份迁移已停止：sea_master_bills 存在历史数据';
  END IF;
  IF EXISTS (SELECT 1 FROM "sea_master_bill_versions" LIMIT 1) THEN
    RAISE EXCEPTION '海运船公司身份迁移已停止：sea_master_bill_versions 存在历史数据';
  END IF;
  IF EXISTS (
    SELECT 1 FROM "partner_roles"
    WHERE "role_type" NOT IN ('customer', 'supplier', 'foreign_agent')
    LIMIT 1
  ) THEN
    RAISE EXCEPTION '海运船公司身份迁移已停止：partner_roles 存在 carrier 或其他非法角色';
  END IF;
END $$;

DROP INDEX "order_organization_id_carrier_id";
ALTER TABLE "orders" RENAME COLUMN "carrier_id" TO "shipping_line_id";
CREATE INDEX "order_organization_id_shipping_line_id"
  ON "orders" ("organization_id", "shipping_line_id");
ALTER TABLE "orders"
  ADD CONSTRAINT "orders_shipping_lines_orders"
  FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id") ON DELETE NO ACTION;

DROP INDEX "seatransportexecution_organization_id_carrier_id";
ALTER TABLE "sea_transport_executions" RENAME COLUMN "carrier_id" TO "shipping_line_id";
ALTER TABLE "sea_transport_executions" ALTER COLUMN "shipping_line_id" SET NOT NULL;
CREATE INDEX "seatransportexecution_organization_id_shipping_line_id"
  ON "sea_transport_executions" ("organization_id", "shipping_line_id");
ALTER TABLE "sea_transport_executions"
  ADD CONSTRAINT "sea_transport_executions_shipping_lines_sea_transport_executions"
  FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id") ON DELETE NO ACTION;

DROP INDEX "seamasterbill_organization_id_issuer_partner_id_normalized_master_no";
ALTER TABLE "sea_master_bills" RENAME COLUMN "issuer_partner_id" TO "shipping_line_id";
CREATE UNIQUE INDEX "seamasterbill_organization_id_shipping_line_id_normalized_master_no"
  ON "sea_master_bills" ("organization_id", "shipping_line_id", "normalized_master_no");
ALTER TABLE "sea_master_bills"
  ADD CONSTRAINT "sea_master_bills_shipping_lines_sea_master_bills"
  FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id") ON DELETE NO ACTION;

ALTER TABLE "sea_master_bill_versions"
  DROP CONSTRAINT "sea_master_bill_versions_partners_sea_master_bill_versions";
ALTER TABLE "sea_master_bill_versions" RENAME COLUMN "issuer_partner_id" TO "shipping_line_id";
ALTER TABLE "sea_master_bill_versions" DROP COLUMN "carrier_id";
ALTER TABLE "sea_master_bill_versions"
  ADD CONSTRAINT "sea_master_bill_versions_shipping_lines_sea_master_bill_versions"
  FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id") ON DELETE NO ACTION;

ALTER TABLE "partner_roles"
  ADD CONSTRAINT "partner_roles_role_type_check"
  CHECK ("role_type" IN ('customer', 'supplier', 'foreign_agent'));
