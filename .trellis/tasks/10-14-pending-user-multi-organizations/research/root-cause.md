# 原因分析与修改边界

## 用户触发场景

在 `/admin?tab=users` 中由待分配人员点击分配并开通，只能选择一个公司/部门/组；已开通人员点击加入其他组织可以多选，造成两处成员资格和角色配置能力不一致。

## 根因链路

1. `DingTalkAssignmentModal.tsx:16` 的 AssignmentValues 只有 organizationId 与共用 roleIds，表单没有多选，角色查询仅绑定一目标。
2. `admin.proto:310` 的单人 AuthorizeDingTalkUserRequest 和 `admin.proto:593` 的 BatchAuthorizeDingTalkUsersRequest 只声明单目标字段。
3. `admin_dingtalk_directory.go:13` 只解析一个 organizationId；`dingtalk_directory.go:210` 循环人员调用单目标用例。
4. `admin_user.go:323` 仓储事务只新增/恢复一个目标成员。若前端简单循环旧请求，首个请求已启用账号，后续请求会冲突；也无法保证整人的组织、角色、状态、通知原子提交。
5. 已开通成员入口已使用 items、逐组织角色行与整批事务。本次复用这一交互和载荷项，保留待分配专用身份与权限边界。

## 受保护的已有工作

开始时 main 已领先 origin/main 20 个提交，登录页3个未提交文件不属本任务。开始实现后另一会话新增登录页扫码测试改动，亦不属于本任务。所有提交只显式暂存本任务路径。

初始文件 sha256（后续核验当前内容变化时须区分其他会话的有效编辑）：
- dingtalk-qr-login.tsx：34aed9fda549f329170aa5d27f9607d2a9b1d543629692db3fdc9ae041ef331f
- index.module.less：38e7e326b4b22c857f70cac5628d57213caf3aefa4a25e4f46b2cf303fc1d030
- index.tsx：e4a90c2def8fb789072a81265d7f2898971cb09dc90f9dec7b7e1550994ae621

## 验证环境

创建本任务专属一次性 PostgreSQL 库和角色，凭据仅保存在 /tmp 下权限 0600 环境文件，不写入仓库。沿用隔离 Schema 和正式 SQL 迁移测试助手，完成后仅清理本次测试资源，不重置或修改业务开发库。
