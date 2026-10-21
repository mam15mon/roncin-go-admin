# 订单共享枚举字段配方

> 适用：在共享 `Order` 契约上新增可选枚举字段（proto enum）的跨层变更。
> 范本链：`shipment_type`（首个可空订单枚举）与 `dangerous_packing_group`
> （20261021120000 迁移，空运危险品包装类别）。

## 五层链路（每层照抄范本，不发明新模式）

| 层 | 做法 | 范本位置 |
| --- | --- | --- |
| proto | `enum X { X_UNSPECIFIED = 0; … }` + 三个消息（`Order` / `CreateOrderRequest` / `UpdateOrderRequest`）各加 `optional X x = <下一可用号>` | `order.proto` TradeTerm / DangerousPackingGroup |
| biz | `type OrderX string` + 值常量（`UNSPECIFIED` 不设常量、不落库）；订单结构用指针 `*OrderX`；禁止 import proto 包 | `internal/biz/order_types.go` OrderShipmentType |
| service | `order_enum.go` 手写 `xFromAPI/xToAPI` switch（nil→nil，UNSPECIFIED→nil）；create 直接映射，update `if request.X != nil` 分支，`mergeOrderUpdateRequest` 以 `output := *existing` 起步使省略字段保留旧值 | `order_enum.go:163`、`order_write.go:301` |
| data | ent `field.Enum(...).Optional().Nillable()`；写入走 `SetNillableX` / `setOrderOptionalReferences` 的 nil→Clear、非 nil→Set；读侧 `orderToBiz` 投影 | `ent/schema/order.go`、`data/order_convert.go:249` |
| web | `pnpm run generate:web-client` 生成 TS 枚举；adapter 传值 `values.x !== undefined && values.x !== null ? … : undefined`；整数枚举传输（openapi `type: integer`） | sea form-adapter.ts:210 |

## 迁移

- 列形固定 `character varying NULL`（与 baseline `shipment_type` 一致），
  不设默认值、不加 CHECK（枚举合法性由 Ent 值集在应用层保证）。
- 文件名 `YYYYMMDDHHMMSS_order_<field>.sql`，内容单条
  `ALTER TABLE "orders" ADD COLUMN "<field>" character varying NULL;`，
  禁止 `IF EXISTS`。

## 更新语义（易错点）

- **payload 省略字段 = 保留旧值**（不误清）；
- **显式传 0（UNSPECIFIED）= FromAPI 归 nil → Clear 清空列**。
- 推论：下拉 UI 清空（值变 undefined → 省略）不会清掉已落库的值。若业务
  需要「UI 可清空已保存值」，必须让 adapter 在「原值存在且现值为空」时
  显式发 0，这是一次独立的契约行为决策，不要顺手实现。
- 共享组件若同时服务海运/空运（如 `SeaDangerousGoodsFields` 的
  `showPackingGroup`），海运不传 prop 时渲染必须与变更前逐像素一致，
  由组件测试锁定。

## 测试

- service：FromAPI/ToAPI 覆盖 nil / UNSPECIFIED / 每个值 / 未知值。
- adapter：create 携带、未选省略、UNSPECIFIED 省略、detail 回填。
- 组件：传 prop 渲染新字段、不传无、取消勾选清值列表含新字段。
