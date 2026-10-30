# 研究：海运共同运输段与每票独立目的地的后端边界

- 查询：现有 SE 能否承载共享 TE 段与每票独立最终目的港；现有多 TE 是否等于有序多段路线；创建、更新、查询、拆票改配和单证快照的实际边界。
- 范围：仓库内部，只读代码与规范；仅本文件写入研究结果，未访问数据库、未修改产品代码或任务状态。
- 日期：2026-10-30。

## 结论

现有后端已经具备「多个操作票共享同一运输执行 TE，各票独立保存目的港」的核心模型与写入行为。本期用户已选择只完成共同段与每票目的地，不记录后续航次，因此应复用当前 `Order.destination_location_id`，优先修正页面语义、默认值与跨入口一致性，不需要另建后续 TE、通用路线引擎或数据库模型。

共同段的起运港、卸货港、中转港、船名航次、ETD/ETA 从当前活动 Link 对应的 TE 读取；`destination_location_id` 留在各自 Order，既不参与共享航程冲突检查，也不被 TE 读取投影或共享更新覆盖。

## 已发现文件

| 文件 | 职责与证据入口 |
| --- | --- |
| `server/internal/data/ent/schema/order.go:92` | 四个地点列均存在，最终目的地是本票 `destination_location_id`。 |
| `server/internal/data/ent/schema/sea_transport_execution.go:17` | TE 只存船公司、起运/卸货/中转、船名航次与 ETD/ETA，不存本票最终目的地。 |
| `server/internal/data/ent/schema/sea_master_bill_order_link.go:19` | 每个活动 Link 只有一个 `transport_execution_id`；`:47` 用部分唯一索引保证一票一个活动关系。 |
| `server/api/order/v1/order.proto:343` | Order 契约分开传 origin/destination/discharge/transit；`:859` 的 TE 契约没有 destination。 |
| `server/internal/service/order_write.go:325` | 普通更新独立合并 destination，字段未提供时保留 existing；不从 TE 推断。 |
| `server/internal/service/order_convert.go:27` | Order 读取按各自字段独立输出地点 ID 与名称。 |
| `server/internal/data/order_write.go:81` | 新建原样保存四个地点字段，再经 `:127` 建立共享主单/TE 关联。 |
| `server/internal/data/order_write_draft.go:225` | 更新先校验共享段，`:268` 再调用 `setOrderOptionalReferences` 持久化本票地点。 |
| `server/internal/data/order_convert.go:256` | `setOrderOptionalReferences` 独立设置/清空每个地点；`:158` 的 SE 读取投影只覆盖共同段，不覆盖 destination。 |
| `server/internal/data/order_write_sea_master_bill.go:582` | 航程差异只比船公司、起运/卸货/中转、船名航次、ETD/ETA。 |
| `server/internal/data/sea_order_change_transport_update.go:193` | 共享更新只改 TE 及版本/历史，未写 Order.destination。 |
| `server/internal/data/sea_master_bill.go:83` | 候选枚举活动成员 Link 对应的不同 TE，去重后供显式选择；不是排序后的路线。 |
| `server/internal/data/sea_order_change_split_execute.go:1251` | 拆票创建新 Order 继承源票 destination，共同段另按目标 TE 赋值。 |
| `server/internal/data/sea_order_change_reassignment.go:734` | 改配切换当前 Link/TE 并推进订单版本，不改本票 destination。 |
| `server/api/order/v1/sea_document.proto:183` | SeaBillContent 的全部字段；不存在 PortOfDischarge/PlaceOfDelivery。 |
| `server/internal/data/order_lock_se_snapshot.go:105` | 锁单分别固化 TE 版本、MBL 版本与 HBL 版本，未把它们合成完整逐段行程快照。 |

## 数据流与当前契约

### 1. 新建与关联已有共同段

