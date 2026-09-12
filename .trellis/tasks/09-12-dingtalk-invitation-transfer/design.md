# 钉钉入职专属码、向上追溯与一键转派 Technical Design

## 1. 架构与核心原则 (Architecture & Principles)

1. **服务端凭证驱动，客户端零信任**：
   - 专属入职链接采用 128-bit（32 字符 Hex）密码学安全伪随机 Token（`/login?invite=<token>`）；
   - 服务端根据数据库中合法生效的 `Invitation` 记录解析目标组织，注册请求不再信任前端提交的任意 `organization_id`，防止越权伪造。
2. **两类邀请模型显式解耦 (kind 演进)**：
   - **定向邀请 (`kind = TARGETED`)**：单人一次性，手机号必填，扫码匹配后秒级激活，状态迁至 `CONSUMED`；
   - **通用邀请 (`kind = GENERIC`)**：多人多次扫码，手机号为空，状态永不 `CONSUMED`，新员工扫码落库为 `PENDING` 申请；手机号匹配链显式过滤 `kind = TARGETED`，绝不干扰通用码。
3. **首位祖先追溯算法 (First Ancestor Escalation)**：
   - 目标分公司持权审批人候选为 0 时，沿 `parent_id` 链向上查找**第一个有候选人的祖先节点**；若中途均无候选人则落根组织（总部）；
   - 通知后台任务键沿用 `(注册人, 组织, 收件人)` 三元组确定性主键，杜绝后台任务合并坍缩；
   - 属于通知兜底，非权限变更：总部 `DATA_SCOPE_ALL` 管理员依既有权限模型天然具备跨组织审批权限。
4. **一键转派（Transfer）原子切换**：
   - 仅对通道 B 的 `PENDING` 申请生效；在行级悲观锁事务中原子更新 `users.dingtalk_requested_organization_id`，并向新组织入队审批通知（新组织若无管理员同样触发向上追溯）。

---

## 2. 数据模型与 Schema 演进 (Data Models)

### 2.1 Ent Schema: `dingtalk_invitations`

```text
字段:
- id: UUID (PK)
- token: string(64), UNIQUE NOT NULL       -- crypto/rand 16 字节转 hex（128-bit 不可枚举）
- kind: enum("TARGETED", "GENERIC"), DEFAULT("TARGETED") -- 区分定向单人 vs 通用扩招
- organization_id: UUID (FK -> organizations.id)
- role_id: UUID (FK -> roles.id, 可空)     -- 预设初始角色（可选）
- mobile: string(32), 可空                  -- 定向手机号（TARGETED 必填，GENERIC 为 NULL）
- display_name: string(100), 默认 ""       -- 备注名
- invited_by: UUID (FK -> users.id)
- status: PENDING | CONSUMED | EXPIRED | REVOKED
- consumed_by: UUID (FK -> users.id, 可空)  -- 仅 TARGETED 消费时填充
- consumed_at: timestamp, 可空
- expires_at: timestamp NOT NULL
- created_at / updated_at: timestamp

索引:
- UNIQUE (token)
- UNIQUE (organization_id, mobile) WHERE status = 'PENDING' AND kind = 'TARGETED' AND mobile IS NOT NULL AND mobile != ''
- INDEX (organization_id, status)
- INDEX (expires_at)
```

### 2.2 用户实体 `users`

- `dingtalk_requested_organization_id`: UUID (FK -> organizations.id, 可空)
  - 通用码扫码时记录申请分公司；一键转派时原子更新为此字段。

---

## 3. 核心算法与状态机 (State Machine & Algorithms)

### 3.1 邀请状态机与消费规则

```text
[创建邀请]
   |
   +---> kind = TARGETED (手机号必填)
   |        |
   |        +---> 扫码命中通讯录手机号 ---> 状态迁为 CONSUMED ---> 账号秒级激活
   |        +---> 到期未扫 ---> 状态为 EXPIRED (创建时收割或惰性判定)
   |        +---> 管理员撤销 ---> 状态为 REVOKED
   |
   +---> kind = GENERIC (无手机号，多人扫码)
            |
            +---> 员工 A 扫码申请 ---> 生成 PENDING 注册 ---> 邀请状态维持 PENDING
            +---> 员工 B 扫码申请 ---> 生成 PENDING 注册 ---> 邀请状态维持 PENDING
            +---> 到期 ---> 状态为 EXPIRED
            +---> 管理员撤销 ---> 状态为 REVOKED
            (注：GENERIC 行永不变为 CONSUMED，且不参与手机号等值匹配链)
```

### 3.2 首位祖先追溯算法 (First Ancestor Escalation)

```go
func (uc *DingTalkRegistrationUsecase) ListApproverRecipientsWithEscalation(ctx context.Context, targetOrgID uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error) {
    currID := targetOrgID
    isEscalated := false
    for {
        recipients, err := uc.repo.ListApproverRecipients(ctx, currID)
        if err != nil {
            return nil, uuid.Nil, false, err
        }
        if len(recipients) > 0 {
            return recipients, currID, isEscalated, nil
        }
        parentID, hasParent, err := uc.repo.GetParentOrganizationID(ctx, currID)
        if err != nil {
            return nil, uuid.Nil, false, err
        }
        if !hasParent {
            // 已达根组织（总部），无更高祖先
            return nil, currID, isEscalated, nil
        }
        currID = parentID
        isEscalated = true
    }
}
```

