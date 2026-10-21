# 技术设计：空运出口危险品包装类别 PG 枚举字段

## 0. 海运范式（已验证的照抄模板）

用户要求对齐海运已跑通链路。海运可选枚举（托运类型 ShipmentType）五层实证：

| 层 | 海运做法 | 文件位置 |
| --- | --- | --- |
| proto | `enum TradeTerm/ShipmentType { …_UNSPECIFIED = 0; … }` + `optional` 字段 | `server/api/order/v1/order.proto:112,136` |
| service | 手写 switch 转换 `orderShipmentTypeFromAPI(v1.X) biz.X` / `ToAPI`；update 走 `if request.ShipmentType != nil` | `internal/service/order_enum.go`、`order_write.go:194,303` |
| biz | `type OrderShipmentType string` + 常量；可选枚举为指针 `*OrderShipmentType`，proto 类型不进领域层 | `internal/biz/order_types.go:156` |
| ent | `field.Enum("shipment_type").Values("FCL","LCL").Optional().Nillable()`；orders 表列 `character varying NULL`（baseline 实证） | `internal/data/ent/schema/order.go`、`migrations/20260820000000_baseline.sql` |
| web | `enums.generated` TS 枚举自动生成；sea adapter `values.shipmentType !== undefined && values.shipmentType !== null ? …` 传值 | `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts:210` |

PG 全链路照此实现，不引入任何新模式。

## 1. 契约（order.proto）

```proto
// DangerousPackingGroup 危险品包装类别（IATA DGR）。
// 仅适用于部分危险类别：Class 3/4/5.1/6.1/8 及部分 Class 9；
// Class 1（爆炸品）、2（气体）、7（放射性）没有包装类别。
// UNSPECIFIED 兼「未填」与「不适用」双语义，禁止据此实现
// “勾选危险品必填包装类别”类校验（会误伤 Class 2/7）。
// UN 号与允许的 PG 集合的联动是 DGR 查表关系而非推导关系，
// 未来如做校验须以查表实现；本枚举模型不阻碍该扩展。
enum DangerousPackingGroup {
  DANGEROUS_PACKING_GROUP_UNSPECIFIED = 0;
  DANGEROUS_PACKING_GROUP_I = 1;   // Ⅰ类包装（危险性最大）
  DANGEROUS_PACKING_GROUP_II = 2;  // Ⅱ类包装
  DANGEROUS_PACKING_GROUP_III = 3; // Ⅲ类包装（危险性最小）
}
```

- `Order`、`CreateOrderRequest`、`UpdateOrderRequest` 各加
  `optional DangerousPackingGroup dangerous_packing_group = <下一可用号>`，
  紧邻 `hazard_class` / `declaration_cutoff_at` 语义区。
- 生成：`make -C server api` → `pnpm run generate:web-client`
  （源与生成物同组提交，AGENTS.md 契约变更顺序）。

## 2. 服务端（ShipmentType 同款）

- biz：`type OrderDangerousPackingGroup string` + 四常量
  （`""`/`PG_I`/`PG_II`/`PG_III` 命名对齐既有常量风格，实现时以
  order_types.go 现有命名为准）；`OrderInput`/更新输出结构加
  `*OrderDangerousPackingGroup`（跟随 `*OrderShipmentType` 指针形态）。
- service：`order_enum.go` 加 `orderDangerousPackingGroupFromAPI/ToAPI`
  switch；create 直接映射、update `if request.DangerousPackingGroup != nil`；
  `order_convert.go` 投影（`enumPtrIfSet` 类辅助如无现成则按
  shipment_type 投影写法照抄）。
- data：`order_write.go`/`order_write_draft.go` `SetDangerousPackingGroup`
  （nil 走 `ClearDangerousPackingGroup`，对齐既有 Nillable 枚举写法）；
  `order_convert.go` 读侧投影。
- 无业务校验、无版本字段变化、无权限码。

## 3. 数据库

- ent schema（order.go，un_number:51 附近）：
  `field.Enum("dangerous_packing_group").Values("PG_I","PG_II","PG_III").Optional().Nillable()`
  ——注意 Ent Enum 值即落库字符串；`UNSPECIFIED` 不落库（nil 表示），
  与 shipment_type 的 NULL 语义一致。
- 正式迁移 `server/migrations/2026102X…_order_dangerous_packing_group.sql`：
  `ALTER TABLE "orders" ADD COLUMN "dangerous_packing_group" character varying NULL;`
  （禁止 IF EXISTS；与 baseline 中 shipment_type 列同形）。

## 4. 前端

- `SeaDangerousGoodsFields` 加 `showPackingGroup?: boolean`：true 时动效区
  FormRow 由 cols=3 变 cols=4 并追加 PG 字段（ProFormSearchableSelect /
  Select，三档选项 I/II/III，allowClear，`disabled` 链与三元组一致，值域用
  `enums.generated` 的 `DangerousPackingGroup`）；取消勾选清值 effect 把
  `dangerousPackingGroup` 加入清空列表。不传 prop 时输出与现状一致（海运
  零变化）。
- air-template：`<SeaDangerousGoodsFields … showPackingGroup />`。
- air adapter：`CommonValues`/`commonFields` 加 `dangerousPackingGroup`
  （`values.dangerousPackingGroup || undefined`，UNSPECIFIED 视同未选）；
  detail 回填 `order.dangerousPackingGroup`。
- 海运 adapter/模板不动。

## 5. 风险与控制

- 契约顺序错误会浪费生成时间：严格 proto → server gen → web client gen。
- Nillable 枚举 nil 写入必须走 Clear（否则部分更新把已有值冲掉）——
  update 路径用 shipment_type 的既有分支样式逐行对照。
- 迁移禁止 IF EXISTS，冷启动链必须完整（验收含空库链检查的实现方式：
  以迁移文件 SQL 与 baseline 列形一致性审阅代替真实空库执行，真实冷启动
  验证由 check 子代理跑 `pnpm run migrate:dev` 开发库应用 + 审阅 DDL）。
- 回滚：单任务 revert；列遗留无副作用（nullable）。
