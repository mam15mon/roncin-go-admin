# 代码证据与选择

- 管理入口web/src/pages/admin/index.tsx使用MultiTabCenterTemplate；不存在钉钉配置页，用户明确要求独立Admin页：使用/admin/notification-test，不嵌入现有多页签。
- server/internal/biz/notification.go：7类业务模板、Markdown转义、费用路径匹配；password_reset.go:renderPasswordResetCode为第8类即时模板。
- data/dingtalk.go:SendMarkdown最终追加绝对费用链接、申请企业token并发送sampleMarkdown；预览须共用无网络内容准备，不能只展示少了链接的biz.Text。
- biz.AdminRepo.GetUser已有钉钉身份字段；读取当前UserID，不信任客户端收件人。
- service/order_query.go:orderOrganizationScopesForOperation与biz.OrderUsecase.FindAuthorized为订单列表/读取范围范本。测试不能额外提升系统工作台经营资格；无可读范围明确禁用真实订单模式。
- web/src/access.ts:canAccessPlatform消费权限；新增按钮权限仍从auth/me消费。Manifest新增需同步permissions.generated.ts。
- web当前无Markdown渲染器。react-markdown官方README（2026-10-19读取）：https://github.com/remarkjs/react-markdown ，标准Markdown渲染为React元素，默认不使用dangerouslySetInnerHTML。仅使用基础能力、skipHtml，不启用raw HTML/plugins；实现时按安装版本类型核验。
- 前任务验收记录位于.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-markdown/verification.md，未实际发消息。当前对话与归档已包含决策，无需重复历史检索。
