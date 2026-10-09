# 全站共享布局尺寸设计

## 边界与目标
共享布局基础层负责尺寸唯一真相源；ProLayout、全局样式与公共模板消费。默认配置保持当前视觉，不修改业务结构、主题颜色、权限或接口。领域页面只同步真实共享导航的消费端。

## 唯一配置与桥接
- 新增 `web/config/layout.ts`：纯 TypeScript 数值配置、派生值、CSS变量映射，不依赖 React/app/pages/features。
- 全站壳尺寸：展开侧栏208、折叠侧栏48、header48、TagsView36；派生吸顶基础偏移为 header+TagsView。
- 独立但共享的几何参数：桌面/移动顶部栏水平padding14/8，正文横向padding12、纵向padding10，PageContainer block padding8、公共页面区块间距12（独立于内容padding12），折叠菜单项36，侧栏底部触发区40及菜单底部额外预留10（菜单padding-bottom=触发区+10）。不同语义不因为当前数值相同而共用键。
- 公共表单导航：宽140、右侧边距16、正文额外间隙24，正文预留=导航宽+间隙；底部安全预留64；定位测量后的额外间隙12。现有146/156为经验默认偏移，分别集中命名，不猜测将其分解成不存在的业务尺寸；表格错误定位的默认100及独立间隙16同理集中命名；继续优先测量真实页头，保留调用方显式offset覆盖契约。
- 公共吸底栏的几何参数与布局层级可以分别配置：footer block/inline padding10/24、marginTop16；header/sider/trigger/tags/page-header/footer/anchor层级保持原值，不使用数值相同的别的层级常量。
- `defaultSettings.ts` 的 ProLayout sider/header/page-container tokens、`AppLayout.tsx` 的 collapsedWidth 使用配置；修正过时的配置注释。
- `main.tsx` 在render前将映射通过document.documentElement.style.setProperty安装为CSS自定义属性，覆盖全站与挂到body的portal；不增加运行期设置、兼容默认值或配置校验平台。
- `global.less`、公共模板的尺寸CSS使用 `var(--roncin-...)`，派生偏移使用CSS calc；需要滚动定位数值的函数从同一TS配置读取数值。
- 测试渲染不依赖实际主入口：业务行为单测继续正常，真实布局用Chromium验证配置安装与CSS解析。

## 消费端
`global.less`、`defaultSettings.ts`、`AppLayout.tsx`、`main.tsx`、`PageHeaderShell`、`ParameterSettingTemplate`、`FinanceLedgerTemplate`、`StickyFooterBar`、`FormAnchorNav`、`formErrorUtils`、`OrderFormTemplate`、`partner-detail`；同时纳入 SectionCard、MasterDataTemplate、OrderListSearchFilter、OrderListToolbar、SearchFilterTemplate 的页面区块外间距，以及 orders/detail 加载骨架的内容留白；页面根级同语义区块间距按research证据同步。

## 取舍
- 颜色、字体、控件高与卡片/表格内部间距已有 `theme.ts`，不和页面外壳混用。页面特有图形/视口预留如果没有代码证据，不将其硬拆成顶部栏高度加补数。
- CSS变量不能用于@media条件；本期保留现有响应式断点语义和数值，不为尺寸去重引入编译配置或修改组件响应式策略。固定header现有<768判定必须保留。
- 固定header、页签、吸顶页头连续覆盖正文，组件负责的折叠占位与样式宽度同步；不添加回退双读。
- 若研究发现独立布局自身多处使用同一尺寸，可在其模块集中配置，避免把页面专属值塞入全局配置。

## 风险与回滚
CSS继承范围、默认值安装时机、组件数值与CSS变量同步、portal、sticky偏移及导航测量是验证重点。实现后测试默认配置和一组替代尺寸；替代配置仅用于验证，验收后恢复默认，不写用户数据。失败在该变更组整体回滚，不散落兼容补丁。
