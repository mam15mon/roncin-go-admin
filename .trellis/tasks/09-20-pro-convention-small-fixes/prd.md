# Pro 规范小修清单：路由导航与 DOM 定位偏离

## Goal

修复 2026-09-19 全仓审计确认的具体官方推荐偏离点（6 项），全部为小改动；
系统性服务端状态偏离不在本任务（见 `09-20-data-layer-direction`）。

## Requirements

1. **formErrorUtils antd 内部类名依赖加固**（P1，`web/src/components/ui/form-navigator/formErrorUtils.ts`
   L56/138/175/193）：`.ant-form-item-has-error` 在 antd 6.6 仍由 `FormItem/ItemHolder.js:78`
   输出，但属非公开实现细节。选择器字符串收拢为单一常量；新增「哨兵测试」：
   渲染一个校验失败的 Form.Item，断言该类名存在于 DOM——antd 未来升级移除该类名时
   测试显式失败，不再静默失效。定位行为本身不变。
2. **登录回调页站内跳转 SPA 化**（P2，4 处 `window.location.href = '/user/login'`）：
   `pages/user/login/wecom-callback.tsx:131`、`pages/user/login/dingtalk-callback.tsx:185/243/280`
   改用 umi `history.push`。登录成功后的整页跳转/replace（会话重建场景）与 iframe
   防套壳的 `window.top` 操作**不在修改范围**（刻意行为）。
3. **partner-detail 路由解析结构化**（P2，`pages/partners/partner-detail.tsx:127-148`）：
   弃用 `pathname.includes('/suppliers')` 子串匹配，按实际路由段结构解析 roleType/
   roleLabel/listUrl；解析结果与现有行为逐分支等价。
4. **BusinessTagModal effect 依赖缺项**（P3，`components/business-tag/BusinessTagModal.tsx:91-96`）：
   消除 exhaustive-deps 违例；**保留**「每次打开弹窗重新拉取选项」的现有行为。
5. **enterprise-resources 查询参数读取规范化**（P3，`pages/enterprise-resources/index.tsx:69`）：
   渲染期裸 `new URLSearchParams(location.search)` 改 `useSearchParams`（或等价 umi API）。
6. **tabCloseGuard 静态 Modal.confirm 兜底**（P3，`components/layout/tabCloseGuard.ts:78/111`）：
   确认 `AppFeedbackBridge`（app.tsx:146）挂载时序先于 guard 可触发；确保运行时不会
   落到 antd 静态 `Modal.confirm` 分支（丢定制主题）；若时序无法保证，把兜底改为
   显式告警并走桥接实例。

## 约束

- 只改上述 6 项；不改业务语义与用户可见行为（站内跳转 SPA 化的导航语义变化除外）。
- 不触碰 `web/src/components/ui/master-data-template/`（用户工作区在改）。
- 提交时只暂存本任务文件，保留用户未提交改动。

## Acceptance Criteria

- [x] formErrorUtils 选择器常量化 + 哨兵测试存在且通过。
- [x] 4 处登录回调跳转走 history，grep 无站内 `window.location.href`（特殊场景白名单除外）。
- [x] partner-detail 各路由分支（客户/供应商/散客等）解析结果与改动前等价（定向测试覆盖）。
- [x] BusinessTagModal 无 exhaustive-deps 违例，弹窗打开仍重拉选项（既有测试保持绿）。
- [x] enterprise-resources 无渲染期 URLSearchParams 构造。
- [x] tabCloseGuard 运行时无静态 Modal.confirm 路径（附时序确认结论）。
- [x] 受影响文件定向 vitest 全绿；`pnpm --dir web tsc`、`pnpm --dir web biome:lint` 通过；
      收尾 `pnpm run check:fast` 通过。

## 验收结果（2026-09-20）

- 提交 `0b520206`（9 文件）。tabCloseGuard 时序结论：AppFeedbackBridge 挂载于
  ProLayout childrenRender，先于任何用户交互，静态分支运行时不可达；兜底改为
  原生 confirm 降级（无原生 confirm 时 onCancel 中止，保脏数据）。
- TagsView 测试改为经 setAppFeedback 注入桥接实例（与生产同链路）。
- check:fast 全绿：855 passed / 12 skipped（含用户工作区在改用例）。

## Notes

- 审计来源：2026-09-19 两个只读审计代理报告（会话 79 之后）。
- 「记录即可」项（access.ts 硬编码映射、TagsView 整页刷新、登录整页跳转）不在本任务。
