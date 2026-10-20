# 管理员最大权限角色修复审查

## 结论

审查通过，未发现尚未处理的本任务阻断项。角色候选来自当前 `permissionCapabilities` 与完整字典交集；公司全选提交 287 项，系统管理全选提交 44 项，完整字典保留 305 项。没有新增用户名特判、后端权限豁免或业务数据库授权操作。

## 发现与修复

- 文件：`web/src/pages/admin/roles.tsx`、`components/roles/RoleFormModal.tsx`。
- 问题：首次权限字典失败后 `isPending` 会变为 false，原实现因此重新开放新增和空候选保存；字典重新加载失败时也不能把失败当成已确认候选。
- 修复：新增按钮仅在查询成功时开放；表单区分加载和失败，加载/失败都禁止按钮和直接 `onFinish` 提交。保留缓存候选、已选项和编辑角色原始权限；错误状态明确说明暂不能保存。
- 恢复交互：页面原有「刷新」继续调用列表 `actionRef.reload`，同时通过当前权限门控调用 `permissionQuery.refetch`；弹窗错误提示提供「重新加载权限」，复用同一回调。未新增手写请求链、自动重试或表单重新回填。
- 回归：`roles.grantable.test.tsx` 新增真实按钮交互，覆盖首次失败阻止直接提交、页面刷新恢复、全选后重新加载失败保留 287 项并阻止提交、弹窗重新加载成功后原选择完整提交。

## 实际路径复核

1. `access.ts` 抽取原有判定为共享 `createPermissionCapabilityChecker`，保留 organization / organization_tree / all 等级及未知/self 拒绝；页面不通过 permissions、roleScopes、用户名回退。
2. `RolesPanel` 仅在渲染侧过滤角色候选，树、矩阵、统计、批量、单项和默认基础权限使用同一集合；完整 `PermissionsPanel` 的请求与过滤不变。
3. `RoleFormModal` 对显式空 permissions 不回退 allLeafKeys。编辑对象和所有批量操作保留不能授予的既有键；即使绕过按钮直接执行 onFinish 也返回 false，不向后端发送裁剪后的授权。
4. 当前能力对象变化会重算候选；新建已选项变为不可授予时先阻止保存，用户可重新选择。真实 `OrganizationWorkspace` 按用户 ID + 工作台 ID 重建 Provider 和表单，切换不继承公司选择或旧缓存。
5. Go 新集成测试使用 `newAuthPrincipalFixture` → 专用连接环境变量 → 独立 Schema → 正式 SQL 迁移，不在业务库维护实际 admin。真实用例验证有效公司权限创建和更新成功，全字典创建/更新拒绝，非法更新不改原角色、非法创建不留角色；服务端生产逻辑没有修改。
6. 主会话已在专用 PostgreSQL 实跑 `TestSystemCompanyAdministratorAuthorizationBoundaries` 与 `TestCompanyAdministratorCreatesMaximumRoleFromEffectivePermissions`，两项 PASS 且无 SKIP；本次审查未改 Go 测试，依约没有重复数据库测试。

## 验证证据

- `pnpm run check:web`：退出码 0；权限键、Proto 常量、模块架构、重复扫描、全量 lint、TypeScript、全量 Vitest 均通过。完整日志 `/tmp/roncin-admin-role-check-web.log`。
- 全量 Vitest：269 文件通过、1 文件跳过；1523 用例通过、12 用例跳过；耗时 58.72 秒。命令总耗时约 69 秒。
- 时序说明：全量门禁运行之后，依主会话复核补齐了真实权限重试按钮及相应回归；最终代码另行通过下列定向门禁，不将之前的全量结果冒称为新增重试交互的验证。
- 最终定向 Vitest：`roles.grantable.test.tsx`、`RoleFormModal.test.tsx`、`roles.test.tsx`，3 文件 27 用例全部通过，耗时 14.38 秒；日志 `/tmp/roncin-admin-role-review-targeted.log`。
- 最终 `pnpm --dir web tsc`：退出码 0；日志 `/tmp/roncin-admin-role-review-tsc.log`。
- 最终修改文件 `biome check`：3 文件通过，无修复；`git diff --check` 通过。

## 未修复项与验证限制

- 12 项既有跳过全部位于 `web/src/pages/orders/templates/sea-template.test.tsx:35` 的 `describe.skip`，含 8 项顶层与 4 项嵌套分单回显测试。源码注明海运模板迭代中暂跳过；本任务不恢复或重构该套件，不将跳过计作通过。
- 全量 Biome 存在 1 个 warning（`components/ui/master-data-template/types.ts:107` 的 noConfusingVoidType）及 7 个 info（用户列测试 2 个无用 fragment，提成规则测试 5 个字面键提示）。均位于未改文件，不阻断当前门禁，保留原样。
- 全量测试日志有 localhost:3000 的 ECONNREFUSED 输出，进程与所有执行的测试仍通过；本次未为清理旁支日志修改其他模块。
- 并行会话修改的海运正文、海运测试及其 component-guidelines 规范不属于本任务审查与提交范围，均保留未动。主会话须精确暂存本任务文件。
- 此审查覆盖源代码、实际角色按钮提交回归和已执行集成证据；未代用户在业务库创建正式角色。Git 提交与合并由主会话继续。

## 最终合入集合复验

主会话在独立工作区从 main 提取本任务修复（87fcd7b7），排除并行海运提交后，对包含全部重试交互的最终代码再次运行完整 `pnpm run check:web`，退出 0：269 文件通过、1 文件跳过；1523 用例通过、12 用例跳过；Vitest 56.06 秒。日志 `/tmp/roncin-admin-role-merge-check-web.log`。`git diff --check main` 通过。
