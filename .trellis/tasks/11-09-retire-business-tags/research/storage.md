# 研究： 业务标签存储退役与正式迁移边界

- 研究问题： 全站取消业务标签时，哪些表、字段、关联与生成物应退役，如何只清理标签资产并保留业务单据、其他资源和历史审计。
- 范围： internal（Ent Schema、正式 SQL 迁移、运行期仓储、测试与种子源码；本研究代理未连接数据库）
- 日期： 2026-11-09

## 研究结果

### 1. 明确的持久化资产

| 资产 | 字段/唯一键 | 外键及现状 |
| --- | --- | --- |
| `enterprise_tag_groups` | id、organization_id、name、normalized_name、color、sort_order、search_keywords、时间；唯一 `(organization_id, normalized_name)` | organization_id→organizations；只服务标签组。来源 `enterprise_tag.go:11-39`，初建迁移 `20260829120000_enterprise_resources.sql:24-39`，检索键迁移 `20260830010000_enterprise_tag_group_search_keywords.sql:1-3` |
| `enterprise_tags` | id、resource_id、organization_id、group_id、normalized_name、sort_order、时间；resource_id唯一，`(group_id, normalized_name)`唯一 | resource_id→enterprise_resources、group_id→enterprise_tag_groups，均NO ACTION；Schema `enterprise_tag.go:42-65`，初建迁移`:188-205` |
| `order_enterprise_tags` | id、organization_id、order_id、tag_resource_id、时间；`(order_id, tag_resource_id)`唯一 | 分别→organizations/orders/enterprise_resources，NO ACTION；`business_tag_relation.go:12-38`，迁移 `20260829230000_business_tag_associations.sql:1-17` |
| `order_fee_enterprise_tags` | id、organization_id、order_fee_id、tag_resource_id、时间；`(order_fee_id, tag_resource_id)`唯一 | 标签/组织FK为NO ACTION；order_fee_id→order_fees现为CASCADE（仅费用删除带走关联，不反向删除费用）；Schema`:41-68`，迁移`:19-35`及 `20260922130000_order_fee_delete_by_bill_occupancy.sql:15-18` |
| `finance_bill_enterprise_tags` | id、organization_id、finance_bill_id、tag_resource_id、时间；`(finance_bill_id, tag_resource_id)`唯一 | 分别→organizations/finance_bills/enterprise_resources，NO ACTION；Schema`:71-97`，迁移`:37-53` |
| `enterprise_resources` 中 `resource_type='TAG'` 行 | 通用id、组织、简称/名称、启用、排序、创建更新人、检索键、时间 | 这是标签资源的主体名称，不是独立标签专表；只能删除精确TAG行，不能删除整表。Schema `enterprise_resource.go:11-30` |
| `enterprise_resource_partners` 中指向TAG资源的行 | resource_id、partner_id、resource_type、is_default、时间；`(resource_id,partner_id)`唯一，`(partner_id,resource_type)`的默认项部分唯一索引 | 通用客商关联，可合法连接标签，也服务其他资源；只能按目标TAG资源ID删除关联，保留partners以及其他资源关联。`enterprise_resource_relation.go:11-37`，初建迁移`:141-157` |

所有专表还有ID/时间字段和普通排序/检索索引；DROP专表会自动移除它自身的索引与外键，不需要对orders/order_fees/finance_bills删列，也不得反向删除这些单据。

当前资源类型在数据库正式迁移中是 `character varying`（初建迁移`:5`、`:145`），不是独立 PostgreSQL enum type；Schema生成表信息使用TypeEnum，但未查到资源类型的显式数据库CHECK迁移。不能假设应执行 `DROP TYPE` 或移除不存在的 `resource_type_check`。实施应以Ent差异和正式迁移审阅为准，不为退役顺带建设新的枚举约束体系。

### 2. 必须保留的共享存储

