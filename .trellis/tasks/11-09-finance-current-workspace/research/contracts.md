# Research: 财务当前公司边界与公司候选契约退役

- Query: 财务页面及弹窗统一当前公司后，后端是否需要新增范围限制，哪些公司候选契约可精确删除？
- Scope: internal
- Date: 2026-11-09

## Findings

### 1. 结论

后端已经把所有经营财务权限收敛到当前启用公司；本任务无需扩大权限/事务设计，无需改Ent或迁移数据库。列表及现存业务API的显式 `organization_id` 可以保留，前端统一传当前工作区ID即可。公司选择撤除后，建议同一交付包纵向退役无人消费的 `ListFinanceOrganizationOptions` RPC及专属三层实现和消息，按源重新生成。共享 `FinanceOrganizationPurpose` 不能整体删除：结算单位候选仍以它选择收付创建或核销创建权限；仅保留0、4、9，删除其余纯公司候选值并reserved原编号/名称，既不重排也不新建兼容层。

### 2. 已有后端范围底线

| 文件与行 | 证据与含义 |
| --- | --- |
| `server/internal/access/manifest.go:462` | `IsCompanyBusinessPermission` 把 `system.finance.*`（仅费用设置例外）归入公司经营权限；本期财务列表和写入均在该范围。 |
| `server/internal/biz/auth.go:215`、`:257`、`:837` | 权限集和范围走同一有效授权来源；无对应工作台/角色授权直接 `ErrPermissionDenied`。系统工作区不会投影这些经营权限。 |
| `server/internal/biz/auth.go:822` | `CanOperateBusiness` 要求公司kind、非空ID、组织投影启用，真实WorkspaceOrganizationID须与Organization.ID一致；系统、部门、组、空身份均不能办理。 |
| `server/internal/biz/auth.go:861` | 即便角色底层有All/Tree范围，公司工作台读写集合均只保留当前公司ID。 |
| `server/internal/service/organization_scope.go:12`、`:32` | 列表无organization参数时取该动作有效范围，当前经营范围实际仅一家公司；显式非法UUID返回参数错误，其他公司UUID返回权限错误，不回退到当前公司、不静默纠错。 |
| `server/internal/service/settlement_fee_ledger.go:48` | 费用台账以FinanceFeeRead解析组织集合。 |
| `server/internal/service/settlement_bill.go:44`、`settlement_cashflow.go:33`、`settlement_netting.go:35`、`settlement_commission.go:26` | 财务列表全部复用上述限定；其他列表同模式。 |
| `server/internal/service/settlement_cashflow.go:58`、`:62` | 创建收付必须解析显式ID并命中当前动作WritableOrganizationIDs，缺失/非法ID参数错误，跨公司权限错误。 |
| `server/internal/service/settlement_test.go:415` | 既有账单批次预览/创建验证来源声明与费用公司一致，混合/不匹配失败，跨公司声明拒绝；不因删选择框移除此校验。 |
| `server/internal/server/auth.go:253` | 财务权限中间件验证动作和有效范围，service传allowed IDs，仓储最终按ID+组织过滤；不需要给公司候选退役添加新中间件例外。 |

前端仍应显式防止身份未就绪发请求；后端已经安全收敛不等于UI可以发送未定范围请求或把公司ID从可编辑Form取出。

### 3. 可退役的公司候选纵向链

| 文件与行 | 删除/同步范围 |
| --- | --- |
| `server/api/finance/v1/settlement.proto:113` | `ListFinanceOrganizationOptions`，GET `/api/v1/finance/organization-options`。 |
| `server/api/finance/v1/settlement.proto:1046` | `FinanceOrganizationOption`、`ListFinanceOrganizationOptionsRequest/Response` 均只支撑上述RPC；无其他Proto字段依赖。 |
| `server/internal/service/settlement.go:35` | 公司候选service方法删除；保留下一方法 `ListFinanceSettlementPartyOptions`。 |
| `server/internal/biz/settlement.go:119`、`:137`、`:154` | Repo方法、FinanceOrganizationOption领域类型、Usecase方法删除；同步 `settlement_test.go` 及其他Repo stub实现（全仓搜索方法名）。 |
| `server/internal/data/settlement.go:223` | 仓储公司查询删除；清除因此不再使用的organizationent/predicate import前先核对同文件用途。 |
| `server/internal/server/access_rules_gen.go:149` | 通过正式生成去掉公司候选authenticated注册项，不手工补生成文件；HTTP/gRPC绑定和OpenAPI同理。 |
| `web/src/services/roncin/settlementService.ts:1251` | 公司候选客户端函数、对应typings由OpenAPI生成移除。 |

正式手写前端调用全部处于本次范围：`features/finance/bill-creation/BillCreationWorkbench.tsx:152`；`pages/finance/{fees,cashflows,invoices,verifications,nettings,commissions}/index.tsx`；`VerificationWorkbench.tsx:108`；`InvoiceCreateModal.tsx:68`；`CommissionCreateModal.tsx:128`；`CommissionRulesDrawer.tsx:79,96`。全仓未找到其他正式消费者；测试mock与对应fixture必须同步。未找到scripts命令直接调用该RPC或HTTP路径。

这些公司候选的baseCurrency字段未发现被本次消费者用于经济计算，主要映射label/value；如需要当前公司本币，`server/api/auth/v1/auth.proto:268` 的当前Organization已经返回id/code/name/base_currency/kind，不需要保留候选请求只为拿公司信息。

