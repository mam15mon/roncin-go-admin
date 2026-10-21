# 复用能力导航

> 开发新页面前先查本表：已有能力直接消费公开入口，不复制实现。
> 导航只列「场景 → 入口 → 职责 → 不可放入内容」；细节读入口文件与关联 spec。
> 新能力只有出现真实跨模块使用才提取进来；提取后必须更新本表。

## 按场景找入口

| 场景 | 入口 | 职责 | 不可放入内容 |
|------|------|------|--------------|
| 异步写确认气泡 | [`web/src/components/ui/async-popconfirm/index.ts`](../../../../web/src/components/ui/async-popconfirm/index.ts) | `AsyncPopconfirm`：沿用官方 Promise loading；回调 `false` 或 reject 保留弹层，成功关闭；调用方验证报文并负责已有错误反馈 | 领域请求、响应契约猜测、重复通知；本地同步确认仍用原生 Popconfirm |
| 全站共享布局尺寸与层级 | [`web/config/layout.ts`](../../../../web/config/layout.ts) | 组件数值、CSS变量及顶部栈/导航避让派生尺寸；main首次渲染前安装到html | 普通控件主题、表格列宽、页面专属图形尺寸与媒体查询条件 |
| 页面骨架、分节表单/详情、CRUD 列表 | [`web/src/components/ui/index.ts`](../../../../web/src/components/ui/index.ts) | `PageHeaderShell`/`SectionCard`/`StickyFooterBar`、`OrderFormTemplate`、`MasterDataTemplate`、`FinanceLedgerTemplate`、`OrderListTemplate`、`SearchFilterTemplate`（grid/bar/custom 三模式列表搜索区，内置已提交条件 chips 回显与单个删除/清除全部，消费方零改动生效）等视觉与结构模板 | 领域请求、领域状态映射；模板内不发起新领域请求 |
| 出口订单五卡片表单架构 | [`web/src/pages/orders/templates/`](../../../../web/src/pages/orders/templates/)（`components/FieldGroup.tsx`、`components/sea/SeaCollapsibleNotesField.tsx`） | 海运/空运出口共用 new/detail 页面骨架与 `OrderFormTemplate`，kind 差异只落在 `order-kinds/registry` + `templates/{sea,air}-template.tsx` 的五卡片分节（业务与客户/订舱与运输/货物/单证/内部人员）；卡内小节竖标统一 `TemplateFieldGroup`，备注统一 `SeaCollapsibleNotesField`（name 支持嵌套路径），空运复用 `Sea*` 共享件为既定先例 | 新订单 kind 另起平铺模板或第二套小节/备注组件；为对齐视觉在 kind 间复制粘贴字段定义 |
| 表格窄屏展示与横向滚动 | [`web/src/components/ui/responsive-table/index.ts`](../../../../web/src/components/ui/responsive-table/index.ts) | `ResponsiveTable` / `ResponsiveProTable` / `ResponsiveEditableProTable`：实际可见列宽、统一拖拽调宽/双击展开还原、窄容器固定区、官方吸附滚动条与弹层滚动归属 | 业务请求、编辑状态、列偏好；页面复制滚动适配；以整页裁切掩盖溢出 |
| 组织切换落点 | [`web/src/router/organizationSwitchDestination.ts`](../../../../web/src/router/organizationSwitchDestination.ts)、[`web/src/router/workspaceTabs.ts`](../../../../web/src/router/workspaceTabs.ts) | 从集中路由和新 access 保留功能位置、详情回列表、共享子页签可见性；AppLayout 隔离异步工作区转换 | 复制权限码、保留旧实体查询、仅凭菜单可见性放行 |
| 未保存编辑离开保护 | [`web/src/components/layout/useUnsavedChangesGuard.ts`](../../../../web/src/components/layout/useUnsavedChangesGuard.ts)、[`web/src/components/ui/GuardedModalForm.tsx`](../../../../web/src/components/ui/GuardedModalForm.tsx) | 同步用户编辑标脏、每页签多实例守卫；ModalForm 官方生命周期薄包装，外部 state 显式接入 | 初始化误标脏、请求失败清 dirty、全局匿名表单扫描、跨组织保留编辑闭包 |
| 业务列表页签查询恢复 | [`web/src/components/ui/list-query-state/index.tsx`](../../../../web/src/components/ui/list-query-state/index.tsx) | 工作区实例内已提交filters/labels/pagination、关闭清理及代际拒绝旧事件；布局与多个业务列表共同消费 | 请求结果、loading、选中行、未提交草稿；全局匿名Map或KeepAlive |
| 表格列设置 | [`web/src/components/ui/column-settings/index.ts`](../../../../web/src/components/ui/column-settings/index.ts) | `useColumnSettings`/`ColumnSettingsEntry`：齿轮浮层即时显隐/排序/首尾冻结，返回 `{ columns, entry }`；专有存储复用 `applyColumnFreeze`/`getColumnFreezeTargets`；`advanced` 经「更多设置」二级弹窗注入；缺省消费真实认证身份，偏好按用户+组织+表格标识本地持久化 | 业务请求与数据流；各页面另造第二套设置 UI；列设置禁止用居中 Modal + 保存确认形态（视图偏好可逆，须所见即所得） |
| 业务状态标签/文案 | [`web/src/constants/statusMeta.ts`](../../../../web/src/constants/statusMeta.ts) | `orderFeeStatusMeta` 等各实体状态元数据与通用 `statusTag`/`statusText`/`makeValueEnum` | 页面自写「状态→颜色/文字」映射；不新建平行的全局状态渲染体系 |
| 账单状态映射 | [`web/src/features/finance/bill-status/index.ts`](../../../../web/src/features/finance/bill-status/index.ts) | `billStatusMeta`（账单列表、抽屉、建账结果表共用） | 与费用账单占用事实（`hasActiveBill`）混同；账单语义只归本能力 |
| 财务进度文案映射 | [`web/src/features/finance/fee-progress/index.ts`](../../../../web/src/features/finance/fee-progress/index.ts) | `feeLedgerProgressLabels`：整张关联账单开票与核销综合进度的文字/颜色（财务费用台账与订单费用关联账单列共用） | 行配色 key、台账请求与列配置逻辑（留在各调用方）；不得另建第二套进度文案 |
| 账单建账工作台 | [`web/src/features/finance/bill-creation/index.ts`](../../../../web/src/features/finance/bill-creation/index.ts) | `BillCreationWorkbench` 及 Props（订单费用、财务费用、账单列表三入口共用） | 账单页私有表单/抽屉（`BillEditModal` 等留页面）；未经真实共用不上提新组件 |
| 审计记录展示 | [`web/src/features/audit/index.ts`](../../../../web/src/features/audit/index.ts) | 审计操作/对象/明细的展示转换（管理端审计页与业务页审计分区共用） | 在管理页与业务页各自复制转换逻辑 |
| 汇兑损益展示 | [`web/src/features/finance/exchange-gain-loss/index.ts`](../../../../web/src/features/finance/exchange-gain-loss/index.ts) | `ExchangeGainLossTag`（对冲汇差、核销已实现损益共用展示） | 损益口径与计算逻辑（留在调用方） |
| 业务标签列表渲染 | [`web/src/components/business-tag/BusinessTagList.tsx`](../../../../web/src/components/business-tag/BusinessTagList.tsx) | 分组色描边 Tag 列表（账单、费用台账、订单费用面板共用） | 列宽、搜索等列配置（留在调用方） |
| 人员候选标签 | [`features/personnel`](../../../../web/src/features/personnel/index.ts) | 当前公司人员的姓名与部门/团队显示，无部门显示公司 | 登录账号、跨公司成员查询 |
| 往来单位候选标签 | [`web/src/features/partners/index.ts`](../../../../web/src/features/partners/index.ts) | `PartnerSelectOptionTags`、`searchPartnerOptions` 与 `PartnerOption`（核销、收付、订单等共用） | 往来单位领域展示回流通用 `components/` |
| 信用额度干预提示 | [`web/src/features/finance/credit-control/index.ts`](../../../../web/src/features/finance/credit-control/index.ts) | `useCreditLimitIntervention` 与 `disableCreditExceededOptions` 信用控制查询和候选禁用 | 业务查询 Hook 放全局 `hooks/` |
| 异步竞态防护 | [`web/src/hooks/`](../../../../web/src/hooks/) | 通用 `useLatestAsync`/`useAsyncGuard` 竞态工具 | 任何业务查询 Hook（领域 Hook 进 features） |
| 订单候选缓存 | [`features/orders/options`](../../../../web/src/features/orders/options/index.ts) | 字典、首批港口/机场、人员候选缓存；组织切换/退出调用清理入口 | 把缓存键隔离误认为页面迟到响应保护 |
| 整票费用审核状态与摘要 | [`features/orders/fee-review`](../../../../web/src/features/orders/fee-review/index.ts) | `FeeReviewStatus`、`FeeReviewSummary`、状态筛选选项、`useOrderFeeReview` 及按 Unicode 码点计数的审批意见规则；订单页面、费用台账、建账候选共用，查询按组织隔离 | 审核资格／建账资格在前端自行推算；引用页面私有审批弹窗；将机器人消息当审批结果 |
| 费用更正历史与审批 | [`features/orders/fee-correction`](../../../../web/src/features/orders/fee-correction/index.ts) | `FeeCorrectionSection`：订单与财务费用详情共用的申请历史、原/新费用、批准时利润及提成快照、动态财务进度、审批与撤回；审批资格消费服务端，预览绑定版本和指纹 | 费用录入表单留在订单页面；不在前端推算提成或金融结清，不导入页面私有表单 |
| 币种候选 | [`features/master-data/currencies`](../../../../web/src/features/master-data/currencies/index.ts) | `getCurrencies` / `getCurrencyOptions`，启停后显式强制刷新 | 在每个页面重复维护币种请求缓存 |
| 船公司候选 | [`features/master-data/shipping-lines`](../../../../web/src/features/master-data/shipping-lines/index.ts) | `searchShippingLineOptions` 服务端关键字搜索 | 循环翻页拼接全量 |
| 通用候选类型 | [`types/select-option.ts`](../../../../web/src/types/select-option.ts) | `SelectOption` 基础字段，业务类型按领域组合 | 散客、信用超额等业务字段 |
| 权限判定 | [`web/src/app/access.tsx`](../../../../web/src/app/access.tsx)、[`web/src/access.ts`](../../../../web/src/access.ts) | 按钮与路由使用 `useAccess`；角色候选复用 `createPermissionCapabilityChecker` 消费 auth/me 有效能力（键名来自生成物） | 页面硬编码第二套权限规则、将全量字典当作可授予候选 |
| CSV 序列化与下载 | [`web/src/utils/csv.ts`](../../../../web/src/utils/csv.ts) | `CsvColumn`/`serializeCsv`/`downloadCsv`：BOM、CRLF、文本公式防护、数值精度与下载清理；财务字段映射留各页 `ledgerExport.ts` | 从界面 `dataIndex` 推断业务导出、循环翻页获取全量、重新计算业务金额 |
| 认证业务查询缓存 | [`web/src/components/layout/OrganizationWorkspace.tsx`](../../../../web/src/components/layout/OrganizationWorkspace.tsx) | 用户+工作台独立 QueryClient Provider，身份缺失不挂业务，卸载清理缓存与迟到反馈；组件通过 `useQueryClient` 消费 | 业务直接导入公开登录预热单例、只在普通 effect 清缓存 |
| 后端请求 | [`web/src/services/roncin/`](../../../../web/src/services/roncin/) | OpenAPI 生成客户端（禁手改） | 页面自拼后端主机地址 |

## 何时提取新能力（单一路径）

1. **先搜索**：本表 + `rg` 符号名，确认能力不存在。
2. **私有就近**：单一页面用的实现留在该页面/页面 `components/`。
3. **真实跨模块使用再提取**：出现第二个模块消费时提取到
   `web/src/features/<领域>/<能力>/`，经 `index.ts` 公开入口导出最小面。
4. **验证边界**：提取后运行 `pnpm run check:architecture`，规则见
   [directory-structure.md](./directory-structure.md) 与
   [ADR 0016](../../adr/0016-frontend-module-boundaries.md)。
