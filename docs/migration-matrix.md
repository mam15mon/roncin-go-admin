# Roncin 重构迁移矩阵与设计决策

> 现状核对：2026-11-09。旧模型清单和阶段编号保留历史语义；“已完成”只说明仓库有实现，
> 不表示旧系统行为等价或真实环境验收通过。现行规范与代码优先于早期候选映射，
> 统一入口见[文档导航](README.md)。

> 2026-10-30 更新：独立订单履约里程碑与旧里程碑模板均已退役；旧模型清单和早期阶段记录仅保留历史含义，当前系统不再提供该能力。删除契约见 `.trellis/spec/server/backend/order-milestone-retirement.md`。

本文起源于 P0 阶段，保留旧 `roncin` 到 `roncin-go-admin` 的领域范围、阶段编号和取舍，
并在迁移矩阵及 M-003/M-005 更新当前工作台、标签退役和财务口径。

> 订单人员更新：独立协作团队维护入口和写接口已退役；人员配置保留在订单核心与表单，人员表、只读接口、创建通知及提成归属不删除。

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
| 新 Ent Schema | 已包含 admin、masterdata、partner、order 等领域；当前 Schema 清单以 `server/internal/data/ent/schema/` 为准 |
| 新 API | 已包含 AuthService、AdminService、MasterDataService、PartnerService、OrderService 等 Proto 定义与 RPC |
| 新前端 | Vite + React Router；登录、公司/系统工作台、管理后台、主数据、往来单位、SE/AE 订单与财务页面；主详情使用内存编辑和离开保护，新建/HBL 保留持久草稿 |
| 当前迁移方式 | 没有旧数据迁移工具，也没有长期双写方案 |

## 领域迁移矩阵

