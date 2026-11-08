# 研究：海运退关恢复原主单与原航程

- 查询：恢复订单如何在不复活历史 Link 的前提下重建原主单/原航程活动关系；如何处理已写成 ACTIVE 却无 Link 的现场订单。
- 范围：仓库内部，只读业务代码，仅写本任务研究文档。
- 日期：2026-11-09（Asia/Shanghai）。

## 结论

用户已选择本期恢复原主单和原航程。最薄闭环是一次显式恢复命令：先确定本次退关的精确 ENDED Link，再在同一事务中校验原 MBL/TE/单证结构及共享成员资格、创建一条新的 ACTIVE Link、将 Order 从 TERMINATED 改为 ACTIVE、递增订单版本并写生命周期与审计。任何条件失败保持原退关状态，不允许先返回成功再由详情发现缺 Link。

历史 ENDED Link 继续不可变；重建关系不等于回滚主单/航程/单证正文到退关时刻，不自动克隆或复活 VOIDED MBL/HBL。若原主单或原航程已发生变化，必须向用户明确当前关联对象及资料差异或明确阻断，不能在恢复后静默覆盖订单私有资料或旧审核快照。

现有接口没有可直接复用的海运“恢复关联”命令，改配接口也要求已有 ACTIVE Link，因此无法只从页面换调现有改配接口解决。空运 Link 方法可参考其显式重建关系和原子状态流转模式，但不能跨领域直接复用。

## 现行断点与约束证据

| 文件位置 | 证据 |
| --- | --- |
| `server/internal/data/order_write_lifecycle.go:67-122` | 恢复事务只更新 Order；第91-92行在 ACTIVE 时清除 terminated_at，未创建 SE Link；第87-88行对空运反而阻断普通恢复并要求原主单显式关联。 |
| `server/internal/biz/order_transition.go:147-167` | 现有恢复接口校验合法终止状态转移；ACTIVE→ACTIVE非法，不能把既有异常状态修复悄悄塞为普通幂等分支。 |
| `server/internal/data/sea_document.go:29-69` | 只读关联 helper：TERMINATED 通过当前 terminated_at + ENDED + 订单退关精确定位；非终态只读 ACTIVE。该 helper 是读取定位，不适合作为不加锁的恢复写入保证。 |
| `server/internal/data/order_lifecycle_gate_integration_test.go:369-381` | 旧回归主动断言恢复后状态 ACTIVE、历史 ENDED 不变，但未断言新 ACTIVE Link 或写后详情。新实现须把断言更新为“旧行不变且新行建立”，而非复活旧行。 |
| `server/internal/data/sea_order_change_reassignment.go:25-48,72-78,333-377` | 预览和执行改配都依赖 ACTIVE Link，且订单须 ACTIVE 与普通内容可写；TERMINATED 或 ACTIVE无Link 不能用现行改配修复。 |
| `server/internal/data/order_write_sea_master_bill.go:275-289` | 整单更新缺活动 Link 时可走创建关联逻辑，但正常详情无法加载；该入口是资料保存，不应被用作隐藏的恢复兼容或校正工具。 |
| `server/internal/data/air_export.go:431-486` | 空运显式 Link：检查订单TERMINATED/OPEN、原分单与主单身份及版本，创建新Link与订单恢复在一事务，旧行不复活。仅供设计参考。 |

## 目标对象与单证校验

### 原关联精确确定

普通 TERMINATED 恢复使用同组织、同订单、status=ENDED、ended_reason=订单退关、ended_at=Order.terminated_at 的唯一关系。没有或多于一条必须拒绝，不能按 created_at/ended_at 最后一条静默取值。提交应绑定 `expected_order_version` 以及用户实际确认的原关联身份；若预览展示 MBL/TE 资料，则还应绑定对应版本并在锁内校验，避免用户确认后共享航程被他人修改。

新 ACTIVE Link 应保存原 MasterBillID、TransportExecutionID、DocumentStructure，使用新ID、新started_at、version=1；旧Link的ended_at/reason/version均不改。原Link版本和身份需在锁后重验。

### MBL、HBL、共享成员

