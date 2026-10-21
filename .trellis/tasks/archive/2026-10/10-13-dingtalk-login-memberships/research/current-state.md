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

## 嵌入扫码外部调研（2026-10-13 补充）

官方入口：[实现网页方式登录应用](https://open.dingtalk.com/document/orgapp/tutorial-obtaining-user-personal-information)、[实现登录第三方网站](https://open.dingtalk.com/document/isvapp-server/tutorial-enabling-login-to-third-party-websites)。动态文档工具仍只返回标题，未取得文档正文。

本轮使用 Python urllib 直接读取[钉钉托管的 0.21.0 登录脚本](https://g.alicdn.com/dingding/h5-dingtalk-login/0.21.0/ddlogin.js)，响应为 2232 字符，未执行或存入仓库。可直接验证：

- 导出 DTFrameLogin，创建 login.dingtalk.com/oauth2/auth 的 iframe，使用 iframe=true。
- 必需 client_id、redirect_uri、response_type、scope；转发 state、prompt、corpId、org_type 等参数。
- 监听 message；成功结果带 redirectUrl，从中读取 authCode、state；失败有 error/errorMsg。
- 该版本通过匿名 window message 监听处理结果，没有暴露销毁函数或 removeEventListener 调用。多次初始化会增加监听。

据此设计最小本地 iframe 包装，沿用服务端 authorizeUrl 的 OAuth2 参数；组件自己以可移除函数管理 message，精确验证 origin、source、state 与回调 URL。这个选择避免复制整个 SDK 或引入拦截全局事件监听的补丁，不新增通用扫码框架。

[官方端外扫码示例](https://github.com/open-dingtalk/h5app-out-login-demo) 的 README 使用 sns 临时授权码流程，仅作为历史参考，不替换本仓库 OAuth2 链路。没有依据搜索中第三方文章决定参数或版本。

尚未验证：真实企业应用是否在当前回调域名下正常展示二维码、扫码消息是否来自预期 iframe、回调及开发代理/生产同域行为。上述为实施第一阶段的技术验证项，不降低“打开即扫”的验收标准；失败时明确报告，不降级成按钮跳转并宣称完成。未读取环境秘密或调用企业接口。

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

## 已收敛依赖与决策

实施顺序见父任务 implement.md：先确认扫码协议在本应用可用，再依次取消 primary、批量加入、公司管理员补齐、扫码主页面和整体验收。协议核验不改授权逻辑。共享 auth/admin/bootstrap 文件按顺序交付。

D5 确认仅初始管理员自动加入；D6 确认逐组织选填角色；D7 确认整批原子保存；D8 确认系统管理及启用公司、只补缺失而不恢复停用。用户已于 2026-10-13 在最新完整方案后回复“开始”，批准实施。

## 工作区说明

2026-10-13 本会话发现原工作区切回 main（8f866832），任务文档仍在 codex/background-task-notify。创建独立工作区 `/home/vince/roncin-go-admin-worktrees/dingtalk-login-memberships` 与分支 `codex/dingtalk-login-memberships`，只 cherry-pick 本任务四次文档提交（e94a2117、1b9d5227、e4a9df0f、ba49449f），未引入后台任务代码。后续在本独立分支继续；不要从旧分支覆盖本次收敛稿。

随后发现 main 新增 cee4bffb：默认公司种子删去 SH/CQ，仅 BJ/CD，sync:dev 改归北京。最终规划同步此基线；不恢复删除的默认种子。子任务中“北京/上海”的角色示例仅说明任意实际公司组合，不是创建上海公司的要求。
