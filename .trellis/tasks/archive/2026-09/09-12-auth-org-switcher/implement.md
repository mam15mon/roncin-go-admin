# 登录组织选择与应用内分公司切换器 Implementation Plan

> 前置条件：09-12-netting-commission-and-followups、09-12-exchange-rate-cross-calc
> 均已合并归档后实施。自 main 拉取 `feat/auth-org-switcher` 分支。

## Phase 1: 服务端（proto + 会话 + 登录/切换，原子单元）

1. 契约：`api/auth/v1/auth.proto` — LoginResponse 加 `organization_choices`、Login
   请求加可选 `organization_id`、新 RPC `SwitchOrganization`（POST
   `/api/v1/auth/switch-organization`，需会话鉴权，响应与登录同构）；
   `make -C server api`。
2. biz：抽 `listEnabledMembershipOrganizations`（登录与切换共用）；Login 校验显式
   组织在候选集内；`SwitchOrganization`（候选校验 → ResolvePrincipal 重算 → 事务内
   新建会话 + 按 token hash 删旧会话 + 审计 `auth.organization.switch`）。
3. data：按 token hash 删除单会话（复用已有则不重复建）；候选查询 ForShare 语义。
4. service：Login 组装组织列表、透传所选组织；SwitchOrganization handler 与 DTO
   （复用登录映射）。
5. 测试：biz 候选/切换校验/轮转失效/审计；service 登录列表映射、指定组织 400、
   切换 403 与成功同构；既有 auth 回归。
6. 验证：`go -C server build/vet/test ./...`。

## Phase 2: 前端（登录选择页 + 顶栏切换器）

1. `pnpm run generate:web-client`；登录成功后按 `organization_choices` 分流：多组织
   → 选择视图（组织卡片，点选携 organizationId 完成进入）；单组织 → 直进（现状）。
2. 顶栏切换器：候选列表（排除当前）→ 调 switchOrganization → 替换凭据 → 清空服务端
   状态缓存 → 重拉 auth/me 与菜单 → 回工作台首页。
3. 验证：tsc、biome、定向 vitest（选择页/切换器/状态重置断言）。

## Phase 3: 门禁与收尾

1. `pnpm run check:server` + `pnpm run check:web`；
2. spec：沉淀登录组织选择与切换契约（auth/会话相关规范或新增），登记 index；
3. 提交分组：
   - `feat(auth): 登录组织选择与成员资格候选列表`
   - `feat(auth): 应用内切换组织（会话轮转与权限重算）`
   - `feat(web): 登录组织选择页与顶栏分公司切换器`
