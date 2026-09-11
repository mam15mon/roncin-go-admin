# 系统核心业务矛盾与安全加固 PRD

## 1. 任务定位

本任务处理正式上线前已经由当前代码证实的跨层正确性问题，覆盖订单生命周期、海运提单成员关系、财务结清口径、核销候选、发票状态机、权限范围与前端访问判断。

本任务不是一次泛化重构。目标是在保留现有领域模型和已批准业务契约的前提下，修复会造成越权、状态死锁、候选数据丢失或财务状态错误的最薄业务闭环。

优先级：**P1（上线阻断级正确性与安全问题）**。

## 2. 当前代码事实与问题边界

### 2.1 订单生命周期

- `server/internal/data/order_lock.go` 的 `ensureOrderBusinessEditable` 目前只校验业务锁，未校验 `termination_status` 和 `closure_status`；箱、货、费用、附件、人员、里程碑、异常、海运单证等多个写入口复用该门禁。
- `server/internal/data/order_release_pod.go` 另有一套仅检查业务锁的重复门禁，存在规则漂移。
- `TransitionTermination` 和 `TransitionClosure` 也调用了内容编辑门禁。若直接把生命周期状态塞进现有函数，会错误阻断合法的 `TERMINATING → ACTIVE/TERMINATED`、`TERMINATED → ACTIVE`、`CLOSED → OPEN` 流转。
- `DOCUMENT_RELEASED` 订单通常已经业务锁定，当前 `TransitionClosure` 因此无法结案。
- SE 订单进入 `TERMINATED` 后，活动 `SeaMasterBillOrderLink` 未结束；所有仅依据 Link `ACTIVE` 判断“当前成员”的查询都会继续把退关订单视为在配载中。
- `ListConsolidationSummaries` 仍预加载旧 `ShippingDocuments`，而现行海运分单真相源已经是 `SeaHouseBill`。

### 2.2 财务结清与候选

- `server/internal/data/settlement.go` 的台账 SQL 筛选与行投影都只累计有效资金核销分摊，未累计有效对冲分摊。
- `GetFeeLedgerOrderDetail` 没有复用列表的账单与金融进度投影，导致详情中的 `bill_no`、`financial_progress`、`finance_locked` 不完整。
- `ListCreationCandidates` 先各取最多 200 条账单和资金流水，再在 Go 内过滤余额；当前 200 条均已结清时，后面的未结清候选会消失。
- 账单仓储已经具备 `OnlyUnsettled` 谓词；资金流水仓储尚无等价的内部过滤条件。

### 2.3 发票状态机与并发

- 当前已批准业务契约区分“草稿取消”“已开票作废”和“已开票红冲”：系统不判断税期，只通过危险操作提示让操作者确认线下税务条件。
- `Cancel` 当前只拒绝已经 `CANCELLED` 的发票，因此 `RED_FLUSHED → CANCELLED` 仍可能被接受。
- `Cancel` 与 `RedFlush` 只锁发票及活动关联行，未锁定并推进关联 `FinanceBill.version`，账单侧的并发操作无法感知发票关系已经释放。
- 前端对已开票作废沿用普通“取消”提示，没有明确说明其线下税务适用条件。

### 2.4 权限与前端契约

- 订单锁资格查询以角色代码硬编码排除 `administrator`。即使管理员角色实际持有对应订单类型的 lock 权限，也无法锁单、直接解锁或成为钉钉审批候选。
- 钉钉回调会再次调用同一资格函数；只改锁单入口而不改候选快照和回调复核，会产生 `APPROVER_NOT_QUALIFIED`。
- bootstrap admin 没有普通组织角色成员关系，但具备应急管理语义；不应伪造为普通钉钉审批候选。
- `ListOrganizationRoles` 与外部成员授权接口均为 `DATA_SCOPE_ALL`，服务端会接受显式目标组织。把角色列表接口单独降到组织级既不能让分支管理员完成授权，也会造成目标组织 ID 越权风险。
- 用户页无条件请求全组织列表，未按 `canReadOrganizations` 收敛数据源。
- `scopedFinancePermissionWrites` 遗漏 netting 权限，跨组织附加访问在网关层无法按目标组织解析。

## 3. 已确定的产品与安全决策

### 3.1 内容写入和生命周期命令分离

- “业务内容可编辑”统一要求：订单未业务锁定、`termination_status = ACTIVE`、`closure_status = OPEN`。
- 内容门禁只用于业务字段及子资源写入，不用于终止、恢复、结案、反结案等生命周期命令。
- 生命周期命令在 Order `FOR UPDATE` 后独立校验版本、来源状态、目标状态和业务前置条件，保留现有合法状态图。
- 对 `TERMINATING` 返回“订单已进入终止流程，不允许修改业务数据”；对 `TERMINATED` 返回“订单已终止，不允许修改业务数据”；对 `CLOSED` 返回“订单已结案，不允许修改业务数据”。错误均为稳定 409 领域错误。

