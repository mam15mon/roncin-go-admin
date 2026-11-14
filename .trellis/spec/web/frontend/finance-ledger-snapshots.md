# 财务列表采纳快照与汇总范围

## 1. 适用范围
费用、账单、发票、收付、核销、对冲六类列表，以及复用底栏的整单费用页。修复当前页误称全部、业务指标乱序覆盖和领域字段错用；新增同类列表遵循本契约，不另造分页/请求/选择平台。

## 2. 正式契约与公共边界
- 六列表 List*Response 新增 `matchedSummary: FinanceLedgerMatchedSummary`，`count` 等于列表 `total`，`amounts` 是分页前全部匹配记录的方向/原币/本币金额桶。
- 桶字段：`direction`（RECEIVABLE/PAYABLE，对冲无方向）、`currency`、decimal 字符串 `amount`、optional `baseCurrency/baseAmount`、`unconvertedCount`。
- 业务 `summary` 仍负责有效费用/已确认/已开票等顶部指标；`matchedSummary` 与完整列表筛选一致，不再额外过滤业务状态。
- 公共签名：`FinanceLedgerTemplate<T, TFilter, TSummary>` 接收
  `onAcceptedSnapshot(snapshot)`、`summaryConfig`、`selectionScope`；
  `FinanceSummaryBoard({ summary, selected, config, scopeLabel? })` 只展示已明确的范围。
- 公共模板 `components/ui/finance-ledger-template` 只接收稳定展示模型；领域行与汇总投影由 `features/finance/ledger-summary` 的各业务适配器承担，禁止通用组件依赖页面/服务端枚举并猜测同名字段。

## 3. 数据与选择约定
公司范围统一来自[当前公司工作区](./finance-current-workspace.md)，不作为独立可选或可恢复筛选；公司切换依赖既有工作区重建隔离查询和选择。

1. 列表行、顶部指标、底栏筛选合计、当前页导出只能来自 ProTable 官方已采纳结果。`request` 负责返回数据及两种 summary，禁止 await 后直接 setMetricStats/setSummary。
2. 扩展既有返回数组 WeakMap + `onLoad` 快照机制；失败、取消、迟到请求不得推进快照，不引入 sequenceRef 等手工竞态链，不绕过官方 request/actionRef。
3. 底栏只显示一行：有选择显示“已选 N 条”，无选择显示“筛选结果共 N 条”。后者必须消费服务端全量匹配汇总，不能用本页代算或循环翻页。
4. 原币与本币分别按币种分组，金额用 decimal 累加并由 formatAmount 展示。缺折本币保留未折算语义，不补 CNY，不把缺失当零。费用台账筛选明细含历史/失效行时称“费用明细差额”，不冒充有效业务利润；
   整单有效费用入口可称费用利润。收付/核销/对冲展示各自语义。
5. 翻页、页大小和排序保留选择；已提交的筛选（含关键字、日期）或公司变化立即清空已选键和记录，未提交输入不清空。表头全选仅本页，跨页选中行共同计数和合计。
6. 复用原生 preserveSelectedRowKeys，公共 onChange 必须与调用方组合，不能被对象 spread 覆盖。跨页缓存为用户已选快照，不声称后台实时重读；已采纳当前页同 ID 记录应刷新对应显示值。
7. 顶部明确业务状态口径且跟随完整筛选。对冲与其他列表一致：先按全部列表条件过滤，再统计已确认金额；筛选草稿时已确认金额为零。
8. 首次失败不能显示伪造成功零汇总；已有成功快照则保留上次成功显示并提示错误，不因失败清空金额。
9. 费用的财务进度沿用关联账单口径，开票与实际收付款分别判断。没有有效收付款
   核销时，纯对冲仍显示“未付款”；发生核销后，对冲后的剩余余额付清才显示
   “已付款”，否则为“部分付款”。对冲金额单列，不能称为核销或付款。共享文案、
   筛选、行配色和 CSV 一次同步，后台规则见[账单币种与付款进度](../../server/backend/finance-bill-currency.md)。

