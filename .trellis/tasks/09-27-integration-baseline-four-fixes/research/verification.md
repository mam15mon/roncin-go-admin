# 四项存量集成失败修复与真实验证

## 环境与代码绑定

- 日期：2026-09-27（UTC）。
- 基准提交：`9d40fbc24ba1776707e9b4cd893df85c6b7d1477`；本记录首先对应该提交之上的五个测试/辅助文件工作区改动，最终提交由父任务统一记录。
- Go：`go1.26.8 linux/amd64`。
- PostgreSQL：18.6，Ubuntu 本地真实实例；专用一次性数据库 `roncin_verify_45707338`，不是日常开发库。
- 连接串通过权限 `0600` 的临时环境文件注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`；记录中不保存连接串或凭据。
- 所有用例使用随机隔离 Schema，初始化只执行有序正式 SQL 迁移，不调用 Ent `Schema.Create`。定向结束查询三类测试 Schema 残留数为 **0**。

## 修复前真实复现

| 用例 | 实际失败签名 | 实证根因 |
| --- | --- | --- |
| `TestRoleWorkspaceAnchorBackfillPostgres` | 创建断链部门违反 `organizations_workspace_parent_check`，SQLSTATE 23514 | 最新 Schema 禁止部门缺失父节点，夹具在运行回填断言前失败 |
| `TestFeeCatalogCompanyTemplatesPostgres` | 公司目录 `Total=123`，旧断言期望 1 | 正式迁移已提供 122 个启用模板；测试新建模板后公司应复制 123 个，不能把公司目录当成只有测试自建模板 |
| `TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies` | `abnormal_case` 数量 18，期望 5 | 执行完整目录时，后续 `20260923130000_abnormal_case_seed.sql` 独立追加 13 个类型；被测费用目录种子的目标集合确实是 5 个 |
| `TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots` | `order_fees` 快照或其他字段被修改 | 字段级差异为 `status: "DRAFT" → 字段不存在`，`cancellation_reason / cancelled_at / cancelled_by: null → 字段不存在`；这是后续 `20260926100000` 的正式硬删除 Schema 契约，不能归因给公司边界迁移 |

复现日志：`/tmp/roncin-baseline-data-before.log`（两项目标均 FAIL）、`/tmp/roncin-baseline-migration-before.log`（两项目标均 FAIL）、`/tmp/roncin-baseline-boundary-diff.log`（字段级诊断）。全部目标实际运行，无 SKIP。

## 修改与断言依据

### 工位锚点

- 集成辅助函数支持从正式迁移前缀构造历史前态。回填测试只应用 `< 20260922100000` 的迁移，因此旧断链部门可以真实存在，回填拒绝断链、唯一冲突、清理后自检与幂等断言均可执行。
- 删除该历史夹具里无用途且在旧 Schema 中不合法的 `system` 组织创建；公司、部门与团队仍真实落库。
- 当前 Schema 的相邻 `TestAdminRoleAnchorPostgres` 增加断链部门 CHECK 拒绝断言，保留现行数据库约束保护。没有删除或放松任何生产约束。

### 公司费用目录

- 期望从真实启用模板集合推导，并逐模板核对费用副本的独立 ID、公司归属、代码、名称/别名、类别、币种、计费单位、异常引用、税率、启用与排序配置。
- 核对税务副本属于目标公司，名称、默认税率、简称与商品码等于模板文本。
- 列表必须完整返回该公司的副本集合；`FC_TEMPLATE` 关键字查询必须只返回测试自建模板的副本。
- 保留模板更新不影响既有公司副本、跨公司更新不可见、同名税项配置冲突整笔回滚等原有验证。

### 费用种子

- 首先只应用到 `20260923120000` 费用目录种子，断言 122 个模板与 5 个明确业务代码的异常目标集合。
- 准备两个公司，断言本地 OF 保留、另一公司完整复制、税务引用无跨公司混用、所有新增副本配置与模板文本一致。
- 接着应用完整迁移链，核对原 5 个目标代码保持、后续 13 个代码齐备、总量 18。
- 直接重放正式费用种子 SQL，再重复 `Apply`；对五张目录/税务/主数据表按 ID 汇总整行 JSON，除种子明确更新的 `updated_at` 外，业务值、ID 与 `created_at` 均保持。

### 公司边界迁移

- `boundaryOnlyDir` 只应用到 `20260922100000`，保留此前完整迁移历史，避免让后续 Schema 删除契约污染当前用例。
- 费用与补录申请分别排除唯一允许变化的 `fee_setting_id` 后比较整行 JSON，其余字段不变；失败信息给出排序后的字段级差异。
- 引用必须改到公司 A 的新费用副本且不同于共享源 ID，费用代码正确，税务引用同属于公司 A。
- 生产 SQL、Ent Schema、业务实现与生成物均未修改。

## 定向与相邻真实 PostgreSQL 结果

以下命令均已先注入专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`：

```bash
go -C server test -v -count=1 -p 32 -timeout 20m ./internal/data/ \
  -run '^(TestAdminRoleAnchorPostgres|TestRoleWorkspaceAnchorBackfillPostgres|TestFeeCatalogCompanyTemplatesPostgres|TestFeeCatalogReferenceValidationPostgres)$'
go -C server test -v -count=1 -p 32 -timeout 20m ./internal/platform/migration/ \
  -run '^(TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies|TestCompanyBoundaryMigration.*)$'
```

| 范围 | 顶层 PASS | 子项 PASS | FAIL | SKIP | 包耗时 | 日志 |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| data 定向及相邻 | 4 | 3 | 0 | 0 | 7.829s | `/tmp/roncin-baseline-data-after.log` |
| migration 定向及相邻 | 10 | 2 | 0 | 0 | 20.694s | `/tmp/roncin-baseline-migration-after.log` |

四项目标均实际 PASS。公司边界相邻覆盖策略复制/组织根保留、港口冲突原子拒绝、共享费用与税项复制、税务冲突拒绝、引用本地同码冲突拒绝、系统私有目录转换、模板代码冲突、系统私有经营引用拒绝。

## 完整两包与检查

- `git diff --check`：PASS。
- `go -C server vet ./internal/data/ ./internal/platform/migration/`：PASS，退出码 0，日志 `/tmp/roncin-baseline-vet.log` 无输出。
- 两包完整真实 PostgreSQL 集成由父任务的真实验收编排 Stage A 顺序执行，避免重复运行。需收取完整 JSON 日志及 PASS/FAIL/SKIP/panic 统计后再判断 A4；本子任务此刻不宣告完成。
- Trellis 检查代理审阅与最终提交 SHA 由父任务协调。
