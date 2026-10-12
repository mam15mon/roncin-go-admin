-- 费用审核及净额发放：由 Ent Schema diff 生成新增结构，已有同名 CHECK 由 Schema 注解重新生成。
-- 本迁移只添加审核与执行事实，存量订单未提交；不伪造历史批准或发放金额。
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM order_fee_supplement_requests WHERE status = 'PENDING') THEN
  RAISE EXCEPTION '费用审核迁移前请按原流程处理待审补录申请（order_fee_supplement_requests.status=PENDING），不自动撤回或批准';
 END IF;
 IF EXISTS (SELECT 1 FROM finance_commission_adjustments WHERE source_type='LOCKED_FEE_SUPPLEMENT' AND status='DRAFT') THEN
  RAISE EXCEPTION '费用审核迁移前请按原流程处理补录冲减草稿，不自动确认或忽略';
 END IF;
 IF EXISTS (SELECT 1 FROM finance_commission_adjustments a JOIN finance_commissions c ON c.id=a.commission_id WHERE a.status='PAID' AND c.status<>'PAID') THEN
  RAISE EXCEPTION '费用审核迁移前请核对父提成尚未发放但调整已执行的记录，不自动伪造付款事实';
 END IF;
END $$;

-- modify "finance_commission_adjustments" table
ALTER TABLE "finance_commission_adjustments" ADD CONSTRAINT "commission_adjustment_execution_check" CHECK ((execution_method IS NULL AND parent_payment_id IS NULL) OR (execution_method = 'OFFSET_WITH_PARENT' AND parent_payment_id = commission_id) OR (execution_method = 'SEPARATE' AND parent_payment_id IS NULL)), ADD COLUMN "execution_method" character varying NULL, ADD COLUMN "parent_payment_id" uuid NULL;
-- modify "finance_commissions" table
ALTER TABLE "finance_commissions" ADD COLUMN "paid_amount" numeric(28,8) NULL, ADD COLUMN "payment_key" character varying NULL, ADD COLUMN "payment_fingerprint" character varying NULL, ADD COLUMN "payment_snapshot" character varying NULL;
-- modify "order_fee_supplement_requests" table
ALTER TABLE "order_fee_supplement_requests" ADD COLUMN "approved_fee_review_event_id" uuid NULL, ADD COLUMN "approved_fee_content_version" bigint NULL;
-- create index "order_fee_supplement_one_pending" to table: "order_fee_supplement_requests"
CREATE UNIQUE INDEX "order_fee_supplement_one_pending" ON "order_fee_supplement_requests" ("order_id") WHERE status = 'PENDING';
-- modify "orders" table
ALTER TABLE "orders" ADD CONSTRAINT "orders_fee_review_status_check" CHECK (fee_review_status IN ('UNSUBMITTED', 'PENDING', 'APPROVED', 'REJECTED', 'REVOKED')), ADD COLUMN "fee_review_status" character varying NOT NULL DEFAULT 'UNSUBMITTED', ADD COLUMN "fee_content_version" bigint NOT NULL DEFAULT 1, ADD COLUMN "fee_review_version" bigint NOT NULL DEFAULT 1, ADD COLUMN "current_fee_review_event_id" uuid NULL, ADD COLUMN "approved_fee_review_event_id" uuid NULL;
-- create index "order_organization_id_fee_review_status" to table: "orders"
CREATE INDEX "order_organization_id_fee_review_status" ON "orders" ("organization_id", "fee_review_status");
-- create "order_fee_review_events" table
CREATE TABLE "order_fee_review_events" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "organization_id" uuid NOT NULL, "action" character varying NOT NULL, "submission_id" uuid NULL, "actor_id" uuid NOT NULL, "actor_name" character varying NOT NULL DEFAULT '', "authorization_snapshot" jsonb NOT NULL, "comment" character varying NOT NULL DEFAULT '', "content_version" bigint NOT NULL, "content_fingerprint" character varying NOT NULL, "snapshot" jsonb NOT NULL, "supplement_request_id" uuid NULL, "idempotency_key" character varying NOT NULL, "request_fingerprint" character varying NOT NULL, "order_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "order_fee_review_events_orders_fee_review_events" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION, CONSTRAINT "order_fee_review_events_action_check" CHECK (action IN ('SUBMIT','WITHDRAW','APPROVE','REJECT','REVOKE','SUPPLEMENT_APPROVE')));
-- create index "orderfeereviewevent_updated_at" to table: "order_fee_review_events"
CREATE INDEX "orderfeereviewevent_updated_at" ON "order_fee_review_events" ("updated_at");
-- create index "orderfeereviewevent_order_id_idempotency_key" to table: "order_fee_review_events"
CREATE UNIQUE INDEX "orderfeereviewevent_order_id_idempotency_key" ON "order_fee_review_events" ("order_id", "idempotency_key");
-- create index "orderfeereviewevent_organization_id_order_id_created_at" to table: "order_fee_review_events"
CREATE INDEX "orderfeereviewevent_organization_id_order_id_created_at" ON "order_fee_review_events" ("organization_id", "order_id", "created_at");

-- Ent 当前 diff 不重建已有同名 CHECK，直接由同源注解生成。
ALTER TABLE "notification_deliveries" DROP CONSTRAINT "notification_deliveries_template_check";
ALTER TABLE "notification_deliveries" ADD CONSTRAINT "notification_deliveries_template_check" CHECK (template IN ('ORDER_PERSONNEL_ASSIGNED', 'USER_AUTHORIZED', 'DINGTALK_REGISTRATION_PENDING', 'DINGTALK_REGISTRATION_REJECTED', 'DINGTALK_INVITATION_ACTIVATED', 'EXCHANGE_RATE_WEEKLY_REMINDER', 'FEE_SUPPLEMENT_APPROVAL_PENDING', 'COMMISSION_DECREASE_SUGGESTED', 'FEE_REVIEW_PENDING', 'FEE_REVIEW_RESULT'));

-- Ent 当前 diff 不重建已有同名 CHECK，直接由同源注解生成。
ALTER TABLE "order_fee_supplement_requests" DROP CONSTRAINT "order_fee_supplement_requests_lock_basis_check";
ALTER TABLE "order_fee_supplement_requests" ADD CONSTRAINT "order_fee_supplement_requests_lock_basis_check" CHECK ((lock_basis = 'APPROVAL' AND approved_fee_review_event_id IS NOT NULL AND approved_fee_content_version > 0) OR (lock_basis = 'BUSINESS' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NULL AND financial_lock_evidence_hash IS NULL AND financial_lock_net_amount_snapshot IS NULL) OR (lock_basis = 'FINANCIAL' AND business_lock_generation IS NULL AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0) OR (lock_basis = 'BOTH' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0));
