# 基础框架修复计划

> 本文件记录已确认的基础框架待办项。每完成一项勾选验收项并在提交说明中
> 记录验证结果；新增待办须注明问题来源与优先级。规范依据见 `AGENTS.md`
> 的「数据库事务与并发控制规范」一节。

> 2026-08-30 基线复核：原始计划从 `change.zip` 并入仓库后，按当前代码重新
> 核验完成状态。事务封装已落地；排除统一封装、生成代码与测试后，仍有 114 处
> 生产仓储 `Tx(ctx)` 分布在 27 个文件中。已完成项只按源码、生成校验或测试
> 可以直接证明的结果勾选。

## 待办一：统一事务封装（高 · server/data）

**问题**：全项目共 132 处手动事务模板（`db.Tx(ctx)` 加逐点
`tx.Rollback()` 再 `tx.Commit()`），分布在 37 个文件；另有 561 处
`tx.Rollback()` 调用。样板量大、遗漏 Rollback 的风险高；
`internal/data/auth.go:47` 的 `RecordLoginFailure` 还混用了
`sqlDB.BeginTx` 原生事务，风格不一致。

**方案**：在 `internal/data` 提供统一封装 `WithTx(ctx, fn)`——封装内部
负责 Begin、出错自动 Rollback、panic 安全回滚、成功 Commit。新增代码
一律通过封装调用事务；存量手写事务在触碰对应文件时逐步迁移，迁移进度
在本文件更新。

**验收**：
- [x] `internal/data` 提供 `WithTx` 封装并附单测（覆盖出错回滚、panic
      回滚、成功提交三条路径）
- [ ] 新增仓储代码零手动 `db.Tx` 分散写法
- [ ] 存量 132 处完成迁移或分批列入后续计划
- [x] `internal/data/auth.go:47` 的原生事务写法并入封装

迁移记录：

- [x] 第一批：`finance_custom_setting.go`、`order_attachment.go`、
      `order_milestone.go`、`partner_attachment.go` 共 4 个事务入口。
- [x] 第二批：`partner_settlement_rule.go`、`partner_invoice_profile.go`、
      `partner_shipping_preset.go` 共 6 个事务入口。
- [x] 第三批：`finance_cashflow.go`、`finance_verification.go` 共 4 个事务入口。
- [ ] 剩余生产仓储手写事务：114 处 / 27 个文件。

## 待办二：日志级别可配置（高 · server/platform）

**问题**：`cmd/server/main.go` 将日志级别硬编码为 `slog.LevelInfo`，
`configs/` 三份环境配置与 `internal/conf/v1/conf.proto` 均无日志配置段。
直接后果：`configs/config.yaml` 中 `data.database.debug: true` 本意输出
ent 查询日志（Debug 级），但被 Info 级别整体过滤，该开关实际失效。

**方案**：`internal/conf/v1/conf.proto` 新增日志配置段（至少含 `level`），
执行 `make config` 重新生成；`cmd/server/main.go` 依据配置初始化 logger；
`config.yaml`（开发）默认 debug 使 `database.debug` 恢复作用，
`config.production.yaml` 保持 info 与现状一致。

**验收**：
- [x] 修改配置文件日志级别后重启即生效，无需改代码重新编译
- [x] `database.debug: true` 且级别为 debug 时，ent 查询日志可见
- [x] 生产配置默认 info，行为与现状一致

---

# 重复开发治理（2026-08 审计）

按模块区分。共性规律：正确范式已存在但覆盖不全，修复策略是**推广既有
范式**（`permissions.generated.ts`、`allowedActions`、`PagedList[T]`、
`writeAudit`），不发明新抽象。

## 模块 A：前后端契约（高 · 已含 5 处实际漂移）

### A1. 先修已漂移项（quick fix，不依赖生成链路）
- [x] `web/src/pages/orders/common.ts:163` `PARTNER_ROLES` 中
      `BOOKING_AGENT: 2` 与 `SUPPLIER: 2` 同值冲突（proto 无 BOOKING_AGENT）
- [x] `web/src/pages/orders/common.ts:81` seaServiceTypes 前端 16 项 /
      后端 19 项（`biz/masterdata.go:81`），缺 3 项且
      `requireSeaServiceTypeOptions` 遇缺项抛错
- [x] `web/src/pages/finance/bills/components/billConstants.ts` 前端多出
      INVOICED/VERIFIED 两态，后端仅 DRAFT/CONFIRMED/CANCELLED
- [x] `web/src/pages/settings/components/number-rules/numberRulesConstants.ts`
      前端默认前缀 IV/HB/REF/CREF，后端 `biz/orderconfig.go:110` 无前缀

