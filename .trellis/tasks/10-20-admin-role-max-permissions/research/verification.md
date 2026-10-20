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

完整 check:web 退出 0：269 文件、1523 用例通过；既有 sea-template.test.tsx 的 1 文件、12 用例跳过，不计作通过。完整门禁后补齐重试入口的最终代码另通过 3 文件、27 用例定向回归、tsc 和修改文件 Biome。整合工作区还将针对最终主分支集合再次验收。
