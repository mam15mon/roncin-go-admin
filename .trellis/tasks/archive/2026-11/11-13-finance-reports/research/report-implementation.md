# 四类财务报表首轮实现与验证（2026-11-14）

## 已实现行为

- 利润图和利润表共用当前有效费用集合，含未建账费用，含税/未税并列；出口ETD、进口ETA，空运复用权威主单日期；原币与本币各自分组。
- 实际收付款只读CONFIRMED流水及其金额快照，不将核销、反核销或对冲重复计为现金。
- 当前账龄按有效核销/对冲后的账单余额唯一分桶，保留未设到期日辅助桶；不提供历史截止日。
- 三个列表及三个独立导出共用筛选内核和只读一致快照；导出10000行上限明确拒绝超限。服务端独立report.read权限、当前公司范围及两类只读候选已接通。
- 四页面使用官方ProTable request/采纳快照驱动行、完整汇总与月份图；完整导出读取已采纳筛选，卸载、切公司或切已提交筛选后的迟到结果不下载。

## 文件边界

- 新契约：`server/api/finance/v1/report.proto` 及对应Go生成绑定；OpenAPI、前端生成客户端、权限生成清单一并更新。
- 新用例：`server/internal/biz/finance_report{,_types,_test}.go`。
- 新仓储：`server/internal/data/finance_report{,_profit,_cashflow,_aging}.go` 及四类专项集成测试文件。复杂聚合使用Ent官方sql/execquery特性，生成客户端QueryContext经过Data.client读取快照；不直连池。
- 新传输：`server/internal/service/finance_report{,_test}.go`；provider、wire、HTTP/gRPC注册和鉴权范围同步接通。
- 页面：`web/src/pages/finance/reports/`；四路由、access及角色权限清单同步更新，独立权限测试为`web/src/access.report.test.ts`。
- 规范：`finance-reports.md`、server/web能力导航和server规范索引。

## 已执行验证

- `go -C server test -p 32` 对biz/service报表与server鉴权/路由定向验证通过（日志`/tmp/report-backend-tests.log`）。
- 真实PostgreSQL `go -C server test -p 32 ./internal/data -run '^TestFinanceReport' -count=1 -timeout=15m` 首轮8项通过（31.740秒，日志`/tmp/report-postgres.log`）：当前叶子更正、不建账也计利润、多岗位不倍增、AE与进口业务日期、本地跨日、完整零月份、未归期提醒、确认现金/撤销、账龄边界、多核销多对冲及反向恢复、REPEATABLE READ与独立连接并发提交、半偶正负中点、10001拒绝及10000完整导出。
- `pnpm --dir web exec vitest run` 报表4文件14用例通过（7.64秒，日志`/tmp/report-web-tests.log`）：wire省略零值/optional/缺必填、官方采纳后汇总、当前工作区、完整导出与迟到隔离、双口径月图/负值/切本币。
- 独立权限、角色授予及图表补验3文件14用例通过（16.63秒，日志`/tmp/report-permission-tests.log`）；角色测试数据的新增权限排序已与权限树分组遍历一致。
- `pnpm run check:architecture`通过（18个检查用例及产品依赖扫描）。报表页面、routes/access和角色测试的Biome检查通过；`git diff --check`通过。
- `pnpm --dir web tsc`本次唯一阻塞为并行发票改动中的`invoices/current-workspace.test.tsx`仍引用已退役CreateInvoice接口；已交所属实现代理同步处理。报表源文件本轮无类型错误。
- proto/Ent/OpenAPI/客户端/权限均通过官方生成入口，未手改生成物。未增加报表Schema或索引，无对应迁移；未清空或重建开发数据库，数据库验证使用隔离测试schema。

## 待最终验收

追加的币种维度、中英别名拼音及公司人员候选两个真实PostgreSQL用例已写入`finance_report_dimensions_integration_test.go`，本次运行被并行发票SourceTaxRate从string改optional过程的编译状态阻断（`finance_invoice.go:356`），未执行测试主体；主会话已指定独立检查在发票生成稳定后补跑：

```bash
go -C server test -p 32 ./internal/data -run '^TestFinanceReport(CurrencyDimensions|OptionsSearchAndCompany)' -count=1 -timeout=5m
```

该命令须配置隔离测试库的`RONCIN_INTEGRATION_DATABASE_SOURCE`，不能把无数据库导致的skip当通过。主会话负责独立审阅、并行实现整合后的tsc和全栈门禁、浏览器验收及按组提交；当前不宣称这些最终门禁已完成。
