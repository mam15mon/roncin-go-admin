# 空运出口危险品包装类别 PG 枚举字段

## Goal

`Order` 共享消息新增 `DangerousPackingGroup` 枚举（UNSPECIFIED/I/II/III），
空运危险品动效区加第四字段（仅空运渲染，海运零变化），proto 注释写明
IATA DGR 适用范围与 UNSPECIFIED 双语义；含 Ent 迁移、生成物、adapter/
模板/测试全链路。

## 背景与用户拍板

- 竞品核对结论：HGJ 的 `pg` 同样挂在共享订单模型上（空运响应实测，
  string 型、恒空、海/空模板与 UI 均未暴露）——我们的字段位置与竞品一致，
  类型用 enum 做对。
- 用户批准方案并带两处补强：
  1. proto 注释写明 PG 仅适用 IATA DGR 部分类别（Class 3/4/5.1/6.1/8 及
     部分 9），Class 1（爆炸品）/2（气体）/7（放射性）无包装类别；
     `UNSPECIFIED` 兼「未填 / 不适用」双语义——防止将来误加「勾危险品
     必填 PG」校验伤害 Class 2/7。
  2. 「不做 UN↔PG 联动」记钩子说明：DGR 中同一 UN 允许的 PG 集合不同，
     将来若做是查表校验而非推导，enum 模型不阻碍。
- 用户要求实现方式对齐海运已跑通的枚举链路（详见 design.md「海运范式」）。

## Requirements

1. 契约：`order.proto` 新增 `enum DangerousPackingGroup`（四值）+
   `Order` / `CreateOrderRequest` / `UpdateOrderRequest` 加 optional
   `dangerous_packing_group` 字段，注释含上述适用范围与双语义说明；
   随后 `make -C server api`、`pnpm run generate:web-client` 同组生成。
2. 服务端：严格照抄海运可选枚举链路（ShipmentType 模板）——biz 领域
   枚举类型、service FromAPI/ToAPI switch、create/update 输入、data 写入
   与投影；无额外业务校验（不强制必填）。
3. 数据库：Ent `field.Enum(...).Optional().Nillable()` + 正式迁移 SQL
   （orders 表 `character varying NULL`，与 shipment_type 列同形）。
4. 前端：`SeaDangerousGoodsFields` 加 `showPackingGroup?: boolean` prop，
   true 时动效区渲染第四字段（PG 三档下拉，仅勾选危险品时可编辑）；
   空运模板传 `showPackingGroup`，海运不传（渲染零变化）；air adapter
   序列化 `dangerousPackingGroup`（undefined 不携带）与详情回填。
5. 测试：service 转换、air adapter payload/回填、布局测试（空运卡内存在
   PG 字段、海运卡内不存在）、组件 prop 行为。

## 非目标（Out of Scope）

- UN↔PG 查表联动校验（钩子说明落在 proto 注释与本 PRD）。
- 闪点、海洋污染物（用户已拍板缓 / 不做）。
- 海运出口任何渲染与行为变化（仅允许共享组件加可选 prop 的向后兼容改动，
   海运不传 prop 时输出与现状逐像素一致）。
- 不为 PG 加权限码、审计事件或列表列。

## Acceptance Criteria

- [ ] proto / 生成物（pb + openapi + web client + enums.generated）在同组
      提交中更新，`dangerous_packing_group` 全链路贯通：新建填写 → 落库 →
      详情回显。
- [ ] 空运新建：勾选危险品展开后可见「包装类别」下拉（I/II/III + 可清空）；
      未勾选时随动效收起不可编辑；取消勾选随三元组一起清值。
- [ ] 空运 payload：选择后携带枚举值（整数枚举传输，与 shipment_type
      口径一致），未选/清空不携带；详情回显正确。
- [ ] 海运：不传 prop 时 DG 区块无 PG 字段，全部既有测试通过、渲染零变化。
- [ ] 迁移：`pnpm run migrate:dev` 对开发库成功应用；空库冷启动链完整
      （新库执行整链不报错）。
- [ ] 服务端 `go -C server test ./...`、`go -C server vet ./...` 通过；
      前端 tsc / Biome / 定向测试通过。
- [ ] proto 注释含 DGR 适用范围、UNSPECIFIED 双语义、UN↔PG 查表钩子三要素。
