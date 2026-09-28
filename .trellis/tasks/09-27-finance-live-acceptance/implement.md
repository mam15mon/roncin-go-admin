# 真实验收执行计划

- [x] 用户批准规划后激活本子任务；检查最终代码、专用环境、公司/权限、端口及 Playwright 浏览器。只检查环境变量存在性，不输出秘密。
- [x] 审阅一次性编排实际启动/清理流程，确认只操作本任务创建的隔离资源；凭据由安全环境变量注入。开发库保持不变。
- [x] 核对脚本与当前契约，完成必要修复后检查语法：

```bash
node --check scripts/acceptance-finance-bill-batch.mjs
node --check scripts/acceptance-finance-payable.mjs
node --check scripts/acceptance-finance-foreign-currency.mjs
node --check scripts/run-acceptance-finance-disposable.mjs
```

- [x] 扩展真实浏览器场景，使用同一订单和专用数据完成关键 UI 操作、刷新重读和新费用口径相邻验证；实现后补有价值的针对性断言。
- [x] 显式环境准备完成后，执行完整数据库门禁及最终两阶段运行时编排；根据实际日志确认三类 HTTP 与浏览器均被调用，不能只看顶层退出码。完整门禁见 R2，最终运行时见 R29：

```bash
pnpm run acceptance:finance:disposable
```

- [x] 分阶段排查时，仅在任务专属一次性环境执行以下入口；实际使用编排显式 `--runtime-only` 与 `--stage-b-only`：

```bash
pnpm run acceptance:finance-bill-batch -- --apply
pnpm run acceptance:finance-payable -- --apply
pnpm run acceptance:finance:foreign-currency
pnpm --dir web exec playwright test tests/e2e/finance-bill-batch.e2e.ts
```

- [x] 复用现有 e2e 文件，财务验收入口定向运行该文件；修复阻塞后 R29 重跑 HTTP/UI 全链。
- [x] 修改文件 Biome、tsc 与脚本语法检查通过；父任务统筹最终 pnpm run check:fast。
- [x] research/verification.md 按场景记录输入期望/实际金额、链路编号、SHA、运行时间、命令、PASS/FAIL/SKIP/未执行、截图/trace 位置及资源回收结果。
- [x] Trellis 检查代理复核无降低断言、无模拟关键请求、无凭据入库；git diff --check 通过。由父任务统一提交收口。
