# 订单附件与阿里云 OSS

订单附件使用阿里云 OSS 私有 Bucket，业务人员在订单附件抽屉选择文件即可上传；文档类型默认「其他」，可下拉选择提单、运单、订舱确认、发票、装箱单、报关单或费用凭证，无需手工输入。完成上传后可以预览图片/PDF、下载及解除引用。单文件上限为 100 MiB。其他格式下载到本地查看，不进行 Office 在线转换。

## 本地配置

在仓库根目录被 Git 忽略的 `.env.local` 中填写：

```dotenv
ORDER_ATTACHMENT_OSS_ENABLED=true
ORDER_ATTACHMENT_OSS_ENDPOINT=https://oss-cn-beijing.aliyuncs.com
ORDER_ATTACHMENT_OSS_REGION=cn-beijing
ORDER_ATTACHMENT_OSS_BUCKET=your-dedicated-private-bucket
ORDER_ATTACHMENT_OSS_ACCESS_KEY=your-access-key-id
ORDER_ATTACHMENT_OSS_SECRET_KEY=your-access-key-secret
```

使用订单附件专用 Bucket。Endpoint 和 Region 必须与 Bucket 所在地域一致；浏览器需要能访问该 HTTPS Endpoint，不能填写仅服务器内网可达的地址。配置关闭时附件上传明确报不可用；启用却缺少必要参数时服务启动失败，不转用本地文件或其他存储。

历史部署曾使用北京地域的专用 Bucket；当前实际 Bucket、地域及凭据须以目标环境配置核对，
本次文档核查未验证云端状态。凭据只通过环境变量注入，不得复制到文档、Git、截图或日志。
本机 `.env.local` 建议权限为 `600`。更改环境变量后需重启开发服务，热编译不会刷新父进程环境。

`OBJECT_STORAGE_*` 是企业资源图片的 S3 配置，和订单附件相互独立，不会因为开启订单 OSS 而自动迁移企业图片。

## Bucket 与跨域

Bucket 访问权限设置为 **私有**，标准存储，并使用从未启用版本控制的专用 Bucket。已开启或暂停版本控制的 Bucket 不满足本期禁止覆盖约束，准备上传时会明确拒绝；不要为该专用 Bucket 开启版本控制。应用用限时签名授权文件访问，不要求打开公共读写。不要对正式附件目录设置固定天数自动删除规则。

浏览器上传/预览需要配置 CORS。允许来源必须填写实际访问前端的完整 Origin，包含协议和端口，不包含路径：

- `https://ops.roncin.com.cn`
- `http://localhost:8001`
- `http://127.0.0.1:8001`
- 开发机局域网入口按实际地址添加，不能沿用历史机器 IP

允许方法 `PUT`、`GET`、`HEAD`，允许请求头 `*`，暴露响应头 `ETag`、`Content-Disposition`、`x-oss-request-id`，缓存时间可设为 600 秒。前端入口变化时同步更新来源，生产环境可删除不再使用的开发来源。跨域允许来源不等于授予读写权限，访问仍需有效签名。

不要把应用 Cookie 或后端认证头附加到 OSS 直传请求；请求头应严格使用后端签名时返回的值。

## 访问凭据权限

运行应用的 RAM 凭据需要专用 Bucket 的版本状态读取权限及文件操作权限；创建 Bucket、修改跨域等配置权限只用于初始化。可用如下自定义策略限制到订单对象目录（将示例 Bucket 替换成实际名称）：

```json
{
  "Version": "1",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["oss:GetBucketVersioning"],
    "Resource": ["acs:oss:*:*:your-dedicated-private-bucket"]
  }, {
    "Effect": "Allow",
    "Action": ["oss:PutObject", "oss:PutObjectAcl", "oss:GetObject", "oss:DeleteObject"],
    "Resource": ["acs:oss:*:*:your-dedicated-private-bucket/orders/*"]
  }]
}
```

应用不需要列举账号下其他 Bucket。HEAD 核验使用对象读取权限。授权只在服务端使用，浏览器只收到短期链接与本次上传授权。

## 业务行为与排查

- 上传完成不代表已登记。服务端核验云端对象后才写入订单附件；最终写入仍检查订单业务锁。
- 同名文件采用不同对象键，不能覆盖现有附件；重新上传视为新附件。
- 下载链接短期有效，过期后重新点击下载；不要将带签名的完整链接粘贴到日志或任务文档。
- 拆票可共享物理文件。解除一张订单的引用不会删除其他订单仍引用的文件；最后一条引用解除后由后台任务执行 OSS 删除，失败按现有任务机制重试。
- CORS 错误检查来源地址、请求方法和签名返回的请求头；签名错误检查地域、Endpoint、系统时间与请求头，不通过关闭签名或改公共读写排障。
- 浏览器中断、登记被订单锁阻断等情况可能留下已上传但未登记的孤立对象。本期没有自动清理扫描；人工处理前须核对对象键在 `order_attachment_assets` 中没有引用，并确认不处于正在上传/登记过程。不得整目录清空或设置会删除正式附件的生命周期规则。

## 验证

自动化验证覆盖授权绑定、对象属性校验、权限边界、订单锁、幂等和共享引用删除。真实环境还应走一次“准备上传 → 浏览器 PUT → 登记 → 预览/下载 → 解除最后引用 → 后台 DELETE”，检查下载内容与原文件一致，诊断文件使用独立随机对象键并及时清理。

官方依据：[签名上传](https://www.alibabacloud.com/help/en/oss/developer-reference/v2-presign-upload)、[签名下载](https://www.alibabacloud.com/help/en/oss/developer-reference/v2-presign-download)、[创建私有 Bucket](https://www.alibabacloud.com/help/zh/oss/developer-reference/putbucket)。

可选的真实云端适配验证命令（只生成随机诊断对象并清理，不写业务订单）：

```bash
node --env-file-if-exists=.env.local --input-type=module -e 'import { spawn } from "node:child_process"; const p = spawn("go", ["-C", "server", "test", "./internal/data", "-run", "^TestOrderAttachmentOSSLive$", "-count=1", "-v"], { stdio: "inherit", env: { ...process.env, RONCIN_OSS_LIVE_TEST: "1" } }); p.on("exit", code => { process.exitCode = code ?? 1; });'
```

默认单元测试不会连接云端或消耗真实存储；只有显式设置 `RONCIN_OSS_LIVE_TEST=1` 才启用上述验证。
