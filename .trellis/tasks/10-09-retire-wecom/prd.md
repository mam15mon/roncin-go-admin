# 下线企业微信功能

## 目标

退出本系统的企业微信集成：前端不再展示企业微信，后端不再提供企业微信认证、成员授权或用户绑定字段，继续支持钉钉和本地账号。

## 背景与证据

用户在 `/admin?tab=users` 选中「企业微信」表头，明确要求创建 Trellis 任务并下线前后端企业微信功能。

- 列表列：`web/src/pages/admin/components/users/userColumns.tsx:199`；授权表单：`UserFormModal.tsx:333`；待授权判断：`userConstants.ts:27`。
- 登录入口：`web/src/pages/user/login/index.tsx:103`；回调：`wecom-callback.tsx`；路由：`web/config/routes.ts:19`。
- 契约：`server/api/auth/v1/auth.proto:21` 的登录配置/登录 RPC，`server/api/admin/v1/admin.proto:89` 的成员授权 RPC，`:348` 的用户绑定字段。
- 身份提供器：`server/internal/data/wecom.go`；认证、授权和状态判断同时涉及 biz、service、data、Wire 及测试。
- Schema：`server/internal/data/ent/schema/user.go:39` 的绑定字段、`:139` 的唯一索引；检索 Hook 和回填也包含企业微信名称。
- 权限同步：`server/internal/data/permission_manifest_sync.go:31` 已支持清理清单外权限，无需新增机制。

## 需求

1. 移除企业微信列表列、专属授权表单、密码弹窗姓名回退、登录按钮/扫码弹窗、回调路由及白名单。
2. 移除企业微信 RPC、响应字段、领域模型、提供器、用例、仓储、状态 Cookie、注入、权限和配置；同步更新仓库内调用方及生成物。
3. 拟新增正式迁移删除 `wecom_userid`、`wecom_name` 和专属唯一索引，并更新用户检索键；实施前需确认专属绑定信息删除范围。
4. 保留用户记录、用户名/密码、钉钉绑定、启停状态、组织关系、角色及已有审计，不自动生成密码、开通账号或转移组织权限。
5. 更新受影响测试与当前规范；历史迁移及历史前态 SQL 夹具保持原有语义。

## 验收标准

- A1：用户管理与登录页无企业微信界面，前端不再请求企业微信接口。
- A2：服务端不注册企业微信端点；OpenAPI、生成客户端、AdminUser 响应和权限目录不含企业微信能力。
- A3：正式迁移可从已有迁移链升级及空库冷启动；专属字段/索引移除，用户与业务关联保留。
- A4：钉钉同步、待分配/开通、停用、资料只读、本地密码登录、密码设置及组织切换相邻路径验证通过。
- A5：定向检查、生成物一致性和 `pnpm run check:fast` 通过，记录结果并提交 Git。

## 范围外

不修改外部企业微信管理后台、不删除用户或业务记录、不新增历史兼容或自动账号转换、不调整钉钉业务、不清理历史审计、不重建数据库、不推送或部署生产环境。

## 审阅状态

处于 planning。等待用户审阅最终方案并确认企业微信专属绑定字段删除范围；尚未修改产品代码或数据库。
