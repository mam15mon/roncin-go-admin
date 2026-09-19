# ADR 0011: antd 6 测试交互惯例（vitest + @testing-library）

- 状态：已采纳
- 日期：2026-09-19 沉淀成文（antd 6 自 2026-08-20 脚手架即为基线）
- 主题：前端（测试）

## 背景

前端从初始脚手架起就是 antd 6（`antd ^6.6.0`，随 `@rc-component/picker`
1.x）。antd 5 时代流传的测试写法在 antd 6 下**静默失效**：不是报错，
而是「元素找不到」或「回调未触发」——测试失败容易被误判为组件 bug，
写错方向的断言则会漏测。踩坑后把正确惯例沉淀为规范（2026-09-19
`825ec5e4`）。

## 决策

三条强制惯例（规范全文见
[../web/frontend/quality-guidelines.md](../web/frontend/quality-guidelines.md)）：

1. **DatePicker / MonthPicker 弹层只在 `click` 时打开**。rc-picker 1.x 的
   `SinglePicker` 只绑定 `onClick`，`fireEvent.mouseDown` 不再打开弹层。
2. **`htmlType="submit"` 按钮必须点击按钮元素本身**。原生表单提交的激活
   行为只属于按钮节点；`getByText` 取到内层 `<span>`，点击它只触发 React
   冒泡事件，不触发 `onFinish`。正确写法是
   `fireEvent.click(screen.getByRole('button', { name: /查询/ }))`。
3. **`modal.confirm` 已移除 `confirmLoading`**（tsc 直接报错），加载态改用
   `okButtonProps: { loading }`；且其配置对象是调用时一次性快照，`onOk`
   返回 Promise 时 antd 会自动为确认按钮加转圈，优先依赖内置行为而非
   手工状态镜像。

配套的测试纪律：

- 定向测试统一 `pnpm --dir web exec vitest run <相对 web/ 的路径>`；
  禁止 `pnpm --dir web test -- <file>`（会跑全量套件）。
- 路径写成 `web/src/...` 前缀时 vitest 过滤器静默匹配 0 个测试且退出码
  为 0，形成「全绿」假通过；运行后必须确认输出中的测试文件数大于 0。

## 理由

- 这些差异无法从组件代码看出，只能靠既有测试范本（如
  `OrderListTemplate.test.tsx`）传递；不沉淀成文则每个新测试都会重新踩。
- 「静默失效」比「报错」更危险：错误写法产出的测试是负资产。

## 后果

- 新测试 review 时按这三条对照；发现新的 antd 6 差异直接补进
  quality-guidelines 而不是只在测试里写注释。
- 这是「分阶段验证」策略（开发期定向测试、提交期定向+tsc、验收期完整
  门禁）的组成部分，见 [../../../AGENTS.md](../../../AGENTS.md)。
