-- 费用科目系统模板种子丰富：主数据补充 + 初始目录 122 条 + 既有公司追加。
-- 迁移器在单个事务中执行；引用缺失或同码税务配置冲突时显式中止。
-- search_keywords 与 internal/platform/searchtext.Build(name_zh, name_en, alias_name) 输出一致，
-- 由一次性生成器预计算（任务 09-23-fee-catalog-seed）。

-- 1. 计费单位：全新库完整种子 + DAY 新增；已存在同码跳过。
-- quantity_must_be_integer 与 20260923115000 的整数计量标志保持同一口径。
INSERT INTO "billing_units" ("id","created_at","updated_at","code","name","sort_order","enabled","is_container_unit","quantity_must_be_integer","search_keywords")
VALUES
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'CONT','箱',10,true,true,true,'箱 XIANG X'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'BL','票',20,true,false,true,'票 PIAO P'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'CBM','立方米',30,true,false,false,'立方米 LIFANGMI LI FANG MI LFM'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'TON','吨',40,true,false,false,'吨 DUN D'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'SET','套',50,true,false,true,'套 TAO T'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'DOC','份',60,true,false,true,'份 FEN F'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'DAY','天',70,true,false,false,'天 TIAN T')
ON CONFLICT ("code") DO NOTHING;

-- 2. 费用类别：幂等补种 19 个基础类别（与 data.CreateDefaultOrderOptions 同值）+ 新增 5 个。
INSERT INTO "master_data_items" ("id","created_at","updated_at","kind","code","name","name_en","source","sort_order","enabled","attributes","search_keywords")
VALUES
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','BOOKING','订舱',NULL,'system',10,true,'{}'::jsonb,'订舱 DINGCANG DING CANG DC'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','TRUCKING','拖车',NULL,'system',20,true,'{}'::jsonb,'拖车 TUOCHE TUO CHE TC'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','STUFFING','内装',NULL,'system',30,true,'{}'::jsonb,'内装 NEIZHUANG NEI ZHUANG NZ'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','CUSTOMS_EXPORT','报关',NULL,'system',40,true,'{}'::jsonb,'报关 BAOGUAN BAO GUAN BG'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','CUSTOMS_IMPORT','清关',NULL,'system',50,true,'{}'::jsonb,'清关 QINGGUAN QING GUAN QG'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','OVERSEA_SEGMENT','海外段',NULL,'system',60,true,'{}'::jsonb,'海外段 HAIWAIDUAN HAI WAI DUAN HWD'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','INSURANCE','保险',NULL,'system',70,true,'{}'::jsonb,'保险 BAOXIAN BAO XIAN BX'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','PALLET_CHARTER','包板',NULL,'system',80,true,'{}'::jsonb,'包板 BAOBAN BAO BAN BB'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','CONTAINER_LEASE','租箱',NULL,'system',90,true,'{}'::jsonb,'租箱 ZUXIANG ZU XIANG ZX'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','FUMIGATION','熏蒸',NULL,'system',100,true,'{}'::jsonb,'熏蒸 XUNZHENG XUN ZHENG XZ'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','DOC_BUY','买单',NULL,'system',110,true,'{}'::jsonb,'买单 MAIDAN MAI DAN MD'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','CERTIFICATE','办证',NULL,'system',120,true,'{}'::jsonb,'办证 BANZHENG BAN ZHENG BZ'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','DOC_PREP','制单',NULL,'system',130,true,'{}'::jsonb,'制单 ZHIDAN ZHI DAN ZD'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','DANGEROUS_SERVICE','危险品',NULL,'system',140,true,'{}'::jsonb,'危险品 WEIXIANPIN WEI XIAN PIN WXP'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','OVERWEIGHT_SERVICE','超重',NULL,'system',150,true,'{}'::jsonb,'超重 CHAOZHONG CHAO ZHONG CZ'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','DOCUMENT_EXCHANGE','换单',NULL,'system',160,true,'{}'::jsonb,'换单 HUANDAN HUAN DAN HD'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','WAREHOUSING','仓储',NULL,'system',170,true,'{}'::jsonb,'仓储 CANGCHU CANG CHU CC'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','INSPECTION','报检',NULL,'system',180,true,'{}'::jsonb,'报检 BAOJIAN BAO JIAN BJ'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','CONTAINER_PURCHASE','买箱',NULL,'system',190,true,'{}'::jsonb,'买箱 MAIXIANG MAI XIANG MX'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','PORT_OPS','港口操作','Port Operations','system',200,true,'{}'::jsonb,'港口操作 GANGKOUCAOZUO GANG KOU CAO ZUO GKCZ PORT OPERATIONS'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','AIR','空运','Air Freight','system',210,true,'{}'::jsonb,'空运 KONGYUN KONG YUN KY AIR FREIGHT'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','TAX','税费','Taxes & Duties','system',220,true,'{}'::jsonb,'税费 SHUIFEI SHUI FEI SF TAXES & DUTIES'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','FIN_ADJ','财务调整','Finance Adjustment','system',230,true,'{}'::jsonb,'财务调整 CAIWUDIAOZHENG CAI WU DIAO ZHENG CWDZ FINANCE ADJUSTMENT'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'charge_category','MISC','其他','Miscellaneous','system',240,true,'{}'::jsonb,'其他 QITA QI TA QT MISCELLANEOUS')
ON CONFLICT ("kind","code") DO NOTHING;

