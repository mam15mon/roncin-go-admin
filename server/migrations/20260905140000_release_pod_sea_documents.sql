-- 放货凭证可显式关联当前真实海运 MBL 或 HBL；不猜测转换存量旧分单引用。
ALTER TABLE "order_release_pods"
  ADD COLUMN "sea_master_bill_id" uuid NULL,
  ADD COLUMN "sea_house_bill_id" uuid NULL;

ALTER TABLE "order_release_pods"
  ADD CONSTRAINT "order_release_pods_document_reference_check" CHECK (
    num_nonnulls("shipping_document_id", "sea_master_bill_id", "sea_house_bill_id") <= 1
  ),
  ADD CONSTRAINT "order_release_pods_sea_master_bills_release_pods"
    FOREIGN KEY ("sea_master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "order_release_pods_sea_house_bills_release_pods"
    FOREIGN KEY ("sea_house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION;

CREATE INDEX "orderreleasepod_sea_master_bill_id"
  ON "order_release_pods" ("sea_master_bill_id");

CREATE INDEX "orderreleasepod_sea_house_bill_id"
  ON "order_release_pods" ("sea_house_bill_id");
