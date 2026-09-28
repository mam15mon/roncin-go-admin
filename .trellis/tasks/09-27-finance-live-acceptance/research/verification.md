# 财务链路真实验收记录

## 结论与运行版本

2026-09-28 完成真实 PostgreSQL、Go HTTP、Vite 和 Chromium 浏览器验收。最终运行 R29 的应收、应付、同一订单 UI 与 USD 外币链路全部 PASS，进程退出码 0；本轮为显式 runtime-only，数据库完整门禁关联此前 R2 的完整两包结果，不能把 R29 描述成重复执行数据库门禁。

- 运行 HEAD：`a16e4dd6630344ae9cff3ad9ec8b43d05a0a4fb1`；验收脚本与 E2E 为本任务未提交工作区代码，文件指纹见下。
- 最终脚本/用例 SHA256：bill-batch `b567119b8f763f810e4bc5e0b0e7eadba012f393fa4bf1a056a2fb16091a74a2`、payable `562c3b7e0b705dfe389768881b23c71eaf793b41b436a79e60fbe315ba382f07`、foreign `ea62bb173ad7b83675b6d7d2a3b5d3764f59afe9873a307d6f09e75e8dfcadba`、fixture `f7cf5c784ed944c2921a9e1352abc9cb98e4ef88cbf0d439c1f3509ae3813707`、财务 E2E `ddadbf93e3dc2b1f1d0e1c63e2e5578420b7778c6204d54ffae6d09213aadbfd`。runner 后续只有成功文案改动，按最终提交内容复核指纹。
- 关联集成修复提交：`57d7b1a0`、`5366a54b`；真实 UI 暴露的原始 UTC 订单时间由父任务修复并提交为 `a16e4dd6`。
- 最终完整运行日志：`/tmp/roncin-finance-live-acceptance-r29.log`。
- 分阶段成功证据：R25 的 CNY HTTP/UI PASS（UI 1 passed，39.9s）；R27 的 USD PASS。R29 在最终代码上再次串联 A+B 全部通过（UI 1 passed，40.4s）。
- 人工走查：未执行。这里是实际 Chromium 页面操作与响应校验；不能表述为用户人工点过。
- 通用 E2E 的另外两条订单输入用例：不属于本任务，未纳入最终财务验收；保留通用 `web test:e2e`，财务入口只运行财务文件。

## 环境与实际命令

Ubuntu，PostgreSQL 127.0.0.1:5432；正式 SQL 迁移初始化；Go HTTP 8010 / gRPC 9010；Vite 8021（strictPort，保留已有 8001 服务）；Chromium 151.0.7922.34；浏览器时区 Asia/Shanghai。管理员凭据与有 CREATEDB/CREATEROLE 的临时管理员连接串只从私密环境变量加载，没有写入报告或 Git。

最终命令：

```bash
node --env-file=.env.local --env-file=/tmp/roncin_verify_45707338.env -e 'process.env.RONCIN_ACCEPTANCE_WEB_PORT="8021"; process.env.RONCIN_ACCEPTANCE_ARTIFACT_DIR="/tmp/roncin-finance-browser-evidence"; process.argv.push("--runtime-only"); import("./scripts/run-acceptance-finance-disposable.mjs")'
```

默认 `pnpm run acceptance:finance:disposable` 仍依次执行完整数据库门禁、CNY 应收/应付/UI、USD 外币。新增 `--runtime-only` 仅显式复用独立完整门禁证据；`--stage-b-only` 仅显式执行独立 USD 场景。所有执行失败都保持非零退出，不自动跳过或回退。

公司经营夹具通过正式组织、成员资格、角色和切换接口准备。全局船公司/计费单位在系统工作台准备后返回公司工作台。订单创建携带 SALES/OPERATION/CUSTOMER_SERVICE 三岗快照；提成方案按员工和身份生效。无 --apply 的应收/应付入口不创建夹具或经营数据。

## 数据库完整门禁 R2

实际顺序执行（同一任务专属隔离库，两个包各自内部串行迁移初始化）：

```bash
go -C server test -p 32 -timeout 40m -v ./internal/data -count=1
go -C server test -p 32 -timeout 40m -v ./internal/platform/migration -count=1
```