## 4. 验证与错误矩阵
| 场景 | 必须结果 |
| --- | --- |
| 数据超过一页，无勾选翻页 | 筛选合计不变，金额/计数来自全部匹配记录 |
| 第1页选2条，第2页选3条 | 已选5条且金额是这5条，不只算本页 |
| 修改筛选/公司，新请求失败 | 选择仍立即清空，不恢复旧范围记录 |
| 多原币/多本币 | 分组展示，不能混加后标最后币种 |
| 发票本币缺失 | 原币仍统计且显示未折算数，显式本币0保留为零 |
| B先完成，A后完成 | 表格、顶部、底栏和导出都保持B |
| 成功后失败/取消 | 不覆盖成功快照；首次失败不冒成功零值 |
| 草稿、取消等筛选命中行 | 底栏明细合计包含所列行，顶部仅按明确业务状态统计 |

## 5. 正常、基础与错误案例
- 正常：筛选100条跨三种原币，底栏无选择时显示100条合计，跨页选5条后仅显示5条。
- 基础：空筛选结果成功返回count=0，币种集合可省略，显示真实空合计。
- 错误：request中直接setState，使旧请求覆盖新指标；把currentData当全部；用费用的totalAmount读取核销amount。

## 6. 必需测试
确定性 deferred 真实框架竞态、空/失败/取消、跨页勾选和筛选清空、真实六类字段投影、金额精度与多币种、发票未折算；重型suite按功能拆分。后端按完整筛选谓词验证分页独立、组织范围、状态、未折算与count=total；生成物经正式生成。

## 7. 错误与正确
```tsx
// 错误：请求完成不等于表格采纳。
const response = await list(params);
setMetricStats(response.summary);
return response;

// 正确：请求无指标副作用，官方 onLoad 采纳后通知指标消费者。
const response = await list(params);
return { ...toTableRequest(response), summary: response.summary,
  matchedSummary: response.matchedSummary };
```

## 费用台账主单号

- `FeeLedgerItem.masterNo` 由服务端批量投影海运 MBL / 空运 MAWB，列表与订单费用详情共用同一读取规则；前端直接消费生成字段。
- 活动订单读取活动关联；终态仅读取结束原因和结束时间与本次终止匹配的保留关联，不取旧改配或往次终止关系。
- 「主提单号」通过独立 `masterNo` 参数模糊筛选，与其他筛选条件共同生效；综合 `keyword` 同样覆盖主单号。空运搜索支持带连字符的 MAWB 输入，纯连字符不能变成匹配全部。
- 主单号筛选在分页与汇总之前执行，并保留当前公司范围。回归入口：`fee_ledger_master_no_integration_test.go`（真实 PostgreSQL）、`fee_ledger_balances_test.go`（ProtoJSON）、费用台账的 `fee-query-state.test.tsx` 与 `feeLedgerProgress.test.tsx`。

## 筛选项设置（移除清值并刷新）

- 页面通过 `components/ui/filter-settings` 声明可选字段与默认可见集合（默认等于既有显示集合），经 `SearchFilterTemplate.visibleNames` 生效；公司工作区与权限范围不是筛选字段，不可通过设置移除。
- 字段由可见变不可见：清除其表单值、已提交条件与候选 label（范围字段起止值同清），仅当该字段已有已提交值时，沿用页面 `onSearch` 路径（`commitFilters` + `reload(true)`：回第一页、清空勾选）用剩余已提交条件刷新；不连带提交其他未提交输入；重新加入不恢复旧值。列表、指标、合计、导出沿用既有 `onLoad` 采纳快照，不在 request 完成时直接更新指标。
- 普通折叠/展开只改变展示：折叠而未挂载的字段值并入提交值，不改变查询。
- 偏好 key `roncin:filter-settings:v1:<pageKey>:<userId>:<orgId>`，只存可见/隐藏字段名；未知字段忽略，偏好未提及的新字段按声明默认补入；身份未就绪不读不写；存储失败提示一次，当次仍生效。
- 页签恢复经 `useListQueryState` 的 `sanitizeRestored` 剔除已隐藏字段的值。

- 筛选设置移除字段后执行重置，须对全部声明字段清值并显式清除隐藏字段的初始值，不能让 `Form.resetFields` 在字段重新显示时复活旧条件；范围字段起止值一并验证。
