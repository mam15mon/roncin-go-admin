# 财务链路实弹验收收口报告

## 结论与提交

2026-09-28 完成两项子任务：四项存量 PostgreSQL 集成失败清零；当前费用模型在真实服务上的应收、应付、外币 HTTP 与同一订单浏览器闭环通过。浏览器由 Chromium 自动操作，**用户人工走查未执行**。

| 提交 | 内容 |
| --- | --- |
| `57d7b1a0`、`5366a54b` | 四项历史集成用例和历史迁移目标用例校准；只改测试与测试规范，不改生产迁移、数据结构或业务规则 |
| `a16e4dd6` | 提成明细订单时间改为本地可读格式 |
| `ac213e8b` | 当前财务验收脚本、隔离编排、同单 Playwright 用例与实跑证据 |

运行时 R29 的代码 HEAD 为 `a16e4dd6` 加当时未提交的脚本/E2E 工作树；这些实现文件随后收入 `ac213e8b`。R29 后仅修改一次性编排的结束提示文案，使 `--runtime-only` 和 `--stage-b-only` 分别报告真实范围；独立静态复核与 `node --check` 通过，未改业务断言。审阅结果见子任务 `09-27-finance-live-acceptance/research/review.md`。

## 数据库集成与门禁

真实 PostgreSQL 专属库上完整执行 `go -C server test -p 32 -timeout 40m -v ./internal/data -count=1` 及 `./internal/platform/migration -count=1`，迁移包显式开启 `RONCIN_POSTGRES_MIGRATION_TEST=1`。日志 `/tmp/roncin-finance-live-acceptance-r2.log`：

| 包 | 顶层 PASS | 一级子项 PASS | FAIL | SKIP |
| --- | ---: | ---: | ---: | ---: |
| `internal/data` | 291 | 453 | 0 | 0 |
| `internal/platform/migration` | 27 | 9 | 0 | 0 |

原四项工位锚点、两项费用目录及公司边界迁移目标用例全部运行到断言并 PASS，未通过删除、跳过或放宽约束消除失败。额外暴露的四个历史迁移目标时点漂移也已修正，最新迁移链的完整冷启动仍单独验证。详见子任务 `09-27-integration-baseline-four-fixes/research/verification.md`。

最终 `pnpm run check:fast` 退出码 0，日志 `/tmp/roncin-finance-closure-check-fast.log`：Web 184 文件／1089 用例 PASS、1 文件／12 用例既有 SKIP；Server 门禁与漏洞扫描 PASS。快速门禁自身不替代上述真实数据库集成记录。定向组件 `CommissionDetailDrawer.test.tsx` 3 项 PASS；验收脚本语法、Biome、`web tsc`、`git diff --check` 均 PASS。

## 真实业务验收

最终完整运行时编排命令与环境见 `09-27-finance-live-acceptance/research/verification.md`。R29 使用 `--runtime-only` 在新的 CNY 与 USD 随机隔离库中顺序执行正式 SQL 迁移、启动 Go 与 Vite、真实 HTTP 写入及 Chromium 操作；数据库完整测试关联上节 R2 独立记录。R29 日志 `/tmp/roncin-finance-live-acceptance-r29.log` 退出码 0，三类 HTTP 场景与 1 条浏览器用例 PASS，0 FAIL，未跳过其中任一业务场景。

- 应收链：125 CNY 账单、开票、收款、核销与提成；并发幂等、超额拒绝、余额回收、提成来源和调整等断言实跑。独立 7.25 CNY 样本验证草稿账单占用、重复建账／普通修改／删除拒绝、取消释放与同费用重建。
- 应付链：88 CNY 费用→账单→付款→核销，方向不匹配的资金核销被拒绝。
- 外币链：EUR 100 的费用、账单、发票、流水在 USD 公司分别保留 1.10／1.20／1.22／1.25 自然周汇率快照；核销账单本币分摊 48、流水本币分摊 50、汇差 2 USD；毛利 48 USD 的 10% 提成为 4.8 USD。现行提成旧 `cny*` 字段是 1／`BASE_CURRENCY` 恒等快照，本轮**没有验证 USD→CNY 二次折算**。
- 浏览器同单链：订单 `SE2026092800004`，费用 ID `01a0e55e-d38e-7e91-930a-5d70941fbfc6`，账单 `BI2026092800005`，收款 `PR2026092800005`，核销 `WO2026092800004`，提成 `CM2026092800006`。页面实际录费、建账并确认、登记并确认收款、核销、预览生成并查看提成；费用＝账单＝核销 `125.00000000`，收入 125／成本 0／利润 125／提成 `12.50000000`。刷新重读后，账单行、核销分配和提成明细均链接同一订单与费用。`[finance-ui-evidence]` 摘要在 R29 日志第384行。

浏览器成功节点五张截图位于私有目录 `/tmp/roncin-finance-browser-evidence/roncin-finance-browser-TvOneA`（权限 `0700`）；目录内原件不提交。脚本收紧了失败请求判定，关键写入中止、非预期 HTTP 4xx/5xx 与页面错误都会使测试失败。提成明细原始 ISO 订单时间问题已修复，R29 页面时间断言 PASS。

## 环境与限制

验收使用 PostgreSQL `127.0.0.1:5432`、Go HTTP `8010`、gRPC `9010`、Vite `8021`、Chromium；原有 `8001` 服务保持运行。R29 创建的两组库与角色均有清理成功记录。父会话额外创建的 `roncin_verify_45707338` 库、角色和私密环境文件也已删除并确认不存在；没有重建或清空日常开发库。

尚未执行用户人工走查；发票 UI、应付 UI、提成确认／付款 UI 和通用订单输入 E2E 不属于本次浏览器验收范围。HTTP 应收已覆盖开票、提成确认和调整。失败迭代的原始 trace 可能含认证信息，仅保留在仓库外私有目录，不分享或提交；报告中的链路编号和金额可用于复核。
