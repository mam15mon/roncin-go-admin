# Roncin 重构迁移矩阵与设计决策

本文是 P0 阶段的第一份实施产物，记录旧 `roncin` 到 `roncin-go-admin` 的领域范围、目标阶段、当前状态和必须先解决的设计问题。

## 使用规则

- 本文按 `prisma/models/*.prisma` 的领域文件盘点，不把 Prisma 模型机械地一对一翻译成 Ent 实体。
- `迁移状态` 使用四种值：`已完成`、`部分完成`、`未开始`、`明确不迁移`。
- 未作出“迁移”或“明确不迁移”决定的领域，不得在新系统中通过临时字段、隐式兼容或兜底逻辑带过。
- 旧模型的完整清单以旧仓库对应 Prisma 文件为准；本文列出关键模型和容易遗漏的扩展模型。
- 本文中的“已完成”只表示新仓库已有可审阅实现，不代表与旧系统行为已经完全等价。

## 基线事实

| 项目 | 当前事实 |
|---|---|
| 旧系统技术栈 | Next.js App Router、React、Prisma、PostgreSQL |
| 旧 Prisma 结构 | `prisma/models/` 下 12 个领域文件、143 个 `model`、81 个 `enum` |
| 新系统技术栈 | Kratos v3、Go、Ent、PostgreSQL、Ant Design Pro |
| 新 Ent Schema | User、Organization、Membership、Role、Permission、RoleAssignment、Session、AuditLog、Partner，共 10 个 schema 文件（含 mixin） |
| 新 API | AuthService 4 个 RPC，PartnerService 3 个 RPC，共 7 个 RPC |
| 新前端 | 登录、工作台、往来单位和通用错误页；业务管理页目前主要是往来单位 |
| 当前迁移方式 | 没有旧数据迁移工具，也没有长期双写方案 |

## 领域迁移矩阵

