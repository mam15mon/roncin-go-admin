# 实施计划：多组织加入

## 前提与步骤

1. [ ] 取消主要组织子任务已完成；用户已批准整体方案；激活本子任务并加载定向规范。
2. [ ] 确认现有新增路由/权限绑定、全部客户端调用点，完成批量 Proto 与领域输入。
3. [ ] 为 AdminUsecase 显式注入 Transactor，同步 Wire 与测试构造；实现共享事务、锁内授权校验、角色归属和原子写入，复用原成员/审计错误，确定锁序无环。
4. [ ] 生成 API/OpenAPI/Web；实现多选逐行角色表单、候选隔离与失败保留。
5. [ ] 实现后补前端交互及真实库失败/并发验证，不 TDD。
6. [ ] 更新成员管理规范和验证记录，完成质量检查并提交。

## 命令

```bash
make -C server api
pnpm run generate:web-client
go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run 'Admin|Membership' -count=1
pnpm --dir web exec vitest run src/pages/admin/components/users/UserMembershipModal.test.tsx src/pages/admin/components/users/UserFormModal.test.tsx src/pages/admin/users.test.tsx
pnpm --dir web tsc
git diff --check
```

按最终新增测试名执行真实库 -v 定向测试并确认无 SKIP。组件重型用例单文件不超过 10 个，耗时超过 20 秒按场景拆分。完成阶段运行 check:fast，因为本子任务改变权限相关写接口及事务。

## 必测结果

- [ ] 三个组织中第二个失效，第一和第三也不落库，前端保留全部输入。
- [ ] 人为触发第二条写入/审计失败，数据库本批零变化。
- [ ] 两个并发请求不能重复加入或覆盖角色，失败方全部撤回。
- [ ] 不同公司同名角色仍按 ID 和归属精确绑定，角色全空可加入。
- [ ] 撤销组织选择后异步返回不串行；重试修正组织后只提交当前可见行。

接口一次性替换，回退需前后端成组；不自动删除已经成功加入的成员作为回退。
