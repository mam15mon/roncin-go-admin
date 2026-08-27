CREATE EXTENSION IF NOT EXISTS "pg_trgm";

ALTER TABLE "partners" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "partner_alias" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "master_data_items" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "ports" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "airports" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "airlines" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "shipping_lines" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "administrative_regions" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "users" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "fee_settings" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';

CREATE INDEX "partners_search_keywords_trgm" ON "partners" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "partner_alias_search_keywords_trgm" ON "partner_alias" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "master_data_items_search_keywords_trgm" ON "master_data_items" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "ports_search_keywords_trgm" ON "ports" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "airports_search_keywords_trgm" ON "airports" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "airlines_search_keywords_trgm" ON "airlines" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "shipping_lines_search_keywords_trgm" ON "shipping_lines" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "administrative_regions_search_keywords_trgm" ON "administrative_regions" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "users_search_keywords_trgm" ON "users" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "fee_settings_search_keywords_trgm" ON "fee_settings" USING gin ("search_keywords" gin_trgm_ops);