| 编号 | 旧领域与来源 | 关键模型 | 新系统状态 | 目标阶段 | 迁移/设计要求 |
|---|---|---|---|---|---|
| D01 | 认证：`00-auth.prisma` | User、Account、Session、VerificationToken、LoginRateLimitBucket、AccessPolicyCacheEpoch | 部分完成 | P2 | 新会话已落到 Go/Ent；补齐用户管理、会话撤销和登录限流；明确旧密码哈希是否可复用；不复制 JWT 权限快照作为后端安全依据。 |
| D02 | 组织与 RBAC：`01-org-rbac.prisma` | Organization、RoleGroup、RoleGroupPermission、UserOrganizationAssignment、UserRoleGroupAssignment、AccessImpactSnapshot | 部分完成 | P2/P3 | 新 `Role`/`Permission`/`RoleAssignment` 已有基础模型；必须先完成旧角色组、权限、组织成员和数据范围的映射决策；AccessImpactSnapshot 是否保留需单独判断。 |
| D03 | 往来单位：`02-party-master.prisma` | Carrier、Party、PartyRole、CustomerProfile、SupplierProfile、AgentProfile、PartyContact、PartyRoleAccount、PartyContract、PartyRoleSettlementRule、PartyAttachment 及日志 | 部分完成 | P5/P6 | 当前 `Partner` 只是简化档案；在模型评审通过前不得继续扩展订单、费用和财务对 Partner 的依赖。必须决定是否保留角色、账户、合同、附件、结算和黑名单。 |
| D04 | 订单核心：`03-order-core.prisma` | Order、OrderProfile、OrderCustomFieldDefinition、OrderCustomFieldValue、服务/货物字典、OrderMilestone、OrderStatusLog、佣金与利润相关模型 | 未开始 | P6 | 先确定订单聚合边界、业务类型、服务类型、货物类别、编号规则、模板选择和状态机；自定义字段必须有权限和版本策略。 |
| D05 | 订单扩展与执行：`04-order-extension.prisma` | OrderContainer、OrderCargoItem、OrderShippingDocument、OrderReleasePod、OrderCollaborator、OrderAbnormalCase、OrderAttachment、OrderAlertTask、OrderPersonnel 等 | 未开始 | P7 | 不能只按“订单执行”笼统迁移；需逐项覆盖集装箱、提单、附件、异常、人员、提醒和审计。 |
| D06 | 报关与 AE 扩展：`04-order-extension.prisma` | OrderCustomsDeclaration、CustomsTaskStatus、OrderAeMonitor、OrderAeTransitInfo、OrderAeInsuranceDraft 及相关枚举 | 未开始 | P6/P7 | 这是独立业务范围，不得因通用订单模型存在而视为自动覆盖；P0 必须决定是否属于 MVP、支持哪些业务类型，以及数据是否迁移。 |
| D07 | 费用与账单：`05-finance-billing.prisma`、`08-dictionaries.prisma` | BillingUnit、FeeSetting、TaxableService、Fee、Bill、BillFee、BillOrderLink、FinanceTag、VerificationRecord、PaymentRecord、InvoiceRecord、VerificationAllocation | 未开始 | P8/P9 | 计费单位、费用项和应税服务直接参与金额计算，随财务聚合一起设计；明确应收应付、含税/不含税、税务模式、账单生成后修改、核销和状态日志，金额规则必须在领域层统一。 |
| D08 | 对账单工作流：`05-finance-billing.prisma` | FinanceStatement、FinanceStatementLineItem、FinanceStatementAuditLog、StatementWorkflowStatus | 未开始 | P9 | 不将“财务工作台”作为默认覆盖；需明确对账单的创建、提交、审核、确认、关闭、撤销和审计流程。 |
| D09 | 汇率：`06-fx.prisma` | ExchangeRateSetting、ExchangeRatePolicy、FxRateQuote、FxQuoteRevision、FxSnapshotRevision、BillFxSnapshotRevision、ChargeLineFxOverride、FxApproval | 未开始 | P8 前置 | 先完成币对、来源、日期规则、快照、锁定、容差和审批；费用/账单只能依赖已确定的汇率快照，不得在页面临时计算。 |
| D10 | 通知与工作流：`07-workflow-notification.prisma` | NotificationMessage、NotificationPreference、NotificationDeliveryLog、OrderReminderRule、OrderReminderDispatch、OrderTrackingEvent | 未开始 | P10 | 区分消息记录、用户偏好、提醒规则、投递日志和跟踪事件；明确失败重试、重复投递和用户可见状态。 |
| D11 | 主数据、参数与模板：`08-dictionaries.prisma` | CurrencyDictionary、CountryDictionary、Region、ContainerSpec、NumberRule、SerialSequence、PageTemplate、StatusTemplate、BusinessRuleTemplate、TemplateBundle、MilestoneTemplate、AirportDictionary、Unlocode* 等 | 部分完成 | P4 | 已完成首批组织级主数据目录、编号规则/并发序列、状态与里程碑模板版本发布和默认版本切换；待补主数据导入契约。BillingUnit、FeeSetting、TaxableService 调整到 D07/P8，页面模板按固定表单决策后置。 |
| D12 | 企业基础资源：`09-enterprise-resources.prisma` | BaseAddress、BaseConsignee、BaseShipper、BaseNotify、BaseResourcePartyRel、BaseNote、BaseTag、BaseBusinessCode、BaseImage、BaseTextSnippet | 未开始 | P4/P5 | 这是订单表单和往来单位的基础资源，不是普通字典；明确与 Party 的关系、文件存储、组织隔离、引用删除和权限。 |
| D13 | 幂等：`10-idempotency.prisma` | IdempotencyKey | 未开始 | P1/P7 | API 写入幂等和后台任务幂等分别定义；明确键的作用域、请求摘要、过期、冲突响应和清理策略。 |
| D14 | 数据维护作业：`11-data-maintenance-jobs.prisma` | RegionSyncJob、PortImportJob；另含 `UnlocodeImportBatch` | 未开始 | P4/P7/P10 | 主数据导入契约在 P4 定义，任务租约/重试/死信在 P7 建立，运维查询和回放在 P10 接入；不得与普通 IntegrationTask 混成无类型任务。 |

## 必须先完成的设计决策

### M-001：存量数据策略

在 P0 选择一条路径：

- 冷启动：只初始化组织、管理员、权限和必要主数据，旧系统历史数据只读归档。
- 保留数据：执行一次性迁移，明确 ID、枚举、组织、用户、附件和审计映射。

不采用长期双写，除非另行批准并定义一致性、失败补偿和下线时间。

当前实施暂按“冷启动”推进：新系统初始化组织、管理员、权限和主数据；旧系统数据库保留只读归档，不在新系统中加入旧 API、旧运行时或长期双写兼容。正式生产切换前仍需完成旧数据保留范围、归档位置和查询责任确认。

### M-002：Partner 与 Party 的目标模型

需要明确以下范围是否进入新系统：

- 一个 Party 多角色，还是客户/供应商拆成独立资源。
- 客户、供应商、代理档案是否保留独立扩展表。
- 联系人、账户、合同、结算规则、附件、黑名单和角色变更日志是否迁移。
- 新订单、费用和财务引用的是 Party 聚合 ID，还是特定角色 ID。

