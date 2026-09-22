# 钉钉注册审批与通知路由契约

## 1. Scope / Trigger

适用于钉钉扫码注册（双通道：邀请自动激活 / 首扫认领审批）、邀请管理与审批通知路由。
修改注册流程、邀请实体或审批通知时必须遵循。

## 2. Signatures

```text
POST /api/v1/admin/dingtalk/invitations          创建邀请（kind: TARGETED|GENERIC + 目标组织 + 初始角色 + TTL + 生成 128-bit Token）
GET  /api/v1/admin/dingtalk/invitations          列表（手机号脱敏直显，展示类型与二维码）
DELETE /api/v1/admin/dingtalk/invitations/{id}   撤销（仅 PENDING）
GET  /api/v1/admin/dingtalk/registrations        PENDING 审批队列
POST /api/v1/admin/dingtalk/registrations/{id}:approve|reject
POST /api/v1/admin/dingtalk/registrations/{id}:transfer   一键转派至兄弟分公司
GET  /api/v1/auth/dingtalk/invitations/info      扫码落地页公开信息（最小化响应 + IP 限流保护）
权限码：统一使用 system.user.dingtalk_invitation.manage + DATA_SCOPE_ORGANIZATION（目标组织二次校验）
```

```text
Table: ding_talk_invitations
  token: VARCHAR(64) UNIQUE NOT NULL (128-bit 随机十六进制串)
  kind: TARGETED (1) | GENERIC (2)
  mobile: 可空（TARGETED 必填，GENERIC 留空）
  role_id: 可空（TARGETED 必填，GENERIC 可选）
  (organization_id, mobile) 部分唯一 WHERE status='PENDING' AND kind=1 AND mobile IS NOT NULL
  status: PENDING | CONSUMED | EXPIRED | REVOKED（GENERIC 永不 CONSUMED）
users.dingtalk_requested_organization_id  -- 通道 B 自选/Token锁定目标组织（空=系统管理兜底）
```

## 3. Contracts

- **钉钉是身份真相源**：账号身份字段（unionId/userId/姓名/头像）只来自扫码返回；
  邀请仅存管理员决策（手机号/组织/角色/TTL），预存姓名仅备注。
- **通道 A 自动激活（定向邀请 TARGETED）**：扫码换出企业 userId → 企业 token 点查通讯录 mobile（user/get）
  → 归一化（去空白连字符、剥 +86/0086）→ 等值匹配活跃 TARGETED 邀请（排除 GENERIC，多组织命中**先建先得**）
  → 单事务：消费邀请 + 启用账号 + membership(首个组织 primary) + 初始角色 + 双通知 + 审计。
- **通用入职码（GENERIC）**：多人多次扫码，永不 CONSUMED；手机号留空；扫码注册时
  携带 invitationToken 由服务端锁死目标组织（杜绝前端伪造），进入通道 B 审批队列。
- **扫码落地页最小化公开（GetDingTalkInvitationInfo）**：未登录接口，复用登录同款 IP 限流；
  仅返回组织名、邀请人与有效期，绝不透出 Token、预设角色配置或手机号。
- **向上追溯通知兜底（Escalation）**：审批通知路由目标组织内持权审批人；若目标组织
  审批人集合为空，沿 `parent_id` 逐级向上追溯首个存在有效审批人的祖先节点（直至系统管理根节点），
  确保通知永不死锁；追溯仅为通知路由兜底，不改变人员申请归属。
- **一键转派（Transfer）**：针对误选本组织的 PENDING 申请，管理员可一键转派至兄弟分公司；
  原子更新 `requested_organization_id`，并向新分公司审批人（集成向上追溯）重新入队确定性任务。
- **降级红线**：目录查询失败/归一化失败/无匹配/消费冲突（含并发）**一律静默降级通道 B**
  （RegistrationRequired），warn 日志不含手机号与令牌；钉钉 API 故障永不阻断扫码登录。
- **通道 B 认领**：注册确认可选目标组织（>1 时下发候选，有 Token 时由 Token 锁定隐藏下拉），未选走系统管理兜底；审批通知
  **路由目标组织**（或向上追溯祖先）内启用成员资格 × 启用角色真实持权限者（活跃截断 10），确定性任务
  键为 `(注册人, 组织, 收件人)` 三元组——**每位收件人独立任务+明细**，禁止共享键
  （1 任务=1 明细=1 收件人模型下共享键会静默坍缩为单人）。
- **审批一站式**：approve = 启用+membership+角色+通知本人+吊销历史会话；reject = 停用
  成员资格出队+原因通知本人；幂等（已处理 409）；跨组织审批 403。
- **手机号安全**：完整号码仅匹配用；DTO 出口即脱敏（138****8000）；日志与审计只记
  脱敏形式。钉钉 API 错误链不得 `%w` 原样上抛（`*url.Error` 含 token URL）——失败返回
  固定文案（对齐 resolveEnterpriseUserID 范式）。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 扫码命中活跃定向邀请 | 自动激活直达会话 |
| 扫码通用入职码 | 由 Token 锁定目标组织，进入待审批队列 |
| 目标分公司无审批人 | 沿组织树向上追溯首个有审批人的祖先分发通知 |
| 目标分公司转派兄弟公司 | 原子更新申请组织并向新组织审批人重新发送通知 |
| 邀请过期/撤销/已消费 | 按无邀请处理，落通道 B |
| 同 (组织, 手机号) 重复活跃定向邀请 | 409（部分唯一索引兜底；过期行创建时收割） |
| 通道 B 未选组织且无 Token | 系统管理兜底，通知系统管理管理员 |
| 非目标组织管理员调审批/转派/邀请接口 | 403 |
| 重复审批/驳回 | 409 已处理 |

## 5. Tests Required

- biz：归一化/脱敏、三档降级、确认路由（自选→该组织、未选→系统管理、非法 400）、越权 403。
- data/集成：邀请→匹配→消费全链路（含双通知与脱敏审计）、双收件人各自独立任务+明细
  （防坍缩核心断言）、唯一冲突/先建先得、队列路由（成都不进系统管理）、审批幂等。
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

## 独立公司工作台补充（2026-09-22）
未指定公司的申请仅在系统管理入口待处理，不得直接审批成系统管理成员；须先转派到启用公司，再赋予该公司角色。定向邀请只允许公司目标。系统管理通知兜底通过显式系统节点定位，不能沿虚构集团父链推断。
