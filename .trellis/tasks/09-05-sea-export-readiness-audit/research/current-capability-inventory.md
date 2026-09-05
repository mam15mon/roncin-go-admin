# Research: 海运出口当前内部业务管理能力盘点

- Query: 当前仓库的海运出口是否已具备可上线的内部业务管理闭环；重点覆盖 FCL/LCL/BREAK_BULK × DIRECT/HOUSE、共享 MBL、多 HBL、箱货分配、拆票/改配、改单/作废/Switch、订单锁、异常/附件/节点、费用到财务闭环、权限/组织隔离/并发/审计。船公司、海关、港区及 SI/VGM/舱单等外部平台的一键提交明确不在范围内，不得把缺少外部提交接口列为缺陷。
- Scope: internal
- Date: 2026-09-05

## Findings

### 1. 结论摘要

当前仓库已经形成一套相当完整的海运出口“订单—真实 MBL/HBL—箱货—变更—费用—财务”内部数据骨架和大部分写入闭环，不应再按“只有订单表单”评价。尤其是共享 MBL、多 HBL、HOUSE 箱货守恒分配、拆票/改配、不可变单证版本、改单/作废/Switch、全订单业务锁、普通订单费用进入账单/发票/核销/提成链路，均可找到契约、领域、持久化、前端及测试证据。

但当前还不适合无条件宣布为“海运出口全场景正式上线完成”。候选正式启用阻断项集中在内部运营闭环本身，而不是外部提交：

1. 订单主流程止于“已放单”，且“已放单 + 无活动异常 + 无未账单费用”即可结案；没有开船、到港、交付或操作完成的结案门禁。
2. 新的真实 `SeaMasterBill` / `SeaHouseBill` 模型虽然声明了 `CONFIRMED` / `RELEASED` 状态，但 `SeaDocumentService` 没有确认、放单/签发状态流转命令；正常业务写入只创建 `DRAFT`，仅作废/Switch 能进入 `VOIDED` / `REPLACED`。因而不能在真实海运单证模型内可靠记录“确认/放单结果”。
3. 订单附件当前只是人工登记对象存储键和元数据，页面明确不上传二进制文件；契约中也没有订单附件上传、预览、下载或对象存在性确认接口。若系统需要独立承载操作证据，这是正式启用阻断项。
4. 共享 MBL/航段/箱/HBL 级费用归集和分摊仍只是未完成的第六阶段 Trellis 任务。普通单票费用到财务链路已具备，但共享承运人成本不能按审计友好的方式跨订单/HBL 分摊。

除此之外，节点、异常、放单回单与旧 HBL 模型残留存在明显 P1 缺口；并发现两个可复现的前端参数缺陷。建议按文末优先级与验收条件切分正式启用范围。

### 2. 范围边界：外部提交不是缺陷

本盘点把系统定位为内部货代作业与财务平台：能够记录订舱/截关/截单/VGM 等事实、责任、结果、文件和异常即可；不要求直接向船公司、海关、港区、单一窗口发送 EDI/API，也不要求自动提交 SI、VGM 或舱单。

因此，以下内容没有列为 P0/P1/P2：

- 没有船公司订舱、改配、取消订舱 API；
- 没有海关报关、舱单申报 API；
- 没有港区进港、放行、VGM 回执 API；
- 没有 SI/VGM/舱单“一键提交”按钮；
- 没有第三方平台回执自动同步。

但是，内部不能记录截止时间、负责人、实际结果、附件、状态节点或异常，仍属于本次范围内的缺口。

### 3. 核心文件与职责