| 编号 | 旧领域与来源 | 关键模型 | 新系统状态 | 目标阶段 | 迁移/设计要求 |
|---|---|---|---|---|---|
| D01 | 认证：`00-auth.prisma` | User、Account、Session、VerificationToken、LoginRateLimitBucket、AccessPolicyCacheEpoch | 部分完成 | P2 | 数据库会话、会话撤销、登录与自助密码重置已有实现，支持钉钉及可配置通行密钥；工作台资格由当前成员关系校验。旧密码与历史账号迁移另行决策，不使用 JWT 权限快照作为安全依据。 |
| D02 | 组织与 RBAC：`01-org-rbac.prisma` | Organization、RoleGroup、RoleGroupPermission、UserOrganizationAssignment、UserRoleGroupAssignment、AccessImpactSnapshot | 部分完成 | P2/P3 | Role/Permission/Membership/RoleAssignment 已实现；权限清单由后端 Manifest 维护。角色数据范围字段已退役，旧 self 角色由正式迁移停用；角色库归属系统/公司工作台，经营边界由当前工作台与具体权限推导，系统管理不代办公司经营。 |
| D03 | 往来单位：`02-party-master.prisma` | Carrier、Party、PartyRole、CustomerProfile、SupplierProfile、AgentProfile、PartyContact、PartyRoleAccount、PartyContract、PartyRoleSettlementRule、PartyAttachment 及日志 | 部分完成 | P5/P6/P7 | Partner 核心、账户、合同、结算规则、信用与固定导入导出已有实现；默认账户、角色黑名单、散客账期及出款账户分别有约束。订单、费用和财务引用 Partner 聚合 ID；共享企业资源与订单附件各有独立存储契约，不把订单 OSS 能力等同于所有 Partner 附件闭环。 |
| D04 | 订单核心：`03-order-core.prisma` | Order、OrderProfile、OrderCustomFieldDefinition、OrderCustomFieldValue、服务/货物字典、OrderMilestone、OrderStatusLog、佣金与利润相关模型 | 部分完成 | P6/P7/P8/P9 | SE 海运出口及 AE 空运出口已注册录单入口，共用订单聚合并有品类单证与运输扩展。独立里程碑已退役，保留操作进度；人工审核锁、完结/退关、并发版本与费用按各专项契约校验，不沿用早期仅草稿可编辑。 |
| D05 | 订单扩展与执行：`04-order-extension.prisma` | OrderContainer、OrderCargoItem、OrderShippingDocument、OrderReleasePod、OrderCollaborator、OrderAbnormalCase、OrderAttachment、OrderAlertTask、OrderPersonnel 等 | 部分完成 | P7 | 海运共享 MBL/运输执行、每票 HBL、共享箱、拆票改配及不可变历史已有实现；空运共享 MAWB、每票 HAWB 与重量确认已有实现。独立协作写入口退役，人员通过订单表单维护；订单 OSS 支持上传、私有访问与共享引用解除，复杂分支及外部系统仍需独立验收。 |
| D06 | 报关与线路特有执行信息：`04-order-extension.prisma` | OrderCustomsDeclaration、CustomsTaskStatus、OrderAeMonitor、OrderAeTransitInfo、OrderAeInsuranceDraft 及相关枚举 | 未开始 | P6/P7 | 这些旧模型不是独立订单域。报关按订单执行能力设计；`OrderAe*` 只表示空运出口线路特有的执行信息，并附着于统一订单聚合。P0 必须决定 MVP 字段范围、适用线路以及数据是否迁移。 |
| D07 | 费用与账单：`05-finance-billing.prisma`、`08-dictionaries.prisma` | BillingUnit、FeeSetting、TaxableService、Fee、Bill、BillFee、BillOrderLink、FinanceTag、VerificationRecord、PaymentRecord、InvoiceRecord、VerificationAllocation | 部分完成 | P8/P9 | 费用目录、订单费用、分批账单/NORMAL与NETTING批次、发票、资金流水、核销及提成已有契约和实现。订单资料与费用统一人工审核锁，建账无需先审核，待审更正仅占用源费。业务标签已退役；财务列表及写入绑定当前公司。脚本存在不代表本轮验收通过。 |
| D08 | 对账单工作流：`05-finance-billing.prisma` | FinanceStatement、FinanceStatementLineItem、FinanceStatementAuditLog、StatementWorkflowStatus | 未开始 | P9 | 不将“财务工作台”作为默认覆盖；需明确对账单的创建、提交、审核、确认、关闭、撤销和审计流程。 |
| D09 | 汇率：`06-fx.prisma` | ExchangeRateSetting、ExchangeRatePolicy、FxRateQuote、FxQuoteRevision、FxSnapshotRevision、BillFxSnapshotRevision、ChargeLineFxOverride、FxApproval | 部分完成 | P8 前置 | 汇率设置、自定义汇率设置、汇率时间标准、汇率导入批次与账单汇率快照已落地，费用录入支持手动汇率重写（override）；Fx 报价、报价修订与审批流仍未建模，费用/账单不得在页面临时计算的约束继续生效。 |
| D10 | 通知与工作流：`07-workflow-notification.prisma` | NotificationMessage、NotificationPreference、NotificationDeliveryLog、OrderReminderRule、OrderReminderDispatch、OrderTrackingEvent | 部分完成 | P10 | 类型化通知意图、投递明细、租约/重试/死信、钉钉 Worker 和解锁审批派发回调已有实现；独立费用审核退役后提醒渠道仍可供补审使用。四类消费者及其恢复检查见后台任务规范，外部收件/审批结果需真实环境验证；不迁移任意 JSON payload。 |
| D11 | 主数据、参数与模板：`08-dictionaries.prisma` | CurrencyDictionary、CountryDictionary、Region、ContainerSpec、NumberRule、SerialSequence、PageTemplate、StatusTemplate、BusinessRuleTemplate、TemplateBundle、MilestoneTemplate、AirportDictionary、Unlocode* 等 | 部分完成 | P4 | 公共币种、机场、海港、航司、船公司及行政区划已有全局目录及对应初始化/同步入口；系统工作台维护、公司按权限消费。编号规则与状态模板保留，独立里程碑及旧模板已退役；财务目录归属 D07，前端使用固定代码表单，不恢复旧低代码运行时。 |
| D12 | 企业基础资源：`09-enterprise-resources.prisma` | BaseAddress、BaseConsignee、BaseShipper、BaseNotify、BaseResourcePartyRel、BaseNote、BaseTag、BaseBusinessCode、BaseImage、BaseTextSnippet | 已完成 | P4/P5 | 统一企业资源基表、分类详情、企业关联、地址/人员、单证主体及图片已有实现；旧 partner_shipping_presets 已替换。TAG 类型、标签组和订单/费用/账单标签关联已正式退役。图片删除在事务中删资源并入队，提交后 Worker 删对象，历史建设计划只作交付记录。 |
| D13 | 幂等：`10-idempotency.prisma` | IdempotencyKey | 部分完成 | P1/P7 | 订单创建、账单/批次、补审、拆票等写路径已有幂等键或请求指纹，任务也有独立幂等边界；各动作按作用域、请求意图和冲突规则验证，不笼统声称 API 幂等尚未开始，也不建立无现实需求的万能幂等平台。 |
| D14 | 数据维护作业：`11-data-maintenance-jobs.prisma` | RegionSyncJob、PortImportJob；另含 `UnlocodeImportBatch` | 部分完成 | P4/P7/P10 | 主数据外部文件解析、冲突预检、幂等写入及来源摘要已有 CLI。通知、解锁审批、对象删除等 Worker 已执行租约任务，按类型处理到期与恢复；不等同于所有旧维护作业、查询历史和运维告警均已迁移。 |

