ALTER TABLE "finance_commissions"
  ADD COLUMN "commission_date" varchar(10) NOT NULL,
  ADD COLUMN "cny_exchange_rate" numeric(18,8) NOT NULL,
  ADD COLUMN "cny_exchange_rate_source" varchar NOT NULL,
  ADD COLUMN "cny_exchange_rate_date" varchar(10) NOT NULL,
  ADD COLUMN "cny_exchange_rate_setting_id" uuid,
  ADD COLUMN "cny_commission_amount" numeric(28,8) NOT NULL,
  ADD CONSTRAINT "commission_cny_exchange_rate_source_check" CHECK("cny_exchange_rate_source" IN('BASE_CURRENCY','DERIVED'));

CREATE INDEX "financecommission_org_commission_date" ON "finance_commissions"("organization_id","commission_date");
