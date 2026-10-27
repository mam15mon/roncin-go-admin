# Research: 船代公共主数据后端与订单引用

> 本文记录规划初期的研究候选。其中 SQL 种子建议已由最终批准方案替换：正式迁移只处理类型与约束，29 条快照通过 `pnpm run sync:shipping-agents` 写入；实施口径以同任务 `design.md` 和 `implement.md` 为准。

- Query: 是否能复用 MasterDataItem 扩展 shipping_agent，如何同步契约、初始化、权限、拼音、全量维护接口与订单引用。
- Scope: internal（仓库源码、已导出来源、开发库最小只读统计）
- Date: 2026-10-27

## Findings

### 结论与最薄范围

可以复用现有 A 型全局 `master_data_items`，增加 `shipping_agent` 类型。无需新实体、独立仓储或独立权限体系。系统管理工作台统一维护，各公司读取同一份记录；现有领域规范明确系统管理并非经营公司的业务上级，不宜在产品文案中称为“总公司”。

除了扩展 kind，需要同步改动订单已有 `shipping_agent_id` 的语义：当前它在写入校验与名称投影中均被当作本公司 Partner/supplier，不能只改前端下拉框。订舱代理、国外代理继续沿用现有 Partner 路径。

### 文件与锚点

| 文件 | 锚点与作用 |
| --- | --- |
| `server/internal/biz/masterdata.go:23` | MasterDataKind 常量与 Valid，新增 shipping_agent。 |
| `server/internal/biz/masterdata.go:43` | MasterDataItem 现有字段足够：UUID、kind、code、中文名、name_en、source、sort_order、enabled。 |
| `server/internal/biz/masterdata.go:242` | MasterDataRepo 接口，当前只有分页 List 与启用全量 ListEnabled，无含停用项的全量维护方法。 |
| `server/internal/biz/masterdata.go:276` | Create / Update / Import 已统一 RequireGlobalMasterDataWrite 和业务审计。 |
| `server/internal/data/ent/schema/master_data_item.go:26` | Ent kind 枚举，增加 shipping_agent；现文件没有 Annotations，需要把正式 kind CHECK 补到 Ent 真相源。 |
| `server/internal/data/ent/schema/master_data_item.go:40` | 名称与英文名拼音 hook；新增 kind 自动复用。 |
| `server/internal/data/ent/schema/master_data_item.go:50` | `(kind,code)` 唯一，kind/enabled/sort_order 与 kind/name 索引可复用。 |
| `server/internal/data/masterdata.go:76` | 分页列表现有 keyword 支持 code/name/name_en/search_keywords。 |
| `server/internal/data/masterdata.go:96` | ListEnabled 返回全部启用目录，新增 kind 后会自然进入订单现有字典缓存。 |
| `server/internal/data/masterdata.go:108` | CRUD/导入统一 WithTx，审计失败整批回滚，无删除接口，停用保留引用。 |
| `server/api/masterdata/v1/masterdata.proto:161` | Proto kind 枚举最后为 ABNORMAL_CASE=10，可追加 SHIPPING_AGENT=11，禁止复用 reserved 值。 |
| `server/internal/service/masterdata.go:444` | kind 双向 DTO 转换 switch，必须同步新增。 |
| `server/internal/biz/organization.go:31` | 公共写双门禁：系统管理工作台 + 对应权限；公司管理员同名角色不拥有公共写权。 |
| `server/internal/access/manifest.go:103` | 可复用 `system.master_data.item.read/create/update/import`；不必新增权限。 |
| `server/internal/data/order_sync.go:27` | 创建与草稿更新共用引用校验；船代应在此独立校验 MasterDataItem。 |
| `server/internal/data/order_sync.go:133` | ShippingAgentID 现进入 Partner supplier/blacklist 规则，须移除该分支。 |
| `server/internal/data/order_query.go:197` | 列表/详情共用 attachOrderDisplayNames；船代从 Partner 批查改为 shipping_agent 主数据批查。 |
| `server/internal/data/ent/schema/order.go:45` | 现 shipping_agent_id 是可空 UUID，未定义 Ent edge；字段形状可以保留。 |
| `server/internal/data/order_write.go:37` | 创建在同事务 validateOrderReferences 后写入 ShippingAgentID。 |
| `server/internal/data/order_write_draft.go:202` | 更新在订单悲观锁+version 门禁后执行同一引用校验。 |
| `server/internal/data/order_convert.go:231` | 更新 nil 清空 / 非 nil 设置 UUID 的语义无需改变。 |
| `server/internal/data/sea_order_change_split_execute.go:1250` | 拆票新子单复制源订单 ShippingAgentID，需回归 ID/名称保留。 |
| `server/internal/biz/order_usecase.go:598` | agents 请求指纹已包含船代 ID，保留现状即可。 |
| `server/api/order/v1/order.proto:368` | Order DTO 与 create/update 已有 shipping_agent_id / shipping_agent_name，不需新增 JSON 字段；可补中文语义注释。 |

