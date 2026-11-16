# 四类订单列表实施与验证

## 改动

- 空运列表复用公共主分单两行展示，明确 MAWB/HAWB；DIRECT、HOUSE缺号及未知结构各自展示，不借海运结构推导航空关系。
- AE/AI增加航司、ETD/ETA、本票确认计费重及可选单证状态列；件重尺默认展开。SI/AI默认显示ETA，现有服务派生进度列称“进口服务进度”。SE默认列不变，个人偏好继续按品类/用户/公司隔离。
- `AirOrderInfo.weight_summaries`为只读repeated摘要：source、confirmed_weight_kg、confirmation_stale、confirmed_at。PLANNED/ACTUAL独立返回最新确认，缺少确认时不返回该来源；不混用MASTER，不选择费用采用值。
- data按已授权订单及公司批量读取最新确认和货量事实，增加固定4次查询/公司，不逐行HTTP请求。重量事实构造从旧预览路径提取共用，过期仍调用`biz.BuildAirWeightPreview`。无Schema或写规则变化。
- 列表/导出共用映射，CSV增加两个来源的确认值和状态。API绑定、OpenAPI及前端客户端通过`make -C server api`、`pnpm run generate:web-client`生成。

## 已执行验证

- `pnpm --dir web exec vitest run src/pages/orders/list-query.air-weight.test.ts src/pages/orders/air-list-columns.test.tsx src/pages/orders/list-query.test.ts src/pages/orders/list-export.test.ts src/components/ui/order-list-template/OrderListTemplate.bill-relations.test.tsx src/components/ui/order-list-template/OrderListTemplate.default-columns.test.tsx src/components/ui/order-list-template/OrderListTemplate.air-columns.test.tsx`：7文件65项通过。
- `pnpm --dir web tsc`：通过（模板代理完成variant属性后的整合状态）。
- 本组11个修改/新增前端文件Biome检查通过；`antd lint`航空列零问题；`git diff --check`通过。
- `go -C server test ./internal/service ./internal/biz -run 'TestAirOrderWeightSummaryWirePresence|TestAirWeight|TestAirOrderOptionalHouseAndZeroWire' -count=1`通过；`go -C server vet ./internal/data ./internal/service ./internal/biz`通过。
- 显式注入本机私有连接运行`go -C server test ./internal/data -run TestAirWeightConfirmationAndFeeSourcePostgres -count=1 -v`：AE/AI两个真实PG用例通过，零跳过，随机隔离Schema按正式迁移创建后清理。新增断言覆盖同源二次确认选最新值、PLANNED保持新鲜而ACTUAL过期、无确认票不伪造值及摘要时间存在，既有费用来源/历史/并发回归保持通过。

## 整合说明

- 早期前端测试曾因并行模板文件尚未创建而解析失败，待模板落地后已完整重跑通过；这不作为最终失败保留。
- 尚未在本代理执行完整check:fast或浏览器验收，由主会话统一安排。
- 现有`attachAirOrders`对不同主单的逐主单装饰读取未在本次扩张重构；新增重量摘要自身是固定批量查询，不宣称既有整套主单聚合已全部消除逐主单查询。