| 文件 | 当前职责/证据 |
| --- | --- |
| `server/api/order/v1/order.proto` | SE 订单、FCL/LCL/BREAK_BULK、主流程、截止时间、货物与摘要字段的契约真相源。 |
| `server/api/order/v1/sea_document.proto` | 真实 MBL/HBL、DIRECT/HOUSE、版本、改单、作废、Switch 的 API 契约。 |
| `server/api/order/v1/sea_cargo_allocation.proto` | HOUSE 场景货物—HBL—集装箱的草稿、确认、撤回和汇总回写契约。 |
| `server/api/order/v1/sea_order_change.proto` | 拆票、改配上下文、预览、执行、历史与幂等契约。 |
| `server/internal/data/ent/schema/sea_*.go` | 运输执行、共享 MBL、订单关联、真实 HBL、版本、事件、箱货分配的数据模型。 |
| `server/internal/data/sea_document.go` | 共享单证结构与内容写入、固定加锁顺序、版本冲突和审计。 |
| `server/internal/data/sea_document_change.go` | 改单、作废、Switch 的预览/执行、不可变版本及下游财务事实保护。 |
| `server/internal/data/sea_cargo_allocation.go` | 箱货分配确认/撤回、守恒校验与写保护。 |
| `server/internal/data/sea_order_change.go` | 拆票/改配的跨实体锁定、守恒、幂等与历史。 |
| `server/internal/data/order_lock.go`、`order_write.go` | 全订单业务锁门禁、共享 MBL 成员锁、订单并发版本及结案门禁。 |
| `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx` | 新真实 MBL/HBL 内容、多 HBL、DIRECT/HOUSE、版本与变更操作界面。 |
| `web/src/pages/orders/split.tsx` | 海运拆票/改配操作页。 |
| `.trellis/spec/server/backend/sea-export-document-contract.md` | 当前共享 MBL/多 HBL/箱货/锁顺序契约。 |
| `.trellis/spec/server/backend/sea-document-change-history.md` | 单证版本、改单/作废/Switch 的不可变历史约束。 |
| `.trellis/spec/server/backend/order-lock-and-document-version.md` | 订单锁与版本冲突的统一规则。 |
| `.trellis/tasks/09-02-sea-export-finance-allocation/prd.md` | 尚未实施的共享费用分摊范围与验收标准。 |

### 4. 运输方式 × 单证结构矩阵

`order.proto:87-135` 定义 SE 与 FCL/LCL/BREAK_BULK；`sea_document.proto:149-173` 把 DIRECT/HOUSE 与 HBL 状态作为独立维度。当前模型因此能表达六种组合，但测试证据强度不完全相同。

| 组合 | 当前可记录内容 | 箱货规则 | 结论 |
| --- | --- | --- | --- |
| FCL + DIRECT | 订单、共享/独立 MBL 内容、实际箱、订单货物摘要 | 不建 HBL 分配；可维护实际箱 | 核心数据已具备；真实 MBL 确认/放单状态仍缺 |
| FCL + HOUSE | MBL、多 HBL、货物、实际箱、货物—HBL—箱分配 | 确认时要求订单货量、HBL 货量和各实际箱三轴守恒 | 六种组合中实现与集成测试最强 |
| LCL + DIRECT | 订单、MBL、订单货物摘要 | 无实际箱级必填 | 核心数据已具备；放单结果闭环仍缺 |
| LCL + HOUSE | MBL、多 HBL、货物—HBL 分配 | 分配行的箱可空，确认要求货物和 HBL 守恒 | 领域规则已覆盖，缺完整浏览器 E2E |
| BREAK_BULK + DIRECT | 订单、MBL、件毛体与计费吨 | 禁止箱计划和 VGM 截止时间 | 模式约束明确，缺完整浏览器 E2E |
| BREAK_BULK + HOUSE | MBL、多 HBL、无箱货物分配 | HBL 分配可不带箱，按货量守恒 | 数据模型可表达，缺完整浏览器 E2E |

关键规则证据：

- `server/internal/biz/order_usecase.go:269-410`：创建/更新仅允许 SE/export，并校验时间、散杂货箱计划和 VGM 限制；创建时要求 MBL 输入。
- `server/internal/biz/sea_cargo_allocation.go:848-903`：HOUSE 分配确认要求订单货物全部分完、每张 HBL 有分配；FCL 还要求每个实际箱完全守恒。
- `server/api/order/v1/order_container.proto:45-97` 与 `order_cargo_item.proto:45-94`：实际箱和货物明细各有独立版本。
- `web/src/pages/orders/templates/sea-template.tsx:29-67`：海运表单由基础、运输、单证、货物、备注、人员等区块组成。

