# Research: 海运出口当前内部业务管理能力盘点

- Query: 当前仓库的海运出口是否已具备可上线的内部业务管理闭环；重点覆盖 FCL/LCL/BREAK_BULK × DIRECT/HOUSE、共享 MBL、多 HBL、箱货分配、拆票/改配、改单/作废/Switch、订单锁、异常/附件/节点、费用到财务闭环、权限/组织隔离/并发/审计。船公司、海关、港区及 SI/VGM/舱单等外部平台的一键提交明确不在范围内，不得把缺少外部提交接口列为缺陷。
- Scope: internal
- Date: 2026-09-05

## Findings

### 1. 结论摘要

当前仓库已经形成一套相当完整的海运出口“订单—真实 MBL/HBL—箱货—变更—费用—财务”内部数据骨架和主要写入闭环，不应再按“只有订单表单”评价。尤其是共享 MBL、多 HBL、HOUSE 箱货守恒分配、拆票/改配、不可变单证版本、改单/作废/Switch、全订单业务锁、普通订单费用进入账单/发票/核销/提成链路，均可找到契约、领域、持久化、前端及测试证据。

结合用户在审计后的业务确认，当前上线口径是简化人工流程：每个业务步骤由人员判断完成后在 UI 点击对应流程按钮；订单级流程状态是权威事实；“已放单”是批准的终点，满足活动异常和费用门禁后允许人工结案。在这一明确边界内，海运出口可以正式使用。

以下原候选缺口已因产品边界确认而调整为“不适用”或后续增强，不再阻断上线：

1. 不要求实际开航、到港、交付或操作完成状态，也不要求相应结案门禁。
2. 不要求每张真实 `SeaMasterBill` / `SeaHouseBill` 独立确认、签发或放单；订单级“已放单”是当前权威业务口径。模型中的 `DRAFT` / `CONFIRMED` / `RELEASED` 属于未启用冗余语义，UI 不应将其展示为与订单流程冲突的业务状态。
3. 关键文件由现有受控外部文件库保存，订单附件只登记对象键和元数据；系统不承担唯一文件真相源职责。
4. 共享成本由人员在线下计算后逐票录入；系统负责录入后的订单费用及财务链，不保存或自动验证分摊依据、版本与守恒过程。

仍需保留并修复的真实软件问题是：SE 列表旧“分单管理”入口、ReleasePod 仍引用旧 HBL、箱/货物删除漏传 `expectedVersion`，以及真实 PostgreSQL 完整集成套件触发的拆票并发错误语义不稳定。它们不推翻主体业务可正式使用的结论，但相关入口必须按文末操作边界规避。

### 2. 范围边界：外部提交不是缺陷

本盘点把系统定位为简化的内部货代作业与财务平台：员工在外部渠道完成实际业务后，在 UI 人工点击订单流程按钮作为系统内正式确认；不要求直接向船公司、海关、港区、单一窗口发送 EDI/API，也不要求自动提交 SI、VGM 或舱单。

因此，以下内容没有列为 P0/P1/P2：

- 没有船公司订舱、改配、取消订舱 API；
- 没有海关报关、舱单申报 API；
- 没有港区进港、放行、VGM 回执 API；
- 没有 SI/VGM/舱单“一键提交”按钮；
- 没有第三方平台回执自动同步。

复杂节点负责人、结果、SLA、自动提醒和完整异常工单也不是当前必需范围；现有订单人员、截止时间、流程按钮、备注、里程碑和活动异常门禁满足批准的简单协作方式。需要更细粒度协作时再作为后续增强评估。

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
| FCL + DIRECT | 订单、共享/独立 MBL 内容、实际箱、订单货物摘要 | 不建 HBL 分配；可维护实际箱 | 核心数据已具备；放单结果按订单级按钮人工确认 |
| FCL + HOUSE | MBL、多 HBL、货物、实际箱、货物—HBL—箱分配 | 确认时要求订单货量、HBL 货量和各实际箱三轴守恒 | 六种组合中实现与集成测试最强 |
| LCL + DIRECT | 订单、MBL、订单货物摘要 | 无实际箱级必填 | 核心数据已具备；放单结果按订单级按钮人工确认 |
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

#### 5.3 当前未启用的单证状态冗余

真实 Sea MBL/HBL 的正常状态流不完整：

