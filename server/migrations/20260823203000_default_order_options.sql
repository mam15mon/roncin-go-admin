INSERT INTO "master_data_items" (
    "id", "created_at", "updated_at", "kind", "code", "name",
    "source", "sort_order", "enabled", "organization_id"
)
SELECT
    gen_random_uuid(), NOW(), NOW(), defaults.kind, defaults.code,
    defaults.name, 'system', defaults.sort_order, true, organizations.id
FROM "organizations"
CROSS JOIN (
    VALUES
        ('service_type', 'BOOKING', '订舱', 10),
        ('service_type', 'TRUCKING', '拖车', 20),
        ('service_type', 'STUFFING', '内装', 30),
        ('service_type', 'CUSTOMS_EXPORT', '报关', 40),
        ('service_type', 'CUSTOMS_IMPORT', '清关', 50),
        ('service_type', 'OVERSEA_SEGMENT', '海外段', 60),
        ('service_type', 'INSURANCE', '保险', 70),
        ('service_type', 'PALLET_CHARTER', '包板', 80),
        ('service_type', 'CONTAINER_LEASE', '租箱', 90),
        ('service_type', 'FUMIGATION', '熏蒸', 100),
        ('service_type', 'DOC_BUY', '买单', 110),
        ('service_type', 'CERTIFICATE', '办证', 120),
        ('service_type', 'DOC_PREP', '制单', 130),
        ('service_type', 'DANGEROUS_SERVICE', '危险品', 140),
        ('service_type', 'OVERWEIGHT_SERVICE', '超重', 150),
        ('service_type', 'DOCUMENT_EXCHANGE', '换单', 160),
        ('service_type', 'WAREHOUSING', '仓储', 170),
        ('service_type', 'INSPECTION', '报检', 180),
        ('service_type', 'CONTAINER_PURCHASE', '买箱', 190),
        ('cargo_category', 'GENERAL', '普货', 10),
        ('cargo_category', 'REEFER', '冷藏货物', 20),
        ('cargo_category', 'OVERSIZE', '超限货', 30),
        ('cargo_category', 'DANGEROUS', '危险品', 40),
        ('cargo_category', 'BREAK_BULK_PIECE', '散杂件货', 50)
) AS defaults(kind, code, name, sort_order)
ON CONFLICT ("organization_id", "kind", "code") DO NOTHING;
