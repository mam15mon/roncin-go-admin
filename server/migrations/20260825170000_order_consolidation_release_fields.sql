-- 主单批次独立维护单证类型和签放方式，不再复用分单 release_type。
ALTER TABLE "order_consolidations"
  ADD COLUMN "document_type" character varying(64) NULL,
  ADD COLUMN "release_method" character varying(64) NULL;
