# Research: R6 主单号 / R7 统一筛选设置

- **Scope**: internal（只读代码 + 开发库只读 SQL，未输出凭据）  **Date**: 2026-11-14

## R6 费用台账主单号
结论先行：R6 的修复已在提交 904343f6（2026-11-13 09:16，"fix: 补齐费用明细台账主单号展示与筛选"）落地；该提交早于本子任务 prd 创建后的规划。本调研仅核实现状，prd 中"尚未确定根因"已过时。
- 列：`web/src/pages/finance/fees/components/feeLedgerColumns.tsx:108-117` 绑定 `masterNo`，空值 `-`。
- 筛选：`FeeLedgerSearchFilter.tsx:153` 的 `masterNo`；`index.tsx:364` 传 `masterNo: filterParams.masterNo || undefined`。
- proto：`server/api/finance/v1/settlement.proto` 在 904343f6 新增 `master_no`（FeeLedgerItem + 请求筛选）；DTO 映射 `server/internal/service/settlement_fee_ledger.go:31`（请求）、`:101`（响应 `MasterNo: item.MasterNo`）。
- 投影：`server/internal/data/fee_ledger_master_no.go:64-107` `enrichFeeLedgerMasterNos`，批量查 `SeaMasterBillOrderLink`(+MasterBill) 与 `AirMasterWaybillOrderLink`(+MasterWaybill)，map 按订单 ID 合并；调用点 `settlement.go:625`（列表）、`:388`（订单费用详情）；筛选谓词 `fee_ledger_master_no.go:39-62` 在 `settlement.go:421`(keyword)、`:434`(masterNo)。
- 读取规则 `:21-37`：订单业务类型须与关联表匹配（SE↔海运，AE↔空运）、link.organization_id 与订单一致；活动订单取 `status=ACTIVE`，终止订单仅取 `status=ENDED` 且 `ended_reason`=终止原因且 `ended_at`=订单 `terminated_at` 的关系（不取改配遗留）。主单组织须等于 link 组织（`:90`、`:99`）。
- 空运搜索会去掉关键字中的 `-`（`:48`）；纯连字符→airKeyword 为空，仅查海运。
- 回归测试：`fee_ledger_master_no_integration_test.go`、`service/fee_ledger_balances_test.go:31`（ProtoJSON 空值省略 masterNo）。spec：`finance-ledger-snapshots.md:65-70`。
- 前端导出：`ledgerExport.ts` 无 masterNo 引用（grep 未命中），导出列是否含主单号未核实。
- 历史根因（据提交内容推断，非实测）：修复前 proto/DTO/投影均无 masterNo，前端只有列无数据。
- 开发库只读 SQL（表名复数 `sea_master_bill_order_links` 等）：sea 链接 ACTIVE 9 / ENDED 6；air 链接 0；`order_fees` 共 16 行，16 行订单有 ACTIVE 海运关联、0 行空运；多 ACTIVE 链接订单 0；ACTIVE 主单号为空 0；链接/主单组织不一致 0。**未发现"有关联但台账为空"的样例**；开发库也无空运样例、无无主单费用（无法实测空/无主单表现，仅有测试覆盖）。
- 最小修复点：无需再修；若用户仍复现，先确认运行二进制/OpenAPI 是否含 904343f6，再看该订单是否落在 `:21-37` 的活动/终止规则之外（如终止单 ended_at 与 terminated_at 不等）。

## R7 现有机制
### 列偏好（`web/src/components/ui/column-settings/`）
- `preference.ts:12-24` key=`roncin:column-settings:v1:<tableKey>:<userId|anonymous>:<organizationId|default>`；`:28` load、`:63` save 返回 bool、`:80` clear；仅存 order/hidden/freeze。
- `useColumnSettings.tsx:208-222` scope 默认取 initialState 用户+当前组织；`:326-336` 存储失败 `message.error('列设置保存到本地浏览器失败，本次设置仅当前页面生效')`（每 identity 仅提示一次）。公开面 `index.ts`。
- 不含"页面"维度的独立参数，由 `tableKey` 承担；类型 `types.ts` 的 `ColumnSettingsScope`。

### 搜索区实现（`web/src/components/ui/search-filter-template/`）
- `SearchFilterTemplate.tsx`（641 行）：`items: SearchFilterFieldItem[]`（types.ts:14-43，含 name/label/type/span/initialValue/request）；折叠仅 `items.slice(0, defaultVisibleCount)`（:155-160），form store 值保留→折叠保留条件天然成立；`committed` 状态+chips（:139-146、:262-）；`deleteChip`（:196-215）已实现"清字段值+剔出已提交条件+`onSearch(next)`"，可作为移除字段清值刷新的参照；`handleReset`（:176-194）。
- 不存在 SearchForm/QueryFilter 的"字段增删"能力：需要 `items` 外层的可见字段集合 + 设置入口（新增）。chips 与 `items` 绑定（:240-），隐藏字段需从 `items` 剔除后其 chip 不显示，且已清值。
- `bar` 模式（keyword+quickFilters）与 `custom` 模式不产生 chips，需单独考虑。

