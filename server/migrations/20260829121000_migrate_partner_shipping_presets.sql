INSERT INTO "enterprise_resources" (
  "id", "created_at", "updated_at", "resource_type", "short_name", "enabled", "sort_order", "search_keywords", "organization_id", "created_by", "updated_by"
)
SELECT
  presets."id", presets."created_at", presets."updated_at", presets."preset_type", presets."title", presets."enabled", presets."sort_order", presets."title", partners."organization_id", NULL, NULL
FROM "partner_shipping_presets" AS presets
JOIN "partners" ON partners."id" = presets."partner_id";

INSERT INTO "enterprise_resource_parties" (
  "id", "created_at", "updated_at", "organization_id", "resource_type", "company_name", "business_code", "normalized_business_code", "address", "country_code", "contact_name", "contact_phone", "email", "tax_identifier", "aeo_code", "custom_display", "display_content", "remark", "resource_id"
)
SELECT
  gen_random_uuid(), presets."created_at", presets."updated_at", partners."organization_id", presets."preset_type", presets."company_name", NULL, NULL, presets."address", presets."country_code", presets."contact_name", presets."phone", presets."email", presets."tax_identifier", NULL, false,
  concat_ws(E'\n', presets."company_name", presets."address", presets."contact_name", presets."phone"), presets."remark", presets."id"
FROM "partner_shipping_presets" AS presets
JOIN "partners" ON partners."id" = presets."partner_id"
WHERE presets."preset_type" IN ('SHIPPER', 'CONSIGNEE', 'NOTIFY_PARTY');

INSERT INTO "enterprise_resource_shipping_texts" (
  "id", "created_at", "updated_at", "content", "code", "remark", "resource_id"
)
SELECT
  gen_random_uuid(), presets."created_at", presets."updated_at", presets."content", presets."code", presets."remark", presets."id"
FROM "partner_shipping_presets" AS presets
WHERE presets."preset_type" IN ('ENGLISH_CARGO_NAME', 'HS_CODE', 'MARKS');

INSERT INTO "enterprise_resource_partners" (
  "id", "created_at", "updated_at", "resource_type", "is_default", "resource_id", "partner_id"
)
SELECT
  gen_random_uuid(), presets."created_at", presets."updated_at", presets."preset_type", presets."is_default", presets."id", presets."partner_id"
FROM "partner_shipping_presets" AS presets;

DO $$
DECLARE
  source_count bigint;
  resource_count bigint;
  detail_count bigint;
  link_count bigint;
BEGIN
  SELECT count(*) INTO source_count FROM "partner_shipping_presets";
  SELECT count(*) INTO resource_count FROM "enterprise_resources" WHERE "resource_type" IN ('SHIPPER', 'CONSIGNEE', 'NOTIFY_PARTY', 'ENGLISH_CARGO_NAME', 'HS_CODE', 'MARKS');
  SELECT (SELECT count(*) FROM "enterprise_resource_parties") + (SELECT count(*) FROM "enterprise_resource_shipping_texts") INTO detail_count;
  SELECT count(*) INTO link_count FROM "enterprise_resource_partners" WHERE "resource_type" IN ('SHIPPER', 'CONSIGNEE', 'NOTIFY_PARTY', 'ENGLISH_CARGO_NAME', 'HS_CODE', 'MARKS');
  IF source_count <> resource_count OR source_count <> detail_count OR source_count <> link_count THEN
    RAISE EXCEPTION '企业常用单证迁移数量不一致: source %, resource %, detail %, link %', source_count, resource_count, detail_count, link_count;
  END IF;
END $$;

DROP TABLE "partner_shipping_presets";
