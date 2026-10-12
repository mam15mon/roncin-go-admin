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

菜单权限/导航定向测试；真实Playwright遍历375/390/767/768/769/1440px，记录整页/表格容器宽度及可见操作。实现前定位根因，避免猜测性CSS。
`pnpm --dir web tsc` 按类型影响运行；最终跨公共层集成采用 `pnpm run check:web`。普通UI修改不固定运行build，后端未变不重复服务端门禁。

## 回退及协调

每组以独立提交可逆回退，禁止git reset覆盖用户工作。若发现新业务契约或产品范围差异，回到规划，不偷偷追加兼容/回退。仍处planning时不执行产品写入或实施。

## 执行前置条件

10-12-ui-keyboard-access完成并提交后启动，复用最新HeaderMenus/global.less，不能并发覆盖。

## 本组验收

菜单4suite11例、订单8suite54例，修改文件Biome、tsc、architecture、diff通过；浏览器24项真实页面与12项详情GET fixture组合全部通过，375/390无整页溢出、768无导航空档。布局只修已定位标题/箱计划及375通知人Segmented，FormRow保持。shared菜单与窄屏规范已同步，完整证据及独立复核在research。最终check:web运行中，验收通过再归档。
