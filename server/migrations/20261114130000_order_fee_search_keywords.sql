-- 费用名称快照独立维护检索键，保留受控更正的人工名称与历史名称。
-- 既有费用的拼音由迁移命令完成 Schema 后统一调用 BackfillSelectorSearchKeywords 补齐。
ALTER TABLE "order_fees" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';

CREATE INDEX "order_fees_search_keywords_trgm" ON "order_fees" USING gin ("search_keywords" gin_trgm_ops);
