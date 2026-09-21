# 权限范围一致性独立检查

## 结论
未发现本次修改中需要修复的阻断问题；未修改实现文件。检查以当前代码路径和定向验证为依据，最终全量门禁由主会话执行。

## 核对范围
- `Principal.PermissionCapabilities` 仅聚合同权限有效 grant，最高范围不借用无关角色；`PermissionKeys` 同源，bootstrap 仅对已持有的有效权限提升阈值。
- `HasPermission`、`HasPermissionInScope`、组织范围解析均排除 self 和未知范围。总部经营权限过滤保留，真实对象写权限继续受当前公司限制。
- Create/UpdateRole 明确拒绝 self；管理员角色及钉钉审批授权画像不再借用停用 self 的 administrator 身份；历史数据库枚举和值保留，未引入数据迁移或自动升级。
- 直接查询角色权限的通知收件人已过滤有效范围；订单 lock 资格谓词原有三种有效范围白名单，不会包含 self。
- Proto、Go、OpenAPI、前端生成类型含对应 permissionCapabilities 字段，service 投影返回该字段；前端 access 无旧字段回退，开发模拟用户同步。
- 生产前端 permissions/roleScopes 剩余使用为角色展示和权限计数，不参与按钮授权。
- 详情和费用页先按具体 kind 门控，缺权限传递 undefined 抑制 hooks 请求；lock-only 分支仅挂载补录审批组件，审批操作消费响应 canApprove 等能力，不请求普通费用选项和锁状态。
- 本人工作台、本人提成 API 仍使用 AUTHENTICATED，原本人过滤链未修改。
- 后端与前端规范已记录能力字段与 self 停用。

## 验证
- `pnpm --dir web tsc`：通过。
- 7 个修改实现文件 Biome check：通过，无修复。
- `go -C server test ./internal/biz ./internal/server ./internal/service -run 'Test(Principal|DisabledSelf|NormalizeRole|.*Privilege)' -count=1`：三个包通过。
- 定向 Vitest：access、access.workspace、fees、detail-change-actions、RoleFormModal，5 个文件 50 个用例通过，15.91 秒，无 stderr 警告。
- `git diff --check`：通过。
- 未运行真实 PostgreSQL 集成，不将默认单测视为数据库验证；未重复主会话 check:fast。

## 交付说明需保留的边界
- self 授权全量停用，包括来自该角色的 platform.access。若账号唯一的平台访问权限来自 self，工作台路由也会关闭；本人 API 的业务规则保留，不等于停用角色仍获得平台访问。管理员必须显式配置有效角色，不能自动扩大范围。
- 未查询现存角色或用户数据，不能据此宣称现有账号无需调整。
