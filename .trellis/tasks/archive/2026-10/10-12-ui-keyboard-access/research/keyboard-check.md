# 公共键盘交互最终审查

日期：2026-10-12。对应 A07 / A08 / A09，独立审查通过。

## 已修复发现

审查发现 KeyboardDropdown 的 Escape 直接关闭受控 open，但未清 keyboardOpening；后续 hover 可能沿用 autoFocus 并夺取焦点。实现者已补 Escape 同时清 keyboardOpening，最终源码已核对并完成定向重跑。

## 范围与路径核对

- HeaderDropdown 通过 `keyboardAccessible` 显式启用，仅企业资源、设置中心、组织切换三入口接入；其它消费者默认 hover 行为保持。使用真实原生按钮 Enter/Space 的 click 激活，ArrowDown 局部展开，菜单内键盘交给 Antd；Escape 关闭并返回真实触发元素焦点。
- 已核对安装版 antd Dropdown ref 的正式类型为 HTMLElement，与焦点返回实现一致。选项权限过滤、disabled 行为和原有 hover 保留。
- SectionCard 仅 collapsible 时呈现原生 button，type=button 防止表单误提交；唯一 useId 内容关联、aria-expanded 与受控/非受控状态一致。内容保持挂载但 hidden，extra 在折叠按钮外，点击不折叠。
- 页签及关闭按钮 focus-visible 使用不占布局尺寸的描边；鼠标 hover/active 保留。关闭按钮 keydown 停止冒泡，避免父页签提前切换；原生 click 仍经过已有未保存确认。
- 保存确认 Promise、列表查询清理及组织/用户身份逻辑没有漂移；无权限、生成物、后端或业务数据变更。

## 验证

- 审查者独立修改文件 Biome：8 文件通过，无修复；`git diff --check` 通过。
- 实现者最终 7 文件 64 例通过，覆盖下拉 opt-in/默认 hover、原生菜单选择与权限、分节受控/非受控/extra、Tags 键盘关闭及相邻保存/查询保护。
- Escape 标记清理后定向 2 例再次通过；最终 `pnpm --dir web tsc` 与修改文件 Biome 通过。无产品新变化，审查不重复同批 64 例。
- 主会话浏览器 11 项通过：三入口分别 Enter/Space/ArrowDown 打开与 Esc 返回焦点、分节原生键盘折叠及状态关联、页签/关闭真实焦点可见。0 页面错误，页面写请求全拦截且 0 业务写入。

## 未修复问题

无任务范围内具体未修复发现。最终跨任务门禁、提交和归档由主会话继续执行；本子审查已就绪并停止产品编辑。
