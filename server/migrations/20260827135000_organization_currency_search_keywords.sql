ALTER TABLE "organizations" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
ALTER TABLE "currencies" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';

CREATE INDEX "organizations_search_keywords_trgm" ON "organizations" USING gin ("search_keywords" gin_trgm_ops);
CREATE INDEX "currencies_search_keywords_trgm" ON "currencies" USING gin ("search_keywords" gin_trgm_ops);
