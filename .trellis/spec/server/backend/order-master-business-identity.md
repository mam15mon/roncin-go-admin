# 海空主单进出口身份

## 1. 适用范围

海空主单的创建、候选、按 ID 关联、共享资料修改、恢复、单证版本和正式迁移适用。
主单自身保存不可变业务类型，成员归零或终止不改变身份。进口业务闭环由对应专项任务接通，
不能仅凭数据库接受 SI/AI 就开放创建入口。

## 2. 接口与数据库签名

- `SeaMasterBill.business_type`、`SeaMasterBillVersion.business_type`：必填 `SE | SI`。
- `AirMasterWaybill.business_type`：必填 `AE | AI`。三个字段均为 Ent `Immutable()`，无默认值。
- 海运唯一索引 `sea_master_bill_business_no`：
  `(organization_id, business_type, shipping_line_id, normalized_master_no)`。
- 空运唯一索引 `air_master_waybill_business_no`：
  `(organization_id, business_type, master_no)`；号码仍按航空既有规则规范化。
- 两主单额外索引 `(organization_id, business_type)`；替换旧的不含方向唯一索引。
- `MatchSeaMasterBillCandidateRequest.business_type`、
  `ListAirMasterWaybillsRequest.business_type` 必填。当前出口入口分别显式提交 SE、AE，缺失或
  不适用类型拒绝；HTTP/生成客户端使用 `businessType`。响应投影保留主单的真实业务类型。
- 正式迁移：`20261116120000_order_master_business_identity.sql`。

## 3. 契约

- 同公司、同船司、同号可分别登记海运进口和出口；空运同公司同号也按方向独立。
  同方向重复仍由唯一索引拒绝，业务不能自动拼单或改挂。
- 候选、直接 ID、原主单恢复及共享成员必须同时符合公司与业务类型；预览和最终写入均复验。
  不把前端传来的类型当作绕过实际 Order、主单身份校验的凭证。
- 共享写入沿用既有 Order 集合先行锁序、完整成员集合复验和预期版本；主单身份从其自身列读取，
  不从第一张票或当前活动成员猜测。海运新版本快照保存其主单的同一业务类型。
- 海运共享正文更新保留实际预锁的 Order ID 集合；取得 MBL 锁后按 ID 一次锁定完整活动 Link，
  身份和成员集合均须一致。新成员在两阶段之间加入时返回冲突，禁止拿到 MBL 锁后补锁未知订单，
  也不先锁目标 Link 再锁全部 Link。
- 海运拆票、改配在 Preview 和 Execute 各自验证原/目标已有 MBL 的完整活动成员身份。
  预览通过不能替代执行事务内复验；沿用原操作的订单锁范围，不为身份检查追加反向订单锁。
- 升级先检查全部历史关联，包括已结束关系：每张主单只能推导出一种合法业务类型；
  `Order.trade_direction` 在数据库使用小写 `export/import`，分别对应 SE/AE 与 SI/AI。
- 孤立且无可判定历史关系、未知类型、进出口混挂、类型与方向不符、跨公司关系，以及海运版本
  公司不符均显式停止迁移。禁止为无法判定的主单默认出口，禁止静默修正原有关系。
- 回填只新增身份，保留原号码、正文、版本、时间、成员和不可变历史。一个迁移文件的 SQL 与
  revision 同事务执行；失败不留下部分回填或列，重复执行不再改写数据。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 当前出口候选省略 `businessType` 或传进口类型 | 参数错误，不默认出口 |
| 同公司同号、不同方向 | 两张独立主单 |
| 同公司同方向重复号 | 唯一约束拒绝，失败事务回滚 |
| 按 ID 引用另一方向或公司 | 不可见或业务拒绝，不改挂 |
| 只有已结束的合法同方向关系 | 正确回填并保留历史 |
| 无关系、混向、未知类型或跨公司存量 | 迁移显式失败，无目标 revision |
| 身份为空或海空类型互换 | 数据库 NOT NULL / CHECK 拒绝 |

## 5. 正常、基准与错误场景

- 正常：同公司 SE 和 SI 的相同海运号独立存在，SE 候选只返回 SE。
- 基准：旧 AE 主单所有关系已结束，仍通过历史关系回填 AE，不改变关系状态。
- 错误：把所有既有主单直接写成 SE/AE，或仅检查 ACTIVE Link 后忽略历史混向。

## 6. 必需验证

- `TestOrderMasterDirectionMigrationPreservesHistoricalRowsPostgres`：正式 SQL 前态构造四类型、
  已结束关系和海运快照；比较新增列之外的全行内容，重复 Apply 后保持不变、revision 唯一。
- `TestOrderMasterDirectionMigrationRejectsUnclassifiableRowsPostgres`：海空各覆盖孤立、未知、
  混向、类型方向不符、跨公司订单与关联；断言异常、DDL 回滚及无 revision。
- `TestOrderMasterDirectionDatabaseConstraintsPostgres`：真实数据库验证方向/公司唯一性、
  NOT NULL、海空类型 CHECK 和无默认值。
- 候选和直接 ID 写入另有业务回归；现有 SE/AE 创建、共享修改、终止恢复、锁和版本继续通过。
  测试必须注入 `RONCIN_INTEGRATION_DATABASE_SOURCE` 并核对无 SKIP，不使用自动建表。
- `sea_master_content_membership_integration_test.go`：用实际 MBL 锁等待确定并发加票发生在
  正文请求预锁成员之后；断言结构冲突且正文、版本、审计均无写入。
- `sea_change_master_identity_integration_test.go`：两操作 × 原/目标 MBL 四个直接 Execute
  场景，已有成员类型错误时拒绝，且无新增订单、Link 或审计，原票版本与状态保留。

## 7. 错误与正确示例

```sql
-- 错误：未验证关联便把不明主单都默认归出口。
UPDATE air_master_waybills SET business_type = 'AE';

-- 正确：先拒绝未知、混向、公司错误和零关系；仅从全部已核实历史关系回填。
-- order.trade_direction 的持久化值为 export/import，而非 API 展示枚举名。
SELECT l.master_waybill_id, count(DISTINCT o.business_type)
FROM air_master_waybill_order_links l JOIN orders o ON o.id = l.order_id
GROUP BY l.master_waybill_id;
```