需要注意：“模型能表达”不等于六种组合都已经走过一个完整的真实前端—API—PostgreSQL—财务验收；当前最强证据集中在 FCL + HOUSE 的复杂路径。

### 5. 共享 MBL、多 HBL 与箱货分配

#### 5.1 已具备

- `server/internal/data/ent/schema/sea_transport_execution.go:11-43`：运输执行独立于订单，带组织、承运人、起运/卸货/中转、船名航次、ETD/ETA 和版本。
- `server/internal/data/ent/schema/sea_master_bill.go:12-54`：MBL 关联运输执行，以组织、签发方、标准化 MBL 号建立唯一性。
- `server/internal/data/ent/schema/sea_master_bill_order_link.go:14-52`：共享 MBL 与多张操作订单通过历史化 link 关联，每张订单仅一个 active link，link 记录 DIRECT/HOUSE、箱货分配状态和版本。
- `server/internal/data/ent/schema/sea_house_bill.go:12-64`：HBL 是真实一等实体，可一张订单多 HBL，并保留签发主体、状态和版本。
- `server/api/order/v1/sea_document.proto:210-291`：MBL/HBL 各自拥有 shipper、consignee、notify、唛头、品名、件毛体、运费条款、运输条款、提单形式、放单方式和条款等内容，而不是仅保存号码。
- `server/internal/data/order_write.go:188-349`：修改共享 MBL 基础运输事实时会按固定顺序锁定全部成员订单，防止只改一票造成共享事实分叉。
- `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx:363-700`：可标记/取消 DIRECT、新增/复制/删除/保存多 HBL、保存 MBL 内容，并将箱货汇总回写到单证内容。

#### 5.2 箱货守恒

- `server/api/order/v1/sea_cargo_allocation.proto:14-65`：支持读取、保存草稿、确认、撤回、将汇总应用到提单。
- `server/api/order/v1/sea_cargo_allocation.proto:68-184`：分配行明确关联 cargo item、HBL 和可选 container，并返回货物/HBL/箱三个维度的进度。
- `server/internal/data/ent/schema/sea_cargo_allocation.go:13-58`：分配量以字符串精确数值保存，并有组织、订单、link、HBL、货物、箱等外键。
- `server/internal/data/sea_cargo_allocation_test.go:244-264`：确认后状态为 `CONFIRMED`，并测试确认态禁止继续修改货物或保存草稿。
- `server/internal/data/sea_cargo_allocation_test.go`、`server/internal/biz/sea_cargo_allocation_test.go`：已有真实 PostgreSQL 与领域边界/并发测试。

#### 5.3 当前缺口

真实 Sea MBL/HBL 的正常状态流不完整：

- `server/api/order/v1/sea_document.proto:11-146` 的服务方法包含查询、版本、改单、作废、Switch、DIRECT 切换、HBL CRUD、MBL 内容更新，但没有 `ConfirmSeaMasterBill`、`ConfirmSeaHouseBill` 或 `ReleaseSea*Bill`。
- 同文件 `:165-173` 却定义 HBL `DRAFT → CONFIRMED → RELEASED`，数据层也会把 `RELEASED` 当作不可编辑状态（`server/internal/data/sea_document.go:667-691`）。
- 正常新增 HBL 固定写为 `DRAFT`（`server/internal/data/sea_document.go:532-542`）；全仓搜索未发现正常路径把真实 HBL/MBL写为 `CONFIRMED` 或 `RELEASED`，只有作废和 Switch 会写 `VOIDED` / `REPLACED`（`server/internal/data/sea_document_change.go:728-857`）。

这不是外部提交问题，而是内部单证“已确认/已签发/已放单”的结果无法落在当前真实单证实体上。

### 6. 拆票、改配、改单、作废与 Switch

#### 6.1 拆票/改配已具备