### A2. proto 枚举生成链路（根治，消除 30+ 处手抄）
**根因**：`web/config/openapi.generated.json` 把所有 proto enum 输出为
裸 `integer`，前端 `typings.d.ts` 只见 `number`，被迫手抄枚举映射。
**方案**：仿 `scripts/generate-permission-keys.mjs` 先例，新增脚本从
`server/api/**/*.proto` 生成 `web/src/enums.generated.ts`（枚举名 →
数值常量 + 联合类型），与 `pnpm run generate:web-client` 同一提交；
前端仅保留「枚举名 → 中文文案/颜色」的展示层映射。
**验收**：
- [x] 生成脚本接入根 `package.json`，与 web-client 生成串联
- [ ] 订单/费用/账单/发票/伙伴/后台任务等手抄枚举改用生成常量
- [ ] finance 域 stringly 状态（`biz/finance_bill.go:37`、
      `biz/settlement.go:20`）提升为 proto enum 后纳入同一链路

### A3. 状态标签三份真相收敛
**问题**：状态文本/颜色同时存在于 proto 枚举、DB 种子
（`migrations/20260823200000_default_status_templates.sql` 的
status_template_items，含 label + color_token，本就按组织可配）与前端
常量，已互不一致（SE 的 DRAFT 在 DB 叫「新建」，proto/前端叫「草稿」，
节点集合也不同）。
**方案**：前端状态标签/颜色改为消费后端 `status_template_items` 接口。
- [ ] 订单状态标签改由 status template 驱动，删除前端 `seStatusTabs`
      等手抄颜色表

### A4. 状态机流转改为服务端下发
**问题**：3 处状态机与后端逐边复制——`TransitionModal.tsx:26` ≡
`biz/order_transition.go:42`；`release-pod-panel.tsx:52` ≡
`biz/order_release_pod.go:40`；`ContractsPanel.tsx:59` ≡
`biz/partner_contract.go:120`。后端改一条流转，前端静默不同步。
**方案**：仿订单详情 `allowedActions`（`detail.tsx:236`）先例，响应中
下发 `allowedTargetStatuses`，前端下拉只做展示。
- [ ] 订单流转、放货凭证、伙伴合同三处改造完成

### A5. reason 错误码收口
**问题**：前端 11+ 处硬编码 reason 字符串做控制流（如
`use-fee-exchange-preview.ts:76`、`finance/commissions/index.tsx:131-228`），
而 reason 真相在 `errors.Conflict("...")` 字符串里，不在
`error_reason.proto`，改名即静默失效。
**方案**：service 层错误 reason 统一取自 proto ErrorReason 值，生成
`web/src/errorReasons.generated.ts` 供前端分支判断。

## 模块 B：后端 server

### B1. service 层响应封套（高，215 处 / 31 文件）
`&v1.XxxResponse{Success: true, Code: 0, Message: "OK", ..., TraceId:
requestmeta.TraceID(ctx)}` 同构重复，封套字段调整需改 200+ 站点。
- [ ] 提供泛型 `ok[T](ctx, data)` / `okList[T](ctx, items, total, page,
      pageSize)` helper 并全量替换

### B2. ent 错误映射统一（高，213 处）
NotFound/Constraint 映射样板 168+45 处；10 个仓储已各自私写同构 helper
（`mapOrderConstraint`、`mapPartnerConstraint` 等），其余约 30 个内联。
- [ ] 统一 `mapEntError(err, notFoundErr, constraintErr)`，替换 10 个
      私有版本与全部内联点

### B3. 分页解析与类型收敛（中高）
5 份逐行相同的 `xxxPageValues`（admin/background_task/enterprise_resource/
order_tag/partner，默认页大小均 20）+ 8 处散调；泛型 `biz.PagedList[T]`
已存在但 11 个逐域手写 `XxxList` 并存。
- [x] 通过 `biz.ListPagination`、`biz.ValidListPagination` 与 service 层
      `listPageValues` 统一分页默认值和边界校验
- [x] 11 个手写列表结构体迁移到 `PagedList[T]`

### B4. data 层 List 查询样板（中高，16-21 处 / 8 文件）
Count → Offset/Limit → 循环转换 → 组装 五步同构
（`industry_reference.go` 4 份、`fee_catalog.go` 3 份等）。
- [ ] 抽 `data.paginate[E, T](ctx, countFn, itemsFn, page, pageSize, conv)`

### B5. Principal 提取三连（中，151 处 / 31 文件）
`PrincipalFromContext + if !ok + ErrSessionRequired` 完全一致。
- [ ] `biz.RequirePrincipal(ctx) (*Principal, error)` 一行替换

### B6. 微样板收口（中低）
- [ ] UUID helper 5 份拷贝（`parseUUIDs`/`parseUUIDStrings`/`parseUUIDList`/
      `enterpriseParseUUIDs`、`formatOptionalUUIDString` ≡ `uuidStringPtr`）
      收口到 service 包 `idutil`
- [ ] 26 处 `decimal.NewFromString` + err 三连抽 `data.decimalOf(s)`

