# 钉钉注册审批与通知路由契约

## 1. Scope / Trigger

适用于钉钉扫码注册（双通道：邀请自动激活 / 首扫认领审批）、邀请管理与审批通知路由。
修改注册流程、邀请实体或审批通知时必须遵循。

## 2. Signatures

```text
POST /api/v1/admin/dingtalk/invitations          创建邀请（mobile+目标组织+初始角色+TTL）
GET  /api/v1/admin/dingtalk/invitations          列表（手机号脱敏直显）
DELETE /api/v1/admin/dingtalk/invitations/{id}   撤销（仅 PENDING）
GET  /api/v1/admin/dingtalk/registrations        PENDING 审批队列
POST /api/v1/admin/dingtalk/registrations/{id}:approve|reject
权限码：system.user.dingtalk_invitation.manage + DATA_SCOPE_ORGANIZATION（目标组织二次校验）
```

```text
Table: ding_talk_invitations
  (organization_id, mobile) 部分唯一 WHERE status='PENDING'
  status: PENDING | CONSUMED | EXPIRED | REVOKED（过期为惰性判定+创建时收割）
users.dingtalk_requested_organization_id  -- 通道 B 自选目标组织（空=总部兜底）
```

## 3. Contracts

- **钉钉是身份真相源**：账号身份字段（unionId/userId/姓名/头像）只来自扫码返回；
  邀请仅存管理员决策（手机号/组织/角色/TTL），预存姓名仅备注。
- **通道 A 自动激活**：扫码换出企业 userId → 企业 token 点查通讯录 mobile（user/get）
  → 归一化（去空白连字符、剥 +86/0086）→ 等值匹配活跃邀请（多组织命中**先建先得**）
  → 单事务：消费邀请 + 启用账号 + membership(首个组织 primary) + 初始角色 + 双通知 + 审计。
- **降级红线**：目录查询失败/归一化失败/无匹配/消费冲突（含并发）**一律静默降级通道 B**
  （RegistrationRequired），warn 日志不含手机号与令牌；钉钉 API 故障永不阻断扫码登录。
- **通道 B 认领**：注册确认可选目标组织（>1 时下发候选），未选走总部兜底；审批通知
  **只路由目标组织**内启用成员资格 × 启用角色真实持权限者（活跃截断 10），确定性任务
  键为 `(注册人, 组织, 收件人)` 三元组——**每位收件人独立任务+明细**，禁止共享键
  （1 任务=1 明细=1 收件人模型下共享键会静默坍缩为单人）。
- **审批一站式**：approve = 启用+membership+角色+通知本人+吊销历史会话；reject = 停用
  成员资格出队+原因通知本人；幂等（已处理 409）；跨组织审批 403；纠错环为拒绝后重新
  扫码自选（一键转派延后）。
- **手机号安全**：完整号码仅匹配用；DTO 出口即脱敏（138****8000）；日志与审计只记
  脱敏形式。钉钉 API 错误链不得 `%w` 原样上抛（`*url.Error` 含 token URL）——失败返回
  固定文案（对齐 resolveEnterpriseUserID 范式）。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 扫码命中活跃邀请 | 自动激活直达会话 |
| 邀请过期/撤销/已消费 | 按无邀请处理，落通道 B |
| 同 (组织, 手机号) 重复活跃邀请 | 409（部分唯一索引兜底；过期行创建时收割） |
| 通道 B 未选组织 | 总部兜底，通知总部管理员 |
| 非目标组织管理员调审批/邀请接口 | 403 |
| 重复审批/驳回 | 409 已处理 |

## 5. Tests Required

- biz：归一化/脱敏、三档降级、确认路由（自选→该组织、未选→总部、非法 400）、越权 403。
- data/集成：邀请→匹配→消费全链路（含双通知与脱敏审计）、双收件人各自独立任务+明细
  （防坍缩核心断言）、唯一冲突/先建先得、队列路由（成都不进总部）、审批幂等。
- 泄漏防护：注入 `*url.Error` 断言错误串不含 access_token。

## 6. Wrong vs Correct

### Wrong

```text
# 通知任务键 = (注册人, 组织)  → 多收件人撞 background_task_id 唯一索引被 DoNothing 吞掉
# fmt.Errorf("%w", urlErr)    → 带钉钉 token 的 URL 泄入日志
```

### Correct

```text
# 任务键 = (注册人, 组织, 收件人)；每人独立任务+明细
# LookupMobile 失败返回固定文案「查询钉钉通讯录成员失败」，原始错误就地丢弃
```
