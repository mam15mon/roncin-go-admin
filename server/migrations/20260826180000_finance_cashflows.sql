-- 收付资金层：独立保存真实资金进出，确认流水后仍须通过核销关联账单。
CREATE TABLE "finance_cashflows" (
 "id" uuid NOT NULL,"created_at" timestamptz NOT NULL,"updated_at" timestamptz NOT NULL,"organization_id" uuid NOT NULL,"flow_no" varchar(64) NOT NULL,"idempotency_key" varchar(128) NOT NULL,
 "direction" varchar NOT NULL,"status" varchar NOT NULL DEFAULT 'DRAFT',"settlement_party_id" uuid NOT NULL,"settlement_party_name" varchar(200) NOT NULL,"currency" varchar(3) NOT NULL,
 "amount" numeric(28,8) NOT NULL,"exchange_rate" numeric(18,8) NOT NULL,"base_currency" varchar(3) NOT NULL,"base_amount" numeric(28,8) NOT NULL,"transaction_date" varchar(10) NOT NULL,
 "our_account" varchar(200) NOT NULL,"counterparty_account" varchar(200) NULL,"payment_method" varchar(50) NOT NULL,"bank_reference_no" varchar(100) NULL,"note" varchar(500) NULL,"version" bigint NOT NULL DEFAULT 1,
 "confirmed_at" timestamptz NULL,"confirmed_by" uuid NULL,"cancelled_at" timestamptz NULL,"cancelled_by" uuid NULL,"cancellation_reason" varchar(500) NULL,PRIMARY KEY("id"),
 CONSTRAINT "finance_cashflows_org_fk" FOREIGN KEY("organization_id") REFERENCES "organizations"("id"),CONSTRAINT "finance_cashflows_party_fk" FOREIGN KEY("settlement_party_id") REFERENCES "partners"("id"),CONSTRAINT "finance_cashflows_confirmed_by_fk" FOREIGN KEY("confirmed_by") REFERENCES "users"("id"),CONSTRAINT "finance_cashflows_cancelled_by_fk" FOREIGN KEY("cancelled_by") REFERENCES "users"("id"),
 CONSTRAINT "finance_cashflows_direction_check" CHECK("direction" IN('RECEIVABLE','PAYABLE')),CONSTRAINT "finance_cashflows_status_check" CHECK("status" IN('DRAFT','CONFIRMED','CANCELLED')),CONSTRAINT "finance_cashflows_amount_positive" CHECK("amount">0 AND "exchange_rate">0 AND "base_amount">0),CONSTRAINT "finance_cashflows_version_positive" CHECK("version">0),
 CONSTRAINT "finance_cashflows_confirm_consistency" CHECK(("status"='DRAFT' AND "confirmed_at" IS NULL AND "confirmed_by" IS NULL) OR ("status"<>'DRAFT' AND (("confirmed_at" IS NULL AND "confirmed_by" IS NULL) OR ("confirmed_at" IS NOT NULL AND "confirmed_by" IS NOT NULL)))),
 CONSTRAINT "finance_cashflows_cancel_consistency" CHECK(("status"='CANCELLED' AND "cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL AND "cancellation_reason" IS NOT NULL) OR ("status"<>'CANCELLED' AND "cancelled_at" IS NULL AND "cancelled_by" IS NULL AND "cancellation_reason" IS NULL))
);
CREATE UNIQUE INDEX "financecashflow_organization_id_flow_no" ON "finance_cashflows"("organization_id","flow_no");
CREATE UNIQUE INDEX "financecashflow_organization_id_idempotency_key" ON "finance_cashflows"("organization_id","idempotency_key");
CREATE INDEX "financecashflow_organization_id_status_transaction_date" ON "finance_cashflows"("organization_id","status","transaction_date");
CREATE INDEX "financecashflow_settlement_party_id_direction_currency" ON "finance_cashflows"("settlement_party_id","direction","currency");
CREATE INDEX "financecashflow_bank_reference_no" ON "finance_cashflows"("bank_reference_no");
