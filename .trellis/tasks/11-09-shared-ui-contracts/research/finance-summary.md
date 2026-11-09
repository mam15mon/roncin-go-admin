# Research: 财务列表汇总口径与采纳快照

- Query: 六类财务列表的全量汇总契约、行模型与底栏是否匹配；指标竞态是否真实；最低成本修复方案。
- Scope: internal（仓库 Proto、biz/data/service、页面、公共模板及已安装框架源码）
- Date: 2026-11-09

## Findings

### 1. 已确认：底栏将本页数据错误命名为全部汇总

- `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:380` 只在 ProTable 的 `onLoad` 采纳后同步 currentData/totalCount；`:392` 将 currentData 作为 allRows 传给底栏，而 totalCount 为服务端全量记录数。
- `FinanceSummaryBoard.tsx:28` 的 aggregateFees 只累加传入行；`:101` 的 allMetrics 从 allRows 计算，其后只覆盖旧平铺字段 receivableBaseAmount/payableBaseAmount/profitBaseAmount/baseCurrency。
- `types.ts:44` 的 FinanceLedgerGlobalSummary 是旧平铺可选字段；正式 FeeLedgerSummary 的汇总是 `amountsByBaseCurrency` 数组（`server/api/finance/v1/settlement.proto:350`），因此当前费用台账传入正式 summary 仍不能覆盖这些金额。
- 六个列表都没有主动关闭 showSummaryBoard；公共模板默认 true（`FinanceLedgerTemplate.tsx:68`）。费用页是唯一把 summary 返回公共模板的页面（`pages/finance/fees/index.tsx:583`）；另外五页只更新页面 metricStats，不返回 summary。
- 还有多本币误合并：`FinanceSummaryBoard.tsx:35-45` 将所有行的本币金额相加，baseCurrency 则取最后一条。跨公司不同本币的行混在一起会被标成最后一个本币。必须按币种分组，不能靠默认 CNY。

### 2. 六类正式契约与状态口径

全部现有 summary 都是按**本币**分组，不提供原币全量汇总。

| 列表 | Proto summary | 服务端计算口径 | 行金额字段与当前底栏差异 | 是否适用利润 |
| --- | --- | --- | --- | --- |
| 费用 | FeeLedgerSummary: activeCount + amountsByBaseCurrency，每组 receivable/payable/profitBaseAmount | 当前筛选下的有效费用，GroupBy(direction,base_currency)，在分页前聚合。`data/settlement.go:478-516`；total 是筛选匹配总行数，activeCount 只算有效行 | totalAmount/baseCurrencyAmount 与费用模板基本匹配，但经济失效行参与已选/筛选明细合计，必须区别于顶部有效金额 | 费用应收减应付可展示“费用利润”；不能拿其他实体差额沿用此名 |
| 账单 | FinanceBillSummary: amountsByBaseCurrency，每组 receivable/payable/unverified/overdueReceivableBaseAmount | 筛选谓词额外 AND status=CONFIRMED；未核销扣活动核销及对冲分配，逾期只算应收逾期未清。`data/finance_bill.go:95-100,118-159,161-234` | totalAmount/baseCurrencyAmount 与原币字段可映射；草稿等行可见但全量有效金额不含它们 | 应收账单减应付账单只是账单差额，不等同订单利润 |
| 发票 | FinanceInvoiceSummary: amountsByBaseCurrency + issuedCount | 当前筛选额外 AND status=ISSUED；仅 BaseCurrencyAmountNotNil 的行进入金额，issuedCount 可包含无折本币值记录。`data/finance_invoice.go:57-96` | totalAmount 可映射，baseCurrencyAmount 为 optional；缺折本币不能伪造为 0 或 CNY | 不适用费用利润 |
| 收付 | FinanceCashflowSummary: amountsByBaseCurrency，含 receivable/payable/unverified | 当前筛选额外 AND status=CONFIRMED；未核销扣 active 核销分配。`data/finance_cashflow.go:56-107` | 行是 amount/baseAmount（`settlement.proto:850`），模板读取 totalAmount/baseCurrencyAmount，直接累加会得到 0 | 应展示收款/付款，差额若展示只能明确为资金净额 |
| 核销 | FinanceVerificationSummary: amountsByBaseCurrency，实际填 receivable/payable | 当前筛选额外 AND status=ACTIVE，金额为核销 BaseAmount。`data/finance_verification.go:57-80` | 行是 amount/baseAmount，另有 billBaseAmount/cashflowBaseAmount/exchangeGainLoss（`settlement.proto:868`）；默认费用底栏取错字段 | 不适用费用利润；核销汇差与费用利润是不同概念 |
| 对冲 | FinanceNettingSummary: amountsByBaseCurrency（nettingBaseAmount）+ confirmedCount | 只算已确认，保留 keyword/party/currency，但有意不追加列表 status。`data/finance_netting.go:56-88` | 行是 amount/baseCurrencyAmount，无 direction（`settlement.proto:876`）；当前底栏应收/应付全为 0。BaseCurrencyAmount 是应收侧抵销本币，不应解释为对冲利润（`biz/finance_netting.go:58-60`） | 不适用；应该展示抵销金额 |

