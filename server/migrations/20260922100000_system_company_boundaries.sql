-- 系统管理与独立公司拆分。迁移器在单个事务中执行；歧义数据明确中止。
DO $$
DECLARE bad text; business_table text;
BEGIN
 SELECT string_agg(id::text, ',') INTO bad FROM organizations WHERE kind='headquarters' AND parent_id IS NOT NULL;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '总部不是根节点，需人工处理: %',bad; END IF;
 SELECT string_agg(id::text, ',') INTO bad FROM orders WHERE organization_id IN (SELECT id FROM organizations WHERE kind='headquarters');
 IF bad IS NOT NULL THEN RAISE EXCEPTION '系统管理节点存在经营订单，需明确归属: %',bad; END IF;
 FOREACH business_table IN ARRAY ARRAY['partners','finance_bills','finance_invoices','finance_cashflows'] LOOP
  EXECUTE format('SELECT string_agg(b.id::text, '','') FROM %I b JOIN organizations o ON o.id=b.organization_id WHERE o.kind=''headquarters''',business_table) INTO bad;
  IF bad IS NOT NULL THEN RAISE EXCEPTION '系统管理节点存在经营数据: table=%, ids=%',business_table,bad; END IF;
 END LOOP;
 SELECT string_agg(un_locode, ',') INTO bad FROM (SELECT un_locode FROM ports GROUP BY un_locode HAVING count(*)>1) d;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '港口存在重复代码，需确认引用映射及属性: %',bad; END IF;
 SELECT string_agg(iata_code, ',') INTO bad FROM (SELECT iata_code FROM airports GROUP BY iata_code HAVING count(*)>1) d;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '机场存在重复IATA代码，需确认引用映射及属性: %',bad; END IF;
 SELECT string_agg(icao_code, ',') INTO bad FROM (SELECT icao_code FROM airports WHERE icao_code IS NOT NULL GROUP BY icao_code HAVING count(*)>1) d;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '机场存在重复ICAO代码，需确认属性: %',bad; END IF;
END $$;

-- 在拆除父子关系之前保存当前实际生效的总部策略；无策略行沿用应用默认值。
WITH RECURSIVE ancestors AS (
 SELECT id AS company_id,id,parent_id,kind FROM organizations WHERE kind='company'
 UNION ALL SELECT a.company_id,p.id,p.parent_id,p.kind FROM ancestors a JOIN organizations p ON p.id=a.parent_id
)
INSERT INTO finance_custom_settings(id,created_at,updated_at,organization_id,billed_fee_edit_enabled,billed_fee_name_editable,billed_fee_currency_editable,billed_fee_exchange_rate_editable,billed_fee_quantity_editable,billed_fee_unit_price_editable,billed_fee_tax_rate_editable,credit_limit_selection_allowed,version,updated_by)
SELECT gen_random_uuid(),f.created_at,f.updated_at,a.company_id,f.billed_fee_edit_enabled,f.billed_fee_name_editable,f.billed_fee_currency_editable,f.billed_fee_exchange_rate_editable,f.billed_fee_quantity_editable,f.billed_fee_unit_price_editable,f.billed_fee_tax_rate_editable,f.credit_limit_selection_allowed,f.version,f.updated_by
FROM ancestors a JOIN finance_custom_settings f ON f.organization_id=a.id WHERE a.kind='headquarters'
ON CONFLICT (organization_id) DO UPDATE SET billed_fee_edit_enabled=excluded.billed_fee_edit_enabled,billed_fee_name_editable=excluded.billed_fee_name_editable,billed_fee_currency_editable=excluded.billed_fee_currency_editable,billed_fee_exchange_rate_editable=excluded.billed_fee_exchange_rate_editable,billed_fee_quantity_editable=excluded.billed_fee_quantity_editable,billed_fee_unit_price_editable=excluded.billed_fee_unit_price_editable,billed_fee_tax_rate_editable=excluded.billed_fee_tax_rate_editable,credit_limit_selection_allowed=excluded.credit_limit_selection_allowed,updated_by=excluded.updated_by,version=excluded.version,created_at=excluded.created_at,updated_at=excluded.updated_at;
DELETE FROM finance_custom_settings WHERE organization_id IN (SELECT id FROM organizations WHERE kind='headquarters');
ALTER TABLE organizations DROP CONSTRAINT organizations_kind_check;
ALTER TABLE organizations DROP CONSTRAINT organizations_base_currency_by_kind;
UPDATE organizations SET parent_id=NULL WHERE kind='company';
UPDATE organizations SET kind='system',name='系统管理',search_keywords='系统管理 xitongguanli xtgl' WHERE kind='headquarters';
ALTER TABLE organizations ADD CONSTRAINT organizations_kind_check CHECK(kind IN ('system','company','department','team'));
ALTER TABLE organizations ADD CONSTRAINT organizations_base_currency_by_kind CHECK((kind IN ('system','company') AND base_currency IS NOT NULL) OR (kind IN ('department','team') AND base_currency IS NULL));
ALTER TABLE organizations ADD CONSTRAINT organizations_workspace_parent_check CHECK((kind IN ('system','company') AND parent_id IS NULL) OR (kind IN ('department','team') AND parent_id IS NOT NULL));