- `enterprise_resources` 其余类型：ADDRESS、REMARK、IMAGE、SHIPPER、CONSIGNEE、NOTIFY_PARTY、ENGLISH_CARGO_NAME、HS_CODE、MARKS。通用创建人、更新人和organization外键不变。
- 专属表 `enterprise_resource_addresses/remarks/images/parties/shipping_texts` 保留；资源关联表 `enterprise_resource_partners/assignees/address_types` 保留给对应合法资源。
- 对enterprise_resources的所有入向FK，本轮源码扫描在初建资源迁移与业务标签关联迁移中找到：上述各detail/通用relation及三张tag关联表，未发现其他后续正式SQL新加标签资源FK。
- 标签可关联客商不等于客商本体是标签，删除标签关联不能删除 `partners`；同理不可删除orders、order_fees、finance_bills、账单行、核销、提成或关联单据历史。
- `audit_logs` 保留全部历史。`ent/schema/audit_log.go:17-28` 的 resource_type/resource_id 是可选字符串，details是JSON，无标签资源/标签组外键。历史action/详情里的标签字样不是运行期兼容层，不应清空或改写（初建见 `20260820000000_baseline.sql` audit_logs）。

### 3. Ent与运行期依赖清理

- 删除Schema `enterprise_tag.go`（Group/Tag两实体）及 `business_tag_relation.go`（三关联实体），再通过正式Ent生成流程更新；`ent/generate.go:6` 当前入口 `go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/lock,sql/upsert ./schema`，通常 `go -C server generate`。
- 从 `enterprise_resource.go:11-13` 的共享类型集合移除TAG，同时移除`:46` tag、`:50-52` 三个业务标签links；`EnterpriseResourcePartner` 共用该类型集合，也必须同步，不删除partner_links。
- 移除 `organization.go:74-77` 四条标签相关edges、`order.go:141`、`order_fee.go:83`、`finance_bill.go:76` 的enterprise_tag_links。其余edges保留。
- 生成后审查client/tx/mutation/migrate/schema.go、实体builder/query及实体子包不再遗留已删除实体；不能手改生成文件“修编译”。
- 不能只删CRUD与列表标签字段：`sea_order_change_split_execute.go:1363-1378` 拆票复制订单标签，`:1941-1962` 复制费用标签并删除原标签关联；这些运行期依赖要同步移除，保留相邻订单品类复制、费用迁移、守恒校验及原事务。
- `enterprise_resource.go:309-340` 现有删除资源时会锁标签、统计客商/订单/费用/账单引用；`:343-370` 为统一资源删除清理各detail，包含标签分支。全站退役后清除标签分支，保留图片对象删除队列及其他类型分支，不能删整个资源删除用例。

### 4. 权限数据

- 独立费用标签权限是 `system.finance.fee.tag`，`access/manifest.go:48,:222`；组织级企业资源CRUD权限同时服务地址/备注/图片/主体，不能连权限一起删除，只去掉其名称/说明中的标签组承诺。
- `permission_manifest_sync.go:31-34,:59-86` 已有正式权限清单同步：删除manifest外的permission；`cmd/migrate/main.go:47` 在Schema迁移后执行同步。
- `role_permissions` 的 permission_id→permissions FK为ON DELETE CASCADE（`20260820000000_baseline.sql:122`）；废弃permission删除会清除对应授权，不会删除角色或用户。
- 推荐复用已有manifest同步删除专属权限及关联，不另造迁移权限同步器或直接删通用角色。独立业务类型订单标签权限要由契约研究准确列清，存储机制相同。

### 5. 正式退役迁移顺序（需先取得标签数据删除授权）

推荐只增加一份新的正式迁移；不改历史SQL，冷启动仍按顺序创建旧结构后由新迁移退役。以下是设计顺序，不是本轮执行过的SQL：

