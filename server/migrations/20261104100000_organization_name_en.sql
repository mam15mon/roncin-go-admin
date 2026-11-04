-- modify "organizations" table
ALTER TABLE "organizations" ADD COLUMN "name_en" character varying NULL;

-- 北京正式英文名来自用户确认的 HBL 模板；其他公司保留空值，由管理员维护。
UPDATE "organizations"
SET "name_en" = 'RONCIN (BEIJING) SUPPLY CHAIN MANAGEMENT CO., LTD'
WHERE "code" = 'BJ' AND "kind" = 'company' AND "name_en" IS NULL;

-- 增补本次已确认的英文检索键，保留既有中文全拼和首字母。
UPDATE "organizations"
SET "search_keywords" = concat_ws(' ', NULLIF("search_keywords", ''), upper("name_en"))
WHERE "name_en" IS NOT NULL;