未决前，当前 `Partner` 只能视为试验性基础资料实现，不能作为完整 Party 的最终契约。

当前目标决定：`Partner` 作为新系统中的 Party 聚合名称继续保留 API 稳定性，但内部扩展独立的角色、联系人、账户、合同、结算规则和黑名单边界；订单和费用只引用 Party 聚合 ID，不直接引用某个旧 Prisma 表。客户、供应商、代理和承运人通过显式角色关系区分，不再用 `customer/supplier/both` 三值字段承载所有业务语义。

### M-003：旧 RBAC 到新 RBAC 的映射

旧模型到新模型的候选映射如下，最终名称和语义需评审：

| 旧模型 | 新模型候选 | 必须确认 |
|---|---|---|
| RoleGroup | Role | 角色是组织级还是全局级，是否允许同码跨组织复用 |
| RoleGroupPermission | Permission + RoleAssignment 关系 | 权限定义是否全局唯一，角色是否只保存授权关系 |
| UserOrganizationAssignment | Membership | 主组织、启停、组织树和成员有效期 |
| UserRoleGroupAssignment | RoleAssignment | 角色作用域、过期时间、数据范围和继承规则 |
| AccessImpactSnapshot | 审计/权限变更记录 | 是保留快照，还是只保留变更事件 |
| global admin / dataScope | Role.data_scope + 特殊系统权限 | 全局管理员是否绕过组织范围，后端如何强制隔离 |

当前 P2 实施结论：新系统采用组织级 `Role`，权限定义由后端 Manifest 维护，角色只保存授权关系；每个当前会话同时保留角色范围与该角色的权限集合，后端按“角色自身同时拥有权限且范围满足要求”判断请求，不能用多个角色的权限并集去扩大另一个角色的数据范围。组织管理接口要求 `all` 范围，用户、角色、权限目录和往来单位接口要求当前组织及以上范围。组织列表只返回当前组织和直接子组织，新增组织只能挂在当前组织下；根组织由冷启动初始化流程创建。

这项结论修正了旧系统中“角色组、权限快照、数据范围分散在多处”的风险，但组织树的递归范围、角色停用策略和历史 `AccessImpactSnapshot` 是否保留仍属于 P2/P3 门禁，不能以当前实现默认为已迁移。

密码重置采用独立管理用例，不复用用户资料更新：新密码在领域层校验并哈希，数据层把密码更新与该用户现有会话撤销放在同一事务中；审计日志查询只读且固定按当前组织过滤，避免旧系统中修改用户资料时遗漏会话失效、或跨组织查看安全日志的问题。

### M-004：模板与表单策略

- 固定表单 MVP：先迁移编号、服务/货物字典、状态和里程碑模板，页面模板后置。
- 动态表单 MVP：P4 必须同时迁移 PageTemplate、字段分组、字段显隐/只读和版本发布。

不能一边按固定表单实现，一边保留未定义的动态模板兼容字段。

当前实施选择固定表单 MVP：先实现服务类型、货物类别、币种、地区/港口/机场、承运人、箱型、编号规则和订单状态模板；旧 `PageTemplate`、`BusinessRuleTemplate` 和 `TemplateBundle` 不做隐式兼容，待字段契约和版本发布模型评审后再单独实现。

### M-005：汇率与财务口径

费用开发前必须冻结：金额精度、舍入、税务模式、含税/不含税、应收应付方向、币对、汇率来源、日期规则、快照锁定、容差审批、账单后修改和冲销规则。

当前实施口径草案：金额使用 PostgreSQL `numeric(18, 6)`，展示按业务币种规则舍入；费用保存原币金额、币种和计算后本位币快照；汇率必须带来源、有效日期和版本，账单生成后快照不可被普通编辑覆盖。未有汇率快照时拒绝生成账单，不在页面使用临时汇率。

### M-006：订单扩展范围

对报关和 AE 扩展分别标记：迁移、后置或明确不迁移，并记录业务理由。不能把 `OrderCustomsDeclaration` 或 `OrderAe*` 隐藏在“订单扩展已完成”的笼统状态中。

当前 MVP 将报关和 AE 扩展后置：订单核心不创建这些字段的空兼容列，也不伪装支持相关状态；待业务确认适用线路、责任主体和验收场景后，在 P7 以独立扩展聚合实现。

## 旧系统问题与新系统修正原则

以下不是对旧系统所有实现的否定，而是迁移时已经能确认或需要重点防止复制的设计风险。

