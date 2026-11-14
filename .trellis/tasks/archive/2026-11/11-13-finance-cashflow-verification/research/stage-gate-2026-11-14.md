# 已提交财务检查点的全量门禁（2026-11-14）

范围：f1e73b9d账户快建、bd6afafa逐组建账/免票/差额核销/费用检索、e5634273核销候选分页。后续发票三层分配、直接修订、建账F与报表不在本轮编译快照中，不能以本记录代替新实现最终验收。

## 前端

`pnpm run check:web` 第二轮通过：460个测试文件、2,499个用例PASS；11个既有`describe.skip`海运新增模板用例SKIP（1个文件），本次未修改或把它们当作通过。权限键、Proto常量、架构/映射/重复扫描、lint/tsc、全Vitest均通过。

首轮角色测试夹具保留退役确认权限、漏新免票权限，以及对冲列表mock漏SearchFilterTemplate已修复；Workbench中间写入快照失败经最终定向与第二轮全量复验消除，没有忽略失败或降低断言。

## 服务端静态

`test:duplicates:go`、`check:layers:go`、`lint:proto`、`go vet ./...`、`check:server:vuln`已通过。免票RPC各自专用消息满足buf规则，API/客户端重新生成。

## PostgreSQL与其他集成

全量`go test -p 32 -timeout 25m -count=1 -json ./...`正在独立数据库执行，明确开启正式迁移与实时验收；只注入本轮专用DATABASE_SOURCE/RONCIN_INTEGRATION_DATABASE_SOURCE及必要标志，未导入整份.env。

首轮使用关键字DSN导致bootstrap-admin、sync-dev和实时验收三项的URL构造/库名前缀检查失败。修正为同一专用库的URL连接后，三个完整包复验全部PASS（含实时验收）；不修改测试规则或生产代码掩盖该验证配置错误。

主全量进程未开启真实OSS因此对应一项SKIP；随后仅提取所需OSS配置独立执行`TestOrderAttachmentOSSLive`，已真实PASS并按测试清理临时对象。没有将SKIP当作OSS验收。

全量数据包已完成并PASS，耗时1121.523秒。首进程3325个子用例PASS、3项验证连接配置FAIL、1项OSS SKIP；后续三个整包188个子用例及真实OSS单项均PASS。按package/test身份合并后，实际覆盖3345个子用例，全部PASS，23个含测试包全部PASS、无未复验失败或跳过。首命令本身退出1，不能把其原始结果写成单次全通过；这是修正验证环境后合并的完整覆盖。原始日志在仓库外，限制当前用户可读；本文件不记录任何连接或凭据。