- `ent/schema/sea_master_bill.go:23-25` 与 `sea_house_bill.go:27-29`：MBL/HBL具有 DRAFT/CONFIRMED/RELEASED/VOIDED 状态和工作版本。原对象 VOIDED 时拒绝恢复，不自动改回草稿、不复活分单、不自动创建替代对象。
- `order_write_sea_master_bill.go:86-94`：现有加票拒绝 VOIDED MBL，并要求目标身份与所选船公司一致。
- `order_write_sea_master_bill.go:109-124`、`sea-export-document-contract.md:108-112`：DIRECT独占主单；存在活动成员时只能全员HOUSE。恢复DIRECT时若原MBL已有其他活动成员，必须拒绝；恢复HOUSE时若其他成员为DIRECT，必须拒绝。
- HOUSE恢复必须仍存在同组织、同订单、同原MBL且非VOIDED的唯一HBL。DIRECT恢复必须没有非VOIDED的本票HBL；不通过模式自动切换修复结构。模式切换规则见 `sea_document_change_mode_change.go:88-95,122-158`。
- 批次分单号唯一覆盖含作废行，`sea_master_bill.go:262-279`；恢复原HBL不新建分单或改号，排重时应排除同一真实HBL自身，不错误把自身历史占号当冲突。
- 原MBL当前存在其他活动成员时，需要按已批准策略校验成员业务锁/生命周期。当前共享写入规范要求任何成员被审核锁、终止中、终止均阻断共享变更；恢复加入关系会改变成员集合，应明确采用同样的严格门禁，避免绕过共享审核边界。
- 当前生命周期恢复允许自身审核锁存在（`order_write_lifecycle.go:76-85`），但新重关联是否允许自身锁仍有效是产品/审计决策：若锁快照已指向旧Link/原航程版本，不能仅凭“恢复是生命周期”默认为新运输关系也已审核。建议本期要求目标订单未审核锁或在发现原审核快照不一致时明确阻断，用户选择后同步旧生命周期规范。

### 航程与资料版本

`SeaTransportExecution`无独立生命周期状态，仅组织、船公司、港口、船名航次、ETD/ETA、current_version_id/version（`ent/schema/sea_transport_execution.go:17-29`）。原TE必须存在于同公司，且船公司同MBL；恢复引用原TE ID，不能任取同MBL“最新航次”。

TE本身没有MBL外键，其与MBL的有效配对由历史Link确立。对于原TE已无其他活动Link但确有本次退关历史的场景，不能套用普通加票的“至少一个活动Link使用该TE”前置：`order_write_sea_master_bill.go:96-107`会阻断最后一票退关后恢复。恢复应以经验证的本次结束关系为配对依据。

恢复新关系不需要修改MBL/HBL正文，因此无需为恢复伪造改单/审核版本。若采纳当前TE变化而写回Order的船公司/港口/日期投影，应明确只同步共享运输字段，不能覆盖本票最终目的港/输入；相关业务口径见 `sea-export-document-contract.md:129-135` 及 `sea-shared-write-consistency.md`。若本期不做差异采纳UI，建议对显著运输身份/资料冲突明确拒绝，而不是静默写回。

## 锁序与原子性建议

这是建议设计，不是现有可直接调用的一体化恢复函数，需主会话在最终方案中明确。

1. 无锁只定位原ENDED Link与目标MBL当前活动成员ID，合并待恢复订单ID后按UUID排序。
2. 在统一 `Data.WithTx` / 共享事务中按UUID排序 `FOR UPDATE` 锁定完整Order集合；不能先锁目标Order再追补更小ID的其他成员。
3. 锁原MBL，再按ID排序锁原ENDED Link与当前成员Link，接着原TE及本票HBL；始终在同组织谓词内。
4. MBL锁后重读当前活动成员集合，与锁前定位集合精确一致；变化返回409，禁止锁后补取新成员Order。
5. 锁内检查订单版本/状态、唯一无ACTIVE、原Link身份状态版本、本次退关时间、MBL/TE/HBL结构及目标/成员门禁，再创建新Link、更新订单/审计。
6. 唯一索引 `idx_sea_mbl_order_links_active_order` 对每票一个ACTIVE兜底；驱动约束错误映射业务冲突。

可复用模式：

- `order_write_draft.go:35-122` 预读成员→按ID锁Order；`:125-161` MBL锁后重验成员；`:165-179` 共享成员锁定门禁。
- `sea_document_change_shared.go:238-260` 在MBL锁后排序锁Link并重验成员集合；该函数要求本票当前ACTIVE，因此恢复应复用思路/小函数，不直接调用后伪造activeLinkID。
- `order_write_sea_master_bill.go:275-345` 固定Order→MBL→Link→TE锁序及身份重验。
- `sea_order_change_reassignment.go:628-646` 保留旧ENDED行并新建ACTIVE Link的写法。

