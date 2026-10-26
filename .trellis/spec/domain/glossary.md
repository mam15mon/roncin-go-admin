# 领域术语表（货代 + 财务 + 平台）

> AI 会话遇到陌生业务词先查这里；术语后的代码定位是唯一真相源。
> 与实体/schema 不一致的口径一律以代码为准并回改本表。

## 海运运输域

| 术语 | 英文/代码 | 含义与要点 | 代码入口 |
| --- | --- | --- | --- |
| 母单 | MBL, `SeaMasterBill` | 海运主提单，多张订单可共享同一 MBL | `data/ent/schema/sea_master_bill.go` |
| 分单 | HBL, `SeaHouseBill` | 货代出具的房单；订单单证结构为 `HOUSE` 时存在，`DIRECT`（直单）时无 HBL | `sea_house_bill.go` |
| 运输执行 | `SeaTransportExecution` | MBL 下的实际航次（船名/航次/泊位/ETD/ETA），独立版本化，一个 MBL 多航次需改配明确选择 | `sea_transport_execution.go` |
| 母单-订单关联 | `SeaMasterBillOrderLink` | 订单挂到 MBL 的活动链接；订单状态流转/退关会结束链接 | `sea_master_bill_order_link.go` |
| 单证版本 | `SeaMasterBillVersion` 等 | 每次修订生成不可变版本快照（换单历史），含外部确认信息 | `sea_*_version.go` |
| 共享箱 | `SeaSharedContainer` / `...Allocation` | 跨订单共用的集装箱；箱货分配必须件重尺守恒 | `sea_shared_container*.go` |
| 货物分配 | `SeaCargoAllocation` | 货物项在各订单/结果票间的件数/毛重/体积切分 | `sea_cargo_allocation.go` |
| 拆票 | Split, `SeaOrderSplitEvent/Result` | 一张订单拆为多票（原票+新票），独占箱整箱归属、货物件重尺零误差守恒、未建账费用整行转移；幂等键+请求指纹防重 | `data/sea_order_change_split*.go` |
| 改配 | Reassignment | 订单换入其他 MBL/新建 MBL；产生内嵌外部确认要求 | `sea_order_change_reassignment.go` |
| 单证变更 | `SeaDocumentChange` | 修订(amendment)/作废(void)/模式切换(mode change) 三类，全部先预览后执行 | `data/sea_document_change_*.go` |
| 船司 | `ShippingLine` | 船公司主数据（含集装箱前缀规则） | `shipping_line.go` |

## 空运运输域

| 术语 | 英文/代码 | 含义与要点 | 代码入口 |
| --- | --- | --- | --- |
| 主运单 | MAWB, `AirMasterWaybill` | 同公司多票共享航司、机场、航班与运输时刻；显式关联、版本控制和历史 | `data/ent/schema/air_master_waybill.go` |
| 分运单 | HAWB, `AirHouseWaybill` | 每票一份，号码同主单永久占号且跨主单可重复；内容保留不可变版本 | `air_house_waybill.go`、`air_house_waybill_version.go` |
| 空运活动关联 | `AirMasterWaybillOrderLink` | 一票同时一个 ACTIVE 关系；终止结束，恢复重新创建 Link | `air_master_waybill_order_link.go` |
| 计费重确认 | `AirWeightConfirmation` | 本票/主单和委托/实际来源分别确认，保存规则/货量指纹与采用值；输入变化保留旧值并标过期 | `biz/air_weight.go` |
| 体积重/泡差/分泡 | Volumetric/Bubble Difference/Shared Bubble | kg/m³ 归一后按明确除数计算，三者分别表达；参考值与人工采用值分开 | `.trellis/spec/server/backend/air-export-contract.md` |

## 财务域

