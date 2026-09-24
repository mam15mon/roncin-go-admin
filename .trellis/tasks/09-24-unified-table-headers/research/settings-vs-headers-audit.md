# 列设置与真实表头一致性审计与修复记录（2026-09-24）

## 审计方式

对全站 70 个 `useColumnSettings` / 统一列设置消费文件做逐文件审计，核对
「设置面板条目集合」与「表格真实表头受管列集合」是否一一对应。审计维度：
同源性、title 可解析性、key 唯一性、结构列覆盖、分组表头、双入口、名称一致、
动态列。审计结论已全部人工复核关键现场。

## 机制层面的静默失效规则（useColumnSettings）

- title 解析不出纯文本（ReactNode/函数）→ 列从设置面板消失，但仍是受管列，
  存在已存偏好时被挪到表尾；
- 列无 key 且无 dataIndex（纯 render 列）→ 不进设置面板且不受管理；
- 两列 key/dataIndex 重复 → 应用偏好后第二列被直接丢弃；
- structuralKeys/lockVisibleKeys/defaultHiddenKeys/titleOverrides 按 key
  精确字符串匹配，拼错静默失效。

## 发现与修复（四个提交）

### 1. 费用台账两套口径（最严重，b2cc9b08）

`fields-meta.ts` 静态维护 153 项，实际表格 38 列：结算单位
（settlementPartyName≠settlementPartyId）、所属公司（orgName≠organizationName）
key 错位；「费用状态」条目实际控制「财务进度」表头；标签/费用状态列缺条目；
约 120 个幻影字段可勾选但永不渲染。保存偏好后标签/所属公司/结算单位/费用状态
四列被 `buildUserOrderedColumns` 兜底挤到宽表末尾且设置无法再隐藏。

修复：元数据以实际受管列为准重写为 35 项（key/名称/宽度一一对应），
导出更名 `FEE_LEDGER_FIELDS`；规范化映射补充 settlementPartyName→
settlementPartyId、orgName→organizationName（弹窗与列应用两侧同步），
存量服务端偏好即时命中正确列。服务端仅校验 key 格式，无枚举依赖。

### 2. 死入口（7ed80450）

异常协同、订单费用面板、放货回单、自拼汇总（含嵌套成员表）、单证抽屉、
补录费用共 6 文件 7 表只渲染 `entry` 未渲染 `modal`，点击齿轮无响应，
遗留偏好单向生效无法清除。全部补齐 modal 渲染。

### 3. 纯渲染列补 key（a689c831）

14 个业务列因缺 key/dataIndex 从设置面板缺席（账单详情费用折本币/关联状态、
开票候选金额、发票关联、对冲分摊数、核销分配数/关联明细/状态、结算账户默认
用途、变更内容摘要、企业资源备注类型/标签组/文件/大小）。逐一补稳定 key。
汇率同步「买卖点差」表头为 Tooltip 包裹 ReactNode，改由 `titleOverrides`
提供纯文本名。企业资源各页签列集合不同，tableKey 按页签隔离。

### 4. 结构列与入口治理（eafea8f5）

6 处带 key 操作列 + 3 处序号列（dataIndex 'index'）此前可进设置面板被隐藏/
拖离锚点，统一显式声明 structuralKeys；订单列表操作列补 `key: 'option'`
使既有死配置生效；参数设置与子实体抽屉模板操作列补 key 并显式声明。
台账模板默认 CSV 导出改用应用设置后的生效列。开票抬头面板补
`options.setting: false` 消除双入口。客商列表偏好按角色视图 + 黑名单口径
隔离。订单费用表外部注入列时禁用入口并隐藏弹窗。

## 未动项与理由

- 财务费用页 `financial_progress`/`customerName` 历史兼容映射保留（存量偏好）。
- `SplitAllocationSection` 货物分配表多实例共享 tableKey、`BillGroupCard`
  多卡片同 key：同结构表共享偏好属设计意图，仅存在重挂载前短暂不同步，不改。
- `SeaSharedContainerAllocationTable`「快捷操作」列可隐藏：业务操作性质列，
  设计选择非错配。
- 提成规则抽屉开启 ProTable search 时列排序联动搜索表单字段顺序：ProTable
  官方行为，不在本次范围。
- `PendingDecreasePanel`「原提成」、`CommissionRuleRosterModal`「记录时间」
  列名与内容语义不符：面板名与表头一致，不构成设置错位，另立任务处理。

## 验证

- `pnpm --dir web tsc` 通过；`pnpm --dir web test:changed` 603 通过；
- 改动文件 Biome 检查通过；`git diff --check` 干净；
- 全栈门禁 `pnpm run check:fast` 见任务日志。

## 追加：保存偏好 400 修复（2026-09-24 第二轮）

用户保存台账列设置报「费用明细表头设置不合法」。服务端
`normalizeFeeLedgerPreference` 有两条契约被前端违反：

1. 排序字段为空时不得携带排序方向（biz L172-175），而弹窗保存始终发送
   `sortDirection: 'DESC'`——默认不选排序字段时保存必然 400；
2. 存量偏好的 `sortField` 指向已删除扩展字段、钉住列或旧 key 时，过不了
   「sortField 须在提交列内」校验。

另发现同路径隐患：仓储层乐观锁要求「已存在偏好时 version 必须匹配」，
弹窗从不传 version，第二次保存起必然 409。

修复（TableColumnConfigModal）：`sortDirection` 跟随 `sortField` 有无；
打开弹窗时按当前字段清单清洗存量 sortField（含旧 key 规范化）；保存
载荷回传 `version`。新增保存载荷定向测试（4 用例）锁住全部路径。

教训：迁移验证清单里「保存成功」只在单测 mock 层验证过调用发生，没验证
载荷与服务端校验规则逐条对齐；涉及服务端契约校验的提交路径，联调验收
必须真实请求一次。

## 追加：台账偏好去服务端化（2026-09-24 第三轮，用户决策）

用户决策：台账偏好不再做服务端存储，与其他表格统一为浏览器本地。

- 服务端契约整体移除：settlement.proto 删除 Get/Update/ResetFeeLedgerPreference
  三个 RPC 与偏好消息组；biz/service/data 层用例、仓储、Ent Schema
  （FinanceFeeLedgerPreference 及 user/organization 边）删除；新增迁移
  20260924100000_drop_finance_fee_ledger_preferences.sql 删表（dev 库已应用）。
- 前端：新增 `pages/finance/fees/feeLedgerViewPreference.ts` 本地存取
  （key `roncin:fee-ledger-view:v1:<用户>:<组织>`），列显隐顺序与行配色
  按 FinanceLedgerViewConfig 形状存储；TableColumnConfigModal 改为回调保存
  （写入失败保持弹窗并报错）；buildUserOrderedColumns 改消费 {order, hidden}。
- 顺带清理死设置：高级页原「分页大小/默认排序」保存后从未被任何代码消费
  （ListFeeLedger 契约无排序参数，模板分页硬编码 40），本次一并移除，
  高级页只留行配色。存量服务端偏好不迁移（无兼容契约要求）。
- 模板 rowColors/getRowStatusColorKey 类型从 API.FeeLedgerRowColors（已删）
  切换为 RowColorsConfig。
