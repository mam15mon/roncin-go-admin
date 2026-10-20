# 取消主要组织技术设计

## 契约与存储

删除 Membership.primary 的 Ent 字段、领域 Primary、Credential.PrimaryOrganizationID 及自动 normalizePrimaryMembership 逻辑。新增迁移只删该字段及其依赖，保留用户、组织、成员、角色与业务关联。历史迁移不重写。

Proto 删除 AdminUserOrganizationSummary.primary、AdminUserMembership.primary、Create/UpdateUserMembershipRequest.primary 和 OrganizationChoice.is_default，原编号与名称 reserved。UpdateUserRequest.role_ids 同样移除并 reserved，领域及仓储 UpdateUser 不再携带角色参数。CreateUser 明确初始组织的角色不在删除范围。

## 登录初始会话

密码和钉钉均从现有有效成员关系得到工作台集合，按组织名称、ID 稳定排序。无候选返回现有禁止进入错误；一个候选直接进入；多个候选内部以第一项建立会话，前端必须展示选择页。显式密码 organization_id 按候选严格校验，非法值不回退。

该内部顺序不落库、不暴露为偏好。principal 的 currentOrganization 仍是当次实际会话，前端判断“选中的是否为当前会话”以此 ID 为准，禁止继续拿删除的 is_default 代替。两种登录候选排列使用同一服务端排序口径；钉钉 principal.organizations 的构造也应一致。

复用现有 SwitchOrganization 原子轮转，不引入预登录票据；不修改其他设备会话。选择页去掉默认标签和突出样式，不自动提交选择。

## 用户资料与角色

编辑用户基础资料只提交允许的资料字段和现有账号状态意图，保持钉钉资料只读、离职/开通专用规则。角色编辑统一在明确的 membershipId 弹窗内完成。移除 UserFormModal 的 anchor role query、roleIds 回填与隐式角色选择。

新建用户时仍允许给已知目标组织选择角色；不能因为共用 UserFormModal 就把创建功能误删。调用方一次性同步 UpdateUser 参数；伪造被移除角色参数不能触发角色修改。

列表摘要保持可读但无“主要”标记：组织名称稳定排序；若暂需一条成员作展示摘要，以启用态、创建时间、ID 确定性选择，函数只负责显示，不得用于 UpdateUser 写角色。完整组织角色仍在成员表查看。

## 影响文件

server/api/{auth,admin}/v1、internal/{biz,data,service}/auth* 与 admin_user*、ent/schema/membership.go、cmd/bootstrap-admin、所有 SetPrimary 调用的种子和夹具；web/pages/user/login/**、admin/components/users/{UserFormModal,UserMembershipModal,userConstants,userColumns} 及 users 调用方。

## 生成、迁移、风险

先 Proto/Ent 源再生成 Go/OpenAPI/Web；迁移从 Schema 差异产生并审阅，仅一次性删除废弃字段。验证空库完整链与前态升级两种情况；不可调用 Schema.Create。以类型检查找到调用方仍引用 roleIds/isDefault 的路径，结合语义搜索确认 primary 剩余仅是按钮样式或历史迁移等无关项。

现有 auth-session-org-switch 的 primary 条款被新设计替代，其他权限/范围/会话约定保留。bootstrap 自动加入公司由后续子任务实现，本子任务只移除它的 primary 写入，保持每组变更可验证。