## 已恢复 ACTIVE 无 Link 的现场处理

### 无法从当前Order直接取得本次退关时间

旧恢复第92行已清除 `terminated_at`。`ent/schema/order_lifecycle_event.go:28-39` 保存不可变 from_status/to_status/action/changed_at，reference_id可空，但不保存期望/结果订单版本，也未由旧恢复填Link引用。因此“最近一条ENDED Link”不是可靠恢复依据。

### 推荐最薄处理

普通恢复修复只处理 TERMINATED→ACTIVE；对于已损坏现场，增加本期一次性受控修复命令（或明确同一恢复命令的独立repair模式），必须显式输入 `orderID + expectedOrderVersion + endedLinkID + reason`，只用于满足严格历史证据的具体订单，不在GET/详情加载/普通保存中自动执行。

关联历史证据须同时满足：

1. 当前SE、OPEN、ACTIVE、无ACTIVE Link，用户有目标公司的相应业务授权。
2. 完整按 `changed_at` 与稳定ID排序读取TERMINATION事件；最新真实事件为TERMINATED→ACTIVE，不能只过滤出最后一条恢复事件而忽略后续退关。
3. 该恢复之前紧邻的终止维度事件为TERMINATING→TERMINATED（或仓库实际允许的合法完成退关边），两者时间顺序及状态链连续；同时间且无法唯一判定则拒绝。
4. 指定ENDED Link同组织同订单、ended_reason=订单退关，ended_at精确等于该完成退关事件changed_at，且唯一；同一时刻的旧改配或非本次结束原因不得混入。
5. 核对恢复之后没有新Link、没有后续替换运输关联等冲突事实；其他普通资料编辑可能存在，应以当前expectedVersion锁内校验并展示差异，不假定当前version=恢复时version（旧事件并不保存这个值）。
6. 复用普通恢复的对象/成员/锁序校验，创建新ACTIVE Link，递增Order版本，记录独立repair审计和新Link引用；不伪造又一次TERMINATED→ACTIVE事件、不回写旧生命周期事件、不把Order先回退TERMINATED再调用恢复。

如果这些证据不能唯一确定目标，停止并要求人工明确目标，不能回退到最后一条历史Link。一次性工具比为了一个现场问题给正常业务接口增加隐式ACTIVE→ACTIVE兼容分支更薄、更可审阅；是否给列表/错误页展示“完成恢复关联”能力由主会话收敛，不应顺便构建通用数据修复平台。

## 用户/主会话仍需明确的决策

- 已确认：本期恢复原主单和原航程，不开放重新选MBL/TE。
- 原MBL/TE已更新时，是显式展示并采纳当前运输资料，还是本期冲突即阻断；不建议悄悄恢复退关前快照覆盖共享资料。
- 目标订单自身或原MBL其他活动成员已有审核锁时，是否要求先解锁；现行生命周期恢复允许自身锁，重建新关系后审核快照语义必须明确。
- 已ACTIVE无Link现场采用一次性受控修复工具还是显式业务修复模式；应将输入目标与历史证据写入最终方案，不能自动迁移扫库修复。

## 相关规范与外部资料

- `.trellis/spec/server/backend/sea-export-document-contract.md`：主单/航程/单证结构、批次全员HOUSE、锁序。
- `.trellis/spec/server/backend/sea-shared-write-consistency.md`：关联已有主单不写正文、共享写入成员门禁、改配共享箱占用。
- `.trellis/spec/server/backend/order-lock-and-document-version.md`：现行生命周期恢复、审核锁和快照；需要随新恢复契约同步旧“只恢复状态”约定。
- `.trellis/spec/server/backend/sea-terminated-order-read.md`：精确本次退关关系和恢复后非终态必须ACTIVE。
- `.trellis/spec/server/backend/database-guidelines.md`：统一事务、固定排序锁和唯一索引。
- 本次未使用外部资料，未引入第三方选型；全部结论来自当前仓库。

## 限制与未验证项

- 未访问或改写任何业务数据库；现场SE2026110800002的ACTIVE/version7/无Link事实由主会话提供。
- 未执行测试、未修改业务代码/规范/配置、未执行Git操作。
- 上述锁序及修复接口是建议，需最终规划确认后实现，不应作为已存在能力引用。
