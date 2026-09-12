# 钉钉分公司专属邀请与入职审批 Implementation Plan

## Phase 1: 服务端核心模型与业务闭环 (Server Models & Biz)

1. **Schema & 迁移脚本**：
   - 完善 `dingtalk_invitations` Schema（增加 `token`，`mobile` 与 `role_id` 设为可选）；
   - `users` 增加 `dingtalk_requested_organization_id`；
   - 更新数据库迁移 SQL；执行 `go -C server generate`。
2. **Proto 契约与权限清单**：
   - `RegisterDingTalkUserRequest` 改造为携带 `invitation_token`；
   - 增加邀请管理（创建/查询/撤销）与审批流转（查询/同意/拒绝/转派）接口；
   - 维护 `server/internal/access/manifest.go` 权限清单，重新生成权限键与 OpenAPI/客户端代码。
3. **biz 领域层业务逻辑**：
   - 实现 `DingTalkRegistrationUsecase`：邀请创建、撤销、列表；
   - 通道 A 手机号匹配与免审秒激活逻辑；
   - 通道 B 待审批入队、审批通过（User 激活 + Membership + 赋权）、驳回、一键转派（Transfer）；
   - 钉钉企业通讯录手机号点查与企业白名单校验；
   - 单元测试（针对状态机、权限边界、转派）。
4. **data 仓储层与事务封装**：
   - 事务保证：审批同意/转派的悲观锁与原子提交；
   - 钉钉工作通知卡片异步入队。
5. **service 传输层接入**：
   - 转换 HTTP/gRPC DTO 并接入鉴权中间件；
   - 全套 Go 编译与单元测试验证。

## Phase 2: 前端管理后台与扫码落地页 (Web)

1. **前端邀请管理与审批弹窗**：
   - 在用户管理中增加「+ 邀请成员」弹窗（支持生成二维码与链接、定向手机号）；
   - 待审批人员列表与处理弹窗（支持分配角色同意、驳回理由填写、一键转派兄弟公司）。
2. **扫码落地页改造**：
   - 扫码携带 `?invite=<token>` 时，展示分公司欢迎卡片与安全提示；
   - 彻底移除前端公开自选分公司的下拉框。
3. **前端代码检查**：
   - 定向 vitest 测试与 Biome 检查。

## Phase 3: 全量门禁与收尾 (Verification)

1. `pnpm run check:server` 与 `pnpm run check:web`；
2. 真实流程端到端验证；
3. 按 Conventional Commits 分组提交。
