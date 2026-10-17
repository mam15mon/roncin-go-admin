# 浏览器复核记录

使用 Playwright Chromium 与开发期 Vite 的真实模块；仅拦截 HTTP，未向开发库提交密码或财务写入。
浏览器时区显式为 Asia/Shanghai。临时脚本 `/tmp/roncin-audit-browser.cjs`；原始结果
`/tmp/roncin-audit-browser-result.json`，CSV 样本 `/tmp/roncin-audit-bill-sample.csv`。

## R1：真实重置密码组件与统一请求层

- 挂载真实 `ResetPasswordModal`、Antd App、反馈桥和请求客户端。
- 输入合法密码后模拟 HTTP 409，显示后端冲突文案；输入保持原值，close=0、reload=0，无成功消息。
- 同一弹窗不重建，重试返回真实成功形态 `{success:true}`，reload=1，成功流程恢复。

## R2：真实个人提成抽屉与工作区边界

- 在真实 `OrganizationWorkspace`、初始状态 Context、MemoryRouter 下挂载 `MyCommissionDrawer`。
- A 公司请求成功后显示 A 的私有提成；切 B 并挂起网络响应时，A 记录已消失。
- B 返回后显示 B 记录；再切回 A，重新请求并显示 A，不残留 B。
- 整个过程中浏览器 pageerror=0。

## R3：真实业务列映射与本地时间

- 在同一浏览器导入 `utils/csv.ts` 和账单 `ledgerExport.ts`，使用多币种、八位尾零金额、
  小数汇率、UTC 时间和带公式前缀的账单编号构造样本。
- 输出包含 `15362.50`、独立 `USD` / `CNY`，以及本地 `2026-10-17 08:00`。
- `=SUM(1,2)` 文本加单引号保护且逗号被正确引用。
- 六领域逐列值、选择/当前页范围及下载清理由实现后的定向回归进一步验证。
