# 浏览器验收

## 方法

在隔离 worktree `/tmp/roncin-se-container-worktree` 启动独立 Vite 8016，使用真实 React 页面与 Chromium。所有 `/api/**` 请求均由 Playwright stub，禁止真实业务落库；不读取任何账号秘密。

测试文件 `web/tests/e2e/sea-container-entry-permission.e2e.ts`，7 项：

- 1366×768 / 1920×1080：直接从详情打开普通实际箱，新增箱后确认实际摘要更新，原未保存订单货物描述保持。
- 共享箱列表 403：显示明确权限错误，不展示正常零空态和创建空态。
- 候选第二页 5xx：独立重试后，原手工分配件数 37 保持。
- 新建未保存：实际箱区显示保存后维护引导，不提供详情写入入口。
- LCL / 散杂：不提供普通独占箱新增；拼箱允许共享箱，散杂禁止共享箱。

## 验证边界

这里只验证浏览器交互与数据流；权限真正是否拒绝由后端 HTTP 鉴权回归证明，不能以全权限 stub 代替。未对真实生产数据、跨订单完整分配守恒流程进行操作。

## 复跑

```bash
RONCIN_WEB_BASE_URL=http://127.0.0.1:8016 RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/roncin-se-container-artifacts pnpm --dir web exec playwright test tests/e2e/sea-container-entry-permission.e2e.ts --workers=1
```

产物位于 `/tmp/roncin-se-container-artifacts`，不进入 Git。Vite 本次独立启动，验收完成后停止，不影响原工作树服务。

## 最终结果

最终单轮 7/7 通过：1366 实际箱写入保留订单输入 10.5s；1920 13.9s；共享箱403 6.5s；候选失败重试保留草稿 7.0s；新建引导 4.6s；LCL限制 5.5s；散杂限制 5.6s；整轮耗时 54.5s。所有用例无真实业务写入。
`pnpm --dir web exec biome lint tests/e2e/sea-container-entry-permission.e2e.ts` 和 `git diff --check` 通过。