### 接口形状与全量维护

现有契约：

- `GET /api/v1/master-data/items?page=1&pageSize=20&kind=MASTER_DATA_KIND_...&keyword=...&enabled=...`：目录维护读取权限、常规分页最大 200。
- `POST /api/v1/master-data/items`：kind/code/name/nameEn/source/sortOrder 等。
- `PUT /api/v1/master-data/items/{id}`：kind/name/nameEn/source/sortOrder/enabled 等；code 只在新建时写且不可变。
- `POST /api/v1/master-data/import`：kind/source/mode/items，可复用但无需为这 29 项另造导入系统。
- `GET /api/v1/master-data/options`：已登录，所有启用项全量。
- `GET /api/v1/master-data/options/search?kind=...&keyword=...&page=1&pageSize=20`：已登录，固定 enabled=true，支持服务端拼音搜索与 200 上限。

小数据维护必须全量 + 前端分页。现 ListItems 不能用 pageSize=200 代替真正全量；ListOptions 又漏掉停用项，不能直接复用维护页。建议新增最薄通用 `ListAllItems` / `GET /api/v1/master-data/items/all`，request 仅 `kind`（必填）与可选 `enabled`，无 page/pageSize；response 用 repeated MasterDataItem + total/trace envelope。复用现 `MasterDataItemRead` 权限，biz 校验 kind，data 按 kind/sortOrder/code 返回全部，不循环分页。仅本次船代页面消费，其他现有目录不做无关改造。也可命名 ListShippingAgents 专用全量接口，但会重复现有通用目录类型表达，通用按 kind 读取更薄。

订单搜索直接复用 SearchOptions(kind=shipping_agent)，默认字典候选可复用 ListOptions 自然带出的新 kind。无需新候选 API 或新分页策略。

### 初始化与来源

来源文件：`/home/vince/.codex/visualizations/2026/10/27/01a17c24-6c08-7e62-88cb-46f2616d6bff/eyun-ship-agents/ship-agents.json`。

- 已读取 JSON 并验证共 29 项，ID 唯一 29、中文名唯一 29。
- 所有 code=null、aliasNames=null，部分 enValue 仍为中文；不能宣称这些是行业标准船代编码或擅自翻译英文名。
- 原始 JSON SHA-256：`93dca1753a3db49847b6396853b12beac20d5f6772e6b16cd7ebbd86d193069c`。
- 已知来源接口为 `GET https://anoingress.hgj.com/saas-support/data/item/1020`（本研究复用已导出内容，没有重新登录或网络请求）。
- 建议在实现时把无凭据来源 JSON/清洗 CSV 归档 `server/seeds/` 并记录来源日期、摘要、映射规则。可用 `HGJ-<外部整数ID>` 生成明确标识来源的稳定内部 code；这是导入码，不冒充官方业务编码。UUID 是本系统 ID，与外部数值 ID 分离。name=value，name_en 可忠实保留 enValue，sort_order 按来源 order 确定（相同 order 用 code 稳定排序）。
- 29 项由正式有序 SQL 迁移一次性初始化，`ON CONFLICT(kind,code) DO NOTHING` 保留管理员后续改名与停用；新库无需依赖 sync-dev、bootstrap 或在线服务可用性。
- `server/migrations/20260923130000_abnormal_case_seed.sql:4` 是现有合适范本：同码跳过且预计算 search_keywords。无需同时在多个 Go 种子入口重复维护同一船代目录。
- `server/internal/data/masterdata.go:19` 的 DefaultOrderOptions 仅是 bootstrap 幂等补种范本，当前没有加载任意 seeds JSON 的通用机制。本次不必扩建导入框架。

### CHECK、索引与拼音