## 必须先完成的设计决策

### M-001：存量数据策略

在 P0 选择一条路径：

- 冷启动：只初始化组织、管理员、权限和必要主数据，旧系统历史数据只读归档。
- 保留数据：执行一次性迁移，明确 ID、枚举、组织、用户、附件和审计映射。

不采用长期双写，除非另行批准并定义一致性、失败补偿和下线时间。

当前实施暂按“冷启动”推进：新系统初始化组织、管理员、权限和主数据；旧系统数据库保留只读归档，不在新系统中加入旧 API、旧运行时或长期双写兼容。正式生产切换前仍需完成旧数据保留范围、归档位置和查询责任确认。

### M-002：Partner 与 Party 的目标模型

本决策覆盖以下范围：

- 一个 Party 多角色，还是客户/供应商拆成独立资源。
- 客户、供应商、代理档案是否保留独立扩展表。
- 联系人、账户、合同、结算规则、附件、黑名单和角色变更日志是否迁移。
- 新订单、费用和财务引用的是 Party 聚合 ID，还是特定角色 ID。

当前目标决定：`Partner` 作为新系统中的 Party 聚合名称继续保留，但内部使用独立的角色、联系人、账户、合同、结算规则和黑名单边界；订单和费用只引用 Party 聚合 ID，不直接引用某个旧 Prisma 表。客户、供应商、代理和承运人通过显式角色关系区分，不再用 `customer/supplier/both` 三值字段承载所有业务语义。

第一组已落地以下修正：

- 编码、归一化法人名称和统一社会信用代码均按组织建立唯一约束，不沿用旧系统的全局唯一。
- 联系人从 Partner 主表拆出，数据库使用条件唯一索引保证每个 Partner 最多一个主联系人。
- 别名保存归一化值并按 Partner 唯一，避免仅靠页面判断重复。
- 供应商黑名单只能通过带原因的专用命令变更，并写入业务审计；存在黑名单状态时禁止通过普通编辑删除供应商角色。
- 页面为仓库内固定 TypeScript/React 表单，没有低代码配置、动态脚本或代码默认回退。

第二组至第四组已完成账户、合同、角色级结算规则和附件元数据登记；真实对象存储上传、授权下载、删除、导入和导出作为后续固定接口/作业实现；信用额度留到 P8 金额模型。旧 `meta`、`advantages` 等无明确契约的 JSON 字段不原样迁移。旧合同的 `isCompanyTemplate`、多方主体评价等字段没有明确新业务契约，当前不迁移；旧账户和附件软删除字段也不复制，分别使用账户启停和后续文件生命周期策略表达。

### M-003：旧 RBAC 到新 RBAC 的映射

旧模型映射的历史候选已收敛到以下现行边界：

