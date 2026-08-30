# 基础框架修复计划

> 本文件记录已确认的基础框架待办项。每完成一项勾选验收项并在提交说明中
> 记录验证结果；新增待办须注明问题来源与优先级。规范依据见 `AGENTS.md`
> 的「数据库事务与并发控制规范」一节。

> 2026-08-30 基线复核：原始计划从 `change.zip` 并入仓库后，按当前代码重新
> 核验完成状态。事务封装已落地；排除统一封装、生成代码与测试后，当时仍有
> 96 处生产仓储 `Tx(ctx)` 分布在 20 个文件中。已完成项只按源码、生成校验或测试
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
- [x] 新增仓储代码零手动 `db.Tx` 分散写法
- [x] 存量 132 处完成迁移或分批列入后续计划
- [x] `internal/data/auth.go:47` 的原生事务写法并入封装

迁移记录：

- [x] 第一批：`finance_custom_setting.go`、`order_attachment.go`、
      `order_milestone.go`、`partner_attachment.go` 共 4 个事务入口。
- [x] 第二批：`partner_settlement_rule.go`、`partner_invoice_profile.go`、
      `partner_shipping_preset.go` 共 6 个事务入口。
- [x] 第三批：`finance_cashflow.go`、`finance_verification.go` 共 4 个事务入口。
- [x] 第四批：`admin_organization.go`、`admin_role.go` 共 4 个事务入口。
- [x] 第五批：`admin_user_membership.go`、`order_personnel.go` 共 5 个事务入口。
- [x] 第六批：`order_cargo_item.go`、`order_container.go` 共 6 个事务入口。
- [x] 第七批：`order_abnormal_case.go` 共 3 个事务入口。
- [x] 第八批：`order_release_pod.go`、`order_shipping_document.go` 共 8 个事务入口。
- [x] 第九批：`order_write.go` 三个订单状态流转事务入口。
- [x] 第十批：`exchange_rate_import.go`、`industry_reference_sync.go` 共 4 个批处理事务入口。
- [x] 第十一批：`exchange_rate.go` 三个汇率停用及配置事务入口。
- [x] 第十二批：`background_task.go` 共 4 个任务领取及状态变更事务入口。
- [x] 第十三批：`masterdata.go` 共 3 个主数据写入及导入事务入口。
- [x] 第十四批：`partner_secondary.go` 共 4 个伙伴账户及合同事务入口。
- [x] 第十五批：`partner.go` 两个伙伴创建及供应商黑名单事务入口。
- [x] 第十六批：`partner.go` 两个伙伴主档更新及批量导入事务入口。
- [x] 第十七批：`exchange_rate.go` 最后一个汇率主保存事务入口。
- [x] 第十八批：`order_write.go` 最后一个订单草稿更新事务入口。
- [x] 第十九批：`finance_invoice.go` 两个发票开具及取消事务入口。
- [x] 第二十批：`finance_invoice.go` 两个发票创建及红冲事务入口。
- [x] 第二十一批：`fee_catalog.go` 一个计费单位创建事务入口。
- [x] 第二十二批：`fee_catalog.go` 三个计费单位更新及应税服务事务入口。
- [x] 第二十三批：`fee_catalog.go` 两个费用项创建及更新事务入口。
- [x] 第二十四批：`business_tag.go` 六个订单、费用及账单标签事务入口。
- [x] 第二十五批：`admin_user.go` 三个用户创建、终止及密码重置事务入口。
- [x] 第二十六批：`admin_user.go` 一个用户编辑事务入口。
- [x] 第二十七批：`admin_user.go` 最后一个外部账号授权事务入口。
- [x] 第二十八批：`auth.go` 三个会话创建、组织切换及撤销事务入口。
- [x] 第二十九批：`auth.go` 两个企业微信及钉钉待授权账号注册事务入口。
- [x] 第三十批：`auth.go` 最后一个钉钉返聘准备事务入口。
- [x] 第三十一批：`order_fee.go` 三个费用新增、状态流转及取消事务入口。
- [x] 第三十二批：`order_fee.go` 最后一个账单联动费用更新事务入口。
- [x] 第三十三批：`industry_reference.go` 港口与机场新增、更新事务入口。
- [x] 第三十四批：`industry_reference.go` 航空公司新增、更新事务入口。
- [x] 第三十五批：`industry_reference.go` 船公司及箱号前缀新增、更新事务入口。
- [x] 第三十六批：`finance_bill.go` 账单更新、确认及取消事务入口。
- [x] 第三十七批：`finance_bill.go` 批量账单确认事务入口。
- [x] 第三十八批：`finance_bill.go` 单账单创建事务入口。
- [x] 第三十九批：`finance_bill.go` 批量账单创建事务入口。
- [x] 第四十批：`finance_commission.go` 提成规则新增、更新事务入口。
- [x] 第四十一批：`finance_commission.go` 提成单创建事务入口。
- [x] 第四十二批：`finance_commission.go` 提成单状态流转事务入口。
- [x] 第四十三批：`finance_commission.go` 提成调整单新增、状态流转事务入口。
- [x] 第四十四批：`enterprise_resource.go` 企业资源新增、更新事务入口。
- [x] 第四十五批：`enterprise_resource.go` 企业资源删除事务入口。
- [x] 第四十六批：`enterprise_resource.go` 企业资源批量合作方关联事务入口。
- [x] 第四十七批：`enterprise_resource.go` 地址类型与负责人批量关联事务入口。
- [x] 第四十八批：`enterprise_resource.go` 标签组新增、更新及删除事务入口。
- [x] 第四十九批：`enterprise_resource.go` 企业资源导入事务入口。
- [x] 第五十批：`permission_manifest_sync.go`、
      `default_order_options_sync.go` 两处原生 SQL 同步事务并入统一生命周期。
