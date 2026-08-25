DROP INDEX "ordershippingdocument_order_id_master_no";

CREATE UNIQUE INDEX "ordershippingdocument_order_id_house_no"
  ON "order_shipping_documents" ("order_id", "house_no");

CREATE INDEX "ordershippingdocument_order_id_master_no"
  ON "order_shipping_documents" ("order_id", "master_no");

CREATE TABLE "order_container_requests" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "order_id" uuid NOT NULL,
  "container_spec_id" uuid NOT NULL,
  "quantity" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_container_requests_orders_container_requests"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "ordercontainerrequest_updated_at"
  ON "order_container_requests" ("updated_at");

CREATE UNIQUE INDEX "ordercontainerrequest_order_id_container_spec_id"
  ON "order_container_requests" ("order_id", "container_spec_id");

CREATE INDEX "ordercontainerrequest_container_spec_id"
  ON "order_container_requests" ("container_spec_id");