-- 地点保留所有原ID和业务引用，仅移除公司归属，不推测合并重复记录。
ALTER TABLE ports DROP COLUMN organization_id;
CREATE UNIQUE INDEX ports_locode_unique ON ports(un_locode);
CREATE INDEX port_enabled_sort_order ON ports(enabled,sort_order);
ALTER TABLE airports DROP COLUMN organization_id;
CREATE UNIQUE INDEX airports_iata_unique ON airports(iata_code);
CREATE UNIQUE INDEX airports_icao_unique ON airports(icao_code) WHERE icao_code IS NOT NULL;
CREATE INDEX airport_enabled_sort_order ON airports(enabled,sort_order);

-- 系统初始目录仅保存税务文本；实际科目与应税劳务都属于公司。
CREATE TABLE "fee_setting_templates" (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 fee_code varchar(32) NOT NULL,name_zh varchar(64) NOT NULL,name_en varchar(128),alias_name varchar(64),
 charge_category_id uuid NOT NULL,default_currency varchar(3) NOT NULL,billing_unit_id uuid NOT NULL,abnormal_case_id uuid,
 tax_rate numeric(5,2) NOT NULL,taxable_service_name varchar(128) NOT NULL,taxable_service_short_name varchar(64),taxable_service_goods_code varchar(64),taxable_service_default_tax_rate numeric(5,2) NOT NULL,
 enabled boolean NOT NULL DEFAULT true,sort_order bigint NOT NULL DEFAULT 100,search_keywords text NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX feesettingtemplate_fee_code ON fee_setting_templates(fee_code);
CREATE INDEX feesettingtemplate_enabled_sort_order ON fee_setting_templates(enabled,sort_order);
CREATE INDEX feesettingtemplate_updated_at ON fee_setting_templates(updated_at);

-- 旧管理节点的私有目录经确认转为系统模板，不向已有公司复制。
DO $$
DECLARE bad text;
BEGIN
 SELECT string_agg(f.id::text,',') INTO bad FROM fee_settings f JOIN organizations o ON o.id=f.organization_id LEFT JOIN taxable_services t ON t.id=f.taxable_service_id WHERE o.kind='system' AND t.organization_id IS DISTINCT FROM f.organization_id;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '系统费用模板税务归属不合法: %',bad; END IF;
 SELECT string_agg(f.id::text,',') INTO bad FROM fee_settings f JOIN organizations o ON o.id=f.organization_id WHERE o.kind='system' AND (EXISTS(SELECT 1 FROM order_fees r WHERE r.fee_setting_id=f.id) OR EXISTS(SELECT 1 FROM order_fee_supplement_requests r WHERE r.fee_setting_id=f.id));
 IF bad IS NOT NULL THEN RAISE EXCEPTION '系统私有费用存在经营引用，需明确归属: %',bad; END IF;
 SELECT string_agg(fee_code,',') INTO bad FROM (SELECT f.fee_code FROM fee_settings f LEFT JOIN organizations o ON o.id=f.organization_id WHERE f.organization_id IS NULL OR o.kind='system' GROUP BY f.fee_code HAVING count(*)>1) duplicates;
 IF bad IS NOT NULL THEN RAISE EXCEPTION '系统费用模板存在同码冲突: %',bad; END IF;
END $$;

INSERT INTO fee_setting_templates
SELECT f.id,f.created_at,f.updated_at,f.fee_code,f.name_zh,f.name_en,f.alias_name,f.charge_category_id,f.default_currency,f.billing_unit_id,f.abnormal_case_id,f.tax_rate,t.name,t.short_name,t.goods_code,t.default_tax_rate,f.enabled,f.sort_order,f.search_keywords
FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.organization_id IS NULL OR f.organization_id IN (SELECT id FROM organizations WHERE kind='system');

-- 模板保存原科目ID和税务默认文本；原税项保留完整记录，避免丢失未引用配置。
DELETE FROM fee_settings WHERE organization_id IN (SELECT id FROM organizations WHERE kind='system');

DO $$
DECLARE src fee_settings%ROWTYPE; tax taxable_services%ROWTYPE; local_tax taxable_services%ROWTYPE; local_fee fee_settings%ROWTYPE; company record; target_tax uuid; target_fee uuid; bad text;
BEGIN
 SELECT string_agg(f.id::text,',') INTO bad FROM fee_settings f LEFT JOIN organizations o ON o.id=f.organization_id LEFT JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.organization_id IS NOT NULL AND (o.kind IS DISTINCT FROM 'company' OR t.organization_id IS DISTINCT FROM f.organization_id);
 IF bad IS NOT NULL THEN RAISE EXCEPTION '费用科目公司或税务归属不合法: %',bad; END IF;
 SELECT string_agg(r.id::text,',') INTO bad FROM (
 SELECT f.id,o.organization_id FROM order_fees f JOIN orders o ON o.id=f.order_id JOIN fee_settings s ON s.id=f.fee_setting_id WHERE s.organization_id IS NULL
 UNION ALL SELECT f.id,f.organization_id FROM order_fee_supplement_requests f JOIN fee_settings s ON s.id=f.fee_setting_id WHERE s.organization_id IS NULL
 ) r LEFT JOIN organizations o ON o.id=r.organization_id WHERE o.kind IS DISTINCT FROM 'company';
 IF bad IS NOT NULL THEN RAISE EXCEPTION '共享费用引用无合法公司归属: %',bad; END IF;
 FOR src IN SELECT * FROM fee_settings WHERE organization_id IS NULL ORDER BY id LOOP
  SELECT * INTO STRICT tax FROM taxable_services WHERE id=src.taxable_service_id;
  FOR company IN SELECT id FROM organizations WHERE kind='company' ORDER BY id LOOP
   SELECT * INTO local_fee FROM fee_settings WHERE organization_id=company.id AND fee_code=src.fee_code;
   IF FOUND THEN
    target_fee:=local_fee.id;
    -- 同码本地科目继续优先，但旧共享引用不能被静默改成不同配置。
    IF EXISTS(SELECT 1 FROM order_fees f JOIN orders o ON o.id=f.order_id WHERE f.fee_setting_id=src.id AND o.organization_id=company.id) OR EXISTS(SELECT 1 FROM order_fee_supplement_requests WHERE fee_setting_id=src.id AND organization_id=company.id) THEN
     SELECT * INTO STRICT local_tax FROM taxable_services WHERE id=local_fee.taxable_service_id;
     IF (to_jsonb(local_fee)-ARRAY['id','organization_id','created_at','updated_at','taxable_service_id','search_keywords']) IS DISTINCT FROM (to_jsonb(src)-ARRAY['id','organization_id','created_at','updated_at','taxable_service_id','search_keywords']) OR (to_jsonb(local_tax)-ARRAY['id','organization_id','created_at','updated_at','search_keywords']) IS DISTINCT FROM (to_jsonb(tax)-ARRAY['id','organization_id','created_at','updated_at','search_keywords']) THEN
      RAISE EXCEPTION '共享费用引用与本地同码配置冲突: company=%, source=%, local=%',company.id,src.id,local_fee.id;
     END IF;
    END IF;
   ELSE
    SELECT * INTO local_tax FROM taxable_services WHERE organization_id=company.id AND name=tax.name;
    IF FOUND THEN
     IF ROW(local_tax.short_name,local_tax.goods_code,local_tax.default_tax_rate,local_tax.enabled) IS DISTINCT FROM ROW(tax.short_name,tax.goods_code,tax.default_tax_rate,tax.enabled) THEN
      RAISE EXCEPTION '初始化费用税务同名配置冲突: company=%, source_tax=%, local_tax=%',company.id,tax.id,local_tax.id;
     END IF;
     target_tax:=local_tax.id;
    ELSE
     target_tax:=gen_random_uuid();
     INSERT INTO taxable_services(id,created_at,updated_at,organization_id,name,short_name,goods_code,default_tax_rate,enabled,search_keywords) VALUES(target_tax,tax.created_at,tax.updated_at,company.id,tax.name,tax.short_name,tax.goods_code,tax.default_tax_rate,tax.enabled,tax.search_keywords);
    END IF;
    target_fee:=gen_random_uuid();
    INSERT INTO fee_settings(id,created_at,updated_at,organization_id,fee_code,name_zh,name_en,alias_name,charge_category_id,default_currency,billing_unit_id,abnormal_case_id,tax_rate,taxable_service_id,enabled,sort_order,search_keywords) VALUES(target_fee,src.created_at,src.updated_at,company.id,src.fee_code,src.name_zh,src.name_en,src.alias_name,src.charge_category_id,src.default_currency,src.billing_unit_id,src.abnormal_case_id,src.tax_rate,target_tax,src.enabled,src.sort_order,src.search_keywords);
   END IF;
   UPDATE order_fees f SET fee_setting_id=target_fee FROM orders o WHERE o.id=f.order_id AND f.fee_setting_id=src.id AND o.organization_id=company.id;
   UPDATE order_fee_supplement_requests SET fee_setting_id=target_fee WHERE fee_setting_id=src.id AND organization_id=company.id;
  END LOOP;
 END LOOP;
END $$;
-- 所有共享目录已经转为模板及公司副本，原始单据快照保持不变。
DELETE FROM fee_settings WHERE organization_id IS NULL;
ALTER TABLE fee_settings ALTER COLUMN organization_id SET NOT NULL;
DROP INDEX fee_settings_baseline_code_unique;
DROP INDEX fee_settings_org_code_unique;
CREATE UNIQUE INDEX fee_settings_org_code_unique ON fee_settings(organization_id,fee_code);
