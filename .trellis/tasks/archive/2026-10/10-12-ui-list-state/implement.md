# 执行计划（最终方案已批准）

- [x] 关联审查编号、证据与目标文件。
- [x] 产品范围已确认，PRD收敛完成。
- [x] 用户已回复“好的”，明确批准855e467d最新最终规划。
- [x] 按本任务design核对真实调用/报文/布局，必要技术研究只读产品。
- [x] 主会话在批准后启动本子任务，dispatch trellis-implement；明确文件所有权，同文件不并发。
- [x] 实现后补触发场景及相邻路径回归，dispatch trellis-check。
- [x] 定向验证、修改文件Biome、必要tsc及git diff --check；按共同阶段计划提交。
- [x] 实际实现的spec与能力导航同步或无变更判断完成；验收通过，进入归档。

## 最小验证

pnpm --dir web exec vitest run src/components/ui/order-list-template/OrderListTemplate.test.tsx src/components/layout/TagsView.test.tsx；追加有真实跨路由卸载的快照生命周期、筛选及假计时器回归。
`pnpm --dir web tsc` 按类型影响运行；最终跨公共层集成采用 `pnpm run check:web`。普通UI修改不固定运行build，后端未变不重复服务端门禁。

## 回退及协调

每组以独立提交可逆回退，禁止git reset覆盖用户工作。若发现新业务契约或产品范围差异，回到规划，不偷偷追加兼容/回退。仍处planning时不执行产品写入或实施。

## 执行前置条件

保存保护和计息移除完成并提交后启动，与保存保护共享财务/客商调用文件，依照总任务顺序串行交付。

## 本组验证记录

17suite88例定向回归（最后订单2例定向重跑7.63秒）、修改文件Biome、tsc、architecture、diff均通过。浏览器7项通过；完整证据见research/browser-verification.json，实现及独立检查见research/list-implementation.md和list-check.md。公共查询恢复契约与能力导航已同步，最终跨任务check:web后归档。

## 最终集成验收

2026-10-12完整`pnpm run check:web`退出码0；218测试文件、1181例通过，保留既有1文件/12例跳过。全部12项及62项浏览器证据通过，集成结果见同月archive中父任务research/final-acceptance.md。
