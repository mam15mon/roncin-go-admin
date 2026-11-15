# 第二轮审计最终收尾（2026-11-15）

## 授权与范围

用户本轮要求所有进行中任务完成并合并，结束此前持续审计。只收敛现存发现、核对交付与必要相邻回归，不将任务无限扩展为新一轮系统审计；归档和最终集成门禁由主会话协调。本轮不读取或修改日常业务库，不删除业务数据。

## B1–B7 已交付核对

对下列提交逐一执行 `git merge-base --is-ancestor <commit> main`，均退出 0：

| 问题 | 修复提交 | 既有验收依据 |
| --- | --- | --- |
| B1 聚合结算规则权限、B2 默认抬头旧版本 | `dbc3ee8b` | 子任务 `research/fix-verification.md`：真实库及独立 review 通过 |
| B3 取消增提净额、B4 对冲未结候选 | `1a0f4a9d` | 同上，真实库定向和相邻事务回归通过 |
| B5 HAWB 同页保存基线、B6 实际箱并发 | `fee61336` | 同上，前端及两锁竞争顺序真实库通过 |
| B7 禁用默认预设请求 | `7e36d4e0` | `preset-default-fix.md`：非法组合无写入、合法对照真实库通过 |

这些是既有交付证据，本轮不冒称重跑了当时全部测试；主会话另执行最终全栈门禁。修复前 `findings.md`、动态复现报告与子任务交叉复核均按历史时点保留。

## 遗留预设 SQL 疑点闭环（B8 / P2）

此前 `preset-default-dynamic-confirmation.md` 已记录 MARKS 更新出现 content 重复赋值；本轮检查当前 `updateShippingPresetDetail` 和 Ent 生成代码确认缺陷仍在：更新器先 `ClearContent/Code/Remark` 再 `SetNillable`，非空输入使生成 SQL 同一列同时赋值和清空，PostgreSQL 拒绝整次更新。Party 的地址、联系人、电话、邮箱、税号及备注也使用同一错误模板，非空更新同样受影响。未据此推断发生过部分保存：事务会整体回滚。

局部修复：该函数复用包内现有 `setOptionalString`，每个可空字段按空/非空互斥 Clear/Set。保留原空字符串归一化、必填校验、事务和审计边界，不改契约、Schema、权限或生成物。数据库规范已有同类规则，本次补充单证预设回归入口。

实施后补 `TestPartnerShippingPresetOptionalUpdatePostgres`，覆盖 SHIPPER、MARKS、HS_CODE 的非空更新与清空。通过真实用例→仓储→正式 SQL 迁移 PostgreSQL，核对响应投影来自提交后重读；与既有 B7 默认项验证一起运行。测试不标记 `t.Parallel()`，不使用 Ent 自动建表。

## 本轮验证

- 专用 PostgreSQL 18 临时集群，仅 Unix socket；UTF8 一次性测试库，各用例使用正式迁移及随机隔离 Schema，未操作业务库。
- 定向命令：`go -C server test -p 32 ./internal/biz ./internal/data -run TestPartnerShippingPreset -count=1 -v`，真实 PASS，无 SKIP：biz 3 个顶层测试（含非法组合 4 个子例），data 2 个顶层测试（B7 的 5 个子例与 B8 的 3 个子例），data 3.194s。
- `go -C server vet ./internal/data ./internal/biz` 通过。
- `git diff --check` 通过；无前端修改，前端类型检查由主会话最终门禁覆盖。

## 结束判定

本期已确认清单收敛为 B1–B8；B1–B7 已在 main，B8 为本轮局部修复，验收通过并提交后统一归档。此前默认切换“锁循环”疑点已有反证排除，不再计缺陷；未证明的订单自动带入后果不计入结论。不保留虚构的待办，不以归档等同系统以后不会再有缺陷。

首次临时集群默认 SQL_ASCII 导致中文迁移执行失败，未进入产品用例；已创建独立 UTF8 测试库后完整重跑通过，不将首次失败记录为产品通过。测试结束确认隔离 Schema 为 0，临时集群已停止。