### 3.2 退关与海运提单关系

- `ACTIVE → TERMINATING` 不结束 Link，避免尚未最终确认的退关申请提前脱离配载。
- `TERMINATING → TERMINATED` 在同一事务中结束该订单的活动 Link：状态改为 `ENDED`，写入 `ended_at`、`ended_reason` 并递增 Link 版本。
- `TERMINATING → ACTIVE` 保留原 Link。
- `TERMINATED → ACTIVE` 不静默恢复历史 Link。订单恢复为活动后，若需要继续 SE 业务，必须通过显式提单关联/改配流程建立新的活动 Link；历史 Link 保持不可变历史。
- 当前成员查询同时以 Link `ACTIVE` 和 Order `termination_status = ACTIVE` 为准，避免旧数据或竞争窗口污染汇总。

### 3.3 发票终态

- `DRAFT → CANCELLED`：取消草稿。
- `ISSUED → CANCELLED`：作废已开票发票，保留现有能力，但前端必须明确提示“仅在已于线下税控/开票系统完成或满足作废条件时操作”。
- `ISSUED → RED_FLUSHED`：红冲。
- `CANCELLED`、`RED_FLUSHED` 均为终态；不得互转或重复执行。
- 本任务不接入税控平台，不自动判断税期，不创建兼容分支。

### 3.4 管理员与审批资格

- 普通 `administrator` 不再因角色代码被排除；是否合格只取决于用户/成员关系/角色是否有效、角色是否持有目标业务类型 lock 权限，以及该权限的数据范围是否覆盖目标订单组织。
- `ORGANIZATION_TREE` 和 `ALL` 按真实组织树范围计算，不要求角色归属组织必须等于订单组织。
- bootstrap admin 可执行锁单和现有应急直接解锁，但不进入普通钉钉审批候选池；它没有可供候选快照保存的常规成员关系/角色事实。
- 候选生成、当前调用人判断、锁单命令、直接解锁和钉钉回调必须复用同一资格口径。

### 3.5 组织角色读取

- 保持 `ListOrganizationRoles` 和外部成员授权的 `DATA_SCOPE_ALL`，不降低服务端权限。
- 全局管理员继续使用跨组织接口。
- 普通组织管理员只使用当前组织的 `ListRoles` 和当前组织信息；前端不得为其调用 `ListOrganizations` 或 `ListOrganizationRoles`。
- 本任务不把外部成员跨组织授权开放给普通组织管理员；这需要独立的目标组织写授权设计。

## 4. 功能需求与验收标准

### ORD-01 统一业务内容写门禁

- 所有现有订单业务资料及子资源写入口，在 Order `FOR UPDATE` 后统一执行内容门禁。
- 至少覆盖：订单草稿字段、标签、异常、附件、货物、集装箱、费用、里程碑、人员、放货 POD、旧 shipping document、SeaDocument/SeaCargoAllocation/SeaOrderChange 的实际业务写入。
- 移除 `ensureReleasePodOrderEditable` 的重复规则。
- 验收：`TERMINATING`、`TERMINATED`、`CLOSED` 和已业务锁定四类场景分别返回稳定且准确的领域错误；正常 `ACTIVE + OPEN + unlocked` 不受影响。

### ORD-02 生命周期命令不被内容门禁误杀

- `TransitionStatus` 仍仅允许 `ACTIVE + OPEN`，但通过生命周期专属校验表达。
- `TransitionTermination` 保留 `ACTIVE → TERMINATING → TERMINATED`、`TERMINATING → ACTIVE`、`TERMINATED → ACTIVE`。
- `TransitionClosure` 允许已锁定的 `DOCUMENT_RELEASED` 订单结案，也允许符合现有状态图的反结案；结案 readiness 仍在事务内重验。
- 验收：锁定后的放单订单可结案；合法恢复流转成功；非法来源状态或旧版本返回现有稳定冲突错误。

### ORD-03 退关结束活动 Link

- 最终进入 `TERMINATED` 时，同事务结束 SE 活动 Link；非 SE 无 Link 操作。
- 结束 Link 与更新订单、生命周期事件、审计任一步失败时全部回滚。
- 恢复订单不复活旧 Link。
- 验收：退关完成后不存在该订单的活动 Link，历史记录包含结束时间与原因；退关取消不影响现有 Link；并发旧版本不能产生双活动 Link 或部分提交。

### ORD-04 自拼汇总使用当前模型

- `ListConsolidationSummaries` 只统计 Link `ACTIVE` 且 Order `ACTIVE` 的 LCL 成员。
- house numbers 从当前 `SeaHouseBill` 读取，不再依赖 `ShippingDocuments`。
- 验收：退关订单不计入件重尺/成员数；活动 HBL 号正确返回；已结束或非当前 HBL 不返回。

