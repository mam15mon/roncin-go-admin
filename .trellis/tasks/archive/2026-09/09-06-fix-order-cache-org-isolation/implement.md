# 实施计划：修复订单缓存复用与组织切换隔离

## 阶段 1：新建与详情 Hook

- [x] 调整 `use-order-create-options.ts`：空关键字返回当前组织首批地点；非空搜索校验活动组织；加载错误绑定组织与业务配置。
- [x] 调整 `use-order-detail-data.ts`：空关键字返回当前详情有效地点；非空搜索校验活动组织；错误绑定组织与订单。
- [x] 补充对应 Hook 单测，覆盖正常、空白关键字、组织切换迟到结果和失败状态切换。

## 阶段 2：列表搜索结果隔离

- [x] 调整 `list-resources.ts` 中客户、港口、地点、承运人和人员搜索：无组织不请求、迟到结果返回空数组、仅当前组织允许写 state。
- [x] 扩展 `list-resources.test.ts`，同时断言 Hook state 和搜索 Promise 返回值，不再只检查 `customerMap`。

## 阶段 3：定向验证与独立检查

- [x] 运行受影响测试：
  `pnpm --dir web exec vitest run src/pages/orders/use-order-create-options.test.ts src/pages/orders/use-order-detail-data.test.ts src/pages/orders/list-resources.test.ts`
- [x] 对修改文件运行 Biome 定向检查。
- [x] 运行目标 Playwright：
  `node --env-file=.env.local scripts/run-with-env.mjs pnpm --dir web exec playwright test tests/e2e/order-create-loading.e2e.ts`
- [x] 由独立检查代理核对 PRD、真实差异、跨组织返回路径和测试有效性；未发现需要自修的代码问题。

独立复核证据：

- 三组定向 Vitest：3 个测试文件、28 个测试全部通过。
- 六个修改文件的 Biome 定向检查通过，未产生自动修复。
- `pnpm --dir web tsc` 与 `git diff --check` 通过。
- 目标 Playwright 用例通过（1/1，约 35 秒），保留并验证了二次进入时主数据、港口、机场、币种和人员新增请求数均为 0 的严格断言。

## 阶段 4：最终验收与提交

- [x] 最终执行一次 `pnpm run check:web`，包含生成一致性、`pnpm --dir web lint`（Biome + TypeScript）和全量 Vitest：79 个测试文件、323 个测试全部通过。
- [x] 执行 `git diff --check`，确认没有生成物、临时产物或无关改动。
- [x] 更新任务清单，并将组织级异步联想的请求身份双重校验规则写入前端 Hook 规范。
- [ ] 使用 Conventional Commit 提交本组代码与任务文档，建议提交信息：`fix(web): 完善订单缓存复用与组织隔离`。
- [ ] 完成 Trellis finish/archive 和开发日志记录。

## 风险与回滚点

- 空关键字行为会影响地点下拉初始候选项；单测和 Playwright 必须证明首批数据仍可见且不产生额外请求。
- 搜索返回保护必须覆盖返回值而不仅是 Hook state；单测需要直接检查迟到 Promise 的解析结果。
- 错误状态改造不得使当前组织真实失败重新进入无限 loading；测试需同时验证当前错误仍正常暴露。
