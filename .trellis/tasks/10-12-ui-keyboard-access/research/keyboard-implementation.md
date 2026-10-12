# 公共键盘交互实施记录

## 范围及最终实现

A07/A08/A09 产品已完成并停止编辑，交主会话独立复核及提交。未修改后端、请求客户端、依赖、真实数据、权限规则或任务状态；未增加页签 roving tabindex 或重设计导航。

- `HeaderDropdown/index.tsx` 增加显式 `keyboardAccessible?: boolean`，默认 false。只有 HeaderMenus 的企业资源/设置中心以及 OrganizationSwitcher 三个目标按钮启用；AvatarDropdown 等其它消费者保持既有默认行为。
- opt-in 分支内部管理 open，使用官方 Dropdown 的 hover+click trigger、open、onOpenChange、autoFocus 和 HTMLElement ref；原生按钮 Enter/Space 产生 click，触发层仅补 ArrowDown 打开。键盘打开让官方 menu 获取焦点并负责候选方向键/选择，鼠标悬停不自动移入焦点。
- 克隆原按钮只补 aria-haspopup=menu、aria-expanded 和组合 onKeyDown，保留原先 disabled/权限/样式。菜单 Escape 在开放期间关闭，清键盘打开状态并返回同一触发按钮焦点；监听在关闭/卸载清理。Antd 原生 menu、候选权限过滤及业务 onClick 保留。
- OrganizationSwitcher 仅启用 opt-in；既有 Promise 保存保护、失败/取消放行清理和组织身份切换逻辑没有覆盖。
- `SectionCard` 只有 collapsible 标题改为原生 type=button，保留字号、品牌竖线、间距和既有点击面积。aria-expanded 与受控/非受控真实状态同步，useId 为内容提供唯一 aria-controls。内容始终挂载，用 hidden 与既有 body display 隐藏，恢复展开保留输入；extra 仍在按钮外，非折叠标题仍是展示 div。折叠装饰图标 aria-hidden 不污染按钮名称。
- `TagsView` 关闭按钮 keydown stopPropagation，避免原生 Enter/Space 冒泡到父 tab 同时导航；原生 click 继续走保存 guard 与查询清理。没有改变 tab 导航协议。
- `global.less` 仅给业务 tab、关闭按钮、分节折叠按钮添加局部 focus-visible 品牌蓝 2px outline/2px offset，不占布局尺寸；鼠标 hover 与 active 原样。

## 修改文件

产品六文件：`components/HeaderDropdown/index.tsx`、`components/OrganizationSwitcher.tsx`、`components/layout/HeaderMenus.tsx`、`components/layout/TagsView.tsx`、`components/ui/page-shell/SectionCard.tsx`、`global.less`。

验证新增 `HeaderDropdown/keyboard.test.tsx`、`page-shell/SectionCard.test.tsx`，并在既有 `layout/TagsView.test.tsx` 补关闭按钮不冒泡且仍确认脏数据的用例。没有修改 HeaderDropdown 其它消费者默认行为。

## 验证

实现后补风险回归，未采用 TDD：

- 定向七 suite **64 用例通过**，整体 5.79 秒：HeaderDropdown keyboard、SectionCard、HeaderMenus、OrganizationSwitcher、TagsView、TagsView.query-state、PageHeaderShell。
- 覆盖 ArrowDown 打开与官方菜单焦点、Escape 关闭回触发按钮、Enter/Space 浏览器原生激活对应的 click、候选 disabled/选择、opt-in hover 与默认 hover、不启用时 click 行为不变、受控/非受控折叠、唯一 controls/hidden、extra 不折叠、关闭按钮键盘不冒泡和 dirty 保护，以及前置页签查询清理矩阵。
- happy-dom 不会由 keydown/keyup 自动生成原生按钮 click；测试显式发送该 click，不增加自造 Enter/Space handler。初次测试错误期望 menu ul 本身聚焦，官方实现实际聚焦首个可用 menuitem；已改为验证 menu 包含 activeElement。
- 最后 Escape 清键盘开启状态后，HeaderDropdown 两例定向再次通过。
- 修改八个 TS/TSX 文件 Biome check --write 通过。
- 最终 `pnpm --dir web tsc` 通过，`git diff --check` 通过。
- `antd lint components/HeaderDropdown --format json` 0 条问题。Dropdown ref 官方声明为 HTMLElement，类型及浏览器均确认 focus 指向真实按钮。

主会话真实浏览器 **11 项全部通过**，证据见本 research 下 browser-verification.json：三个菜单分别 Enter/Space/ArrowDown 打开，Escape 关闭返回触发按钮；分节原生按钮 expanded/controls；页签/关闭按钮 focus-visible 计算样式。CSS 焦点样式依赖该浏览器证据，不用 happy-dom 样式断言代替。

## 交接与限制

没有已知阻塞。opt-in 分支由组件管理菜单 open，当前三个消费者不外部控制 open；保持原生 Antd menu 键盘行为。完整前端门禁由主会话最终阶段执行，本组没有重复 check:web、后端门禁或 build。未 commit；主会话负责 spec、任务进度和最终提交。