-- 3. 异常情况：新增待时/压夜/查验，并幂等补种客户取消出运/甩柜改配（全新库缺行）。
INSERT INTO "master_data_items" ("id","created_at","updated_at","kind","code","name","name_en","source","sort_order","enabled","attributes","search_keywords")
VALUES
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-WAITING','待时','Waiting Time','system',10,true,'{}'::jsonb,'待时 DAISHI DAI SHI DS WAITING TIME'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-OVERNIGHT','压夜','Overnight','system',20,true,'{}'::jsonb,'压夜 YAYE YA YE YY OVERNIGHT'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-INSPECTION','查验','Customs Inspection','system',30,true,'{}'::jsonb,'查验 CHAYAN CHA YAN CY CUSTOMS INSPECTION'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-CUSTOMER-CANCEL','客户取消出运','Customer Cancellation','system',40,true,'{}'::jsonb,'客户取消出运 KEHUQUXIAOCHUYUN KE HU QU XIAO CHU YUN KHQXCY CUSTOMER CANCELLATION'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-ROLL-REASSIGN','甩柜改配','Roll-over Reassignment','system',50,true,'{}'::jsonb,'甩柜改配 SHUAIGUIGAIPEI SHUAI GUI GAI PEI SGGP ROLL-OVER REASSIGNMENT')
ON CONFLICT ("kind","code") DO NOTHING;

-- 4. 系统初始目录种子：开发库已有 10 条同码模板保持原样，全新库一次种全。
INSERT INTO "fee_setting_templates" ("id","created_at","updated_at","fee_code","name_zh","name_en","alias_name","charge_category_id","default_currency","billing_unit_id","abnormal_case_id","tax_rate","taxable_service_name","taxable_service_default_tax_rate","enabled","sort_order","search_keywords")
SELECT gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,v.fee_code,v.name_zh,v.name_en,v.alias_name,
  (SELECT m."id" FROM "master_data_items" m WHERE m."kind"='charge_category' AND m."code"=v.category_code),
  v.currency,
  (SELECT u."id" FROM "billing_units" u WHERE u."code"=v.unit_code),
  (SELECT a."id" FROM "master_data_items" a WHERE a."kind"='abnormal_case' AND a."code"=v.abnormal_code),
  v.tax_rate,v.service_name,v.service_rate,true,v.sort_order,v.search_keywords
