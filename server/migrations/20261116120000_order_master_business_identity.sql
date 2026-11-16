-- 海空主单业务方向为不可变身份；先核对全部历史关联，不从活动成员猜测。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM sea_master_bill_order_links l JOIN orders o ON o.id=l.order_id
    JOIN sea_master_bills m ON m.id=l.master_bill_id
    WHERE o.business_type NOT IN ('SE','SI') OR o.trade_direction <> CASE o.business_type WHEN 'SE' THEN 'export' ELSE 'import' END OR o.organization_id<>m.organization_id OR l.organization_id<>m.organization_id)
    OR EXISTS (SELECT 1 FROM air_master_waybill_order_links l JOIN orders o ON o.id=l.order_id
      JOIN air_master_waybills m ON m.id=l.master_waybill_id
      WHERE o.business_type NOT IN ('AE','AI') OR o.trade_direction <> CASE o.business_type WHEN 'AE' THEN 'export' ELSE 'import' END OR o.organization_id<>m.organization_id OR l.organization_id<>m.organization_id)
    OR EXISTS (SELECT m.id FROM sea_master_bills m LEFT JOIN sea_master_bill_order_links l ON l.master_bill_id=m.id LEFT JOIN orders o ON o.id=l.order_id GROUP BY m.id HAVING count(DISTINCT o.business_type)<>1)
    OR EXISTS (SELECT m.id FROM air_master_waybills m LEFT JOIN air_master_waybill_order_links l ON l.master_waybill_id=m.id LEFT JOIN orders o ON o.id=l.order_id GROUP BY m.id HAVING count(DISTINCT o.business_type)<>1)
    OR EXISTS (SELECT 1 FROM sea_master_bill_versions v JOIN sea_master_bills m ON m.id=v.master_bill_id WHERE v.organization_id<>m.organization_id)
  THEN RAISE EXCEPTION '主单历史关联包含未知/混合业务方向、公司或无法判定的主单，必须先核实，禁止自动归类'; END IF;
END $$;
ALTER TABLE sea_master_bills ADD COLUMN business_type varchar(2);
ALTER TABLE air_master_waybills ADD COLUMN business_type varchar(2);
ALTER TABLE sea_master_bill_versions ADD COLUMN business_type varchar(2);
-- 只从全部历史关系中的唯一合法方向回填；不使用当前活动成员，也不为孤立主单猜默认值。
UPDATE sea_master_bills m SET business_type=x.business_type FROM (SELECT l.master_bill_id,min(o.business_type::text) business_type FROM sea_master_bill_order_links l JOIN orders o ON o.id=l.order_id GROUP BY l.master_bill_id) x WHERE m.id=x.master_bill_id;
UPDATE air_master_waybills m SET business_type=x.business_type FROM (SELECT l.master_waybill_id,min(o.business_type::text) business_type FROM air_master_waybill_order_links l JOIN orders o ON o.id=l.order_id GROUP BY l.master_waybill_id) x WHERE m.id=x.master_waybill_id;
UPDATE sea_master_bill_versions v SET business_type=m.business_type FROM sea_master_bills m WHERE m.id=v.master_bill_id;
ALTER TABLE sea_master_bills ALTER COLUMN business_type SET NOT NULL,
  ADD CONSTRAINT sea_master_bill_business_type_check CHECK (business_type IN ('SE','SI'));
ALTER TABLE air_master_waybills ALTER COLUMN business_type SET NOT NULL,
  ADD CONSTRAINT air_master_waybill_business_type_check CHECK (business_type IN ('AE','AI'));
ALTER TABLE sea_master_bill_versions ALTER COLUMN business_type SET NOT NULL,
  ADD CONSTRAINT sea_master_bill_version_business_type_check CHECK (business_type IN ('SE','SI'));
DROP INDEX seamasterbill_organization_id_shipping_line_id_normalized_master_no;
DROP INDEX airmasterwaybill_organization_id_master_no;
CREATE UNIQUE INDEX sea_master_bill_business_no ON sea_master_bills(organization_id,business_type,shipping_line_id,normalized_master_no);
CREATE UNIQUE INDEX air_master_waybill_business_no ON air_master_waybills(organization_id,business_type,master_no);
CREATE INDEX seamasterbill_organization_id_business_type ON sea_master_bills(organization_id,business_type);
CREATE INDEX airmasterwaybill_organization_id_business_type ON air_master_waybills(organization_id,business_type);