1. 修改 Ent kind enum、biz 常量、Proto enum、service 双向转换。
2. 新迁移以稳定名 `master_data_items_kind_check` 收敛到当前合法 kind + shipping_agent。正式旧定义在 `20260914220000_master_data_storage_governance.sql:33`；仍含 charge_category，`20261024100000_order_fee_refinements.sql:14` 只删除该类数据，未更新 CHECK。当前 Ent 已去掉 charge_category。不要继续把旧 charge_category 加回新合法集合。
3. 根据 database-guidelines 的 CHECK 同源要求，同步 `MasterDataItem.Annotations` 的 `entsql.Checks`，生成 `ent/migrate/schema.go` 并测试稳定约束名与表达式。
4. 本机只读已确认该稳定名 CHECK 缺失（见下一节），故本次新迁移可针对该已确认前态使用 `DROP CONSTRAINT IF EXISTS` 后 ADD；不是新增通用静默修复分支。冷启动正式链上的已有 CHECK 也须正确替换。
5. 现有 `search_keywords` 列与 trigram 索引来自 `20260827133000_selector_search_keywords.sql:5/16`；无需新增列、索引或独立拼音库。
6. `server/internal/data/ent/schema/search_keywords.go:15` 自动更新中英名称检索键；`search_keywords_backfill.go:22` 已覆盖 master_data_items，迁移命令可补空检索键。
7. SQL 种子不能跑 Ent hook；必须用现有 `internal/platform/searchtext.Build(name,name_en)` 预计算，种子里写入正确键，测试逐行与 Build 相等。不能只依赖后置回填，使迁移后到回填前缺检索。

### 订单引用正确性、停用与旧引用

- 新建/更换船代：查询 MasterDataItem 的 id+kind=shipping_agent+enabled=true；在订单事务内 ForShare 锁住引用行，数据库异常原样传出。
- 草稿不更换已有船代：参照同文件 ShippingLine（`order_sync.go:31`），允许保留已停用的原引用，但仍须存在且 kind 正确；编辑其他字段无需先换船代。
- 清空可选船代：继续 nil/clear 语义。
- 新建时复制已停用船代应按“新建选择”拒绝，前端可要求重新选择；拆票是已有订单身份延续，现直接复制源 UUID 的路径与当前既有字段语义一致，需明确回归但无需顺便改造拆票规则。
- 名称投影：船代 ID 单独收集，一次 MasterDataItem IN 查询并按 kind 过滤；不加 enabled=true，使已停用历史引用继续显示。列表/详情共用投影，无需逐行请求或 Partner 回退。
- 不应更改订舱代理/国外代理 Partner 校验、黑名单与公司范围；当前“船务代理属于 supplier”的旧测试分支应替换为公共字典语义测试。
- 已有字段没有 FK，本次可保留最薄逻辑引用方案（没有删除接口，只有停用）；如计划新增 FK，应同时通过 Ent edge 表达，正式迁移与生成物一起更新，不手写孤立数据库关系。该扩展并非本期闭环必须。

开发库最小只读核查（2026-10-27，使用 `.env.local` 连接、psql 只读会话、5 秒 statement_timeout，未打印连接串，未执行数据修改）：

| 项目 | 结果 |
| --- | --- |
| orders 总行数 | 0 |
| shipping_agent_id 非空订单数 | 0 |
| 非空船代不同 ID 数 | 0 |
| 已有 kind=shipping_agent 项数 | 0 |
| orders.shipping_agent_id 外键数量 | 0 |
| 稳定名 master_data_items_kind_check | 未查到 |

因此当前开发数据无需 Partner→公共船代数据映射。正式迁移仍应对不属于 shipping_agent 主数据的非空旧订单引用显式 RAISE EXCEPTION，原子拒绝而不是自动清空、按名称猜配或创建兼容分支。跨环境不能用“本地为零”推定其他环境也为零。若真的遇到旧引用，再由用户确认映射方案。

### 建议验证矩阵与现有测试入口

| 路径 | 最低验证 |
| --- | --- |
| kind / service | Proto 双向转换、创建更新正常、非法 kind、全量无分页参数、含停用项、返回超过200项仍完整。 |
| 权限 | 系统工作台+正确权限可维护；公司管理员即使伪持同码仍禁止；公司登录用户可查候选，无主数据维护权也能使用订单。 |
| 拼音 | 代码、中文、真实英文、全拼、首字母命中；停用排除、其他 kind 排除；更名更新检索键。 |
| 全量页读取 | 全部项含停用、稳定排序、kind隔离；禁止循环200条分页拼接。 |
| 订单新建/更新 | 有效船代成功，Partner ID/错误 kind/不存在/停用新选择失败；保留停用旧引用、清空、替换成功；查询驱动错误保持。 |
| 订单展示/复制 | 列表与详情输出 shippingAgentName；复制源订单保留有效公共UUID、从停用源复制的新建需重选；拆票新子单引用与展示不丢。 |
| 正式迁移 | 空Schema完整链、升级前态、29项数目/映射/搜索键、管理员改名停用后重跑不覆盖、非法旧引用原子拒绝、CHECK物理与Ent元数据一致、revision/checksum、非目标订单/人员/组织保持。 |

现有可复用测试：