- `server/api/order/v1/sea_document.proto:11-146` 的服务方法包含查询、版本、改单、作废、Switch、DIRECT 切换、HBL CRUD、MBL 内容更新，但没有 `ConfirmSeaMasterBill`、`ConfirmSeaHouseBill` 或 `ReleaseSea*Bill`。
- 同文件 `:165-173` 却定义 HBL `DRAFT → CONFIRMED → RELEASED`，数据层也会把 `RELEASED` 当作不可编辑状态（`server/internal/data/sea_document.go:667-691`）。
- 正常新增 HBL 固定写为 `DRAFT`（`server/internal/data/sea_document.go:532-542`）；全仓搜索未发现正常路径把真实 HBL/MBL写为 `CONFIRMED` 或 `RELEASED`，只有作废和 Switch 会写 `VOIDED` / `REPLACED`（`server/internal/data/sea_document_change.go:728-857`）。

这些事实仍然成立，但用户已确认当前不按每张 MBL/HBL 跟踪确认、签发或放单状态，订单级流程按钮才是正式业务事实。因此该状态流不再计为缺口或上线阻断；UI 应避免把未启用的 DRAFT 状态展示成与订单“已放单”相冲突的业务结论。若未来需要逐张提单跟踪，再单独启用并设计这组状态流。

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

这部分变更能力可以独立使用；当前批准流程不依赖真实单证的独立确认/签发状态基线。

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

#### 8.2 复杂节点管理属于后续增强

当前截止时间、人员、里程碑是三组彼此松散的数据：

- 里程碑只有实际发生时间和备注，没有计划/承诺时间、截止时间、状态、节点负责人、结果代码、附件关联、逾期标识或提醒（`server/api/order/v1/milestone.proto:29-56`）。
- 页面要求用户手输任意类型，例如 `BOOKING_CONFIRMED`（`web/src/pages/orders/components/drawers/MilestoneDrawer.tsx:111-184`），没有当前有效的强制节点模板或按 FCL/LCL/BREAK_BULK、DIRECT/HOUSE 派生的节点清单。
- 主流程也不校验关键里程碑是否完成。因此，有截止时间不代表有人负责，有人员角色不代表负责某节点，有里程碑不代表已有计划截止或证据。

用户已确认当前以订单级流程按钮人工确认各步骤，不要求将计划截止、负责人、结果、附件和提醒组合成独立任务工作台。上述字段缺失不影响当前批准流程正式使用；如果团队以后需要精细排班、逾期统计或自动提醒，再将其作为增强设计，且不能把届时新增的节点事实描述成外部平台自动回传。

#### 8.3 当前异常能力满足简化门禁

`OrderAbnormalCase` 只有异常类型和 ACTIVE/RESOLVED 状态，没有异常描述、严重级别、责任人、发现/要求解决时间、原因、处置方案、解决说明、损失金额或附件关系（`server/api/order/v1/order_abnormal_case.proto:52-87`）。用户当前只要求异常能够登记、解决并阻止结案，因此这套标签式能力满足批准范围；完整异常工单属于后续增强。

### 9. 附件与操作证据

#### 9.1 已具备的元数据与审计

- `server/api/order/v1/order_attachment.proto:11-101`：可按组织权限列出、注册和解除附件引用，保存文档类型、文件名、MIME、大小、对象键、校验和、上传人和资产 ID。
- `server/internal/data/order_attachment.go:52-113`：在事务中锁订单、校验订单业务锁、按对象键复用/创建资产、创建幂等引用并写审计。

#### 9.2 当前由外部受控文件库承担文件真相源

- 页面直接说明“登记外部对象存储引用与元数据，不直接进行二进制文件上传”，并要求用户手填 object key、MIME、大小和可选 checksum（`web/src/pages/orders/components/drawers/AttachmentDrawer.tsx:145-207`）。
- 订单附件 proto 仅有 list/register/remove，没有 prepare upload、确认上传、预签名下载、预览或获取内容接口（`server/api/order/v1/order_attachment.proto:11-35`）。
- 数据层在找不到 object key 时直接创建资产元数据，没有调用存储服务验证对象存在或大小/校验和（`server/internal/data/order_attachment.go:62-90`）。
- 当前附件没有直接关联节点、异常或真实 MBL/HBL，只能依赖自由文档类型和订单级引用表达语义。

用户已明确现有受控外部文件库是关键文件真相源，订单附件只承担引用和元数据登记。因此不能把当前实现描述成“系统内已可上传、查看、下载关键文件”，但完整文件闭环也不属于当前上线条件；未来若要把本系统改为唯一文件库，必须重新评估上述缺失能力。

### 10. 放单回单与新旧 HBL 模型错位

仓库中仍存在旧的通用 `OrderShippingDocument`，但海运出口已经迁移到真实 `SeaHouseBill`：