- Proto 锚点：`settlement.proto:350-353`（费用）、`:774-782`（公共本币与账单）、`:833-834`（发票）、`:850-852`（收付）、`:868-870`（核销）、`:891-893`（对冲）。
- biz summary 与 service 转换保持这些字段；例如 `service/settlement_bill.go:179-192`、`service/settlement_verification.go:81` 没有补原币汇总或利润。
- 前五类对当前 status 进一步 AND 有效状态，选择非有效状态时有效金额可能为 0；对冲的有效汇总特意独立列表状态。这是现有后端语义，本期不能悄悄统一成另一套计算。顶部标题/说明必须能表达这个区别。

### 3. 已确认：顶部指标竞态可确定性复现

六页都在 request 完成时直接 setState：

- 费用 `index.tsx:578` setSummary；账单 `:637`、发票 `:586`、收付 `:541`、核销 `:407`、对冲 `:471` setMetricStats。
- ProTable 对 request 的完成与是否采纳是分开处理的：已安装 `@ant-design/pro-components` **3.1.14-6**（含仓库 pnpm patch），`es/table/Table.js:303-313` await 用户 request；`es/table/useFetchData.js:169-187` 在 getData 返回后才检查 signal.aborted，然后 setDataAndLoading、onLoad。
- 旧请求即使被内部 abort 标记为取消，用户 request 中 await 后执行的 setState 仍先发生；内部取消只能阻止表格/onLoad 采纳，无法撤销这个外部副作用。
- `useFetchData.js:354-370` effects 变化会 abortFetch 并启动新请求；`:397-409` reload 同样先取消再请求。适用于快速切筛选和连续刷新。

已运行研究实验（不改产品和现有测试）：

```bash
pnpm --dir web exec vitest run --config ../.trellis/tasks/11-09-shared-ui-contracts/research/race-repro.config.mjs
```

结果：**1 文件 / 1 用例通过**，耗时 3.32s。实验直接使用仓库安装的 useFetchData 和真实 React hooks，不是手写模拟采纳引擎：

1. A 请求未完成，切换筛选到 B，框架取消 A。
2. B 先完成，表格采纳 B，指标为 B-200。
3. A 后完成，表格仍为 B、onLoad 仅收到 B，但指标已变成 A-100。

复现文件：`research/race-repro.test.mjs`；配置：`research/race-repro.config.mjs`（以 `pnpm --dir web` 的 cwd 找依赖，以 import.meta.url 找测试；归档后更新 --config 路径即可）。实验没有真实网络请求、没有 sleep，按 deferred promise 控制顺序并用 waitFor 等待框架行为。

### 4. 用户最新要求与最薄正式契约扩展（本节替代早期“本页”建议）

用户已明确底栏只保留**单行条件切换**：有勾选展示已勾选合计；没有勾选展示当前全部筛选结果合计。不能再用当前页合计充当默认项，也不同时堆两行。

#### 4.1 分离两个同时真实存在的汇总口径