### FIN-01 台账统一“有效结清金额”

- 有效结清金额定义为：有效核销分摊金额 + 有效对冲分摊金额。
- 有效核销：allocation `active = true` 且 verification `status = ACTIVE`。
- 有效对冲：allocation `active = true` 且 netting `status = CONFIRMED`。
- 列表筛选 SQL、列表行投影和订单详情投影必须使用同一口径。
- 验收：纯核销、纯对冲、二者混合、反核销/反对冲四组数据的进度与筛选结果一致；不能因重复关联导致金额倍增。

### FIN-02 台账订单详情完整

- `GetFeeLedgerOrderDetail` 返回与列表相同的 `bill_no`、`financial_progress`、`finance_locked`。
- 详情不得另写一套会与列表漂移的状态算法，应复用共享投影/转换函数。
- 验收：同一费用在列表和详情的三个字段完全一致。

### FIN-03 核销候选在 LIMIT 前过滤

- 账单候选传入 `OnlyUnsettled: true`。
- 为资金流水内部过滤增加 `OnlyUnverified`（或等价命名），由数据库按“流水金额 > 有效核销分摊总额”过滤。
- 过滤必须发生在分页/`LIMIT 200` 之前；保留公共列表 `page_size <= 200` 约束，不循环翻页、不扩大上限。
- 验收：账单与流水各构造 200 条以上已结清记录，并在其后放置未结清记录；候选接口仍能返回未结清项。

### INV-01 发票状态机闭合

- `Cancel` 只接受 `DRAFT` 或 `ISSUED`；`RedFlush` 只接受 `ISSUED`。
- `CANCELLED` 和 `RED_FLUSHED` 拒绝任何后续取消/红冲。
- 前端对草稿显示“取消”，对已开票显示“作废”，两类确认文案不同；已开票作废包含危险提示和线下税务条件说明。
- 验收：所有允许边和非法边均有 Biz/Data 或集成测试，且 UI 操作与后端一致。

### INV-02 发票释放账单的事务一致性

- Cancel/RedFlush 在一个事务中锁定发票、活动关联行和按 UUID 排序的账单行。
- 释放活动关联后，每张受影响账单的 `version` 恰好递增一次。
- 发票更新、关联释放、账单版本、审计任一步失败时全部回滚。
- 验收：旧账单版本的并发操作失败；并发发票命令只有一个成功；无部分释放。

### ACC-01 订单锁资格按权限与组织范围计算

- 移除对普通 `administrator` 角色代码的排除。
- 当前组织、组织树、全局三种授权范围均按现有权限范围语义计算。
- 资格口径同时用于状态展示、锁单、直接解锁、候选快照与钉钉回调复核。
- bootstrap admin 具备锁单/应急解锁，但不进入普通审批候选。
- 验收：组织业务角色、上级组织树主管、全局 administrator、bootstrap、无权限用户分别符合上述行为；撤销权限后旧候选回调被稳定拒绝。

### ACC-03 用户页按权限选择角色数据源

- 只有具备全局组织读取/外部授权能力时才请求全组织列表及跨组织角色。
- 普通组织管理员的新建/编辑当前组织用户继续使用 `ListRoles`，角色下拉正常加载，不产生预期外 403。
- 验收：组织管理员进入用户页无全组织请求；全局管理员的跨组织成员关系与外部授权流程不回归。

### ACC-04 对冲网关目标组织解析

- 将 netting read/create/confirm/reverse 权限加入财务目标组织解析表，其中 read 为只读，create/confirm/reverse 为写入。
- cancel 继续复用 reverse 权限，不新增权限码。
- 验收：附加组织只读访问可读不可写；可写访问按各命令权限放行；无目标组织授权仍返回 403。

## 5. 非目标

- 不开发尚未进入上线范围的其他业务类型（陆运、铁路、空运等）前端页面。
- 不新增税控平台集成或税期判断。
- 不降低跨组织管理接口的权限范围。
- 不自动恢复退关前的历史 MBL Link。
- 不重做通用权限引擎、财务状态模型或订单状态机。
- 不兼容未知旧行为，不增加双读、双写、静默回退。
- 不清理或重建数据库；本任务预计不需要 Schema 迁移。

## 6. 完成定义

- ORD-01～ORD-04、FIN-01～FIN-03、INV-01～INV-02、ACC-01、ACC-03、ACC-04 全部通过。
- 订单锁规范同步更新，代码、测试、规范不再互相矛盾。
- 定向测试先通过；任务最终验收执行一次风险匹配的完整门禁。
- 没有手改生成文件；若实施中确实改变 Proto 或权限 Manifest，必须按仓库生成流程同步生成物，否则不得提交。
