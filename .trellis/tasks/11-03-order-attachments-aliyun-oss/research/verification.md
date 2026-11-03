# 验证记录

## 云配置（2026-11-03）

- 用户明确要求新建专用 Bucket，已创建 `roncin-order-attachments-a74bdfb8`，地域 `cn-beijing`，私有 ACL、Standard/LRS。
- 凭据仅保存本地 Git 忽略的 `.env.local`（权限 600），不写代码或本记录。
- 专用 Bucket 原始 OSS API 诊断：PUT 200、HEAD 200、GET 原文匹配、DELETE 204；匿名 GET 403；Origin 为正式站点的浏览器 CORS 预检 200。诊断对象使用随机键且已删除。
- 原有 Bucket 设置未修改；先前短暂使用它的诊断对象已删除。本地已切换为新 Bucket。
- 上述仅证明云资源和权限可用，不代替应用签名、登记、权限和后台删除链路验收。

## 应用验证

- 官方 SDK 真实链路：签名 PUT 200、同一授权二次 PUT 409、HEAD/GET 实际 SHA-256 核验、上传人绑定拒绝、签名 GET 下载原文匹配、DELETE 成功。诊断对象已清理，临时诊断程序已移除。
- 实测发现 OSS 拒绝 response-content-type 覆盖（400 InvalidRequest），已移除该下载参数并真实复测通过。
- 专用 PostgreSQL 正式迁移链下 TestOrderAttachmentRepositoryPostgres 最新版本通过，覆盖幂等、锁、范围、共享引用，以及最后解除与旧令牌重登记的确定性并发；企业图片删除相邻测试由实现代理验证通过。
- 前端组件 6 项与文件/哈希 19 项定向验证通过，tsc/Biome/antd lint 通过。
- 真实 Chromium 登录公司工作区、加载海运订单页与附件入口成功；未更改已有订单业务资料。
- 当前开发库存在既有迁移错误：20261027110000_shipping_line_smdg_codes.sql 删除不存在的 shipping_lines_scac_code_check 约束。pnpm dev 因此失败；未修改该迁移或重置数据库，已使用既有 dev:server + dev:web 命令加载新环境并恢复 8000/8001 服务（就绪 HTTP 200）。
- `pnpm run check:server` 完整通过，包含全包 Go 测试、vet、分层、proto lint、可达漏洞扫描。
- `pnpm run build` 前后端生产构建通过。
- `check:fast` 首轮前端 357 个 Suite 通过，3 个未改动的登录/最终港测试在高并发下失败；定向重跑 3 个文件、17 项全部通过。独立 `check:web` 的静态检查通过，全量出现旧列表和附件预览等待超时；附件测试按仓库要求拆分，改为等待真实异步完成。16 Worker 全量执行 360 个 Suite、1979 项通过、1 个 Suite 跳过；唯一失败是测试拆分中间态的组件重导出初始化问题，已修复，最新版两组附件测试 6 项定向通过。未更改全局测试配置或关闭断言，不宣称 `check:fast` 单次全绿。
- 真实浏览器最终验证“订单更多 → 附件档案 → 上传表单”通过，确认旧 Object Key 手填字段已移除。
- 最终变更凭据扫描通过，真实 AK/SK 不存在于任何可提交变更中，`.env.local` 未跟踪。
- 错误码目录使用 `node scripts/generate-error-catalog.mjs` 重新生成并通过 `--check`，同时同步原有目录的既存滞后条目。

最终独立复跑 `AttachmentDrawer.access.test.tsx`：3 项全部通过（8.33 秒）。
