# 多组织成员加入：审查与验证记录

日期：2026-10-13。比较基线为前置提交 `46b2da23`；本记录仅覆盖批量成员子任务及其必需的生成器修复。

## 结论

本子任务代码可提交。已核对真实调用路径及生成物，未发现尚未处理的阻塞问题；没有将静态审查记为数据库事务或浏览器交互实测通过。

## 已核对的行为

- 单项新增 RPC 与客户端调用已统一替换为 `CreateUserMemberships`，HTTP 路径为 `POST /api/v1/admin/users/{user_id}/memberships:batchCreate`，沿用 `system.user.update` 权限。一项请求同样走批量入口，没有保留旧接口转发。
- service 与 biz 拒绝空批次、非法/空 UUID、重复组织。角色选填；每项按目标组织所属工作台校验，不复制不同公司的角色 ID。
- 用例显式注入 `biz.Transactor`，Wire 和调用方同步；回调错误原样返回。仓储嵌套事务合并到同一事务上下文，成员、角色绑定和成功审计一起提交。返回成员 ID 后，用例在事务提交后的普通上下文重读响应。
- 待分配与停用账号均拒绝新增成员。已有启用、停用成员关系都返回冲突，不能自动恢复或替换既有授权。
- 操作者与目标用户按 ID 排序独占锁定，随后组织全树按 ID 共享锁、双方现有成员按 ID 独占锁、目标及操作者角色按 ID 共享锁。锁内重新解析操作者 Principal，检查 `UserUpdate` 权限、管理范围、组织祖先启用状态和领域提权规则。空角色也执行管理权限及组织范围检查。
- 新成员绑定直接创建，不在持有角色锁后新增已有成员锁。钉钉开通相邻路径已调整为相同用户、组织、成员、角色顺序；已查阅成员编辑/移除及角色修改/删除的相关锁路径，未发现本次修改产生的反序锁环。此结论属于代码审查，尚未进行真实并发验收。
- 只有已通过管理范围校验的目标才附带组织名称，越权目标不暴露名称。数据库写入或审计失败会退出共享事务，未吞错后继续提交。
- UI 多选后每个组织独立一行，角色查询按组织 key 隔离。撤销选择从 `rolesByOrganization` 清除对应项，动态字段 `preserve={false}`，重新选择不复用已删除的角色。提交仅构造当前所选组织的载荷。
- 新增及编辑均验证有效成功响应后才提示成功、关闭并刷新；返回空响应或失败时返回 `false`，保留表单。此前发现的编辑分支无条件成功关闭问题已处理。
- 新增规范 `user-membership-batch.md` 与当前实现一致，认证和钉钉开通相关规范已同步。

## 审查期间处理的问题

1. 既有系统授权用例的目标已有系统成员关系，新冲突检查原先可能先返回 `ADMIN_USER_MEMBERSHIP_EXISTS`，无法到达角色提权断言。实现代理将角色提权复核放在已有关系冲突之前，先检查授权再报告冲突；既有用例也补入真实有限成员管理员的角色与绑定，避免依赖伪 Principal。已复核最新代码和既有夹具，未执行该测试。
2. `@umijs/openapi` 会把字面动作后缀 `:batchCreate` 误转为模板变量。实现代理在 `web/scripts/generate-services.mjs` 按 OpenAPI 声明生成字面动作路径修复映射：普通路径不进入映射；动作名称与声明路径参数同名时不处理；所有 `{param}` 必须有声明，声明参数必须以 `{param}` 出现。最终客户端 URL 为字面 `memberships:batchCreate`，未手改生成客户端。

审查代理没有额外修改产品代码，仅补本记录。

## 实际检查结果

以下定向检查全部退出码为 0；本轮没有运行测试包装脚本或应用测试：

- `go -C server vet -p 32 ./internal/biz ./internal/service ./internal/data`。
- `pnpm --dir web tsc`，使用 `tsc --noEmit`。
- 定向 `biome lint`：`UserMembershipModal.tsx`、`UserFormModal.tsx`、`userConstants.ts`、既有 `UserMembershipModal.test.tsx`、`scripts/generate-services.mjs`，5 个文件通过，无修补。
- `node web/scripts/check-architecture.mjs`：431 个产品文件、1592 条本地依赖、4 条 feature 依赖边，违规 0。
- `go -C scripts/layer-check-go run .`：1227 个 internal 文件、11573 条 import，违规 0。
- `git diff --check`：通过。
- 重跑 `pnpm run generate:web-client`：成功；客户端目录、OpenAPI 输入和生成枚举的 SHA-256 指纹前后变化为空，生成内容可复现。

实现代理另已报告全服务端 `go build`、定向 vet、TypeScript、定向 Biome 与 Ant Design 检查通过。检查脚本已先确认实际内容，未使用含测试的 `check:fast`、`check:architecture` 或 `check:layers:go` 包装入口。

## 验证限制

- 按本轮约束，没有新增或运行业务单元/集成测试、Vitest、Playwright 或检查器自身测试。既有测试仅同步构造器、接口、夹具及陈旧断言/文案。
- 未连接或写入实际数据库；没有实测后续项失效、绑定/审计失败时的整批回滚、并发重复加入、角色撤权和组织停用竞争。
- 未进行真实浏览器多行选角、撤销重选、迟到候选及失败重试交互验收。UI 状态保留和请求隔离结论来自代码路径审查及类型检查。
- 提交后响应重读若遇到独立的数据库读取失败，已提交写入不会由响应读取回滚；该行为与已批准的提交后重读方案一致，不能将响应读取错误理解为写事务未提交。