1. 在迁移事务中固定目标为 `enterprise_resources WHERE resource_type='TAG'`，执行一致性检查。有效设计不会为TAG创建地址/备注/图片/主体/文本detail或地址人员/类型关联；若这些非标签表实际引用目标TAG行，显式失败并报告，不能用CASCADE删除意外业务内容。对 generic partner links 按真实resource_id识别，不只凭冗余resource_type文本做全表删除。
2. `DROP TABLE order_enterprise_tags; DROP TABLE order_fee_enterprise_tags; DROP TABLE finance_bill_enterprise_tags;`，移除三类标签关系及其外键。使用默认RESTRICT，不写DROP...CASCADE；这样有未知外部依赖时会显式拒绝。
3. 删除 `enterprise_resource_partners` 中 `resource_id IN (SELECT id FROM enterprise_resources WHERE resource_type='TAG')` 的行。若冗余resource_type='TAG'却指向非TAG资源，先报错核对，不能为了“清标签”删错其他资源关联。
4. `DROP TABLE enterprise_tags`，再 `DROP TABLE enterprise_tag_groups`（次序由group FK决定），两者默认RESTRICT，不反向删除通用资源。
5. `DELETE FROM enterprise_resources WHERE resource_type='TAG'`。此时三标签关联、tag detail和合法partner links已清理，其他detail非法引用会被前置检查/NO ACTION FK阻止；不应通过多表CASCADE兜底。
6. 正式迁移框架提交成功后，既有 `cmd/migrate` 权限manifest同步移除废弃标签permission与role_permissions。Schema与运行时均不再支持TAG。

对于Tag detail漏建而基表TAG存在的旧数据，上述设计仍只精确删除TAG基表；对于专属标签表指向错误resource_type的历史不一致，可整表退役但绝不能扩大删除主体对象。所有核心单据及非TAG资源行在迁移前后应完全一致。

迁移并非“重置开发库”，但它仍永久移除标签元数据/关联。因此主会话应在最终规划明确删除集合及授权；当前“创建任务并规划”不是应用迁移授权。不要由本轮研究运行任何SQL写操作。

### 6. 数据保留备选与成本

| 方案 | 成本/影响 | 建议 |
| --- | --- | --- |
| 明确授权后彻底退役上述标签资产 | Runtime与Ent无残留，正式迁移精确删除，需迁移前备份和后置保护测试 | 符合用户取消功能及上线前最薄闭环；推荐最终方案明确这项不可逆变化 |
| 功能/Ent删除，但原标签表和TAG基表行原样留库 | 不触发删除，但Schema源与实际库存在废弃对象；后续迁移/新Ent查询碰到不认识类型需谨慎；仍有清理债务 | 仅在用户不授权删除数据时可作为明确分阶段范围，不能声称存储彻底清理 |
| 建运行期archive表/兼容查询/软删平台 | 需额外表/生成器/保留规则，形成未有现实用途的历史兼容方案 | 本期不推荐 |
| 迁移前按环境做离线备份/导出，随后正常退役 | 不保留运行期兼容；备份需外部受控存放，禁止把真实数据/凭据提交Git | 若用户有保留需要，低于维护archive功能的成本 |

**必须由用户确认的一项数据处置问题**：是否同意正式迁移永久移除标签组、标签定义、TAG通用资源行及其客商/订单/费用/账单关联，同时保留业务单据、其他企业资源和全部历史audit？若用户要求保留标签内容，应明确仅导出保存还是推迟数据库删除。

### 7. 本地只读数量快照与种子

- 主会话于2026-11-09通过开发环境连接执行 `BEGIN READ ONLY` / 数量查询 / `ROLLBACK`：enterprise_tag_groups、enterprise_tags、order_enterprise_tags、order_fee_enterprise_tags、finance_bill_enterprise_tags、enterprise_resources WHERE resource_type='TAG'、enterprise_resource_partners WHERE resource_type='TAG'均为 **0**。这是主会话提供的本地快照，不能推断其他环境也没有标签数据；本研究未取得连接、未读业务内容、未写DB。
- `server/cmd/sync-dev/*.go` 定向搜索 `Tag/标签/tag_resource` 未发现种子创建业务标签；无须为了退役改无关订单/费用种子。正式实施仍应全仓确认其他运行期种子入口。
- 标签关联夹具见 `order_fee_delete_integration_test.go:224-245` 与 `order_fee_bulk_integration_test.go:161-189`，当前验证费用删除带走标签关系；Ent删除实体后这些断言需退役或移到针对历史迁移的raw SQL fixture，保留原费用删除/占用保护的其他断言。
- service stub `enterprise_resource_import_test.go:54-60` 实现TagGroup方法，仅接口依赖清理；不应删除整个import测试。

### 8. 最小验证方案