| 旧概念 | 当前模型与边界 |
| --- | --- |
| RoleGroup | Role；仅归属系统/公司工作台，部门成员使用所属公司角色库 |
| RoleGroupPermission | Permission 目录来自 Manifest，角色保存授权关系 |
| UserOrganizationAssignment | Membership；有效成员资格是进入公司工作台的前提 |
| UserRoleGroupAssignment | RoleAssignment；校验绑定、角色与所属工作台，不恢复角色范围配置 |
| global admin / dataScope | Role.data_scope 已删除；系统工作台治理公共资料与授权，公司经营需真实公司资格及权限 |
| AccessImpactSnapshot | 不据旧模型名推定已迁移，授权审计与当前接口为准 |

角色上的 `all/organization/self` 配置已退役，不再作为可编辑授权真相。
每个权限的有效能力与当前工作台边界由服务端解析，前端消费 `permissionCapabilities`，
不得把角色展示信息拼成第二套范围。旧 self 角色按正式迁移停用，bootstrap 不重新启用。
详见[角色范围退役](../.trellis/spec/server/backend/retire-role-data-scope.md)、
[角色库归属](../.trellis/spec/server/backend/role-workspace-ownership.md)及
[工作台会话](../.trellis/spec/server/backend/auth-session-org-switch.md)。

密码重置采用独立管理用例，不复用用户资料更新：新密码在领域层校验并哈希，数据层把密码更新与该用户现有会话撤销放在同一事务中；审计日志查询只读且固定按当前组织过滤，避免旧系统中修改用户资料时遗漏会话失效、或跨组织查看安全日志的问题。

### M-004：模板与表单策略

- 固定表单 MVP：编号、服务/货物字典和状态模板按明确契约实现；原里程碑模板现已退役，页面动态模板不自动纳入范围。
- 动态表单 MVP：P4 必须同时迁移 PageTemplate、字段分组、字段显隐/只读和版本发布。

不能一边按固定表单实现，一边保留未定义的动态模板兼容字段。

当前实施选择固定表单 MVP：先实现服务类型、货物类别、币种、地区/港口/机场、承运人、箱型、编号规则和订单状态模板；旧 `PageTemplate`、`BusinessRuleTemplate` 和 `TemplateBundle` 不做隐式兼容，待字段契约和版本发布模型评审后再单独实现。

### M-005：汇率与财务口径

费用开发前必须冻结：金额精度、舍入、税务模式、含税/不含税、应收应付方向、币对、汇率来源、日期规则、快照锁定、容差审批、账单后修改和冲销规则。

当前订单费用/账单金额使用 PostgreSQL `numeric(28,8)`，汇率 `numeric(18,8)`；订单费用
数量/单价为 `numeric(18,4)`，税率为 `numeric(5,2)`，具体字段以 Ent Schema 为准。
界面金额统一 `formatAmount` 展示两位，数量/单价/汇率使用 `trimDecimal`；输入和提交保留原始值。
普通账单固定按费用币种拆分，在账单日解析汇率并保存快照，不提供跨币种目标账单旁路；
已生成快照不随配置变化漂移，详见[账单币种契约](../.trellis/spec/server/backend/finance-bill-currency.md)。

### M-006：订单扩展范围

对报关和旧 `OrderAe*` 线路执行信息分别标记：迁移、后置或明确不迁移，并记录业务理由。不能把这些能力隐藏在“订单扩展已完成”的笼统状态中，也不能由旧模型名称推导出独立 AE 订单域。

当前 MVP 将报关和线路特有执行信息后置：订单核心不创建这些字段的空兼容列，也不伪装支持相关状态；待业务确认适用线路、责任主体和验收场景后，在 P7 作为统一订单聚合的执行信息实现，不新建 AE 订单聚合。

## 旧系统问题与新系统修正原则

以下不是对旧系统所有实现的否定，而是迁移时已经能确认或需要重点防止复制的设计风险。