- **顶部有效业务指标**：使用正式 summary 的有效状态口径（对冲拟修正见第 8 节），展示有效费用、已确认账单/收付、已开票发票、有效核销、已确认对冲。它表达经营/财务有效事实。
- **底栏全部筛选结果**：金额集合严格与当前列表过滤后的 total 集合一致，包含列表当前条件能看到的草稿、取消、反转或失效记录；未选定状态时也不能额外偷偷加有效状态。它表达“这些明细数值的合计”，不等于有效业绩。
- **底栏已选**：仅已勾选的明细快照，允许用户逐页勾选并跨页累计。各实体独立投影，原币、本币分别分组；对冲没有 direction，不套应收/应付/利润。
- 顶部和底部金额在未限定有效状态时不同是正确结果，应该用口径标签解释，不应为了看起来一致而改变业务聚合。

简洁产品解释建议：顶部是“已生效业务金额”；底部是“当前筛选明细合计／已勾选明细合计”。例：列表有一张 100 元草稿和一张 200 元已确认账单，底部全部合计 300 元，顶部已确认金额 200 元；筛选“已确认”后两者一致。

#### 4.2 推荐新增的共同只读消息

六类现有 summary 增加同语义的 `matched_summary`，不改变原有效指标字段的语义；这是两种真实业务视图，不是历史兼容双读。建议草案：

```proto
message FinanceLedgerAmountBucket {
  string currency = 1;
  string direction = 2; // RECEIVABLE/PAYABLE；无方向的对冲为空，由实体适配器解释。
  string amount = 3;    // decimal 字符串，不传浮点。
}
message FinanceLedgerMatchedSummary {
  repeated FinanceLedgerAmountBucket amounts_by_currency = 1;
  repeated FinanceLedgerAmountBucket amounts_by_base_currency = 2;
  int64 unconverted_count = 3; // 发票等行缺少实际折本币值时计数，不伪造 0。
}
// 每种正式列表 summary 增加一个独立新 tag：
// FinanceLedgerMatchedSummary matched_summary = N;
```

- 也可用两个专有命名消息来区分 currency/base_currency，但不要建新的可配置聚合平台或任意指标 DSL。上述同 bucket + 两个集合是满足本期六页的最薄字段集合。
- 总记录数继续用已有 ListResponse.total，避免重复统计真相。Proto repeated 零值省略时按空集合处理；不可把失败响应视为空集合。
- 新增字段 tag 根据实际消息可用编号选择，不能复用既有 tag；从 Proto 生成 Go/OpenAPI/前端客户端，同一组提交。
- 前端 `FinanceLedgerMatchedSummary` 统一适配成稳定展示分组；只在费用页可由应收减应付展示“费用利润”，其余实体不凭差额生成利润。
- `unconverted_count` 是当前筛选集合中缺实际本币金额的行数。发票草稿有 BaseCurrency 但尚无 BaseCurrencyAmount（`biz/finance_invoice.go:353`；Issue 才写实际值 `:328`），原币仍必须进入底栏；本币只汇总非空已折算值，并提示尚有 N 条未折算，不能把 null 当真实零。

#### 4.3 六页只读聚合对应

| 实体 | 原币 SUM 字段 | 本币 SUM 字段 | 分组与特殊点 |
| --- | --- | --- | --- |
| 费用 | order_fees.total_amount | base_currency_amount | 原币按 direction,currency；本币按 direction,base_currency。全部使用 `baseQuery` 同筛选集合，不额外叠 currentOrderFeePredicate |
| 账单 | finance_bills.total_amount | base_currency_amount | 使用分页 query 的 predicates，不追加 CONFIRMED；应收/应付分组 |
| 发票 | finance_invoices.total_amount | base_currency_amount（nullable） | 原币全体同筛选；本币仅已折算，另计算未折算数量；不追加 ISSUED |
| 收付 | finance_cashflows.amount | base_amount | 使用列表同 p，不追加 CONFIRMED；投影中文“收款／付款” |
| 核销 | finance_verifications.amount | base_amount | 使用列表同 p，不追加 ACTIVE；不把 billBaseAmount 或 cashflowBaseAmount 混作本金 |
| 对冲 | finance_nettings.amount | base_currency_amount | 使用 query 同 predicates（包括当前 status），不复用有意排除列表 status 的有效指标 summaryPredicates；无 direction，显示抵销金额 |

每个 repository 已有 COUNT、有效业务 GroupBy、分页查询。新增 matched_summary 只需在**分页前同一过滤 query**上做原币和本币 GroupBy SUM；Ent 查询封装与 decimal 转换沿用现有方式，不加载全量实体，也不循环翻页。发票可用 COUNT(*) 与 COUNT(nullable base_amount) 差值或明确的缺失计数聚合，避免新建详情遍历。

