-- 为锁定周期和解锁请求保存权威业务类型快照；先保持可空以完成历史回填。
ALTER TABLE "order_lock_records"
  ADD COLUMN "business_type" character varying;

ALTER TABLE "order_unlock_requests"
  ADD COLUMN "business_type" character varying;

-- 锁定周期从关联订单回填，解锁请求再从其关联的锁定周期回填。
UPDATE "order_lock_records" AS lock_record
SET "business_type" = orders."business_type"
FROM "orders"
WHERE orders."id" = lock_record."order_id";

UPDATE "order_unlock_requests" AS unlock_request
SET "business_type" = lock_record."business_type"
FROM "order_lock_records" AS lock_record
WHERE lock_record."id" = unlock_request."lock_record_id";

-- 既有锁事实只能来自原有 SE 锁流程；发现孤儿记录或类型异常时停止迁移。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM "order_lock_records"
    WHERE "business_type" IS NULL OR "business_type" <> 'SE'
  ) THEN
    RAISE EXCEPTION '全业务订单锁迁移已停止：既有锁定周期缺少关联订单或不是 SE';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM "order_unlock_requests"
    WHERE "business_type" IS NULL OR "business_type" <> 'SE'
  ) THEN
    RAISE EXCEPTION '全业务订单锁迁移已停止：既有解锁请求缺少关联锁定周期或不是 SE';
  END IF;
END $$;

ALTER TABLE "order_lock_records"
  ALTER COLUMN "business_type" SET NOT NULL,
  ALTER COLUMN "master_bill_id" DROP NOT NULL,
  ALTER COLUMN "master_bill_version_id" DROP NOT NULL;

ALTER TABLE "order_unlock_requests"
  ALTER COLUMN "business_type" SET NOT NULL;

-- SE 必须保留完整 MBL 快照引用；其他类型不得携带 SE 专属引用。
ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_business_type_document_refs_check" CHECK (
    (business_type = 'SE' AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL)
    OR
    (business_type IN ('SI', 'AE', 'AI', 'LAND', 'RAIL') AND master_bill_id IS NULL AND master_bill_version_id IS NULL)
  );