| 旧系统风险 | 证据/表现 | 新系统修正 |
|---|---|---|
| Prisma 模型容易被当成业务边界 | 143 个模型分散在 12 个文件，关系复杂 | 以业务聚合和用例为边界；Ent 只负责持久化，禁止把生成实体直接泄漏到 `biz`/`service` |
| 开发期 `db push --accept-data-loss` 不适合生产演进 | 旧项目 AGENTS 明确允许开发期 Schema push | 新系统生产关闭 `auto_migrate`，使用审阅后的 Ent/Atlas 迁移；冷启动和生产迁移分开 |
| NextAuth 权限快照与后端实时权限语义不同 | `src/auth.ts` 明确说明 JWT permissions 只作 UI 提示 | 新系统后端每次请求以数据库会话、组织和权限为权威；前端只消费 `/api/v1/auth/me` 的结果 |
| RoleGroup 与新 Role 结构不等价 | 旧有 RoleGroupPermission/UserRoleGroupAssignment，新有 Role/Permission/RoleAssignment | 按 M-003 当前工作台/成员/角色/权限边界验收，不按旧表名或已删角色范围迁移 |
| 动态模板配置复杂，容易变成隐式协议 | 旧系统有 PageTemplate、StatusTemplate、BusinessRuleTemplate、TemplateBundle 多层关系 | 模板必须版本化、发布态明确、输入结构有契约；订单运行时只使用已发布版本或明确的固定模板 |
| 编号规则缺少组织边界且状态职责混杂 | 旧 `NumberRule.docType` 全局唯一并内置 `currentValue`，无法让各组织独立配置和发号 | 新系统按“组织 + 单据类型”唯一保存格式规则，把可变序列拆为独立实体；发号事务锁定规则后再按重置周期分配 |
| 序列唯一键可能造成跨单据碰撞 | 旧 `SerialSequence` 唯一键只有 `scopeKey + dateKey`，虽保存 `docType`/`businessType` 却未纳入唯一性 | 新系统以“规则 ID + 周期”唯一，不允许不同单据规则共享计数行；序列耗尽时事务回滚，不写入越界值 |
| 状态模板默认版本缺少数据库级唯一保障 | 旧 `StatusTemplate.isDefault` 只是普通布尔字段，并发发布可能留下多个默认版本 | 新系统对“组织 + 业务类型”的默认模板建立部分唯一索引；发布和恢复旧版本使用不同显式动作并分别审计 |
| 里程碑模板存在多源读取和静默回退 | 旧订单会依次读取 `MilestoneTemplate`、`BusinessSwitchSetting` JSON，再回退到代码内默认模板；配置缺失和存储不可用可能表现成同一套默认流程 | 独立里程碑和模板已退役，保留订单操作进度；不恢复旧多源里程碑或静默回退 |
| 财务规则分散且模型较多 | Fee、Bill、FinanceStatement、FX 快照、税务枚举跨多个文件 | 先冻结财务口径，再由 `biz` 持有金额和状态规则；页面不自行拼接金额 |
| 财务参数被包装成普通字典 | `BillingUnit`、`FeeSetting`、`TaxableService` 位于 dictionaries 文件，但默认税率和费用关联会直接改变金额结果 | 不按文件位置将它们塞入通用主数据；在 P8 与含税/未税、税种、精度、舍入和快照规则共同建模 |
| 数据维护作业与集成任务容易混淆 | 旧系统同时有 RegionSyncJob、PortImportJob、UnlocodeImportBatch 和 IntegrationTask | 新系统区分任务类型、幂等键、租约、重试和死信；主数据导入不伪装成普通业务集成 |
| 登录限流曾由独立表和原始 SQL 支撑 | `src/lib/auth/login-rate-limit-service.ts` 直接操作 LoginRateLimitBucket | 新系统封装限流仓储/服务，统一错误、清理、指标和审计，业务层不散落 SQL |
| 泛化审计不能替代领域审计 | 旧系统同时存在操作日志、状态日志、财务/模板审计等 | 新系统保留通用安全审计，同时为金额、状态和模板发布保留必要的领域审计，不把所有细节压成一条 JSON |

## P0 完成验收

- 12 个旧 Prisma 领域文件全部在矩阵中有记录，没有“未知”领域。
- 每一行均有目标阶段、迁移状态和“迁移/后置/明确不迁移”决策。
- M-001 至 M-006 有明确结论或记录为阻塞项，不用临时兼容分支替代。
- `Partner`/`Party`、RBAC、页面模板、汇率、报关与线路特有执行信息、企业资源和财务对账单均有单独验收口径。
- 旧系统问题已转换成新系统的架构约束，而不是只写在说明里。
- P0 完成后才能把订单、费用和财务 Proto/Ent Schema 作为稳定契约实现。