- [x] 生产仓储手写事务生命周期已清零：业务函数无显式提交或回滚，Ent 与
      原生 SQL 事务统一通过 `WithTx`、`withSQLTx` 或 `runTransaction` 执行。

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
- [x] 第一批：`finance_invoice.go` 开票与红冲编号约束改用公共
      `mapEntConstraint`，移除私有映射器
- [x] 第二批：`partner_secondary.go` 合同号与默认账户约束按写入场景改用
      公共 `mapEntConstraint`，移除私有映射器
- [x] 第三批：`finance_verification.go` 核销幂等与分配对约束按写入场景改用
      公共 `mapEntConstraint`，移除私有映射器
- [x] 第四批：新增公共有序多约束映射，`partner.go` 主档与子资源按精确
      约束名映射领域错误，移除私有映射器
- [x] 第五批：`finance_invoice.go` 标准 NotFound 分支改用公共
      `mapEntError`，保留幂等查询的正常空结果语义
- [x] 第六批：`finance_cashflow.go` 标准 NotFound 分支改用公共
      `mapEntError`，保留幂等查询的正常空结果语义
- [x] 第七批：`finance_bill.go` 账单与批次标准 NotFound 分支改用公共
      `mapEntError`，保留两类幂等查询的正常空结果语义
- [x] 第八批：`finance_commission.go` 规则、提成单与调整单的标准
      NotFound 分支改用公共 `mapEntError`，保留联合校验及幂等空结果语义
- [x] 第九批：`enterprise_resource.go` 资源与标签组标准 NotFound 分支
      改用公共 `mapEntError`
- [x] 第十批：`fee_ledger_preference.go` 创建约束与乐观锁更新错误改用公共
      `mapEntError`，保留查询、创建分流及幂等删除语义
- [x] 第十一批：`order_query.go` 订单标准 NotFound 分支改用公共
      `mapEntError`，保留重复单号探测的正常空结果语义
- [x] 第十二批：`finance_custom_setting.go` 创建并发约束改用公共
      `mapEntError`，保留默认策略及不存在时创建语义
- [x] 第十三批：`order_shipping_document.go` 标准 NotFound 与提单号命名
      约束改用公共错误映射
- [x] 第十四批：`order_container.go` 标准 NotFound 与箱号命名约束改用
      公共错误映射
- [x] 第十五批：`order_attachment.go` 标准 NotFound 与附件幂等命名约束
      改用公共错误映射
- [x] 第十六批：`order_cargo_item.go` 订单与货物明细标准 NotFound 分支
      改用公共 `mapEntError`

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
