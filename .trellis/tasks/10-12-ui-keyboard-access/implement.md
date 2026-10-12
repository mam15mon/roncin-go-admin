# 执行计划（最终方案已批准）

- [x] 关联审查编号、证据与目标文件。
- [x] 产品范围已确认，PRD收敛完成。
- [x] 用户已回复“好的”，明确批准855e467d最新最终规划。
- [x] 按本任务design核对真实调用/报文/布局，必要技术研究只读产品。
- [x] 主会话在批准后启动本子任务，dispatch trellis-implement；明确文件所有权，同文件不并发。
- [x] 实现后补触发场景及相邻路径回归，dispatch trellis-check。
- [x] 定向验证、修改文件Biome、必要tsc及git diff --check；按共同阶段计划提交。
- [ ] 更新实际实现的spec与能力导航，验收通过再归档。

## 最小验证

pnpm --dir web exec vitest run src/components/layout/HeaderMenus.test.tsx src/components/OrganizationSwitcher.test.tsx src/components/layout/TagsView.test.tsx；SectionCard新增行为回归（实现后）。真实浏览器检验焦点样式，jsdom不能代替计算样式验证。
`pnpm --dir web tsc` 按类型影响运行；最终跨公共层集成采用 `pnpm run check:web`。普通UI修改不固定运行build，后端未变不重复服务端门禁。

## 回退及协调

每组以独立提交可逆回退，禁止git reset覆盖用户工作。若发现新业务契约或产品范围差异，回到规划，不偷偷追加兼容/回退。仍处planning时不执行产品写入或实施。

## 执行前置条件

10-12-ui-list-state完成并提交后启动，共享TagsView相关修改，不能并发覆盖。

## 本组验证记录

7suite64例及最后Esc2例定向重跑、8修改文件Biome、tsc、diff与浏览器11项通过。keyboardAccessible显式opt-in及SectionCard/焦点/冒泡规范已同步spec；独立复核见research/keyboard-check.md。最终跨任务前端门禁后归档。
