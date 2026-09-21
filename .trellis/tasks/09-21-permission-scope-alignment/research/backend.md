# Research: 后端权限范围与前端能力投影

- Query: 核对 self 范围是否具有本人限制，给出权限/范围投影修复边界。
- Scope: internal
- Date: 2026-09-21

## Findings

### 已查阅规范

- `.trellis/workflow.md`：规划、实施、检查、提交分阶段。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：领域和权限入口。
- `.trellis/spec/server/backend/index.md`：跨组织访问必须绑定具体权限及其角色范围来源。
- `.trellis/spec/server/backend/operating-company-commission-attribution.md:18`：总部仅治理与授权读取；经营写必须当前启用公司，管理员同样受限；跨组织读取定位不能变更真实工作台身份。
- `auth-session-org-switch.md`、`role-workspace-ownership.md` 的相关检索没有发现针对 self 本人关系的已批准规则。

### 文件与代码证据

| 文件 | 说明 |
| --- | --- |
| `server/internal/biz/auth.go:305` | HasPermissionInScope 在同一个 RoleGrant 上检查权限与最小范围，正确防止跨角色拼接；bootstrap 特例在持有权限后才适用。 |
| `server/internal/biz/auth.go:337` | ResolvePermissionOrganizationScope 只合并持目标权限的角色，但未检查组织级最小范围。 |
| `server/internal/biz/auth.go:410` | baseOrganizationIDs 将 self 与 organization 同样返回当前组织 ID。 |
| `server/internal/biz/auth_principal_test.go:96` | 明确测试 self 返回当前组织，说明扩张是已存在行为而非偶发空值。 |
| `server/internal/server/auth.go:221` | hasPermission 对核心财务、往来单位和订单走特定组织范围分支，普通管理权限才使用 HasPermissionInScope；特定分支没有消费 proto 声明的 organization 最小范围。 |
| `server/internal/data/order_query.go:39` | FindAuthorized 只按 ID 和 OrderOrganizationScope 查询；List 在 :52 同样消费组织范围。 |
| `server/internal/data/order_query.go:167` | orderOrganizationScopePredicate 仅包含业务类型和组织 ID，无 UserID/本人属性。 |
| `server/internal/data/partner.go:47` | FindAuthorized 只按 ID + 组织；List :59 同样按组织，无本人约束。 |
| `server/internal/service/organization_scope.go:12` | 财务等服务从权限生成允许组织列表。 |
| `server/internal/data/finance_bill.go:38` | 账单列表以组织 ID 集合过滤；组织谓词 :341、费用候选 :360 也是组织维度。 |
| `server/internal/data/finance_bill_batch.go:53` | 批次详情与 :62 锁定写操作只在授权组织内定位批次。 |
| `server/api/partner/v1/partner.proto:13`、`server/api/finance/v1/settlement.proto:17` | 常规读取/写入接口声明最小范围 ORGANIZATION。 |
| `web/src/pages/admin/components/roles/roleConstants.ts:27` | self 文案声称仅本人创建/指派；后端常规业务资源未实现该语义。 |
| `server/api/workbench/v1/workbench.proto:10` | 我的工作台是独立本人模型，使用 AUTHENTICATED 门禁，用户与公司只取会话。 |
| `server/internal/service/workbench.go:28` | WorkbenchScope 携带 UserID 等本人事实，和常规组织列表授权是不同链路。 |
| `server/api/auth/v1/auth.proto:257` | CurrentUser 分别返回 permissions 与 role_scopes，无法重建哪个角色授予哪个权限。 |
| `server/internal/service/auth.go:245` | principalToAPI 把 PermissionKeys、RoleScopes 分别投影，已丢失来源关系。 |
| `web/src/access.ts:140` | 当前 granted 和 hasScope 分别运算，canOrder 及普通管理动作存在跨角色拼接。 |

### 对 self 的实际判断

1. 普通组织管理接口通过 HasPermissionInScope 拒绝仅 self grant。
2. 订单列表、详情、按 ID 写操作的通用鉴权没有本人约束，self 被当作整个当前组织。下游状态/锁仍可能拒绝操作，但并非本人限制。
3. 往来单位列表/详情与写入口存在同类扩张。
4. 核心财务台账、账单、资金、核销、对冲等中间件特定分支存在同类扩张；财务配置中仍走普通最小范围门禁的接口不能一概而论。
5. 我的工作台存在真实本人事实过滤，与 self 范围枚举没有直接等价关系，不能因修常规资源授权而禁掉本人工作台。

### 建议的最小边界（待主会话纳入设计，不代表用户已批准产品语义）

- 按常规接口已声明的 ORGANIZATION 最小范围收紧：self grant 不参与组织资源授权；若同一权限另有 organization/tree/all grant，正常按那个 grant 的范围工作。
- 统一领域组织范围解析与接口门禁语义，不能仅在前端遮按钮或仅在一条中间件路径处理。
- 不在本次临时建设订单/伙伴/账单的通用“本人创建或参与”模型：账单聚合多订单、伙伴多指派、派生费用及附件会产生跨实体语义问题；需要明确业务需求才能实现。
- self 文案明确只适用已实现的本人用例，不承诺常规组织资源自动按本人过滤；不擅自删除角色 self 值或重写数据库授权。
- 当前组织业务写、总部治理与授权读取、bootstrap 行为、工作台本人流程均要定向回归。

