# Research: 系统工作台替代总部的最小完整边界

- Query: 系统管理工作台替代总部所需认证、成员资格、权限、组织树、初始化、注册审批调整。
- Scope: internal
- Date: 2026-09-22

## Findings

### 现有契约与必须修改的文件

| 文件位置 | 事实及改动含义 |
| --- | --- |
| `server/internal/biz/auth.go:270,293,327,365` | HasPermission、PermissionCapabilities、HasPermissionInScope、ResolvePermissionOrganizationScope 四处共同构成权限投影。bootstrap 将范围强制升到 ALL，不能只改 API 展示。 |
| `server/internal/biz/auth.go:1180` | permissionAvailableInWorkspace 仅阻止经营写；workspacePermissionScope 仅对 partner 全读写收窄，其他经营权限只收窄写，订单/财务读仍可能跨公司。 |
| `server/internal/data/auth.go:412,467,633,725,781,890` | 工作台枚举、成员归属、候选、principal、bootstrap 合成、会话轮转六处。bootstrap 当前可进入无成员公司并合成全部权限，必须取消公司准入旁路。保留事务内 ForShare 成员复核与单会话轮转。 |
| `server/internal/data/bootstrap_admin_membership.go:43,178` | 自动给每个 bootstrap 用户建立所有公司成员和全量 administrator 角色；只删 principal 合成不够，持久化授权仍然跨公司。 |
| `server/cmd/migrate/main.go:74` | 每次迁移执行上述全公司自动补齐，必须同步移除或收窄入口，防止新边界被下一次迁移撤销。 |
| `server/internal/data/admin_organization.go:75`、`organization_seed.go:53` | 新建公司与种子同样会自动给 bootstrap 授权，必须移除自动公司加入。 |
| `server/internal/data/permission_manifest_sync.go:31,94` | 所有名为 administrator 的角色均自动补满权限；必须按角色归属工作台过滤许可集，否则公司管理员可以持续获得系统治理权限。 |
| `server/internal/biz/admin_role.go:checkPrivilegeEscalation` | administrator 名称使 profile.IsSuperAdmin=true 后直接放行。仍需独立的“目标工作台可配置权限”校验，不能把公司超级角色等同系统授权人。 |
| `server/internal/data/admin_role.go` | 角色库已按最近工作台锚定，可以复用，不需要另建系统角色平台。需去掉 actor profile 相关 bootstrap 宽权路径，并在写入时验证工作台许可范围。 |
| `server/internal/biz/admin_organization.go:48,65,114` | 现要求新组织必须有父组织，总部→公司→部门→团队；应改为系统管理节点与公司均为根，公司→部门→团队。系统节点不能经普通组织 API 创建。 |
| `server/internal/data/ent/schema/organization.go:23` | kind 为 headquarters/company/department/team；建议正式替换 headquarters→system，保留原系统节点 UUID，解除公司 parent_id。 |
| `server/api/admin/v1/admin.proto:226`、`server/api/auth/v1/auth.proto:242` | 两处 OrganizationKind 必须同步改 SYSTEM，再走 Ent/Proto/OpenAPI/前端生成。无需新增独立工作台表。 |
| `server/cmd/bootstrap-admin/main.go:114,170,217` | 初始化应创建 system 根节点和系统角色；默认公司种子成为独立根，不自动授公司身份。 |
| `server/internal/biz/organization.go:26` | 公共主数据写入当前要求 headquarters 根身份+权限；应改 system 工作台+对应明确权限。 |
| `server/internal/service/admin.go:25,41,60,325` | 组织列表/创建/更新和跨组织角色选择入口需要区分系统公司管理与公司内部部门管理；不可仅依赖一般 User/Role ALL 范围。 |
| `web/src/access.ts`、`pages/admin/components/users/userConstants.ts`、`pages/admin/components/org/*` | 工作台身份、组织类型标签、根列表渲染、新建公司 modal 的父节点必填逻辑均需同步。组织列表将是森林。 |

### 推荐最小授权契约

