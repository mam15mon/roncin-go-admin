ALTER TABLE "enterprise_tag_groups" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';

CREATE INDEX "enterprise_tag_groups_search_keywords_trgm" ON "enterprise_tag_groups" USING gin ("search_keywords" gin_trgm_ops);
