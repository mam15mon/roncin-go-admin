# 验证记录

日期：2026-10-27。全部业务写入验收均使用独立的一次性 PostgreSQL 库与角色，未对开发或生产业务库执行 apply；测试连接信息只保存在仓库外受限文件中。

## 已通过

- `go -C server test -p 32 ./cmd/sync-unlocode`：真实 ZIP/CSV 完整身份、17,524 条覆盖、严格失败矩阵、联合指纹、离线预览及显式 apply 边界通过。
- `TestPortChineseSyncPostgres` 与 `TestIndustryReferenceGlobalSharingPostgres`：独立 PostgreSQL 正式迁移、真实 PASS，无 SKIP；包括旧行补名、已有非空检索键刷新、首次创建、ID及订单引用稳定、幂等、代码/英文/中文/全拼/首字母查询、其他来源冲突、中途写失败整批回滚。
- `go -C server vet ./cmd/sync-unlocode ./internal/data` 通过。
- `pnpm run check:server` 完整通过：Go 测试、vet、Proto、分层、重复扫描、漏洞扫描。本次默认全包检查中依赖额外环境的其他专项测试仍按仓库默认跳过，不能当作其真实数据库验收；本任务相关 PG 专项另行明确执行通过。
- `git diff --check` 通过。
- 完整命令冷启动验收：一次性测试库经 `cmd/migrate` 正式迁移，首次 `sync-unlocode -apply` 新增 17,524、更新 0；重复执行新增 0、更新 17,524、停用 0。SQL 核验中文、启用和非空检索键各 17,524；数据库代码及中英文与 CSV 逐条一致。

## 全栈门禁结果与边界

已运行 `pnpm run check:fast`。最初一轮配套文件硬验收发现 CNMTZ 中文被源候选缩短成“码头”，已纠正为完整地名“码头镇”并重新通过命令测试。

后续两轮全量前端各有 1 条既有海运主单异步断言失败（无本次前端修改）：

- `SeaMasterBillFields.identity.test.tsx` 的“已确认候选切换后查询失败不保留旧确认，并显示可重试错误”；单独重跑整文件 8/8 通过。
- `SeaMasterBillFields.inheritance.test.tsx` 的“主单缺失值也列入差异，采用不会把新票资料当成共享补全”；单独重跑整文件 6/6 通过。

两轮全量运行均为 317 个前端文件通过、1 个失败、1 个既有跳过；前端架构、lint、类型检查完成。不得将 `check:fast` 记为绿色。本任务只修改后端同步与静态字典，按风险匹配以完整后端门禁、真实 PostgreSQL 专项和完整数据落库为验收依据，未扩大到无关 UI 测试的调度稳定性修复。

## 数据质量边界

17,524 条完整中文不意味着全部是官方或行业通行译名；逐行保留 existing/translated 和来源说明，长尾采用模型辅助编制。已有中文来源与按国家批次语义审阅完成，重点纠正普通词误译、跨国误配、汉字原名、港区/岛湾/码头区别；源底表自身的 ES8TC Rejected 等异常在说明中保留。不会用非空、测试通过替代地名语义审定。

## 最终数据版本

- 中文类别：existing 7,357，translated 10,167。已有名称来源为 Wikidata 4,005、GeoNames 2,139、Forwarderkit 1,103、日本官方港口代码表 107、科考站资料 3。
- 日本官方表同码/同县/假名读音复核后纠正 107 项来源或名称（含 JPUCN），不覆盖跨县、合并港或新版更名疑点。最后变更仅是中文源数据，重新执行命令完整测试通过，并在一次性库应用后逐项核验。
- CSV SHA-256：`ab7192891e66487cf90301ecd844f4a7b4b8af03284b384eafc245cbc644c3ab`。
- 联合 source_hash：`26b4a359ae459e29394833fcbd2e13f1fdbe8cc58ba001434f61e434de4d31ff`。
