-- 异常情况主数据种子补全：漏装/甩箱/滞箱/滞港/扣货/40NOR 等海运出口常见异常。
-- 幂等：同码已存在的行（含业务人员后续改名、停用）保持原样，不覆盖。
-- search_keywords 与 searchtext.Build(name, name_en) 输出一致，由一次性生成器预计算。
INSERT INTO "master_data_items" ("id","created_at","updated_at","kind","code","name","name_en","source","sort_order","enabled","attributes","search_keywords")
VALUES
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-MISSED-LOAD','漏装','Missed Loading','system',60,true,'{}'::jsonb,'漏装 LOUZHUANG LOU ZHUANG LZ MISSED LOADING'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-ROLLED','甩箱','Container Rolled','system',70,true,'{}'::jsonb,'甩箱 SHUAIXIANG SHUAI XIANG SX CONTAINER ROLLED'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-DETENTION','滞箱','Container Detention','system',80,true,'{}'::jsonb,'滞箱 ZHIXIANG ZHI XIANG ZX CONTAINER DETENTION'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-DEMURRAGE','滞港','Port Demurrage','system',90,true,'{}'::jsonb,'滞港 ZHIGANG ZHI GANG ZG PORT DEMURRAGE'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-CARGO-HELD','扣货','Cargo Held','system',100,true,'{}'::jsonb,'扣货 KOUHUO KOU HUO KH CARGO HELD'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-40NOR','40NOR冷代干','40ft Non-Operating Reefer','system',110,true,'{}'::jsonb,'40NOR冷代干 LENGDAIGAN LENG DAI GAN LDG 40FT NON-OPERATING REEFER'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-VESSEL-DELAY','船期延误','Vessel Delay','system',120,true,'{}'::jsonb,'船期延误 CHUANQIYANWU CHUAN QI YAN WU CQYW VESSEL DELAY'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-SHUT-OUT','退关','Export Withdrawal','system',130,true,'{}'::jsonb,'退关 TUIGUAN TUI GUAN TG EXPORT WITHDRAWAL'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-DAMAGED-BOX','坏柜','Damaged Container','system',140,true,'{}'::jsonb,'坏柜 HUAIGUI HUAI GUI HG DAMAGED CONTAINER'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-CARGO-DAMAGE','货损','Cargo Damage','system',150,true,'{}'::jsonb,'货损 HUOSUN HUO SUN HS CARGO DAMAGE'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-CARGO-SHORTAGE','货差','Cargo Shortage','system',160,true,'{}'::jsonb,'货差 HUOCHA HUO CHA HC CARGO SHORTAGE'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-WET-DAMAGE','水湿','Water Damage','system',170,true,'{}'::jsonb,'水湿 SHUISHI SHUI SHI SS WATER DAMAGE'),
  (gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'abnormal_case','ABN-SEAL-ISSUE','铅封异常','Seal Discrepancy','system',180,true,'{}'::jsonb,'铅封异常 QIANFENGYICHANG QIAN FENG YI CHANG QFYC SEAL DISCREPANCY')
ON CONFLICT ("kind","code") DO NOTHING;