| 术语 | 英文/代码 | 含义与要点 | 代码入口 |
| --- | --- | --- | --- |
| 费用 | `OrderFee` | 订单应收/应付费用；财务状态由活动账单行及未取消账单推导（含 DRAFT）。普通修改/删除要求订单未审核锁定、ACTIVE（OPEN/CLOSED 均可）、费用当前有效且未建账；已建账先按权限回退。行能力分别返回 can_edit/can_delete/can_bill/can_correct | `data/order_fee.go`、`data/fee_bill_association.go` |
| 费用审核 | `OrderLockRecord` / `OrderFeeReviewEvent` | 与订单资料合为一次人工审核锁，固化费用版本、指纹与利润快照；旧独立审核状态/写接口退役，历史事件只读；建账无需先审核 | `biz/order_lock.go`、`data/order_lock_apply.go` |
| 费用补录补审 | `OrderFeeCorrectionRequest` | 锁后 ADD/REPLACE/VOID，覆盖新增、修改、调减、作废，批准才影响正式费用；有账单先回退；待审只占用源费建账，单票至多一份待审；具备同公司品类真实 lock grant 者补审 | `data/order_fee_correction*.go` |
| 账单 | `FinanceBill` / `...Batch` / `...Line` | 费用汇成账单；支持单张与批量分组建账（分组策略/对冲模式）；状态机 含版本乐观锁 | `biz/finance_bill*.go` |
| 对冲 | Netting, `FinanceNetting` / `...Allocation` | 同结算单位应收应付互抵；对冲配对在建账预览中生成 | `finance_netting*.go` |
| 核销 | Verification, `FinanceVerification` / `...Allocation` | 收付款登记后按分配核销到账单/费用 | `finance_verification*.go` |
| 资金流水 | `FinanceCashflow` | 收付款流水记录 | `finance_cashflow.go` |
| 发票 | `FinanceInvoice` / `...Bill` / `...Line` | 发票及其与账单行关联 | `finance_invoice*.go` |
| 提成方案 | `FinanceCommissionRule` / `...Assignment` | 按员工+角色分配的提成规则（区间不可重叠，生效/草稿/停用） | `biz/finance_commission_rules.go` |
| 提成 | `FinanceCommission` / `...Line` | 计提引擎按核销/对冲来源生成的提成（金额、CNY 快照、来源指纹）；状态 DRAFT→CONFIRMED→PAID，可 Cancel | `data/finance_commission_calculation.go` |
| 提成调整 | `FinanceCommissionAdjustment` | 手工奖励/处罚与费用变化引起的系统差额；普通费用和补审均事务内重算，保留已付事实及手工调整；未付父单发放时合并已确认调整净额，已付父单的后续调整独立执行 | `data/fee_correction_commission.go`、`finance_commission_adjustment.go` |
| 月度提成申请 | `FinanceCommissionApplication` / `...Line` | 员工按完整自然月汇总申请全部未申请合格提成；财务整单批准/驳回（不做部分批准）；驳回原单显式重提（Resubmit） | `biz/finance_commission_application.go` |
| 提成归属 | `OrderCommissionAttribution` | 订单经营归属（销售/操作/客服三岗）锚点，计提依据；创建时按订单人员分工固化，客户档案责任人仅作表单带入默认值 | 计提源解析见 `finance_commission_calculation.go`，契约见 `server/backend/operating-company-commission-attribution.md` |
| 结算单位 | Settlement Party | 费用/账单的往来结算对象（Partner） | `partner.go` |

## 往来单位域（Partner）

| 术语 | 代码 | 含义 |
| --- | --- | --- |
| 往来单位档案 | `Partner` / `...Profile` | 客户/供应商/承运人等外部主体 |
| 信用 | `PartnerCredit` | 应收账期与信用额度，建账时校验 |
| 结算规则 | `PartnerSettlementRule` | 币种/账期等结算约定 |
| 快捷预设 | `PartnerShippingPreset` | 出货预设（港口/船司等快捷填充） |
| 角色黑名单 | `partner_role.go` | 限制某些合作角色 |

## 订单与平台域

| 术语 | 代码 | 含义与要点 |
| --- | --- | --- |
| 订单 | `Order` | 状态机：草稿→确认→…→退关(Termination)/关账(Closure)；`version` 乐观锁；写路径范本 `data/order_write_draft.go` |
| 订单审核锁 | `OrderLockRecord` / `OrderUnlockRequest` | 唯一人工锁，约束业务内容与普通费用写入；授权锁角色直解、管理员应急、普通编辑人审批解锁；待审更正阻断解锁；结算不自动锁单 | `data/order_lock_*.go` |
| 组织/数据范围 | `Organization` / `Membership` / `Role` / `RoleAssignment` / `Permission` | 多组织 + 角色授权；角色范围配置已退役，有效边界由工作台推导（公司本公司、系统治理）；权限清单唯一真相源 `internal/access/manifest.go` | `biz/auth.go` |
| 审计 | `AuditLog` | 业务审计事件（action 非空约束），独立于运行日志 | `audit_log.go` |
| 编号规则 | `NumberRule` / `NumberSequence` | 业务单号生成（如提单号、申请号） | `number_rule.go` |
| 汇率 | `ExchangeRateSetting` / `...ImportBatch` | 币种汇率设置与批量导入；提成/账单 CNY 折算依赖 | `exchange_rate*.go` |
| 企业资源 | `EnterpriseResource` 系列 | 客户资源聚合（地址/影像/关联/备注/出货文本） | `enterprise_resource*.go` |
| 钉钉集成 | `dingtalk_*` | 通讯录同步、待分配开通与业务审批流转 | `biz/dingtalk_*.go` |
| 主数据 | Currency/Port/Airport/Airline/AdministrativeRegion | 全局引用数据；选择器搜索覆盖拼音 | `seeds/` 初始化 |

## 通用机制（贯穿全域）

| 机制 | 约定 | 范本 |
| --- | --- | --- |
| 乐观锁 | `expected_version` 不匹配返回 409「已被更新，请刷新后重试」 | `order_write_draft.go` |
| 悲观锁 | 多行加锁先按主键排序再 `ForUpdate`，固定锁序防死锁 | `order_write_draft.go` |
| 幂等 | 写操作带 `idempotencyKey` + `requestFingerprint`，重复请求回放原结果 | `order_lock_apply.go`、拆票 |
| 共享事务 | `biz.Transactor.WithinTransaction`，仓储经 `Data.client(ctx)` 取客户端 | `data/transaction.go` |
| 不可变历史 | 单证版本/变更事件/拆票事件只追加不修改 | `sea_*_version.go` |
