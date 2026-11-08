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
- 首轮全量前端：388 个测试文件通过、1 跳过；2078 项通过、11 跳过。
- 最终修订后 `pnpm run check:web` 通过：388 文件通过、1 跳过；2088 项通过、11 跳过，Vitest 159.33 秒。包含权限/Proto 常量、架构、重复检测及前端 lint/type-check。
- 后端全量真实库测试：`server/internal/data` 全包通过，878.375 秒；首轮唯一失败为开发环境变量污染的 `cmd/server` 生产默认配置用例。
- 干净环境 `pnpm run check:server` 完整通过（退出码 0）：复用真实库全包成功结果，`cmd/server` 全包通过（0.060 秒），分层、Proto、重复检测、Go vet 与漏洞门禁均通过。
- 最终验收由 `check:web` + `check:server` 两项完整门禁组成，首轮 check:fast 的环境污染已排除；没有为通过检查修改规则或生产默认配置。
- 独立 review 无剩余阻塞项，修复提交 efd30bd1，按用户授权合并 main 后推送并安全删除修复分支。
