# 当前行为与规划证据（2026-10-13）

## 登录链路

| 位置 | 已验证事实 |
| --- | --- |
| web/src/pages/user/login/index.tsx:78 | 密码登录多个工作台展示选择页 |
| web/src/pages/user/login/index.tsx:93 | 点击钉钉按钮后获取配置并跳转 authorizeUrl |
| web/src/pages/user/login/dingtalk-callback.tsx:63 | 钉钉回调同样已有工作台选择；此前说明“钉钉直接进入默认工作台”不完整，以代码为准 |
| server/internal/service/auth.go:44 | 配置请求生成新 state Cookie，300 秒有效期 |
| server/api/auth/v1/auth.proto:76 | 配置仅 enabled、authorize_url |
| server/internal/data/dingtalk.go:92 | OAuth2 授权，scope 为 openid corpid；服务端换码并校验企业 |
| server/internal/data/auth.go:219、:306 | primary 影响初始工作台；部门/组映射到所属工作台 |

## 嵌入扫码外部调研边界

官方入口：[实现网页方式登录应用](https://open.dingtalk.com/document/orgapp/tutorial-obtaining-user-personal-information)、[实现登录第三方网站](https://open.dingtalk.com/document/isvapp-server/tutorial-enabling-login-to-third-party-websites)。本轮工具只读取到动态页面标题，未取得 SDK 参数正文，不能声称验证了 SDK 签名和同源规则。

[官方端外扫码示例](https://github.com/open-dingtalk/h5app-out-login-demo) 可确认存在扫码页面与回调配置示例，但 README 使用 sns 临时授权码流程，不能直接作为本仓库 OAuth2 替换方案。搜索摘要出现 DTFrameLogin，尚未以可读取的当前官方正文核实，未定 SDK 版本/API。

后续核验 SDK 来源与版本、OAuth2匹配、回调、state 生命周期、刷新和卸载行为，并用真实应用验证。未读取环境秘密，未调用企业接口。

## 成员和角色

| 位置 | 已验证事实 |
| --- | --- |
| web/src/pages/admin/components/users/UserMembershipModal.tsx:109 | 组织单选，选择后加载角色，角色不必填 |
| server/api/admin/v1/admin.proto:281 | 一次新增一个组织成员关系 |
| server/internal/data/admin_role.go:397 | 空角色直接成功，有角色按所属工作台校验 |
| server/internal/data/admin_user_membership.go:57 | 单项事务锁用户，校验范围/角色，写成员/绑定/审计并维护 primary |
| server/internal/data/admin_user.go:74、:260、:276 | primary 优先选角色编辑对象并替换绑定 |
| web/src/pages/admin/components/users/userConstants.ts:30 | 前端同样按 primary 解析角色编辑组织 |

## 公司与初始化

| 位置 | 已验证事实 |
| --- | --- |
| server/internal/data/admin_organization.go:82 | 手工新公司已初始化管理员角色 |
| server/internal/data/company_administrator.go:12 | 只建角色，不加成员；按 Manifest 过滤系统专属权限 |
| server/internal/data/organization_seed.go:28 | 已有公司直接跳过，新增公司才初始化角色 |
| server/cmd/bootstrap-admin/main.go:121、:183、:188 | 非空用户表拒绝；有初始管理员标记；只加入系统管理 |
| server/internal/data/system_company_authorization_integration_test.go:17 | 现有测试要求初始化角色不加成员 |
| server/internal/data/admin_role.go:24 | 部门/组共用公司角色库 |

未读取真实数据库，不能将“看不到角色”直接认定为角色记录不存在；后续区分角色缺失、停用、候选读取与没有公司成员关系。

## 需要更新的既有约定

- .trellis/spec/server/backend/auth-session-org-switch.md：primary默认进入、bootstrap不自动授公司资格。
- .trellis/tasks/10-09-dingtalk-directory-provisioning/prd.md：默认工作台由管理员维护、多组织沿用单条维护；仅替代相关局部要求，不改变通讯录与人员状态机。
- 根 AGENTS.md 的迁移、生成物、权限、事务、锁序与秘密保护要求继续适用。

## 建议依赖（尚未作为批准实施计划）

扫码 SDK 核验可独立推进；取消 primary 后，多选加入与公司管理员补齐采用最终成员契约，共享 auth/admin/bootstrap 文件按顺序交付。最终验收：创建公司 → 管理员配置 → 员工多组织分配 → 扫码 → 选择工作台 → 明确成员角色编辑。

以上为代码事实及建议，不代表用户已确认管理员范围、停用恢复、批量失败语义或多选角色交互。
