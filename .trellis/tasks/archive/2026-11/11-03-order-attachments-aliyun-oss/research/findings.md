# 调研记录

## 仓库证据
- `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx:145` 当前调用登记接口，165 行明确仅登记元数据。
- `server/api/order/v1/order_attachment.proto:12` 只有列表、登记和解除引用；权限分别为 attachment.read / attachment.register。
- `server/internal/biz/order_attachment.go:13` 现行单文件上限 100 MiB。
- `server/internal/data/order_attachment.go` 写入与解除引用均在 WithTx 中锁订单、检查业务可编辑；资产可被拆票后多张订单引用；最后引用解除才入队删除。
- `server/internal/biz/object_deletion.go` 当前删除消费者只注入 EnterpriseImageStorage；其实现限制 enterprise-resources/ 前缀，不能删除订单文件。
- `server/internal/conf/v1/conf.proto` 是配置源文件；企业图片已有 object_storage 配置及 S3 实现，本次以订单 OSS 独立配置避免改变其行为。

## 官方依据
- 阿里云 Go SDK V2：https://github.com/aliyun/alibabacloud-oss-go-sdk-v2
- PUT 签名上传：https://www.alibabacloud.com/help/en/oss/developer-reference/v2-presign-upload
- GET 签名下载：https://www.alibabacloud.com/help/en/oss/developer-reference/v2-presign-download
- 预签名 PUT 在有效期内可重复使用，因此必须约束唯一对象键和禁止覆盖，不能将签名过期作为防覆盖手段。
- OSS 浏览器直传需要配置 CORS，上传的签名请求头必须与服务端签发值一致。

## 明确限制
- 用户随后提供凭据并要求新建专用 Bucket，专用云资源和本地环境已配置；完整应用链路仍须生成代码后联调。
- 本期不增加未登记对象扫描清理平台；直传成功但登记失败可能留下孤立对象，说明中记录人工排查方式，不能设置会误删正式附件的全目录过期规则。
