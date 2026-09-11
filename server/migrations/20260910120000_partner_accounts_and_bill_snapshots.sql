DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "partner_accounts") OR EXISTS (SELECT 1 FROM "finance_bills") THEN
    RAISE EXCEPTION '本迁移不兼容旧开发数据：partner_accounts 或 finance_bills 已有记录，需经授权清理或重建后再执行迁移';
  END IF;
END $$;

ALTER TABLE "partner_accounts"
  ADD COLUMN "partner_id" uuid NOT NULL,
  ADD COLUMN "name" character varying NOT NULL,
  ADD COLUMN "account_holder" character varying NOT NULL,
  ADD COLUMN "account_no" character varying NOT NULL,
  ADD COLUMN "usage" character varying NOT NULL,
  ADD COLUMN "is_default_receivable" boolean NOT NULL DEFAULT false,
  ADD COLUMN "is_default_payable" boolean NOT NULL DEFAULT false,
  ADD COLUMN "enabled" boolean NOT NULL DEFAULT true;

ALTER TABLE "partner_accounts"
  ALTER COLUMN "bank_name" SET NOT NULL,
  ADD CONSTRAINT "partner_accounts_partners_accounts" FOREIGN KEY ("partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "partner_accounts_default_usage_check" CHECK (((NOT "is_default_receivable" OR ("enabled" AND "usage" IN ('RECEIVABLE', 'BOTH'))) AND (NOT "is_default_payable" OR ("enabled" AND "usage" IN ('PAYABLE', 'BOTH')))));

DROP INDEX "partner_account_default_key";
DROP INDEX "partneraccount_partner_role_id_status";
DROP INDEX "partneraccount_partner_role_id_created_at";
ALTER TABLE "partner_accounts" DROP CONSTRAINT "partner_accounts_partner_roles_accounts";
ALTER TABLE "partner_accounts" DROP COLUMN "partner_role_id", DROP COLUMN "account_type", DROP COLUMN "bank_account", DROP COLUMN "status";

CREATE UNIQUE INDEX "partner_account_default_receivable_key" ON "partner_accounts" ("partner_id", "currency") WHERE "is_default_receivable";
CREATE UNIQUE INDEX "partner_account_default_payable_key" ON "partner_accounts" ("partner_id", "currency") WHERE "is_default_payable";
CREATE INDEX "partneraccount_partner_id_enabled_currency" ON "partner_accounts" ("partner_id", "enabled", "currency");
CREATE INDEX "partneraccount_partner_id_created_at" ON "partner_accounts" ("partner_id", "created_at");

ALTER TABLE "finance_bills"
  ADD COLUMN "settlement_account_id" uuid NOT NULL,
  ADD COLUMN "settlement_account_name" character varying NOT NULL,
  ADD COLUMN "settlement_account_holder" character varying NOT NULL,
  ADD COLUMN "settlement_bank_name" character varying NOT NULL,
  ADD COLUMN "settlement_bank_account" character varying NOT NULL,
  ADD COLUMN "settlement_account_currency" character varying NOT NULL,
  ADD COLUMN "settlement_swift_code" character varying NULL;

CREATE INDEX "financebill_settlement_account_id" ON "finance_bills" ("settlement_account_id");
