# 方向身份组独立复核

复核日期：2026-11-16。范围为本子任务全部差异，父设计仍为统一合同；不宣称进口、航空直单、结构化运单、PDF 或完整 CSV 已交付。

## 已修复发现

1. `server/internal/data/sea_document.go` 的 `UpdateSeaMasterBillContent` 在预锁旧成员后取得 MBL 锁，只检查了新读成员身份，没有比较实际已锁 Order 集合。并发加票可能让正文保存覆盖未经过预锁门禁的成员。
   - `ensureSharedMBLNotLocked` 返回实际预锁成员 ID；取得 MBL 锁后按 ID 锁完整活动 Link，比较成员集合及原目标 Link。
   - 集合变化返回结构冲突，不在 MBL 锁后追加未知 Order 锁；去掉先单独锁目标 Link 再锁全体 Link 的顺序。
   - 新增 `sea_master_content_membership_integration_test.go`：另一事务持有 MBL 锁，数据库锁等待视图确认正文请求已进入等待后提交新增成员，再验证失败且正文、版本、审计均无写入。

2. `ExecuteReassignment` / `ExecuteSplit` 原本只检查源/目标 MBL 自身身份；完整成员身份检查主要在预览，直接执行不能依赖预览。
   - 两执行入口锁定各个已有 MBL 后复用完整成员身份校验，拒绝跨公司/跨方向异常成员；保持原本仅移动本票的 Order 锁范围。
   - `PreviewSplit` 也对候选完整关联复验身份，避免预览展示可执行而执行才拒绝。
   - 新增 `sea_change_master_identity_integration_test.go`，直接调用 repo 执行，覆盖拆票/改配 × 原/目标主单四种跨方向成员；验证拒绝且订单、关联、审计、原票版本与状态均无部分写入。

## 本组其他审阅结论

- Ent 源的三个身份字段均必填、Immutable、无默认；海空各自 CHECK、方向唯一索引、组织/方向查询索引与正式迁移及生成元数据一致。
- 迁移从全部实际历史关系判定，包括 ENDED；使用数据库真实的 `export/import`。未知类型、混向、孤立、类型方向不符、跨公司关系及海运版本公司错误均停止，不靠出口默认值或兼容分支通过。
- 主单候选/直接 ID /预览/恢复检查与目前仅开放 SE/AE 的业务范围一致，不提前开放 SI/AI。主单没有活动成员时仍保留自身身份。
- 共享成员查询不再用公司条件隐藏异常；海运公共校验和航空完整集合校验覆盖成员公司及业务类型。新增修改沿用既有事务封装，不新建逆序成员锁。
- 海运主单内容哈希含业务类型，两处不可变版本创建均保存业务类型。
- Proto→Go/OpenAPI→前端类型投影及五处现有候选调用均同步；业务类型缺失显式拒绝，不默认出口。前端只有出口范围调用，无新增进口入口。
- 已有测试/开发夹具显式设置类型，无新增默认值迁就夹具；没有手工修补生成物、无无关格式化或兼容抽象。

## 验证事实

复核代理实际执行：

- `pnpm run lint:proto`：PASS。
- `go -C server vet ./internal/biz ./internal/service ./internal/data ./cmd/sync-dev`：PASS；修复后重新 `go -C server vet ./internal/data`：PASS。
- `go -C server test ./internal/data -run '^TestMasterContentHashIncludesBusinessIdentity$'`：PASS。
- `TestSeaMasterContentRejectsConcurrentMemberPostgres`：真实 PostgreSQL PASS，2.36 秒，无 SKIP。
- `TestSeaChangesRejectInvalidMasterMembersPostgres`：真实 PostgreSQL四子场景 PASS，9.57 秒，无 SKIP。
- `git diff --check`：PASS。

主会话已执行并报告、此处未盲目重复：正式迁移三主测试（13 异常子场景与 2 约束子场景）PASS 无 SKIP；五前端测试文件 27 用例 PASS；修改文件 Biome、三组件 antd lint、前端 tsc PASS；biz/service/ent-migrate 完整纯包 PASS。详见 `direction-validation.md`。

实现代理反馈：Ent/API/web 重复生成均零差异 PASS。首次修复后大批 PostgreSQL 回归运行 144.383 秒，无 SKIP，多数通过，暴露下述两项已修问题；不能把该轮整体标记 PASS。修复后的 AirExportShared、MapEntConstraint、OrderCreateTransaction、SeaMasterBillBatchRules、SeaOrderSplitAndReassignment 五主项补跑全部 PASS，16.608 秒，0 FAIL、0 SKIP。实际命令和选择范围见 `direction-core-validation.md`。

## 未修改事项与边界

- 主会话只读预检本地开发库发现 1 张孤立海运主单及其 1 条历史版本，且没有实际 Link、HBL、订单审核或其他订单引用。严格迁移会按合同阻止该库升级；保留数据，不猜类型、不放宽规则、不清库。其归属需在后续整体交付时由用户确定；隔离 Schema 验证不受影响。
- 当前本组没有其余已确认且未修复的产品代码问题。父任务的进口及航空后续能力不在此组验收范围；整体验收门禁由父任务执行。


## 最终增量静态复核

仅复核上次报告后的两类产品增量，没有重复运行已通过检查：

- `air_export.go / lockAirMasterMembers`：先以主单 ID、当前公司和 AE 做无行锁定位，不可见目标返回原有 NotFound；随后仍按 Order 集合→MAWB→完整 Link 加锁，锁内再次以公司/类型查询主单并比较完整成员集合。新增前置读取没有引入逆序行锁，且没有将完整成员查询重新限缩为公司过滤。同公司主单的异常跨公司成员仍返回冲突；跨公司目标自身保持 404 语义。静态复核通过，无新增确定问题。
- 海运唯一约束映射：`order_write_sea_master_bill.go` 新建及修改两处、`sea_order_change_shared.go` 新建一处，均精确使用正式迁移、Ent StorageKey 和生成元数据的 `sea_master_bill_business_no`。保持原来的领域错误选择，修正数据库实际 ConstraintName 的匹配，不扩大匹配范围，也没有增加旧索引回退。`ent_constraint_integration_test.go` 同步断言新名称并保留旧逻辑名称不匹配断言。静态复核通过，无新增确定问题。
