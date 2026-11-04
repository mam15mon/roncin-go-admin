-- 用户确认成都沿用北京抬头结构，仅将城市拼音改为 CHENGDU。
-- 仅补齐既有成都公司的空白名称，保留管理员维护的非空值及原有检索键。
UPDATE "organizations"
SET "name_en" = 'RONCIN (CHENGDU) SUPPLY CHAIN MANAGEMENT CO., LTD',
    "search_keywords" = concat_ws(' ', NULLIF("search_keywords", ''), 'RONCIN (CHENGDU) SUPPLY CHAIN MANAGEMENT CO., LTD')
WHERE "code" = 'CD' AND "kind" = 'company'
  AND ("name_en" IS NULL OR "name_en" ~ '^[[:space:]]*$');
