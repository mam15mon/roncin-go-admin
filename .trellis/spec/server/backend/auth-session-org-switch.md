# 登录组织选择与会话切换契约

## 1. Scope / Trigger

适用于登录组织选择、应用内组织切换（`SwitchOrganization`）与会话轮转。修改 auth 契约、
会话仓储或 membership 可用性语义时必须遵循。

## 2. Signatures

```text
POST /api/v1/auth/login                     body 可选 organization_id（显式选择进组织）
POST /api/v1/auth/switch-organization       body organization_id（REQUIRED，需有效会话）
LoginResponse / SwitchOrganizationResponse: organization_choices[]（与登录同构）
OrganizationChoice: organization_id/name/code（名称+ID 确定性排序，无默认工作台配置）
```

```go
// internal/data/auth.go
ListEnabledMembershipOrganizations(ctx, userID) // 候选谓词唯一实现（三处消费）
RotateSession(ctx, userID, currentTokenHash, targetOrgID, ...)  // 单事务轮转
```

## 3. Contracts

- **管理员初始化及补齐**：`pnpm run bootstrap:admin` 区分空账号首次创建和已有唯一
  `is_bootstrap_admin` 初始管理员的补齐。首次需用户名、姓名、密码；重复执行只核对非空
  配置用户名，不改密码/姓名，也不要求首次密码。非空用户表没有或存在多个初始管理员、
  配置身份冲突均明确失败，禁止接管普通同名账号。
- **真实公司资格**：手工新公司、默认公司种子及 bootstrap 为初始管理员补建公司根成员
  和 administrator 绑定；已有其他角色保留，部门/组不重复加入。只有初始管理员自动
  授予资格，后来授予系统管理员角色的账号仍按显式成员授权。细节见
  [公司初始管理员补齐](./company-admin-provisioning.md)。
- **停用与原子性**：初始账号或系统必要资格不可用时，在任何授权写入前拒绝；公司、
  公司管理员角色或其已有成员停用则明确跳过，不恢复或创建替代。全部可处理项同事务
  写入，数据库错误整体回滚；摘要区分新增、已满足、跳过，不声称全公司成功。

- **系统与公司分离**：system与company均为独立根工作台；bootstrap标记不提供公司准入或权限范围旁路。公司经营读写固定当前公司，系统管理权限不自动带来经营数据访问。
- **成员资格是唯一入场券**：候选 = membership enabled × 组织 enabled；登录显式组织校验、
  登录响应组装、切换校验三处共用同一谓词，禁止旁路查询。
- **切换 = 会话轮转**：单事务四步——成员资格 ForShare 复核（防 TOCTOU）→ 旧会话行
  ForUpdate（撤销/过期/归属校验）→ 新建目标组织会话（沿用 UA/IP）→ 旧令牌 SetRevokedAt
  + 审计。任一步失败整体回滚；**不吊销其他设备会话**。
- **并发语义**：同令牌并发切换由 ForUpdate 串行化，先提交者胜出，后到者 401（无双有效
  令牌窗口）；新令牌哈希唯一约束兜底。
- **工作台经营边界**：系统管理工作台仅管理公共资料、公司授权及无组织待分配人员；经营读取和办理须进入启用公司。登录、auth/me 与切换响应输出相同有效权限集合，全部经营读写权限在系统管理工作台不可用；账号原始授权不因投影过滤而删除。跨组织详情查询临时定位组织必须保留 `Principal.WorkspaceOrganizationID`，不能当作实际切换。
- **权限随组织重算**：切换后 `ResolvePrincipal(userID, targetOrgID)` 重算权限集；
  principal/token 一律取自会话（`RequirePrincipal`），请求体只携带目标组织。
- **工作台选择**：成员关系没有 primary 字段，组织管理不提供默认进入配置。仅一个有效
  工作台时直接进入；多个有效工作台时，密码与钉钉登录均先展示选择页。后端可按名称+ID
  的稳定顺序建立初始会话，该内部顺序不向用户表达为默认推荐，也不记忆上次选择。
- 密码登录响应使用 organization_choices；钉钉登录以 principal.organizations 派生同构
  候选。选择当前会话所属工作台直接进入，选择其他工作台通过 SwitchOrganization 轮转；
  当前组织依据会话 organization_id 判断，禁止使用候选位置推断。
- 前端选择页的候选及 currentOrganizationId 必须同步来自本次认证响应，并放在同一份
  本地选择状态；不能从经 startTransition 延后更新的 initialState 读取当前组织，否则
  旧会话组织 ID 可能导致本次选择错误跳过轮转。
- **角色编辑目标**：成员角色只通过明确的 membership_id 编辑；用户基本资料更新不接受
  role_ids，也不从成员列表中猜测角色写入目标。创建用户时仍可设置已明确初始组织的角色。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 登录显式 organization_id 不在候选集 | 400 `AUTH_ORGANIZATION_INVALID`（不静默回退） |
| 切换目标非成员/成员资格停用/组织停用 | 403 `AUTH_ORGANIZATION_FORBIDDEN` |
| 切换成功 | 旧 token 失效、新 token 生效、审计含来源/目标组织 |
| 旧会话已撤销/过期（含并发后者） | 401 会话失效路径 |

## 5. Good / Base / Bad Cases

- Good：多组织财务登录 → 选择页 → 点成都公司 → 轮转进成都权限集，其他设备会话不受影响。
- Base：单组织用户登录直达，顶栏无切换入口。
- Bad：切换接口信任请求体中的 userID（必须取自会话）；切换后仅前端改状态不轮转令牌。

## 6. Tests Required

- biz：候选过滤/排序确定性、显式组织 400、切换 403 三态、轮转后权限集变化、审计。
- data/集成：真实轮转往返（新令牌可用/旧令牌失效/他设备不受影响）、事务回滚。
- web：多组织选择页、单组织直进、切换状态重置顺序（清缓存先于 initialState 重建）、
  脏表单守卫、请求防重。

## 7. Wrong vs Correct

### Wrong

```text
# 逐请求带组织头信任前端，或切换只改会话字段不轮转令牌
GET /api/v1/...  X-Organization-Id: <任意>
```

### Correct

```text
# 组织只能经会话轮转进入，且轮转必须过成员资格事务内复核
SwitchOrganization → RotateSession（ForShare 复核 + ForUpdate 串行 + 审计）
```
