# 验证记录

## 服务端真实 PostgreSQL

本轮测试仅连接专用 roncin_go_admin_integration 库。使用现有 getIntegrationData 辅助方法建立随机隔离 Schema，按正式 SQL 迁移初始化，用例结束自动清理；未修改业务库授权或创建业务角色。

- `go -C server test -p 32 ./internal/data -run '^TestSystemCompanyAdministratorAuthorizationBoundaries$' -count=1 -v`：PASS，2.206 秒，无 SKIP。
- `go -C server test -p 32 ./internal/data -run '^TestCompanyAdministratorCreatesMaximumRoleFromEffectivePermissions$' -count=1 -v`：PASS，2.203 秒，无 SKIP。日志确认有效公司权限 287 项、完整目录 305 项；实际通过 biz/data 创建并编辑完整公司角色，直接全目录创建/编辑均拒绝，原授权不变且非法角色无残留。
- access/biz 定向：`TestAdminUsecase.*Role`、`TestWorkspaceSeparatesSystemManagementAndCompanyBusiness`、`TestPrincipalPermissionsAreDerivedOnlyFromRoleGrants` 及汇率/钉钉开通/通知工作台边界：两包 PASS。

## 前端定向检查

实现代理已完成 10 文件、92 测试 PASS，涵盖角色页面、真实全选提交、角色编辑、字典页、树/矩阵/依赖、access 和工作区查询；tsc、修改文件 Biome、产品 TSX 的 Ant Design lint 均通过。

roles.grantable.test.tsx 实际执行 RolesPanel、RoleFormModal、AppProvider、React Query、权限矩阵与全选按钮，只浅替换 ProTable / ModalForm 框架外壳；覆盖公司 287/系统 44 载荷、普通用户、缺能力、原授权保留、同身份能力变化及 OrganizationWorkspace 切换重建。

## 独立审查

独立检查代理已修复首次字典加载失败/后台重查失败时误开放提交的相邻缺口，并补充页面刷新和弹窗重新加载的真实恢复交互。审查通过，无剩余阻断项，完整证据见 research/review.md。

完整 check:web 退出 0：269 文件、1523 用例通过；既有 sea-template.test.tsx 的 1 文件、12 用例跳过，不计作通过。完整门禁后补齐重试入口的最终代码另通过 3 文件、27 用例定向回归、tsc 和修改文件 Biome。最终从 main 建立的独立整合工作区已再次运行完整 check:web，退出 0：269 文件、1523 用例通过，既有 1 文件、12 用例跳过，Vitest 56.06 秒。此轮覆盖全部重试交互并排除并行海运提交；`git diff --check main` 通过。

## 验收映射

- A1：roles.grantable 的公司全选实际载荷 + Go 真实最大公司角色创建/更新，均 PASS。
- A2：系统全选 44 项回归、全字典覆盖 305 项及既有字典页测试，PASS。
- A3：普通用户、缺能力、同身份能力变化与真实工作区切换回归，PASS。
- A4：正常编辑成功、不可授予原授权保留且阻止按钮/直接提交，PASS。
- A5：候选范围说明、失败后手动重载与原选择保留、后端非法全字典拒绝和普通角色防提权测试，PASS。

独立审查已完成，无未处理的本任务阻断项。业务数据库未变更；初始管理员无需重新开通。修复代码独立提交为 87fcd7b7，后续记录提交只补验收与归档。
