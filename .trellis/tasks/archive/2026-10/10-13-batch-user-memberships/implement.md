# 实施计划：多组织加入

## 前提与步骤

1. [x] 取消主要组织子任务已完成；用户已批准整体方案；激活本子任务并加载定向规范。
2. [x] 确认现有新增路由/权限绑定、全部客户端调用点，完成批量 Proto 与领域输入。
3. [x] 为 AdminUsecase 显式注入 Transactor，同步 Wire 与测试构造；实现共享事务、锁内授权校验、角色归属和原子写入，复用原成员/审计错误，确定锁序无环。
4. [x] 生成 API/OpenAPI/Web；实现多选逐行角色表单、候选隔离与失败保留。
5. [ ] 实现后补前端交互及真实库失败/并发验证，不 TDD。
6. [x] 更新成员管理规范和验证记录，完成质量检查并提交。

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


## 本轮实施记录（2026-10-13）

- 新增契约改为 `CreateUserMemberships`、`POST /api/v1/admin/users/{user_id}/memberships:batchCreate`，一次提交 `items`，单条编辑继续使用既有入口；旧单项新增 API 与调用已同步移除。
- `AdminUsecase` 显式注入 `Transactor`，全部构造调用及 Wire 生成物同步；共享事务内按「操作者和目标用户主键排序独占锁 → 全组织树主键排序共享锁 → 双方成员主键排序独占锁 → 目标和操作者角色主键排序共享锁」读取权威数据。当前组织树为小数据集，权限、角色归属及工作台解析均复用全树读取，所以本轮直接锁全树，避免另建依赖集合解析框架。
- 锁后重建操作者 Principal，检查 `system.user.update` 与组织管理范围，空角色同样检查；复用领域提权检查并校验所属公司/启用角色。启用、停用的已有成员记录均明确冲突，不自动恢复。所有校验完成后才创建成员、角色绑定和审计，失败整批回滚，提交后用普通上下文重读。
- 相邻钉钉开通仓储同步两用户及组织/成员固定锁序，防止与批量成员操作在操作者、目标互换时反向等待；保留其逐人授权语义。
- 表单改为多选组织，每行角色通过 React Query 按组织 ID 隔离；取消组织删除该行输入。批量新增与单项编辑均检查成功响应；错误/undefined 返回不会关闭表单，成功后刷新父页面。
- 现有 OpenAPI 生成器把字面 `:batchCreate` 误识别为变量；`web/scripts/generate-services.mjs` 依据契约中完整动作路径和已声明参数进行生成期修复，未声明占位或参数形态不处理。生成客户端最终 URL 已核对为字面 `memberships:batchCreate`，未手改生成物。
- 既有测试仅机械同步构造器、接口签名与必须的真实权限夹具，未增加用例、未运行测试。按本轮主会话执行约束，方案原列测试和真实数据库并发/回滚验收尚未执行；不把未执行项标为通过，未操作实际业务库。

已执行并通过的非测试检查：

- `make -C server api`
- `go -C server generate ./cmd/server`
- `pnpm run generate:web-client`
- `go -C server build -p 32 ./...`
- `go -C server vet -p 32 ./internal/biz ./internal/service ./internal/data ./cmd/server`（包含既有测试源码静态编译检查，不运行用例）
- `pnpm --dir web tsc`
- 修改的四个 Web 源/既有用例文件及生成入口 `biome check`
- 两个改动的 Ant Design 组件 `antd lint --format json`，零问题
- `buf lint server`
- 原始前端架构扫描 `node web/scripts/check-architecture.mjs`（431 文件、1592 依赖、零违规）
- 原始 Go 层级扫描 `go -C scripts/layer-check-go run .`（1227 文件、11573 import、零违规）
- `git diff --check`

未运行 `check:fast`、`check:architecture`/`check:layers:go` 包装入口，因为这些包装会执行测试。
