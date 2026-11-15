# 发票登记易用性最终复核与验收

日期：2026-11-15。用户要求完成现有任务并合入 main，本轮覆盖 R1–R8 全范围；业务功能提交 `d13385b6`、`e468813e`、`4ad589ed`、`69f1ff0e` 已在 main。

## 结论与本轮补齐

- 源码复核未发现剩余产品阻断。自动带入仅改变表单；手改保护、逐费建议税额、三层校验、修订前后对照、原票快照、结算阻断与现有事务仍完整。
- 修正 PRD/design 中初期“纯前端、不改 Schema”的陈旧范围描述；R8 的六票种及正式迁移属于追加批准范围，非额外扩展。
- 新增 `server/internal/data/finance_invoice_type_integration_test.go`，验证六类票种实际登记/修订落库、三类专票缺开户资料拒绝，以及正式票种迁移对原票、默认抬头和修订前后快照的精确映射与经济事实保真；数据库仍拒绝旧票种。
- 同步修复财务浏览器验收脚本：使用新票种，按打开即自动带入的表单验证唯一抬头、余额和分配，不再重复添加发票及分配；更新税额字段定位，实际登记纸质普票并验证响应票种。

## 定向验证

- 前端：`vitest run src/features/finance/invoice-registration/ src/pages/finance/bills/components/BillInvoiceRegistrationSection.test.tsx src/features/finance/bill-creation/`，24 文件、103 用例通过。
- 修改范围 Biome：发票共享能力及财务 e2e 共 22 文件通过；`pnpm --dir web tsc` 通过；data `go vet` 通过；`git diff --check` 通过。
- Go 原有发票定向测试：biz 5、service 3、data 14 个顶层用例通过；data 使用自行启动的 PostgreSQL 18 独立 UTF8 库与随机 Schema 执行正式 SQL 迁移，无 SKIP。覆盖部分登记、幂等、并发、修订历史、读快照、余额、免票、终态操作及旧发票登记迁移。
- 新票种两项真实 PG 顶层测试通过，其中六类登记/修订分别执行；增强历史普通/专用精确映射断言后单独复验正式迁移再次通过。
- 首次临时集群按 `--no-locale` 初始化时默认 SQL_ASCII，迁移因中文编码失败；改用独立 UTF8 库后上述原有测试真实通过。新增测试首次夹具存在费用码过长和重复键，修正后真实复验通过；没有修改产品约束来绕过失败。

## 本轮真实浏览器与 HTTP

运行现有 `run-acceptance-finance-disposable.mjs --runtime-only`，使用第二个自行启动的独立 UTF8 PostgreSQL、随机隔离库与角色、Go 8010/9010 和 Vite 8021；不使用或写入日常开发数据库。

- CNY 真实海运浏览器链路通过：1 个 Playwright 用例，约 1.4 分钟。涵盖录费、建账、审核锁、收款核销、提成、发票默认余额/分配、唯一抬头、手填税额、实际纸质普票登记、持久化重读、四报表桌面/窄屏、CSV及 Go 同域资源入口。
- CNY 应收/应付 HTTP 与 USD 外币财务连续 HTTP 链路通过。
- 本轮浏览器实际保存的是纸质普票，不将其表述为六种票种均经浏览器保存；六种票种均经真实 PG 登记/修订，选项与中文展示另由前端定向测试覆盖。
- `--runtime-only` 不运行编排内数据库全量门禁，数据库证据为上面的独立 PG 定向测试，全栈证据为主会话完整门禁；不混称编排执行了全部 PG 集成测试。

原始私有日志在仓库外 `/tmp/trellis-invoice-browser.log`，浏览器原始截图位于 `/tmp/roncin-finance-browser-zNSmsJ`；不提交原始 trace 或认证数据。

## 全栈门禁与资源清理

主会话本轮 `pnpm run check:fast` 完整通过：前端 2572 用例通过，11 个既有跳过；后端全包测试、Go vet 与漏洞扫描通过。新增票种真实 PG 测试另按上文补充，未将普通门禁的数据库 SKIP 当作真实 PG 通过。

编排退出码 0，临时 CNY/USD 数据库与角色已由编排核查删除；独立集成库的隔离 Schema 剩余 0，浏览器集群的 `roncin_acc_fin_*` 数据库/角色均剩余 0。两套本轮自行创建的 PostgreSQL 集群随后正常停止；保留仓库外日志用于核对。未重置、清空或改动日常开发数据。
