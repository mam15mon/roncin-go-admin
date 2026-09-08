-- 收敛曾执行海运主分单模型初始版本的数据库，确保单证结构必须由业务显式指定。
ALTER TABLE "sea_master_bill_order_links"
  ALTER COLUMN "document_structure" DROP DEFAULT;
