-- 由 Ent Schema.WriteTo 生成并审阅，仅保留本次拼音检索列与索引。
-- 既有数据由迁移命令的 BackfillSelectorSearchKeywords 按字段补齐。
ALTER TABLE "enterprise_resource_addresses" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "enterprise_resource_addresses_search_keywords_trgm" ON "enterprise_resource_addresses" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "enterprise_resource_parties" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "enterprise_resource_parties_search_keywords_trgm" ON "enterprise_resource_parties" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "enterprise_resource_remarks" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "enterprise_resource_remarks_search_keywords_trgm" ON "enterprise_resource_remarks" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_bills" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_bills_search_keywords_trgm" ON "finance_bills" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_cashflows" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_cashflows_search_keywords_trgm" ON "finance_cashflows" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_commission_rules" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_commission_rules_search_keywords_trgm" ON "finance_commission_rules" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_commissions" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_commissions_search_keywords_trgm" ON "finance_commissions" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_invoices" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_invoices_search_keywords_trgm" ON "finance_invoices" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_nettings" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_nettings_search_keywords_trgm" ON "finance_nettings" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "finance_verifications" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "finance_verifications_search_keywords_trgm" ON "finance_verifications" USING GIN ("search_keywords" gin_trgm_ops);
ALTER TABLE "orders" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '', ADD COLUMN "shipper_search_keywords" text NOT NULL DEFAULT '', ADD COLUMN "consignee_search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "orders_search_keywords_trgm" ON "orders" USING GIN ("search_keywords" gin_trgm_ops);
CREATE INDEX "orders_shipper_search_keywords_trgm" ON "orders" USING GIN ("shipper_search_keywords" gin_trgm_ops);
CREATE INDEX "orders_consignee_search_keywords_trgm" ON "orders" USING GIN ("consignee_search_keywords" gin_trgm_ops);
ALTER TABLE "partner_contacts" ADD COLUMN "search_keywords" text NOT NULL DEFAULT '';
CREATE INDEX "partner_contacts_search_keywords_trgm" ON "partner_contacts" USING GIN ("search_keywords" gin_trgm_ops);

-- 统一改用词组读音；只重建派生检索键，不变更业务字段、版本和更新时间。
UPDATE "administrative_regions" SET "search_keywords" = '';
UPDATE "airlines" SET "search_keywords" = '';
UPDATE "airports" SET "search_keywords" = '';
UPDATE "billing_units" SET "search_keywords" = '';
UPDATE "currencies" SET "search_keywords" = '';
UPDATE "enterprise_resources" SET "search_keywords" = '';
UPDATE "fee_settings" SET "search_keywords" = '';
UPDATE "master_data_items" SET "search_keywords" = '';
UPDATE "order_fees" SET "search_keywords" = '';
UPDATE "organizations" SET "search_keywords" = '';
UPDATE "partner_alias" SET "search_keywords" = '';
UPDATE "partners" SET "search_keywords" = '';
UPDATE "ports" SET "search_keywords" = '';
UPDATE "shipping_lines" SET "search_keywords" = '';
UPDATE "taxable_services" SET "search_keywords" = '';
UPDATE "users" SET "search_keywords" = '';
UPDATE "fee_setting_templates" SET "search_keywords" = '';
