-- 仅退役业务标签；发现标签主体被非标签明细引用时拒绝迁移，保留异常现场。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM (
      SELECT resource_id FROM enterprise_resource_addresses
      UNION ALL SELECT resource_id FROM enterprise_resource_remarks
      UNION ALL SELECT resource_id FROM enterprise_resource_images
      UNION ALL SELECT resource_id FROM enterprise_resource_parties
      UNION ALL SELECT resource_id FROM enterprise_resource_shipping_texts
      UNION ALL SELECT resource_id FROM enterprise_resource_assignees
      UNION ALL SELECT resource_id FROM enterprise_resource_address_types
    ) AS detail
    JOIN enterprise_resources AS resource ON resource.id = detail.resource_id
    WHERE resource.resource_type = 'TAG'
  ) THEN
    RAISE EXCEPTION '业务标签被非标签明细引用，迁移已拒绝，请核对数据';
  END IF;
  IF EXISTS (
    SELECT 1 FROM enterprise_resource_partners AS link
    JOIN enterprise_resources AS resource ON resource.id = link.resource_id
    WHERE (link.resource_type = 'TAG' OR resource.resource_type = 'TAG')
      AND link.resource_type <> resource.resource_type
  ) THEN
    RAISE EXCEPTION '标签客商关联与资源类型不一致，迁移已拒绝，请核对数据';
  END IF;
END $$;

DROP TABLE "order_enterprise_tags";
DROP TABLE "order_fee_enterprise_tags";
DROP TABLE "finance_bill_enterprise_tags";

DELETE FROM "enterprise_resource_partners" AS link
USING "enterprise_resources" AS resource
WHERE link.resource_id = resource.id AND resource.resource_type = 'TAG';

DROP TABLE "enterprise_tags";
DROP TABLE "enterprise_tag_groups";
DELETE FROM "enterprise_resources" WHERE resource_type = 'TAG';
