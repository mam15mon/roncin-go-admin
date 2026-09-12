# 钉钉分公司专属邀请与入职审批 Technical Design

## 1. 架构与设计原则 (Architecture & Principles)

1. **凭证唯一定位，客户端零信任**：
   - 目标组织由服务端的有效 `Invitation` 记录唯一解析，前端注册请求只传 `invitation_token`，禁止前端直接提交 `organization_id`。
2. **钉钉是身份真相源**：
   - 用户身份字段（UnionID / 企业 UserID / 姓名 / 头像 / CorpID）只来自钉钉扫码返回；
   - 邀请记录仅存储业务决策字段（目标组织、预设角色、定向手机号、邀请人、过期时间）。
3. **两类邀请统一处理，激活原语一致**：
   - 通用邀请（审批入职）与定向邀请（免审秒入职）共享同一底层激活原语：`enable user + create target membership + assign roles + audit + notify`。
4. **精确通知与分公司隔离**：
   - 审批提醒仅推向目标分公司具有审批权限的人员；
   - 本期纠错环为「拒绝后重新扫码自选组织」；一键转派（Transfer）记为延后能力，本期不实现（实施修订）。

---

## 2. 核心模块与数据模型 (Data Models)

### 2.1 Ent Schema: `dingtalk_invitations`

```text
id: UUID (PK)
token: string(64), UNIQUE NOT NULL      -- 邀请短码/Token，用于拼装二维码/URL
organization_id: UUID (FK -> organizations.id)
role_id: UUID (FK -> roles.id, OPTIONAL) -- 预设初始角色
mobile: string(32), OPTIONAL            -- 定向手机号（模式 B 必填，模式 A 为空）
display_name: string(100), OPTIONAL     -- 备注姓名
invited_by: UUID (FK -> users.id)
status: PENDING | CONSUMED | EXPIRED | REVOKED
consumed_by: UUID (FK -> users.id, OPTIONAL)
consumed_at: timestamp, OPTIONAL
expires_at: timestamp NOT NULL
created_at / updated_at: timestamp

索引:
- UNIQUE (token)
- UNIQUE (organization_id, mobile) WHERE status = 'PENDING' AND mobile != ''  -- 活跃定向邀请唯一
- INDEX (organization_id, status)
- INDEX (expires_at)
```

### 2.2 用户与待审批记录扩展

- `users.dingtalk_requested_organization_id`: UUID (FK -> organizations.id, 可空)
  - 记录通过通用邀请扫码后申请加入的分公司；
  - 纠错环为拒绝后重新扫码重选组织（重新确认入队新路由，见 data 层实现）。

### 2.3 审批流转状态机

```text
[发起邀请] (PENDING)
    |
    +---> (模式 B: 定向扫码) ---> 手机号匹配成功 ---> 自动 CONSUMED ---> 账号激活入职
    |
    +---> (模式 A: 通用扫码) ---> 提交申请 (User: PENDING) ---> 钉钉通知分公司管理员
                                    |
                                    +---> 管理员 [同意] ---> User 激活 + Membership 建立
                                    +---> 管理员 [拒绝] ---> 停用并记录理由 + 钉钉通知本人
                                    +---> [纠错环] 被拒者重新扫码自选组织（延后：一键转派）
```

---

## 3. 业务层与接口设计 (biz & API)

### 3.1 业务用例 (`DingTalkRegistrationUsecase`)

- `CreateInvitation(principal, orgID, roleID, mobile, displayName, expiresInDays)`:
  - 校验调用者具有目标分公司的邀请权限；
  - 若为定向手机号，校验手机号合法性并规范化；
  - 生成密码学安全的随机 `token`。
- `ListInvitations(principal, options)`:
  - 按数据范围过滤本分公司的邀请记录。
- `RevokeInvitation(principal, id)`:
  - 撤销未消费的活跃邀请。
- `ListApproverRecipients(ctx, orgID)`:
  - 查找指定组织内持有用户审批权限的启用中人员；
  - 若候选人数为 0，自动沿着 `organization.parent_id` 逐级向上追溯，直至根组织（总部），实现无管理员分公司的总部代管通知路由。
- `ApproveRegistration(principal, userID, roleIDs)`:
  - 校验审批人权限：要求对该申请的目标分公司具有管理权限（支持本分公司管理员或具有跨分公司/集团管理权限的总部管理员）；
  - 事务内激活 User、建立 target membership、赋权、写审计、异步发钉钉通知。
- `RejectRegistration(principal, userID, reason)`:
  - 驳回申请，记录拒绝原因，异步通知新员工。
- `TransferRegistration(principal, userID, targetOrgID, reason)`:
  - 将待审批记录的 `requested_organization_id` 转移至目标分公司；
  - 重新向新目标分公司的管理员推送审批工作通知。

### 3.2 契约变更 (`server/api/auth/v1/auth.proto` & `admin.proto`)

- `RegisterDingTalkUserRequest`:
  - 移除 `optional string organization_id`；
  - 增加 `string invitation_token = 1;`
- 新增 `InvitationService` / 或合并至 `AuthService`：
  - `CreateDingTalkInvitation` / `ListDingTalkInvitations` / `RevokeDingTalkInvitation`
  - `ListPendingDingTalkRegistrations` / `ApproveDingTalkRegistration` / `RejectDingTalkRegistration` / `TransferDingTalkRegistration`

---

## 4. 验证与错误矩阵 (Validation & Errors)

| 场景 | 校验规则 | 预期结果 |
| :--- | :--- | :--- |
| 非本企业钉钉扫码 | `corp_id` 比对不一致 | 403 `ErrDingTalkNotCorporateMember` |
| 无效或过期 Token 扫码 | Token 不存在或 `expires_at < now` | 400 `ErrDingTalkInvitationExpired` |
| 定向邀请手机号不匹配 | 扫码者手机号 != 邀请手机号 | 403 `ErrDingTalkInvitationMobileMismatch` |
| 跨组织非法越权审批 | 审批人可写范围不包含目标分公司 | 403 `ErrPermissionDenied` |
| 重复审批/驳回 | User 已经处于激活或已处理状态 | 409 `ErrDingTalkRegistrationAlreadyProcessed` |

---

## 5. 测试与门禁要求 (Tests)

- **单元测试 (`biz`)**：
  - 邀请 Token 生成与过期验证；
  - 定向手机号自动激活匹配逻辑（mock 钉钉通讯录）；
  - 待审批、同意、拒绝状态流转与权限判定。
- **数据层与集成测试 (`data`)**：
  - PostgreSQL 事务原子性：同意审批时 User 启用与 Membership 创建必须在同一事务；
  - 唯一索引约束校验。
- **前端验证**：
  - 邀请管理列表与弹窗；
  - 扫码落地页锁定目标分公司展示；
  - 待审批弹窗支持同意（分配角色）与拒绝。
