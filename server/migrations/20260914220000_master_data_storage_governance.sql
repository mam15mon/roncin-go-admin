-- 主数据存储三型改造（阶段一）：治理边界物化为存储结构。
-- A 型（master_data_items / airlines / shipping_lines / 前缀表 / billing_units）：
--   物理删除 organization_id，业务码建立全局唯一索引。master_data_items 同
--   (kind, code) 重复行先重指逻辑引用后按「总部行 > 最早创建」保留一行删除；
--   airlines / shipping_lines / 前缀表 / billing_units 的同码多行为未处置冲突，
--   一律 fail-fast 中止迁移（design §8 第 2 步：冲突→报告并中止，禁止自动静默
--   合并或硬删除），人工处置后重跑。
-- B 型（ports / airports / exchange_rate_settings / fee_settings）：
--   organization_id 改可空，总部根组织行置 NULL 成为基线行；重建部分唯一索引：
--   NULL 基线行业务码全局唯一，org 行 (organization_id, 业务码) 唯一。
-- master_data_items kind 枚举 service_type → charge_category 就地更名；
-- fee_settings.service_type_id 更名 charge_category_id 并改为必填（先按费用名称
-- 关键词归类回填，无法归类的行报错中止，由人工裁定后重跑）。

-- ============ 1. master_data_items：kind 更名 + 去组织 ============

-- 1.0 清理重种子残留：开发期曾按组织重复播种，同一组织同码同时存在 charge_category
-- 与 service_type 行时，先保留 charge_category 行、删除 service_type 行，
-- 避免下一步更名触发 (organization_id, kind, code) 唯一冲突。
DELETE FROM "master_data_items" m
USING "master_data_items" k
WHERE m."kind" = 'service_type'
  AND k."kind" = 'charge_category'
  AND m."organization_id" = k."organization_id"
  AND m."code" = k."code";

-- 1.1 数据更名：service_type → charge_category
UPDATE "master_data_items" SET "kind" = 'charge_category' WHERE "kind" = 'service_type';

-- 1.1.1 kind 枚举检查约束同步更名：与新 Ent 枚举（master_data_item.go）对齐，
-- 移除 service_type 并补入 charge_category；开发库若因历史 AutoMigrate 缺失该
-- 约束则由 IF EXISTS 兜底直接补建。
ALTER TABLE "master_data_items" DROP CONSTRAINT IF EXISTS "master_data_items_kind_check";
ALTER TABLE "master_data_items" ADD CONSTRAINT "master_data_items_kind_check"
  CHECK ("kind" IN ('currency', 'country', 'region', 'container_spec', 'charge_category', 'cargo_category', 'abnormal_case'));

-- 1.2 重复行（同 kind+code）删除前，将全部逻辑引用重指到保留行。
-- 保留规则：总部根组织行优先，其次最早创建。
WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code")
    m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY
    m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id"
  FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "order_service_types" t SET "master_data_item_id" = r."kept_id"
FROM removed r WHERE t."master_data_item_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "order_cargo_categories" t SET "master_data_item_id" = r."kept_id"
FROM removed r WHERE t."master_data_item_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "order_abnormal_cases" t SET "abnormal_case_id" = r."kept_id"
FROM removed r WHERE t."abnormal_case_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "order_containers" t SET "container_spec_id" = r."kept_id"
FROM removed r WHERE t."container_spec_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "order_container_requests" t SET "container_spec_id" = r."kept_id"
FROM removed r WHERE t."container_spec_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "sea_shared_containers" t SET "container_spec_id" = r."kept_id"
FROM removed r WHERE t."container_spec_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "fee_settings" t SET "service_type_id" = r."kept_id"
FROM removed r WHERE t."service_type_id" = r."id";

