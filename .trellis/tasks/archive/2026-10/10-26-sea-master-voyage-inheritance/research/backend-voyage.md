# Research: 共享主单关联与航程统一更新

- Query: 已有主单候选关联的继承/冲突规则；共享航程更新的成员范围、权限与门禁；本期最薄页面入口与验证边界。
- Scope: internal
- Date: 2026-10-26

## Findings

### 结论

本期无需变更服务端契约、权限、Schema 或迁移。候选匹配、关联确认、统一修改共享航程均已有正式接口。前端应显式带入选定航次的共享字段，并保留服务端事务校验；统一修改复用已有弹窗和顶部菜单入口。

### 文件定位

| 文件 | 职责 |
| --- | --- |
| `server/internal/data/order_write_sea_master_bill.go` | 建单关联候选、共享主单身份与共享航程冲突复核 |
| `server/internal/biz/sea_master_bill.go` | 共享航程字段和冲突口径 |
| `server/internal/data/sea_master_bill.go` | 查询主单活动成员、去重实际航次候选、成员数与分单号 |
| `server/internal/data/sea_order_change_transport_update.go` | 共享航程修改预览与统一执行 |
| `server/internal/biz/sea_order_change.go` | 更新命令归一化、原因/确认/版本/幂等校验 |
| `server/internal/data/order_convert.go` | 用共享实际航程覆盖 SE 订单读取投影 |
| `server/api/order/v1/sea_order_change.proto` | 共享航程预览/执行正式契约 |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx` | 已有预览、外部确认、统一修改弹窗 |
| `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx` | 顶部共享航次菜单、弹窗状态、成功刷新 |
| `web/src/pages/orders/components/detail/OrderDetailHeader.tsx` | 顶部「更多操作」菜单汇总 |

### 1. 候选关联保存时的真实行为

- 主单身份是组织 + 船公司 + 规范化主单号。建单关联已有候选必须同时传入 MBL ID/version、TE ID/version；TE 必须属于目标主单的活动成员链路，且 MBL/TE/Order 的船公司一致。见 `server/internal/data/order_write_sea_master_bill.go:69-128`。
- 任何活动成员为 DIRECT 则阻止新增拼单；加入已有批次的新票必须为 HOUSE 并有唯一 HBL。没有隐式 DIRECT→HOUSE 转换。见 `server/internal/data/order_write_sea_master_bill.go:103-118`。
- 后端在事务内比较已选 TE 与本票航程，冲突则整个建单回滚，然后才创建活动 Link。见 `server/internal/data/order_write_sea_master_bill.go:131-165`。
- 这个关联分支只创建 Link，不会用本票输入反向填充已有 TE，也不会把候选内容主动写回前端表单。因此“空白自动带入”和“冲突时点击采用”需要前端实现；不能将“请求已带 candidateId”当成表单已继承航程。
- 新订单自身的原始航程列按提交输入写入，之后正式读取会以共享 TE 投影覆盖。见 `server/internal/data/order_write.go:60-102`、`server/internal/data/order_convert.go:157-168`。
- 共享字段：船公司、起运港、卸货港、中转港、船名、航次、ETD、ETA。最终目的地不属于共享航程；客户、HBL、收发货人、货物、件重尺、费用均不得随航程带入而改写。契约见 `.trellis/spec/server/backend/sea-export-document-contract.md:125-126`；目的地独立行为有 `server/internal/data/order_transaction_integration_test.go:711-724` 覆盖。
- `CheckSeaVoyageConflicts` 只比较双方都有值的字段；船名/航次忽略大小写、首尾空白，日期按 UTC 自然日比较。任一方未填不算冲突。见 `server/internal/biz/sea_master_bill.go:217-330`。前端可用生成候选响应解释差异，最终仍由服务端判定。
- 候选接口在恰好一个 TE 时计算冲突；多 TE 需要选择之后在前端明确处理选定航次的差异，最终保存仍有后端复验。见 `server/internal/data/sea_master_bill.go:132-135`。

### 2. 已有统一修改是否足够

足够。现成接口路径：

- `POST /api/v1/orders/{order_id}/sea-order-change/transport-execution/preview`
- `POST /api/v1/orders/{order_id}/sea-order-change/transport-execution`

两者都按订单 `reassign` 权限和公司范围鉴权，见 `server/api/order/v1/sea_order_change.proto:63-80`；不能因为显示区域是只读就开放给仅有 read 权限的人。

现有能力与约束：

1. 更新目标为当前订单活动 Link 对应的 **TE**，影响集合是共享这个 TE 的全部活动 Link，可能包含多个主单的订单，不能用“同一 MBL 成员”代替。预览成员按订单排序，返回 `memberOrderIds`。见 `server/internal/data/sea_order_change_transport_update.go:47-76`。
2. 预览收集全部成员的下游事实和业务可编辑性，任意阻断事实会令 `executable=false`；有费用建账、相关账单/发票/共享箱分配等亦可能阻断。见 `server/internal/data/sea_order_change_transport_update.go:77-104`、`server/internal/data/sea_document_change_shared.go:115-198`。
3. 执行采用已有 WithTx，按固定顺序锁订单、MBL、Link、TE，逐票验证业务锁、终止/结案门禁，复核 TE 版本和成员集合。见 `server/internal/data/sea_order_change_transport_update.go:110-183`。本期不要改变这些事务或锁规则。
4. 修改一个共享 TE 并生成不可变版本/审计后，所有成员在后续读取时看到同样航程；不是逐票手工写一遍。见 `server/internal/data/sea_order_change_transport_update.go:193-234`、`server/internal/data/order_convert.go:157-168`。
5. 输入支持补齐船名、航次、港口、ETD/ETA，也支持显式清空可选字段；原因必填，执行还要有效外部确认、幂等键。见 `server/internal/biz/sea_order_change.go:508-540`。
6. **船公司不能经共享航程调整弹窗修改**；它属于主单身份。统一航程更新输入只有七项航程字段，改船公司仍走已有更正/改配约束。见 `server/api/order/v1/sea_order_change.proto:526-534`。

当前弹窗已实现：预览前后差异、受影响数量、阻断标签、外部确认、统一调整和刷新。见 `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx:126-198`、`:304-332`。缺口是未展示具体受影响订单号。

### 3. 最薄页面入口集成

- `SeaExportDetailFeatures.tsx:170-180` 已有 `shared-voyage-update` 菜单，出现条件为 `!businessWritesDisabled && canOrder('reassign')`；详情头在 `OrderDetailHeader.tsx:161-165` 渲染为「更多操作」。把菜单标签改为「修改共享航程」，共享航程区域说明“如需修改，请使用顶部 更多操作 → 修改共享航程”即可。
- 现有弹窗状态由该详情扩展持有，成功后调用 `refreshOrderAndLock` 和 `loadChangeActions`，见 `SeaExportDetailFeatures.tsx:313-322`。无需给 `TemplateProps`、`OrderDetailFeaturesContext` 或公共模板增加回调，无需全局事件。
- 新建页匹配到旧主单、发现资料不全时，可展示候选成员的 `orderNo` + 订单详情链接，`target="_blank"` 让用户去已保存订单补全；不要从尚未保存的新票直接执行对其他订单的共享航程更新。
- 候选查询和 GetOrder 都需要订单 read 权限，见 `server/api/order/v1/order.proto:17-24`。进入旧订单后，能否修改由原 reassign 权限及业务门禁决定，不能用候选响应推断写权限。
- 受影响订单明细使用 React Query `useQueries`，仅按 `preview.memberOrderIds` 请求现有 GetOrder 生成客户端；保留精确预览数量，并逐 ID 展示加载、读取成功的订单号或失败状态及 ID。失败不得吞掉、过滤掉、显示成少量订单。无需为本期扩展 Proto。
- 不能用 SameBatchOrdersSection 的数据拼影响范围：它含同主单/同参考号等业务关系，且排除当前票，既不等于 TE 成员，也可能漏掉跨 MBL 的 TE 成员。见 `server/internal/data/order_query.go:541-566`。

### 4. 建议文件所有权

主单候选/字段继承实施者：

- `web/src/pages/orders/templates/components/sea/SeaTransportSection.tsx` 及拆出的本目录候选 Hook/展示组件（必要时）。
- 共享字段的 disabled/readonly 连接处、已有模板测试与新增专门的继承/冲突回归测试。

统一修改弹窗/详情入口实施者：

- `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx`
- `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.test.tsx`，必要时分拆成员明细失败/规模用例。
- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx`
- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx`

两组不用共同修改通用 `detail.tsx`、`TemplateProps` 或注册表类型；最终规范由主会话更新。

### 5. 验证建议

- 候选：新主单自由填写；已有单航次自动带入共享字段；有冲突明确采用后才覆盖；两个及以上航次无默认第一个；缺船公司提示待核对；DIRECT 阻断；HBL 即时排重；修改主单号/船公司后旧关联失效。
- 独立资料：目的地、货物/件重尺、HBL、客户不被航程带入覆盖。
- 统一修改：菜单依照 reassign/业务锁出现；预览具体订单号与精确数量；单个成员读取失败仍保留 ID、错误及总数；修改输入后旧预览及成员数据失效；执行使用预览输入+正确 TE 版本，失败不关闭弹窗；成功刷新。
- 日期：按现有 formatDate 和本地自然日序列化，不把 UTC 原串直接展示；真实库读回目前已保留 RFC3339 时刻，见 `server/internal/data/order_convert_test.go:110-185`。
- 可复用后端定向校验：`go -C server test ./internal/data -run 'TestOrderToBiz|TestSeaOrderChangeBusinessLockGate_Postgres'`。Postgres 集成部分必须有显式 `RONCIN_INTEGRATION_DATABASE_SOURCE`，无环境不得宣称已运行真实库；本期不改后端时不要求扩大到全量 Go 验证。
- 现有业务锁集成正/反例见 `server/internal/data/sea_order_change_integration_test.go:1467-1564`；不能以本次前端优化放宽已有门禁。

## External References

- 本次为仓库内部契约研究，未新增网络依赖或第三方库；前端沿用仓库 React Query v5、Ant Design 6 和生成客户端。
- 上下文中已核实的货代业务背景可参考 [马士基 HBL 条款](https://terms.maersk.com/HBL)（多航段及中转）与 [Lloyd’s 货损与追偿教材](https://assets.lloyds.com/assets/pdf-how-we-train-and-examine-lloyds-agents-2019-sept-cargo-claims-and-recoveries-manual/1/pdf-how-we-train-and-examine-lloyds-agents-2019-Sept-Cargo-Claims-and-Recoveries-Manual.pdf)（拼箱主分单关系）；本期系统具体门禁以已批准需求和仓库契约为准。

## Related Specs

- `.trellis/spec/server/backend/sea-export-document-contract.md`
- `.trellis/spec/server/backend/order-lock-and-document-version.md`
- `.trellis/spec/web/frontend/state-management.md`
- `.trellis/spec/web/frontend/component-guidelines.md`
- `.trellis/spec/web/frontend/quality-guidelines.md`

## Caveats / Not Found

- 当前代码尚未提供专门的“查询 TE 所有成员订单号”轻量接口；只有预览的 memberOrderIds 可作为精确范围。本期按 ID 补读即可。
- 当前共享航程修改不会主动同步原始 Order 航程列，读取靠 `orderToBiz` 投影。因此订单关键字/日期过滤若直接查询 Order 原始列，可能存在修改后搜索口径与展示不一致；这是现存相邻风险，本期不应为此扩大契约或数据库设计，可在后续任务记录。
- TE 未填日期时，`orderToBiz` 不覆盖原始订单已有日期（见 `order_convert.go:163-168`），不是“主单空日期就必然读取为空”。资料不全的候选应明确提示补齐共享航程，不能假定用户在新票填写日期会修改已有 TE。
- 现有航程弹窗差异表直接展示 `currentValue/targetValue`，日期可能为 RFC3339、港口可能为 UUID；本期触碰该界面时应按类型转换为人可读文本，保持现有格式化规范。
- 本研究未执行写数据库、接口变更、测试或 Git 操作。