- `server/api/order/v1/sea_order_change.proto:11-71`：提供上下文、预览、执行和历史端点。
- `server/api/order/v1/sea_order_change.proto:113-139`：上下文同时带订单版本、MBL/link/单证结构、箱货分配、HBL、货物、箱、草稿费用、附件和指纹。
- `server/api/order/v1/sea_order_change.proto:233-380`：执行请求要求明确选择 HBL、草稿费用、附件的目标归属并携带期望版本；预览返回守恒结果，执行支持幂等。
- `server/api/order/v1/sea_order_change.proto:386-448`：改配显式记录目标 MBL、原因和责任。
- `web/src/pages/orders/detail.tsx:431-467` 与 `web/src/pages/orders/split.tsx`：详情页可进入拆票/改配流程并查看历史。
- `server/internal/data/sea_order_change_integration_test.go:279-2293`：包含拆票/改配、确认配货状态继承、并发、锁状态、费用/附件归属和精度等集成场景。

#### 6.2 改单/作废/Switch 已具备

- `server/api/order/v1/sea_document.proto:19-94`：版本读取、事件历史、改单预览/执行、作废预览/执行、HBL Switch 预览/执行齐全。
- `server/internal/data/sea_document_change.go:288-446`：预览锁定当前版本并识别已确认/已账单费用、账单、发票、核销、提成、提成调整及箱货分配等下游影响。
- `server/internal/data/sea_document_change_integration_test.go:148-564`：覆盖改单、主单作废、并发改单、幂等、Switch、共享成员重校验和已有财务事实阻断。
- `.trellis/spec/server/backend/sea-document-change-history.md`：规定变更版本不可覆盖，事件与执行结果可追溯。

这部分的主要剩余问题不是变更能力，而是变更前缺少正常的真实单证确认/签发状态基线。

### 7. 订单锁、权限、组织隔离、并发与审计

#### 7.1 订单锁已覆盖海运及费用

- `server/internal/data/order_lock.go:50-68`：统一 `ensureOrderBusinessEditable` 业务锁门禁。
- `server/internal/data/order_write.go:352-380`：订单编辑在事务内先锁行，再校验业务锁、版本和状态。
- 海运单证、箱货分配、拆票/改配、异常、附件、人员、箱、货物、放单回单、费用等主要业务写入均调用统一门禁；费用面板在锁定后仍可读及继续财务只读/账单路径，但不能编辑订单费用。
- `server/internal/data/order_lock_integration_test.go:34-611`：真实 PostgreSQL 测试覆盖锁定/解锁以及多类子资源写入阻断和竞态。
- `web/src/pages/orders/detail.tsx:124-195` 与 `web/src/pages/orders/use-order-lock-state.test.ts`：详情页以 fail-closed 方式处理锁加载失败、未知状态和已锁状态。

因此，“海运出口订单锁能否禁止编辑订单费用”的答案是肯定的：后端门禁覆盖费用写入，详情/费用界面也消费锁状态。锁本身不是当前海运启用缺口。

#### 7.2 权限与组织隔离

- `server/internal/access/manifest.go:117-178,283-302`：SE 订单拥有 read/create/update/transition、里程碑、附件、人员、箱、货物、异常、放单回单、费用、拆票、改配、锁、改单、作废、Switch 等独立操作权限。
- 各海运 proto 的订单权限规则均带 `DATA_SCOPE_ORGANIZATION`；例如 `sea_document.proto:14-145`。
- `server/internal/data/order_query.go:25-34,49-158`：订单详情和列表均以 `organization_id` 谓词过滤。
- Sea MBL、HBL、link、分配、版本和事件实体均保存组织 ID，写入时同时验证订单组织和共享实体组织。

#### 7.3 并发与审计

- 订单、MBL、HBL、link、运输执行、箱货分配、箱、货物、费用、财务单据均有版本/期望版本或状态比较；复杂路径采用 `ForUpdate` 加固定锁顺序。
- `.trellis/spec/server/backend/sea-export-document-contract.md` 与 `order-lock-and-document-version.md` 明确共享 MBL 的订单成员排序加锁、冲突映射和锁后重校验。
- 单证、箱货、拆票/改配、锁、订单状态和费用的关键写入在同一事务中追加业务审计或不可变事件；对应集成测试包含审计失败回滚和并发无死锁场景。

