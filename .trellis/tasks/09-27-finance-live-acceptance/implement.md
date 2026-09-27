# 真实验收执行计划

- [ ] 用户批准规划后激活本子任务；检查最终代码、专用环境、公司/权限、端口及 Playwright 浏览器。只检查环境变量存在性，不输出秘密。
- [ ] 审阅一次性编排实际启动/清理流程，确认只操作本任务创建的隔离资源；凭据由安全环境变量注入。开发库保持不变。
- [ ] 核对脚本与当前契约，完成必要修复后检查语法：

```bash
node --check scripts/acceptance-finance-bill-batch.mjs
node --check scripts/acceptance-finance-payable.mjs
node --check scripts/acceptance-finance-foreign-currency.mjs
node --check scripts/run-acceptance-finance-disposable.mjs
```

- [ ] 扩展真实浏览器场景，使用同一订单和专用数据完成关键 UI 操作、刷新重读和新费用口径相邻验证；实现后补有价值的针对性断言。
- [ ] 显式环境准备完成后，优先执行已有全套编排；根据实际日志确认三类 HTTP 与浏览器均被调用，不能只看顶层退出码：

```bash
pnpm run acceptance:finance:disposable
```

- [ ] 分阶段排查时，可在同一隔离运行环境执行以下入口（外币脚本前置数据单独按源码准备；不得直接指向开发库）：

```bash
pnpm run acceptance:finance-bill-batch -- --apply
pnpm run acceptance:finance-payable -- --apply
pnpm run acceptance:finance:foreign-currency
pnpm --dir web exec playwright test tests/e2e/finance-bill-batch.e2e.ts
```

- [ ] 若新增独立 e2e 文件，将其加入上述定向命令和编排调用；修复阻塞后重跑相关 HTTP/UI 场景。
- [ ] 改前端时执行受影响 Vitest、修改文件 Biome 与按类型影响选择 tsc；后端修复执行定向真实库验证。最终 pnpm run check:fast 由父任务统筹。
- [ ] research/verification.md 按场景写明输入期望/实际金额、链路编号、SHA、运行时间、命令、PASS/FAIL/SKIP/未执行、截图/trace 位置及资源回收结果。
- [ ] Trellis 检查代理复核无降低断言、无模拟关键请求、无凭据入库；git diff --check 后提交，交父任务收口。
