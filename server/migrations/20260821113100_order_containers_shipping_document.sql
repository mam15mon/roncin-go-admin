-- 订单集装箱支持关联提单。
ALTER TABLE "order_containers"
  ADD COLUMN "shipping_document_id" uuid NULL;

ALTER TABLE "order_containers"
  ADD CONSTRAINT "order_containers_order_shipping_documents_containers"
  FOREIGN KEY ("shipping_document_id") REFERENCES "order_shipping_documents" ("id") ON DELETE SET NULL;

CREATE INDEX "ordercontainer_shipping_document_id"
  ON "order_containers" ("shipping_document_id");