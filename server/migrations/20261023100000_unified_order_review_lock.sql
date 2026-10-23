-- 统一人工审核锁。旧锁及独立审批不能推定为新审核事实，先明确阻断需人工处置的对象。
DO $$
DECLARE affected text;
BEGIN
  SELECT string_agg(order_no, '、' ORDER BY order_no) INTO affected
  FROM orders WHERE locked_at IS NOT NULL OR fee_review_status IN ('PENDING', 'APPROVED');
  IF affected IS NOT NULL THEN
    RAISE EXCEPTION '统一订单审核锁迁移被阻断：以下订单仍有旧锁定或独立费用审批，请先明确人工审核及解锁处置：%', affected;
  END IF;
  SELECT string_agg(id::text, '、' ORDER BY id) INTO affected
  FROM order_fee_supplement_requests WHERE status = 'PENDING';
  IF affected IS NOT NULL THEN
    RAISE EXCEPTION '统一订单审核锁迁移被阻断：旧待审补录需先处理：%', affected;
  END IF;
  SELECT string_agg(id::text, '、' ORDER BY id) INTO affected
  FROM order_fee_correction_requests WHERE status = 'PENDING';
  IF affected IS NOT NULL THEN
    RAISE EXCEPTION '统一订单审核锁迁移被阻断：旧待审费用更正缺少人工审核锁依据，需先处理：%', affected;
  END IF;
END $$;

ALTER TABLE order_lock_records
  ADD COLUMN fee_snapshot jsonb NULL,
  ADD COLUMN fee_content_version bigint NULL,
  ADD COLUMN fee_content_fingerprint character varying(64) NULL,
  ADD COLUMN review_comment character varying NULL;
ALTER TABLE order_fee_correction_requests
  ALTER COLUMN approved_fee_review_event_id DROP NOT NULL,
  ADD COLUMN lock_record_id uuid NULL,
  ADD COLUMN lock_generation bigint NULL;
-- 已结束申请保留原审批历史；所有新申请由业务入口强制写入人工锁定记录和代次。
ALTER TABLE order_fee_correction_requests ADD CONSTRAINT order_fee_correction_lock_basis_check
  CHECK ((lock_record_id IS NULL AND lock_generation IS NULL) OR (lock_record_id IS NOT NULL AND lock_generation > 0));
ALTER TABLE order_lock_records ADD CONSTRAINT order_lock_records_fee_review_snapshot_check
  CHECK ((fee_snapshot IS NULL AND fee_content_version IS NULL AND fee_content_fingerprint IS NULL AND review_comment IS NULL)
    OR (fee_snapshot IS NOT NULL AND fee_content_version IS NOT NULL AND fee_content_fingerprint IS NOT NULL AND length(fee_content_fingerprint) = 64 AND review_comment IS NOT NULL));

-- 普通费用变更的提成差额使用独立来源，保持原核销/对冲关联约束。
ALTER TABLE finance_commission_adjustments DROP CONSTRAINT commission_adjustment_source_type_check;
ALTER TABLE finance_commission_adjustments ADD CONSTRAINT commission_adjustment_source_type_check
  CHECK (source_type IN ('MANUAL','VERIFICATION_REVERSAL','NETTING_REVERSAL','LOCKED_FEE_SUPPLEMENT','FEE_CORRECTION','ECONOMIC_REALLOCATION','FEE_MUTATION'));
ALTER TABLE finance_commission_adjustments ADD CONSTRAINT commission_adjustment_source_fee_mutation_check
  CHECK (source_type <> 'FEE_MUTATION' OR (source_verification_id IS NULL AND source_netting_id IS NULL));

DROP INDEX order_organization_id_fee_review_status;

ALTER TABLE order_fee_correction_requests ADD CONSTRAINT order_fee_correction_requests_order_lock_records_fee_correction_requests
  FOREIGN KEY (lock_record_id) REFERENCES order_lock_records(id) ON DELETE NO ACTION;
