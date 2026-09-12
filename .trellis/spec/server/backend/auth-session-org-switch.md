# 登录组织选择与会话切换契约

## 1. Scope / Trigger

适用于登录组织选择、应用内组织切换（`SwitchOrganization`）与会话轮转。修改 auth 契约、
会话仓储或 membership 可用性语义时必须遵循。

## 2. Signatures

```text
POST /api/v1/auth/login                     body 可选 organization_id（显式选择进组织）
POST /api/v1/auth/switch-organization       body organization_id（REQUIRED，需有效会话）
LoginResponse / SwitchOrganizationResponse: organization_choices[]（与登录同构）
OrganizationChoice: organization_id/name/code/is_default（默认置首标记，名称+ID 确定性排序）
```

```go
// internal/data/auth.go
ListEnabledMembershipOrganizations(ctx, userID) // 候选谓词唯一实现（三处消费）
RotateSession(ctx, userID, currentTokenHash, targetOrgID, ...)  // 单事务轮转
```

## 3. Contracts

- **成员资格是唯一入场券**：候选 = membership enabled × 组织 enabled；登录显式组织校验、
  登录响应组装、切换校验三处共用同一谓词，禁止旁路查询。
- **切换 = 会话轮转**：单事务四步——成员资格 ForShare 复核（防 TOCTOU）→ 旧会话行
  ForUpdate（撤销/过期/归属校验）→ 新建目标组织会话（沿用 UA/IP）→ 旧令牌 SetRevokedAt
  + 审计。任一步失败整体回滚；**不吊销其他设备会话**。
- **并发语义**：同令牌并发切换由 ForUpdate 串行化，先提交者胜出，后到者 401（无双有效
  令牌窗口）；新令牌哈希唯一约束兜底。
- **权限随组织重算**：切换后 `ResolvePrincipal(userID, targetOrgID)` 重算权限集；
  principal/token 一律取自会话（`RequirePrincipal`），请求体只携带目标组织。
- **primary 语义不变**：不显式选择时登录进 primary；每次登录默认回 primary，选择只影响
  当次会话（不记忆）。
- 钉钉/企业微信登录响应不携带 organization_choices（设计范围），前端以 principal
  organizations 兜底派生候选，经切换进入非默认组织。

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