- `server/internal/data/order_shipping_document.go:21-47`：读取或写入 SE 订单的旧通用分单会返回 `ErrSeaShippingDocumentsDeprecated`。
- 订单列表仍为所有订单配置 `ShippingDocumentDrawer`，SE 的“分单管理”入口会打开该旧抽屉（`web/src/pages/orders/list.tsx:171-177,301-305`）。这是可直接触发的前端/后端错位。
- `web/src/pages/orders/release-pod-panel.tsx:98-125` 打开放单回单时也查询旧 `OrderShippingDocument`，因此 SE 会加载失败。
- `server/internal/data/order_release_pod.go:30-46` 的 `shipping_document_id` 只验证旧 `OrderShippingDocument`，不能关联真实 `SeaHouseBill`；虽然该字段可空，海运放单回单无法精确归属当前真实 HBL。

这意味着系统同时存在“新真实海运 HBL”和“仍指向已废弃 HBL 的列表/回单 UI”。正确的新单证内容可在订单详情使用，但列表分单与放单回单并未完成迁移。

### 11. 主流程与人工结案闭环

`server/api/order/v1/order.proto:166-176` 和 `server/internal/biz/order_transition.go:42-49` 的 SE 主流程是：

`DRAFT → BOOKED → SPACE_ALLOCATED → TRUCKING_ARRANGED（可跳）→ DOCUMENT_CUTOFF → CUSTOMS_DECLARATION_ARRANGED → DOCUMENT_RELEASED`

当前事实是：

- 没有订舱确认号/SO 结果状态、报关放行、进港、已装船、实际开航、到港、交付或操作完成主节点。
- `TransitionStatus` 只校验状态机和订单版本，不校验真实 MBL/HBL 是否确认/放单、关键里程碑是否发生或附件是否存在（`server/internal/biz/order_transition.go:82-103`）。
- 结案把 `DOCUMENT_RELEASED` 直接视为流程完成；只要没有活动异常和未进入 BILLED/CANCELLED 的订单费用即可关闭（`server/internal/biz/order_transition.go:121-142`；事务内重校验见 `server/internal/data/order_write.go:607-638`）。

用户已确认所有步骤由人员在 UI 人工确认，“已放单”就是当前批准的流程终点；不要求系统记录或验证实际开航、到港、交付或操作完成。因此，“已放单 + 无活动异常 + 无未处理费用”后允许人工结案符合业务规则，不再计为状态缺口。系统仍只证明人员完成了对应按钮确认，不能据此声称已自动验证船舶或外部运输事实。

### 12. 费用到财务闭环

#### 12.1 普通订单级费用链路已具备

- `server/api/order/v1/order_fee.proto:12-142,200-260`：费用有应收/应付、DRAFT/CONFIRMED/BILLED/CANCELLED、币种、汇率、本位币、税额、费用日期和版本，并支持新增、修改、确认、反确认、删除。
- `server/internal/data/ent/schema/order_fee.go:12-75`：费用事实持久化到订单，保存精确小数、结算对象、币种、汇率及状态。
- `server/internal/data/finance_bill.go:303` 后的建账路径会锁定费用、生成账单行并将费用置为 `BILLED`，审计与业务写在同一事务。
- `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/prd.md:51-70` 记录真实 PostgreSQL 的 CNY 与 USD/EUR 两组“费用→账单→发票→流水→核销→提成”验收，0 跳过、0 失败。

#### 12.2 共享海运费用自动分摊尚未实施

当前 `OrderFee` 只有 `order_id`，没有费用归属对象类型及 transport execution/MBL/HBL/container 外键。未完成任务 `.trellis/tasks/09-02-sea-export-finance-allocation/prd.md:3-25` 明确要求：

- 费用声明操作票、运输执行/航段、MBL、HBL 或箱级计费对象；
- 共享承运人成本通过版本化分摊落到操作票/HBL；
- 按币种精确守恒，重分摊不覆盖已确认/开票/核销事实；
- 拆票/改配/改单对账单、发票、核销、提成影响显式处理。

其验收项目前全部未勾选（同文件 `:16-21`）。所以：

- 单票、人工直接归属到订单的费用，可以走完整财务链；
- 共享 MBL、共箱、共航次的承运人成本仍需要人工先拆成各订单费用，缺少可追溯的分摊依据、版本和守恒门禁；
- 用户已批准由人员在线下计算共享成本并逐票录入；当前系统负责保存人工分摊后的订单费用并继续财务链，不负责保存分摊依据、版本或自动守恒过程。自动分摊因此是后续增强，不是当前 P0/P1，也不得宣称已经实现。

