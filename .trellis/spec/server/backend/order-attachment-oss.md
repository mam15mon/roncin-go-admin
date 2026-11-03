# 订单附件阿里云 OSS 契约

## 1. 范围与触发

适用于订单附件的文件上传、访问、解除引用和后台删除；企业资源图片继续使用独立的 S3 存储。涉及接口或存储变更时同时检查前端抽屉、订单锁、拆票共享资产及对象删除消费者。

## 2. 接口签名

契约唯一源为 `server/api/order/v1/order_attachment.proto`：

- `PrepareAttachmentUpload`：`POST /api/v1/orders/{order_id}/attachments/upload`。
- `RegisterAttachment`：`POST /api/v1/orders/{order_id}/attachments`。
- `GetAttachmentAccess`：`GET /api/v1/orders/{order_id}/attachments/{id}/access`，`download` 区分下载与预览。
- `ListAttachments`、`RemoveAttachmentReference` 沿用订单列表/解除引用路径。

读接口使用 `attachment.read`，准备上传、登记、解除引用使用 `attachment.register`，由现有订单范围授权，不另建权限真相。

## 3. 请求、响应与配置

准备上传请求包含 `docType/fileName/mimeType/fileSize/checksum`，校验和为 SHA-256 十六进制；响应 `data` 返回 `uploadUrl/uploadToken/headers/expiresAt`。浏览器 PUT 只使用签发 URL 与返回的请求头，不附加应用 Cookie。登记只接受 `uploadToken`，不接收任意外部对象键。访问接口返回 `data.url/expiresAt`。

上传授权必须绑定组织、订单、上传人及本次文件全部元数据，且有过期时间和服务端完整性保护。对象前缀为 `orders/{组织ID}/{订单ID}/{随机对象ID}`；签名有效期内仍须禁止覆盖。登记前 HEAD 核对对象属性、GET 流式核对实际 SHA-256，最终写入重新检查业务门禁。准备上传读取 Bucket 版本控制状态，仅允许从未启用版本控制的 Bucket，已开启/暂停状态均拒绝，以确保禁止覆盖头生效。

配置源 `server/internal/conf/v1/conf.proto`，环境变量：`ORDER_ATTACHMENT_OSS_ENABLED/ENDPOINT/REGION/BUCKET/ACCESS_KEY/SECRET_KEY`。关闭时不回退；开启但缺项时启动失败。真实凭据只存在环境或被忽略的本地配置；签名 URL、令牌和密钥不得记入日志或文档。配置步骤见 `docs/order-attachment-oss.md`。

## 4. 校验与错误矩阵

| 场景 | 要求 |
| --- | --- |
| 空文件、超过 100 MiB、非法名称/类型/校验和 | 拒绝准备上传，不产生附件记录 |
| 对象不存在、大小/类型/元数据不匹配 | 拒绝登记 |
| 无权限、跨组织、跨订单或不同上传人冒用授权 | 拒绝授权和登记 |
| 授权过期、篡改令牌 | 拒绝登记 |
| 准备后订单锁定或状态不可编辑 | 最终事务内拒绝登记 |
| 同一次上传重复登记 | 内容相同返回已有记录，不生成重复资产；内容冲突明确失败 |
| 已解除最后引用的旧授权再次登记 | 资产 ForUpdate 查询不存在后检查 ObjectStorageDeletion，已有删除任务则拒绝；检查必须在锁查询后，覆盖跨票最后解除的并发窗口 |
| 未知对象前缀或所需存储关闭 | 删除明确失败并按任务机制重试，不转用其他存储 |

## 5. 正常、基础与错误案例

- 正常：选择 PDF → 准备 → PUT → 登记 → 获取授权 → 下载原内容 → 解除最后引用 → 后台删除。
- 基础：订单锁定时仍可按读权限下载；共享文件解除一票引用后另一票仍能访问。
- 错误：把 A 订单的令牌发到 B 订单、把对象键当登记参数、复用签名覆盖已完成的对象，均应失败。
- 未登记孤立对象本期只记录人工排查方式，不设覆盖正式文件的自动过期规则。

## 6. 必要验证与断言

- 用例测试：上传元数据边界、令牌范围/期限/完整性、重复登记和业务锁最终校验。
- OSS 适配测试：签名 PUT 及禁止覆盖头、HEAD 核验、GET 下载头、对象前缀和删除。
- 数据层：资产与引用幂等、共享引用最后删除、企业图片删除路由相邻路径。
- 前端：成功/失败、大小边界、上下文切换、预览对象 URL 释放；浏览器非安全 HTTP 开发入口仍可计算校验和。
- 真实云端：新私有 Bucket 下验证 PUT/HEAD/GET/DELETE、匿名读取拒绝和 CORS，诊断文件及时清理；无云配置时不得将替身测试描述为云端验收。

## 7. 错误与正确实现

错误：直接登记客户端填写的 `objectKey`，或者在最后引用解除后调用只能处理企业图片前缀的删除器。

正确：通过有完整性保护且绑定订单的上传授权登记，OSS 核验后在已有短事务中写入；最后引用解除在事务内登记删除任务，后台按明确前缀选择对应存储。云端网络调用不进入数据库事务，不新增分散的手写事务模板。
