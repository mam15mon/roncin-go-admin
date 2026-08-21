-- 放货凭证表；结构与 ent/schema/order_release_pod.go 保持一致。
CREATE TABLE "order_release_pods" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "release_no" character varying(64) NULL,
  "pod_no" character varying(64) NULL,
  "status" character varying NOT NULL DEFAULT 'PENDING',
  "signed_at" timestamp with time zone NULL,
  "signed_by" uuid NULL,
  "note" character varying(500) NULL,
  "order_id" uuid NOT NULL,
  "shipping_document_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_release_pods_status_check"
    CHECK ("status" IN ('PENDING', 'SIGNED', 'RETURNED')),
  CONSTRAINT "order_release_pods_orders_release_pods"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_release_pods_order_shipping_documents_release_pods"
    FOREIGN KEY ("shipping_document_id") REFERENCES "order_shipping_documents" ("id") ON DELETE SET NULL
);

CREATE INDEX "orderreleasepod_updated_at"
  ON "order_release_pods" ("updated_at");

CREATE INDEX "orderreleasepod_order_id_status"
  ON "order_release_pods" ("order_id", "status");

CREATE INDEX "orderreleasepod_shipping_document_id"
  ON "order_release_pods" ("shipping_document_id");
