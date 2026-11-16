# 方向隔离实施验证记录

## 已执行

- 2026-11-16：新增 `server/internal/data/order_direction_migration_integration_test.go`，
  在 SQL 初版实现后补针对性验证，没有先写测试驱动功能。
- 正式 PostgreSQL 前态→目标迁移实际运行：
  `TestOrderMasterDirectionMigrationPreservesHistoricalRowsPostgres`、
  `TestOrderMasterDirectionMigrationRejectsUnclassifiableRowsPostgres`、
  `TestOrderMasterDirectionDatabaseConstraintsPostgres`，共 3 主测试，13 异常子场景及
  2 主单约束子场景，全部 PASS，无 SKIP；最终截止目标版本目录路径重跑 36.05 秒。
- 使用 `getIntegrationDataBeforeMigration` 在随机隔离 Schema 执行正式迁移；cleanup
  只清理该 Schema，没有执行开发数据清空/重建或 Ent 自动建表。
- 测试进程内通过本地环境注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`，未打印或提交连接串。
- 测试验证：四类型实际历史方向回填、仅已结束关联、海运不可变版本身份；原列完整快照
  保留及重复迁移；未知/孤立/混向/类型方向不符/跨公司订单与关联/历史版本公司错误拒绝；
  无半完成列与 revision；同号跨方向/跨公司可用，同方向重复、错误海空类型和空身份拒绝，
  三张表身份无默认值。

## 审阅中修正

- SQL 初版比较 `EXPORT`，但 `orders.trade_direction` 实际持久化为 `export/import`；
  已修正为真实值。
- SQL 初版将全部主单默认回填出口，遗漏孤立身份与已存在一致进口关系；已改为全部历史关系
  唯一方向聚合，未知/混向/孤立显式停止，并检查海运版本公司归属。
- 首次实际测试夹具遗漏船司必填中文名，修正夹具后上述测试实际通过。
- 测试随后将 Apply 目录限定为正式前缀截至目标迁移，避免后续独立内容迁移改变列形使当前
  全行保留断言发生假失败；最终目录路径已实际重跑上述 3 主测试通过。

## 前端定向验证

- 4 个候选调用源已显式提交 `businessType`，不为缺失请求身份默认出口；已同步原候选流程的
  3 处请求断言（初查、切换船司仍为 SE、航空已有主单查询为 AE）。
- 定向 Vitest 共 5 文件 27 用例最终全部 PASS：`air-create-layout.test.tsx`、
  `sea-create-candidate.test.tsx`、`HouseBillNumberField.candidate.test.tsx`、
  `split.test.tsx`、`detail-reassignment-refresh.test.tsx`。航空新增断言曾误写现有下拉的
  分页大小，改为仅校验本次新增的业务类型合同后，单文件 7 用例重跑通过。
- 7 个修改文件 Biome、`pnpm --dir web tsc` 及 3 个修改组件的 antd lint 通过，
  antd lint 零 issue、零跳过；`git diff --check` 通过。
- 完整纯包 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data/ent/migrate`
  全部 PASS；受影响 `biz/data/service` 的 `go vet` PASS。

## 开发库升级就绪核对

- 实际开发库只读预检：1 张海运孤立主单，保留 1 条海运不可变版本；无主单关联、分单、
  锁单或其他订单引用。海空混向/类型方向错配/跨公司关联及版本公司错误计数均为 0。
- 按已批准设计，无法判定身份的主单必须阻止迁移。当前未在开发库应用本组迁移，没有删除、
  重建或默认分类该记录；先继续随机隔离 Schema 验证。完整实现完成后应用开发库前须由
  用户明确该记录的真实业务归属或另行批准具体处理，不为通过升级而放宽正式规则。
- 本报告不保存实际业务单号、主体、ID 或原始数据。

## 尚待

- 实现代理已完成方向门禁与出口相邻真实 PostgreSQL 回归，最终全部通过、无 SKIP；
  生成元数据及 Ent/API/web 生成幂等通过，详见 `direction-core-validation.md`。
- 独立检查代理已全范围复核并自修共享正文并发、拆票/改配执行门禁；两新增真实 PostgreSQL
  主测试通过，最后两类错误语义/唯一索引映射修正也已静态复核通过，详见 `direction-review.md`。
- 本组涉及跨层契约和共享写入，主会话额外执行完整 `pnpm run check:fast`，最终 PASS，
  227.8 秒（WEB 227.8 / SERVER 225.5）。前端 479 文件/2576 用例通过；既有
  `sea-template.test.tsx` 的 1 文件/11 用例仍 SKIP，已与 HEAD 对照确认不是本组新增，
  不计为通过。本组前端定向及真实 PostgreSQL 专项无 SKIP。普通 Go 全量门禁未注入数据库
  开关，不能代替上述真实 PostgreSQL 验证。
- 全量输出保留了仓库既有 Biome 警告/提示及测试环境 getComputedStyle、localhost:3000
  连接拒绝等噪音；本组 7 文件定向 Biome 与 3 组件 antd lint 无问题。没有修改无关文件或
  关闭规则来消除既有输出。未跑构建：本组不涉及构建/依赖/生产入口，后续 PDF 组按合同验收。
- 独立复核、风险匹配门禁和生成物已完成，主会话统一提交并继续后续子任务。
- 四类全流程、进口、航空直单/结构化/PDF 和完整 CSV 仍属后续组，尚未验收。
