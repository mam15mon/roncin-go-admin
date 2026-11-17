# 代码核查与删除边界

日期：2026-11-17。工作树初始干净，分支 main。核查为定向源码检索和读调用链，未运行产品、未写数据库、未执行实现。之前 HGJ 调研仅用于解释两种业务形态，本次删除判断以 Roncin 代码和用户业务范围为准。

## 模式字段真实链路

| 层 | 定位与行为 |
| --- | --- |
| Proto | `server/api/order/v1/order.proto:169` 定义 ShipmentMode；Order:313、CreateOrderRequest:651、UpdateOrderRequest:720 分别使用字段号13、11、12 |
| 领域 | `server/internal/biz/order_types.go:188` 定义两值；319落入Order；`order_usecase.go:476`验证枚举，619加入创建幂等意图哈希 |
| 传输 | `server/internal/service/order_enum.go:324`双向转换，`order_write.go:194`及312解析写入；`order_convert.go:23`输出 |
| 数据 | `server/internal/data/ent/schema/order.go:71`持久列；`order_write.go:75`创建；`order_convert.go:108`及249读写；`sea_order_change_split_execute.go:1250`拆票复制 |
| 前端 | `web/src/pages/orders/common.ts:101`选项，`templates/components/sea/SeaBasicInfoSection.tsx:670`及`components/air/AirCommercialFields.tsx:34`控件；海空form-adapter及form-values保存、回显和默认值 |
| 无效策略项 | `web/src/pages/orders/sea-order-policy.ts:31`返回modeLabel、emphasizeCrossBorderCompliance；检索只有本文件及测试消费。showConsolidationReference同样无产品调用，真实菜单另有判断。showContainerPlan/showRevenueTon仍有真实调用，须保留 |

搜索 ShipmentMode/shipmentMode/shipment_mode 后，未发现独立跨境小单实体、合并出运写用例或按模式调度的运输闭环；结论限定已检索的服务端、前端、脚本与规范目录，不把名称相近的普通共享主单业务当成该能力。

## 自拼汇总并非独立小单

- `web/src/pages/orders/list.tsx:28`接入ConsolidationDrawer；公共模板`OrderListTemplate.tsx:641`仅对海运LCL提供菜单，types.ts:250暴露回调。
- `server/api/order/v1/order.proto:60`的ListOrderConsolidations提供 `/api/v1/orders/{id}/consolidations`；427、436定义成员及汇总；OrderCargoMeasurement仅供该输出使用。
- `server/internal/data/order_query.go:430`实际查询SeaMasterBillOrderLink ACTIVE，再以MBL聚合同公司ACTIVE的LCL普通订单，读取SeaHouseBill及货物。这不是另一个Consolidation实体；Schema目录没有独立自拼表。
- `server/internal/data/order_lifecycle_gate_integration_test.go:420`已有汇总退关过滤测试。退役专用查询后可删除其专用断言，但要核对同文件及相邻测试仍覆盖共享主单关联、退关成员和HBL状态，不重写既有核心能力。
- 汇总RPC只复用订单读取权限，`server/internal/server/access_rules_gen.go:274`为Proto派生项；不需要为退役删除普通订单权限Manifest。

## 名称、测试与生成物

- `web/config/routes.ts:268`、`web/src/pages/workbench/WorkbenchSideCards.tsx:272`和layout相关测试有“集运费用明细”。拟统一为“费用明细”，路由不变。
- `scripts/acceptance-se-order.mjs:262`后的共享MBL场景消费汇总API；应调整为现存主单/订单契约验证，不能保留已删除接口调用。
- 其他验收脚本：acceptance-hgj-cross-module、acceptance-finance-payable、acceptance-finance-bill-batch、acceptance-finance-foreign-currency；E2E：sea-operation-stability.fixture、sea-import、finance-bill-batch；均有模式字段。
- API枚举兼容测试、订单选项/适配器/标签/航空布局测试、OrderListTemplate测试、data SQL驱动列投影测试也需同步。
- `server/migrations/20260820000000_baseline.sql`和`20260826110000_order_form_policy_fields.sql`包含历史模式列与旧值修正；保留历史链，在末尾新增正式删列迁移。
- 现行规范：air-order-commercial-fields.md、component-guidelines.md的自拼汇总要求、order-lock-and-document-version.md的汇总规则、order-personnel-editor-retirement.md的相邻菜单说明；domain/error-catalog.md由脚本生成，错误删除后再生成。

## 保留清单

ShipmentType（FCL/LCL/BREAK_BULK）、ContainerOwnership、SeaMasterBill/TransportExecution/HouseBill、AirMasterWaybill/HouseWaybill、活动成员关系、共享箱/货物分配、拆票改配、订单生命周期、并发版本、普通费用账单及金额重量规则均为真实现有能力。删除名字含Consolidation的专用汇总投影，不删除这些底层实体。

`ShippingLinesPanel`及其测试中的“中远海运集运”是企业名称；能力导航中的“页面自拼后端主机地址”是请求构造禁令，均不是退役对象。历史任务、迁移和审计资料保留原事实，不要求词语全仓零命中。