- `server/internal/biz/masterdata_test.go:152` 分页校验；`:488` 系统/公司权限上下文与后续拒绝公司公共写用例。
- `server/internal/service/masterdata_options_test.go:30` 搜索转换/分页，`:67` access 规则反射。
- `server/internal/service/masterdata_baseline_convert_test.go` 主数据 DTO 转换。
- `server/internal/data/masterdata_options_integration_test.go:12` 真实 PostgreSQL 代码/中文/英文/全拼/首字母+分类启用过滤。
- `server/internal/data/masterdata_baseline_unique_integration_test.go:16` 公共主数据全局唯一冲突。
- `server/internal/data/order_create_transaction_test.go:189` 旧 Partner/黑名单矩阵，`:219` 船务代理旧分支必须移除或改写，其余代理相邻路径保留回归。
- `server/internal/data/shared_location_order_integration_test.go:11/58` 公共引用读取/校验的集成范本。
- `server/internal/data/order_transaction_integration_test.go:53` 完整订单事务创建范本。
- `server/internal/data/integration_test_helper_test.go:28` getIntegrationDataBeforeMigration 通过正式迁移前缀构造前态，随机隔离 schema，不调用 Schema.Create。
- `server/internal/data/billing_unit_quantity_migration_integration_test.go:10` 默认种子重放保留管理员配置范本。
- `server/internal/data/mobile_login_migration_test.go:49` 生成表 Annotation.Checks 元数据断言范本。

### 生成与验收命令（实现阶段执行，本研究未执行）

```bash
# 修改源文件后生成 Ent / API / OpenAPI 客户端，不能手改生成文件。
go -C server generate ./internal/data/ent
make -C server api
pnpm run generate:web-client

# 本地定向测试，具体新增测试命名以最终实现为准。
go -C server test ./internal/biz ./internal/service -run 'TestMasterData|TestOrder' -count=1
go -C server test ./internal/data -run 'TestValidateOrder|TestOrderCreateRollsBack|TestMigrationManifestCoversEntSchema' -count=1

# 显式连接专用一次性数据库后执行真实迁移/订单/搜索集成；不能用日常开发库做测试。
# RONCIN_INTEGRATION_DATABASE_SOURCE 在 shell 环境中注入，日志不可输出连接值。
go -C server test ./internal/data -run 'TestMasterData.*Postgres|TestSharedLocationOrder.*Postgres|TestOrderCreateTransactionPostgres|TestShippingAgent' -count=1 -timeout 25m -json

# 正式任务验收后执行并确认真实 DB 测试没有 SKIP。
pnpm run check:fast
```

现有 `server/internal/data/ent/generate.go:6` 固定 Ent 生成特性 sql/lock、sql/upsert。仓库未找到独立“SQL迁移差异生成”脚本，不应编造 atlas 命令或安装工具；采用已有正式有序 SQL 迁移组织方式并审阅、真实执行。Manifest 不改时不需要 generate:permission-keys。

## Related Specs

- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：主数据与公司经营实体边界。
- `.trellis/spec/server/backend/organization-shared-masterdata.md`：系统管理统一维护，各公司共享；初始化保留管理员配置；SQL种子拼音键要求。
- `.trellis/spec/server/backend/database-guidelines.md`：Ent CHECK/SQL同源，事务与锁、正式迁移。
- `.trellis/spec/server/backend/quality-guidelines.md:206`：专用PostgreSQL必须真实执行，SKIP不能报通过。
- `.trellis/spec/adr/0006-pagination-limit-and-masterdata-strategy.md`：小数据维护全量+本地分页。
- `.trellis/spec/adr/0007-pinyin-search-keys.md`：选择器多维检索。
- `.trellis/spec/server/backend/partner-role-blacklist.md`：相邻订舱代理/国外代理继续保持公司与角色门禁。

## External References

- 本次未查外部技术文档，技术判断以仓库现有实现及规范为据。
- 已导出来源来自海管家船代字典接口（见初始化章节）；该来源不能证明对方底层数据库表结构或完整行业覆盖。

## Caveats / Not Found

- 本阶段仅研究，未修改产品代码、生成物或数据库，未运行测试。
- 开发库订单为零是本次观察，不构成其他环境旧引用不存在的证明。
- 开发库缺稳定名 kind CHECK 与正式迁移链预期有差异；实施时必须同步验证本机实际迁移、空库正式链与新增Ent元数据，不能只测试内存 enum。
- 未找到当前源码中的独立后端“复制订单 RPC”；普通复制应由前端读取旧值发新建请求，因而仍走创建引用校验。后端实际存在拆票复制UUID路径，已列出锚点。
- 若本次被扩展为船代真实结算主体（银行账户、联系人、账期等），需要另作公司Partner关联设计；当前需求仅公共船代选择字典，无需预建这些能力。
