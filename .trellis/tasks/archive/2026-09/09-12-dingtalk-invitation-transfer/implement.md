# 钉钉入职专属码、向上追溯与一键转派 Implementation Plan

> 分支：`feat/dingtalk-invitation-transfer`（基于 `main`）

---

## Phase 1: 专属邀请码与 Schema 演进 (Generic Invitation & Token)

1. **Schema & 迁移更新**：
   - 更新 `server/internal/data/ent/schema/dingtalk_invitation.go`：
     - 新增 `token` 字段（string 64, unique, not null, 128-bit 随机 Hex）；
     - 新增 `kind` 枚举字段（`TARGETED` / `GENERIC`）；
     - `mobile` 设为可选，部分唯一索引更新为：
       `WHERE status = 'PENDING' AND kind = 'TARGETED' AND mobile IS NOT NULL AND mobile != ''`；
     - `role_id` 设为可选；
   - 编写手写 SQL 迁移脚本并执行 `go -C server generate`；
   - 执行 `pnpm run migrate:dev` 验证。
2. **Proto 契约与 Token 解析**：
   - 更新 `server/api/admin/v1/admin.proto`：
     - `CreateDingTalkInvitationRequest` 增加 `kind` 枚举，`mobile` 设为可选；响应返回 `invitation_url`；
     - 新增 `GetDingTalkInvitationInfoRequest`（提供落地页展示用）；
   - 执行 `make -C server api` 生成绑定代码与 OpenAPI。
3. **biz 领域层业务逻辑**：
   - 改造 `DingTalkRegistrationUsecase`：
     - `CreateInvitation` 支持 `kind = GENERIC` 生成与 `crypto/rand` 128-bit Token 签发；
     - 手机号自动匹配链仅处理 `kind = TARGETED`，GENERIC 行显式隔离不被消费；
   - 编写针对 Token 防篡改、GENERIC 多次扫码不变 CONSUMED 的单元测试。
4. **OAuth 穿透与服务层打通**：
   - 改造登录流程，支持 `invite` Token 随 OAuth state / 临时 Cookie 穿透；
   - 注册请求解析 Token 并绑定目标组织，验证客户端防伪造测试。

---

## Phase 2: 向上追溯通知路由 (First Ancestor Escalation)

1. **祖先追溯算法实现**：
   - 仓储层实现 `GetParentOrganizationID(ctx, orgID)`；
   - 用例层实现 `ListApproverRecipientsWithEscalation`：沿 `parent_id` 查找首个有候选审批人的祖先节点（直至总部）；
   - 保证收件人确定的三元组防坍缩任务键：`dingtalk.reg:%s:%s:%s`。
2. **代管通知模板渲染**：
   - 当 `isEscalated == true` 时，通知卡片中注入代管提示：`【XX 分公司新员工待审批（该组织暂无管理员，由总部代管审批）】`；
3. **测试验证**：
   - 编写单元测试模拟：有管理员直接命中、单级无管理员父级命中、多级无管理员总部兜底断言。

---

## Phase 3: 一键转派与前端全链路交互 (Transfer & Web)

1. **一键转派后端能力**：
   - Proto 新增 `TransferDingTalkRegistrationRequest`；
   - 用例层实现 `TransferRegistration`（仅允许转派 PENDING 申请，悲观锁原子更新 `users.dingtalk_requested_organization_id`，新组织触发向上追溯通知）；
   - 编写转派权限与原子状态机单元测试。
2. **前端管理后台与弹窗开发**：
   - 重新生成客户端 `pnpm run generate:web-client`；
   - 邀请弹窗：支持选择「通用入职码」与「定向手机号邀请」，生成后展示二维码与复制链接；
   - 审批弹窗：新增「一键转派」表单（选择目标兄弟分公司并输入原因）；
   - 扫码落地页：强锁定展示对应分公司欢迎卡片与醒目提示，隐藏公司自选下拉框。
3. **前端检查**：
   - 执行 `pnpm --dir web tsc` 与 `pnpm --dir web biome:lint`。

---

## Phase 4: 全量门禁与收尾 (Verification & Specs)

1. **端到端业务闭环验证**：
   - 生成通用码 -> 员工扫码申请 -> 目标分公司无管理员自动追溯总部 -> 总部转派给兄弟分公司 -> 兄弟分公司管理员审批分配角色 -> 员工激活入职；
2. **全量代码门禁**：
   - `pnpm run check:server`；
   - `pnpm run check:web`；
3. **更新架构规范**：
   - 修订 `.trellis/spec/server/backend/dingtalk-registration-approval.md`，沉淀专属码与向上追溯契约；
4. 按 Conventional Commits 分组提交。
