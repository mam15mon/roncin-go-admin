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
| 费用 | FeeLedgerSummary: activeCount + amountsByBaseCurrency，每组 receivable/payable/profitBaseAmount | 当前筛选下的有效费用，GroupBy(direction,base_currency)，在分页前聚合。`data/settlement.go:478-516`；total 是筛选匹配总行数，activeCount 只算有效行 | totalAmount/baseCurrencyAmount 与费用模板基本匹配，但经济失效行是否参与已选/本页需要明确文案 | 费用应收减应付可展示“费用利润”；不能拿其他实体差额沿用此名 |
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

复现文件：`research/race-repro.test.mjs`；配置：`research/race-repro.config.mjs`。实验没有真实网络请求、没有 sleep，按 deferred promise 控制顺序并用 waitFor 等待框架行为。

### 4. 推荐的最低成本实现

#### 4.1 明确三种数据口径

- **顶部全量指标**：继续使用现有服务端 summary，按 baseCurrency 独立展示，准确写“有效费用”“已确认账单/收付”“已开票”“有效核销”“已确认抵销”；不把它们全部称为无条件“全部金额”。
- **底栏当前页**：将“总计(全部)”改为“本页明细合计”，记录数取 currentData.length；按行原币和本币分别分组，显示本页可见行的字面金额，不宣称跨页全量或有效业绩。
- **底栏已选**：只计算明确已选行，沿用真实跨页选择数据；各实体显式行投影，收付/核销映射 amount/baseAmount，对冲无 direction 只显示抵销金额。optional 本币未取得的发票不虚构折本币，展示未折算或缺失状态。
- 公共 UI 可接收配置化分组/金额项或 renderSummary 插槽；领域把行/summary 投影成稳定展示模型。不要在通用组件 import 财务实体 DTO 或枚举，不靠同名 optional 字段强行把六种业务套成费用。
- 仅费用显示费用利润；账单/发票/收付/核销/对冲不能默认展示“利润”。

#### 4.2 统一采纳快照

可以扩展当前已有 WeakMap + onLoad 机制，不另造请求引擎：

```ts
type FinanceLedgerRequestResult<T, TSummary> = {
  data: T[];
  total: number;
  success?: boolean;
  summary: TSummary;
};
type FinanceLedgerAcceptedSnapshot<T, TSummary> = {
  rows: T[];
  total: number;
  summary: TSummary;
};
```

- 页面 request 只返回数据和正式 summary，不在里面 setMetricStats/setSummary。
- 模板用带 summary 泛型的 WeakMap 绑定请求返回 data 数组和完整快照，ProTable `onLoad` 才同步 currentData/total/summary，并通过 typed `onAcceptedSnapshot` 回调通知页面顶部指标。保证表格、顶部、底栏、导出都引用同一个被采纳的结果。
- 复用现有 WeakMap 机制因为 ProTable onLoad 的公开强类型签名主要只保证 data 数组（`es/table/typing.d.ts:233`）；运行时第二参数 extra 虽包含元数据，但某些签名是 any，不必为此绕类型或依赖未承诺细节。
- 失败、取消和迟到结果不触发采纳；新请求期间保留上一份成功快照并遵循现有表格 loading，不能把请求失败显示为全量 0。
- 禁止新增 sequenceRef 等手工竞态令牌；继续官方 request、onLoad、actionRef.reload。

#### 4.3 是否需要 Proto 改动

- 推荐方案不改 Proto、后端业务聚合、数据库和生成物。
- 如果用户要求底栏继续显示**全部记录的原币金额**，现有 summary 没有相关字段，必须正式扩展对应全量汇总契约，生成 Go/OpenAPI/客户端，补后端全量与币种测试；不能循环翻页拼全量。
- 如果只想底栏全量本币，可直接组合现有 summary，但六类不同状态口径和页面顶部信息重复，仍需准确的 entity adapter 与文案；这不是保留原底栏原币语义的等价替代。

### 5. 最小验收建议

- 让数据大于一页，翻页时顶部全量指标不变，底栏“本页”跟随当前行变化，已选金额仅跟选择变化；明确记录数各自匹配其口径。
- 六类实体按真实字段投影，不能让 amount/baseAmount 或无 direction 的记录静默变成 0；非费用不出现“利润”。
- 两种本币在同页/跨页选择时独立显示，不能相加后标最后一条币种。
- 原币/本币、optional 未折本币、草稿/取消/反转等行与已确认/有效全量指标的差别有明确定义。
- 实施后用同一真实框架/deferred 顺序重放 A→B 乱序；表格和指标都保留 B，取消 A 不影响导出和底栏。
- 请求失败保留上一成功快照并显示原有错误，不回退到 0；首次无成功快照不能伪造成功零汇总。

### 6. 相关规范与文件

- `.trellis/spec/web/frontend/state-management.md`：服务端状态、ProTable 官方请求豁免、错误响应与快照隔离。
- `.trellis/spec/web/frontend/component-guidelines.md:121`：官方 request/actionRef 范式；金额格式规范。
- `.trellis/spec/web/frontend/type-safety.md`：ProtoJSON 零值/optional presence 与领域映射。
- `FinanceLedgerTemplate.tsx`：公共模板已存在的采纳快照及导出边界。
- `FinanceSummaryBoard.tsx`：当前费用专用聚合被错误用作六类通用聚合。

## Caveats / Not Found

- 本轮没有浏览器点击复现或真实数据库查询；汇总范围依据正式 Proto 和 repository 聚合代码，竞态依据真实安装框架 hook 的确定性实验。
- 底栏到底保留“全量原币”还是采用“已选/本页 + 顶部有效本币”是产品口径选择，**需要用户确认，推荐不能视为默认同意**。除该选择外没有发现新的阻塞产品决策。
- 研究没有修改任何产品代码、现有测试、生成物、数据库或 Git 状态；仅在本任务 research 下写入证据与可执行研究实验。
