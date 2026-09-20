# 验证记录

## 开发数据只读核验

2026-09-20 通过本地环境变量向 psql 发起只读连接，设置 default_transaction_read_only=on；连接未成功。未输出连接串或错误中的凭据，未执行数据修改。总部存量业务归属尚未核验，不宣称数据已清理或迁移。

2026-09-20（补充核验，已成功）改用与集成测试一致的本地凭据连接开发库 `roncin_go_admin`，全程 `default_transaction_read_only=on`，仅执行 SELECT/统计：

- 41 张经营表（partners、orders、订单子对象、sea_* 单证、finance_* 单据与提成）总部归属行数全部为 0。
- `partners`、`orders` 归属组织 `kind <> 'company'` 行数为 0。
- `order_personnels` / `partner_assignments` 与根对象组织不一致行数为 0。

结论：开发库不存在总部存量经营数据，无需处置；未做任何写入、改挂或清理。

## 实施验证

2026-09-20 定向验证（本会话补跑）：

- `go -C server build ./...` 通过。
- `go -C server test -p 32 ./internal/access ./internal/biz ./internal/service ./internal/server` 全部 ok。
- `go -C server test -p 32 ./internal/data` ok（13.9s）。
- 提成申请 Postgres 集成测试以专用集成库 `roncin_go_admin_integration` 真跑：`TestCommissionApplication*` 全部 PASS（26.7s，非 skip）；订单锁/补录事务集成测试同库真跑 PASS。未触碰开发库数据。
- 前端 `pnpm tsc` 通过；`pnpm test:changed` 38 文件通过 / 1 跳过，281 用例通过 / 12 跳过（跳过项为环境依赖用例，未计入通过）。
- Biome：改动文件退出码 0；`split.tsx`、`SeaDocumentSection.tsx` 存量 20 个 noUnusedImports warning，经与 HEAD 版本比对确认为先前重构遗留、非本任务引入，未顺带清理。
- 生成物同步：`generate:permission-keys`（304 键）、`buf generate` + `go generate ./internal/server`（两次运行 diff 哈希一致）、`generate:web-client` 均无新增差异，生成物与源同步。

最终门禁 `pnpm run check:fast` 结果见下方。

## 最终门禁

2026-09-20 `pnpm run check:fast` 通过（退出码 0）：WEB 门禁 47.0s（permission-keys/proto-constants/biome/tsc/全量 vitest），SERVER 门禁 115.0s（buf lint、`go test -p 32 ./...`、`go vet ./...`、漏洞扫描，豁免仅已审计的 Excelize GO-2026-6452）。`git diff --check` 无空白错误。