当前没有发现跨组织读取共享 MBL/HBL 的直接证据缺口，但本次是静态研究，没有执行新的越权渗透测试。

### 8. 截止时间、负责人、节点、结果和异常

#### 8.1 已能记录的离散事实

- `server/api/order/v1/order.proto:226-312`：订单可记录 ETD/ETA、SI cutoff、document cutoff、customs cutoff、VGM cutoff、declaration cutoff、cargo ready、收货时间、船名航次、订舱/配舱/操作备注等。
- `server/api/order/v1/order_personnel.proto:35-70`：可按 creator/operator/sales/customer service/document/commercial/associate 等角色分配人员；`server/internal/data/ent/schema/order_personnel.go:18-42` 约束一票每角色一人。
- `server/api/order/v1/milestone.proto:29-56`：可保存里程碑类型、模板代码/名称、实际发生时间、备注、更新人，并以订单版本并发控制。
- `server/api/order/v1/order_abnormal_case.proto:45-87`：可引用组织级异常类型，记录活动/已解决、标记人/时间、解决人/时间。

#### 8.2 节点管理仍是部分能力

当前截止时间、人员、里程碑是三组彼此松散的数据：

- 里程碑只有实际发生时间和备注，没有计划/承诺时间、截止时间、状态、节点负责人、结果代码、附件关联、逾期标识或提醒（`server/api/order/v1/milestone.proto:29-56`）。
- 页面要求用户手输任意类型，例如 `BOOKING_CONFIRMED`（`web/src/pages/orders/components/drawers/MilestoneDrawer.tsx:111-184`），没有当前有效的强制节点模板或按 FCL/LCL/BREAK_BULK、DIRECT/HOUSE 派生的节点清单。
- 主流程也不校验关键里程碑是否完成。因此，有截止时间不代表有人负责，有人员角色不代表负责某节点，有里程碑不代表已有计划截止或证据。

建议内部节点至少覆盖：订舱确认/拿 SO、截补料、截 VGM、截关/报关放行结果、进港/装船、实际开航、到港/交付、提单确认与放单；每节点应有计划截止、负责人、实际完成、结果、附件/外部回执引用和异常关联。是否需要每个节点都成为订单主状态，可在设计阶段区分“门禁节点”和“记录节点”。

#### 8.3 异常管理仍是标签式能力

`OrderAbnormalCase` 只有异常类型和 ACTIVE/RESOLVED 状态，没有异常描述、严重级别、责任人、发现/要求解决时间、原因、处置方案、解决说明、损失金额或附件关系（`server/api/order/v1/order_abnormal_case.proto:52-87`）。它能阻止结案，但不能独立支撑异常工单闭环。

### 9. 附件与操作证据

#### 9.1 已具备的元数据与审计

- `server/api/order/v1/order_attachment.proto:11-101`：可按组织权限列出、注册和解除附件引用，保存文档类型、文件名、MIME、大小、对象键、校验和、上传人和资产 ID。
- `server/internal/data/order_attachment.go:52-113`：在事务中锁订单、校验订单业务锁、按对象键复用/创建资产、创建幂等引用并写审计。

#### 9.2 不能视为完整附件闭环

- 页面直接说明“登记外部对象存储引用与元数据，不直接进行二进制文件上传”，并要求用户手填 object key、MIME、大小和可选 checksum（`web/src/pages/orders/components/drawers/AttachmentDrawer.tsx:145-207`）。
- 订单附件 proto 仅有 list/register/remove，没有 prepare upload、确认上传、预签名下载、预览或获取内容接口（`server/api/order/v1/order_attachment.proto:11-35`）。
- 数据层在找不到 object key 时直接创建资产元数据，没有调用存储服务验证对象存在或大小/校验和（`server/internal/data/order_attachment.go:62-90`）。
- 当前附件没有直接关联节点、异常或真实 MBL/HBL，只能依赖自由文档类型和订单级引用表达语义。

除非上线环境已有另一个受控文件系统、用户能可靠取得 object key，且该外部过程经过业务验收，否则不能把它算作“内部系统已可上传、查看、下载关键回执与结果附件”。