### 4. 共享purpose保留与精简

`server/api/finance/v1/settlement.proto:1050` 的 `ListFinanceSettlementPartyOptionsRequest.purpose` 复用FinanceOrganizationPurpose。真实保留调用：`web/src/pages/finance/cashflows/index.tsx:651` 按CASHFLOW_CREATE查询结算单位；`web/src/pages/finance/verifications/VerificationWorkbench.tsx:139` 按VERIFICATION_CREATE查询结算单位。service `settlement.go:59` 只接受这两个用途，并使用不同动作权限；不能用通用客商候选替换而意外增加权限需求。

| 枚举编号 | 当前名称后缀 | 处理建议 |
| --- | --- | --- |
| 0 | UNSPECIFIED | 保留零值，仍由service显式拒绝业务查询。 |
| 4 | CASHFLOW_CREATE | 保留原值与名称，结算单位候选仍消费。 |
| 9 | VERIFICATION_CREATE | 保留原值与名称，结算单位候选仍消费。 |
| 1,2,3,5,6,7,8,10,11,12 | BILL_READ、INVOICE_READ、CASHFLOW_READ、VERIFICATION_READ、COMMISSION_READ、COMMISSION_MANAGE、FEE_READ、INVOICE_CREATE、BILL_CREATE、NETTING_READ | 仅公司候选链使用，移除并在同enum reserved原编号及全名，不复用编号。 |

`financeOrganizationPurposePermission`（service/settlement.go:92）改成只识别4、9即可，或就地收敛到结算单位候选helper；不需要另造抽象/重命名整个公共enum。原 `settlement_test.go:298`、`:339` 的invoice/bill purpose映射断言须删除或改为退役值拒绝测试，旁边真实建账/开票候选权限测试必须保留。

### 5. 前端工作区与kind模式

- `web/src/access.ts:164`、`:168` 以生成的AuthOrganizationKind判定系统/公司，公开 `canOperateBusiness`；`:171` 的 `canOperateOrganization(id)` 额外要求ID与当前公司完全相等。业务入口优先复用这些已有结果，不能用新硬编码数字或把有任意organization ID视为公司。
- `web/src/pages/admin/components/users/userConstants.ts:33` 的 `isWorkspaceKindValue` 是治理模块判定“系统或公司都是工作台”的函数，不能用于财务经营身份（系统应拒绝），更不能跨pages深引。
- `web/src/components/layout/OrganizationWorkspace.tsx:24` 未完整身份不挂业务，`:27` 按用户+组织重建QueryProvider，`:44` 同步切换请求域并清缓存。`AppLayout.tsx:233` 整体工作区也按同key重建；本任务应沿用，不增加第二套切换状态机。
- 查询和写入公司来源应直接是auth/me当前公司，脱离Form可编辑字段和queryState恢复字段。其他组织记录/固定fee选择若不等于当前公司应显式拒绝，禁止用旧record公司覆盖workspace公司。

### 6. 最薄实施与验证建议

1. 前端先去selector及候选加载，列表和弹窗统一当前公司ID；保留公司展示、现有写权限和提交guard。完整身份之外还复用canOperateBusiness防系统/部门fixture直接挂载时发业务请求。
2. 同一组修改删除公司候选RPC链，精简purpose枚举并reserved，保留结算单位候选的4/9和权限差异。保留其他业务请求的organization_id；不改财务实体/Schema/锁/版本。
3. 只由一个owner生成 `make -C server api`、必要的项目生成目标及 `pnpm run generate:web-client`（含枚举），确认access_rules/OpenAPI/typings全同步；不改manifest权限键，因没有新增/删除业务权限，不需凭空删权限。
4. 后端定向覆盖：公司候选HTTP/gRPC与描述符缺席、保留结算单位RPC与purpose4/9编号、退役名称/编号reserved、0/退役用途拒绝；现有scope/创建候选服务测试、`biz/business_workspace_test.go` 与 `auth_system_company_test.go`。运行受影响包test/vet及buf lint。纯RPC/代码删除无Schema无需真实库迁移。
5. 前端定向覆盖：未就绪/非company不请求不提交；旧恢复org和Form残留不能改公司；当前公司候选/预览/提交与列表一致；切公司隔离；现金和核销结算单位候选仍发正确4/9，无公司候选网络调用；保留已采纳快照与行归属展示。再tsc、修改文件Biome，最终跨层check:fast及独立review。

## Related specs

- `.trellis/spec/server/backend/operating-company-commission-attribution.md`
- `.trellis/spec/server/backend/auth-session-org-switch.md`
- `.trellis/spec/server/backend/role-workspace-ownership.md`
- `.trellis/spec/web/frontend/organization-switch.md`
- `.trellis/spec/web/frontend/finance-ledger-snapshots.md`
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`
- 根 `AGENTS.md` 与 `.trellis/workflow.md` 的规划/生成约束。

## External references

无第三方API用法变更，本研究仅依据仓库已锁定契约及实现，未使用外部资料。

## Caveats / Not Found

本研究仅检查源码、测试及规范，未运行测试、未操作数据库、未修改产品文件。行号基于研究时工作树，后续实施可能移动。没有发现需要新增后端安全框架或数据库结构的真实缺口；公司候选接口退役会带来必要的Proto/OpenAPI生成差异，应作为当前任务明确范围一并评审。
