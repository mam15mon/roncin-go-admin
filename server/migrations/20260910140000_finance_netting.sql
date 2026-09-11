CREATE TABLE "finance_nettings"("id" uuid PRIMARY KEY,"created_at" timestamptz NOT NULL,"updated_at" timestamptz NOT NULL,"organization_id" uuid NOT NULL REFERENCES "organizations"("id"),"netting_no" varchar(64) NOT NULL,"idempotency_key" varchar(128) NOT NULL,"request_hash" varchar(64) NOT NULL,"batch_id" uuid REFERENCES "finance_bill_batches"("id") ON DELETE SET NULL,"status" varchar NOT NULL DEFAULT 'DRAFT',"settlement_party_id" uuid NOT NULL REFERENCES "partners"("id"),"settlement_party_name" varchar(200) NOT NULL,"currency" varchar(3) NOT NULL,"amount" numeric(28,8) NOT NULL,"base_currency" varchar(3) NOT NULL,"base_currency_amount" numeric(28,8) NOT NULL,"note" varchar(500),"version" bigint NOT NULL DEFAULT 1,"confirmed_at" timestamptz,"confirmed_by" uuid REFERENCES "users"("id") ON DELETE SET NULL,"cancelled_at" timestamptz,"cancelled_by" uuid REFERENCES "users"("id") ON DELETE SET NULL,"cancellation_reason" varchar(500),"reversed_at" timestamptz,"reversed_by" uuid REFERENCES "users"("id") ON DELETE SET NULL,"reversal_reason" varchar(500),CONSTRAINT "financenetting_status_check" CHECK("status" IN('DRAFT','CONFIRMED','CANCELLED','REVERSED')),CONSTRAINT "financenetting_amount_positive" CHECK("amount">0),CONSTRAINT "financenetting_base_amount_non_negative" CHECK("base_currency_amount">=0));

CREATE TABLE "finance_netting_allocations"("id" uuid PRIMARY KEY,"created_at" timestamptz NOT NULL,"updated_at" timestamptz NOT NULL,"netting_id" uuid NOT NULL REFERENCES "finance_nettings"("id"),"bill_id" uuid NOT NULL REFERENCES "finance_bills"("id"),"bill_no" varchar(64) NOT NULL,"direction" varchar NOT NULL,"amount" numeric(28,8) NOT NULL,"base_currency_amount" numeric(28,8) NOT NULL,"active" boolean NOT NULL DEFAULT false,CONSTRAINT "financenettingallocation_direction_check" CHECK("direction" IN('RECEIVABLE','PAYABLE')),CONSTRAINT "financenettingallocation_amount_positive" CHECK("amount">0));

CREATE UNIQUE INDEX "financenetting_organization_id_netting_no" ON "finance_nettings"("organization_id","netting_no");CREATE UNIQUE INDEX "financenetting_organization_id_idempotency_key" ON "finance_nettings"("organization_id","idempotency_key");CREATE INDEX "financenetting_organization_id_status_created_at" ON "finance_nettings"("organization_id","status","created_at");CREATE INDEX "financenetting_settlement_party_id_currency" ON "finance_nettings"("settlement_party_id","currency");CREATE INDEX "financenetting_batch_id" ON "finance_nettings"("batch_id");CREATE INDEX "financenetting_updated_at" ON "finance_nettings"("updated_at");
CREATE INDEX "financenettingallocation_netting_id_active" ON "finance_netting_allocations"("netting_id","active");CREATE INDEX "financenettingallocation_bill_id_active" ON "finance_netting_allocations"("bill_id","active");CREATE UNIQUE INDEX "netting_allocation_pair_unique" ON "finance_netting_allocations"("netting_id","bill_id");CREATE INDEX "financenettingallocation_updated_at" ON "finance_netting_allocations"("updated_at");

-- 为既有组织幂等补齐对冲单编号规则；新组织由创建流程按默认规则集生成。
INSERT INTO "number_rules" (
    "id",
    "created_at",
    "updated_at",
    "document_type",
    "prefix",
    "date_format",
    "sequence_length",
    "reset_policy",
    "enabled",
    "organization_id"
)
SELECT
    gen_random_uuid(),
    NOW(),
    NOW(),
    'netting',
    'NT',
    'yyyyMMdd',
    5,
    'daily',
    true,
    organizations.id
FROM "organizations"
ON CONFLICT ("organization_id", "document_type") DO NOTHING;
