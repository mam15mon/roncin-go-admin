# 迁移冲突报告（阶段一：存储地基 + 契约）

任务：主数据多组织治理口径统一 —— 阶段一存储三型改造。
迁移文件：

- `server/migrations/20260914220000_master_data_storage_governance.sql`（主数据存储三型改造，2026-09-14 22:26 已应用，随后文件微调经 `pnpm run migrate:dev` 重录校验和）
- `server/migrations/20260914231000_exchange_rate_weekly_dual_rate.sql`（汇率周结构 + 双轨点差列，2026-09-14 已应用）

迁移全程无阻断性冲突（fail-fast 检测未触发 EXCEPTION 中止），以下为各步骤处置结论与终态核验。

## 1. master_data_items（A 型：kind 更名 + 去组织）

- 步骤 1.0 清理重种子残留（同组织同码并存 charge_category / service_type 行）：先保留 charge_category 行。
- 步骤 1.1 `service_type` → `charge_category` 数据更名完成。
- 步骤 1.1.1（验证阶段补齐）`master_data_items_kind_check` 检查约束同步重建为新枚举集：
  初版迁移漏更新该约束（开发库因历史 AutoMigrate 无此约束而被掩盖；冷启动 Schema 上
  插入 charge_category 会触发 23514 违约，由集成测试暴露），已补 `DROP CONSTRAINT IF EXISTS`
  + `ADD CONSTRAINT ... CHECK (kind IN ('currency','country','region','container_spec',
  'charge_category','cargo_category','abnormal_case'))`，并手工补执行到开发库。
- 步骤 1.2/1.3 同 (kind, code) 重复行按「总部根组织行 > 最早创建」保留一行，8 张引用表
  （order_service_types、order_cargo_categories、order_abnormal_cases、order_containers、
  order_container_requests、sea_shared_containers、fee_settings.service_type_id /
  abnormal_case_id）的逻辑引用先重指到保留行，再删除冗余行。
- 终态：101 行，重复 (kind, code) 为 0；业务码全局唯一索引 `masterdataitem_kind_code` 生效。
- 双轨收敛：currency / region kind 不再播种（币种走全局 `currencies` 表、区划走
  `administrative_regions` 表），开发库中相应 kind 已无数据行。

## 2. airlines / shipping_lines / 前缀表 / billing_units（A 型收口，Fail-fast）

处置策略（执行时版本）：同业务码多行按「总部根组织行 > 最早创建」保留一行，其余行
删除并逐行 `RAISE NOTICE` 记录（迁移输出）；不静默合并。开发数据处置已获用户授权
（prd.md §8.5）。

检查代理复核后，迁移文件已按 design §8 第 2 步修订为真 fail-fast：同业务码多行冲突
改为 `RAISE EXCEPTION` 阻止迁移（冲突数 + 样例码入异常消息），人工处置后重跑；不再
自动保留/删除。该修订不影响本次执行结果（开发库当前无冲突数据），仅约束冷启动与
后续环境的迁移语义。本节以下终态数据为修订前实际执行结果。

| 表 | 终态行数 | 唯一性核验 |
|---|---|---|
| airlines | 988 | IATA 重复 0、ICAO 重复 0（`airline_iata_code` / `airline_icao_code` / `airline_awb_prefix` 全局唯一索引生效） |
| shipping_lines | 358 | SCAC 重复 0（`shippingline_scac_code` 生效） |
| shipping_line_container_prefixes | 随主表清理悬空行后按 prefix 去重 | `shippinglinecontainerprefix_prefix` 生效 |
| billing_units | 0（开发库本无数据） | `billingunit_code` 生效 |

`organization_id` 物理列已从上述四表删除，org 组合索引同步去 org。

## 3. ports / airports / fee_settings / exchange_rate_settings（B 型基线化）

总部根组织（当前开发库 1 个）名下行 `organization_id` 置 NULL 成为基线行，列改可空，
部分唯一索引落位（NULL 基线行业务码全局唯一，org 行 (org, 业务码) 唯一）：

| 表 | 基线行（NULL） | org 行 | 冲突 |
|---|---|---|---|
| ports | 17524 | 0 | 0（`ports_baseline_locode_unique` / `ports_org_locode_unique`） |
| airports | 9055 | 0 | 0（`airports_baseline_iata_unique` / `airports_org_iata_unique` / `airports_baseline_icao_unique`） |
| fee_settings | 0（开发库本无数据） | 0 | — |
| exchange_rate_settings | 0（开发库本无数据） | 0 | — |

## 4. fee_settings.charge_category_id 必填回填

列更名 `service_type_id` → `charge_category_id` 并去 Nillable。按费用名称关键词归类
（订舱→BOOKING、拖车→TRUCKING 等 19 条规则）回填后 fail-fast 校验 NULL 残留。
开发库 fee_settings 为 0 行，无归类失败行，未触发中止。

## 5. exchange_rate_settings 周结构 + 双轨点差（20260914231000）

- 新增 `ar_rate`（现汇卖出价/应收汇率）、`ap_rate`（现汇买入价/应付汇率）`numeric(18,8)` 可空列；
  `rate` 基准价列保留。
- 存量任意区间/开口区间行按自然周（Asia/Shanghai，周一 00:00:00 至周日 23:59:59）
  清理重建并补齐双轨初值（ar_rate = ap_rate = rate）。执行次序为「先按归一化后的
  目标周键（同作用域同货币对同周）幂等去重（保留 created_at/id 最新一行），再周归一化」，
  保证归一化 UPDATE 期间部分唯一索引不会因同作用域同周多行中断，任意存量库态可一次跑通；
  开发库为 0 行，实际无行重建。
- 周唯一索引沿用阶段一治理迁移落位的部分唯一索引
  （`(from_currency, to_currency, effective_from) WHERE organization_id IS NULL`
  及 org 作用域对应索引，`effective_from` 即当周周一）。
- `order_fees.exchange_rate_source` 枚举扩展 `WEEKLY / INHERITED_LAST_WEEK / BOC_SYNC`；
  `DERIVED / BASE_CURRENCY` 为应用层枚举值，开发库 order_fees 为 0 行，无历史值需清理。
  枚举值不随阶段一退役：阶段一解析链仍产出 SYSTEM/DERIVED/MANUAL，BASE_CURRENCY 为
  历史保留值；二者退役随阶段二 ResolveBaseRate 退役与容灾链落地一并执行（本阶段交付
  报告「与 design.md 的偏差」第 1 条已说明，schema 注释与迁移头注释口径一致）。

## 6. 结论

- 所有 fail-fast 检测点均未触发中止；开发数据按授权的「总部行 > 最早创建」规则处置完毕。
- 终态核验：各表重复业务码为 0，部分唯一索引全部就位，A 型表无 organization_id 物理列。
- 本报告归档于任务目录，处置明细另见迁移文件内 `RAISE NOTICE` 说明与迁移注释。

## 7. 集成测试库重建记录（测试基础设施，非业务数据）

阶段一验证发现 `roncin_go_admin_integration` 集成测试库停留在旧 Ent 结构
（master_data_items / shipping_lines 等仍带 NOT NULL organization_id 列）。该库为
`AutoMigrate: true` 的纯测试池（无 schema_migrations 历史、抽样表均为 0 行），
Ent AutoMigrate 只增不删无法收敛。已按测试基础设施维护重建（DROP/CREATE DATABASE），
重建后 TestOrderCreateTransactionPostgres、TestOrderIdempotencyPostgres、
TestSyncDefaultOrderOptionsPostgres、TestCreateDefaultOrderOptionsPostgres 全部通过。