本期改动范围为 `settlement.proto` 六种 summary、新只读 biz DTO、六个 data.List 聚合、service 映射、对应生成物及六页/公共 UI，**不改数据库 Schema、财务计算规则、交易写路径、锁或事务配置**。不要为了“一个快照”新建事务策略；列表各聚合 SQL 的数据库一致性策略沿用现有接口。

### 5. 原生跨页选择、刷新与采纳快照

#### 5.1 已有基础与成本

- 公共模板已传 controlled selectedRowKeys、preserveSelectedRowKeys=true，并在 rowSelection.onChange 同步 keys/rows（`FinanceLedgerTemplate.tsx:345-363`）。不需要为手动跨页选择新建后台选择 token 或全量 ID 解析接口。
- 实际安装 AntD `es/table/hooks/useSelection.js:71-90` 的 preserveRecordsRef 缓存已选行；`:156-179` 通过缓存重建 selectedRows，并调用 onChange。翻页后的其他页已选行仍可参与合计。
- 表头全选只针对 pageData 展平后的 recordKeys（同文件 `:270-297`），已有跨页 keys 不被删除；本页全选是把本页 key 加入现有集合，取消本页全选只删除本页 key。不能把它标成“全部筛选结果全选”。
- 内存和本地金额计算按已选数量 O(k) 增长；翻页只执行正常列表请求，不为保持选择额外请求全部数据。此处是用户逐页勾选，不含“一键选中后端全量结果”。
- ProTable `clearSelected` 走 onCleanSelected，后者调用 rowSelection.onChange([],[],{type:'none'})（`es/table/Table.js:421-429,467-473`），可同时清模板 keys/rows，不需要另一套清理协议。

#### 5.2 最小必要生命周期策略（筛选策略已确认）

1. **分页、pageSize、排序变化**：结果集合身份没变，保留已选 keys/records，合计跨页累加；排序本身不能当筛选变化清选择。
2. **已提交筛选、公司/身份变化**：用户已确认清空选择。公司/身份切换已由工作区隔离卸载保护；同页面新筛选必须有显式统一选择清理，不把“新筛选下全部”与旧筛选已选混在一起。应在筛选提交时清理而非每次打字清理。本期不保留跨筛选手选，不需再次询问这一点。
3. **新增/编辑/删除/确认等成功后的 reload**：建议清空已处理选择，再按官方 actionRef reload。避免刚建账、取消或删除记录仍以旧值、旧能力参与底栏和下一次批量操作。失败不清，保留用户选择便于纠正。
4. **普通手动刷新**：未出现后台主动变更保证时，缓存行是选择时的快照，尤其非当前页已选无法由当页请求证明新鲜。最低成本建议成功刷新后清空选择（失败保留），使底栏回到当前筛选全量 server totals。若要求刷新仍保留并确保全页已选最新金额，则需按已选 ID 批量重读/验证，超出原生缓存能力，应另定范围；不能假称缓存就是实时值。
5. **新页采纳**：如果选择保留，优先使用框架缓存对当前可见已选行的更新，不能只更新 keys 而让模板 selectedRows 永远保留旧对象。实施时补“已选行刷新/翻回后金额变化”回归；不为普通分页强行全量校验离页选项。
6. **失败与取消**：不把失败当零合计。请求被取消/迟到不能切换底栏模式或推进 summary；整个响应正式采纳时才替换全部筛选 totals。

普通刷新策略也是语义选择；可以在同一个高价值问题中描述“筛选/主动刷新时清空，翻页保留”的完整规则，或暂不承诺刷新保持。

#### 5.3 顶部、底栏、表格统一采纳快照

扩展当前已有 WeakMap + onLoad，request 返回数据、total、正式 summary，不在其中 setMetricStats/setSummary：

```ts
type FinanceLedgerAcceptedSnapshot<T, TSummary> = {
  rows: T[];
  total: number;
  summary: TSummary; // 同时带原有效业务指标和 matchedSummary。
};
```

