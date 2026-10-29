# 浏览器验收记录

## 环境与方法

- 使用 Playwright Chromium headless，真实应用布局、表单与 React Query；所有 `/api/**` 请求通过 `page.route` 返回固定测试数据，无真实登录凭据和数据库写入。
- 原共享 Vite 8001 在运行中无响应，改用本任务专用 Vite `http://127.0.0.1:8015`；没有停止或修改原服务。
- 测试文件：`web/tests/e2e/sea-export-entry-experience.e2e.ts`。
- 浏览器产物目录：`/tmp/roncin-se-entry-artifacts`，不提交截图/trace。

## 验收覆盖

1. 1366×768、1920×1080：客户与港口搜索、方向键选择、Enter/Tab；中文 composition 事件仅作 smoke，不作为真实输入法误提交的充分证据；ETD 后移保留原 ETA 并显示校验错误；底部创建入口可达。
2. 延迟客户人员响应，手动指定操作员后再放行响应，默认值不得覆盖；业务人员自动带出后直接刷新，人员草稿完整恢复。
3. 新建请求在途冻结输入（含金额/币种、件数/单位、危险品、船公司、HBL、主分单正文）；模拟 503 进入待确认，刷新后使用同一幂等键与完整载荷确认，成功进入已创建订单且不弹放弃编辑确认框。

4. 空表提交后定位并聚焦首个必填客户字段。

## 结果

- 原 4 项集成用例整轮通过，耗时 51.9 秒。
- 扩大为 HOUSE + 危险品 + 金额/单位/主分单正文的冻结断言后，创建恢复用例定向通过，耗时 50.8 秒（同期开发与检查进程并行，较原 16.8 秒慢）。
- 新增空表错误定位及焦点用例定向通过，耗时 5.7 秒。
- 当前文件共 5 项；本记录区分原整轮与后续定向运行，不将未重新整轮运行的结果表述为最终 5 项整轮通过。
- `pnpm --dir web exec biome lint tests/e2e/sea-export-entry-experience.e2e.ts`、`git diff --check` 通过。

复跑：

```bash
RONCIN_WEB_BASE_URL=http://127.0.0.1:8015 RONCIN_ACCEPTANCE_ARTIFACT_DIR=/tmp/roncin-se-entry-artifacts/review pnpm --dir web exec playwright test tests/e2e/sea-export-entry-experience.e2e.ts --workers=1
```

## 验证边界

- API stub 验证 UI 集成及请求载荷，不替代真实后端事务验收。
- 中文输入法通过浏览器 composition/key 事件模拟，未使用操作系统真实输入法，也没有验证浏览器原生组合键默认行为；需要真实输入法人工验收。
- 本轮不覆盖完整海运订单财务、拆票、改配或放货生命周期；相邻路径由定向组件测试与主会话门禁验证。