### 10. 放单回单与新旧 HBL 模型错位

仓库中仍存在旧的通用 `OrderShippingDocument`，但海运出口已经迁移到真实 `SeaHouseBill`：

- `server/internal/data/order_shipping_document.go:21-47`：读取或写入 SE 订单的旧通用分单会返回 `ErrSeaShippingDocumentsDeprecated`。
- 订单列表仍为所有订单配置 `ShippingDocumentDrawer`，SE 的“分单管理”入口会打开该旧抽屉（`web/src/pages/orders/list.tsx:171-177,301-305`）。这是可直接触发的前端/后端错位。
- `web/src/pages/orders/release-pod-panel.tsx:98-125` 打开放单回单时也查询旧 `OrderShippingDocument`，因此 SE 会加载失败。
- `server/internal/data/order_release_pod.go:30-46` 的 `shipping_document_id` 只验证旧 `OrderShippingDocument`，不能关联真实 `SeaHouseBill`；虽然该字段可空，海运放单回单无法精确归属当前真实 HBL。

这意味着系统同时存在“新真实海运 HBL”和“仍指向已废弃 HBL 的列表/回单 UI”。正确的新单证内容可在订单详情使用，但列表分单与放单回单并未完成迁移。

### 11. 主流程与结案闭环

`server/api/order/v1/order.proto:166-176` 和 `server/internal/biz/order_transition.go:42-49` 的 SE 主流程是：

`DRAFT → BOOKED → SPACE_ALLOCATED → TRUCKING_ARRANGED（可跳）→ DOCUMENT_CUTOFF → CUSTOMS_DECLARATION_ARRANGED → DOCUMENT_RELEASED`

问题在于：

- 没有订舱确认号/SO 结果状态、报关放行、进港、已装船、实际开航、到港、交付或操作完成主节点。
- `TransitionStatus` 只校验状态机和订单版本，不校验真实 MBL/HBL 是否确认/放单、关键里程碑是否发生或附件是否存在（`server/internal/biz/order_transition.go:82-103`）。
- 结案把 `DOCUMENT_RELEASED` 直接视为流程完成；只要没有活动异常和未进入 BILLED/CANCELLED 的订单费用即可关闭（`server/internal/biz/order_transition.go:121-142`；事务内重校验见 `server/internal/data/order_write.go:607-638`）。

因此，一票仍未实际开船或未完成操作的订单，在系统层面可以在“已放单”后结案。这是内部状态和结案完整性问题，与是否调用外部平台无关。

### 12. 费用到财务闭环

#### 12.1 普通订单级费用链路已具备

- `server/api/order/v1/order_fee.proto:12-142,200-260`：费用有应收/应付、DRAFT/CONFIRMED/BILLED/CANCELLED、币种、汇率、本位币、税额、费用日期和版本，并支持新增、修改、确认、反确认、删除。
- `server/internal/data/ent/schema/order_fee.go:12-75`：费用事实持久化到订单，保存精确小数、结算对象、币种、汇率及状态。
- `server/internal/data/finance_bill.go:303` 后的建账路径会锁定费用、生成账单行并将费用置为 `BILLED`，审计与业务写在同一事务。
- `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/prd.md:51-70` 记录真实 PostgreSQL 的 CNY 与 USD/EUR 两组“费用→账单→发票→流水→核销→提成”验收，0 跳过、0 失败。

#### 12.2 共享海运费用分摊尚未实施

当前 `OrderFee` 只有 `order_id`，没有费用归属对象类型及 transport execution/MBL/HBL/container 外键。未完成任务 `.trellis/tasks/09-02-sea-export-finance-allocation/prd.md:3-25` 明确要求：

- 费用声明操作票、运输执行/航段、MBL、HBL 或箱级计费对象；
- 共享承运人成本通过版本化分摊落到操作票/HBL；
- 按币种精确守恒，重分摊不覆盖已确认/开票/核销事实；
- 拆票/改配/改单对账单、发票、核销、提成影响显式处理。

其验收项目前全部未勾选（同文件 `:16-21`）。所以：

