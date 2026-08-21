-- 订单提单表；结构与 ent/schema/order_shipping_document.go 保持一致。
CREATE TABLE "order_shipping_documents" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "master_no" character varying(64) NOT NULL,
  "house_no" character varying(64) NOT NULL,
  "release_type" character varying(64) NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
  "note" character varying(500) NULL,
  "order_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_shipping_documents_status_check"
    CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED')),
  CONSTRAINT "order_shipping_documents_orders_shipping_documents"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "ordershippingdocument_updated_at"
  ON "order_shipping_documents" ("updated_at");

CREATE UNIQUE INDEX "ordershippingdocument_order_id_master_no"
  ON "order_shipping_documents" ("order_id", "master_no");

CREATE INDEX "ordershippingdocument_order_id_status"
  ON "order_shipping_documents" ("order_id", "status");