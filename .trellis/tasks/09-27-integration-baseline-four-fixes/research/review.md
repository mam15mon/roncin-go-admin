# 存量集成修复审阅记录

## 范围与结论

2026-09-27，审阅五个已修改的服务端集成测试／辅助文件，依据本子任务 PRD、设计、执行计划及角色工作台归属、公共主数据与公司独立配置、数据库和质量规范。审阅发现两处局部断言覆盖缺口，已直接补强；未发现需要修改生产契约、业务实现、正式迁移或生成物的问题。

本审阅只修改 `server/internal/platform/migration/fee_catalog_seed_integration_test.go` 和本记录。三个 data 文件及 `company_boundaries_integration_test.go` 未被本审阅修改，因此父任务已启动的 data 全量集成可以继续。代码修改前已通知父任务暂缓 migration 包编译，修改后已通知代码冻结。

## 已修复的问题

### 公司种子副本代码集合可能被内连接遗漏

- 文件：`server/internal/platform/migration/fee_catalog_seed_integration_test.go`。
- 问题：副本数量检查与 `INNER JOIN` 内容比较合用时，未知代码替换一个应有模板代码仍可能同时满足数量和「已匹配行无差异」断言。
- 修复：模板与两个公司交叉展开后左连接公司副本，断言每个公司的缺失代码数为零。配合原有每公司 122 条数量断言及 `(organization_id, fee_code)` 唯一约束，验证每家公司代码集合精确覆盖模板。

### 本地同码 OF 只保护名称与税务名称

- 文件：同上。
- 问题：原有保留断言只核对费用中文名称与税务名称，没有证明初次种子执行不改其他本地字段。
- 修复：在种子前保存本地 OF 与所关联本地税务的完整 JSON 行，种子后整行比较，包含 ID、全部时间、引用及业务配置字段。后续种子直接重放仍保留原五表快照幂等性断言。

## 其他契约核对

- 工位锚点：历史夹具按 `< 20260922100000` 正式 SQL 前缀初始化；现行 Schema 相邻测试仍拒绝无父节点部门，没有删除或放松生产 CHECK。回填的部门／团队归一、重复执行、断链失败、唯一冲突回滚及清理后自检保持实际可执行。
- 迁移截止点：当前目录中 `20260922100000` 是公司边界迁移，后续第一项为 `20260922130000`；`boundaryOnlyDir` 的截止点确实只保留被测迁移及此前历史。费用种子截止 `< 20260923121000` 同理只执行到 `20260923120000`，且用例之后继续执行完整迁移链。
- 公司费用目录：期望从启用模板集合推导；逐模板核对独立 ID、公司归属与配置、税务文本；完整列表与关键字查询分别验证，跨公司更新拒绝、模板更新不传播及初始化冲突回滚仍在。
- 公司边界快照：仅排除允许重写的 `fee_setting_id`；费用与补录请求的其余整行字段（含时间和原费用状态）仍须逐字保持。新引用额外验证费用代码和费用／税务归属同一公司。字段级差异诊断没有取代整行不变断言。
- 费用种子：两个公司各自 122 条副本，税务记录归属一致，新增副本的完整配置／税务文本比较、本地同码保护、5 个原异常代码和13个后续异常代码、直接重放种子业务值及 ID 不变均受保护。
- 原始四项测试名保持，无删测试、改名绕过筛选或新增 SKIP；生产 SQL、Ent 和生成物未变。已有规范足以解释本次修复，无需修改业务规范。

## 验证

| 检查 | 结果 | 证据 |
| --- | --- | --- |
| 五文件 `gofmt -l` | PASS，无输出 | 审阅时执行 |
| 两包 `go vet` | PASS，退出码 0 | 修改前后均执行 |
| `git diff --check` | PASS | 修改前后均执行 |
| data 定向／相邻集成 | 4 顶层 + 3 子项 PASS，0 FAIL，0 SKIP | 审阅读取 `/tmp/roncin-baseline-data-after.log`；审阅未修改 data 代码 |
| migration 定向／相邻集成 | 10 顶层 + 2 子项 PASS，0 FAIL，0 SKIP | 审阅读取 `/tmp/roncin-baseline-migration-after.log`，为审阅补强之前结果 |
| 补强后种子原测试真实 PostgreSQL | 1 顶层 PASS，0 FAIL，0 SKIP，2.20s | `/tmp/roncin-baseline-seed-review.log` |
| 两包完整真实 PostgreSQL | 父任务编排正在统一执行 | 本审阅不重复执行；需最终收取 Stage A 结果后才可判断 A4 |

补强后命令通过专用环境文件注入，未打印连接串。第一次仅加载 shell 变量导致 SKIP，随后以 `set -a` 导出变量重新真实执行；上述日志与结果均为后一次有效运行，首次 SKIP 不作为验收证据。

```bash
set -a
. /tmp/roncin_verify_45707338.env
set +a
go -C server test -v -count=1 -p 32 -timeout 20m ./internal/platform/migration/ \
  -run '^TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies$'
go -C server vet ./internal/data/ ./internal/platform/migration/
git diff --check
```

## 代码冻结指纹

以下是提交前文件 SHA256；父任务应补入最终提交 SHA，并核对完整集成对应这些断言。

| 文件 | SHA256 |
| --- | --- |
| `admin_role_anchor_integration_test.go` | `002f5c26defebaf6e7d20c28f7621092ff245e83fcca59eb256a315697d7da78` |
| `fee_catalog_integration_test.go` | `5e568237b2ecbba87ac5e8f2f58f51941541d016b953ffbb6032849183151c0a` |
| `integration_test_helper_test.go` | `d50a3649a287bf82e038590b513143dd1f0e88cfbcc6dd562fdde2a0772d516e` |
| `company_boundaries_integration_test.go` | `5196ac4545c50d245367122669bef8d7f104095eeb727d6987e87a7561d38767` |
| `fee_catalog_seed_integration_test.go` | `e13624d41075142bffbc6e590687d0e640a7a33551a390e388597389d95f5a3b` |

## 未修复问题与交接

无遗留代码审阅问题。尚需父任务收取最终完整两包统计、记录最终提交及验收环境清理结果；这属于在进行的验收步骤，本记录不提前宣告子任务完成。未提交 Git，按父任务约定由主会话提交。
