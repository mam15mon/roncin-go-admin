-- 海运分单号唯一性从「组织 + 签发主体 + 号」全局唯一收敛为「同一主单批次内一号一案」：
-- 撤销两个全局部分唯一索引，新增 (master_bill_id, normalized_house_no) 唯一索引
-- （不过滤状态，作废分单号同样禁止复用；外部主体分单号跨批次合法复用不再被误挡）。
-- 建唯一索引前先预检存量同批次重号，存在即输出中文明细并终止迁移，交人工确认。

DO $$
DECLARE
  v_duplicate_groups integer;
  v_detail record;
  v_messages text;
BEGIN
  IF to_regclass(current_schema() || '.sea_house_bills') IS NOT NULL THEN
    SELECT count(*) INTO v_duplicate_groups
    FROM (
      SELECT "master_bill_id", "normalized_house_no"
      FROM "sea_house_bills"
      GROUP BY "master_bill_id", "normalized_house_no"
      HAVING count(*) > 1
    ) duplicates;
    IF v_duplicate_groups > 0 THEN
      v_messages := '';
      FOR v_detail IN
        SELECT "master_bill_id"::text AS mbl_id,
               "normalized_house_no",
               string_agg("order_id"::text || '（' || "house_no" || '）', '、' ORDER BY "order_id") AS members
        FROM "sea_house_bills"
        GROUP BY "master_bill_id", "normalized_house_no"
        HAVING count(*) > 1
      LOOP
        v_messages := v_messages || format(
          E'\n  主单 %s 分单号 %s 重复：%s',
          v_detail.mbl_id,
          v_detail.normalized_house_no,
          v_detail.members);
      END LOOP;
      RAISE EXCEPTION '海运分单批次排重迁移已停止：sea_house_bills 存在 % 组同批次重复分单号（含作废行），需人工确认去重后重试：%s',
        v_duplicate_groups, v_messages;
    END IF;
  END IF;
END $$;

DROP INDEX "idx_sea_house_bills_self_org_unique";
DROP INDEX "idx_sea_house_bills_partner_unique";
CREATE UNIQUE INDEX "idx_sea_house_bills_batch_no_unique" ON "sea_house_bills" ("master_bill_id", "normalized_house_no");
