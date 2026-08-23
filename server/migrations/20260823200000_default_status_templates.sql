WITH template_defaults(code, name, business_type) AS (
    VALUES
        ('SE_SYSTEM', '海运出口内置状态流转', 'SE'),
        ('SI_SYSTEM', '海运进口内置状态流转', 'SI'),
        ('AE_SYSTEM', '空运出口内置状态流转', 'AE'),
        ('AI_SYSTEM', '空运进口内置状态流转', 'AI'),
        ('LAND_SYSTEM', '陆运内置状态流转', 'LAND'),
        ('RAIL_SYSTEM', '铁路内置状态流转', 'RAIL')
),
inserted_templates AS (
    INSERT INTO "status_templates" (
        "id", "created_at", "updated_at", "code", "name", "business_type",
        "version", "is_default", "published_at", "enabled", "organization_id"
    )
    SELECT
        gen_random_uuid(), NOW(), NOW(), defaults.code, defaults.name,
        defaults.business_type, 1, true, NOW(), true, organizations.id
    FROM "organizations"
    CROSS JOIN template_defaults AS defaults
    WHERE NOT EXISTS (
        SELECT 1
        FROM "status_templates" AS existing
        WHERE existing."organization_id" = organizations."id"
          AND existing."business_type" = defaults.business_type
          AND existing."is_default" = true
    )
    ON CONFLICT ("organization_id", "business_type", "code", "version") DO NOTHING
    RETURNING "id", "business_type"
),
item_defaults(business_type, code, label, sort_order, color_token) AS (
    VALUES
        ('SE', 'DRAFT', '新建', 0, 'slate'),
        ('SE', 'BOOKED', '已订舱', 1, 'blue'),
        ('SE', 'SPACE_ALLOCATED', '已配舱', 2, 'cyan'),
        ('SE', 'TRUCKING_ARRANGED', '拖车已安排', 3, 'amber'),
        ('SE', 'WAREHOUSE_DELIVERED', '已送仓', 4, 'amber'),
        ('SE', 'INVOICE_ISSUED', '已开票', 5, 'blue'),
        ('SE', 'CUSTOMS_DECLARATION_ARRANGED', '报关已安排', 6, 'violet'),
        ('SI', 'DRAFT', '新建', 0, 'slate'),
        ('SI', 'TRUCKING_ARRANGED', '拖车已安排', 1, 'amber'),
        ('SI', 'CUSTOMS_DECLARATION_ARRANGED', '报关已安排', 2, 'violet'),
        ('SI', 'BILL_EXCHANGED', '已换单', 3, 'teal'),
        ('SI', 'INSPECTION_ARRANGED', '报检已安排', 4, 'violet'),
        ('AE', 'DRAFT', '新建', 0, 'slate'),
        ('AE', 'BOOKED', '已订舱', 1, 'blue'),
        ('AE', 'SPACE_ALLOCATED', '已配舱', 2, 'cyan'),
        ('AE', 'TRUCKING_ARRANGED', '拖车已安排', 3, 'amber'),
        ('AE', 'DOCUMENT_CUTOFF', '已截单', 4, 'rose'),
        ('AE', 'CUSTOMS_DECLARATION_ARRANGED', '报关已安排', 5, 'violet'),
        ('AE', 'DOCUMENT_SIGNED', '已签单', 6, 'indigo'),
        ('AI', 'DRAFT', '新建', 0, 'slate'),
        ('AI', 'TRUCKING_ARRANGED', '拖车已安排', 1, 'amber'),
        ('AI', 'CUSTOMS_DECLARATION_ARRANGED', '报关已安排', 2, 'violet'),
        ('AI', 'BILL_EXCHANGED', '已换单', 3, 'teal'),
        ('AI', 'INSPECTION_ARRANGED', '报检已安排', 4, 'violet'),
        ('LAND', 'DRAFT', '新建', 0, 'slate'),
        ('LAND', 'TRUCKING_ARRANGED', '拖车已安排', 1, 'amber'),
        ('RAIL', 'DRAFT', '新建', 0, 'slate'),
        ('RAIL', 'DOCUMENT_SIGNED', '已签单', 1, 'indigo'),
        ('RAIL', 'BOOKED', '已订舱', 2, 'blue'),
        ('RAIL', 'SPACE_ALLOCATED', '已配舱', 3, 'cyan'),
        ('RAIL', 'TRUCKING_ARRANGED', '拖车已安排', 4, 'amber'),
        ('RAIL', 'DOCUMENT_CUTOFF', '已截单', 5, 'rose'),
        ('RAIL', 'CUSTOMS_DECLARATION_ARRANGED', '报关已安排', 6, 'violet')
)
INSERT INTO "status_template_items" (
    "id", "created_at", "updated_at", "code", "label", "sort_order",
    "enabled", "color_token", "system", "template_id"
)
SELECT
    gen_random_uuid(), NOW(), NOW(), defaults.code, defaults.label,
    defaults.sort_order, true, defaults.color_token, true, templates.id
FROM inserted_templates AS templates
JOIN item_defaults AS defaults
  ON defaults.business_type = templates.business_type
ON CONFLICT ("template_id", "code") DO NOTHING;
