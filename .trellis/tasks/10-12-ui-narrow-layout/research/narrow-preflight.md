# Research: 窄屏订单溢出根因与导航菜单共享预研

- Query: 为已批准的 A10/A11 修复锁定 390px 新建海运订单 537px 整页溢出根因，并研究移动 Drawer 复用桌面权限菜单的最薄路径。
- Scope: internal
- Date: 2026-10-12

## Findings

### 原始材料、规范与文件

已阅读 `.trellis/workflow.md`、领域 glossary/architecture-map、前端 component-guidelines、后续子任务 `10-12-ui-narrow-layout` 的 prd/design/implement，以及原审查 `browser-confirmed.json`、`browser-baseline.json`。公共表格继续在自身容器内横滚；不能通过全页 overflow-x:hidden、居中 maxWidth 或取消详情 hideInMenu 掩盖问题。本次只写研究，后续实现须接续键盘任务最新 HeaderMenus/global.less。

| 文件 | 作用 |
| --- | --- |
| `web/src/components/layout/HeaderMenus.tsx` | 权限过滤后的企业资源和设置项唯一现有构建处 |
| `web/src/app/AppLayout.tsx` | ProLayout route、顶栏操作与 Link 的组合处 |
| `web/src/router/adaptRoutes.tsx` | routes.ts → 权限过滤并剔除 hideInMenu 的侧栏树 |
| `web/src/global.less` | <=768px 隐藏顶栏快捷菜单 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx` | 分节卡及 Row gutter16 包装；实际不是 537px 的直接根因 |
| `web/src/components/ui/form-row/FormRow.tsx` | 992px 以下单列 minmax(0,1fr)，实测已生效 |
| `web/src/pages/orders/templates/sea-template.tsx` | 顶层匿名 grid 和 SeaFieldGroup 默认 auto 单列 |
| `web/src/pages/orders/templates/components/sea/SeaBillContentFormFields.tsx` | 提单字段标题/快捷 Tag/带入按钮的无换行 flex |
| `web/src/pages/orders/order-plan-fields.tsx` | 箱规格/数量/新增按钮的固定宽 inline-flex |

### 浏览器方法和结果

本地 Vite 8001、服务 8000；从 web 依赖调用 Playwright，独立 Chromium context。账号密码只由 `.env.local` 注入进程，未输出/保存。页面所有非 GET/HEAD/OPTIONS API 请求均在浏览器内截断；login/switch-organization/logout 仅由 context.request 管理独立认证 session。没有点击提交，无真实业务写。等待订单表单和 loading 完成，再等待 900ms 响应式稳定。无截图、身份信息、业务值或原始完整 DOM dump 留存。

精简几何证据见本目录 `narrow-evidence.json`。新建订单 375/390px 实测 scrollWidth 均为 537；767/768/769/1440px 分别等于视口，没有整页溢出。

### A11 的两处根因与一个传播路径

1. **提单字段标题/按钮 flex 没有换行**。`SeaBillContentFormFields.tsx:538` 通知人标题的快捷 `+ SAME AS CONSIGNEE` Tag 最右缘 537px；`:673` 实际与提单货物描述标题和“从订单货物信息带入”按钮右缘425px。父容器宽322px，`display:flex;alignItems:center;justifyContent:space-between` 且默认 nowrap。多个相同标题行存在，英文长标题内层 flex 也需要换行。当前 `flexWrap/gap` 仅由 `createLayout` 控制；`SeaDocumentSection.tsx:573` 主单字段调用没有传 `createLayout`，因此即使新建订单，其 MBL 也走默认 false。不要只给 HBL 或仅新建调用补开关：详情和共享主单编辑也需要在窄屏换行。
2. **订舱分组匿名 grid 是内在最小宽的传播路径**。`sea-template.tsx:71` 订舱分节外层、`:159` `SeaFieldGroup` 都是 `display:grid;gap:12;width:100%`，未声明 minmax(0,1fr)/min-width:0。内层箱计划不可收缩宽度通过网格的 intrinsic size 向上传播，分组宽389px，从left25伸至right414。`FormRow.tsx:18-66` 自身 minmax(0,1fr) 单列规则正确，单改其列数不能消除上游 anonymous-grid min-size。最终实验仅修标题flex和箱计划inline-flex换行已得到390px，说明匿名grid不是必须独立修改的第三处根因；其389px是内部箱计划不可折行宽度向上传播的结果。
3. **箱计划空状态 inline-flex 不能折行**。`order-plan-fields.tsx:287` 空状态 wrapper 把 Select130px、InputNumber90px、按钮127px和两个 gap8px放同一行，实测373px。外层已经 `flex-wrap:wrap`，但内部整体作为一个子项不能收缩，仍超过分组324px。`:321` 非空状态的每条箱计划也是 inline-flex，应验证已填项和多个项的邻近路径。

临时浏览器 style 因果实验（逐项插入后撤回；不是产品代码）：

| 实验 | 390px scrollWidth | 意义 |
| --- | --- | --- |
| 原始 | 537 | 完整复现 |
| 仅表单内 inline-style flex 允许换行（不含 inline-flex） | 414 | 最大单证标题溢出消失，上游grid仍被箱计划最小宽撑开 |
| flex 与 inline-flex 都允许换行 | 390，溢出元素0 | 标题和箱计划一起修正即闭环，无需独立grid修改 |
| 匿名 grid 指定 minmax(0,1fr)、自身及直接子项 min-width:0 | 537 | 只修grid仍保留标题/Tag溢出 |
| flex/inline-flex换行与grid轨道约束合用 | 390，溢出元素0 | 结果相同，没有额外收益，未用 overflow 裁切 |

此前仅改 `.ant-space`、FormRow子项或 `.ant-form-item` max-width 的探针均537；强行清 Row gutter会541。因此不能根据第一条可见form-item宽389直接去改Antd字段默认宽度或SectionCard横向边距。

建议最薄正式实现：先在 SeaBillContentFormFields 对具名标题行及内层长标题于窄屏添加 wrap/min-width:0/gap，桌面保持现有排版；在 order-plan-fields 空状态 inline-flex 支持窄屏换行。不要预先修改FormRow或匿名grid：仅这两处换行在当前空表已足够消除整页溢出；若有值详情发现新的独立内在尺寸问题再定位。临时 CSS 使用 style 子串选择器是诊断探针，不能照搬进正式全局 CSS。

真实相邻消费者：`getSeaTemplateSections` 新建/详情同构；MBL 与 HBL 都消费 SeaBillContentFormFields，详情可含只读字段、单证切换工具条；箱计划组件也走新建和详情。实施后需验证375/390px、HOUSE/DIRECT、MBL/HBL活动页签、已填箱计划、只读详情和1440px桌面对照，不以本次新建空表结果宣布详情已过。

### A10 权限菜单共享与断点

`HeaderMenus.tsx:40-117` 设置项已有 canManageOrganizations/canManageUsers/canManageRoles/canReadMasterDataNumberRules/canReadMasterData/canReadAudit/canReadTasks/canAccessPlatform；`:120-153` 企业资源已有 canReadPartners（客户/供应商/国外代理）和 canReadEnterpriseResources（配置管理）。继续消费同一 useAccess 结果，不另复制权限码、不修改服务器Manifest。

`adaptRoutes.tsx:112-151` 当前剔除 hideInMenu；routes.ts 客商/企业配置/master-data/admin及业务详情都采用该语义，不能全部改为false。`AppLayout.tsx:29` 当前 buildMenuData只供一般侧栏，`:65-76` Link消费item.path，`:83-86` HeaderMenus在顶栏actionsRender。

实测公司和系统工作区：390/767px头部菜单none，移动菜单图标存在，Drawer只有工作台/订单/费用（系统工作区无订单）；768px为screen-md，头部菜单仍none且没有Drawer图标；769px为screen-md，菜单flex。**768px存在明确导航断点间隙**。`global.less:1371-1396` 头部隐藏<=768，而已安装ProLayout使用sm/xs为移动态，md从768开始。建议将快捷菜单隐藏的单独规则对齐<768px，不能为此调整所有768px媒体块或只修390px。

最薄共享方式：把当前HeaderMenus的权限筛选构造抽成 layout 范围内的菜单数据函数/Hook，返回企业资源与设置两个非空组的具名条目（key/to/label/icon）；桌面Dropdown和移动菜单都消费该数据。AppLayout仅在对应移动断点将两组转成ProLayout菜单children追加至现有一般菜单，保留原desktop路由树和所有详情hideInMenu。不使用“显示全部hidden路由”策略；没有任何可见children时不输出空组。

可选择在AppLayout使用Antd `Grid.useBreakpoint` 的md标记生成移动附加route，或复用ProLayout提供的 `menuRender(props,defaultDom)` 中实际 `props.isMobile`；后一种需保持ProLayout默认SiderMenu并补menuData，同时验证查询参数路由高亮，不能自建第二个Drawer。现有安装源码支持menuRender传入真实isMobile和默认SiderMenu，移动SiderMenuWrapper自动用Drawer包裹。

设置项 `'/admin?tab=users'` 等必须保留query导航。当前AppLayout.location仅传pathname，Dropdown.selectedKeys也是pathname，移动新增项不能直接照抄而宣称query选中已正确：需明确设置项菜单key/to与当前pathname+search的对应。链接点击还需调用ProLayout的`item.onClick`（MenuItemRenderItem提供 onCollapse(true)），当前自定义Link未传该回调，后续应显式保留移动点击关Drawer行为。

## External references / versions

未用外部网页；API依据已安装的官方包源码核对，`@ant-design/pro-components` 实际3.1.14-6、antd实际6.6.1。源码入口为 `node_modules/.pnpm/@ant-design+pro-components@3.1.14-6_.../node_modules/@ant-design/pro-components/es/layout/ProLayout.js:258-261`（sm/xs移动态）、`:87-100`（menuRender）、`es/layout/components/SiderMenu/index.js`（默认移动Drawer）、`BaseMenu.js:270-280`（自定义菜单项onClick关Drawer）。不是独立旧pro-layout 7.x包；实施者应核对web实际依赖。

## Caveats / Not Found

- 只读预研并未修改产品，不代表A10/A11验收完成；临时style得到390px是根因验证，正式目标还需完整断点、权限、详情和宽表复测。
- 菜单可见性已分别测公司/系统工作区，但没有创建无权限账号或修改授权。正式权限单测需构造不同useAccess结果，不能用当前管理员全权限浏览器覆盖代替无权限测试。
- keyboard任务还会修改 HeaderMenus/global.less；本报告只给数据共享方向，正式实现接续其最终版本，禁止覆盖其键盘/焦点修复。
