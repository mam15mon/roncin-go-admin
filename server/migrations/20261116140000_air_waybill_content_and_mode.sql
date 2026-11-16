-- 根据航空 Ent Schema 差异生成结构变更；升级预检与正文保留按已批准合同审阅补充。
-- 旧数据全部为 HOUSE；缺失/错挂分单不可猜测为 DIRECT，先修复事实再升级。
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM air_master_waybills m WHERE NOT EXISTS (SELECT 1 FROM air_master_waybill_order_links l WHERE l.master_waybill_id=m.id)) OR EXISTS (
  SELECT 1 FROM air_master_waybill_order_links l
  LEFT JOIN air_house_waybills h ON h.order_id=l.order_id AND h.master_waybill_id=l.master_waybill_id AND h.organization_id=l.organization_id
  LEFT JOIN air_master_waybills m ON m.id=l.master_waybill_id
  LEFT JOIN orders o ON o.id=l.order_id
  WHERE h.id IS NULL OR m.organization_id<>l.organization_id OR o.organization_id<>l.organization_id OR o.business_type<>m.business_type
 ) OR EXISTS (
  SELECT 1 FROM air_house_waybills h
  WHERE NOT EXISTS (SELECT 1 FROM air_master_waybill_order_links l WHERE l.order_id=h.order_id AND l.master_waybill_id=h.master_waybill_id AND l.organization_id=h.organization_id)
 ) OR EXISTS (
  SELECT 1 FROM air_house_waybill_versions v JOIN air_house_waybills h ON h.id=v.house_waybill_id
  WHERE v.organization_id<>h.organization_id OR v.order_id<>h.order_id OR v.house_no<>h.house_no
 ) THEN RAISE EXCEPTION '航空关系或历史不完整，停止模式/正文迁移；不得推断直单或丢弃原文本'; END IF;
END $$;

ALTER TABLE air_master_waybill_order_links ADD COLUMN document_mode varchar NOT NULL DEFAULT 'HOUSE';
ALTER TABLE air_master_waybill_order_links ALTER COLUMN document_mode DROP DEFAULT;
ALTER TABLE air_master_waybill_order_links ADD CONSTRAINT air_links_document_mode_check CHECK (document_mode IN ('DIRECT','HOUSE'));
CREATE UNIQUE INDEX idx_air_links_active_direct_master ON air_master_waybill_order_links(master_waybill_id) WHERE status = 'ACTIVE' AND document_mode = 'DIRECT';

ALTER TABLE air_master_waybills ADD COLUMN waybill_content jsonb NOT NULL DEFAULT '{}', ADD COLUMN content_hash varchar(64) NOT NULL DEFAULT '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a';
UPDATE air_master_waybills SET content_hash=encode(sha256(convert_to('{}','UTF8')),'hex');

ALTER TABLE air_house_waybills ALTER COLUMN content DROP DEFAULT;
ALTER TABLE air_house_waybills ALTER COLUMN content TYPE jsonb USING CASE WHEN content='' THEN '{}'::jsonb ELSE jsonb_build_object('remarks',content) END;
ALTER TABLE air_house_waybills ALTER COLUMN content SET DEFAULT '{}';
ALTER TABLE air_house_waybills ADD COLUMN content_hash varchar(64) NOT NULL DEFAULT '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a';
ALTER TABLE air_house_waybill_versions ALTER COLUMN content TYPE jsonb USING CASE WHEN content='' THEN '{}'::jsonb ELSE jsonb_build_object('remarks',content) END;
ALTER TABLE air_house_waybill_versions ADD COLUMN content_hash varchar(64) NOT NULL DEFAULT '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a';

-- Go encoding/json 的单 remarks 对象编码，包含 HTML 与 Unicode 行分隔符转义。
-- 临时函数只参与本次迁移，不作为运行期兼容或业务逻辑。
CREATE FUNCTION air_legacy_content_hash(v jsonb) RETURNS text LANGUAGE sql IMMUTABLE AS $$
 SELECT encode(sha256(convert_to(CASE WHEN v='{}'::jsonb THEN '{}' ELSE
  '{"remarks":'||replace(replace(replace(replace(replace(to_json(v->>'remarks')::text,'<','\u003c'),'>','\u003e'),'&','\u0026'),chr(8232),'\u2028'),chr(8233),'\u2029')||'}' END,'UTF8')),'hex')
$$;
UPDATE air_house_waybills SET content_hash=air_legacy_content_hash(content);
UPDATE air_house_waybill_versions SET content_hash=air_legacy_content_hash(content);
DROP FUNCTION air_legacy_content_hash(jsonb);

ALTER TABLE air_master_waybill_histories ADD COLUMN content_hash varchar(64) NOT NULL DEFAULT '44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a';
ALTER TABLE air_master_waybill_histories ALTER COLUMN content_hash DROP DEFAULT;

-- JSON 空对象由 Ent 创建默认值提供，数据库临时默认仅用于安全回填。
ALTER TABLE air_master_waybills ALTER COLUMN waybill_content DROP DEFAULT;
ALTER TABLE air_house_waybills ALTER COLUMN content DROP DEFAULT;
ALTER TABLE air_house_waybill_versions ALTER COLUMN content_hash DROP DEFAULT;