1. 正式迁移前态建一套各标签表/客商关联都有数据、并混合地址/备注/图片/主体/文本资源的隔离PostgreSQL fixture，执行新迁移后断言标签五表消失、TAG基表与标签partner links为0。
2. 比较迁移前后orders/order_fees/finance_bills/账单行/其他资源及audit_logs的主键、计数、关键字段快照保持不变。真实标签异常引用应显式失败并回滚，不用CASCADE消音。
3. 最新态完整迁移冷启动通过；历史前缀测试只执行其所属时点，不在最新Schema要求已退役标签表存在。用raw SQL构造前态，不能继续引用已移除Ent类型。
4. 迁移后SyncPermissionManifest删除专属标签权限和授权，通用资源CRUD仍可用；已有角色、用户、公司不变。
5. 拆票、费用删除/批量删除、列表查询、其他企业资源CRUD/导入回归通过；清理标签专属依赖不改变写入事务与金额守恒。
6. `go -C server generate`、`make -C server api`、`pnpm run generate:web-client`、`pnpm run generate:permission-keys`按实际源变化生成；最后风险匹配全栈门禁。迁移与集成测试不得并发Schema.Create，不把SKIP当PASS。

## 文件清单

- `server/internal/data/ent/schema/enterprise_tag.go`：标签组与标签详情的Ent真相源。
- `server/internal/data/ent/schema/business_tag_relation.go`：订单、费用、账单标签三关联实体。
- `server/internal/data/ent/schema/enterprise_resource.go`、`enterprise_resource_relation.go`：通用类型集合与共享资源关系。
- `server/internal/data/ent/schema/{organization,order,order_fee,finance_bill,audit_log,permission,role}.go`：标签edges、审计及权限关系。
- `server/migrations/20260829120000_enterprise_resources.sql`、`20260829230000_business_tag_associations.sql`、`20260830010000_enterprise_tag_group_search_keywords.sql`、`20260922130000_order_fee_delete_by_bill_occupancy.sql`：现行结构和费用关联FK变更。
- `server/internal/data/enterprise_resource.go`、`business_tag.go`、`sea_order_change_split_execute.go`：资源清理、业务关联与拆票复制运行期依赖。
- `server/internal/data/permission_manifest_sync.go`、`server/cmd/migrate/main.go`：已有权限退役同步。
- `server/internal/data/order_fee_{delete,bulk}_integration_test.go`、`server/internal/service/enterprise_resource_import_test.go`：删除/导入测试依赖。
- `server/cmd/sync-dev/*.go`：本轮未找到标签数据种子。

## 外部依据

未使用网络或外部资料。结论基于当前仓库正式SQL及Ent Schema；未猜测数据库框架新版本行为。

## 相关规范

- 根 `AGENTS.md`：Ent Schema真相源、正式迁移、禁止自动删重置数据、生成物不手改、无历史兼容默认要求。
- `.trellis/spec/server/backend/directory-structure.md`：DTO/领域/持久化边界。
- `.trellis/spec/server/backend/quality-guidelines.md:238-299`：专用集成库、迁移前态时点、冷启动与SKIP不得算PASS。
- `.trellis/spec/server/backend/finance-bill-currency.md:181`：费用删除时标签关联CASCADE是旧已确认契约，退役后同步更新现行规范而不改历史SQL。
- `.trellis/spec/domain/architecture-map.md:33`、`.trellis/spec/server/backend/capability-navigation.md:37`：标签能力导航需在实施后退役，不能留下失效入口。

## 限制与未确认项

- 本研究没有执行数据库连接、DDL/DML、迁移或测试；只写本research文件，未修改产品/Schema/生成物/Git。
- 主会话本地数量快照为0不代表其他环境；正式发版仍需审阅迁移删除集合。
- 未发现显式resource_type数据库CHECK或独立PostgreSQL enum type，不应据Ent的Go枚举假设存在；若目标环境有手工漂移，应让正式迁移显式失败并核对，而非盲目CASCADE。
- 所有当前表/外键清单来自正式迁移源码，实施前还应对专用测试库查询pg_catalog验证实际依赖，尤其新增迁移不能删除无关共享表。
