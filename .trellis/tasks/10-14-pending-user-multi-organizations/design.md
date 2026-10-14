# 待分配多组织开通设计

## 边界与根因

已开通的 UserMembershipModal 支持 organizationIds 与 rolesByOrganization。待分配的 DingTalkAssignmentModal、单人与批量 AuthorizeDingTalkUser 请求、biz 方法和 data 仓储均只有一个目标组织，因此仅修改 Select 无法完成多组织开通。

修改范围为 admin Proto 及生成物、service/biz/data 开通链路和对应测试、admin 用户页面的角色字段复用及开通表单。保留权限码、账号身份检查、停用人员明确重新开通规则、通知模板和已有成员编辑行为。无 Schema/迁移改动，不触碰登录页用户改动。现有 CI 有 PostgreSQL 服务但未向默认测试注入专用变量，本任务在该服务下补定向开通事务步骤，确保真实执行而非 SKIP。

## 契约与分层

单人 AuthorizeDingTalkUserRequest 与批量 BatchAuthorizeDingTalkUsersRequest 都使用 repeated UserMembershipCreation items，每项 organization_id、role_ids。复用现有成员新增项消息和纯 Go AdminUserMembershipCreation，旧单组织字段 reserved，不兼容双读。RPC/HTTP 路径与权限不变。

service 统一转换 items：非空、最多 200、合法非零 UUID、组织不重复；角色 UUID 按既有规则去重。biz 复核结构、所选组织整体至少一个角色，保持系统工作台限制。批量用户人数上限仍 200，逐人调用完整多组织开通仓储，收集既有成功/失败结果。

## 事务与并发

一个人的全部目标成员、角色、账号开通状态、审计及一条 USER_AUTHORIZED 通知在现有 WithinTransaction + WithTx 中原子提交。不得循环现有单组织开通事务；不得在启用账户后补写其他组织。

保持用户（操作者和目标按 ID 排序 ForUpdate）→组织（现有全树主键顺序 ForShare）→成员（按 ID ForUpdate）→角色（全部目标和操作者角色按 ID ForShare）的锁序。统一锁入全部目标角色，然后在同一共享事务重建操作者 Principal，逐个组织检查有效公司祖先、角色归属和提权。全部验证通过后再执行写入。

重新开通沿用清除该人的旧角色、停用旧关系、恢复/新增明确选定关系。任何失败回滚全部旧关系变化和新写入。数据库唯一性错误按现有统一映射。事务提交后普通上下文重读用户响应。

通知每人一条，业务内容只表示账号已开通；后台任务组织归属使用所选第一项所属公司，不据此定义主要组织或登录默认组织，登录已有候选规则不变。

## 前端

提取本页面领域内 OrganizationRolesField，已开通与待分配都复用；React Query key 包含 organizationId，角色候选请求互不串用。待分配多选后每个组织一行角色，选填，但整体没有角色时明确提示而不调用接口。

去掉组织只清除该组织角色，保留其余输入；重新选回空角色。保留 ModalForm.onFinish Promise、actionRef.reload、人员独立结果和只重试失败者。确认响应完整后再报告成功；响应中断/畸形需刷新核实，不能盲目重试可能已成功者。

## 风险与回滚

风险在跨公司角色隔离、返聘时原有成员全撤销/恢复原子性、锁序、人员独立结果，以及所有单组织调用方同步。以真实 PostgreSQL 正式迁移隔离 Schema 测试验证事务和并发；只使用临时测试数据库/Schema，不重置开发库。回滚以 revert 本任务提交整体回退契约与实现，不引入历史兼容。
