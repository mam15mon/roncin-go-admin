# Research: 全站共享布局尺寸消费端

- Query: 检查公共模板、业务页面、独立样式及内联样式中的吸顶偏移、视口高度、导航避让和内容区间距，识别适合统一尺寸配置的真实共享语义。
- Scope: internal
- Date: 2026-10-09

## Findings

### 研究边界与相关规范

已阅读 `.trellis/workflow.md`、本任务 `prd.md`、`.trellis/spec/domain/{glossary,architecture-map}.md`、`.trellis/spec/web/frontend/{index,component-guidelines,directory-structure,capability-navigation}.md`。本研究不重复主代理负责的 `global.less`、`defaultSettings`、`AppLayout` 及 CSS 变量桥接。未修改代码，也未执行 Git 操作。

规范要求页面流式铺满、内容外边距统一；公共 UI 禁止反向依赖 pages/features；当前任务要求保留默认视觉，不因相同数字误合并不同职责。建议布局尺寸配置放独立通用配置/常量层，模板及页面均直接消费；无需再提取领域 feature。

### 确定纳入的消费端

下表中的配置语义是建议名称，最终命名由主代理统一；行号为研究时源码位置。

| 文件与位置 | 当前值/行为 | 建议共享语义 | 纳入理由与薄适配 |
| --- | --- | --- | --- |
| `web/src/components/ui/page-shell/PageHeaderShell.tsx:31` | sticky `top:84`，明确注释48 Header+36 Tags | `headerHeight + tagsHeight` / `topStackHeight` | 必须纳入，CSS `top` 直接消费派生变量；保留 `sticky=false` 和调用方 `style` 覆盖。 |
| `web/src/components/ui/parameter-setting-template/ParameterSettingTemplate.tsx:164` | Tabs `tabBarStyle.top:84` | 同一 `topStackHeight` | 必须纳入，与页头位置同源；这里是业务Tabs，不是全站TagsView的高度。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:223` | `locateSectionError(sectionKey,84)` | 数值型 `topStackHeight` | 必须纳入，滚动算法需要 number；这里84是无shell测量时的fallback，不能改成默认146。 |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:391` | `minHeight:'calc(100vh - 48px)'` | `headerHeight` | 必须纳入，只替换扣减48为变量，保留当前仅扣header的公式；不能额外扣Tags并改变默认页面高度。 |
| `web/src/components/ui/form-navigator/FormAnchorNav.tsx:159` | 展开浮层width140 | `formNavigatorWidth` | 必须纳入：与两个正文容器的164避让有真实跨组件联动。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:289` | 导航展开时正文paddingRight164 | `formNavigatorWidth + formNavigatorContentGap`（140+24） | 必须纳入，折叠或不可见仍为0；24为导航与正文避让量，不能借用footer的24。 |
| `web/src/pages/partners/partner-detail.tsx:752` | 导航展开时正文paddingRight164；注释110也有164 | 同上 | 必须纳入，往来单位页真实复用FormAnchorNav；默认折叠与状态契约保留。 |
| `web/src/components/ui/form-navigator/FormAnchorNav.tsx:113`、`:155` | 浮标与展开层right16 | `formNavigatorRight` | 建议纳入，两种状态位置同源；不绑定成内容padding12，保持当前默认16。 |
| `web/src/components/ui/form-navigator/FormAnchorNav.tsx:158` | maxHeight=`100vh - topOffset - 64`，注释底部预留吸底栏 | `formNavigatorBottomReserve`=64 | 纳入，顶部仍来自动态实测，底部reserve单独配置；64不是页脚实测高度，不能硬把两者设成同一语义。 |
| `web/src/components/ui/page-shell/StickyFooterBar.tsx:21` | padding `10px 24px` | `stickyFooterPaddingBlock`=10 / `stickyFooterPaddingInline`=24 | 可纳入公共吸底栏布局；不能直接用内容padding12替换24。实际最终样式须由主代理检查global中的`!important`是否覆盖。 |
| `web/src/pages/orders/detail.tsx:443` | 初次加载骨架外层padding12 | `contentPadding` | 纳入，这里是PageHeader下骨架外容器的内容区留白，非Card内装饰padding。 |

### 表单导航和错误定位的偏移

这些默认偏移均涉及顶部叠层，但源码未给出146/156来自哪些固定高度；不要将推测当成页头高度契约。主代理设计决定保持146/156/100为独立经验偏移配置，不造62/72/16等无直接证据的顶部叠层公式，保留调用方完整数值参数。

| 文件与位置 | 原值 | 建议表达（默认效果不变） | 说明 |
| --- | --- | --- | --- |
| `FormAnchorNav.tsx:25` | targetOffset默认146 | `formScrollOffset`=146（独立默认） | 保留显式targetOffset作为完整数值；不要又加一次顶部叠层。 |
| `FormAnchorNav.tsx:37`、`:39` | 顶部初始化/无shell fallback156 | `formNavigatorTopOffset`=156（独立默认） | fallback与点击offset原本不同，不能合并为一个数。 |
| `formErrorUtils.ts:175` | `scrollToFirstFormError` headerOffset146 | 同一`formScrollOffset` | 此函数滚动居中公式仍使用headerOffset/3（:241），只替换默认来源，不改算法。 |
| `formErrorUtils.ts:278` | `measureStickyTopOffset` fallback146 | 同一`formScrollOffset` | shell存在且bottom>0时优先返回真实bottom+12（:282），不要改成硬编码页头总高。 |
| `formErrorUtils.ts:282` | 测量shell bottom后加12 | `formScrollGap`=12 | 这是落点与页头之间的间隙，独立于内容外padding和卡片间距，即使默认也是12。 |
| `formErrorUtils.ts:323`、`:343` | 分节滚动与分节错误定位fallback146 | 同一`formScrollOffset` | 保留显式fallback参数原语义。 |
| `formErrorUtils.ts:375` | 表格错误headerOffset100 | `tableErrorHeaderOffset`=100（独立默认） | 保持默认100，不宣称来自顶部叠层；与下一行额外间隙分开命名。 |
| `formErrorUtils.ts:452` | 滚动补偿额外16 | `tableErrorScrollGap`=16 | 与100阈值之外的落点间隙不同职责，可独立命名；不能因都为16就借用浮层right。 |
| `form-navigator/types.ts:26`、`:39`、`:67` | 文档硬写默认146/100 | 描述派生默认语义 | 实现接入配置后同步注释，避免未来配置变更时文档仍承诺旧数值。 |

`web/src/pages/partners/components/PartnerAnchorNav.tsx:33` 复用FormAnchorNav且不传targetOffset，`:68` 调用locateSectionError不传fallback；无需在页面重复配置。`partner-detail.tsx:631`、`FeeFormModal.tsx:121`、订单模板`:206`调用表单错误定位均不传headerOffset，公共默认来源变化即可联动。`OrderFeeTableTabs.tsx:696`及后续多处调用表格错误定位也不传headerOffset。

`FormAnchorNav.tsx:58` 的额外120是滚动激活阈值缓冲，和Header/Tags高度没有同一职责；可以保留本地或单独命名为导航激活缓冲，不能并入顶部栏高度。

### 组织页视口高度（本期不纳入）

- `web/src/pages/admin/components/org/OrgChartCanvas.tsx:218`：`height:calc(100vh - 270px)`，源码没有拆分说明，原 `minHeight:600` 可能钳制结果。
- `web/src/pages/admin/organizations.tsx:420`：组织树 `maxHeight:calc(100vh - 340px)`，源码没有拆分说明，原 `minHeight:400` 保留。
- 270/340可能同时包含全局顶部栈与页面局部占位，但没有直接推导证据。主代理设计决定不为本次参数化造84+186/256的公式，维持现有页面局部估算；不扩成DOM测高框架。
- 图内节点间距48/36（`OrgChartCanvas.tsx:240`）、画布padding60（:241）、组织树列宽380（`organizations.tsx:390`）属于组织图/树自己的布局，不与侧栏或页签尺寸同源，保留。

### 公共页面区块间距（可单独纳入，不绑定内容padding）

公共模板的外层卡片/区块之间12px有相同布局语义，可统一成 `pageSectionGap`=12。这与内容外padding12是不同职责，即使初始值相等也分别配置：

- `page-shell/SectionCard.tsx:87`：分节卡片marginBottom。
- `master-data-template/MasterDataTemplate.tsx:513`、`:524`：页面notice与统计卡片网格marginBottom；网格内部gap10不纳入。
- `parameter-setting-template/ParameterSettingTemplate.tsx:166`：顶层业务页签与内容的间隔。
- `order-template/OrderFormTemplate.tsx:252`、`:257`：骨架外marginTop与骨架提示卡marginBottom。
- `order-list-template/OrderListSearchFilter.tsx:245`、`OrderListToolbar.tsx:168`：搜索区/工具条与后续表格的间隔。
- `finance-ledger-template/FinanceLedgerTemplate.tsx:407`、`:420`：顶层筛选卡与统计区的外间隔；Row内部gutter12不因数值相同自动替换。
- `search-filter-template/SearchFilterTemplate.tsx:382`、`:464`、`:487`：三模式搜索区的外marginBottom。
- 页面根级同类候选：`pages/admin/organizations.tsx:297`、`pages/admin/users.tsx:160`/`:339`、`pages/finance/nettings/index.tsx:421`（分别是页面卡片/卡片区域外间隔）。实施前确认所在层级后可消费pageSectionGap。

不需要全站搜到12就替换。模态表单内部提示、联系人卡片、审计记录项、输入Row gutter、Space按钮间距属于各自内部布局。`StickyFooterBar.tsx:25`外marginTop16原本不同，也不应强改成12；若配置化须独立保留默认16。

### 不纳入的范围与理由

- `100vh` / `100%` 本身：`OrderListTemplate.tsx:563`、`ParameterSettingTemplate.tsx:149`及订单/客商/提成/财务详情多个页面没有硬扣Header或Tags；这些是纯视口或父容器比例，不受顶部尺寸变更影响，保持当前公式。登录页100vh同理。
- `pages/user/login/index.module.less:57` left48、`:138`/`:208`控件height48：独立登录页装饰/输入控件布局，与侧栏折叠48及Header48没有同一语义。
- `pages/partners/components/ContractCardList.tsx:413` label宽84：标签列宽，不是Header+Tags84。
- `page-shell/SectionCard.tsx:97`/`:102`、SearchFilterTemplate等Card内padding，字大小、圆角、品牌蓝竖标3×15：内部视觉密度，按PRD不做全站装饰数值替换。
- `FinanceSummaryBoard.tsx:179` sticky bottom0：0是视口下边界，不是缺失可配置布局值；其内容随文本换行，无固定高度需要和header联动。
- `FinanceLedgerTemplate.tsx:107`/`:108` 拖列参考线position fixed/top：已取tableRect真实坐标，没有共享数字问题。
- `CurrencyAmountInput.tsx:195`与`PackageCountInput.tsx:88`的calc100%-控件宽度：已经由组件自己的width prop派生，不并入全站布局。
- 表格列width140不与FormAnchorNav宽140同源；`FormRow.tsx:22` gap12/16与992/1400响应式断点：表单自己的栅格与断点，不是ProLayout的移动侧栏边界，保留。
- `ui/dialog-sizes/constants.ts:8`/`:21`已有MODAL_SIZE/DRAWER_SIZE标准尺寸配置，是另一套窗口容量语义，直接保留既有唯一来源；不搬入壳层尺寸。
- `ColumnSettingsPanel.less`、`SearchFilterTemplate.less`仅内部控件样式；`organization-picker.module.less`仅登录组织选择器；研究范围内未找到独立CSS中遗漏的header联动尺寸。

### 文件索引

除上表逐项引用的消费端外，关键适配/回归文件：

- `web/src/components/ui/form-navigator/types.ts`：公共数值偏移契约，不改变调用方传参含义。
- `web/src/pages/partners/components/PartnerAnchorNav.tsx`：客商表单导航适配，默认消费公共参数。
- `web/src/components/ui/index.ts`：公共模板导出入口，无需因常量配置扩大业务导出面。
- `web/src/components/ui/dialog-sizes/constants.ts`：已集中管理的弹窗尺寸，保持独立。

### 定向验证建议

研究阶段未运行测试；实施后优先已有相关定向测试，路径相对web目录：

```bash
pnpm --dir web exec vitest run src/components/ui/page-shell/PageHeaderShell.test.tsx src/components/ui/parameter-setting-template/ParameterSettingTemplate.test.tsx src/components/ui/form-navigator/FormAnchorNav.test.tsx src/components/ui/form-navigator/formErrorUtils.test.ts src/components/ui/form-navigator/formErrorAntdContract.test.tsx src/components/ui/order-template/OrderFormTemplate.test.tsx src/components/ui/order-template/OrderFormTemplate.visibility.test.tsx src/components/ui/finance-ledger-template/FinanceLedgerTemplate.test.tsx src/pages/partners/partner-detail.test.tsx src/pages/admin/organizations.test.tsx src/pages/admin/components/org/OrgChartCanvas.test.tsx
```

若纳入公共区块间距，再运行 `MasterDataTemplate.test.tsx`、`SearchFilterTemplate.test.tsx`、`OrderListTemplate.test.tsx` 对应完整路径。这些组件测试保证行为未受影响；目前PageHeader、导航等测试没有验证真实浏览器尺寸，不应把样式字符串断言当成覆盖证明。

真正有价值的布局验证：默认配置下Header/Tags/页头位置连贯；修改Header/Tags尺寸后业务Tabs与页头跟随、点击导航不会被页头遮住；修改导航宽度后订单与客商展开避让同变、折叠归零；修改内容padding后订单加载骨架对齐；组织图保持本期既有尺寸；`style`、targetOffset、headerOffset显式覆盖仍保持原契约。默认桌面展开/折叠/滚动、移动端和767/768/769边界由主代理统一浏览器验收。

### 外部参考

本题为仓库内部尺寸消费端调查，未使用外部文档。当前技术栈版本依据仓库规范：React 19、Vite 7、React Router v8、Ant Design 6。没有需要查询新版组件API的结论。

## Caveats / Not Found

- 146/156/100来源没有直接高度公式证据，最终设计分别集中为独立默认偏移；270/340为页面局部估算，最终设计不纳入本期；继续保留页头真实测量。
- 布局尺寸由全局CSS和ProLayout配置共同决定，本研究未检查主代理负责的global覆盖优先级；StickyFooterBar/SectionCard最终computed样式应结合global复核。
- `measureStickyTopOffset`目前只在存在shell时真实测高；不要趁本次尺寸集中管理改变错误定位的滚动算法、引入额外降级分支或改调用方完整数值参数的含义。
- 不同职责的默认12/16/24不能绑定为一个万能spacing；内容padding、区块间距、滚动落点间隙、导航避让、footer内padding分别管理。

## 主会话规划采纳结果

- 确定纳入 header/tags 派生84、表单导航宽与正文避让、公共footer几何参数、公共页面区块间距、订单加载骨架内容padding。
- 146/156/100缺乏可分解高度契约：只集中命名现有完整偏移，保留真实测量优先与显式参数语义，不引入62/72/16的推测性顶部栈分解。
- 组织画布270和组织树340没有明确顶部栈拆分证据：保留页面局部当前公式，本期不造186/256补数，不修改图形布局。
- 同值字体、列宽、card内部padding继续保留或使用既有theme/dialog-size来源。