1. 复用 organization/membership/role/session 模型，唯一 system 根和多个 company 根；system 成员资格独立，公司加入必须真实 membership，不借系统成员资格自动进入。
2. 系统节点只投影系统管理与公共资料权限；公司节点只投影本公司经营、人员角色与业务配置权限。不能仅用 `system.*` 前缀判断，因为现有公司人员/角色管理也使用 system 前缀；需 access 层显式定义可用权限集或分类函数，供投影、授予校验、manifest 同步统一复用。
3. 经营读写固定当前公司。公司组织管理可扩展到自身部门/团队；系统公司管理拥有独立的全公司管理范围，不产生经营权限。
4. bootstrap 标记仅用于初始系统管理员身份/保护，不用于公司候选、ALL 范围、角色提权旁路。已有真实公司角色按数据决策保留，不能推断自动建立的 membership 是用户主动分配。
5. 权限授予先校验目标工作台许可，再校验操作人的授予范围。公司管理员无论角色名 administrator 还是 ALL，均不能添加 system 工作台成员或系统治理权限。

### 钉钉流程不能只改总部文案

- `biz/auth.go:180,190,935,991` 与 `data/auth.go:181,230,338`：未指定公司者创建 PENDING 禁用账号+总部收口成员，通知沿组织树向上兜底。可将收口替换 system，但待审批 membership 不得用于准入，用户 disabled 必须始终检查。
- `data/dingtalk_registration.go:413`：查唯一启用总部根，替换 system 根。公司独立后，通知按公司部门链→公司→system 显式兜底，不沿虚构集团父链。
- **高风险**：`biz/dingtalk_registration.go:251` 审批把 `RoutingOrganizationID()` 直接作为角色校验/目标；未选公司时 routing 为总部。照搬会让普通注册者通过审批加入 system。
- 最薄闭环：未指定公司的申请在 system 队列中先明确选择/转派到公司，再审批并赋予该公司角色。审批应显式拒绝将 system 作为普通注册人的入职目标。现有 Transfer 接口可复用，避免新建复杂审批流；前端禁止收口申请直接加载系统角色。若希望一次审批完成则扩展审批 DTO 为明确 targetCompanyID，不能默认 routing=target。
- `web/src/pages/admin/components/dingtalk/constants.ts` 和 RegistrationApproveModal 当前从组织树根推导兜底，森林下必须明确按 system kind 找入口，不能选择第一个 root。
- 公司管理员不得向另一家公司转派或建立成员身份，系统审批人可执行明确的公司管理操作；邀请 TARGETED 自动激活路径也需阻止目标为 system。

### 正式迁移与顺序

1. 只读盘点 headquarters 节点数量、子公司、总部业务记录、bootstrap 成员/角色分配来源。不能删除业务记录，也不能盲删全部 bootstrap 公司 membership。
2. 正式迁移保留系统节点 ID、kind 改 system、名称系统管理，解除 company→旧总部 parent；部门团队边保持。同步 schema 和生成物。
3. 先停止 bootstrap 自动补齐及权限目录全量授予，再应用角色许可归属清理。否则脚本会再补回。
4. 公司角色从许可集中排除系统专属权限；系统角色排除经营权限。保留业务快照和角色关联 ID。已有异常角色分配需明确报告。
5. 旧会话逐请求 ResolvePrincipal 应重新收窄，切换路径使用同一成员谓词；必要会话撤销必须明确迁移设计，不要保留旧全局授权缓存。

### 必需回归

- 系统管理员无公司成员资格时不能切入公司；有 A/B 资格时角色独立。
- 公司 administrator+ALL 不能读 B 公司订单/账单或授系统权限；系统主数据管理员无法读任何经营数据。
- 迁移后再次执行 permission sync、公司 seed、建公司都不恢复穿透。
- 系统根+两个公司根的角色解析、组织列表、创建公司/部门、公司隔离。
- 未选公司注册不获得系统角色；指定公司/邀请自动激活仍仅进入目标公司；审批、转派通知显式到 system 的兜底有效。
- 现有会话并发轮转、其他设备不受影响、停用成员拒绝切换保持。

### 相关规范与外部来源

已读取 `.trellis/workflow.md`、任务 prd/design/implement；领域 glossary/architecture-map；server/backend/auth-session-org-switch.md、role-workspace-ownership.md、dingtalk-registration-approval.md、organization-shared-masterdata.md。以上旧总部、穿透、跨公司转派条款需要随实现更新。
外部参考：无，本次为仓库内部契约审计，未引用第三方 API 新用法。

## Caveats / Not Found

- 未执行数据库查询或测试，没有确认存量自动 membership 的可辨别来源；主会话正在盘点。
- 任务文档读取时仍写“尚未批准实施”，用户最新消息已批准，主会话应更新状态文案。
- 本报告为方案建议，尚未修改产品代码，具体权限分类要与完整 access manifest 一次性核对。