### 财务页筛选清单（范围依据：routes.ts:206-280，/finance 下）
| 页面 | 实现 | 可选字段（现状） |
|---|---|---|
| fees | `FeeLedgerSearchFilter`→SearchFilterTemplate grid，经 `FinanceLedgerTemplate customSearch`+`search={false}`（fees/index.tsx:222-240） | keyword、direction、financialProgress、orderLocked、hasPendingFeeSupplement、settlementPartyId、customerId、billNo、createdAtRange、orderNo、masterNo、houseNo、feeName、invoiceNo、consignee、shipper、businessType、currency、salesName、operatorName、vesselName、voyageNo（FeeLedgerSearchFilter.tsx:15-38） |
| bills | SearchFilterTemplate（bills/index.tsx:404-406） | keyword、direction、status、billDateRange、dueDateRange、onlyUnsettled、onlyOverdue、settlementPartyId、currency（:117-177） |
| commissions（主列表/申请面板/规则抽屉） | SearchFilterTemplate（commissions/index.tsx:691；ApplicationsPanel） | 关键词、状态等少量（申请月、employee 等） |
| cashflows / invoices / nettings / verifications | FinanceLedgerTemplate + ProTable 内建搜索（列 `search` 非 false） | 近似：direction、status、keyword（invoices/nettings）、nettings 另有金额/有效等（依据 awk 粗扫，需逐页核实） |
| exchange-rates / fee-settings | `search={false}` / 非本期列表 | 待确认是否属于"财务列表" |
| 四类财务报表 | 现 routes 无对应页面（由其他子任务新建） | — |
→ 两套实现并存：SearchFilterTemplate 与 ProTable 内建 QueryFilter；统一设置需对二者各给适配，或把后者改走 SearchFilterTemplate（prd 未决）。

### 约束
- AGENTS.md / `finance-ledger-snapshots.md:19,23`：沿官方 request+`onLoad` 采纳快照；已提交筛选变化立即清空选择与记录；迟到请求不推进快照。fees 使用 `useListQueryState`（fees/index.tsx:100-106）+`useLedgerSelectionScope.commitFilters`（:107-123）；`handleSearch` → `commitFilters` + `actionRef.reload(true)`。移除筛选项应走同一 `commitFilters`+`reload(true)` 路径。
- `finance-current-workspace.md:6-14,30`：公司仅来自 `currentUser.currentOrganization`；未就绪/非公司不发请求（fees/index.tsx:172、:337）。公司不得成为可移除筛选项（现无独立公司筛选）。
- 偏好存储必须与 `useListQueryState` 的 tabKey/listId（'finance:fees'）区别：后者存查询值，R7 仅存字段集合。
- 架构检查 `web/scripts/check-architecture.mjs:15,48-49,337`：`components/ui`/hooks/utils 禁止导入 pages 与 features；pages/finance/<模块> 之间禁止互引（:112-120）；features 仅经 `index.ts` 公开。故公共组件放 `components/ui/`（建议靠近 search-filter-template 或 column-settings）且不得引用 `@/features/*`（如 partners 选项请求须由页面传入 items）；若放 features/finance/ 则 ui 层不能引用它。

## 最小 API 草案（文字，仅描述）
- 纯逻辑：`filter-settings/preference.ts` 复刻 column-settings：key=`roncin:filter-settings:v1:<pageKey>:<userId>:<orgId>`，值仅 `visible: string[]`（字段 name，保持页面字段顺序）；load/save(bool)/clear；未知 key 忽略，新增字段按默认可见规则补入（默认可见集需页面声明）。
- Hook `useFilterSettings({ pageKey, fields:[{name,label,required?,defaultVisible}], scope? })` → `{ visibleNames, setVisible, reset, persistFailed }`；scope 默认取 initialState（同 useColumnSettings:220-222）；存储失败 `message.error` 一次，当次仍生效。身份未就绪时不读写（避免写入 anonymous/default）。
- 入口组件 `FilterSettingsEntry`（齿轮+勾选面板，仿 ColumnSettingsEntry）。
- SearchFilterTemplate 扩展：新增 `filterSettings?`（或页面自行按 visibleNames 过滤 `items`）；字段由可见→不可见时，在内部调用与 `deleteChip` 同一套逻辑清值，并调用 `onCommit/onSearch(next)`；重新加入不恢复旧值。仅当被移除字段已有已提交值时才触发刷新。
- 接入：fees/bills/commissions 直接把 items 经 hook 过滤；ProTable 内建搜索的页面需先改为 SearchFilterTemplate 或在列上以 `hideInSearch` 动态控制并手动清 `formRef` 与查询状态（后者与 `commitFilters` 的集成点待设计）。导出/汇总参数沿用既有 `queryState.snapshotRef.current.filters`，因值已在移除时清除，自然一致。

## 风险 / 待用户决定
1. R6 是否按"已修复、仅补验收"处理，还是用户仍可在某环境复现（需订单号）？开发库无空运与无主单样例，空/无主单仅靠测试覆盖。导出是否需含主单号？
2. 非 SearchFilterTemplate 的 4 个页面（cashflows/invoices/nettings/verifications）是否统一迁移到 SearchFilterTemplate，还是为 ProTable 内建搜索另做适配？
3. 哪些字段 `required`/不可移除（如公司范围、权限范围不暴露；关键字是否可移除）；默认可见集合；是否允许调整筛选项顺序。
4. 偏好中已存字段被后续版本移除/改名时的处理（忽略未知项，不做迁移，与"不兼容历史"一致）。
5. 移除字段触发刷新时若当前有未提交的其他输入，是否随之提交（现有 deleteChip 仅用已提交条件，不带未提交输入）。
6. 公共组件放置层：components/ui（受限于不能 import features，选项请求需页面传入）——需确认。
7. 报表页尚未存在，接入时机依赖其他子任务；exchange-rates/fee-settings 是否纳入"财务列表"。
8. `useListQueryState` 页签恢复的筛选值里可能含已隐藏字段，恢复时需同步剔除，否则出现"隐藏条件继续生效"。
