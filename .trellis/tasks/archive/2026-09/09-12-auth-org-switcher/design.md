# 登录组织选择与应用内分公司切换器 Technical Design

## 1. Architecture & Design Principles

1. **会话即组织**：沿用"会话绑定组织"模型，不引入逐请求组织头；切换组织 = 换发会话，
   权限集随新会话重算——身份层收口，业务层零改动。
2. **成员资格是唯一入场券**：登录可选组织列表与切换目标都取自同一查询（本人启用中
   membership × 组织启用中），单一谓词两处复用，杜绝"列表看得见、切换进不去"的漂移。
3. **轮转不吊销全家**：切换只失效当前 token（按 token hash 删除该会话），其他设备
   会话不受影响。

## 2. Core Modules & Data Models

### 2.1 契约（api/auth/v1/auth.proto）

- `LoginResponse` 增加 `repeated OrganizationChoice organization_choices`：
  `{organization_id, organization_name, organization_code, is_default}`（按组织名称稳定
  排序，默认组织置首/标记）；
- 新增 RPC `SwitchOrganization`：
  `POST /api/v1/auth/switch-organization`，请求 `{organization_id}`（REQUIRED），响应
  与 `LoginResponse` 同构（新 token、principal、organization_choices）；鉴权模式与
  登录后续接口一致（需有效会话）。
- `make -C server api` 重生成，`generate:web-client` 同步。

### 2.2 biz（internal/biz/auth.go）

- 抽公共查询 `listEnabledMembershipOrganizations(ctx, userID)`：membership
  `user_id = ? AND enabled` JOIN 组织 `enabled`，返回组织快照列表；登录响应与切换
  入口共用；
- 登录：会话组织 = 请求显式指定（Login 请求体新增可选 `organization_id`，校验在候选
  集内）或 primary；
- `SwitchOrganization(ctx, actorSession, targetOrgID)`：
  1. 校验目标在候选集内（否则 `ErrAuthOrganizationForbidden`，403 语义稳定错误）；
  2. `ResolvePrincipal(userID, targetOrgID)` 重算权限主体；
  3. 事务内：新建会话（新 token、目标组织）+ 按**当前 token hash** 删除旧会话 +
     审计（`auth.organization.switch`，含来源/目标组织）；
  4. 返回与登录同构响应。

### 2.3 data（internal/data/auth.go 或会话仓储）

- 会话创建/删除按现有实现；新增"按 token hash 删除单会话"（若已有则复用）；
- 成员资格候选查询带 `ForShare`（若在事务内）。

### 2.4 service（internal/service/auth.go）

- Login handler 组装 `organization_choices`；请求体透传可选 organization_id；
- SwitchOrganization handler：会话鉴权中间件后可用，DTO 映射与 Login 复用。

### 2.5 前端（web）

- **登录页**：登录成功后 `organization_choices.length > 1` → 渲染组织选择视图（组织
  卡片：名称/编码/默认标记），点选后携 `organizationId` 重新完成登录（或调用切换），
  进入应用；`length == 1` 直接进入（现状路径）；
- **顶栏切换器**：当前组织名 + 下拉（候选列表，排除当前），选择后调
  `switchOrganization` → 用新 token 替换凭据 → 重置服务端状态缓存（React Query
  clear）→ 重新拉取 `auth/me` 与菜单 → 回到工作台首页；
- 权限集/菜单/路由全部由 `auth/me` 派生，无需页面级改动。

## 3. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 登录指定 organization_id 不在候选集 | 400 参数错误（不静默回退 primary） |
| 切换目标非本人成员资格 / 成员资格停用 / 组织停用 | 403 `AUTH_ORGANIZATION_FORBIDDEN` |
| 切换成功 | 旧 token 失效、新 token 生效、审计落库 |
| 无候选组织（异常态：全部停用） | 登录拒绝（沿用现有凭据校验失败语义） |

## 4. Tests Required

- biz：候选列表查询（启停组合）、切换校验（非成员/停用拒绝）、轮转后旧会话失效、
  审计事件；
- service：Login 响应组织列表映射、指定组织登录 400、SwitchOrganization 403/成功
  同构响应；
- web：登录选择页渲染与选择流程（多组织）/直进（单组织）、顶栏切换触发 token 替换
  与状态重置；
- 回归：既有登录、会话过期、权限解析、`auth/me` 测试全绿。

## 5. Wrong vs Correct

### Wrong

```text
# 逐请求带组织头，信任前端选择
GET /api/v1/...  Header: X-Organization-Id: <任意组织>
```

### Correct

```text
# 组织只能通过会话变更进入，且会话变更必须过成员资格校验
POST /api/v1/auth/switch-organization {organization_id}
→ 校验候选集 → 换发 token → 权限集随新会话重算
```
