-- 主数据目录添加 abnormal_case 类型枚举检查约束。
ALTER TABLE "master_data_items" DROP CONSTRAINT "master_data_items_kind_check";

ALTER TABLE "master_data_items" ADD CONSTRAINT "master_data_items_kind_check"
  CHECK ("kind" IN ('currency', 'country', 'region', 'port', 'airport', 'carrier', 'container_spec', 'service_type', 'cargo_category', 'abnormal_case'));