# 技术设计

## 入口与权限

在web/src/pages/admin/notification-test.tsx实现独立Admin页面，集中式路由新增/admin/notification-test；不嵌入admin/index.tsx多页签中心。提供有权限者可发现的独立Admin入口，沿用PageContainer、PageHeaderShell、SectionCard等公共组件。新权限system.notification.preview和system.notification.test_send；预览依赖PlatformAccess，发送依赖预览。Manifest唯一真相，Proto标记并重新生成权限键/access规则；默认administrator通过正式权限同步取得。按权限开放系统管理入口，不通过角色名称硬编码。

## 分层与数据流

Admin Proto新增ListNotificationTestTypes（GET /api/v1/admin/notification-tests/types）、PreviewNotificationTest（GET /api/v1/admin/notification-tests/preview）、SendNotificationTest（POST /api/v1/admin/notification-tests/send），响应遵循独立类型与统一信封。仅接受受控类型及可选order_id，不接受正文、标题、验证码、收件人、任意URL或目标组织。

service只解析DTO并调用独立biz通知测试用例；用例构造示例NotificationDelivery复用renderNotification，以及固定无效验证码复用renderPasswordResetCode，最后统一加测试标识。用户绑定从现有认证/用户仓储读取，不能相信请求中的接收人。必要时新增最薄只读投影，Ent留data。

订单候选复用已有ListOrders关键字分页，不循环全量加载；服务端预览与发送对可选ID重新按当前Principal具体订单读取权限解析范围，用OrderUsecase.FindAuthorized校验，只投影编号和路径。不支持显式公司参数，也不在系统工作台扩大经营权限。没有可读资格时仍可做全部示例测试。

现有data.SendMarkdown负责将固定OrderEntryPath拼接至配置回调的scheme/host并追加Markdown链接。为保证预览与发送一致，将这段无网络内容准备提成同一内部函数/最薄接口供两条路径复用，避免在前端拼站点或复制模板；发送仍走既有企业机器人端点。预览绝不申请token或发送；集成禁用可展示正文并明确不能发送。站点配置无效时不得伪造链接。

不建立测试通知表或事务；同步HTTP发送，前端禁止在途重复操作，不做自动重试。接口受理与最终送达分别表述；运行日志不记完整载荷或验证码。

## 前端

请求使用OpenAPI生成客户端和React Query，query key包含类型/订单，切换时隔离迟到结果；mutation不自动重试。已确认web无Markdown渲染依赖，使用react-markdown安全渲染，禁用raw HTML，链接仅接受后端生成的受控本站费用目标；不引入编辑器、GFM或HTML插件。只支持当前模板实际使用的标准Markdown。可折叠查看源码；新测试辅助模块放web/tests避免生产扫描。

## 契约与发布

顺序：修改Proto/Manifest→make api→generate:web-client及generate:permission-keys→实现和验证，必要Wire通过生成更新。无数据库Schema迁移；发布使用现有migrate同步权限。新依赖通过pnpm加入并提交lock。回滚以整组功能提交为单位，不新增兼容分支。

## 风险与验证

重点验证收件人强制绑定、越权订单、无业务写入、无效验证码、预览与实际载荷一致及前端在途操作。HTTP替身用于受理/失败验证；实际钉钉排版与免登由用户手动发给自己验收。自动化阶段不发真实消息。
