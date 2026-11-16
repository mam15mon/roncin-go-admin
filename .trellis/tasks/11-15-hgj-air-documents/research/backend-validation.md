# 航空模式与结构化单证：后端实施验证

## 实现交付

- Proto：`air_export.proto` 增加航空独立正文、DIRECT/HOUSE、可选真实 HAWB、主分正文哈希及历史；`order_lock.proto` 增加航空审核资料指纹。
- Biz：新增纯 Go 航空印字结构、长度/纯日期/声明/十进制校验。金额、单价、重量独立；计重运费与其他费用摘要使用 decimal。AS ARRANGED 要求币种与付款目标，相关金额/总计为空，不能冒充零。
- Data：模式锁内复验，DIRECT 不建 HAWB，HOUSE 保留本主单永久占号；仅主单 ID + 预期版本可建票，运输/正文从锁内权威主单读取。普通保存不能切模式、改 HAWB 号或换主单；正文省略保持、`{}` 清空。
- 正文修改分别推进 Master 或 House，House 修改同时推进 Link；主分各留独立历史/哈希。审核快照含模式、主单、可选分单及 Link 版本，旧资料指纹拒绝锁单。
- 重量继续 ORDER/MASTER 与 PLANNED/ACTUAL 独立；MASTER 指纹以不可变 Link ID 与模式表达成员，单纯印字修改推进 Link.version 不使重量过期；AE 重量入口明确拒绝 AI 主单。
- Ent Schema、正式迁移 `20261116140000_air_waybill_content_and_mode.sql`、开发种子与必要夹具同步。有效旧关系回填 HOUSE，自由文本完整移入当前分单及各版本备注；主单原运输备注与历史载体不改写；缺失/错挂分单或历史身份异常停止迁移。
- 生成顺序：Proto → `make -C server api` → `pnpm run generate:web-client`；Ent 用 `go -C server generate ./internal/data/ent`。未手改生成代码。

## 数据库与迁移证据

通过 Node 的 `.env.local` 进程内环境把 `DATABASE_SOURCE` 传为 `RONCIN_INTEGRATION_DATABASE_SOURCE`，未打印连接串。每例使用 `getIntegrationData` / `getIntegrationDataBeforeMigration` 随机隔离 Schema，执行正式有序 SQL 迁移，清理只删除该 Schema，未向开发业务 Schema 应用迁移或清空开发数据。

Ent 差异在独立随机 Schema 上执行前置正式迁移后，以 `client.Schema.WriteTo` 生成到 `/tmp/roncin-air-schema-generated.sql`。审阅航空列、JSONB、哈希及模式 CHECK 差异，正式迁移补充一次性数据转换和异常预检；未采纳生成器列出的无关历史类型/约束漂移。

## 已完成检查

| 命令 / 检查 | 结果 | 实际范围 |
| --- | --- | --- |
| `go -C server test -p 32 ./internal/biz ./internal/service ./cmd/sync-dev ./internal/data/ent/migrate` | PASS | Biz、Service 完整纯包测试，种子与 Ent 元数据测试 |
| `go -C server vet ./...` | PASS | 全服务端静态检查 |
| `go -C server test -p 32 ./internal/data -run 'TestAir(Direct\|Waybill\|Weight)' -count=1 -v`（注入集成环境） | PASS，24.699s | 新增双模式、仅 ID 关联/幂等、主分独立正文/清空/历史、旧审核指纹、审计回滚、永久号、正式迁移保留与异常停止、重量费用；无 SKIP |
| `go -C server test -p 32 ./internal/data -run 'TestAir\|TestOrderMasterDirection\|TestOrderLock_PostgresFlows' -count=1 -v`（注入集成环境） | PASS，107.544s | 航空全套、方向迁移/异常矩阵/身份约束、通用六类审核锁与并发/回滚相邻回归；无 SKIP |
| `go -C server test -p 32 ./internal/data -run TestAirDirectCreateEditRestoreWeightFeesAndReviewPostgres -count=1 -v`（注入集成环境） | PASS，4.525s | 最后补充 AE 重量权限拒读同公司 AI 主单；直单真实闭环再次通过 |
| `go -C server test -p 32 ./internal/data -run 'TestFeeLedgerMasterNoPostgres\|TestFinanceReportAuthoritativeAirAndImportDatePostgres' -count=1 -v`（注入集成环境） | PASS，10.213s | 更新为真实 DIRECT 的财务读取夹具，权威主单号与空运/进口经营日期相邻路径；无 SKIP |
| `make -C server api` + `go -C server generate ./internal/data/ent` + `pnpm run generate:web-client` 前后 962 个生成文件 SHA256 比较 | PASS | 重跑内容完全一致 |
| `git diff --check` | PASS | 实施阶段变更空白检查 |

新增 PostgreSQL 核心用例：

- `TestAirDirectCreateEditRestoreWeightFeesAndReviewPostgres`：真实直单创建/普通编辑/原票恢复，无 HAWB/无分单历史，MASTER 不能代 ORDER 确认，本票重量费用与审核快照、锁后拒写。
- `TestAirDirectConcurrentAttachAndHistoricalModePostgres`：两个并发新挂直单只能成功一票，DIRECT 历史主单不能混入 HOUSE。
- `TestAirWaybillIndependentContentVersionsAndReviewPostgres`：完整中文长文/换行、显式零声明、主分独立版本、仅 ID 关联/幂等、House/Link 基线、显式清空、Master 备注与正文隔离，印字修改不使 MASTER 重量过期。
- `TestAirWaybillBodyAuditFailureAndPermanentNumberPostgres`：主分正文审计失败全量回滚；终止后的 HAWB 同主单永久占号，跨主单可重。
- `TestAirWaybillMigrationPreservesTextAndHistoryPostgres`：升级前态包含 HTML、反斜杠、引号、Unicode 行分隔符、3500 字旧正文；逐字符保留，SQL 回填哈希与 Go 相同，原时间/版本/主单运输备注和历史字节不变，迁移重放只一条 revision。
- `TestAirWaybillMigrationStopsInvalidLegacyRelationsPostgres`：缺分单、历史订单不符、历史号码不符均报错并保留迁移前正文/DDL，无半迁移。

## 交接边界

服务端产品与生成写权已交独立后端检查代理；普通订单输入中可选 `expectedMasterVersion` 的锁内复验与针对性回归由检查代理负责补齐，本实施代理不再并行改写该处。

本记录只覆盖当前航空基础组。SI/AI 启用、航空制作专页/PDF、完整筛选导出和父任务整体验收仍属后续组。未执行真实外站写入或通知发送。最终全栈门禁与普通提交由主会话统合。