- 单票、人工直接归属到订单的费用，可以走完整财务链；
- 共享 MBL、共箱、共航次的承运人成本仍需要人工先拆成各订单费用，缺少可追溯的分摊依据、版本和守恒门禁；
- 若正式启用范围包含共享 MBL 集运业务的真实毛利与成本结算，该缺口应按 P0；若首期明确只做单票直归或允许受控人工分摊，可按 P1 并记录上线限制。

### 13. 当前测试证据强度

已找到的强证据：

- `server/internal/data/sea_document_test.go`：真实 PostgreSQL 单证聚合、审计失败回滚、并发无死锁、更新校验。
- `server/internal/data/sea_cargo_allocation_test.go`：HOUSE 配货草稿/确认/撤回/写保护与并发。
- `server/internal/data/sea_order_change_integration_test.go`：拆票/改配跨实体集成路径。
- `server/internal/data/sea_document_change_integration_test.go`：改单/作废/Switch、幂等、并发及财务影响阻断。
- `server/internal/data/order_lock_integration_test.go`：全订单业务锁及子资源竞态。
- `web/src/pages/orders/templates/sea-template.test.tsx`：新海运表单和多 HBL 交互。
- `web/src/pages/orders/components/SeaDocumentHistoryActions.test.tsx`：单证变更历史操作。
- 已归档外币财务 E2E 任务记录真实财务链路验收结果。

未找到一条覆盖六种运输/单证组合并串起“订单创建→节点→真实 MBL/HBL→箱货→费用→财务→结案”的浏览器级全链路 E2E。它首先是验收证据缺口，不应在未运行现有测试的本研究阶段直接断言为代码故障。

### 14. 候选缺口优先级

以下优先级按“系统内部能否形成真实、可审计、不可误关单的业务闭环”评估。P0 表示正式启用前应解决或以书面方式缩小上线范围；不是生产事故定级。

#### P0 候选：正式启用阻断或必须限制范围

| 缺口 | 触发场景与风险 | 当前绕行 | 建议最小验收 |
| --- | --- | --- | --- |
| 订单流程止于已放单并可提前结案 | 货未开/未到/未交付仍可结案，系统状态失真 | 依赖人工不点击结案、自由里程碑备注 | 建立“已开船/操作完成”或等价结案门禁；结案在同事务重校验关键节点 |
| 真实 Sea MBL/HBL 无确认/放单状态命令 | HBL/MBL 正常状态长期停在 DRAFT，无法记录确认/签发/放单结果 | 订单主状态写“已放单”，但与真实单证不一致 | 提供受权限、版本、订单锁、共享成员锁和审计保护的确认/放单流转；订单主状态与单证事实有明确一致性规则 |
| 订单附件无可用上传/查看/下载闭环 | SI、VGM、报关放行、提单确认件、异常证据不能在系统内可靠存取 | 手工从其他系统取得并填写 object key；当前未找到已验收流程 | 上传准备/完成确认、存储对象校验、受权下载/预览、审计；能关联节点/异常/真实单证 |
| 共享海运费用分摊未实施（条件 P0） | 一 MBL/航次/箱跨多票时，共享成本无法守恒且可追溯地分摊，毛利和财务依据不可靠 | 人工拆成订单费用 | 完成现有 finance-allocation 任务；至少两种批准方法、币种守恒、版本化调整及财务事实保护 |

#### P1 候选：核心流程明显不完整