两个命令均注入任务专属 `DATABASE_SOURCE`、`RONCIN_INTEGRATION_DATABASE_SOURCE`；migration 显式启用 `RONCIN_POSTGRES_MIGRATION_TEST=1`。日志为 `/tmp/roncin-finance-live-acceptance-r2.log`。实际使用完整 -v 日志解析统计，没有用测试名后缀过滤。

| 包 | 顶层 PASS | 一级子项 PASS | FAIL | SKIP | 包耗时 |
| --- | ---: | ---: | ---: | ---: | ---: |
| internal/data | 291 | 453 | 0 | 0 | 401.146s |
| internal/platform/migration | 27 | 9 | 0 | 0 | 37.967s |

原四项目标均 PASS：RoleWorkspaceAnchorBackfill、FeeCatalogReferenceValidation、FeeCatalogCompanyTemplates、CompanyBoundaryMigrationRejectsReferencedSystemPrivateFeeAtomically；费用目录种子相邻用例也 PASS。这里的子项计数仅一级缩进，不与顶层混加。R2 后续旧脚本曾失败，不影响已完成的数据库门禁证据；本报告以 R29 作为最终运行时链路证据。

## 最终 HTTP 场景 R29

| 场景 | 同链编号 | 输入及期望 | 实际 | 结果 |
| --- | --- | --- | --- | --- |
| 应收 | SE2026092800001；BG2026092800003；BI2026092800003；INV2026092800001；PR2026092800001/2；WO2026092800001/2；CM2026092800005 | 100+25=125 CNY；40 CNY 来源按10%产生4 CNY提成 | 金额与八位精度断言一致，最终反核销账单余额125、流水余额40恢复 | PASS |
| 应付 | SE2026092800003；BI2026092800004；PR2026092800004；WO2026092800003 | 费用/账单/付款/核销88 CNY；收款方向不能核销应付账单 | 全部88.00000000，方向错误被拒绝 | PASS |
| USD/EUR | SE2026092800001；BI2026092800001；INV2026092800001；PR2026092800001；WO2026092800001；CM2026092800001（独立USD库） | 100 EUR费用@1.1=110 USD；账单@1.2=120；发票@1.22=122；40 EUR流水@1.25=50；核销账单分摊48/流水50/汇差2；毛利48×10%=4.8 | 110/120/122/50、48/50/2、4.80000000全部吻合，原单据快照重读保持原汇率 | PASS |

USD 汇率设置 ID `01a0e55f-6038-7239-9f17-595a273eb7f4` 在业务节点前合法更新，单据分别固化自己的八位金额/汇率/日期/来源/设置 ID。当前人工维护自然周来源为 WEEKLY。提成的旧 cny* 字段按现行原币记账契约固化恒等快照：1 / BASE_CURRENCY / 无设置 ID / 4.80000000，未做旧口径二次 CNY 派生折算。

保留并实跑的应收相邻断言包括：建账/收付/核销/提成并发幂等、竞争键冲突、超额核销拒绝、账单/资金余额回收、非默认开票主体快照、提成来源指纹失效、未建账费用确认拒绝、财务锁、调整幂等/生效/超减拒绝、反核销自动冲回提成。仅备注/版本变化不改变经济来源指纹；新增未建账费用使旧提成先 SOURCE_CHANGED，重新生成同来源草稿后精准验证 UNCONFIRMED_FEES。

独立占用样本金额7.25000000：订单 `01a0e55e-a0d0-76e3-896b-b48623a43dca`、费用 `01a0e55e-a0e3-7ccd-814b-d4aa1a602d9a`、取消账单 `01a0e55e-a10d-7e23-a821-9e402c0e216b`、重建账单 `01a0e55e-a169-7df5-b6ec-0f7bc367691e`。草稿占用、重复建账409、编辑409、删除409、取消释放、重建均 PASS，未扰动主链。

## 同一订单浏览器链路 R29

API 仅准备客户、合法公司员工、结算账户、空订单与10%方案；关键费用、账单、流水、核销、提成均由 UI 真正写入。没有 mock、路线拦截或 API 预写关键业务单据。