### auth/me 的可选投影

A. 返回每个权限的有效最小范围能力（如 key + scope，scope 已由后端按匹配 grant 计算；bootstrap 规则在后端应用）。前端能按同一个条目判断最小范围，roleScopes 仅供显示。该方案保留前端必要范围概念但不会交叉拼接。

B. 后端按接口契约与权限生成可调用能力列表，前端只 has(key)。前提是同一个 permission 在各个接口的最低范围一致，必须先枚举 proto 做一致性检查；否则同一权限可能有多种门禁，简单 key 列表无法表达。不要人工复制一份最低范围字典到前端，也不要令 service 依赖 server 层。

C. 返回每个 RoleGrant 的权限列表与范围，前端复制 HasPermissionInScope。能够修复拼接，但会复制 bootstrap、当前组织工作台等后端细节；优先 A/B。

资源级组织归属与状态动作仍应消费响应 AllowedActions 或等价能力；auth/me 只能决定入口，不保证任意 ID 都可修改。

### 最小验证矩阵

- 同一权限只有 self；另一个无关角色 organization/all：常规组织资源仍拒绝、按钮隐藏。
- 同一权限分别来自 self 与 organization：允许组织内，禁止越界。
- organization_tree/all 授权读取保持；经营写始终当前公司。
- 总部与公司身份切换后投影同步；bootstrap 持权限才具备豁免。
- 订单、伙伴、财务的列表/详情/写入口分别覆盖 self 拒绝。
- 我的工作台本人接口与员工提成申请不因组织资源修复误关。
- proto → Go/OpenAPI → Web 客户端同提交生成；测试数据形状同步新契约，不保留旧数据回退。

## External references

未依赖外部框架或版本资料；全部结论来自仓库静态代码。

## Caveats / Not Found

- 未查询开发数据库，无法判断现有角色实际组合与受影响账号。
- 本报告未运行测试；引用的是现有测试代码，不应宣称验证通过。
- 未穷举所有财务派生资源和每个 handler 的后置校验；证据足以确认公共范围模型缺乏本人过滤，但不能把全部 API 一概描述为同一行为。
- 未找到批准“self 对所有领域等价当前组织”的设计依据；现有单测说明当前行为，不能自动视为正确业务契约。

### 若本期选择“停用 self”而不实现本人访问

这是等待用户确认的产品决策。最小且不扩大权限的方案应同时覆盖写入、配置与运行时：

- 角色 Create/Update 接口不再接受 `self` 作为新配置值；UI 角色范围选择器移除 self（或仅历史只读展示），不要把历史 self 自动改成 organization/all，也不要自动迁移或升权。
- 运行时组织资源授权解析应把 self 视为“不满足 ORGANIZATION”，不能继续由 `baseOrganizationIDs` 转成当前组织范围；历史角色仍可在角色详情展示原值，并在保存其它字段时明确要求管理员调整，避免静默升权。
- 所有声明 `scope=ORGANIZATION` 的订单、往来单位、财务等组织资源入口统一拒绝 self-only grant；同一权限若另有 organization/tree/all grant，仍按后者允许。
- 已明确的本人工作台使用 UserID/协作事实，不应依赖 RoleGrant self；停用角色 self 不应关闭 `/workbench/my-*`。
- 需要为 Create/Update、ResolvePermissionOrganizationScope、权限中间件和 UI 选择器补定向回归；历史数据库值先保留，等待显式管理员调整。

该方案属于“停用 self”而非实现“本人创建/指派”语义，必须用户确认；不可把 roleConstants 的现有文案当作已批准完整业务契约。

### auth/me 能力投影的薄层可行性

返回每个**有效权限的最小/有效 scope**（例如 `permission_capabilities: [{key, scope}]`）可以薄层解决前端把 `permissions` 与无关 `roleScopes` 交叉拼接的问题，但必须由后端按同一 `RoleGrant` 过滤、工作区规则和 scope 合并生成；不要返回所有 RoleGrant 的权限明细让前端重演授权。

建议语义：同一 permission 的 scope 取匹配 grant 中最高有效级别（self < organization < organization_tree < all），并先应用 `permissionAvailableInWorkspace`；经营权限在总部工作台直接不出现在有效能力中。前端按能力条目判断 `capability.scope >= 页面/动作要求`，`roleScopes` 仅作角色信息展示。这样普通权限的 organization/all 阈值一致，不会再出现 `has(permission) && hasScope(anyRole)` 的拼接。

局限必须写入设计：

- 若同一 permission 在不同 proto 方法需要不同最低 scope，单一能力 scope 只表达该 permission 的最高可用范围；调用方仍要按接口约定使用阈值。当前多数常规 proto 以 ORGANIZATION 声明，订单按业务操作另有规则，需先扫描契约。
- 财务与经营工作区过滤必须在后端生成能力时完成；只把原始 permissions 搬成 `{key,scope}` 仍会暴露总部经营写权限，不能接受。
- 资源 ID 是否可访问、订单状态动作、锁定/本人协作属于请求级授权，auth/me 不能替代接口后端校验。
- 生成该字段需修改 auth.proto，再生成 Go/OpenAPI/Web 客户端；旧 `permissions` 是否保留需在设计中决定，避免两套前端真相长期并存。