### B7. cmd 同步工具框架（低）
`sync-airports` 与 `sync-unlocode` 高度同构（options 结构体、main 流程、
`fail()`、`openStore()` 逐字重复）；`sync-regions` 裸 SQL 未复用
`IndustryReferenceSyncStore`。
- [ ] 抽共享 `syncRunner` 框架；sync-regions 迁到 SyncStore

## 模块 C：前端 web

### C1. 响应解包与表格 request 样板（高，107 处 / 59 文件）
`.data ?? []` 75 处与 `.data || []` 32 处两种风格并存。
- [ ] `utils/api.ts` 提供 `unwrapList` / `unwrapPage` / `toTableRequest`
      并统一 `??` 写法

### C2. 状态映射集中（高，27 文件 / 50+ 张表，同一状态 3 种键制）
费用状态同时存在 string 键（`feeLedgerColumns.tsx:16`）与 number 键
（`finance/fees/detail.tsx:39`）且文案不同；业务类型字典两处键制不同。
- [ ] 建 `constants/statusMeta.ts` 单一键制（依赖 A2 的枚举生成），
      提供 `statusTag()` 渲染器与 `makeValueEnum()` 工具

### C3. 格式化工具收敛（中高）
日期空值守卫 `val ? dayjs(val).format(...) : '-'` 重复 52 处 / 29 文件；
金额格式化 4 套实现（`utils/format.ts` 已有但仍被绕过）；小数去尾零
2 套；`ExchangeRatesPanel` 跨页面目录相对引用 `order-fee-decimal`。
- [ ] 扩展 `utils/format.ts`：`formatDate(v, 'datetime'|'date')`、
      `formatAmount`、`trimDecimal`；`order-fee-decimal.ts` 迁入 utils

### C4. 财务单据 confirm/cancel 复用（中高，7 页逐行重复）
`src/pages/orders/fee-reason-confirm.tsx` 的 `confirmWithReason()` 已存在，
finance 五页全部绕过自写；应收/应付聚合在 5 个页面重复且**只统计当前页**
（分页后口径错误）。
- [ ] `confirmWithReason` 提升到 `src/utils/`
- [ ] 抽 `makeVersionActions()` 与 `aggregateByDirection()`，修复当前页
      口径问题

### C5. 选项加载收敛（中，10 文件 / 13+ 处）
合作方下拉 10 处重复且 label 3 种格式；币种 options 4 种实现（含 2 处
硬编码列表）；公共函数 `searchPartnersByRole`（`orders/common.ts:205`）
已存在但 finance 全部绕过。
- [ ] `utils/options.ts`：`usePartnerSearch(role?)`、
      `getCurrencyOptions()`（模块级缓存）

### C6. partners 双轨 CRUD 收敛（中）
旧手写套（AccountCardList/ContractCardList/ShippingPresetSection 等 6 个）
与新模板套（secondary/*Panel 用 SubEntityDrawerTemplate）并存，合同状态
映射因此两份；`ContactCardList.tsx:18 ContactItem` 与
`API.PartnerContact`（typings.d.ts:3893）类型漂移。
- [ ] 旧卡片组迁移到 SubEntity 模板，或至少共享状态映射与 fetch 骨架

### C7. 详情抽屉骨架统一（中低）
4 套手写 Drawer + Descriptions 骨架，props 契约两种。
- [ ] 抽 `DescriptionsDetailDrawer`（统一 open/detail/loading 契约 +
      空值兜底 `DItem`）

### C8. react-query 决策（低 · 需确认方向）
依赖已装、插件已开但全项目 0 处使用；4 个选项加载 hooks 结构雷同。
二选一：启用 react-query 统一缓存/重试（对齐 CLAUDE.md 声明），或删除
依赖与插件（对齐实际）。短期可先抽 `useAsyncOptions(loader, deps)`。

### C9. 费用列定义合并（低）
`orderFeeColumns.tsx` 与 `order-fee-panel-columns.tsx` 约 80% 相同。
- [ ] 抽 `feeBaseColumns(opts)` builder，两个入口只传差量

---

## 待评估项（优先级待确认）

以下问题来自 2026-08 基础框架检查，尚未排期：

- CI workflow 仍为 Kratos 模板旧版（Go ^1.20），未接入 `pnpm run check`
  （proto lint / vet / govulncheck / 前端 lint 与测试均不在 CI 中）。
- 前端请求无超时配置，慢请求会无限挂起（`web/src/app.tsx`）。
- 访问日志缺 HTTP 状态码、路径、客户端 IP、UA；无慢查询日志。
- `internal/data/data.go`、`internal/data/wecom.go` 两处混用 Kratos 全局
  log，未用注入的 slog logger。
- `web/config/proxy.ts` 仅导出 `dev`，`start:test`、`start:pre` 脚本
  实际无法代理到后端。
- `data.redis` 配置存在但代码零使用（死配置）。
- 跨仓储用例无共享事务边界（如订单号分配与订单落库非原子），需结合
  事务封装一并设计。
- `web/types/index.d.ts` 为 Ant Design Pro 模板遗留 mock 类型，建议删除。