| 实体 | 编号 | ID |
| --- | --- | --- |
| 订单 | SE2026092800004 | 01a0e55e-bd12-7bf8-9be3-71c4f70e5306 |
| 费用 | 单条应收125 CNY | 01a0e55e-d38e-7e91-930a-5d70941fbfc6 |
| 账单 | BI2026092800005 | 01a0e55e-e132-75d3-b510-bb00d63ba811 |
| 收款 | PR2026092800005 | 01a0e55f-0c25-72f9-aa38-859f55c26aa1 |
| 核销 | WO2026092800004 | 01a0e55f-237e-7da8-8cfd-b157b76387c6 |
| 提成 | CM2026092800006 | 01a0e55f-3b3d-74a6-acb4-35e077e74233 |

期望和实际：费用=账单=收款=核销125.00000000；收入125.00000000、成本0.00000000、毛利125.00000000；提成12.50000000。账单仅一行且 orderId/orderFeeId 正确，核销仅一分配且 billId/cashflowId 正确，提成仅一订单行及一费且 orderId/feeId/verificationId 正确。刷新后同提成详情和资金/账单已核销额均重读一致，资金余额0。

UI 同时验证：草稿账单已占用费用；确认账单后删除入口隐藏；再次建账明确警告且不打开工作台；金额显示125.00/12.50 CNY，比例10%，时间是中国本地可读格式。意外 API 4xx/5xx、pageerror、非导航 GET 中止之外的 requestfailed 都计为失败；最终 errors=[]。导航只读中止单独保留在私有附件，写请求中止绝不豁免。

## 截图与原始诊断

最终私有目录：`/tmp/roncin-finance-browser-evidence/roncin-finance-browser-TvOneA`，运行后复核权限700。runner 设置 umask077，确保 Playwright 清空后重建仍700，所有原始 trace 均仓库外保存，不提交认证 storage-state。

该目录的财务用例子目录内有5张成功节点截图：

- `01-fee.png`：同订单新增125.00应收。
- `02-bill.png`：同费用账单125.00 CNY、批次全部确认。
- `02-billed-guard.png`：已建账重复生成警告与删除入口隐藏。
- `03-verification.png`：本次核销列表与编号。
- `04-commission.png`：本次12.50 CNY提成、来源WO编号、同订单行、本地时间。

成功运行没有保留 trace；失败迭代的 trace 原件仍在相应 `/tmp/roncin-finance-browser-evidence/roncin-finance-browser-*` 私有目录，只用于本地诊断。不能把原件上传/提交，因网络快照可含登录密码、CreateUser密码和Cookie。可分享证据只使用上述无可见凭据的业务截图及本报告脱敏摘要。

## 清理与最终检查

R29 仅清理本次自己创建的资源：

- CNY库/角色 `roncin_acc_fin_cny_1790554573122_f5fa4f0b`。
- USD库/角色 `roncin_acc_fin_usd_1790554623510_54ea7481`。

两组均在日志显示数据库及角色清理验证成功；任务Go/Vite进程被回收，已有8001用户服务与开发库保持原状。早期 R2 的 `callback is not a function` 栈来自 esbuild 子进程在失败清理期的 Vite 异常输出；未把它当作本任务清理逻辑已修复。最终 R29 未复现该异常，且编排清理验证全通过。

R29 运行时 runner 的尾部成功文案曾概括为“双环境全链路”；现已按 `--runtime-only`、`--stage-b-only` 和默认全量分别标明实际范围。文案调整后只做语法检查，R29 的业务代码、执行范围和结果不变。

定向检查：5个Node脚本 node --check PASS；Playwright配置与E2E Biome PASS；pnpm --dir web tsc PASS；git diff --check PASS。父任务负责最终 check:fast、审阅、提交与归档。

## 未覆盖项与限制

不覆盖发票UI、提成确认/付款UI、应付UI和通用订单输入E2E；这些未执行项不冒充浏览器PASS。HTTP应收覆盖发票、提成确认及调整。用户人工走查未执行。未发现阻塞本次财务最薄闭环的问题；生产契约没有为验收放宽。