WITH kept AS (
  SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
  FROM "master_data_items" m
  ORDER BY m."kind", m."code",
    (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
    m."created_at", m."id"
),
removed AS (
  SELECT m."id", k."kept_id" FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
)
UPDATE "fee_settings" t SET "abnormal_case_id" = r."kept_id"
FROM removed r WHERE t."abnormal_case_id" = r."id";

-- 1.3 删除重复主数据行并输出处置报告
DO $$
DECLARE
  removed_count INT;
BEGIN
  WITH kept AS (
    SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
    FROM "master_data_items" m
    ORDER BY m."kind", m."code",
      (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
      m."created_at", m."id"
  )
  SELECT count(*) INTO removed_count
  FROM "master_data_items" m
  JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id";

  WITH kept AS (
    SELECT DISTINCT ON (m."kind", m."code") m."kind", m."code", m."id" AS kept_id
    FROM "master_data_items" m
    ORDER BY m."kind", m."code",
      (m."organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters')) DESC,
      m."created_at", m."id"
  ),
  removed AS (
    SELECT m."id" FROM "master_data_items" m
    JOIN kept k ON k."kind" = m."kind" AND k."code" = m."code" AND k."kept_id" <> m."id"
  )
  DELETE FROM "master_data_items" USING removed WHERE "master_data_items"."id" = removed."id";

  RAISE NOTICE '[master_data_items] 同 (kind, code) 重复行按「总部行 > 最早创建」保留一行，删除 % 行', removed_count;
END $$;

-- 1.4 索引重建与删列（A 型：业务码全局唯一）
DROP INDEX "masterdataitem_organization_id_kind_code";
DROP INDEX "masterdataitem_organization_id_kind_enabled_sort_order";
DROP INDEX "masterdataitem_organization_id_kind_name";
CREATE UNIQUE INDEX "masterdataitem_kind_code" ON "master_data_items" ("kind", "code");
CREATE INDEX "masterdataitem_kind_enabled_sort_order" ON "master_data_items" ("kind", "enabled", "sort_order");
CREATE INDEX "masterdataitem_kind_name" ON "master_data_items" ("kind", "name");
ALTER TABLE "master_data_items" DROP COLUMN "organization_id";

-- ============ 2. airlines：A 型收口 ============

-- 2.1 Fail-fast 冲突检测（design §8 第 2 步）：同 IATA 多行属未处置冲突，
-- 产出报告并中止迁移，禁止自动静默合并或硬删除（处置需人工授权后重跑）。
DO $$
DECLARE
  conflict_count INT;
  sample_codes TEXT;
BEGIN
  SELECT count(*), COALESCE(string_agg("iata_code", ', ' ORDER BY "iata_code"), '')
    INTO conflict_count, sample_codes
  FROM (
    SELECT "iata_code" FROM "airlines" GROUP BY "iata_code" HAVING count(*) > 1
  ) duplicated;
  IF conflict_count > 0 THEN
    RAISE EXCEPTION '[airlines] % 个 IATA 码存在多行冲突（样例：%），fail-fast 中止迁移；请核对冲突报告并处置开发数据后重跑', conflict_count, sample_codes;
  END IF;
END $$;

DROP INDEX "airline_organization_id_iata_code";
DROP INDEX "airline_organization_id_icao_code";
DROP INDEX "airline_organization_id_awb_prefix";
DROP INDEX "airline_organization_id_enabled_sort_order";
CREATE UNIQUE INDEX "airline_iata_code" ON "airlines" ("iata_code");
CREATE UNIQUE INDEX "airline_icao_code" ON "airlines" ("icao_code");
CREATE UNIQUE INDEX "airline_awb_prefix" ON "airlines" ("awb_prefix");
CREATE INDEX "airline_enabled_sort_order" ON "airlines" ("enabled", "sort_order");
ALTER TABLE "airlines" DROP COLUMN "organization_id";

-- ============ 3. shipping_lines 与箱主前缀：A 型收口 ============

-- 3.1 同 SCAC 多行为未处置冲突：fail-fast 中止迁移（同 2.1）。
DO $$
DECLARE
  conflict_count INT;
  sample_codes TEXT;
BEGIN
  SELECT count(*), COALESCE(string_agg("scac_code", ', ' ORDER BY "scac_code"), '')
    INTO conflict_count, sample_codes
  FROM (
    SELECT "scac_code" FROM "shipping_lines" GROUP BY "scac_code" HAVING count(*) > 1
  ) duplicated;
  IF conflict_count > 0 THEN
    RAISE EXCEPTION '[shipping_lines] % 个 SCAC 码存在多行冲突（样例：%），fail-fast 中止迁移；请核对冲突报告并处置开发数据后重跑', conflict_count, sample_codes;
  END IF;
END $$;

-- 3.2 清理悬空的前缀行（船司行已在库中缺失的历史遗留）；前缀业务码全局唯一（BIC 箱主唯一）
DELETE FROM "shipping_line_container_prefixes"
WHERE "shipping_line_id" NOT IN (SELECT "id" FROM "shipping_lines");

-- 3.3 同前缀多行为未处置冲突：fail-fast 中止迁移（同 2.1）。
DO $$
DECLARE
  conflict_count INT;
  sample_codes TEXT;
BEGIN
  SELECT count(*), COALESCE(string_agg("prefix", ', ' ORDER BY "prefix"), '')
    INTO conflict_count, sample_codes
  FROM (
    SELECT "prefix" FROM "shipping_line_container_prefixes" GROUP BY "prefix" HAVING count(*) > 1
  ) duplicated;
  IF conflict_count > 0 THEN
    RAISE EXCEPTION '[shipping_line_container_prefixes] % 个前缀存在多行冲突（样例：%），fail-fast 中止迁移；请核对冲突报告并处置开发数据后重跑', conflict_count, sample_codes;
  END IF;
END $$;

DROP INDEX "shippingline_organization_id_scac_code";
DROP INDEX "shippingline_organization_id_enabled_sort_order";
DROP INDEX "shippinglinecontainerprefix_organization_id_prefix";
CREATE UNIQUE INDEX "shippingline_scac_code" ON "shipping_lines" ("scac_code");
CREATE INDEX "shippingline_enabled_sort_order" ON "shipping_lines" ("enabled", "sort_order");
CREATE UNIQUE INDEX "shippinglinecontainerprefix_prefix" ON "shipping_line_container_prefixes" ("prefix");
ALTER TABLE "shipping_lines" DROP COLUMN "organization_id";
ALTER TABLE "shipping_line_container_prefixes" DROP COLUMN "organization_id";

-- ============ 4. billing_units：A 型收口 ============

-- 4.1 同代码多行为未处置冲突：fail-fast 中止迁移（同 2.1）。
DO $$
DECLARE
  conflict_count INT;
  sample_codes TEXT;
BEGIN
  SELECT count(*), COALESCE(string_agg("code", ', ' ORDER BY "code"), '')
    INTO conflict_count, sample_codes
  FROM (
    SELECT "code" FROM "billing_units" GROUP BY "code" HAVING count(*) > 1
  ) duplicated;
  IF conflict_count > 0 THEN
    RAISE EXCEPTION '[billing_units] % 个代码存在多行冲突（样例：%），fail-fast 中止迁移；请核对冲突报告并处置开发数据后重跑', conflict_count, sample_codes;
  END IF;
END $$;

DROP INDEX "billingunit_organization_id_code";
DROP INDEX "billingunit_organization_id_enabled_sort_order";
CREATE UNIQUE INDEX "billingunit_code" ON "billing_units" ("code");
CREATE INDEX "billingunit_enabled_sort_order" ON "billing_units" ("enabled", "sort_order");
ALTER TABLE "billing_units" DROP COLUMN "organization_id";

-- ============ 5. ports：B 型基线 + 本地 ============

ALTER TABLE "ports" ALTER COLUMN "organization_id" DROP NOT NULL;
UPDATE "ports"
SET "organization_id" = NULL
WHERE "organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters');
DROP INDEX "port_organization_id_un_locode";
CREATE UNIQUE INDEX "ports_baseline_locode_unique" ON "ports" ("un_locode") WHERE "organization_id" IS NULL;
CREATE UNIQUE INDEX "ports_org_locode_unique" ON "ports" ("organization_id", "un_locode") WHERE "organization_id" IS NOT NULL;

-- ============ 6. airports：B 型基线 + 本地 ============

ALTER TABLE "airports" ALTER COLUMN "organization_id" DROP NOT NULL;
UPDATE "airports"
SET "organization_id" = NULL
WHERE "organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters');
DROP INDEX "airport_organization_id_iata_code";
DROP INDEX "airport_organization_id_icao_code";
CREATE UNIQUE INDEX "airports_baseline_iata_unique" ON "airports" ("iata_code") WHERE "organization_id" IS NULL;
CREATE UNIQUE INDEX "airports_org_iata_unique" ON "airports" ("organization_id", "iata_code") WHERE "organization_id" IS NOT NULL;
CREATE UNIQUE INDEX "airports_baseline_icao_unique" ON "airports" ("icao_code") WHERE "organization_id" IS NULL AND "icao_code" IS NOT NULL;

-- ============ 7. exchange_rate_settings：B 型基线 + 本地 ============

ALTER TABLE "exchange_rate_settings" ALTER COLUMN "organization_id" DROP NOT NULL;
UPDATE "exchange_rate_settings"
SET "organization_id" = NULL
WHERE "organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters');
DROP INDEX "exchange_rate_setting_unique_effective_from";
CREATE UNIQUE INDEX "exchange_rate_setting_baseline_effective_from_unique"
  ON "exchange_rate_settings" ("from_currency", "to_currency", "effective_from") WHERE "organization_id" IS NULL;
CREATE UNIQUE INDEX "exchange_rate_setting_org_effective_from_unique"
  ON "exchange_rate_settings" ("organization_id", "from_currency", "to_currency", "effective_from") WHERE "organization_id" IS NOT NULL;

-- ============ 8. fee_settings：B 型基线 + 本地；charge_category 必挂 ============

-- 8.1 列更名与外键更名（历史库与冷启动链上该外键名不同，按名称特征定位后统一更名）
ALTER TABLE "fee_settings" RENAME COLUMN "service_type_id" TO "charge_category_id";
DO $$
DECLARE
  src TEXT;
  target TEXT := 'fee_settings_master_data_items_charge_category_fee_settings';
BEGIN
  SELECT conname INTO src
  FROM pg_constraint
  WHERE conrelid = 'fee_settings'::regclass AND contype = 'f' AND conname LIKE '%service_type%';
  IF src IS NOT NULL AND src <> target THEN
    EXECUTE format('ALTER TABLE "fee_settings" RENAME CONSTRAINT %I TO %I', src, target);
  END IF;
END $$;

-- 8.2 存量 NULL 行按费用名称关键词归类到 A 型费用大类；无法归类的行 fail-fast
UPDATE "fee_settings" f
SET "charge_category_id" = m."id"
FROM "master_data_items" m
WHERE m."kind" = 'charge_category'
  AND f."charge_category_id" IS NULL
  AND (
    (f."name_zh" LIKE '%订舱%' AND m."code" = 'BOOKING') OR
    (f."name_zh" LIKE '%拖车%' AND m."code" = 'TRUCKING') OR
    (f."name_zh" LIKE '%内装%' AND m."code" = 'STUFFING') OR
    (f."name_zh" LIKE '%报关%' AND m."code" = 'CUSTOMS_EXPORT') OR
    (f."name_zh" LIKE '%清关%' AND m."code" = 'CUSTOMS_IMPORT') OR
    (f."name_zh" LIKE '%海外段%' AND m."code" = 'OVERSEA_SEGMENT') OR
    (f."name_zh" LIKE '%保险%' AND m."code" = 'INSURANCE') OR
    (f."name_zh" LIKE '%包板%' AND m."code" = 'PALLET_CHARTER') OR
    (f."name_zh" LIKE '%租箱%' AND m."code" = 'CONTAINER_LEASE') OR
    (f."name_zh" LIKE '%熏蒸%' AND m."code" = 'FUMIGATION') OR
    (f."name_zh" LIKE '%买单%' AND m."code" = 'DOC_BUY') OR
    (f."name_zh" LIKE '%办证%' AND m."code" = 'CERTIFICATE') OR
    (f."name_zh" LIKE '%制单%' AND m."code" = 'DOC_PREP') OR
    (f."name_zh" LIKE '%危险品%' AND m."code" = 'DANGEROUS_SERVICE') OR
    (f."name_zh" LIKE '%超重%' AND m."code" = 'OVERWEIGHT_SERVICE') OR
    (f."name_zh" LIKE '%换单%' AND m."code" = 'DOCUMENT_EXCHANGE') OR
    (f."name_zh" LIKE '%仓储%' AND m."code" = 'WAREHOUSING') OR
    (f."name_zh" LIKE '%报检%' AND m."code" = 'INSPECTION') OR
    (f."name_zh" LIKE '%买箱%' AND m."code" = 'CONTAINER_PURCHASE')
  );

DO $$
DECLARE
  unclassified INT;
BEGIN
  SELECT count(*) INTO unclassified FROM "fee_settings" WHERE "charge_category_id" IS NULL;
  IF unclassified > 0 THEN
    RAISE EXCEPTION '[fee_settings] % 行费用大类无法按名称归类，fail-fast 中止；请人工裁定后重跑', unclassified;
  END IF;
END $$;

ALTER TABLE "fee_settings" ALTER COLUMN "charge_category_id" SET NOT NULL;

-- 8.3 总部行置 NULL + 部分唯一索引
ALTER TABLE "fee_settings" ALTER COLUMN "organization_id" DROP NOT NULL;
UPDATE "fee_settings"
SET "organization_id" = NULL
WHERE "organization_id" IN (SELECT "id" FROM "organizations" WHERE "parent_id" IS NULL AND "kind" = 'headquarters');
DROP INDEX "feesetting_organization_id_fee_code";
DROP INDEX "feesetting_organization_id_service_type_id_abnormal_case_id";
CREATE UNIQUE INDEX "fee_settings_baseline_code_unique" ON "fee_settings" ("fee_code") WHERE "organization_id" IS NULL;
CREATE UNIQUE INDEX "fee_settings_org_code_unique" ON "fee_settings" ("organization_id", "fee_code") WHERE "organization_id" IS NOT NULL;
CREATE INDEX "feesetting_organization_id_charge_category_id_abnormal_case_id" ON "fee_settings" ("organization_id", "charge_category_id", "abnormal_case_id");
