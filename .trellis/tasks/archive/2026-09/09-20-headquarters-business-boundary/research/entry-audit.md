# 非常规入口审计

- `server/internal/biz/admin_role.go`：授权提权保护使用仓储返回的原始角色画像，不能改成使用工作台有效权限，避免总部经营权限投影过滤后不能管理角色。初始化管理员治理能力保持原语义。
- `server/internal/service/background_task.go`：重试按当前会话组织取得任务，不接受外部组织 ID；不能将 TaskRequeue 全部视为经营权限。
- 实际 worker 的 ClaimAny 消费者：钉钉审批派发、钉钉通知、对象删除。审批派发关联订单解锁，需要防止总部借重试办理；通知和删除包含治理/补偿能力，不宜全禁。ORDER_REMINDER/INTEGRATION 未找到实际 worker，不为未来能力增加复杂分类框架。
- `server/api/enterprise_resource/v1/enterprise_resource.proto` 的 PrepareEnterpriseResourceImageUpload 属现有企业资源主数据维护，应保留总部维护；订单/客户附件按各自 Register 权限限制。
- `server/api/workbench/v1/workbench.proto` 本人提成提交/重提只要求已登录，须增加公司工作台门禁；只读历史继续原规则。
- `server/internal/server/auth.go` 跨组织读取会复制 Principal 并重定位 Organization.ID，能力投影必须保留真实会话工作台锚点，不能把只读定位目标当作切换组织。
