# 模式列迁移生成依据

退役前后均以 Ent Schema 生成的 `migrate.OrdersColumns` 为结构依据，比较列集合得到：

```text
removed: shipment_mode
added: （无）
```

服务端实现代理据此生成 `server/migrations/20261117100000_retire_order_shipment_mode.sql`，SQL 为严格 `ALTER TABLE "orders" DROP COLUMN "shipment_mode"`，不修改历史迁移、不删除订单或其他业务关系。

迁移测试 `TestRetireShipmentModeMigrationPreservesOrdersPostgres` 使用正式迁移前缀构造三类旧值（NULL、TRADITIONAL_FORWARDING、CROSS_BORDER），比较除目标列外订单以及MBL、航程、活动关联、HBL全行JSON快照，并检查重复执行只保留一条迁移记录。`TestRetireShipmentModeMigrationColdStartPostgres` 验证最新正式迁移完整链后列不存在，订单可正常创建更新。

本地专用隔离数据库实际执行：2 PASS、0 FAIL、0 SKIP。此前两轮失败均为新测试夹具不符合现有主单唯一确认／纯英数字约束，已修夹具后重新执行通过；没有放宽业务约束。