| 缺口 | 证据/影响 | 建议方向 |
| --- | --- | --- |
| 节点缺计划截止、负责人、结果、附件、逾期与提醒 | milestone 只有自由 type、occurred_at、note；截止和人员互不关联 | 建立受控节点定义和订单节点实例，区分计划/实际/结果/负责人/证据 |
| 异常只是类型标签 | 无描述、级别、责任人、截止、原因、处置/解决说明和附件 | 扩展为可追责的异常事件/工单；保留结案阻断 |
| SE 列表仍打开已废弃通用 HBL | 前端入口调用后端明确拒绝的旧模型 | SE 隐藏旧入口并跳到真实 SeaDocumentSection，或改造列表抽屉消费真实 API |
| 放单回单关联旧 HBL | SE 加载旧 HBL 失败，`shipping_document_id` 不能指向 SeaHouseBill | 将海运放单/回单关联迁移到真实 MBL/HBL；明确 DIRECT 与多 HBL 的归属 |
| 箱/货物删除漏传 `expectedVersion` | `ContainerDrawer.tsx:166-170`、`CargoItemDrawer.tsx:151-155` 只传 orderId/id；服务端把 `GetExpectedVersion()` 传入并要求非零 | 前端传记录版本并补组件/API 测试；保留 409 冲突提示 |
| 运输执行只有单一主船名航次 | 当前 schema 只有一个 vessel/voyage/ETD/ETA；未见多航段、订舱确认号/SO、每段实际状态 | 若业务含中转/驳船，新增 booking/segment 内部事实；若首期仅单航段，写清限制 |

#### P2 候选：一致性、体验和技术债

| 缺口 | 影响 |
| --- | --- |
| 列表页部分子资源抽屉不消费订单锁状态 | 后端会安全拒绝，但按钮仍可见，用户提交后才得到锁错误；详情页已 fail-closed |
| `server/internal/data/order_milestone.go:19-68` 使用 `r.data.db` 而不是 `Data.client(ctx)` | 偏离当前共享事务规范；现路径通常独立事务，暂未证明数据错误，但增加未来事务组合风险 |
| 六种组合缺统一全链路验收矩阵 | 局部领域/PG/组件测试强，但无法一眼证明每个模式从录单走到结案和财务 |
| 若干单证业务值为自由文本 | bill form、release type、terms、自由 milestone type 易产生口径不一致；需结合业务主数据治理，不宜在本任务中猜测枚举 |

### 15. 建议的正式启用边界

如果马上做受控试运行，可以把范围限定为：

- 内部订单录入、共享 MBL/多 HBL 内容、FCL HOUSE 箱货分配；
- 拆票/改配、改单/作废/Switch 只在已有测试覆盖的状态与财务门禁内使用；
- 单票直接归属的订单费用进入现有财务链；
- 订单锁作为冻结业务字段和订单费用的统一门禁；
- 暂不把系统中的 `DOCUMENT_RELEASED` 当作实际运输完成，不允许在操作完成前结案；
- 在附件闭环完成前，明确由哪个受控文件系统保存证据，并把 object key 登记作为临时且经业务批准的操作规程；
- 共享成本仍需人工拆分时，不宣称共享 MBL 毛利/成本已自动、可审计分摊。

若要宣布“海运出口正式全量可用”，建议至少先解决上述无条件 P0，并对共享 MBL 成本分摊是否纳入首发作明确业务决定。

## Caveats / Not Found

- 本研究是规划阶段的静态代码、规格、历史任务和测试源盘点；按父任务要求没有运行完整审计、生成、测试或真实业务数据演练。
- 没有把缺少船公司、海关、港区、SI/VGM/舱单外部提交接口列为缺陷；所有优先项均为内部记录、状态、责任、证据或财务闭环问题。
- 未找到真实 Sea MBL/HBL 的正常 `CONFIRMED` / `RELEASED` 写入端点；如果存在尚未提交的外部分支或运行期私有服务，本仓库静态证据无法覆盖。
- 未找到订单附件的上传准备、上传完成确认、预签名下载或预览端点；仓库其他领域可能有对象存储能力，但没有证据表明已接入订单附件流程。
- 未执行权限越权、跨组织、死锁或浏览器 E2E；“已具备”表示仓库存在相符的跨层代码和既有测试，不等价于本次重新验收通过。
- FCL/LCL/BREAK_BULK × DIRECT/HOUSE 的业务合法性可能还受公司具体产品规则限制；当前判断只说明模型与领域规则可表达，不替业务确认哪些组合允许销售和执行。
- P0/P1/P2 是候选规划优先级。共享费用分摊是否为 P0 取决于首发是否包含一 MBL/航次/箱跨多票的真实结算；多航段运输执行是否为 P1 取决于首发航线和中转业务范围。
