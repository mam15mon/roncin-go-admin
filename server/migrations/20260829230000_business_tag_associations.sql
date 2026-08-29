CREATE TABLE "order_enterprise_tags" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "tag_resource_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_enterprise_tags_enterprise_resources_order_tag_links" FOREIGN KEY ("tag_resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_enterprise_tags_orders_enterprise_tag_links" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_enterprise_tags_organizations_order_enterprise_tags" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "orderenterprisetag_updated_at" ON "order_enterprise_tags" ("updated_at");
CREATE UNIQUE INDEX "orderenterprisetag_order_id_tag_resource_id" ON "order_enterprise_tags" ("order_id", "tag_resource_id");
CREATE INDEX "orderenterprisetag_organization_id_tag_resource_id" ON "order_enterprise_tags" ("organization_id", "tag_resource_id");
CREATE INDEX "orderenterprisetag_order_id" ON "order_enterprise_tags" ("order_id");

CREATE TABLE "order_fee_enterprise_tags" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "tag_resource_id" uuid NOT NULL,
  "order_fee_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_fee_enterprise_tags_enterprise_resources_order_fee_tag_links" FOREIGN KEY ("tag_resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_fee_enterprise_tags_order_fees_enterprise_tag_links" FOREIGN KEY ("order_fee_id") REFERENCES "order_fees" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_fee_enterprise_tags_organizations_order_fee_enterprise_tags" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "orderfeeenterprisetag_updated_at" ON "order_fee_enterprise_tags" ("updated_at");
CREATE UNIQUE INDEX "orderfeeenterprisetag_order_fee_id_tag_resource_id" ON "order_fee_enterprise_tags" ("order_fee_id", "tag_resource_id");
CREATE INDEX "orderfeeenterprisetag_organization_id_tag_resource_id" ON "order_fee_enterprise_tags" ("organization_id", "tag_resource_id");
CREATE INDEX "orderfeeenterprisetag_order_fee_id" ON "order_fee_enterprise_tags" ("order_fee_id");

CREATE TABLE "finance_bill_enterprise_tags" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "tag_resource_id" uuid NOT NULL,
  "finance_bill_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_bill_enterprise_tags_enterprise_resources_finance_bill_tag_links" FOREIGN KEY ("tag_resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bill_enterprise_tags_finance_bills_enterprise_tag_links" FOREIGN KEY ("finance_bill_id") REFERENCES "finance_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bill_enterprise_tags_organizations_finance_bill_enterprise_tags" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "financebillenterprisetag_updated_at" ON "finance_bill_enterprise_tags" ("updated_at");
CREATE UNIQUE INDEX "financebillenterprisetag_finance_bill_id_tag_resource_id" ON "finance_bill_enterprise_tags" ("finance_bill_id", "tag_resource_id");
CREATE INDEX "financebillenterprisetag_organization_id_tag_resource_id" ON "finance_bill_enterprise_tags" ("organization_id", "tag_resource_id");
CREATE INDEX "financebillenterprisetag_finance_bill_id" ON "finance_bill_enterprise_tags" ("finance_bill_id");

DROP INDEX IF EXISTS "order_tags_gin";
ALTER TABLE "orders" DROP COLUMN IF EXISTS "tags";
