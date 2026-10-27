-- 船代统一引用公共主数据。旧供应商引用必须先明确映射，禁止猜配或清空。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM orders o
    WHERE o.shipping_agent_id IS NOT NULL
      AND NOT EXISTS (
        SELECT 1 FROM master_data_items m
        WHERE m.id = o.shipping_agent_id AND m.kind = 'shipping_agent'
      )
  ) THEN
    RAISE EXCEPTION '存在未映射为公共船代主数据的订单 shipping_agent_id，请确认映射后再迁移';
  END IF;
END $$;

-- 正式链有该约束；已核查的开发库可能没有，统一收敛到与 Ent 同名的约束。
ALTER TABLE master_data_items DROP CONSTRAINT IF EXISTS master_data_items_kind_check;
ALTER TABLE master_data_items ADD CONSTRAINT master_data_items_kind_check
  CHECK (kind IN ('currency', 'country', 'region', 'container_spec', 'cargo_category', 'abnormal_case', 'shipping_agent'));
