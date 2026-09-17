-- 经营数据只能归属公司。总部测试经营数据须在执行迁移前显式清理，禁止静默改挂。
DO $$
DECLARE
  headquarters_partner_count bigint;
  headquarters_order_count bigint;
BEGIN
  SELECT count(*) INTO headquarters_partner_count
  FROM partners AS p
  JOIN organizations AS o ON o.id = p.organization_id
  WHERE o.kind = 'headquarters';

  SELECT count(*) INTO headquarters_order_count
  FROM orders AS ord
  JOIN organizations AS o ON o.id = ord.organization_id
  WHERE o.kind = 'headquarters';

  IF headquarters_partner_count > 0 OR headquarters_order_count > 0 THEN
    RAISE EXCEPTION '经营组织收敛迁移已停止：总部仍有客户 % 条、订单 % 条，请先确认并清理或迁移',
      headquarters_partner_count, headquarters_order_count;
  END IF;
END $$;

-- 客户责任人员的业务归属统一为客户所属公司；人员真实部门仍由 memberships 保存。
UPDATE partner_assignments AS pa
SET organization_id = p.organization_id,
    updated_at = now()
FROM partners AS p
WHERE p.id = pa.partner_id
  AND pa.organization_id <> p.organization_id;

-- 订单协作人员的业务归属统一为订单所属公司。
UPDATE order_personnels AS op
SET organization_id = ord.organization_id,
    updated_at = now()
FROM orders AS ord
WHERE ord.id = op.order_id
  AND op.organization_id <> ord.organization_id;
