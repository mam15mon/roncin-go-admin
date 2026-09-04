-- 为审批创建增加持久化的发送中状态，确保 Worker 崩溃或租约过期后不会盲目重建外部实例。
ALTER TABLE "ding_talk_approval_dispatches"
  DROP CONSTRAINT "ding_talk_approval_dispatches_dispatch_status_check",
  ADD CONSTRAINT "ding_talk_approval_dispatches_dispatch_status_check"
  CHECK ("dispatch_status" IN ('PENDING', 'SENDING', 'DISPATCHED', 'FAILED', 'UNKNOWN'));

ALTER TABLE "ding_talk_approval_inbox_events"
  ADD COLUMN "attempts" bigint NOT NULL DEFAULT 0,
  ADD COLUMN "next_run_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN "processing_token" character varying(64),
  ADD COLUMN "processing_expires_at" timestamp with time zone,
  DROP CONSTRAINT "ding_talk_approval_inbox_events_status_check",
  ADD CONSTRAINT "ding_talk_approval_inbox_events_status_check"
  CHECK ("status" IN ('RECEIVED', 'PROCESSING', 'PROCESSED', 'IGNORED', 'FAILED')),
  ADD CONSTRAINT "ding_talk_approval_inbox_events_attempts_check"
  CHECK ("attempts" >= 0);

DROP INDEX "dingtalkapprovalinboxevent_status_received_at";
CREATE INDEX "dingtalkapprovalinboxevent_status_next_run_at_received_at"
  ON "ding_talk_approval_inbox_events" ("status", "next_run_at", "received_at");
