# 当前模型审计

## 结论

现有模型具备共享 MBL、不可变提单版本、拆票、整体改配和放货引用等基础能力，但核心聚合仍按
“一张 MBL 固定一个航次、一张订单可有多张 HBL、箱货分配只发生在单订单内”设计。目标业务则是
“HOUSE 一票一张当前 HBL、MBL 与实际航次独立、少量物理箱跨订单共享”，因此不能只隐藏前端
按钮，必须同步调整数据库关系、命令契约、版本快照、历史事件和前端交互。

## 已核对的代码事实

- `server/internal/data/ent/schema/sea_master_bill.go:21` 把
  `transport_execution_id` 直接放在 MBL 上，导致一张 MBL 只能读取一个航次。
- `server/internal/data/ent/schema/sea_master_bill_order_link.go:25` 保存
  `UNDETERMINED | DIRECT | HOUSE` 三态，且 Link 当前没有实际运输执行外键。
- `server/internal/data/ent/schema/sea_house_bill.go:20` 的 `order_id` 只有普通组合索引，当前允许
  同一订单存在多张有效 HBL。
- `server/internal/data/ent/schema/sea_cargo_allocation.go:21-25` 同时绑定单个 Order、HBL 与
  `OrderContainer`；`server/internal/data/ent/schema/order_container.go:21` 又强制物理箱属于单个
  Order，因此无法表达一只真实箱跨多张订单/HBL。
- `server/internal/data/ent/schema/order.go:21` 已有客户业务号，`:82` 只有订舱备注，没有独立
  Booking No. 字段。
- `server/api/order/v1/sea_document.proto:79-116` 同时暴露 Switch、回到未确定状态和任意添加 HBL
  的命令；`:149-155` 暴露三态单证结构。
- `server/api/order/v1/order.proto:158-164` 的复合单号筛选只支持订单号、主单号和加拼主单号，
  尚不能按客户业务号或 Booking No. 作为明确类型查询。
- `SeaMasterBillVersion` 当前保存单个 `transport_execution_id` 与航次快照；MBL/执行解耦后，
  历史锁定需要独立的运输执行版本，不能继续把某一票航次伪装成共享 MBL 版本内容。
- `SeaHouseBillVersion` 和 `SeaHouseBill` 仍包含 `SWITCH/REPLACED` 语义；
  `SeaHouseBillSwitchEvent` 是专用持久化实体，必须连同生成代码、API、Biz/Data 和前端入口删除。
- 当前 `SeaOrderReassignmentEvent` 已分别保存前后 MBL 与 TransportExecution ID，可以作为
  “同船公司只换执行”和“换船公司同时换 MBL/执行”两条路径的改造基础。
- `OrderLockRecord` 当前只引用 MBL 及 MBL 版本；解耦后必须额外固定订单当时的
  TransportExecution 版本，否则锁定历史无法重现真实航次。
- 订单列表当前已有真实活动 MBL 关系查询和“加拼主单”抽屉，可收敛为同批订单视图，不需要
  新建 Booking 聚合或继续维护号码派生的第二套共享实体。

## 数据与迁移前提

- 本次规划前已对本地开发库核对：SE Order、MBL、Link、HBL、SeaCargoAllocation、MBL/HBL
  Version、Split/Reassignment/Switch 事件均为 0 行。
- 因此目标迁移采用“相关表任一非空则在 DDL 前失败”的明确策略，不设计自动折叠多 HBL、
  猜测 HOUSE/DIRECT、拆解旧分配或把旧 MBL 航次隐式搬到 Link。
- 历史迁移文件保持不可变；新增一条增量迁移完成新表、字段、约束和旧 Switch/Allocation
  结构删除。

## 工作区边界

- `package.json`、`server/internal/data/industry_reference_sync.go`、
  `server/cmd/sync-unlocode/main.go`、`server/cmd/sync-shipping-lines/` 与 `data/` 存在用户改动或
  未跟踪内容，不属于本任务；实施和提交必须持续排除，不能格式化、覆盖或顺带提交。
