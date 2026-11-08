# 实测证据与验证记录

## 只读核验

- 核验订单：SE2026110600004。
- 数据库订单 termination_status = TERMINATED。
- 共 4 条 ENDED 关联，前三条为旧改配关系；仅本次退关行 ended_reason = 订单退关 且 ended_at = orders.terminated_at。
- 未修改此订单或数据库数据。
- 已有独立集成测试库可连接；测试使用该库的隔离 Schema 与正式迁移，不使用业务库作为集成测试目标。

## 根因与防复发

- 跨层契约缺口：结束活动关联的生命周期写入正确，但聚合读取隐含「订单必有活动关联」。
- 覆盖缺口：只断言退关写入和关联结束，未同时断言退关后详情所依赖的单证读取。
- 前端相邻问题：状态操作后详情刷新不等于同步变动检查参考点；以表单旧版本继续检查会将本人操作误报为外部变更。
- 预防：真实生命周期集成回归覆盖写后重读；前端使用确切写响应识别本人操作，保留外部变更和输入保护；规范见 sea-terminated-order-read.md。

## 执行结果

- 修复后的 `NewSeaDocumentRepo(...).GetSeaOrderDocuments` 已直接读取实测订单 SE2026110600004，通过设置 `default_transaction_read_only=on` 强制数据库只读；返回 TERMINATED / DIRECT，MBL 存在，allowedActions 数量为 0。临时验证程序已清理。
- 首轮定向：真实库 `TestSeaDocumentTerminationPostgres` 通过（DIRECT/HOUSE 两个子用例，2.537 秒）；前端 6 文件 66 测试通过（8.68 秒），tsc、修改文件 Biome、diff --check 通过。
- 独立 review 找到本人写成功但刷新未完成时轮询先报变化的竞态，已交回实施修复，完成后复核。
- 首轮 check:fast 加载整个开发 .env.local，导致生产默认配置用例受 OTLP 开发开关影响；干净环境定向 `TestProductionConfigUsesSafeDefaults` 已通过（0.051 秒）。最终门禁改为仅读取并注入专用测试库连接，避免开发运行配置污染。
- review 补修：本人写入轮询竞态、更晚外部版本立即提醒、终态关账与迟到身份守卫。3 文件 52 项通过（13.22 秒）；tsc/Biome/diff --check 通过。
- 独立 review 通过，无剩余阻塞项；独立前端回归 3 文件 38 项通过（13.37 秒），详见 review.md。
- 首轮全量前端：388 个测试文件通过、1 跳过；2078 项通过、11 跳过。最终修订后的门禁待补录。
