ALTER TABLE "billing_units" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "taxable_services" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';

CREATE INDEX "billing_units_search_keywords_trgm" ON "billing_units" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "taxable_services_search_keywords_trgm" ON "taxable_services" USING gin ("search_keywords" gin_trgm_ops);
