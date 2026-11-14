# 四类财务报表独立审查（2026-11-14）

## 范围与结论

独立沿Proto→service→biz→data→页面核对报表契约，未沿实现代理的“通过”描述直接接受结论。检查四入口路由/标题/access、Manifest与服务端RPC鉴权、当前公司门禁、只读候选权限、查询/完整合计/月图/导出共同来源，以及前端官方request/onLoad采纳和导出迟到隔离。

除下述已修金额边界外，未发现其他需要修改的报表缺陷。最终全栈门禁、生产构建、HTTP与浏览器验收由主会话统一执行；本审查不声称替代这些验收。

## 已修复

`server/internal/data/finance_report_profit.go` 原先直接对 `base_currency_amount * net_amount / total_amount` 的SQL商执行8位半偶舍入，漏掉领域 `CommissionNetBase` 内 `decimal.Div` 默认16位中间舍入。快照总额 `1000000000.00000001`、未税 `600000000.50000001`、含税本币 `10.00000000` 时，领域结果为 `6.00000000`，省略中间除法的SQL会得 `6.00000001`，费用/提成与报表不再精确对账。

现SQL先按 `decimal.DivisionPrecision` 进行与域一致的半离零中间舍入，再执行8位半偶舍入，各费用先计算再汇总。没有修改领域金额公式、公开契约、费用或提成数据。追加真实PG回归核对同一近中点快照的利润行、完整合计、月图，应收正利润与应付负利润分别验证。专项规范同步记录完整舍入顺序。

## 已核对的关键边界

- 利润SQL与 `currentOrderFeePredicate` 的INCREASE且未替代条件一致，包含未建账当前费用，不把反向清偿或旧叶子重复计入。
- 日期复用订单列表的权威航期表达式；出口ETD、进口ETA按业务时区归日，AE不借用陈旧订单投影。空日期明确提示，日期筛选不把它伪塞入范围。
- 岗位筛选为EXISTS；同一订单不同本币分行，原币和本币分桶；月序列/完整合计不取当前页代算。
- 真实收付固定CONFIRMED与交易日期，反核销不减真实发生，对冲不计现金。
- 账龄按有效核销与有效对冲分别预聚合后扣余额，避免明细乘积；五段与缺日期独立，包含更正清偿用途，仅当前余额。
- 三类导出与列表复用筛选/事实CTE和只读REPEATABLE READ，10000行明确限额；没有用循环分页绕200上限。
- 候选及报表全部只需 `system.finance.report.read`，service对当前公司再校验，系统工作区及跨公司拒绝；不附加订单/客商/账单/发票读取权限。
- 四路由名称与access来自routes.ts，金额/日期展示和CSV沿公共工具，导出使用已采纳筛选。筛选配置复用本地偏好与移除清值能力。

## 本次独立执行

- 新增币种及候选真实PG：`TestFinanceReport(CurrencyDimensions|OptionsSearchAndCompany)` 通过，data包4.676秒（`/tmp/report-review-dimensions.log`）。覆盖多本币、原币中英代码别名拼音、公司人员范围。
- 修复后利润及相邻真实PG：`TestFinanceReport(CurrencyDimensions|OptionsSearchAndCompany|NetAmountMatchesDomainDivision|Profit|BankRound)` 通过，data包11.832秒（`/tmp/report-review-financial.log`）。包含现有正负半偶中点、来源更正/未建账、日期/分页/汇总与同源导出；其中近中点新增例首轮为应收，之后补应付方向单独复验。
- 近中点应收/应付最终真实PG定向通过，data包2.367秒（`/tmp/report-review-division.log`），两方向的行、完整合计和月图均与域公式一致。
- biz/service `TestFinanceReport` 定向通过（0.012秒/0.028秒），包括独立报表权限、当前公司及三导出/两候选同范围（`/tmp/report-review-scope.log`）。
- 整合 `pnpm --dir web tsc` 通过（`/tmp/report-review-tsc.log`）。
- 报表及独立权限13个文件Biome通过，`go -C server vet ./internal/data`、`git diff --check`通过。
- 未重复运行早先已通过且此次未改的全部前端或全量data测试；主会话完整门禁将覆盖最终整合代码。没有把SKIP记为通过，没有清空/重建开发数据库，没有提交Git或手改生成物。


## 最终静态门禁追加：RPC标准消息命名

全栈静态门禁发现首轮report.proto让列表/导出复用响应、两候选复用请求及响应，违反仓库Buf的RPC_STANDARD_NAME及RPC_REQUEST_RESPONSE_UNIQUE。已按RPC各自声明唯一标准消息：三类List/Export各自响应、两类Options各自请求与响应；原有六类List/Export请求本已各自命名，保持不变。HTTP路径、字段号码、JSON名称、金额精度与optional语义均保持。

正式执行 `make -C server api` 和 `pnpm run generate:web-client` 同步Go、OpenAPI及前端客户端，service/页面/测试一次性更新，不留旧类型别名或兼容层。增加 `finance_report_contract_test.go`，用包含金额、optional、汇总和月序列的真实ProtoJSON验证三类列表与独立导出的JSON结构/值保持一致。

本次验证：Buf lint通过；biz/service报表定向通过（0.020秒/0.025秒，含新增三类wire子用例）；前端报表及独立权限5文件16用例通过（7.55秒）；整合tsc、报表Biome、git diff --check通过。生成器已释放，无需Ent Schema生成，未改查询/数据口径，不重复PG全包。最终全量及生产构建仍由主会话在此契约冻结后统一完成。
