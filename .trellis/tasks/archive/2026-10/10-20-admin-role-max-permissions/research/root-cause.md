# 根因证据

2026-10-20 本地只读核对：admin 为启用初始管理员，系统管理/北京/成都均有真实启用成员和 administrator 绑定；全字典 305，系统管理 44/44、北京 287/287、成都 287/287，无缺失或额外权限。

RolesPanel 请求全字典；RoleFormModal.handleSelectAllWrite 直接复制 matrixModel.allLeafKeys。公司全选混入不适用公司的 18 项系统权限；data.CreateRole/UpdateRole 经 PermissionAllowedInWorkspace 返回 ErrAdminPrivilegeEscalation。biz.checkRoleFunctionalPrivileges 已允许 IsSuperAdmin，非 admin 身份识别失败。

permissionCapabilities 已由后端按当前工作台投影，足以派生可授予候选；type-safety.md 禁止用 permissions/roleScopes 回退。完整字典也用于权限浏览页，不能全局裁剪 API。

已通过的排查验证：access/biz 工作台及权限派生定向测试；web/src/access.workspace.test.ts 的 8 项用例。尚未实现，不能据此声称创建场景已修复。
