-- 汇率是全公司共享主数据，统一归根总部；分公司及部门不保存独立副本。
DO $$
DECLARE
  headquarters_count integer;
  headquarters_id uuid;
  has_conflict boolean;
BEGIN
  SELECT count(*), min("id"::text)::uuid
  INTO headquarters_count, headquarters_id
  FROM "organizations"
  WHERE "kind" = 'headquarters' AND "parent_id" IS NULL;

  IF headquarters_count <> 1 THEN
    RAISE EXCEPTION '全公司共享汇率要求且仅允许一个根总部，当前数量：%', headquarters_count;
  END IF;

  SELECT EXISTS (
    SELECT 1
    FROM "exchange_rate_settings"
    GROUP BY "rate_type", "from_currency", "to_currency", "time_standard", "effective_from"
    HAVING count(*) > 1
  ) INTO has_conflict;

  IF has_conflict THEN
    RAISE EXCEPTION '已有分公司汇率迁移到总部后存在唯一键冲突，请先明确保留记录';
  END IF;

  UPDATE "exchange_rate_settings"
  SET "organization_id" = headquarters_id,
      "updated_at" = CURRENT_TIMESTAMP
  WHERE "organization_id" <> headquarters_id;
END $$;
