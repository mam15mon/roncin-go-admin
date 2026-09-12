# 钉钉注册审批与通知路由 Technical Design

## 1. Architecture & Design Principles

1. **钉钉是身份真相源**：账号身份字段（unionId/userId/姓名/头像）只来自扫码返回；
  邀请预存仅承载管理员决策（手机号/组织/角色），不复制身份。
2. **双通道统一出口**：邀请（自动）与认领（审批）最终都走同一"激活原语"——
  enable credential + membership + 初始角色 + 通知本人，差别只在触发方式。
3. **通知即后台任务**：复用既有 BackgroundTask + 钉钉工作通知通道，注册/审批事件
  即时入队，不做轮询。

## 2. Core Modules & Data Models

### 2.1 Ent Schema

`dingtalk_invitations`（新实体）：

```text
id, organization_id (FK), role_id (FK, 初始角色),
mobile (加密或明文 + 展示脱敏，见 2.5), display_name (备注，可空),
invited_by (FK user), status: PENDING | CONSUMED | EXPIRED | REVOKED,
consumed_by (FK credential/account, 可空), consumed_at,
expires_at, created_at/updated_at
部分唯一索引: (organization_id, mobile) WHERE status = 'PENDING'  -- 活跃邀请唯一
```

`dingtalk_credentials`（既有）增加：`requested_organization_id`（可空 FK，通道 B 自选
目标公司，存量行为 NULL = 总部兜底）。

迁移手写（时间戳命名），含脱敏与索引。

### 2.2 biz 层

- `CreateDingTalkInvitation`（管理端）：校验调用者对目标组织持用户管理权限（组织范围
  判定复用既有 scope 谓词）、初始角色属于目标组织；手机号规范化（+86 前缀处理）。
- `LoginDingTalk` 改造（`biz/auth.go`）：换出企业 userId 后：
  1. 既有 `FindDingTalkCredential` 命中 → 原逻辑不变；
  2. 未注册 → 调 `matchInvitationByUser(ctx, userID, corpID)`：企业 token 按手机号
     反查 userId（getbymobile，逐条活跃邀请比对；或先按 userId 取通讯录 mobile 再等值
     查——实施时按 API 权限成本择一）；
  3. 命中 → 事务内消费邀请 + 激活原语（enable + membership + 角色 + 通知本人/邀请人）
     → 直接返回 Authenticated 会话；
  4. 未命中 → RegistrationRequired，registration token 携带自选组织要求（通道 B）。
- `ConfirmDingTalkRegistration` 改造：请求带目标组织 ID，落 `requested_organization_id`，
  入队审批通知。
- `ApproveDingTalkRegistration` / `RejectDingTalkRegistration`（新）：一站式审批原语，
  幂等（凭 credential 状态机 PENDING→active）；拒绝写原因并通知本人。

### 2.3 审批通知路由

- 新增后台任务类型 `dingtalk.registration.pending`：收件人 = 目标组织（无目标 = 总部）
  下持用户管理权限且启用中的用户集合（复用权限清单查询，Limit 上限保护 + 取最新
  活跃 N 人策略）；内容含注册人姓名/头像/目标组织/时间与审批入口路径。
- 复用 `NotificationUsecase` 渲染与重试；审批完成事件（已批准通知本人模板已有）。

### 2.4 API（api/auth/v1 + admin 域）

- `POST /api/v1/auth/dingtalk/invitations`（创建邀请，目标组织的管理权限 +
  DATA_SCOPE 校验目标组织）、`GET .../invitations`（列表，组织范围）、
  `DELETE .../invitations/{id}`（撤销）；
- `POST /api/v1/auth/dingtalk/registrations/{id}:approve|reject`（审批，目标组织管理
  权限）+ `GET .../registrations?status=PENDING`（审批队列）；
- 注册确认请求带 `organization_id`（可选）。
- 权限码：新增 `system.admin.dingtalk_invitation.manage`（或并入既有用户管理权限组，
  实施时按 manifest 分组惯例定），manifest + permission-keys 重生成。

### 2.5 手机号处理

- 存储明文（匹配需要等值查询；数据库在本系统信任边界内）+ 展示层脱敏
  `138****1234`；日志与审计只记脱敏形式。
- 规范化：去空格、+86/0086 前缀归一，存规范化结果。

## 3. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 扫码 userId 与活跃邀请手机号反查命中 | 自动激活直达会话 |
| 邀请存在但已过期/撤销 | 按无邀请处理，落通道 B |
| 通道 B 未选组织 | 默认总部兜底，通知总部管理员 |
| 非目标组织管理员调审批/邀请接口 | 403 |
| 同手机号重复活跃邀请 | 409（唯一索引兜底） |
| 邀请已被消费 | 状态 CONSUMED，再扫走通道 B |

## 4. Tests Required

- biz：邀请生命周期（建/过期/撤销/消费）、扫码自动激活匹配链（mock 反查）、通道 B
  自选组织落库、审批原语幂等、通知入队断言、权限路由（目标组织外 403）；
- data/集成（隔离 schema）：真实邀请→扫码→自动激活全链路、审批同意/拒绝、
  唯一约束冲突、脱敏展示；
- service/web：邀请管理页与审批队列页基础用例；登录选择组织流程；
- 回归：既有钉钉登录/注册/会话测试全绿。

## 5. Wrong vs Correct

### Wrong

```text
# 预存整套身份（姓名/部门/头像复制进邀请表），扫码后两边数据打架
# 或：注册后群发通知给所有管理员，总部被分公司加人刷屏
```

### Correct

```text
# 预存仅手机号+组织+角色；匹配键 = 企业 token 手机号反查 userId
# 通知只路由目标组织的管理员；身份字段永远以钉钉扫码返回为准
```
