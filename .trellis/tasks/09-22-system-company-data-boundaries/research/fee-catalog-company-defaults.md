# Research: 公司费用科目与系统初始目录

- Query: 费用科目公司独立、系统提供初始目录的最小实现及共享费用引用迁移。
- Scope: internal
- Date: 2026-09-22

## Findings

### 当前文件与契约

- `server/internal/data/ent/schema/fee_setting.go:20`：organization_id 可空；NULL 目前是可直接消费的共享科目。taxable_service_id 必填且关联公司私有应税劳务；税率、币种、单位及费用大类都在科目上。两个部分唯一索引分别约束共享 code 和公司+code。
- `server/internal/data/ent/schema/taxable_service.go:20`：organization_id 必填，公司+name 唯一；name/short_name/goods_code/default_tax_rate/enabled 是公司税务资料。
- `server/internal/biz/fee_catalog.go:116`：创建根据总部上下文决定 NULL/当前组织归属；更新允许总部改 NULL。`normalizeFeeSetting:259` 强制非空应税劳务 ID。
- `server/internal/data/fee_catalog.go:20`：列表调用 feeSettingBaselineScope，按本地同码遮蔽。`:133` 校验应税劳务归当前组织；`:285` 转换强制加载 TaxableService edge，否则整个列表出错。
- `server/internal/data/order_fee.go:172`：订单选项消费共享+本地科目；`:306` 按 ID 允许 NULL 或本公司，并将税率/税务名称写入费用快照。
- `server/internal/data/order_fee_supplement_approval.go:35`：补录审批也允许 NULL 或本公司，必须与普通费用一起收紧。
- `server/internal/data/ent/schema/order_fee.go:24`、`order_fee_supplement_request.go:60` 是两个直接持有 fee_setting_id 的实体。后者 Immutable 只限制 Ent 日常写入，不阻止正式 SQL 迁移。
- `server/api/finance/v1/fee_catalog.proto`、`server/internal/service/fee_catalog.go:19` 为目录 CRUD 契约和转换；公司 ID 来自 principal，不接受自由公司参数。
- `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx:137` 有共享覆盖提示；`:161` 根据 organizationId/总部决定编辑；`:351` 应税劳务必选。模板 UI 不能继续复用这个必选公司税务下拉框而伪造系统公司。
- `server/internal/data/admin_organization.go:49` 创建公司已有 WithTx，可在该事务内初始化目录。`organization_seed.go:25` 批量初始公司创建是另一路径，也需调用相同初始化 helper。
- `server/cmd/sync-dev/main.go:465,551` 已按公司生成应税劳务与科目；它是开发演示数据，不是生产系统默认目录。不应让生产初始化依赖 sync-dev。
- `server/migrations/20260914220000_master_data_storage_governance.sql:329` 为现有共享化迁移；新调整新增正式迁移，不改该历史迁移。

### 建议的最小模型

首选一个明确的系统模板实体（例如 FeeSettingTemplate），公司 FeeSetting 全部非 NULL company ID。模板保存名称、代码、公共费用大类、公共计费单位、默认币种、可选异常类型及初始税率。若模板需开箱可用，可保存应税劳务的文本默认值（名称/简称/商品编码/税率），创建公司时复制为本公司 TaxableService 后再建立本公司 FeeSetting；模板绝不能保存任何公司的 taxable_service_id。

这是一张直接用途的模板表和一个公司初始化 helper，不需要版本、订阅、更新同步、继承关系或模板绑定状态。公司初始化后无需保留模板外键；系统更新模板不会触碰公司科目。现有公司在正式迁移一次性独立化，新公司仅创建时复制。

若不愿增加独立表，也可以保留 NULL FeeSetting 作为模板，但必须让 taxable_service_id 可空、补模板文本字段、公司必有税务/模板不得有税务的数据库 CHECK，并分开 biz/API/转换/列表逻辑。这样虽然少一张表，却扩大现有 FeeSetting、订单 DTO 的可空性和误用风险；不推荐为省表采取这个方案。

尚需实施者与主会话确定：默认目录是否包含税务文本初始值，或公司科目复制后先禁用、由公司补齐税务才启用。不能默默虚构税目，也不能仅因模板无税务而把现有费用保存校验放宽。本报告推荐用既有共享目录真实税務值制作初始默认文本；本地新公司复制是配置初始值，不自动修改旧公司。

### 数据迁移策略

1. 迁移前检查共享科目引用是否完整，组织是否公司，是否有旧总部经营数据；归属不明立即报明细中止，不自动归到任意公司。
2. 从 NULL 科目生成系统模板；模板提取应税劳务文本，不能保留原公司税务 ID。
3. 建临时映射 `(source_fee_id, company_id) -> target_fee_id`。为每家公司复制当前可用共享目录，但已有同码本地科目优先，保持当前可选择集合语义。
4. 应税劳务也建立 `(source_taxable_id, company_id) -> target_taxable_id`。不存在同名则复制；同名且所有有效配置一致可复用；同名但配置不同需明确冲突报告，不能用 ON CONFLICT DO NOTHING 后无条件复用导致业务改变。
5. 已有共享科目被 order_fees / order_fee_supplement_requests 引用时，按行的 organization_id 找目标。没有同码本地覆盖则可映射到步骤 3 副本；已有同码本地科目且配置不同不能把旧共享引用直接指向不同配置，应报冲突由用户裁定（或经明确批准产生独立不同代码副本）。不要仅以 code 相同认定语义相同。
6. 更新两个引用列，保留所有费用名称、币种、金额、税率和应税劳务名称等历史快照不变。补录中的快照字段同样不重算。审计日志不改。
7. 确认共享引用数量归零后清除已转换共享科目并收紧 fee_settings.organization_id NOT NULL/唯一索引，不能先删再更新外键。
8. 公司初始化只复制一次，同一事务创建公司+税务+科目；不得在读取列表时惰性补种，也不得每次启动覆盖公司设置。

### 实施影响与验证

- Ent 源：新增模板 schema，FeeSetting 归属必填，生成 Ent 和有序 SQL 迁移；禁止手改 generated。
- biz/data/service：模板独立 CRUD（仅系统管理），公司目录纯本公司，订单选项、费用保存、补录审批三个消费入口统一收紧。
- API/frontend：新增最小模板 CRUD 契约并生成客户端；系统管理展示默认目录，公司工作台展示自己的配置。
- 公司创建两条路径接入同一复制 helper；bootstrap 初始化顺序须先存在公共单位/大类/币种与模板，再建公司。
- 测试：共享科目迁移引用完整与快照不变；覆盖冲突明确失败；跨公司 ID 拒绝；模板修改不影响已建公司；新公司独立副本/税务本公司归属；模板不可用于普通费用或补录；事务失败不留下半初始化公司。

### Related specs / external references

- `.trellis/spec/server/backend/organization-shared-masterdata.md`：现行 A/B/C 型规范，需要随实施改写费用目录边界。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：费用、补录、组织实体关系。
- 外部资料：无；全部为本仓源码核查，不依赖第三方版本行为推断。

## Caveats / Not Found

- 本次仅源码研究，没有查询真实数据库，不能声称现存共享科目或同码冲突的数量。
- 未发现生产费用模板初始化；sync-dev 是演示公司数据，不能等价默认目录。
- 仅保留公司科目而不提供系统模板维护入口/新公司复制，会遗漏用户批准的系统管理员提供默认目录需求。
