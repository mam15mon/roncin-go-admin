-- 订单锁定来源与自动触发审计字段：支持人工锁定（MANUAL）与应收结清驱动的
-- 系统自动锁定（AUTO_SETTLEMENT）。存量锁定事实统一标记为人工锁定。

-- 1. orders 增加锁定来源与自动触发审计字段
ALTER TABLE "orders"
  ADD COLUMN "lock_source" text,
  ADD COLUMN "auto_lock_trigger_type" text,
  ADD COLUMN "auto_lock_trigger_resource_id" uuid,
  ADD COLUMN "auto_lock_triggered_by" uuid;

-- 存量锁定事实统一标记 MANUAL
UPDATE "orders"
  SET "lock_source" = 'MANUAL'
  WHERE "locked_at" IS NOT NULL AND "lock_source" IS NULL;

-- 锁定来源一致性：未锁定时全部锁字段为空；MANUAL 必须有 locked_by 且触发
-- 字段为空；AUTO_SETTLEMENT 必须 locked_by 为空且触发字段完整。
ALTER TABLE "orders"
  ADD CONSTRAINT "orders_lock_source_check" CHECK (
    (
      "locked_at" IS NULL
      AND "locked_by" IS NULL
      AND "lock_source" IS NULL
      AND "auto_lock_trigger_type" IS NULL
      AND "auto_lock_trigger_resource_id" IS NULL
      AND "auto_lock_triggered_by" IS NULL
    )
    OR (
      "locked_at" IS NOT NULL
      AND "lock_source" = 'MANUAL'
      AND "locked_by" IS NOT NULL
      AND "auto_lock_trigger_type" IS NULL
      AND "auto_lock_trigger_resource_id" IS NULL
      AND "auto_lock_triggered_by" IS NULL
    )
    OR (
      "locked_at" IS NOT NULL
      AND "lock_source" = 'AUTO_SETTLEMENT'
      AND "locked_by" IS NULL
      AND "auto_lock_trigger_type" IS NOT NULL
      AND "auto_lock_trigger_resource_id" IS NOT NULL
      AND "auto_lock_triggered_by" IS NOT NULL
    )
  );

-- 2. order_lock_records 增加锁定来源与触发审计字段
ALTER TABLE "order_lock_records"
  ADD COLUMN "lock_source" text,
  ADD COLUMN "trigger_type" text,
  ADD COLUMN "trigger_resource_id" uuid,
  ADD COLUMN "triggered_by" uuid;

-- 存量锁定记录统一标记 MANUAL
UPDATE "order_lock_records"
  SET "lock_source" = 'MANUAL'
  WHERE "lock_source" IS NULL;

ALTER TABLE "order_lock_records"
  ALTER COLUMN "lock_source" SET DEFAULT 'MANUAL',
  ALTER COLUMN "locked_by" DROP NOT NULL;

-- 锁定记录来源一致性：人工锁定必须有 locked_by 且触发字段为空；系统自动
-- 锁定必须无 locked_by 且触发类型、触发单据与触发操作人完整。
ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_lock_source_check" CHECK (
    (
      "lock_source" = 'MANUAL'
      AND "locked_by" IS NOT NULL
      AND "trigger_type" IS NULL
      AND "trigger_resource_id" IS NULL
      AND "triggered_by" IS NULL
    )
    OR (
      "lock_source" = 'AUTO_SETTLEMENT'
      AND "locked_by" IS NULL
      AND "trigger_type" IS NOT NULL
      AND "trigger_resource_id" IS NOT NULL
      AND "triggered_by" IS NOT NULL
    )
  );

-- 触发操作人外键与生成元数据同源（SET NULL：用户删除仅清空审计引用）
ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_users_auto_triggered_order_lock_records"
  FOREIGN KEY ("triggered_by") REFERENCES "users" ("id") ON DELETE SET NULL;
