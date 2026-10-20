-- 空运重量采用不可变确认记录；费用保存采用来源快照，货量变化不改写已保存费用。
CREATE TABLE "air_weight_confirmations" (
 "id" uuid NOT NULL,
 "created_at" timestamptz NOT NULL,
 "updated_at" timestamptz NOT NULL,
 "organization_id" uuid NOT NULL,
 "scope" varchar NOT NULL,
 "target_id" uuid NOT NULL,
 "source" varchar NOT NULL DEFAULT 'ACTUAL',
 "version" bigint NOT NULL,
 "input_fingerprint" varchar(64) NOT NULL,
 "snapshot" jsonb NOT NULL,
 "confirmed_weight_kg" numeric(18,4) NOT NULL,
 "reason" varchar(500) NOT NULL DEFAULT '',
 "confirmed_by" uuid NOT NULL,
 PRIMARY KEY ("id"),
 CONSTRAINT "air_weight_confirmation_scope_check" CHECK ("scope" IN ('ORDER','MASTER')),
 CONSTRAINT "air_weight_confirmation_source_check" CHECK ("source" IN ('PLANNED','ACTUAL')),
 CONSTRAINT "air_weight_confirmation_positive_check" CHECK ("confirmed_weight_kg">0 AND "version">0)
);
CREATE INDEX "airweightconfirmation_updated_at" ON "air_weight_confirmations" ("updated_at");
CREATE UNIQUE INDEX "airweightconfirmation_org_scope_target_source_version" ON "air_weight_confirmations" ("organization_id","scope","target_id","source","version");
CREATE INDEX "airweightconfirmation_org_scope_target_source_created_at" ON "air_weight_confirmations" ("organization_id","scope","target_id","source","created_at");
ALTER TABLE "order_fees" ADD COLUMN "quantity_source" jsonb NULL;
