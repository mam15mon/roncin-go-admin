-- 将主单提升为跨订单共享的拼载批次，分单继续归属于具体订单。
CREATE TABLE "order_consolidations" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "business_type" character varying NOT NULL,
  "master_no" character varying NOT NULL,
  "normalized_master_no" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_consolidations_organizations_order_consolidations"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_consolidations_business_type_check"
    CHECK ("business_type" IN ('SE', 'SI', 'AE', 'AI', 'LAND', 'RAIL'))
);

CREATE UNIQUE INDEX "orderconsolidation_organization_id_business_type_normalized_master_no"
  ON "order_consolidations" ("organization_id", "business_type", "normalized_master_no");

CREATE INDEX "orderconsolidation_organization_id_business_type_master_no"
  ON "order_consolidations" ("organization_id", "business_type", "master_no");

CREATE INDEX "orderconsolidation_updated_at"
  ON "order_consolidations" ("updated_at");

INSERT INTO "order_consolidations" (
  "id", "created_at", "updated_at", "organization_id", "business_type",
  "master_no", "normalized_master_no"
)
SELECT
  md5('order_consolidation:' || source."organization_id" || ':' || source."business_type" || ':' || source."normalized_master_no")::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  source."organization_id",
  source."business_type",
  MIN(source."master_no"),
  source."normalized_master_no"
FROM (
  SELECT
    orders."organization_id",
    orders."business_type",
    order_shipping_documents."master_no",
    LOWER(BTRIM(order_shipping_documents."master_no")) AS "normalized_master_no"
  FROM "order_shipping_documents"
  JOIN "orders" ON orders."id" = order_shipping_documents."order_id"
) AS source
GROUP BY source."organization_id", source."business_type", source."normalized_master_no";

ALTER TABLE "order_shipping_documents"
  ADD COLUMN "consolidation_id" uuid;

UPDATE "order_shipping_documents"
SET "consolidation_id" = order_consolidations."id"
FROM "orders", "order_consolidations"
WHERE orders."id" = order_shipping_documents."order_id"
  AND order_consolidations."organization_id" = orders."organization_id"
  AND order_consolidations."business_type" = orders."business_type"
  AND order_consolidations."normalized_master_no" = LOWER(BTRIM(order_shipping_documents."master_no"));

DROP INDEX "ordershippingdocument_order_id_master_no";

ALTER TABLE "order_shipping_documents"
  ALTER COLUMN "consolidation_id" SET NOT NULL,
  ADD CONSTRAINT "order_shipping_documents_order_consolidations_shipping_documents"
    FOREIGN KEY ("consolidation_id") REFERENCES "order_consolidations" ("id") ON DELETE NO ACTION,
  DROP COLUMN "master_no";

CREATE INDEX "ordershippingdocument_order_id_consolidation_id"
  ON "order_shipping_documents" ("order_id", "consolidation_id");