- `order_write.go:81-87` 原样写入本票地点与船期；`order_write_sea_master_bill.go:48-65` 构造待比对航程时只取起运/卸货/中转、船名航次、ETD/ETA，没有 destination。
- 关联已有 MBL 时要求明确的候选 MBL/TE ID 与版本，事务内读取实际 TE；`order_write_sea_master_bill.go:145` 用 `CheckSeaVoyageConflicts` 比对共同字段。
- 新票 destination 可与共同段卸货港相同，也可不同。服务器没有「若 destination 为空则补 discharge」的推断。

### 2. 普通更新与读取

- `service/order_write.go:325-329` 仅在请求携带 destination 字段时更新本票领域值，与 discharge 分别处理。
- `data/order_write_sea_master_bill.go:357-358` 分别判定主单身份与航程变化；目的港独立修改不会触发共同航程差异。
- `data/order_convert.go:159-167` 对 SE 将 TE 的共同字段覆盖到响应 Order；destination 保持 `order_convert.go:59` 从本票行读出的值。
- 因此当前数据库保留 Order 上的共同字段冗余列，但有活动 TE 时读投影以 TE 为准；本票 destination 以 Order 为准。

### 3. 共享 TE 更新

- `sea_order_change_transport_update.go:128-150` 先按目标 TE 查询精确受影响成员，校验所有成员业务写入门禁；不是按所有同 MBL 票无条件更新。
- `:193-219` 只设置 TE 的共同字段并推进 TE 版本，`:222` 生成不可变 TE 历史。没有 `SetDestinationLocationID` 或订单目的地重写。
- 更新后各票读取共同 TE 的新值，但各自 destination 不动。当前范围不必为这一行为新增模型。

### 4. 拆票与改配

- 拆票新票 destination 在 `sea_order_change_split_execute.go:1251` 继承源票；目标共同段字段在 `:1266-1285` 来自目标 TE 或当前共同段，不影响继承的独立目的地。
- 拆票请求当前没有结果票各自目的地输入。若希望拆票同时分拨不同最终港，需要单独批准该交互范围；最小方案是拆出后各票通过普通编辑改独立目的地。
- 改配会结束旧 Link、创建指向目标 MBL/TE 的活动 Link；`sea_order_change_reassignment.go:734` 只推进 Order.version，保持本票 destination。不能把改配当作给同一票增加第二个并行运输段。

### 5. 单证正文与历史

- `SeaBillContent`（proto `sea_document.proto:183`、领域 `biz/sea_document.go:326`、Ent mixin `schema/sea_bill_content.go:8`）只含收发通知人、货描、件重尺、运费/运输条款、提单形式、放单方式等字段，**没有独立 HBL 卸货港或交付地字段**。
- `SeaHouseBillVersion` 与 `SeaMasterBillVersion` 使用相应不可变正文 mixin，不自动包含 Order.destination。不能声称当前 HBL 正文/历史已经记录每票完整路线或最终交付地。
- TE 快照 `sea_transport_execution_version.go:55-67` 固化共同字段；锁单 `order_lock_se_snapshot.go:105-125` 分别引用 TE、MBL 和 HBL 不可变版本。
- 本期只整理操作票共同段/独立目的港，可保持单证内容模型；若要求打印或历史重现各票最终交付地，需要另明确单证输出与历史口径，不能静默把 Order 活值当成历史单证事实。

## 一张 MBL 下多个 TE 的含义

- 当前结构不是 `MBL → 有序 legs[] → 每票路线`。
- `SeaMasterBillOrderLink` 一个活动关系只选一个 TE；TE 无 sequence、previous、next、父路线或每票段序字段，MBL 也不持有有序 TE 关系。
- `sea_master_bill.go:83-99` 通过活动成员收集不同 TE，候选响应供选择当前实际航次；多条候选不意味着各票先后走完这些 TE。
- 现有 reassignment 是当前运输关系变更；不能用它冒充已批准的后续转船履历。

## 默认值建议（已按用户最新范围收敛）