### 13. 当前测试证据强度

已找到并在本次正式审计中复核的强证据：

- `server/internal/data/sea_document_test.go`：真实 PostgreSQL 单证聚合、审计失败回滚、并发无死锁、更新校验。
- `server/internal/data/sea_cargo_allocation_test.go`：HOUSE 配货草稿/确认/撤回/写保护与并发。
- `server/internal/data/sea_order_change_integration_test.go`：拆票/改配跨实体集成路径。
- `server/internal/data/sea_document_change_integration_test.go`：改单/作废/Switch、幂等、并发及财务影响阻断。
- `server/internal/data/order_lock_integration_test.go`：全订单业务锁及子资源竞态。
- `web/src/pages/orders/templates/sea-template.test.tsx`：新海运表单和多 HBL 交互。
- `web/src/pages/orders/components/SeaDocumentHistoryActions.test.tsx`：单证变更历史操作。
- 已归档外币财务 E2E 任务记录真实财务链路验收结果。

本次审计实际验证结果：

- `go -C server test ./...`：通过。
- `go -C server vet ./...`：通过。
- `pnpm --dir web test`：62 个测试文件、202 个测试通过。
- `pnpm --dir web tsc`：通过。
- 海运针对性前端测试：`sea-template`、`SeaDocumentHistoryActions`、`split`、`release-pod-panel` 共 4 个文件、14 个测试通过。
- 真实 PostgreSQL：SeaCargoAllocation、SeaDocument 聚合、审计失败回滚、并发无死锁、更新校验、SeaDocumentChange、MBL Void、并发 Amendment、共享成员重验、历史发票/核销阻断、OrderLock 均通过。
- `TestSeaOrderSplitAndReassignment_PostgresIntegration` 的绝大多数子场景通过，但完整父测试连续两次在“并发拆票竞争”失败：仍保持 1 个成功、另 1 个失败，没有双成功或重复写入；失败请求却返回 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT`（“分单数量不足”），而测试和契约要求 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT`。单独执行该子场景 3 次通过，说明结果受完整套件时序影响，但完整父测试已稳定复现语义漂移，因此质量门记为失败，不能被单独子测试掩盖。
- OrderLock 首次使用另一专用集成库运行时因该测试库缺少 `pg_trgm` extension/operator class 失败；切换到已完成正式迁移的隔离 Schema 后通过。该项属于测试数据库前置环境说明，不计为产品功能故障。

仍未找到一条覆盖六种运输/单证组合并串起“订单创建→节点→真实 MBL/HBL→箱货→费用→财务→结案”的浏览器级全链路 E2E。它首先是验收证据缺口；但上述拆票完整集成测试失败属于已复现代码/契约问题，不再只是证据不足。

### 14. 软件问题与后续增强优先级

按用户批准边界，本次没有因缺少复杂货代 ERP 能力而成立的 P0。下列 P1 是从现有 UI 可触发或真实集成测试已证实的软件问题；它们不推翻海运出口主体正式使用结论，但启用对应操作前必须修复或执行明确规避。

#### P1：已证实的软件问题

| 问题 | 证据/影响 | 当前操作边界 | 建议最小验收 |
| --- | --- | --- | --- |
| SE 列表仍打开已废弃通用 HBL | 前端入口调用后端明确拒绝的旧模型，入口会直接失败 | SE 仅从订单详情的 SeaDocumentSection 维护真实单证 | SE 隐藏旧入口或跳到真实单证区；非 SE 旧接口保持不变 |
| ReleasePod 仍关联旧 HBL | SE 加载旧 HBL 失败，`shipping_document_id` 不能指向 SeaHouseBill | SE 放单回单不关联旧 document_id；需要时在备注人工标明 HBL | ReleasePod 支持真实 MBL/HBL 或明确隐藏 SE 入口，并通过前端与真 PG 关联测试 |
| 箱/货物删除漏传 `expectedVersion` | `ContainerDrawer.tsx:166-170`、`CargoItemDrawer.tsx:151-155` 只传 orderId/id，正常删除会被后端拒绝 | 不使用删除入口，不直接改数据库 | 前端传记录版本；无版本失败关闭；409 提示刷新；补两个组件请求断言 |
| 并发拆票失败请求返回错误语义 | 完整 PG 套件中只有一票成功且无重复写入，但失败方返回 400“分单数量不足”而非 409 版本冲突；完整质量门 FAIL | 同一订单只允许一人拆票；异常错误先刷新，不盲目重试 | 版本检查优先或稳定映射 VERSION_CONFLICT；完整父测试 `-count=3` 和单独子测试均通过 |

