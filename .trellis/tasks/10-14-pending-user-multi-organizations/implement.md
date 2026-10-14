# 实施与验收

## 顺序

1. 创建 codex/pending-user-multi-organizations 分支，记录并保护登录页已有改动。
2. 后端 implement 子代理修改单人/批量契约、统一 items 转换、领域规则、仓储逐人原子写入和全部受影响调用方测试；运行 make -C server api 与 pnpm run generate:web-client。
3. 前端 implement 子代理独立负责用户管理组件和测试，抽取角色行供两个入口复用，实现多选、删除重选清空和整体角色要求；使用生成 items 客户端。
4. 产品实现之后补充针对性验证（禁止 TDD），包括业务/传输参数、跨公司多目标 PostgreSQL 原子性和并发、前端相邻成员操作。
5. 在已有 CI PostgreSQL 服务下新增显式注入专用变量的定向开通事务测试步骤，防止关键用例只被默认扫描而 SKIP；trellis-check 子代理审查最终跨层代码、规范和测试。
6. 主会话运行 pnpm run check:fast（契约/授权高风险需要完整门禁），核实真实 PostgreSQL 目标用例 PASS，git diff --check。未涉及构建入口/配置，无固定 build。
7. 更新专用开通及成员契约规范与任务验收记录，Conventional Commit 中文提交。
8. 合并到本地 main 后核验提交可达、用 git branch -d 安全删除任务分支；归档本任务并记录会话，不归档其他任务。

## 验收重点

- 单人与批量、多公司/同公司多部门、部分组织无角色合法、全部无角色拒绝。
- 非法 UUID/重复组织/空项/组织数量上限/批量人数上限拒绝。
- 公司工作台拒绝开通、系统工作台目标拒绝、停用祖先拒绝、角色错公司/停用/操作者撤权拒绝。
- 无效后续目标和写入/审计/通知失败时整人回滚，既有停用关系/账号状态保持；并发第二次不覆盖第一次。
- 跨人员部分成功保留，仅失败人员重试；每人只一条通知。
- 前端快速角色查询、删除重选、无角色提示、响应中断、失败输入保留，已开通成员加入/编辑回归。

## 命令

- go -C server test ./internal/biz ./internal/service ./internal/data -run 'DingTalk|AdminUsecaseAuthorize|AdminUsecaseBatch' -count=1
- pnpm --dir web exec vitest run src/pages/admin/components/users/DingTalkAssignmentModal.test.tsx src/pages/admin/components/users/UserMembershipModal.test.tsx
- pnpm --dir web exec biome check <本次修改文件>
- pnpm --dir web tsc
- 专用 RONCIN_INTEGRATION_DATABASE_SOURCE 下运行受影响真实 PostgreSQL 测试，记录 PASS/SKIP，完成后清理本次测试资源。
- pnpm run check:fast
- git diff --check

## 审查与分工

后端代理拥有 server/ 与本次必需生成物；前端代理拥有 web/src/pages/admin/ 的相关组件与测试（不修改生成物）。主会话拥有 Trellis 任务、spec、现有 CI 的定向 PostgreSQL 步骤、提交/合并和清理。多个代理共享工作区，不能回退其他人的改动。危险点为 admin_user.go 的返聘覆盖语义与锁序，单人 API 外部调用测试，以及通知事务。
