# 窄屏导航及订单布局实施记录

## 最终范围

A10/A11 实施完成，产品已停止编辑，交主会话独立检查及提交。未修改请求、后端、生成物、依赖、真实业务数据、任务状态或spec。保持键盘任务的 HeaderDropdown opt-in、保存保护和查询关闭清理。没有增加全页 overflow-x:hidden、居中 maxWidth 或独立移动Drawer。

依据预研精确根因，未修改 FormRow、SeaBasicInfoSection、OrderFormTemplate、匿名grid或adaptRoutes。产品共六文件：

- `components/layout/workspaceMenus.tsx`（新增共享菜单入口）。
- `components/layout/HeaderMenus.tsx`、`app/AppLayout.tsx`、`global.less`。
- `pages/orders/templates/components/sea/SeaBillContentFormFields.tsx`、`pages/orders/order-plan-fields.tsx`。

## 导航

`buildWorkspaceMenus(access: AccessState)` 持有原先企业资源/设置菜单构造，返回 `WorkspaceMenus={settingsItems,enterpriseResourceItems}`；条目为 key/label/icon。不复制权限码，桌面和移动都消费真实 useAccess 结果及同一函数。HeaderMenus保留原按钮、键盘 opt-in、onClick导航与active样式。

`toMobileWorkspaceMenus(menus)` 按原条目转换为官方 MenuDataItem 的 key/path/name/icon/children，空组不输出。AppLayout仅在官方 Grid.useBreakpoint().md明确false时，以menuDataRender给既有一般菜单追加两组；md（768px起）和更大屏保持原桌面侧栏。没有改hideInMenu，业务详情/历史页签不进入新增导航。

移动菜单Link显式调用官方item.onClick，保留ProLayout关闭当前Drawer行为；导航保留/admin?tab=...。`resolveWorkspaceMenuKey(pathname,search,menus)`仅按实际query匹配设置tab，通过menuProps.selectedKeys精确高亮；没有query时不猜默认，避免设置页read权限与快捷菜单manage权限不对称导致标错。桌面Dropdown同样消费该精确key。设置页默认项及query同步仍归原ParameterSettingTemplate。

顶栏快捷菜单单独隐藏规则改为 width<768px，对齐ProLayout移动判定，修复原来768px没有Drawer且快捷菜单隐藏的入口间隙。其它768px媒体块不修改。

## 订单几何修复

- SeaBillContentFormFields 给真实标题flex行与内层长标题添加具名 class；仅<768px允许wrap/min-width:0。主单/分单、新建/详情共享生效，不依赖createLayout开关，不改变桌面inline排版。
- order-plan-fields空状态及每条有值箱计划inline-flex添加具名class，仅窄屏允许换行，保持Select/InputNumber宽度及字段行为；消除内部不可换行宽度向匿名grid传播。
- 初次正式浏览器390px已过，但375px仍386px。主会话几何定位精确到通知人Segmented长标签，补官方classNames语义root/label/item及具名Space；窄屏max-width100%、item min-width0及弹性收缩、label/Space允许换行。没有全局Antd内类耦合或裁切。正式重测375px恢复375px。

## 验证结果

实现后按风险补回归，未TDD；未写CSS文本镜像测试代替浏览器几何。

- 菜单及桥接 **4 suite、11例全部通过**（最终4.19秒）：workspaceMenus、AppLayout.menu、既有HeaderMenus、键盘HeaderDropdown。验证真实access下公司/系统权限、无权限无空组、桌面不追加移动组、字段映射、实际query精确高亮、Link保query及调用官方关闭回调，并保留键盘及hover行为。
- 订单相邻 **8 suite、54例全部通过**（整体24.35秒，并行tests69.49秒）：new、detail-draft-lifecycle、sea-create-layout、sea-document-visibility、sea-document-validation、SeaDocumentSection、SeaDocumentSection.fields、order-plan-fields。没有增加重型订单用例。部分既有单证测试打印localhost:3000 ECONNREFUSED/socket hang up，测试结果通过；没有以此变更产品请求或网络契约。
- 修改7个TS/TSX文件Biome check --write通过，最终复查无改动。
- 最终 `pnpm --dir web tsc` 通过。
- `pnpm run check:architecture` 18检查通过，418产品文件、1534本地依赖，没有边界违规。
- `git diff --check`通过。
- SeaBillContentFormFields `antd lint --format json` 0条问题。

主会话真实浏览器 **36项全部通过**，详见 `research/browser-verification.json`：

- 24项真实读取：订单新建375/390/767/768/769/1440，有值两条箱计划375/390/1440，另一可选公司现存订单详情375/390/1440，公司和系统菜单12断点以及移动点击关闭。
- 12项详情GET fixture：HOUSE/DIRECT × 编辑/只读 × 375/390/1440，以fixture让真实详情组件覆盖布局邻接组合；没有向真实业务接口写入。

Drawer关闭仍保留DOM，这是官方正常生命周期；通过root没有ant-drawer-open、content-wrapper-hidden证明关闭，不要求DOM销毁。几何验收使用实际scrollWidth与innerWidth，不用CSS字符串或happy-dom宣称通过。

## 交接

没有已知产品阻塞。完整前端门禁由主会话最后整合阶段运行，本组没有重复check:web、后端门禁或build。产品停止编辑，未commit；共享菜单入口及语义样式的spec由主会话维护。无query不设置高亮是明确选择，不新增第二套设置页可见分节registry。
