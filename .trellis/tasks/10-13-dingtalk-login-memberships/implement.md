# 整体实施与验收计划（已批准）

## 启动前提

- [x] 任务创建获授权，D1–D8 产品决策已确认。
- [x] 父子任务 PRD、设计、实施计划及上下文清单齐备。
- [x] 用户于 2026-10-13 在最新整体摘要后回复“开始”，明确批准实施。
- [x] 实施前 git status、确认独立分支和子任务基线；按需加载 trellis-before-dev 及前端规范。

工作区：/home/vince/roncin-go-admin-worktrees/dingtalk-login-memberships；分支 codex/dingtalk-login-memberships，创建时基线为 main 的 8f866832；最终规划同步 cee4bffb（默认公司仅北京/成都、开发数据归属北京）。实施启动前仍需检查 main 是否继续推进，保留相邻改动。旧 background-task-notify 分支中同名任务只是此前规划版本，不覆盖当前文档。

## 执行顺序

1. 批准后先验证扫码嵌入协议在本应用可用，记录域名/回调前提及未完成项，不提交凭据。若存在改变 MVP 的平台限制，先回到规划更新方案。
2. 激活 remove-primary-organization，实施、针对性检查、规范同步、本地提交；完成该子任务才改变成员/角色基础契约。
3. 激活 batch-user-memberships，落实 D6/D7，完成角色隔离与原子回滚验收后提交。
4. 激活 company-admin-provisioning，落实 D5/D8，验证新公司/种子/重复 bootstrap、停用跳过、身份错误后提交。
5. 激活 dingtalk-qr-login，完成扫码默认页面及最终候选契约，实测登录并提交。
6. 父任务联合验收，更新各验收记录、相关规范及原任务替代说明，完成最终检查与归档。

每个实现派发以 Active task: <task.py current 的路径> 开头，使用 trellis-implement，质量检查使用 trellis-check；主会话协调，不创建额外用户聊天。实现/检查不得越过已批准子任务范围。

## 联合验收清单

- [ ] 建立新公司 → 初始管理员能切换进入 → 能分配公司管理员角色。
- [ ] 普通系统管理员不被自动加入其他公司。
- [ ] 单员工多选公司/部门，逐行选填角色，角色全空和部分空都可保存。
- [ ] 一个组织失效或后续写入失败 → 整批回滚，表单保留。
- [ ] 员工直接扫码 → 单公司直进，多公司需选；管理员密码同口径。
- [ ] 修改成员角色只作用于该成员；修改基础资料不修改角色。
- [ ] bootstrap 首次/重复补齐不改密码、不重复建授权，停用项明确保留。
- [ ] Schema/契约/生成物/种子同步，没有 primary 或 is_default 的运行时消费。

## 验证命令与记录

各子任务 implement.md 包含定向测试，先局部、再门禁；不写测试驱动实现。普通提交先 git diff --check、相关测试和必要类型检查。

```bash
pnpm --dir web tsc
pnpm run check:fast
git diff --check
```

真实数据库测试需有效的 RONCIN_INTEGRATION_DATABASE_SOURCE，通过现有测试夹具创建独立 schema、按正式 SQL 初始化；使用 -v 审阅 SKIP，不打印连接串。所有结果写入子任务 verification.md，父任务最终记录 AC1–AC10 对应证据。

本轮为文档收敛，只检查 JSON、父子关系、路径和 diff，无需运行产品测试或构建。实施不默认执行 build；若修改生产入口/构建配置或新增依赖，则补对应构建验收。

## 完成与回退

提交仅包含本任务文件，保留其他用户工作；不得用禁用规则解决测试失败。子任务验收完成后再归档。出现方案变更先更新 PRD/设计，不直接越过新产品决策；Schema 已变更时不能只回退旧代码，按 design.md 评估完整回退。
