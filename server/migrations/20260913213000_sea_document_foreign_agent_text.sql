-- 为海运提单内容增加外国代理信息（foreign_agent_text）
-- 包含 MBL、HBL 及其实体对应的不变版本表

ALTER TABLE "sea_master_bills" ADD COLUMN "foreign_agent_text" text;
ALTER TABLE "sea_house_bills" ADD COLUMN "foreign_agent_text" text;
ALTER TABLE "sea_master_bill_versions" ADD COLUMN "foreign_agent_text" text;
ALTER TABLE "sea_house_bill_versions" ADD COLUMN "foreign_agent_text" text;