ProTable 的官方 onLoad 采纳后才通知页面顶部指标和公共底栏；六页统一 typed onAcceptedSnapshot。框架内部取消仍负责舍弃迟到结果，不添加 sequenceRef 等手工竞态令牌。当前 WeakMap 已满足 data 数组与 metadata 的关联，无需依赖 onLoad 第二参数 any。

底栏判定用已选 key 数而不只用 selectedRows.length（防止 keys/records 缺损时错误退回全量）；原生手选路径应保证一一对应，发现记录丢失要显式处理而非悄悄少算。

### 6. 最小验收建议

- 无勾选时单行显示“全部筛选结果 N 条”，金额来自正式 matched_summary；跨页数据超过 200 仍不循环请求全量。
- 有勾选时同一位置切为“已勾选 N 条”，跨页手选、取消本页全选保留其他页选择，全部取消立即回全量。
- 六种字段投影正确，尤其 amount/baseAmount、无 direction 对冲不能静默归零；非费用不显示利润。
- 多原币、多本币独立分组；发票未折算单据原币入账、本币明确缺失，不伪造真实零。
- 顶部有效指标采用最终批准的状态规则（包含第 8 节对冲调整）；底部筛选合计与列表集合一致，包括状态筛选、无状态、草稿/失效/取消/反转。
- 明确并验证筛选、排序、翻页、成功写后 reload、普通刷新成功/失败的选择生命周期。
- 原有乱序实验在实现后变为表格/顶部/底栏/导出都保留 B，A 的迟到返回不影响采纳快照。

### 7. 相关规范与文件

- `.trellis/spec/web/frontend/state-management.md`：服务端状态、ProTable 官方请求豁免、错误响应与快照隔离。
- `.trellis/spec/web/frontend/component-guidelines.md:121`：官方 request/actionRef 范式；金额格式规范。
- `.trellis/spec/web/frontend/type-safety.md`：ProtoJSON 零值/optional presence 与领域映射。
- `FinanceLedgerTemplate.tsx`：公共模板已存在的采纳快照及导出边界。
- `FinanceSummaryBoard.tsx`：当前费用专用聚合被错误用作六类通用聚合。

## Caveats / Not Found

- 本轮没有浏览器点击复现或真实数据库查询；汇总范围依据正式 Proto 和 repository 聚合代码，竞态依据真实安装框架 hook 的确定性实验。
- 用户已选择“单行：有选中显示已选，否则显示全部筛选结果”；早期“本页”方案已被替代。用户已确认“翻页保留勾选，筛选条件/公司变化清空”。普通手动刷新和写成功后的已选快照更新仍按第 5 节建议明确到最终设计，不把离页缓存冒充实时数据，不额外建设后台全量选择平台。
- 研究没有修改任何产品代码、现有测试、生成物、数据库或 Git 状态；仅在本任务 research 下写入证据与可执行研究实验。

### 8. 最终规划补充（主会话最新决策）

- 底栏正式字段建议命名 `matchedSummary`（Proto `matched_summary`），明确“与列表完整筛选条件匹配的明细汇总”；上文已同步该命名。bucket 可复用，按原币及本币各一个集合，不引入任意查询/维度 DSL。
- 用户已确认跨页手选保留、筛选/公司变化清空。原生 preserveRecordsRef 足以支持这一级能力；排序/pageSize 不改变集合，不必清空。
- 主会话拟将对冲顶部也调整为“完整列表 predicates + CONFIRMED”，与其余有效业务指标的 status 交集规则一致。此处是**有意修正现有读取口径**：旧实现不带 filter.status，所以选草稿仍显示已确认金额；新方案选草稿时已确认金额为零。务必在最终摘要明确供用户批准，不能作为纯技术重构偷偷改变。
- 在批准该读取口径变化后，复用列表 query/predicates 计算 matchedSummary 和已确认 summary 是最薄的统一方式；不修改对冲、核销、账单交易写路径，不加新事务/锁配置。
- decimal 聚合以 Ent SQL SUM + 既有 decimalOf/StringFixed 口径完成；前端已选合计使用项目已有 decimal.js 或等价精确十进制工具，不把 Number 浮点误差固化为金额。未折本币计数仍需要，以区分发票 null 和真实零。

研究实验可移植配置已重跑通过：1 文件 / 1 用例，3.27s；没有修改产品代码。
