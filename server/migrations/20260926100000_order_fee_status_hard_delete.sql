-- 费用状态彻底退役（09-26-fee-hard-delete）：
-- 1. 删列前校验存量 CANCELLED 费用的事务事实：存在有效账单关联（活动账单行且
--    所属账单未取消）或关联冲减调整曾确认/已扣回/进入终态时整体失败，禁止通过
--    级联删除金融历史解决冲突，不自动纠正。
-- 2. 迁移归档审计：为将物理删除的 CANCELLED 费用写入迁移来源审计快照（费用关键
--    内容与旧撤销元数据；user_id 置空，不伪造操作人）。
-- 3. 物理删除已作废费用：账单行来源外键 ON DELETE SET NULL 保留历史行快照；
--    费用标签关联按实际外键级联清理；补录申请与调整事实保留。
-- 4. 删除 status/cancelled_* 列、状态 CHECK、cancellation_consistency CHECK、
--    cancelled_by 外键与 (order_id,status,created_at) 索引；是否已建账此后统一由
--    有效账单关联（活动账单行且所属账单未取消，含 DRAFT）表达。
-- 本文件由迁移执行器在单一事务内提交。结构严格匹配预期，不符立即失败；
-- 唯二例外是开发库相对迁移链的两处既有漂移（08-26 文件补入 consistency CHECK
-- 晚于开发库应用、cancelled_by 外键在 Ent 自动迁移期命名不同），对这两处使用
-- IF EXISTS 双命名兼容，正式链与开发库均可重放。

DO $$
DECLARE
    occupied_count bigint;
    adjustment_conflict_count bigint;
    cancelled_count bigint;
BEGIN
    SELECT count(*) INTO occupied_count
    FROM "order_fees" f
    JOIN "finance_bill_lines" l ON l.order_fee_id = f.id AND l.active = TRUE
    JOIN "finance_bills" b ON b.id = l.bill_id
    WHERE f."status" = 'CANCELLED' AND b."status" <> 'CANCELLED';
    IF occupied_count > 0 THEN
        RAISE EXCEPTION '存量已作废费用存在有效账单关联（活动账单行且所属账单未取消）：%，需先经财务链路处置后再迁移', occupied_count;
    END IF;

    SELECT count(*) INTO adjustment_conflict_count
    FROM "order_fees" f
    JOIN "finance_commission_adjustments" a
      ON a.source_fee_supplement_request_id = f.supplement_request_id
    WHERE f."status" = 'CANCELLED'
      AND f.supplement_request_id IS NOT NULL
      AND (a."status" NOT IN ('DRAFT', 'CANCELLED') OR a.confirmed_at IS NOT NULL OR a.paid_at IS NOT NULL);
    IF adjustment_conflict_count > 0 THEN
        RAISE EXCEPTION '存量已作废费用存在曾确认或已扣回的关联冲减调整：%，需人工核处后再迁移', adjustment_conflict_count;
    END IF;

    SELECT count(*) INTO cancelled_count FROM "order_fees" WHERE "status" = 'CANCELLED';

    INSERT INTO "audit_logs" ("id", "created_at", "updated_at", "organization_id", "user_id",
        "action", "resource_type", "resource_id", "result", "details")
    SELECT gen_random_uuid(), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, o.organization_id, NULL,
        'order.fee.migration_hard_delete', 'order_fee', f.id::text, 'success',
        jsonb_build_object(
            'source', 'migration:20260926100000_order_fee_status_hard_delete',
            'fee.id', f.id::text,
            'fee.code', f.fee_code,
            'fee.direction', f.direction,
            'fee.amount', f.total_amount,
            'fee.currency', f.currency,
            'fee.settlement_party_id', f.settlement_party_id::text,
            'fee.version', f.version::text,
            'fee.supplement_request_id', COALESCE(f.supplement_request_id::text, ''),
            'cancelled_at', COALESCE(f.cancelled_at::text, ''),
            'cancelled_by', COALESCE(f.cancelled_by::text, ''),
            'cancellation_reason', COALESCE(f.cancellation_reason, '')
        )
    FROM "order_fees" f
    LEFT JOIN "orders" o ON o.id = f.order_id
    WHERE f."status" = 'CANCELLED';

    DELETE FROM "order_fees" WHERE "status" = 'CANCELLED';

    RAISE NOTICE '已写入迁移归档审计并物理删除存量已作废费用 % 行', cancelled_count;
END
$$;

ALTER TABLE "order_fees" DROP CONSTRAINT IF EXISTS "order_fees_cancellation_consistency";
ALTER TABLE "order_fees" DROP CONSTRAINT "order_fees_status_check";
-- cancelled_by 外键在正式链与历史 Ent 自动迁移环境下的命名不同，两者择一存在
ALTER TABLE "order_fees" DROP CONSTRAINT IF EXISTS "order_fees_users_cancelled_by";
ALTER TABLE "order_fees" DROP CONSTRAINT IF EXISTS "order_fees_users_cancelled_order_fees";
DROP INDEX "orderfee_order_id_status_created_at";
ALTER TABLE "order_fees" DROP COLUMN "cancelled_at";
ALTER TABLE "order_fees" DROP COLUMN "cancelled_by";
ALTER TABLE "order_fees" DROP COLUMN "cancellation_reason";
ALTER TABLE "order_fees" DROP COLUMN "status";