- **通知任务防坍缩键**：
  确定性 Key 沿用昨日规范：`fmt.Sprintf("dingtalk.reg:%s:%s:%s", registrantID, routingOrgID, approverUserID)`；
  若 `isEscalated == true`，模板渲染注入 `[由总部/上级代管]` 提示标签。

### 3.3 一键转派流程 (Transfer)

```go
func (uc *DingTalkRegistrationUsecase) TransferRegistration(ctx context.Context, principal *Principal, userID, targetOrgID uuid.UUID, reason string) error
```

1. **权限复核**：调用者必须对申请当前路由的组织持有 `system.user.dingtalk_invitation.manage` 权限；
2. **入参防呆**：目标分公司 `targetOrgID` 必须存在、启用，且不能等于当前已申请组织；
3. **单事务悲观锁流转**：
   - `tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx)`；
   - 验证用户必须仍处于 `enabled == false` 待审批状态（防止并发冲突）；
   - 更新 `user.SetDingtalkRequestedOrganizationID(targetOrgID)`；
   - 重新计算新目标分公司的审批人（触发向上追溯）；
   - 入队新的审批通知任务（任务说明注明转派人与转派理由）；
   - 写入审计日志：`admin.dingtalk.registration.transfer`。

### 3.4 登录与 OAuth 凭证穿透 (Token Passthrough)

```text
1. 用户访问 /login?invite=<token>
2. 前端解析 query 中的 invite 参数，存储于 sessionStorage 并注入钉钉 OAuth authorize URL 的 state 参数（或临时 Cookie）；
3. 钉钉扫码回调后，回调接口将 invite token 传回后端 RegisterDingTalkUserRequest；
4. 后端根据 token 查出目标组织 ID，忽略任何前端可能伪造的组织字段；
5. 若 token 失效（过期/撤销/不存在），后端返回明确错误码 ErrDingTalkInvitationInvalid；前端清空会话并提示。
```

---

## 4. 契约定义 (Protobuf)

统一挂载在既有 `server/api/admin/v1/admin.proto`（权限码 `system.user.dingtalk_invitation.manage`）：

```protobuf
enum DingTalkInvitationKind {
  DING_TALK_INVITATION_KIND_UNSPECIFIED = 0;
  DING_TALK_INVITATION_KIND_TARGETED = 1;
  DING_TALK_INVITATION_KIND_GENERIC = 2;
}

message CreateDingTalkInvitationRequest {
  string organization_id = 1 [(google.api.field_behavior) = REQUIRED];
  DingTalkInvitationKind kind = 2 [(google.api.field_behavior) = REQUIRED];
  optional string mobile = 3;        // TARGETED 必填，GENERIC 留空
  optional string role_id = 4;       // 预设初始角色（可选）
  optional string display_name = 5;
  int32 expires_in_days = 6;        // 默认 7 天 (1~30)
}

message CreateDingTalkInvitationResponse {
  DingTalkInvitation data = 1;
  string invitation_url = 2;         // 完整链接: /login?invite=<token>
}

message TransferDingTalkRegistrationRequest {
  string user_id = 1 [(google.api.field_behavior) = REQUIRED];
  string target_organization_id = 2 [(google.api.field_behavior) = REQUIRED];
  string reason = 3 [(google.api.field_behavior) = REQUIRED];
}
```

---

## 5. 验证与错误矩阵 (Validation & Error Matrix)

| 场景 | 触发条件 | 预期行为 |
| :--- | :--- | :--- |
| 创建 GENERIC 邀请 | `kind = GENERIC, mobile = ""` | 成功生成 128-bit Token，状态 PENDING |
| 创建 TARGETED 邀请 | `kind = TARGETED, mobile = "13800000000"` | 成功生成 Token，部分唯一索引校验 |
| TARGETED 缺少手机号 | `kind = TARGETED, mobile = ""` | 400 `ErrAdminInvalidArgument` |
| 扫已失效 Token | `expires_at <= now` 或 `status = REVOKED` | 400 `ErrDingTalkInvitationExpired` / `Revoked` |
| GENERIC 多人扫码 | 多个新员工先后扫同一 GENERIC 码 | 多人均生成 PENDING 申请，邀请状态仍为 PENDING |
| 目标组织无管理员 | 分公司管理员数为 0 | 沿 parent 链命中首个有候选人的祖先（或总部） |
| 转派到相同组织 | `targetOrgID == currentOrgID` | 400 `ErrAdminInvalidArgument` |
| 越权转派 | 审批人无原组织审批权限 | 403 `ErrPermissionDenied` |
| 并发审批与转派 | 一方先提交成功 | 另一方命中行锁后比对状态返回 409 `ErrDingTalkRegistrationProcessed` |

---

## 6. 测试与质量门禁计划 (Tests)

1. **biz 单元测试**：
   - 向上追溯算法：单级分公司、多级树形办事处、全链为空兜底至根组织断言；
   - 邀请双模式状态机断言：GENERIC 多次扫码不变 CONSUMED，TARGETED 扫码秒变 CONSUMED；
   - 转派状态机断言：越权拦截、同组织拦截、原子变更与新通知入队断言。
2. **data 集成测试**：
   - PostgreSQL 部分唯一索引：验证 TARGETED 唯一生效，GENERIC 允许并发无手机号行；
   - 转派悲观锁与通知任务确定性三元组键生成（防坍缩）。
3. **接口防伪造测试**：
   - 注册接口传入伪造 `organization_id`，断言服务端根据 Token 强绑定组织，忽略传入值。
