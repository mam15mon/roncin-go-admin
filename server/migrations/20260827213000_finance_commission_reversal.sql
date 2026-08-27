-- 标记提成调整来源，使核销撤销产生的待追回冲减可追溯且具备数据库幂等兜底。
ALTER TABLE "finance_commission_adjustments"
  ADD COLUMN "source_type" varchar NOT NULL DEFAULT 'MANUAL',
  ADD COLUMN "source_verification_id" uuid REFERENCES "finance_verifications"("id"),
  ADD CONSTRAINT "commission_adjustment_source_type_check"
    CHECK("source_type" IN('MANUAL','VERIFICATION_REVERSAL'));

CREATE UNIQUE INDEX "financecommissionadjustment_reversal_source"
  ON "finance_commission_adjustments"(
    "commission_id","order_id","source_type","source_verification_id"
  );