FROM (VALUES
  ('OF'::varchar,'海运费'::varchar,'Ocean Freight'::varchar,NULL::varchar,'BOOKING'::varchar,'USD'::varchar,'CONT'::varchar,NULL::varchar,0.00::numeric(5,2),'国际海运运费'::varchar,0.00::numeric(5,2),1010::bigint,'海运费 HAIYUNFEI HAI YUN FEI HYF OCEAN FREIGHT'::text),
  ('BL','提单费','B/L Fee',NULL,'BOOKING','USD','BL',NULL,0.00,'国际海运运费',0.00,1020,'提单费 TIDANFEI TI DAN FEI TDF B/L FEE'),
  ('DCF','订舱费','Booking Fee',NULL,'BOOKING','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1030,'订舱费 DINGCANGFEI DING CANG FEI DCF BOOKING FEE'),
  ('CDF','舱单费','Manifest Filing Fee',NULL,'BOOKING','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1040,'舱单费 CANGDANFEI CANG DAN FEI CDF MANIFEST FILING FEE'),
  ('AMS','AMS 舱单费（美线）','AMS Filing Fee',NULL,'BOOKING','USD','BL',NULL,0.00,'海运代理订舱服务',6.00,1050,'AMS 舱单费（美线） CANGDANFEIMEIXIAN CANG DAN FEI MEI XIAN CDFMX AMS FILING FEE'),
  ('AFR','AFR 舱单费（日本）','AFR Filing Fee',NULL,'BOOKING','USD','BL',NULL,0.00,'海运代理订舱服务',6.00,1060,'AFR 舱单费（日本） CANGDANFEIRIBEN CANG DAN FEI RI BEN CDFRB AFR FILING FEE'),
  ('ENS','ENS 舱单费（欧盟）','ENS Filing Fee',NULL,'BOOKING','USD','BL',NULL,0.00,'海运代理订舱服务',6.00,1070,'ENS 舱单费（欧盟） CANGDANFEIOUMENG CANG DAN FEI OU MENG CDFOM ENS FILING FEE'),
  ('VGM','重量验证费','VGM Fee',NULL,'BOOKING','CNY','CONT',NULL,6.00,'港口操作及港杂服务',6.00,1080,'重量验证费 ZHONGLIANGYANZHENGFEI ZHONG LIANG YAN ZHENG FEI ZLYZF VGM FEE'),
  ('DET','滞箱费','Container Detention Fee',NULL,'BOOKING','CNY','DAY',NULL,0.00,'港口操作及港杂服务',6.00,1090,'滞箱费 ZHIXIANGFEI ZHI XIANG FEI ZXF CONTAINER DETENTION FEE'),
  ('KC','空舱费','Dead Freight','亏仓费','BOOKING','CNY','BL','ABN-CUSTOMER-CANCEL',0.00,'海运代理订舱服务',6.00,1100,'空舱费 KONGCANGFEI KONG CANG FEI KCF DEAD FREIGHT 亏仓费 KUICANGFEI KUI CANG FEI'),
  ('FDFW','放单服务费','Release Order Fee',NULL,'BOOKING','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,1110,'放单服务费 FANGDANFUWUFEI FANG DAN FU WU FEI FDFWF RELEASE ORDER FEE'),
  ('DLF','代理费','Agency Fee',NULL,'BOOKING','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,1120,'代理费 DAILIFEI DAI LI FEI DLF AGENCY FEE'),
  ('SXF','手续费','Handling Charge',NULL,'BOOKING','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,1130,'手续费 SHOUXUFEI SHOU XU FEI SXF HANDLING CHARGE'),
  ('DOC','文件费','Documentation Fee',NULL,'BOOKING','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1210,'文件费 WENJIANFEI WEN JIAN FEI WJF DOCUMENTATION FEE'),
  ('TLX','电放费','Telex Release Fee',NULL,'BOOKING','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1220,'电放费 DIANFANGFEI DIAN FANG FEI DFF TELEX RELEASE FEE'),
  ('SEAL','封条费','Seal Fee','封志费','BOOKING','CNY','CONT',NULL,6.00,'海运代理订舱服务',6.00,1230,'封条费 FENGTIAOFEI FENG TIAO FEI FTF SEAL FEE 封志费 FENGZHIFEI FENG ZHI FEI FZF'),
  ('EIR','打单费','Equipment Interchange Receipt','设备交接单费','BOOKING','CNY','CONT',NULL,6.00,'海运代理订舱服务',6.00,1240,'打单费 DADANFEI DA DAN FEI DDF EQUIPMENT INTERCHANGE RECEIPT 设备交接单费 SHEBEIJIAOJIEDANFEI SHE BEI JIAO JIE DAN FEI SBJJDF'),
  ('GDF','改单费','Amendment Fee',NULL,'DOC_PREP','EUR','BL',NULL,0.00,'海运代理订舱服务',6.00,1250,'改单费 GAIDANFEI GAI DAN FEI GDF AMENDMENT FEE'),
  ('LDF','联单费','Combined Documentation Fee',NULL,'DOC_PREP','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1260,'联单费 LIANDANFEI LIAN DAN FEI LDF COMBINED DOCUMENTATION FEE'),
  ('TDF','调单费','Document Retrieval Fee',NULL,'DOC_PREP','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,1270,'调单费 DIAODANFEI DIAO DAN FEI DDF DOCUMENT RETRIEVAL FEE'),
  ('KDF','快递费','Courier Fee',NULL,'DOC_PREP','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,1280,'快递费 KUAIDIFEI KUAI DI FEI KDF COURIER FEE'),
  ('HDF','换单费','Delivery Order Fee','换单服务','DOCUMENT_EXCHANGE','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,1410,'换单费 HUANDANFEI HUAN DAN FEI HDF DELIVERY ORDER FEE 换单服务 HUANDANFUWU HUAN DAN FU WU HDFW'),
  ('HKG','香港申报费','Hong Kong Declaration Fee',NULL,'DOCUMENT_EXCHANGE','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,1420,'香港申报费 XIANGGANGSHENBAOFEI XIANG GANG SHEN BAO FEI XGSBF HONG KONG DECLARATION FEE'),
  ('CUSTOMS','代理报关费','Customs Declaration Fee',NULL,'CUSTOMS_EXPORT','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,1610,'代理报关费 DAILIBAOGUANFEI DAI LI BAO GUAN FEI DLBGF CUSTOMS DECLARATION FEE'),
  ('BGF','报关费（无票）','Customs Declaration Fee (No Invoice)','报关费','CUSTOMS_EXPORT','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1620,'报关费（无票） BAOGUANFEIWUPIAO BAO GUAN FEI WU PIAO BGFWP CUSTOMS DECLARATION FEE (NO INVOICE) 报关费 BAOGUANFEI BAO GUAN FEI BGF'),
  ('HZQD','核注清单费','Bonded List Filing Fee',NULL,'CUSTOMS_EXPORT','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,1630,'核注清单费 HEZHUQINGDANFEI HE ZHU QING DAN FEI HZQDF BONDED LIST FILING FEE'),
  ('CYF','查验费','Customs Inspection Fee','查验服务','CUSTOMS_EXPORT','CNY','BL','ABN-INSPECTION',0.00,'报关报检代理服务',6.00,1640,'查验费 CHAYANFEI CHA YAN FEI CYF CUSTOMS INSPECTION FEE 查验服务 CHAYANFUWU CHA YAN FU WU CYFW'),
  ('HGFK','海关罚款','Customs Penalty',NULL,'CUSTOMS_EXPORT','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1650,'海关罚款 HAIGUANFAKUAN HAI GUAN FA KUAN HGFK CUSTOMS PENALTY'),
  ('ZBJ','滞报金','Late Declaration Penalty',NULL,'CUSTOMS_EXPORT','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1660,'滞报金 ZHIBAOJIN ZHI BAO JIN ZBJ LATE DECLARATION PENALTY'),
  ('MD','买单','Export Doc Purchase',NULL,'DOC_BUY','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1670,'买单 MAIDAN MAI DAN MD EXPORT DOC PURCHASE'),
  ('T1','T1 转关费（欧盟）','EU T1 Transit Fee',NULL,'CUSTOMS_IMPORT','EUR','BL',NULL,0.00,'报关报检代理服务',6.00,1680,'T1 转关费（欧盟） ZHUANGUANFEIOUMENG ZHUAN GUAN FEI OU MENG ZGFOM EU T1 TRANSIT FEE'),
  ('DUTY','关税','Customs Duty',NULL,'CUSTOMS_IMPORT','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1690,'关税 GUANSHUI GUAN SHUI GS CUSTOMS DUTY'),
  ('BJF','报检费','Inspection Declaration Fee',NULL,'INSPECTION','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,1810,'报检费 BAOJIANFEI BAO JIAN FEI BJF INSPECTION DECLARATION FEE'),
  ('SJF','商检费','Commodity Inspection Fee',NULL,'INSPECTION','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1820,'商检费 SHANGJIANFEI SHANG JIAN FEI SJF COMMODITY INSPECTION FEE'),
  ('JYF','检验费','Inspection & Testing Fee',NULL,'INSPECTION','USD','BL',NULL,0.00,'报关报检代理服务',6.00,1830,'检验费 JIANYANFEI JIAN YAN FEI JYF INSPECTION & TESTING FEE'),
  ('JD','鉴定费（6%）','Appraisal Fee (6%)',NULL,'INSPECTION','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,1840,'鉴定费（6%） JIANDINGFEI JIAN DING FEI JDF APPRAISAL FEE (6%)'),
  ('JDF','鉴定费','Appraisal Fee',NULL,'INSPECTION','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1850,'鉴定费 JIANDINGFEI JIAN DING FEI JDF APPRAISAL FEE'),
  ('XDF','消毒费','Disinfection Fee',NULL,'INSPECTION','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1860,'消毒费 XIAODUFEI XIAO DU FEI XDF DISINFECTION FEE'),
  ('YDF','验电费','Electrical Inspection Fee',NULL,'INSPECTION','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,1870,'验电费 YANDIANFEI YAN DIAN FEI YDF ELECTRICAL INSPECTION FEE'),
  ('COO','代办产地证','Certificate of Origin',NULL,'CERTIFICATE','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,2010,'代办产地证 DAIBANCHANDIZHENG DAI BAN CHAN DI ZHENG DBCDZ CERTIFICATE OF ORIGIN'),
  ('DJF','登记费','Registration Fee',NULL,'CERTIFICATE','USD','BL',NULL,0.00,'报关报检代理服务',6.00,2020,'登记费 DENGJIFEI DENG JI FEI DJF REGISTRATION FEE'),
  ('XZF','熏蒸费','Fumigation Fee',NULL,'FUMIGATION','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,2030,'熏蒸费 XUNZHENGFEI XUN ZHENG FEI XZF FUMIGATION FEE'),
  ('BXF','保险费','Insurance Fee',NULL,'INSURANCE','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,2040,'保险费 BAOXIANFEI BAO XIAN FEI BXF INSURANCE FEE'),
  ('BXF6','保险费（6%）','Insurance Fee (6%)',NULL,'INSURANCE','CNY','BL',NULL,6.00,'海运代理订舱服务',6.00,2050,'保险费（6%） BAOXIANFEI BAO XIAN FEI BXF INSURANCE FEE (6%)'),
  ('TRUCK','集装箱拖车费','Container Trucking Fee','拖车费','TRUCKING','CNY','CONT',NULL,9.00,'集装箱陆路运输服务',9.00,2210,'集装箱拖车费 JIZHUANGXIANGTUOCHEFEI JI ZHUANG XIANG TUO CHE FEI JZXTCF CONTAINER TRUCKING FEE 拖车费 TUOCHEFEI TUO CHE FEI TCF'),
  ('YF9','运费（9%）','Inland Freight (9%)',NULL,'TRUCKING','CNY','BL',NULL,9.00,'集装箱陆路运输服务',9.00,2220,'运费（9%） YUNFEI YUN FEI YF INLAND FREIGHT (9%)'),
  ('YF6','运费（6%）','Inland Freight (6%)',NULL,'TRUCKING','CNY','BL',NULL,6.00,'集装箱陆路运输服务',9.00,2230,'运费（6%） YUNFEI YUN FEI YF INLAND FREIGHT (6%)'),
  ('PY','运费（普票 6%）','Inland Freight (General Invoice)',NULL,'TRUCKING','CNY','BL',NULL,6.00,'集装箱陆路运输服务',9.00,2240,'运费（普票 6%） YUNFEIPUPIAO YUN FEI PU PIAO YFPP INLAND FREIGHT (GENERAL INVOICE)'),
  ('KCF9','卡车费（9%）','Trucking Fee (9%)',NULL,'TRUCKING','CNY','BL',NULL,9.00,'集装箱陆路运输服务',9.00,2250,'卡车费（9%） KACHEFEI KA CHE FEI KCF TRUCKING FEE (9%)'),
  ('GNYDL6','国内运输代理费（6%）','Domestic Forwarding Fee (6%)',NULL,'TRUCKING','CNY','BL',NULL,6.00,'集装箱陆路运输服务',9.00,2260,'国内运输代理费（6%） GUONEIYUNSHUDAILIFEI GUO NEI YUN SHU DAI LI FEI GNYSDLF DOMESTIC FORWARDING FEE (6%)'),
  ('SHF','送货费','Delivery Fee',NULL,'TRUCKING','CNY','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2270,'送货费 SONGHUOFEI SONG HUO FEI SHF DELIVERY FEE'),
  ('SHF9','送货费（9%）','Delivery Fee (9%)',NULL,'TRUCKING','CNY','BL',NULL,9.00,'集装箱陆路运输服务',9.00,2280,'送货费（9%） SONGHUOFEI SONG HUO FEI SHF DELIVERY FEE (9%)'),
  ('THF','提货费','Pick-up Fee',NULL,'TRUCKING','CNY','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2290,'提货费 TIHUOFEI TI HUO FEI THF PICK-UP FEE'),
  ('P','派送费','Delivery Fee (Domestic)',NULL,'TRUCKING','CNY','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2300,'派送费 PAISONGFEI PAI SONG FEI PSF DELIVERY FEE (DOMESTIC)'),
  ('RFT','铁路运费','Rail Freight',NULL,'TRUCKING','USD','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2310,'铁路运费 TIELUYUNFEI TIE LU YUN FEI TLYF RAIL FREIGHT'),
  ('DSF','待时费','Waiting Time Fee',NULL,'TRUCKING','CNY','BL','ABN-WAITING',0.00,'集装箱陆路运输服务',9.00,2320,'待时费 DAISHIFEI DAI SHI FEI DSF WAITING TIME FEE'),
  ('YYF','压夜费','Overnight Detention Fee',NULL,'TRUCKING','CNY','BL','ABN-OVERNIGHT',0.00,'集装箱陆路运输服务',9.00,2330,'压夜费 YAYEFEI YA YE FEI YYF OVERNIGHT DETENTION FEE'),
  ('YCF','压车费','Vehicle Detention Fee',NULL,'TRUCKING','CNY','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2340,'压车费 YACHEFEI YA CHE FEI YCF VEHICLE DETENTION FEE'),
  ('GG','过港费','Cross-Harbor Transfer Fee',NULL,'TRUCKING','CNY','BL',NULL,0.00,'集装箱陆路运输服务',9.00,2350,'过港费 GUOGANGFEI GUO GANG FEI GGF CROSS-HARBOR TRANSFER FEE'),
  ('THC','码头操作费','Terminal Handling Charge',NULL,'PORT_OPS','CNY','CONT',NULL,6.00,'港口操作及港杂服务',6.00,2410,'码头操作费 MATOUCAOZUOFEI MA TOU CAO ZUO FEI MTCZF TERMINAL HANDLING CHARGE'),
  ('STORAGE','码头堆存费','Port Storage Fee','堆存费','PORT_OPS','CNY','CONT',NULL,6.00,'港口操作及港杂服务',6.00,2420,'码头堆存费 MATOUDUICUNFEI MA TOU DUI CUN FEI MTDCF PORT STORAGE FEE 堆存费 DUICUNFEI DUI CUN FEI DCF'),
  ('GZF','港杂费','Port Miscellaneous Fee',NULL,'PORT_OPS','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2430,'港杂费 GANGZAFEI GANG ZA FEI GZF PORT MISCELLANEOUS FEE'),
  ('GZF6','港杂费（6%）','Port Miscellaneous Fee (6%)',NULL,'PORT_OPS','CNY','BL',NULL,6.00,'港口操作及港杂服务',6.00,2440,'港杂费（6%） GANGZAFEI GANG ZA FEI GZF PORT MISCELLANEOUS FEE (6%)'),
  ('DMF6','地面服务费（6%）','Ground Handling Fee (6%)','地面操作费','PORT_OPS','CNY','BL',NULL,6.00,'港口操作及港杂服务',6.00,2450,'地面服务费（6%） DIMIANFUWUFEI DI MIAN FU WU FEI DMFWF GROUND HANDLING FEE (6%) 地面操作费 DIMIANCAOZUOFEI DI MIAN CAO ZUO FEI DMCZF'),
  ('CZF','操作费','Local Handling Fee',NULL,'PORT_OPS','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2460,'操作费 CAOZUOFEI CAO ZUO FEI CZF LOCAL HANDLING FEE'),
  ('ZX','装卸费','Loading/Unloading Fee',NULL,'PORT_OPS','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2470,'装卸费 ZHUANGXIEFEI ZHUANG XIE FEI ZXF LOADING/UNLOADING FEE'),
  ('TSC','特殊操作费','Special Handling Fee',NULL,'PORT_OPS','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2480,'特殊操作费 TESHUCAOZUOFEI TE SHU CAO ZUO FEI TSCZF SPECIAL HANDLING FEE'),
  ('JBF','加班费','Overtime Fee',NULL,'PORT_OPS','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2490,'加班费 JIABANFEI JIA BAN FEI JBF OVERTIME FEE'),
  ('RGF','人工费','Labor Fee',NULL,'PORT_OPS','USD','BL',NULL,0.00,'港口操作及港杂服务',6.00,2500,'人工费 RENGONGFEI REN GONG FEI RGF LABOR FEE'),
  ('CCF','仓储费','Warehouse Storage Fee',NULL,'WAREHOUSING','CNY','DAY',NULL,0.00,'港口操作及港杂服务',6.00,2610,'仓储费 CANGCHUFEI CANG CHU FEI CCF WAREHOUSE STORAGE FEE'),
  ('CCF6','仓储费（6%）','Warehouse Storage Fee (6%)',NULL,'WAREHOUSING','CNY','DAY',NULL,6.00,'港口操作及港杂服务',6.00,2620,'仓储费（6%） CANGCHUFEI CANG CHU FEI CCF WAREHOUSE STORAGE FEE (6%)'),
  ('DDCCF','代垫仓储费','Advance Storage Fee',NULL,'WAREHOUSING','CNY','DAY',NULL,0.00,'港口操作及港杂服务',6.00,2630,'代垫仓储费 DAIDIANCANGCHUFEI DAI DIAN CANG CHU FEI DDCCF ADVANCE STORAGE FEE'),
  ('DDCCF6','代垫仓储费（6%）','Advance Storage Fee (6%)',NULL,'WAREHOUSING','CNY','DAY',NULL,6.00,'港口操作及港杂服务',6.00,2640,'代垫仓储费（6%） DAIDIANCANGCHUFEI DAI DIAN CANG CHU FEI DDCCF ADVANCE STORAGE FEE (6%)'),
  ('JCF','进仓费','Inbound Warehousing Fee',NULL,'WAREHOUSING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2650,'进仓费 JINCANGFEI JIN CANG FEI JCF INBOUND WAREHOUSING FEE'),
  ('CRKF','出入库费','In/Out Warehousing Fee',NULL,'WAREHOUSING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2660,'出入库费 CHURUKUFEI CHU RU KU FEI CRKF IN/OUT WAREHOUSING FEE'),
  ('ZZF','转栈费','Yard Transfer Fee',NULL,'WAREHOUSING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2670,'转栈费 ZHUANZHANFEI ZHUAN ZHAN FEI ZZF YARD TRANSFER FEE'),
  ('TK','退库费','Warehouse Return Fee',NULL,'WAREHOUSING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2680,'退库费 TUIKUFEI TUI KU FEI TKF WAREHOUSE RETURN FEE'),
  ('XXF','洗箱费','Container Cleaning Fee',NULL,'WAREHOUSING','CNY','CONT',NULL,0.00,'港口操作及港杂服务',6.00,2690,'洗箱费 XIXIANGFEI XI XIANG FEI XXF CONTAINER CLEANING FEE'),
  ('REP','修箱费','Container Repair Fee',NULL,'WAREHOUSING','CNY','CONT',NULL,0.00,'港口操作及港杂服务',6.00,2700,'修箱费 XIUXIANGFEI XIU XIANG FEI XXF CONTAINER REPAIR FEE'),
  ('ZXF','装箱费','Stuffing Fee',NULL,'STUFFING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2810,'装箱费 ZHUANGXIANGFEI ZHUANG XIANG FEI ZXF STUFFING FEE'),
  ('TXF','掏箱费','Devanning Fee',NULL,'STUFFING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2820,'掏箱费 TAOXIANGFEI TAO XIANG FEI TXF DEVANNING FEE'),
  ('KXF','开箱费','Container Opening Fee',NULL,'STUFFING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2830,'开箱费 KAIXIANGFEI KAI XIANG FEI KXF CONTAINER OPENING FEE'),
  ('FXF','放箱费','Container Release Fee',NULL,'STUFFING','CNY','BL',NULL,0.00,'港口操作及港杂服务',6.00,2840,'放箱费 FANGXIANGFEI FANG XIANG FEI FXF CONTAINER RELEASE FEE'),
  ('JHTP','交换托盘','Pallet Exchange Fee',NULL,'STUFFING','USD','BL',NULL,0.00,'港口操作及港杂服务',6.00,2850,'交换托盘 JIAOHUANTUOPAN JIAO HUAN TUO PAN JHTP PALLET EXCHANGE FEE'),
  ('KYF','空运费','Air Freight',NULL,'AIR','CNY','BL',NULL,0.00,'国际航空运输服务',0.00,3010,'空运费 KONGYUNFEI KONG YUN FEI KYF AIR FREIGHT'),
  ('AWB','运单费','Air Waybill Fee',NULL,'AIR','EUR','BL',NULL,0.00,'国际航空运输服务',0.00,3020,'运单费 YUNDANFEI YUN DAN FEI YDF AIR WAYBILL FEE'),
  ('HZF','航站费','Airport Terminal Fee',NULL,'AIR','USD','BL',NULL,0.00,'国际航空运输服务',0.00,3030,'航站费 HANGZHANFEI HANG ZHAN FEI HZF AIRPORT TERMINAL FEE'),
  ('CSF','安检费','Security Screening Fee',NULL,'AIR','EUR','BL',NULL,0.00,'国际航空运输服务',0.00,3040,'安检费 ANJIANFEI AN JIAN FEI AJF SECURITY SCREENING FEE'),
  ('AMF','机场快递费','Airport Messenger Fee',NULL,'AIR','EUR','BL',NULL,0.00,'国际航空运输服务',0.00,3050,'机场快递费 JICHANGKUAIDIFEI JI CHANG KUAI DI FEI JCKDF AIRPORT MESSENGER FEE'),
  ('AF','A/F 费','A/F Fee',NULL,'AIR','EUR','BL',NULL,0.00,'国际航空运输服务',0.00,3060,'A/F 费 FEI F A/F FEE'),
  ('ATB','ATB 费','ATB Fee',NULL,'AIR','CNY','BL',NULL,0.00,'国际航空运输服务',0.00,3070,'ATB 费 FEI F ATB FEE'),
  ('CLC','箱板费','Air Pallet Fee',NULL,'PALLET_CHARTER','USD','BL',NULL,0.00,'国际航空运输服务',0.00,3080,'箱板费 XIANGBANFEI XIANG BAN FEI XBF AIR PALLET FEE'),
  ('DTHC','目的港码头操作费','Destination THC',NULL,'OVERSEA_SEGMENT','USD','BL',NULL,0.00,'国际海运运费',0.00,3210,'目的港码头操作费 MUDEGANGMATOUCAOZUOFEI MU DE GANG MA TOU CAO ZUO FEI MDGMTCZF DESTINATION THC'),
  ('ETS','ETS 附加费','EU Emission Trading Surcharge',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3220,'ETS 附加费 FUJIAFEI FU JIA FEI FJF EU EMISSION TRADING SURCHARGE'),
  ('PU','提货费（海外）','Pick-up Fee',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3230,'提货费（海外） TIHUOFEIHAIWAI TI HUO FEI HAI WAI THFHW PICK-UP FEE'),
  ('HDL','操作费（海外）','Handling Fee',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3240,'操作费（海外） CAOZUOFEIHAIWAI CAO ZUO FEI HAI WAI CZFHW HANDLING FEE'),
  ('DLV','派送费（海外）','Delivery Fee',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3250,'派送费（海外） PAISONGFEIHAIWAI PAI SONG FEI HAI WAI PSFHW DELIVERY FEE'),
  ('CMP','合规查验费','Compliance Check Fee',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3260,'合规查验费 HEGUICHAYANFEI HE GUI CHA YAN FEI HGCYF COMPLIANCE CHECK FEE'),
  ('ATLAS','ATLAS 系统费','ATLAS Fee',NULL,'OVERSEA_SEGMENT','EUR','BL',NULL,0.00,'国际海运运费',0.00,3270,'ATLAS 系统费 XITONGFEI XI TONG FEI XTF ATLAS FEE'),
  ('GENEST','GENEST 费','GENEST Fee',NULL,'OVERSEA_SEGMENT','CNY','BL',NULL,0.00,'国际海运运费',0.00,3280,'GENEST 费 FEI F GENEST FEE'),
  ('MYD','贸易代理费','Trading Agent Fee',NULL,'OVERSEA_SEGMENT','USD','BL',NULL,0.00,'国际海运运费',0.00,3290,'贸易代理费 MAOYIDAILIFEI MAO YI DAI LI FEI MYDLF TRADING AGENT FEE'),
  ('GJDLF','国际货运代理费','International Freight Forwarding Fee',NULL,'OVERSEA_SEGMENT','CNY','BL',NULL,0.00,'海运代理订舱服务',6.00,3300,'国际货运代理费 GUOJIHUOYUNDAILIFEI GUO JI HUO YUN DAI LI FEI GJHYDLF INTERNATIONAL FREIGHT FORWARDING FEE'),
  ('SJ','税金','Taxes & Duties','税款','TAX','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,3410,'税金 SHUIJIN SHUI JIN SJ TAXES & DUTIES 税款 SHUIKUAN SHUI KUAN SK'),
  ('DDSJ','代垫税金（12%）','Advance Tax (12%)',NULL,'TAX','CNY','BL',NULL,12.00,'报关报检代理服务',6.00,3420,'代垫税金（12%） DAIDIANSHUIJIN DAI DIAN SHUI JIN DDSJ ADVANCE TAX (12%)'),
  ('ZZS9','增值税（9%）','VAT (9%)',NULL,'TAX','CNY','BL',NULL,9.00,'报关报检代理服务',6.00,3430,'增值税（9%） ZENGZHISHUI ZENG ZHI SHUI ZZS VAT (9%)'),
  ('ZZS6','增值税（6%）','VAT (6%)',NULL,'TAX','CNY','BL',NULL,6.00,'报关报检代理服务',6.00,3440,'增值税（6%） ZENGZHISHUI ZENG ZHI SHUI ZZS VAT (6%)'),
  ('ZZS3','增值税（9%-6%补差）','VAT Differential (9%-6%)',NULL,'TAX','CNY','BL',NULL,3.00,'报关报检代理服务',6.00,3450,'增值税（9%-6%补差） ZENGZHISHUIBUCHA ZENG ZHI SHUI BU CHA ZZSBC VAT DIFFERENTIAL (9%-6%)'),
  ('JXS','进项税','Input VAT',NULL,'TAX','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,3460,'进项税 JINXIANGSHUI JIN XIANG SHUI JXS INPUT VAT'),
  ('XXZZS','销项增值税','Output VAT',NULL,'TAX','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,3470,'销项增值税 XIAOXIANGZENGZHISHUI XIAO XIANG ZENG ZHI SHUI XXZZS OUTPUT VAT'),
  ('YHS','印花税','Stamp Duty',NULL,'TAX','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3480,'印花税 YINHUASHUI YIN HUA SHUI YHS STAMP DUTY'),
  ('QYSDS','企业所得税','Corporate Income Tax',NULL,'TAX','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3490,'企业所得税 QIYESUODESHUI QI YE SUO DE SHUI QYSDS CORPORATE INCOME TAX'),
  ('GST','商品服务税','GST',NULL,'TAX','SGD','BL',NULL,0.00,'经纪代理服务',6.00,3500,'商品服务税 SHANGPINFUWUSHUI SHANG PIN FU WU SHUI SPFWS GST'),
  ('ZNJ','滞纳金','Late Payment Surcharge',NULL,'TAX','CNY','BL',NULL,0.00,'报关报检代理服务',6.00,3510,'滞纳金 ZHINAJIN ZHI NA JIN ZNJ LATE PAYMENT SURCHARGE'),
  ('TZ','账务调整','Account Adjustment','财务调账','FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3610,'账务调整 ZHANGWUDIAOZHENG ZHANG WU DIAO ZHENG ZWDZ ACCOUNT ADJUSTMENT 财务调账 CAIWUDIAOZHANG CAI WU DIAO ZHANG CWDZ'),
  ('HDSY','汇兑损益','FX Gain/Loss',NULL,'FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3620,'汇兑损益 HUIDUISUNYI HUI DUI SUN YI HDSY FX GAIN/LOSS'),
  ('CDHP','汇票承兑手续费（无票）','Bank Acceptance Fee (No Invoice)',NULL,'FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3630,'汇票承兑手续费（无票） HUIPIAOCHENGDUISHOUXUFEIWUPIAO HUI PIAO CHENG DUI SHOU XU FEI WU PIAO HPCDSXFWP BANK ACCEPTANCE FEE (NO INVOICE)'),
  ('XJZC','现金支出','Cash Disbursement',NULL,'FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3640,'现金支出 XIANJINZHICHU XIAN JIN ZHI CHU XJZC CASH DISBURSEMENT'),
  ('LRC','利润分成','Profit Share','PROFIT SHARE','FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3650,'利润分成 LIRUNFENCHENG LI RUN FEN CHENG LRFC PROFIT SHARE'),
  ('YFK','预付款','Advance Payment',NULL,'FIN_ADJ','USD','BL',NULL,0.00,'经纪代理服务',6.00,3660,'预付款 YUFUKUAN YU FU KUAN YFK ADVANCE PAYMENT'),
  ('BZJ','保证金','Deposit',NULL,'FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3670,'保证金 BAOZHENGJIN BAO ZHENG JIN BZJ DEPOSIT'),
  ('DDHK','代垫货款','Advance Cargo Payment',NULL,'FIN_ADJ','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3680,'代垫货款 DAIDIANHUOKUAN DAI DIAN HUO KUAN DDHK ADVANCE CARGO PAYMENT'),
  ('QT','其他','Miscellaneous',NULL,'MISC','CNY','BL',NULL,0.00,'经纪代理服务',6.00,3690,'其他 QITA QI TA QT MISCELLANEOUS')
) AS v(fee_code,name_zh,name_en,alias_name,category_code,currency,unit_code,abnormal_code,tax_rate,service_name,service_rate,sort_order,search_keywords)
ON CONFLICT ("fee_code") DO NOTHING;

-- 4.1 引用完整性 fail-fast：类别/单位/异常引用缺失时给出明确报错。
DO $$
DECLARE missing text;
BEGIN
  SELECT string_agg(v.category_code,',') INTO missing FROM (VALUES ('BOOKING'::varchar),('DOC_PREP'::varchar),('DOCUMENT_EXCHANGE'::varchar),('CUSTOMS_EXPORT'::varchar),('DOC_BUY'::varchar),('CUSTOMS_IMPORT'::varchar),('INSPECTION'::varchar),('CERTIFICATE'::varchar),('FUMIGATION'::varchar),('INSURANCE'::varchar),('TRUCKING'::varchar),('PORT_OPS'::varchar),('WAREHOUSING'::varchar),('STUFFING'::varchar),('AIR'::varchar),('PALLET_CHARTER'::varchar),('OVERSEA_SEGMENT'::varchar),('TAX'::varchar),('FIN_ADJ'::varchar),('MISC'::varchar)) v(category_code)
  WHERE NOT EXISTS (SELECT 1 FROM "master_data_items" m WHERE m."kind"='charge_category' AND m."code"=v.category_code);
  IF missing IS NOT NULL THEN RAISE EXCEPTION '费用模板引用的费用类别缺失: %',missing; END IF;
  SELECT string_agg(v.unit_code,',') INTO missing FROM (VALUES ('CONT'::varchar),('BL'::varchar),('DAY'::varchar)) v(unit_code)
  WHERE NOT EXISTS (SELECT 1 FROM "billing_units" u WHERE u."code"=v.unit_code);
  IF missing IS NOT NULL THEN RAISE EXCEPTION '费用模板引用的计费单位缺失: %',missing; END IF;
  SELECT string_agg(v.abnormal_code,',') INTO missing FROM (VALUES ('ABN-CUSTOMER-CANCEL'::varchar),('ABN-INSPECTION'::varchar),('ABN-WAITING'::varchar),('ABN-OVERNIGHT'::varchar)) v(abnormal_code)
  WHERE NOT EXISTS (SELECT 1 FROM "master_data_items" m WHERE m."kind"='abnormal_case' AND m."code"=v.abnormal_code);
  IF missing IS NOT NULL THEN RAISE EXCEPTION '费用模板引用的异常情况缺失: %',missing; END IF;
END $$;

-- 5. 既有模板 THC/STORAGE 类别改挂港口操作（仅模板行；公司副本不随模板变化）。
UPDATE "fee_setting_templates"
SET "charge_category_id"=(SELECT m."id" FROM "master_data_items" m WHERE m."kind"='charge_category' AND m."code"='PORT_OPS'),"updated_at"=CURRENT_TIMESTAMP
WHERE "fee_code" IN ('THC','STORAGE');

-- 6. 既有公司追加：同码跳过；应税劳务按（公司,名称）找不到则按模板文本创建，
--    同名但税率/简称/商品码不一致时显式拒绝（口径与 initializeCompanyFeeSettings 一致）。
DO $$
DECLARE tpl record; tax taxable_services%ROWTYPE; target_tax uuid; company record;
BEGIN
  FOR company IN SELECT "id" FROM "organizations" WHERE "kind"='company' ORDER BY "id" LOOP
    FOR tpl IN SELECT * FROM "fee_setting_templates" WHERE "enabled" ORDER BY "id" LOOP
      IF EXISTS (SELECT 1 FROM "fee_settings" f WHERE f."organization_id"=company."id" AND f."fee_code"=tpl."fee_code") THEN
        CONTINUE;
      END IF;
      SELECT * INTO tax FROM "taxable_services" t WHERE t."organization_id"=company."id" AND t."name"=tpl."taxable_service_name";
      IF NOT FOUND THEN
        target_tax:=gen_random_uuid();
        INSERT INTO "taxable_services"("id","created_at","updated_at","organization_id","name","short_name","goods_code","default_tax_rate","enabled","search_keywords")
        VALUES (target_tax,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,company."id",tpl."taxable_service_name",NULL,NULL,tpl."taxable_service_default_tax_rate",true,
          COALESCE((SELECT s.kw FROM (VALUES ('国际海运运费','国际海运运费 GUOJIHAIYUNYUNFEI GUO JI HAI YUN YUN FEI GJHYYF'),('国际航空运输服务','国际航空运输服务 GUOJIHANGKONGYUNSHUFUWU GUO JI HANG KONG YUN SHU FU WU GJHKYSFW'),('报关报检代理服务','报关报检代理服务 BAOGUANBAOJIANDAILIFUWU BAO GUAN BAO JIAN DAI LI FU WU BGBJDLFW'),('海运代理订舱服务','海运代理订舱服务 HAIYUNDAILIDINGCANGFUWU HAI YUN DAI LI DING CANG FU WU HYDLDCFW'),('港口操作及港杂服务','港口操作及港杂服务 GANGKOUCAOZUOJIGANGZAFUWU GANG KOU CAO ZUO JI GANG ZA FU WU GKCZJGZFW'),('经纪代理服务','经纪代理服务 JINGJIDAILIFUWU JING JI DAI LI FU WU JJDLFW'),('集装箱陆路运输服务','集装箱陆路运输服务 JIZHUANGXIANGLULUYUNSHUFUWU JI ZHUANG XIANG LU LU YUN SHU FU WU JZXLLYSFW')) AS s(name,kw) WHERE s.name=tpl."taxable_service_name"),tpl."taxable_service_name"));
      ELSE
        IF tax."default_tax_rate" IS DISTINCT FROM tpl."taxable_service_default_tax_rate" OR tax."short_name" IS DISTINCT FROM tpl."taxable_service_short_name" OR tax."goods_code" IS DISTINCT FROM tpl."taxable_service_goods_code" THEN
          RAISE EXCEPTION '公司费用追加遇同名税务配置冲突: company=%, service=%',company."id",tpl."taxable_service_name";
        END IF;
        target_tax:=tax."id";
      END IF;
      INSERT INTO "fee_settings"("id","created_at","updated_at","organization_id","fee_code","name_zh","name_en","alias_name","charge_category_id","default_currency","billing_unit_id","abnormal_case_id","tax_rate","taxable_service_id","enabled","sort_order","search_keywords")
      VALUES (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,company."id",tpl."fee_code",tpl."name_zh",tpl."name_en",tpl."alias_name",tpl."charge_category_id",tpl."default_currency",tpl."billing_unit_id",tpl."abnormal_case_id",tpl."tax_rate",target_tax,true,tpl."sort_order",tpl."search_keywords");
    END LOOP;
  END LOOP;
END $$;