- 用户已明确本期不建后续航次，默认各分单最终港口相同，仅后续目的地不同才人工修改。
- 新票前端应显式将「最终目的港」默认填为共同段卸货港，并在提交中携带 destination；后端仍保存显式字段，**不增加服务端隐藏推断或兜底**。
- 默认值不等于绑定关系：人工改过的最终目的港、草稿恢复值及用户主动清空须保护，候选重查/共同段变化不能静默覆盖。
- 已有票的 destination 若为空，不应由读取响应或后台查询隐式改写；本期可在明确新建/新分单初始化入口带入，存量编辑保持现有值，是否清空为必填错误由产品交互单独定义。
- 同一个共同段卸货港可以作为多个分单的默认最终目的港，订单地点验证已按唯一 ID 集合处理，重复引用不再误报字段不合法。

## 最小可交付方案与验收

1. 保持后端模型与现有传输字段，明确 `discharge_location_id` 为共同段卸货港，`destination_location_id` 为本票最终目的港。
2. 前端按共同段／本票去向组织可见信息；共享船期明确为到共同段卸货港的 ETA，不能标作每票最终港预计到达时间。
3. 新增分单的最终港默认共同段卸货港；仅人工指定不同目的港时独立保存。共享缺项提示明确缺哪些共同资料及影响范围，不让用户误以为每票都要重填共同段。
4. 后端补充/强化定向回归：两票同 MBL/TE、不同 destination，改共享段前后 destination 均不变；修改一票 destination 不影响另一票或 TE；拆票/改配保持独立去向。
5. 当前无需新增表、迁移、权限、事务或版本字段；任何后续逐段执行能力另行明确真实用例后再规划。

## 可核查测试与不足

- `server/internal/data/order_transaction_integration_test.go:650` 现有用例名明确「多成员共享 MBL 修改航程冲突阻断且目的地不参与共享校验」，`:711-724` 断言同票独立改目的地成功。
- 该用例的第二票没有显式设置与第一票不同的 destination，不能把用例名当作“两票不同最终港全部保持”的完整证明；建议本任务真实回归补足两个已存在不同目的港的成员和共享更新后的保持断言。
- 本研究仅静态核查，没有执行测试或访问数据库。上述行为证据来自代码路径，不是本轮运行验证。

## 相关规范

- `.trellis/spec/server/backend/sea-export-document-contract.md`：一票一分、共享 MBL/TE、独立目的地、候选及正式变更。
- `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md`：继承共同字段、独立业务字段、缺项维护与草稿保护。
- `.trellis/spec/server/backend/database-guidelines.md`：集合校验、事务、锁序与正式迁移验证。
- `.trellis/spec/domain/glossary.md` 与 `architecture-map.md`：TE/MBL/HBL 导航。

## 未发现与风险提示

- 未发现本期所需的“目的地必须等于共同段卸货港”规则，也未发现 destination 被共享 TE 更新覆盖。
- 当前 `destinationLocationId` 在 SE 仍通过港口表校验（`order_sync.go:76-78`）；页面即使写“目的港或地点”，也不能据此宣称已支持任意内陆地址/一般交付地。若要一般交付地，需要额外业务确认与候选模型设计，本期可先称最终目的港。
- Order 上保留共同字段冗余，而 `order_query.go:77,102-103,149-150` 的关键词/ETD/ETA/起运筛选仍直接查询 Order 列；共享 TE 更新只改 TE，因此共享更新后的列表筛选是否与新显示一致，是现存风险点。若本期涉及这些列表显示/筛选，应定向核验；不要无审批扩为整个查询层重构。
- SE 投影中 TE 的 ETD/ETA 为 nil 时，`order_convert.go:163-167` 不清空已有 Order 值；共同段清空船期与旧冗余值可能产生显示不一致。不过当前业务要求 ETD/ETA 必填，是否允许该清空操作需按现有共享更新校验核实，不能仅凭字段可空就宣称必然缺陷。