#### P2：当前范围外的增强与技术债

| 项目 | 当前结论 |
| --- | --- |
| 实际开航、到港、交付、操作完成及相应结案门禁 | 用户不需要；订单级“已放单”是批准终点，不列缺口 |
| 每张 Sea MBL/HBL 独立确认、签发、放单状态 | 用户不需要；当前枚举属于未启用冗余，UI 不应误导，未来按需启用 |
| 复杂节点、SLA、自动提醒和完整异常工单 | 用户不需要；现有流程按钮、备注、里程碑和活动异常门禁满足当前协作 |
| 共享费用自动分摊 | 人工线下计算后逐票录入是批准方案；系统没有分摊依据、版本或自动守恒能力，未来按需增强 |
| 系统内完整附件上传/预览/下载 | 外部受控文件库是批准的文件真相源；系统仅登记引用和元数据，未来成为唯一文件库时再补 |
| 单一主船名航次 | 当前简单业务接受；多航段成为实际需求后再建立 booking/segment |
| 列表页部分子资源抽屉不消费订单锁状态 | 后端会安全拒绝，但按钮仍可见，用户提交后才得到锁错误；详情页已 fail-closed |
| 多个存量订单子资源查询/事务前预检使用 `r.data.db` 而不是 `Data.client(ctx)` | 暂未证明数据错误，但偏离当前共享事务规范并增加未来组合风险 |
| 六种组合缺统一全链路验收矩阵 | 局部领域/PG/组件证据较强；补全后能提升发布信心，但不是批准边界下的业务阻断 |
| 若干单证业务值为自由文本 | 可能造成口径不一致，后续结合公司主数据治理 |

### 15. 正式使用边界

当前可以在以下边界内正式使用：

- 每一步由有权限人员判断完成后，在 UI 点击对应流程按钮；系统保存的是人工确认事实，不自动验证外部运输事实。
- “已放单”是批准的主流程终点；没有活动异常、没有未处理费用后可以人工结案。
- FCL、LCL、BREAK_BULK 与 DIRECT、HOUSE 按现有模型使用；复杂组合的自动化证据强度仍有差异。
- 共享 MBL、多 HBL、箱货分配、改单、作废、Switch 和订单锁按系统现有门禁使用。
- 共享成本由人员在线下计算后逐票录入；系统只负责录入后的费用、账单、发票、核销和提成，不声称自动分摊。
- 关键文件保存在现有受控外部文件库；系统附件只登记引用和元数据，不声称自身是完整文件库。
- 外部订舱、报关、SI、VGM、舱单等继续在外部渠道办理，系统不承担外部提交或自动回执。
- SE 不使用列表页旧分单入口；ReleasePod 不关联旧 HBL；箱/货物不通过故障删除入口删除；同一票拆票仅由一人操作，直至对应 P1 修复。

在上述人工责任和操作边界内，海运出口可以正式使用。超出边界的自动化、精细化跟踪与文件/分摊能力应另行立项，不能反向描述为当前系统已具备。

## Caveats / Not Found

- 本研究已完成正式静态审计、全量 Go 测试/vet、全量前端测试/类型检查及关键真实 PostgreSQL 验证；没有连接生产环境，也没有使用真实业务数据演练。
- 没有把缺少船公司、海关、港区、SI/VGM/舱单外部提交接口列为缺陷；流程按钮记录人工确认，不代表外部事实被自动验证。
- 未找到真实 Sea MBL/HBL 的正常 `CONFIRMED` / `RELEASED` 写入端点；用户已确认不启用逐张单证状态，UI 仍应避免展示冲突语义。
- 未找到订单附件的上传准备、上传完成确认、预签名下载或预览端点；当前批准方案由外部受控文件库承担真相源，不能声称订单附件已具备这些能力。
- 已执行既有共享单证并发无死锁与组织谓词相关集成路径，但未执行独立的权限越权渗透测试或浏览器 E2E；报告中的“已具备”仍以现有自动化与可追踪代码证据为边界。
- FCL/LCL/BREAK_BULK × DIRECT/HOUSE 的业务合法性可能还受公司具体产品规则限制；当前判断只说明模型与领域规则可表达，不替业务确认哪些组合允许销售和执行。
- 共享费用自动分摊、多航段、复杂节点和系统内完整文件库均为当前范围外增强；若未来产品边界改变，需要重新审计，不能直接沿用本次“可以正式使用”的结论。