| 旧系统风险 | 证据/表现 | 新系统修正 |
|---|---|---|
| Prisma 模型容易被当成业务边界 | 143 个模型分散在 12 个文件，关系复杂 | 以业务聚合和用例为边界；Ent 只负责持久化，禁止把生成实体直接泄漏到 `biz`/`service` |
| 开发期 `db push --accept-data-loss` 不适合生产演进 | 旧项目 AGENTS 明确允许开发期 Schema push | 新系统生产关闭 `auto_migrate`，使用审阅后的 Ent/Atlas 迁移；冷启动和生产迁移分开 |
| NextAuth 权限快照与后端实时权限语义不同 | `src/auth.ts` 明确说明 JWT permissions 只作 UI 提示 | 新系统后端每次请求以数据库会话、组织和权限为权威；前端只消费 `/api/v1/auth/me` 的结果 |
| RoleGroup 与新 Role 结构不等价 | 旧有 RoleGroupPermission/UserRoleGroupAssignment，新有 Role/Permission/RoleAssignment | 先做 M-003 映射，不直接按表名迁移；用组织、角色和数据范围验收 |
| 动态模板配置复杂，容易变成隐式协议 | 旧系统有 PageTemplate、StatusTemplate、BusinessRuleTemplate、TemplateBundle 多层关系 | 模板必须版本化、发布态明确、输入结构有契约；订单运行时只使用已发布版本或明确的固定模板 |
| 编号规则缺少组织边界且状态职责混杂 | 旧 `NumberRule.docType` 全局唯一并内置 `currentValue`，无法让各组织独立配置和发号 | 新系统按“组织 + 单据类型”唯一保存格式规则，把可变序列拆为独立实体；发号事务锁定规则后再按重置周期分配 |
| 序列唯一键可能造成跨单据碰撞 | 旧 `SerialSequence` 唯一键只有 `scopeKey + dateKey`，虽保存 `docType`/`businessType` 却未纳入唯一性 | 新系统以“规则 ID + 周期”唯一，不允许不同单据规则共享计数行；序列耗尽时事务回滚，不写入越界值 |
| 状态模板默认版本缺少数据库级唯一保障 | 旧 `StatusTemplate.isDefault` 只是普通布尔字段，并发发布可能留下多个默认版本 | 新系统对“组织 + 业务类型”的默认模板建立部分唯一索引；发布和恢复旧版本使用不同显式动作并分别审计 |
| 里程碑模板存在多源读取和静默回退 | 旧订单会依次读取 `MilestoneTemplate`、`BusinessSwitchSetting` JSON，再回退到代码内默认模板；配置缺失和存储不可用可能表现成同一套默认流程 | 新系统只使用组织级、版本化且已发布的里程碑模板；节点依赖在创建时校验存在性、启用状态和无环，不内置默认流程或静默回退 |
| 财务规则分散且模型较多 | Fee、Bill、FinanceStatement、FX 快照、税务枚举跨多个文件 | 先冻结财务口径，再由 `biz` 持有金额和状态规则；页面不自行拼接金额 |
| 财务参数被包装成普通字典 | `BillingUnit`、`FeeSetting`、`TaxableService` 位于 dictionaries 文件，但默认税率和费用关联会直接改变金额结果 | 不按文件位置将它们塞入通用主数据；在 P8 与含税/未税、税种、精度、舍入和快照规则共同建模 |
| 数据维护作业与集成任务容易混淆 | 旧系统同时有 RegionSyncJob、PortImportJob、UnlocodeImportBatch 和 IntegrationTask | 新系统区分任务类型、幂等键、租约、重试和死信；主数据导入不伪装成普通业务集成 |
| 登录限流曾由独立表和原始 SQL 支撑 | `src/lib/auth/login-rate-limit-service.ts` 直接操作 LoginRateLimitBucket | 新系统封装限流仓储/服务，统一错误、清理、指标和审计，业务层不散落 SQL |
| 泛化审计不能替代领域审计 | 旧系统同时存在操作日志、状态日志、财务/模板审计等 | 新系统保留通用安全审计，同时为金额、状态和模板发布保留必要的领域审计，不把所有细节压成一条 JSON |

## P0 完成验收

- 12 个旧 Prisma 领域文件全部在矩阵中有记录，没有“未知”领域。
- 每一行均有目标阶段、迁移状态和“迁移/后置/明确不迁移”决策。
- M-001 至 M-006 有明确结论或记录为阻塞项，不用临时兼容分支替代。
- `Partner`/`Party`、RBAC、页面模板、汇率、报关/AE、企业资源和财务对账单均有单独验收口径。
- 旧系统问题已转换成新系统的架构约束，而不是只写在说明里。
- P0 完成后才能把订单、费用和财务 Proto/Ent Schema 作为稳定契约实现。
