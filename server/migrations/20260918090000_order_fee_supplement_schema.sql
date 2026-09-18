-- 锁单后费用补录申请与提成复算快照 Schema：
-- 1) 新增 order_fee_supplement_requests：保存补录申请的不可变应付费用快照与
--    提交当时成立的锁依据，含组织级幂等唯一约束与锁依据完整性 CHECK；
-- 2) order_fees 增加一对一补录申请来源关联（NO ACTION + 唯一索引）；
-- 3) finance_commission_adjustments 扩展 LOCKED_FEE_SUPPLEMENT 来源，CHECK 强制
--    该来源与补录申请关联双向对应，唯一索引保证同一原提成订单行对同一申请
--    至多生成一条冲减建议；
-- 4) finance_commission_lines 增加历史总应收/总应付复算快照字段；存量行保持
--    全空，由后续回填迁移从原计算快照形成时点的不可变事实确定性还原。

-- 1. 补录申请表：除 status、version 与 decided_* 外全部字段创建后不可变。
CREATE TABLE "order_fee_supplement_requests" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "lock_basis" character varying(9) NOT NULL,
  "business_lock_generation" bigint NULL,
  "financial_lock_evidence_version" character varying(64) NULL,
  "financial_lock_evidence_hash" character varying(64) NULL,
  "financial_lock_net_amount_snapshot" numeric(28,8) NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "direction" character varying(10) NOT NULL,
  "fee_setting_id" uuid NULL,
  "fee_code" character varying(30) NOT NULL,
  "fee_name" character varying(80) NOT NULL,
  "fee_name_en" character varying(128) NULL,
  "settlement_party_id" uuid NOT NULL,
  "billing_unit_id" uuid NULL,
  "billing_unit" character varying(32) NOT NULL,
  "tax_rate" numeric(5,2) NULL,
  "taxable_service_name" character varying(128) NULL,
  "quantity" numeric(18,4) NOT NULL,
  "unit_price" numeric(18,4) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "tax_inclusive" boolean NOT NULL DEFAULT true,
  "net_amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "exchange_rate" numeric(18,8) NOT NULL,
  "exchange_rate_source" character varying(19) NOT NULL,
  "exchange_rate_date" character varying(10) NOT NULL,
  "exchange_rate_setting_id" uuid NULL,
  "base_currency" character varying(3) NOT NULL,
  "base_currency_amount" numeric(28,8) NOT NULL,
  "expense_date" character varying(10) NOT NULL,
  "note" character varying(500) NULL,
  "reason" character varying(500) NOT NULL,
  "requested_by" uuid NOT NULL,
  "requested_at" timestamp with time zone NOT NULL,
  "status" character varying(9) NOT NULL DEFAULT 'PENDING',
  "version" bigint NOT NULL DEFAULT 1,
  "decided_by" uuid NULL,
  "decided_at" timestamp with time zone NULL,
  "decision_reason" character varying(500) NULL,
  PRIMARY KEY ("id"),
  -- 锁依据完整性：BUSINESS 仅保存大于零的业务锁代次；FINANCIAL 仅保存非空的
  -- 财务证据版本、证据哈希与大于零的净额快照；BOTH 同时保存两组依据。锁类型
  -- 与依据均由服务端在 Order 行锁内判定固化，不接受客户端覆盖。
  CONSTRAINT "order_fee_supplement_requests_lock_basis_check"
    CHECK (
      (
        "lock_basis" = 'BUSINESS'
        AND "business_lock_generation" IS NOT NULL
        AND "business_lock_generation" > 0
        AND "financial_lock_evidence_version" IS NULL
        AND "financial_lock_evidence_hash" IS NULL
        AND "financial_lock_net_amount_snapshot" IS NULL
      )
      OR (
        "lock_basis" = 'FINANCIAL'
        AND "business_lock_generation" IS NULL
        AND "financial_lock_evidence_version" IS NOT NULL
        AND "financial_lock_evidence_hash" IS NOT NULL
        AND "financial_lock_net_amount_snapshot" IS NOT NULL
        AND "financial_lock_net_amount_snapshot" > 0
      )
      OR (
        "lock_basis" = 'BOTH'
        AND "business_lock_generation" IS NOT NULL
        AND "business_lock_generation" > 0
        AND "financial_lock_evidence_version" IS NOT NULL
        AND "financial_lock_evidence_hash" IS NOT NULL
        AND "financial_lock_net_amount_snapshot" IS NOT NULL
        AND "financial_lock_net_amount_snapshot" > 0
      )
    ),
  -- 申请状态取值；PENDING 只能进入一个终态由状态机与仓储校验保证。
  CONSTRAINT "order_fee_supplement_requests_status_check"
    CHECK ("status" IN ('PENDING', 'APPROVED', 'REJECTED', 'WITHDRAWN')),
  CONSTRAINT "order_fee_supplement_requests_organizations_order_fee_supplement_requests"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_fee_supplement_requests_orders_fee_supplement_requests"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  -- 发起人是永久审计事实，删除用户不得清空申请记录。
  CONSTRAINT "order_fee_supplement_requests_users_requested_order_fee_supplement_requests"
    FOREIGN KEY ("requested_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_fee_supplement_requests_users_decided_order_fee_supplement_requests"
    FOREIGN KEY ("decided_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

-- 同一组织幂等键唯一：同键同指纹语义重放返回原申请，同键不同指纹稳定冲突。
CREATE UNIQUE INDEX "orderfeesupplementrequest_organization_id_idempotency_key"
  ON "order_fee_supplement_requests" ("organization_id", "idempotency_key");

CREATE INDEX "orderfeesupplementrequest_organization_id_order_id"
  ON "order_fee_supplement_requests" ("organization_id", "order_id");

CREATE INDEX "orderfeesupplementrequest_order_id_status"
  ON "order_fee_supplement_requests" ("order_id", "status");

CREATE INDEX "orderfeesupplementrequest_updated_at"
  ON "order_fee_supplement_requests" ("updated_at");

-- 2. 订单费用增加补录申请来源：一条申请至多生成一条费用；外键 NO ACTION，
-- 删除申请不得造成孤儿费用。唯一索引允许多行 NULL。
ALTER TABLE "order_fees"
  ADD COLUMN "supplement_request_id" uuid NULL;

ALTER TABLE "order_fees"
  ADD CONSTRAINT "order_fees_order_fee_supplement_requests_fees"
    FOREIGN KEY ("supplement_request_id")
    REFERENCES "order_fee_supplement_requests" ("id") ON DELETE NO ACTION;

CREATE UNIQUE INDEX "orderfee_supplement_request_id"
  ON "order_fees" ("supplement_request_id");

-- 3. 提成调整扩展锁后费用补录来源：外键 NO ACTION 禁止删除申请造成孤儿调整。
ALTER TABLE "finance_commission_adjustments"
  ADD COLUMN "source_fee_supplement_request_id" uuid NULL;

ALTER TABLE "finance_commission_adjustments"
  ADD CONSTRAINT "finance_commission_adjustments_order_fee_supplement_requests_commission_adjustments"
    FOREIGN KEY ("source_fee_supplement_request_id")
    REFERENCES "order_fee_supplement_requests" ("id") ON DELETE NO ACTION;

-- 来源类型新增 LOCKED_FEE_SUPPLEMENT；既有取值不受影响。
ALTER TABLE "finance_commission_adjustments"
  DROP CONSTRAINT IF EXISTS "commission_adjustment_source_type_check";

ALTER TABLE "finance_commission_adjustments"
  ADD CONSTRAINT "commission_adjustment_source_type_check"
    CHECK ("source_type" IN ('MANUAL', 'VERIFICATION_REVERSAL', 'NETTING_REVERSAL', 'LOCKED_FEE_SUPPLEMENT'));

-- 来源关联互斥：仅 LOCKED_FEE_SUPPLEMENT 允许且必须携带补录申请关联，其他
-- 来源不得携带，防止来源错配绕过来源唯一键。
ALTER TABLE "finance_commission_adjustments"
  ADD CONSTRAINT "commission_adjustment_source_supplement_check"
    CHECK (
      (
        "source_type" = 'LOCKED_FEE_SUPPLEMENT'
        AND "source_fee_supplement_request_id" IS NOT NULL
      )
      OR (
        "source_type" <> 'LOCKED_FEE_SUPPLEMENT'
        AND "source_fee_supplement_request_id" IS NULL
      )
    );

-- 锁后补录建议去重：同一原提成订单行对同一补录申请至多一条调整。
CREATE UNIQUE INDEX "financecommissionadjustment_commission_id_order_id_source_type_source_fee_supplement_request_id"
  ON "finance_commission_adjustments" ("commission_id", "order_id", "source_type", "source_fee_supplement_request_id");

-- 4. 提成行历史复算快照：全部可空；存量行在回填迁移执行前保持全空。
ALTER TABLE "finance_commission_lines"
  ADD COLUMN "total_receivable_snapshot" numeric(28,8) NULL,
  ADD COLUMN "total_payable_snapshot" numeric(28,8) NULL,
  ADD COLUMN "snapshot_status" character varying(11) NULL,
  ADD COLUMN "snapshot_source" character varying(8) NULL,
  ADD COLUMN "snapshot_backfill_version" character varying(64) NULL,
  ADD COLUMN "snapshot_evidence_hash" character varying(64) NULL,
  ADD COLUMN "snapshot_unavailable_reason_code" character varying(64) NULL;

-- 复算快照一致性：READY 必须有完整历史总应收/总应付分母且不得携带不可用
-- 原因码；READY+MIGRATED 必须有回填算法版本与证据哈希（READY+NATIVE 不得
-- 伪填）；UNAVAILABLE 不得伪填估算分母且必须有稳定原因码。
ALTER TABLE "finance_commission_lines"
  ADD CONSTRAINT "finance_commission_lines_snapshot_consistency_check"
    CHECK (
      (
        "snapshot_status" IS NULL
        AND "total_receivable_snapshot" IS NULL
        AND "total_payable_snapshot" IS NULL
        AND "snapshot_source" IS NULL
        AND "snapshot_backfill_version" IS NULL
        AND "snapshot_evidence_hash" IS NULL
        AND "snapshot_unavailable_reason_code" IS NULL
      )
      OR (
        "snapshot_status" = 'READY'
        AND "total_receivable_snapshot" IS NOT NULL
        AND "total_payable_snapshot" IS NOT NULL
        AND "snapshot_unavailable_reason_code" IS NULL
        AND "snapshot_source" IS NOT NULL
        AND (
          (
            "snapshot_source" = 'NATIVE'
            AND "snapshot_backfill_version" IS NULL
            AND "snapshot_evidence_hash" IS NULL
          )
          OR (
            "snapshot_source" = 'MIGRATED'
            AND "snapshot_backfill_version" IS NOT NULL
            AND "snapshot_evidence_hash" IS NOT NULL
          )
        )
      )
      OR (
        "snapshot_status" = 'UNAVAILABLE'
        AND "total_receivable_snapshot" IS NULL
        AND "total_payable_snapshot" IS NULL
        AND "snapshot_source" IS NULL
        AND "snapshot_backfill_version" IS NULL
        AND "snapshot_evidence_hash" IS NULL
        AND "snapshot_unavailable_reason_code" IS NOT NULL
      )
    );
