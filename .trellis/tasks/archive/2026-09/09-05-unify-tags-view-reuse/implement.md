# 全站 TagsView 页面复用实施计划

## 开始前检查

- [x] 确认当前任务已由用户批准并进入 `in_progress`。
- [x] 重新检查 `git status --short`，记录并避开其他任务的未提交文件。
- [x] 完整读取本任务 PRD、设计、路由审计以及 `implement.jsonl` 引用规范。
- [x] 确认 `web/` 下没有更近的 `AGENTS.md`。

## 第一组：集中页签归属解析

- [x] 在 `routeUtils.ts` 定义明确的菜单页签归属规则与 `resolveTabKey(pathname)` 纯函数。
- [x] 覆盖海运出口、三类客商和集运费用详情；未知路径回退自身 pathname。
- [x] 保持现有 `resolveRouteTitle` 行为和订单动态标题能力。
- [x] 补充路由解析矩阵测试，包括列表、内部页、相似但未知路径和登录路径。

## 第二组：TagsView 按稳定键复用

- [x] 将初始化和路由监听改为按稳定 key upsert，同键更新最新完整 path 和静态标题。
- [x] 将 search/hash 纳入 TagItem.path，但不纳入 key。
- [x] 激活判断、关闭当前判断、图标判断统一使用稳定 key。
- [x] 保持页签顺序与现有关闭、重新加载、键盘操作逻辑。
- [x] 动态标题事件按稳定键定位，并增加当前页面匹配门禁，忽略迟到事件。

## 第三组：组件行为测试

- [x] 模拟海运出口列表 → 新建 → 详情 → 费用 → 列表，断言始终只有一个海运出口页签，
  path/title 随当前页面更新。
- [x] 覆盖同菜单连续打开不同记录、直接访问内部 URL、刷新初始化和查询参数保留。
- [x] 覆盖客户/供应商/国外代理分别复用且互不合并。
- [x] 覆盖费用详情复用费用列表、费用详情跳订单时形成目标菜单独立页签。
- [x] 回归关闭当前、关闭非当前、关闭其他、关闭左右、重新加载和动态订单号标题。

## 第四组：质量检查与独立复核

按风险依次运行：

```bash
pnpm --dir web test -- src/components/layout/TagsView.test.tsx
pnpm --dir web tsc
pnpm --dir web biome:lint src/components/layout/TagsView.tsx src/components/layout/TagsView.test.tsx src/components/layout/routeUtils.ts
git diff --check
```

- [x] 若仓库脚本不接受测试或 Biome 的定向参数，使用 package.json 中现有等价 pnpm 命令，
  不改用 npm/npx，也不关闭规则。
- [x] 调用独立 `trellis-check` 复核 PRD、跨路由状态流、边界和测试有效性。
- [x] 修正检查发现的问题并重跑相关门槛。
- [x] 确认差异只包含本任务文件；其他任务变化不暂存、不提交。

## 规范、提交与回滚

- [x] 将“菜单入口作为稳定页签身份、内部路由只更新 path/title”的约定写入前端规范。
- [ ] 由 Codex 主会话提交一个可验证变更，建议提交信息：
  `fix(web): 统一菜单内部页面页签复用`。
- [ ] 提交后执行 Trellis finish/archive 和开发日志记录。
- [ ] 本任务不自动推送；如用户要求推送，再单独执行。

回滚点为本次单一前端提交。回滚不得覆盖配舱分单展示或其他并行任务的提交与工作区内容。
