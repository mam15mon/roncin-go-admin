-- 账单免人工确认：新建账单直接写为 CONFIRMED，存量草稿账单在此统一转为已确认。
-- 确认时间回填为账单创建时间；确认人回填为创建人：批量账单取所属批次的 created_by，
-- 单张账单取该账单最早一条成功的 finance.bill.create 审计操作者。
-- 两者都取不到时迁移显式失败并回滚，由人工核查处理，不静默填充确认人。
-- 状态变化同时递增 version，使持有旧版本的页面在后续写入时得到并发冲突。
DO $$
DECLARE
  unresolved integer;
BEGIN
  SELECT count(*) INTO unresolved
  FROM "finance_bills" b
  WHERE b."status" = 'DRAFT'
    AND b."batch_id" IS NULL
    AND NOT EXISTS (
      SELECT 1 FROM "audit_logs" a
      WHERE a."action" = 'finance.bill.create'
        AND a."result" = 'success'
        AND a."resource_type" = 'finance_bill'
        AND a."resource_id" = b."id"::text
        AND a."user_id" IS NOT NULL
    );
  IF unresolved > 0 THEN
    RAISE EXCEPTION '存在 % 张草稿账单无法确定创建人，请人工核查后再执行迁移', unresolved;
  END IF;
END $$;

UPDATE "finance_bills" b
SET "status" = 'CONFIRMED',
    "confirmed_at" = b."created_at",
    "confirmed_by" = COALESCE(
      (SELECT bb."created_by" FROM "finance_bill_batches" bb WHERE bb."id" = b."batch_id"),
      (
        SELECT a."user_id" FROM "audit_logs" a
        WHERE a."action" = 'finance.bill.create'
          AND a."result" = 'success'
          AND a."resource_type" = 'finance_bill'
          AND a."resource_id" = b."id"::text
          AND a."user_id" IS NOT NULL
        ORDER BY a."created_at", a."id"
        LIMIT 1
      )
    ),
    "version" = b."version" + 1,
    "updated_at" = CURRENT_TIMESTAMP
WHERE b."status" = 'DRAFT';
