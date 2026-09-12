# 钉钉注册审批与通知路由 Implementation Plan

> 默认排在 auth-org-switcher 之后串行；若用户明确插队则提前。
> 自 main 拉取 `feat/dingtalk-registration-approval` 分支。

## Phase 1: 服务端（Schema/迁移/邀请与激活原语/登录改造/通知路由，原子单元）

1. Ent：`dingtalk_invitations` 实体 + 部分唯一索引；credentials 加
   `requested_organization_id`；迁移 SQL（含索引）；`go -C server generate`。
2. Proto：邀请 CRUD、注册审批 approve/reject、审批队列查询、注册确认带组织；
   权限码入 manifest + permission-keys 重生成；`make -C server api`。
3. biz：邀请用例、`LoginDingTalk` 匹配链（企业 token 手机号反查——按 userId 取
   通讯录 mobile 或 getbymobile 择一）、激活原语（enable+membership+角色+通知）、
   审批/拒绝用例（幂等）、手机号规范化与脱敏。
4. data：邀请仓储、通知后台任务类型与收件人路由查询、审计。
5. service：新 RPC handler 与 DTO。
6. 测试（design 第 4 节 biz/data 集成全量）。
7. 验证：build/vet/test 全量 + data 集成全量 + migrate:dev。

## Phase 2: 前端

1. `generate:web-client`；邀请管理页（分公司管理员视角：建邀请/列表/撤销，手机号
   脱敏展示）；审批队列页（PENDING 列表 + 同意/拒绝，钉钉姓名/头像认领）。
2. 登录注册确认视图补"选择要加入的公司"（多组织时）。
3. 验证：tsc/biome/定向 vitest。

## Phase 3: 门禁与收尾

1. `pnpm run check:server` + `pnpm run check:web`；
2. spec：沉淀"注册审批与通知路由契约"（身份真相源/匹配键/双通道/路由规则）；
3. 提交分组：
   - `feat(auth): 钉钉邀请与自动激活通道`
   - `feat(auth): 注册审批队列与目标组织通知路由`
   - `feat(web): 邀请管理与审批队列页面`
