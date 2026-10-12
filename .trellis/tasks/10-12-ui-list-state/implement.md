# 执行计划（尚未批准实施）

- [x] 关联审查编号、证据与目标文件。
- [ ] 取得总任务产品范围选择，PRD收敛并呈现最终规划，等待后续明确批准。
- [ ] 按本任务design核对真实调用/报文/布局，必要技术研究只读产品。
- [ ] 主会话在批准后启动本子任务，dispatch trellis-implement；明确文件所有权，同文件不并发。
- [ ] 实现后补触发场景及相邻路径回归，dispatch trellis-check。
- [ ] 定向验证、修改文件Biome、必要tsc及git diff --check；按共同阶段计划提交。
- [ ] 更新实际实现的spec与能力导航，验收通过再归档。

## 最小验证

pnpm --dir web exec vitest run src/components/ui/order-list-template/OrderListTemplate.test.tsx src/components/layout/TagsView.test.tsx；追加有真实跨路由卸载的快照生命周期、筛选及假计时器回归。
`pnpm --dir web tsc` 按类型影响运行；最终跨公共层集成采用 `pnpm run check:web`。普通UI修改不固定运行build，后端未变不重复服务端门禁。

## 回退及协调

每组以独立提交可逆回退，禁止git reset覆盖用户工作。若发现新业务契约或产品范围差异，回到规划，不偷偷追加兼容/回退。仍处planning时不执行产品写入或实施。
