# 当前实现与规划依据

检查日期：2026-10-09。基于 main 的 `808c6079` 业务代码，仅只读检查；未读取或输出真实人员名单、凭据，未请求本企业钉钉通讯录。

## 代码证据

| 位置 | 当前事实 | 对本任务的影响 |
| --- | --- | --- |
| `server/internal/data/dingtalk.go:20` | 登录用户 token、个人资料、应用 token、unionId 转 userId、单人手机号查询 | 可以复用企业 token 获取机制；需要补通讯录部门和批量人员读取 |
| `server/internal/data/dingtalk.go:175` | 注释明确为按 userId 点查手机号，不做全量同步 | 已有钉钉登录不等于已有全员名单 |
| `server/internal/data/ent/schema/user.go:28` | 用户独立存储，有可空钉钉身份、enabled，没有独立待分配生命周期字段 | 可复用 User，但需设计稳定状态依据和来源资料存储 |
| `server/internal/data/ent/schema/membership.go:13` | Membership 分开连接用户和组织，用户/组织组合唯一 | 无组织 User 在结构上可表达；不能只改数据库而忽略列表、认证假设 |
| `server/internal/data/admin_user.go:21` | 系统管理与公司工作台的可管理组织范围不同 | 新增全员入口不能把公司管理员的范围扩大为全局 |
| `server/internal/data/admin_user.go:81`、`:126` | 行投影和查询都要求当前管理范围内有成员关系 | 无组织待分配查询需显式全局管理路径和无成员关系投影 |
| `server/internal/data/admin_user.go:510` | !enabled 且无有效 Membership 推断为 TERMINATED | 直接导入禁用、无组织用户会与离职混淆，必须调整状态契约 |
| `server/internal/data/auth.go:161` | 钉钉按 unionId 查询，再更新资料、生成凭证 | 同步标识必须与扫码登录一致，避免创建后仍无法匹配 |
| `server/internal/data/auth.go:582`、`server/internal/data/admin_user.go:234` | 扫码更新钉钉姓名和头像；后台另可编辑 display_name 和 email | 引入通讯录同步后必须确定资料维护来源，不能在扫码、本地编辑和同步之间反复覆盖 |
| `server/internal/data/auth.go:180`、`:236`、`:251` | 新注册用户禁用，挂 system 节点成员资格 | 新方案应移除“为了进入审批队列而挂组织”的依赖 |
| `server/internal/data/auth.go:611` | 无默认有效工作台时 credentialForAccount 返回组织权限错误 | 应在业务凭证解析前识别待分配；不能只在登录后加提示 |
| `server/internal/biz/auth.go:758` | 未注册先尝试手机号邀请激活，否则要求注册；禁用凭证返回待授权 | 如统一开通入口，必须同时收敛扫码分支，避免另一入口绕过管理员分配 |
| `server/internal/biz/admin_user_membership.go:44` | 新成员关系复用 validateRolesPrivilege | 开通流程需复用角色工作台归属与提权校验，不能仅校验角色 ID 存在 |
| `server/internal/data/admin_user.go:255` | 离职锁用户，撤销角色、成员资格及会话，保留用户 | 新状态必须与离职、会话和业务历史一致 |
| `server/internal/biz/background_task.go:132` | 已有入队、领取、完成、失败、重试和查询接口 | 优先复用任务能力；具体种类、结果与权限范围需进一步技术设计 |
| `web/src/pages/admin/users.tsx:35`、`:241` | 在职/离职列表与成员/审批/邀请多级导航 | 可将首要工作收敛为待分配和已开通人员管理 |
| `web/src/pages/user/login/dingtalk-callback.tsx:204` | 首扫包含注册与组织分配提示 | 员工扫码页面需与新开通入口统一 |

## 适用规范

- `.trellis/spec/server/backend/dingtalk-registration-approval.md`：记录旧双通道、token 安全与通知；若用户批准替换，需要明确更新契约，不为遵守旧流程保留兼容分支。
- `.trellis/spec/server/backend/auth-session-org-switch.md`：有效成员资格作为工作台准入条件、系统/公司独立、默认工作台与会话轮转。
- `.trellis/spec/server/backend/role-workspace-ownership.md`：角色只归属于公司/系统工作台，部门共享所属工作台角色库。
- 根 `AGENTS.md`：正式 SQL 迁移、统一事务、按实体并发规则、分页上限、拼音搜索、生成物与权限同步、禁止自动清空开发数据。

历史任务 `.trellis/tasks/archive/2026-09/09-12-dingtalk-registration-approval/prd.md` 曾强调分公司自主邀请；其部分邀请与组织树描述已被后续代码和规范调整。用户本轮已明确同意用同步后管理员分配替换邀请与自助注册审批（PRD D1），旧任务不能作为保留这些入口的理由。

## 外部接口依据

- 钉钉官方入口：[获取企业下所有员工信息](https://open.dingtalk.com/document/orgapp/obtains-information-about-all-employees-of-an-enterprise)。页面正文为动态内容，浏览工具未返回正文；不把页面可打开视为接口已实测通过。
- 阿里云官方：[钉钉用户同步](https://help.aliyun.com/zh/qwenwork/enterprise-ultimate-users-user-sync-dingtalk)。已读取正文，确认需要通讯录部门信息、部门成员、成员信息读取权限，资料字段另需对应授权，并受应用授权范围约束。本文描述其产品集成方式，只用于核验钉钉权限及字段条件，不照搬其账号/部门同步策略。
- 接口实现前须继续逐项核对钉钉当前部门、人员分页接口的请求/响应契约，并在本企业授权环境中验证范围及返回字段；本轮未完成该实测。

## 尚待设计的技术点

1. 使用最少的显式状态支持待分配、已开通与离职，不额外建设复杂人事状态机；与现有 enabled 字段的关系需一次确定。
2. 来源姓名、部门、工号及联系方式的字段所有权，以及与扫码资料更新的关系。
3. 单企业配置下 UserID/UnionID 匹配和冲突处理；无身份本地账号的人工绑定需求。
4. 后台同步任务状态与摘要、重复触发和任务重试语义；成功摘要不得掩盖未遍历完的部门。
5. 单人/批量开通的事务边界、锁顺序、并发冲突语义和是否需要版本，按批准的业务语义审阅后确定。
6. 已有待审批记录的准确迁移识别条件；避免误处理真实 system 管理员及企业微信账号。

本轮未修改产品代码，未启动迁移或访问真实通讯录，也未执行功能测试；本轮验证对象为规划文档和任务元数据。
