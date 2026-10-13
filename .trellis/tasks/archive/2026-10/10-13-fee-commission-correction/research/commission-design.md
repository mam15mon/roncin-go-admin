# 研究：费用更正的提成重算、执行分离与来源额度

- 研究问题：成本补录 1,000→100 如何联动原 CONFIRMED/PAID 提成及已确认/已执行冲减；连续补录封顶、首次计提、应收优惠、部分回款与更换结算对象如何避免重复计提或恢复。
- 范围：仓库内部
- 日期：2026-10-13
- 状态：用户已确认按真实已收金额重算，Q1 定稿为25元；全额冲回+新费已选定。前文保留研究过程，权威实施算法见文末「9. 提成确定算法」。

## 研究结论

### 1. 已有实现与规格

| 文件与位置 | 当前行为 / 设计意义 |
| --- | --- |
| `.trellis/spec/server/backend/order-fee-review.md` | 整票批准冻结普通费用；补录采用授权父锁→Order；待审阻止建账、相关提成确认/发放；CLOSED 后受控补录仍允许。 |
| `.trellis/spec/server/backend/finance-commission-lock.md` | 原提成与 CONFIRMED/PAID 有符号调整参与权益；PAID 永不修改；支付证据与经济权益分开。 |
| `.trellis/spec/server/backend/operating-company-commission-attribution.md` | 原员工/岗位归属快照不可随当前人员改动重拍。 |
| `server/internal/biz/finance_commission_lifecycle.go:12`、`:27` | 当前 V3：成本=已实现收入×总成本/总应收，逐步骤 Round(8)；毛利小于零按零计提；收入模式不受成本影响。 |
| `server/internal/biz/finance_commission_types.go:28`、`:64` | 计算版本 CUSTOMER_REALIZED_PROFIT_V3；费用 JSON 明细只有费用原折本币，没有计算聚合基数、经济修订版本或费用级来源贡献。 |
| `server/internal/data/finance_commission_calculation.go:344`、`:359` | 核销/对冲分摊金额÷整张账单总额，乘账单各行本位币快照得到订单已实现收入。这是明确按比例分配的计提口径，与自动锁单禁止推断单票结清不是同一件事。 |
| 同文件 `:402`、`:523`、`:551` | 分母优先使用有效账单行本位币，未建账费用使用费用快照；固化历史总应收/总应付 READY+NATIVE。更正后不能继续把原金额账单覆盖更正后的有效经济值。 |
| `server/internal/biz/order_fee_supplement_usecase.go:419`、`:545` | 审批原子创建费用、系统 CONFIRMED 冲减与新整票批准；目前只算新增成本的 before-after、订单行/父单双层封顶。 |
| `server/internal/data/order_fee_supplement_approval.go:97`、`:143` | 历史快照没包含的补录费用加入成本基线；仅靠 FeeID 检查无法识别原 FeeID 的后续修改及应收优惠。 |
| 同文件 `:265`、`:277` | 系统来源 LOCKED_FEE_SUPPLEMENT，唯一键与幂等键按补录申请+父提成+订单；原冲减只允许 DECREASE。 |
| `server/internal/data/order_fee_supplement_mapping.go:188` | 所有来源的 CONFIRMED/PAID 调整一起计算有效余额；DRAFT 减提占用建议预算。更正时必须另算“费用系统累计净额”，不能把人工调整当费用影响抹平。 |
| `server/internal/data/finance_commission_adjustment.go:207`、`:247`、`:318`、`:339` | 通用状态流转锁完整订单集合→父提成→调整；减提确认双层余额；独立执行要求父单 PAID 与真实付款余额；系统补录禁止通用取消。 |
| `server/internal/data/finance_commission_payment.go:19`、`:93`、`:117` | commissionCashExposure 基于 paid_amount/payment_snapshot 及 SEPARATE 已执行调整，OFFSET_WITH_PARENT 不再扣第二次；待审与支付以 Order 串行。 |
| `server/internal/biz/finance_commission_payment.go:35` | CONFIRMED 父单净额支付，将全部 CONFIRMED 增减一起执行；不容许父单未付已有独立执行事实。 |
| `server/internal/data/finance_verification.go:389`、`:460`、`:478` | 原来源撤销：取消未付父单或按实际现金敞口追回；PAID 父单不动，有待执行增提时先阻断。更正的新来源必须进入这个一致性检查。 |
| `server/internal/data/finance_commission_write.go:99`、`:141`、`:221` | 单笔确认与月度批准重算指纹；不能只修改单笔预览而漏掉批量/月度调用。 |
| `server/internal/data/finance_commission_application_candidates.go:67`、`:89` | 候选发现直接按 ACTIVE 应收核销或 CONFIRMED RECEIVABLE 对冲发现；新调整账单若仍是 RECEIVABLE，须从经济来源层排除冲回/继承性质。 |
| `server/internal/data/ent/schema/finance_commission_line.go` | 原行不可变，已包含原分母/率/口径/金额，缺修订与来源贡献。 |
| `server/internal/data/ent/schema/finance_commission_adjustment.go` | 执行状态实际叫 PAID，用户表达 EXECUTED 对应 PAID；现有执行方式足够支持净重建，无需新增虚假现金的执行方式。 |
| `server/api/finance/v1/settlement.proto:939`、`:946` | 提成明细与调整 DTO；调整尚未暴露补录来源申请 ID，需要同步新来源详情与更正关联。 |
| `web/src/pages/finance/commissions/types.ts:130`、`components/CommissionDetailDrawer.tsx:171`、`components/PendingDecreasePanel.tsx:275` | 来源标签、操作禁用与待冲减面板都识别旧补录类型；新费用更正可增可减，不能仍只显示“冲减”。 |

### 2. 原规则重算与有效经济快照

每条原提成订单行使用原 employee/personnel role、calculation_version、calculation_basis、rate_percent、历史计算聚合基数。禁止查当前方案比例替代。计算器按版本路由：已存在 V3 用原 Round(8) 步骤；未知版本/不可验证原快照返回明确错误，不猜测。

令该行原金额为 C0，原经济总应收 R0、总成本 K0。审批日志对经济内容产生明确的有符号 revenue_delta/cost_delta；每笔有经济修订序号。新总额从原快照加该行检查点之后的有效经济变化得到：R=R0+ΣΔR，K=K0+ΣΔK。首次生成提成直接取当前有效经济内容，并把该时点修订序号固定进原行；之后更正只应用更晚事件。这样补录发生在首次计提前时，不会首次计提已经包含成本、之后再被补录系统重复冲减。

费用聚合必须明确记录 `calculation_base_amount`：可能是账单日折本币，也可能是未建账费用自身折本币，不能从旧 JSON 的 `BaseCurrencyAmount` 推断。费用更正事件保存变更前后该基数及币种；不能用当前汇率重新解释已付快照，也不能把一笔费用折本币差额直接加到另一种本位币的提成行。

后续采用“全额冲回原债权债务 + 替代账单”的会计模型时，利润与提成读取有效经济费用及修订；财务反向账单属于历史清理，不能因收付方向倒转被计为新增营业收入或成本。替代费用沿用原经济根 `economic_fee_root_id`，经济变动必须只入一次。

### 3. 应收优惠、已实现收入与来源额度（用户待答）

全收案例有确定底线：原应收 1,000、成本 100、原核销 1,000、利润率提成 10%=90。应收优惠到 900 后，总可实现收入不能仍是 1,000，正确全部经济目标为 (900−100)×10%=80。仅改分母900却保留 realized1000 将算88.88888889，错误。

用户已确认：原 R=1,000/K=400/收500/10%=30，优惠后 R=800，按实际已收500与新总应收比例重新分摊成本，目标25。实现中不得冻结原来源覆盖比例；以下初稿配额候选已由文末的单一清偿投影和稳定来源占额规则取代。

若用户选择实际回款口径，候选规则如下：

1. 每项应收经济根 f 关联真实来源贡献 `cashContribution(f,s)`：原收款核销、真实商业对冲按原账单行快照比例确定归属；更正冲回对冲 purpose=CORRECTION_SETTLEMENT 明确排除。原回款在替代账单继承仅转移归属引用，不生成第二份真实贡献。
2. 实际退款以关联原经济根的真实执行记录扣减对应贡献；批准退款不写现金执行。核销反转移除真实贡献，对冲反转同理。成本追回即使财务表现为 RECEIVABLE 也不是主营应收，不进入提成收入贡献。
3. 对根 f，有效可实现经济额度 Qf=min(当前有效经济应收Rf,max(0,所有有效来源净真实贡献之和))；额度≤有效应收，退款债务在优惠审批时已通过 Rf 下调经济上限，实际退款再扣现金，不能重复减收入。
4. 多来源按有效净贡献比例分配 Qf（固定 s 类型+来源 UUID 排序处理8位尾差，最后一项接剩余量，确保精确守恒）。同一来源多个费用/账单先合并经济根再分配；不可按提成人或是否已计提分配，否则同人不同岗位和未申请提成会改变收入。
5. 行的新有效已实现收入 r=Σ该来源在相关根上的配额，本次目标 T=V3(r,R,K,原率/原口径)。`r=0` 合法目标0，不调用当前强制正 realized 的公共函数；`R=0,r=0` 同理，不除零。

推荐这条路线的成本必须正视：真实新收款、退款、核销/对冲撤销可能改变多个既有来源的经济配额，因此不能只在费用批准时重算。来源集合变化时要重算该经济根关联的全部既有有效提成及候选，旧已付原行保持不可变、通过调整承接差额；否则先优惠、后补收或退款仍会漂移。固定原来源覆盖比例路线可以更薄，但部分回款目标不同，且替代账单的后续增量仍须去重。

更换结算对象不能把旧客户实收自动变为新客户实收：会计替代与 CORRECTION_SETTLEMENT 仅平历史，实际退款/新客户回款按各自现金事件处理。提成经济根与客户债权归属两者都要显式记录；是否在旧收款尚未退时继续视为这项服务的已实现收入，是同一“实际净回款配额”规则的结果，不能隐式把替代账单对冲当新回款。设计必须展示此期间的金额例子。

### 4. 费用系统目标与手工调整分离的具体算法

旧冲减可能因双层余额与 DRAFT 预留被封顶，不能反向恢复“原费用差额×比例”，也不能直接以理论 before-after 的全部正差创建增提。

每条受影响原行分别汇总：

- C0：不可变原提成金额；T：按上节经济内容、有效来源额度和原规则计算的新理论金额。
- F_paid/F_pending：仅 LOCKED_FEE_SUPPLEMENT 与新增费用更正/经济配额联动来源的已执行/未执行有符号系统调整。PAID减提为负，PAID增提为正。CANCELLED/DRAFT不计权益。
- M：MANUAL 的 CONFIRMED/PAID有符号净额，始终保留。核销/对冲反转另行处理来源失效，不当作费用系统影响。
- D：DRAFT减提预留，只参与新冲减的预算，不当作已确认权益。

无其他余额干预时费用系统目标 A*=T−C0，当前费用系统累计 F=F_paid+F_pending。变化是 A*−F，既能增也能减。特别是已有部分被封顶的冲减，只按实际 F 对账，不恢复从未扣掉的部分。

为沿用双层非负保护，A*<0 时先在锁内算不含本行可替换费用系统调整的预算：

```text
lineBudget = C0 + 本行非费用系统CONFIRMED/PAID调整 - 本行DRAFT减提
parentBudget = 父原金额 + 所有非费用系统有效调整
               + 其他订单行费用系统有效调整 - 父DRAFT减提
cap = max(0, min(lineBudget, parentBudget))
A* = -min(C0-T, cap)  // 只在 T<C0 时使用
```

本行费用系统已执行金额仍保留，只从预算求目标时拿出来单独对账，不能真的删。T≥C0 时 A*=T−C0，不主动抵销人工奖罚。审计记录 nominal target、capped target、理论未能冲减余额、人工净额和DRAFT预留。

例：C0=100、手工减80，错误补成本使理论目标0，系统最多减20。更正后理论目标90：A*=-10，已实际系统减20，只恢复10，最终权益100−80−20+10=10；若直接“理论恢复90”会得到90并抹掉人工惩罚。

源已经反核销/反对冲：该来源经济配额为0，不得因为仍有 PAID 父单就恢复其旧补录冲减。先按既有反转契约处理该来源，再对仍有效其他来源做额度联动；新核销来源用新原行/修订快照生成，不继承旧父单的原金额。须避免反转与费用系统重算各自再减同一份现金。

### 5. 以专用取消未执行系统调整 + 净重建减少虚假往返

首选候选：在一次费用更正批准事务中，针对同一原行所有**未执行的费用来源系统调整**专用作废并建立更正关联，然后按 A*−F_paid 创建一笔 CONFIRMED 净调整。原补录请求仍 APPROVED；原调整保留原确认记录、金额和来源，增加 cancelled_at/by/reason + superseded_by_correction/impact。禁止通过通用取消端点作废。PAID及原 payment_snapshot 一律不动；已执行要真实反向补付/追回。

若目标A*与当前费用系统F一致，不需要取消重建，只保存此次无提成变化影响记录。幂等与事务保证反复批准只生成一次净单据；取消原待执行系统记录和创建新净记录必须同时成功。

| 原事实（R=1000，原K=0，10%，C0=100；误补K1000后改K100） | 更正后处理 |
| --- | --- |
| 父CONFIRMED，系统减100 CONFIRMED未执行 | 专用取消减100；建减10 CONFIRMED；父净发90。 |
| 父PAID100，系统减100 CONFIRMED未追回 | 专用取消减100；建减10 CONFIRMED；待财务真实追回10，不先追回100再补90。 |
| 父PAID100，系统减100 PAID/SEPARATE已追回 | 减100保持；建增90 CONFIRMED；财务实际补付90后才PAID。 |
| 父以净额0支付，减100 PAID/OFFSET_WITH_PARENT | paid_amount=0不动；减100不动；建增90，实际补付90。 |

这复用既有 OFFSET_WITH_PARENT/SEPARATE，不引入“抵扣即伪造现金”的执行方式。新增更正来源不可普通取消，也不可手工确认其已系统确认记录。未付父单按净额发放；已付父单按已有执行入口处理新净额。取消重建后的原冲减不能被现有“关联任一冲减就挡补录撤销”的查询当成重新开放原补录直接删除的许可，保留费用更正为唯一经济变动出口。

### 6. 最薄数据契约与接口增量

以会计/费用研究选择的经济更正模型为基础，提成侧至少需要：

1. 原行新增 `fee_content_version_snapshot`；fee_snapshot 明细新增 `economic_fee_root_id`、`economic_revision`、`calculation_base_amount`及聚合来源；新增只读 `realization_snapshot`保存来源分摊ID、原账单行ID、经济根、真实贡献、配额、版本/指纹。保留原R/K、原率/口径/计算版本，不新建另一套方案表。
2. 更正批准的不可变 `commission_impact_snapshot`：commission/line/order/employee标识，原快照检查点，before/after R/K/r，原规则，T、A*、F_paid/F_pending、预算/封顶余量、取消调整IDs、新调整ID、源证据哈希。可在更正申请批准快照 JSON 固化，不必仅为展示新建一张提成计算表。
3. 调整 source_type 新增 `FEE_CORRECTION`，必须有 source_fee_correction_request_id；专用取消旧系统调整增加 `superseded_by_fee_correction_request_id`（或 approved impact ID）。新唯一键 correction+commission+order；幂等键 `fc:<request>:<commission>:<order>`。零影响仍有批准快照，不插 amount=0调整。
4. 如果采用实际回款额度，现金来源变化触发的重算不是同一费用批准，应单独使用明确的 `ECONOMIC_REALLOCATION`来源+核销/对冲/退款执行事件关联及事件版本，唯一键 event+commission+order。只用FEE_CORRECTION的一次性唯一键无法容纳后续回款重分配。具体事件承载可复用费用经济事实记录，不能藏在重复写原更正申请里。
5. 财务 full reversal/replacement 账单链须保存 economic root/更正申请关联；对冲 purpose 增 CORRECTION_SETTLEMENT，并在所有提成来源查询排除该目的；真实旧回款的继承/退款的净贡献可从链路确定时，优先计算并保存不可变证据，不建通用分摊平台。若现有链不能确定到费用根，最薄新增费用根来源贡献表（root+source allocation唯一）作为明确事实，不能在运行时猜。
6. 预览/批准 API返回受影响提成的 before/after/net difference、待发/待追回及封顶提示；批准请求带 preview_token + expectedVersion。调整来源详情提供更正申请、原补录/费用、作废与替代链、原规则/原支付证据；员工本人专属来源查看也应扩展到新来源，仍严格 employee_id=currentUser。

Schema必需正式迁移、Ent生成；Proto/OpenAPI/客户端同组生成。没有当前比例兜底，没有通过原表自动猜历史缺失。历史快照不足时明确不可计算；项目上线前开发样例一次性同步新格式，是否迁移既有开发行仅用可验证原账单/分摊事实确定性补齐。

### 7. 事务、锁序与具体调用方

费用更正批准：实时授权父 User→Membership→Role（共享锁）→所有受影响 Order（按UUID一次性锁集合）→更正申请FOR UPDATE/expectedVersion→相关财务来源/账单（沿财务契约）→提成父单按UUID→调整按UUID。持有Order之后禁止申请员工User锁；原规则直接取不可变快照，不必为更正重锁/重拍员工。父单的其他订单金额仅只读预算，若审批需要更改其配额也必须在进入事务前定位并一次性锁完整Order联合集合，不得持有A再补锁B。

建立费用经济变化、专用作废未执行系统调整、净调整、整票批准更新、申请批准快照、审计和通知在同一 biz.Transactor.WithinTransaction。批准预览外无锁，批准锁内重算证据与token；补录、手工减提、支付或来源变动使token变化时409，不静默缩金额。审批与来源变动重复回调使用确定性事件幂等键；失败任何环节均回滚。

待审费用更正加入既有门禁：`order_fee_review.go`建账准入/审核摘要，`order_write_lifecycle.go`结案，`finance_commission_payment.go:94`提成确认/执行，`finance_commission_write.go`单笔/批量/月度批准，`order_auto_lock.go`自动锁单。结案后仍可新受控申请，待审不等于反结案。

经济/来源引擎必须同步：`finance_commission_calculation.go`预览/创建/确认、`finance_commission_application_candidates.go`月申候选、`finance_commission_application_submit.go`提交/重提、`finance_commission_application_decision.go`批准、工作台预计提成与 `finance_commission_order_summary.go`、`finance_commission_ledger.go`；不能只修批准补录的 computeSuggestions。支付与 `settlement.go:44/:70`财务锁继续按所有有效有符号权益投影，新source必须纳入同一公式。

来源动作：`finance_verification.go`创建/反核销、`finance_netting.go`确认/反对冲，关联退款/追回现金实际确认入口应在同事务稳定真实贡献与触发重算。先整体锁受影响 Order集合，再锁来源，避免现有 Order→来源锁序反转。核销系统反转与新系统重算须一个计划统一计算，不能两条helper各扣一次。

### 8. 数值回归清单（实现后验证；本研究未运行）

| 场景 | 应验证结果 |
| --- | --- |
| R1000/K0/r1000/率10；补1000→100，四种支付状态 | 上节四行均目标90；只按真实已执行额生成待补/待追回，原付款不变。 |
| r500、R1000、K0；成本补1000→100 | 理论原50→0→45，恢复依据有效系统净额；不能恢复90。 |
| 原R1000/K0/r1000，连续补成本1200再补100，更正1200→100 | 总成本1300→200，总提成0→80；只实际累计扣100，恢复80，不能按1100×10%=110恢复；再更正100→50，总成本150，目标85，仅再恢复5。 |
| 首次计提已含K1000，原行金额0，之后K改100 | 直接按该行原R/r/率生成增90；不把旧补录再加一次，首计提前无父单时也无空壳调整。 |
| 原K已更正100后新来源收500 | 新原行用当前K100且修订检查点固定；以后补50只对新变化算影响，不重复旧900修正。 |
| C0=100/手工-80/系统实际-20，目标从0→90 | 系统仅恢复10，人工-80保留；DRAFT减提仅作为预算。 |
| 原R1000/K100/全收1000/10%，优惠R900 | 目标80而非88.88888889；真正退款100执行前后目标仍80，无额外收入核销/重复减提。 |
| 原R1000/K400/收500/10%，优惠R800 | 用户已确认实际回款口径25。 |
| 两个真实来源各收500，优惠900 | 总有效来源收入900，原规则各10%时总80；配额450+450，尾差守恒，按各原率计算。 |
| 原账单冲回+替代+CORRECTION_SETTLEMENT | 更正清算不产生新提成候选；旧真实来源仅继承一次；后续真新收款才增加配额。 |
| 原收款反核销后更正成本 | 不恢复反核销来源的提成；其他有效来源配额按选定规则重算，真实追回不双减。 |
| 父单有其他订单正余额 | 不转扣/转恢复其他订单；父级预算与订单行预算都守恒。 |
| 不同原率/同人兼岗/后续方案变更 | 每条按其原快照分别算；当前方案不会影响更正。 |
| 原金额零、有效总应收零、成本超过收入、微小舍入边界 | 允许目标零，不除零、不负提成；V3舍入与尾差口径一致。 |
| 更正批准vs支付/补录/来源反转/手工确认；相同批准重复；中途失败 | 固定锁序；过期token或版本拒绝；没有重复调整、部分费用写入或付款历史变更。 |

## 外部资料

本题仅研究仓库内业务与数据合同，无新增第三方依赖，无需外部技术资料；以上金额是算法验收例子，不是数据库实测。

## 相关规范

`.trellis/spec/server/backend/order-fee-review.md`、`finance-commission-lock.md`、`operating-company-commission-attribution.md`、`commission-scheme-assignment.md`、`.trellis/spec/adr/0014-monthly-commission-application.md`；实施后须更新它们关于新来源、待审门禁、原规则重算及未执行调整专用作废的合同。

## 限制与未决项

- Q1已确认25，最终算法必须覆盖已收500、优惠后R800/K400；应收优惠用例属于实施范围。
- 实际回款配额路线覆盖更广，但必须包括真实资金事件与来源反转的联动；只在费用批准算一次不足以闭环。经济配额跨来源的分摊规则是候选技术实现，应随用户口径写入design，不是现有代码合同。
- 已付原行仅有逐订单realized收入；没有原费用级贡献和经济根、修订快照。新字段若历史不可可信还原必须明确UNAVAILABLE，不能用当前账单/当前汇率兜底。
- 现有数据库没有 EXECUTED 状态，执行完成为 PAID；无新状态必要。
- 本研究只读代码并写本文件；没有改产品/规格、运行测试或提交Git。

## 9. 提成确定算法（Q1确认后的权威实施合同）

本章取代前文来源按全部贡献比例分额度的候选。用户已确认真实已收金额口径：R1000/K400/实收500/10%，优惠至R800后目标25。沿用全额经济冲回+新费用重立；不冻结原收入覆盖比例。

### 9.1 唯一账单行清偿投影

财务与提成共用 `biz.FinanceBillLineSettlement`，由会计worker实现；提成不另做静态原账单比例分摊。

```go
type FinanceBillLineSettlement struct {
    BillID, BillLineID, FeeID uuid.UUID
    Direction OrderFeeDirection // 资金方向
    TotalAmount, SettledAmount, RemainingAmount decimal.Decimal // 账单原币
    Contributions []FinanceSettlementContribution
}
type FinanceSettlementContribution struct {
    SourceType string // VERIFICATION / NETTING
    SourceID, AllocationID uuid.UUID
    EffectiveAt time.Time
    Amount decimal.Decimal // 本allocation归属本账单行的原币金额
    IsCorrectionSettlement bool
    ReversedAt *time.Time
}
```

实现对全部原发生和反转事件稳定重放：核销发生用created_at，商业/内部对冲发生用confirmed_at（不能用草稿allocation.created_at）；同一时刻按事件类型、来源UUID、allocation UUID排序，反转使用reversed_at作为后续释放事件。普通发生按**当时账单行剩余金额**比例分摊，经济更正对冲按source_fee_id定点清偿；原allocation→line归属在重放内存中固定，反转准确释放原分配，不使后来的既有事件重新分配。保留已REVERSED来源原发生及反转时间，不能查询时直接删掉所有历史事件。尾差采用会计统一的RoundBank(8)+稳定末行补差，禁止提成自己另建另一舍入口径。

原A1000+B200合并账收400，投影先分A333.33333333、B66.66666667；A更正100后内部对冲定点清偿旧A剩666.66666667。随后旧账收133.33333333全部归B，不能再次按1000:200分到已清偿A。历史400分摊不改。

更正旧账尚未完成必要定点清偿时，拒绝在该旧账新增普通核销，允许先登记未核销资金；完成后普通分摊只分给仍有合法剩余的普通经济行。此保护由会计投影负责，提成只消费已通过财务准入的行归属。

### 9.2 经济根、对象和币种桶

沿正常/替代费的source_fee_id链确定 `EconomicRootID`；链首正常费是根，冲回行不独立建根。新增费用是新根。链必须同Order、同经济方向且无环；Root查找结果可批量缓存，不新建通用根实体或事件平台。

经济应收根f的当前叶子给出 `CurrentFeeID/SettlementPartyID/Currency/Amount/BaseCurrency/CalculationBaseAmount`。桶键为(root,settlementParty,currency)，当前叶子匹配的桶才有当前应收经济额度，其余历史桶上限0。换对象/币种不能自动把旧对象旧币现金当新对象新币回款；旧退款与新真实回款分别执行。成本根无收入额度，追回成本即使账单资金方向RECEIVABLE也不入计提收入。

`CalculationBaseAmount`是明确的有效经济计提基数：普通原费用沿用其有效正常账单行的冻结本位币基数，未建账时费用快照；受控补录/更正生成的新经济内容使用提交/批准固化的经济基数，不因后来资金账单汇率改变而二次重算已批准费用影响。当前费用经济helper及初次计提必须统一这条规则。原行仍保存原R/K/率/计算版本；不能用当前市场汇率修改历史快照。

### 9.3 真实贡献、退款与配额

只从9.1行归属构造真实贡献，不从账单头Amount重新猜。

1. 正贡献：有效正常营业应收行上的真实应收核销，或purpose=NORMAL商业对冲的应收行归属。每份记录(source kind/id,allocation id,bill line id,root,party,currency,effectiveAt,native amount)。旧原费用虽已被替代，其真实正贡献仍跟随同根、同对象、同币桶保留。
2. 内部purpose=CORRECTION_SETTLEMENT全部贡献恒0；旧→反向→新清偿不生成第二份正贡献，不消耗或增加真实贡献。继承通过原根查询已有来源，不复制新source。
3. 退款贡献：营业应收根对应CORRECTION_REVERSAL应付行上的真实PAYABLE核销，或者NORMAL商业对冲清偿该应付行。按9.1归属追到同根/对象/币种；冲回成本的收款不当退款，也不当收入。
4. 真实正贡献、退款发生及它们的反转按同一稳定时间规则重放。每次退款在该桶中已有的未退正贡献之间按余额比例分配，使用同一8位尾差函数；缓存退款allocation→原正贡献扣减份额，退款反转恢复原份额，不把后来的新回款分给更早退款。先退款后新收款时只新收款增加新可用份额。
5. 原正来源反转移除该源原贡献；若尚有依赖它的有效退款，使该桶任一真实来源净余额负数或总退款超过真实入账，则拒绝本次反转并要求先反转关联退款核销。不制造负回款、不静默挪给别对象/币种。反转发生与拒绝都不受订单业务锁/CLOSED影响。
6. 当前根原币可计提收入 `Q=min(CurrentAmount,max(0,Σ同桶净真实贡献))`。对净贡献按(原发生EffectiveAt, SourceType, SourceID, AllocationID, BillLineID)升序依次占额：`q_i=min(net_i,remainingQ)`，扣减remainingQ，来源份额是其所有q_i之和。先来的真实来源先占额，过量后来的来源份额0；客户多付不能让旧提成被挪到新超额来源。
7. 原币q按当前有效经济快照折本币：`sourceBase=q/CurrentAmount*CalculationBaseAmount`；按稳定贡献顺序分摊根的整体base目标，统一尾差确保 ΣsourceBase=Q/CurrentAmount*CalculationBaseAmount。CurrentAmount=0时所有配额/base=0，禁止除零。跨币种只在各原币桶内分配，最后在同一组织本位币汇总，不把USD与CNY原币裸加。

例如两来源各收500，应收优惠至900：按来源先后分配500、400，总实现900；同为10%且成本100，总提成80。原收1000优惠至900，再多收100：旧源仍900，新超额源0。原R1000/K400/收500优惠R800：q仍500，成本分摊250，利润250，提成25。优惠审批降低经济上限；退款实际执行扣净贡献，已全收1000→优惠900→退款100时实现额始终900，不重复减少。

### 9.4 初次计提与已存在父提成

对每个真实商业来源、员工与岗位，r为9.3给该source在所属订单全部应收根的base配额。R/K取当前有效经济费用的统一基数。计算仍按原V3逐步骤Round(8)，原CONFIRMED/PAID使用各自原rate/basis/employee/role，不读当前方案；尚未计提才按既有来源日期方案解析。`r=0`或`R=0`的目标直接0；亏损按零；收入模式目标r×原率，不受K影响。

新来源r=0不生成空壳提成候选；r>0而利润0可沿既有零金额提成规则。初次行固化经济版本、R/K、配额证据，后续补录/更正从当前经济投影计算目标，不再把全部历史补录累加到已经包含补录的R/K中。

现有DRAFT只允许通过显式重新生成得到新原快照：经济指纹改变后不得确认旧金额；月申仍按已提交快照拒绝批准，先退回后显式重提并重新生成未确认来源。原CONFIRMED/PAID行不可原地重算，使用下节调整。

### 9.5 一次净调整计划

复用本文件4/5节公式，但费用系统集合统一包括LOCKED_FEE_SUPPLEMENT、FEE_CORRECTION、ECONOMIC_REALLOCATION。有效源的新理论T来自9.4；C0是原行。人工M独立保留，负目标按订单行和父单预算封顶，DRAFT减提仅预留预算。目标系统累计A*与**已实际系统累计净额**对账，绝不恢复此前未扣掉的封顶额。

审批或真实来源变化若目标发生变化：专用取消本行所有未执行系统调整（保留确认/取消历史与本动作来源），不动PAID调整；新CONFIRMED调整金额=abs(A*−F_paid)，符号按差额。相同目标与当前系统累计一致时零写调整，只保留审批/动作证据。父CONFIRMED净发放，父PAID真实补付/追回。

**真实来源反转是同一个计划，不能再执行旧reconcileCommissionsForSourceReversal之后重复重算：** 被反转源未付父单，取消其未执行调整与父单，保留原来源与月申历史；被反转源PAID父单保持PAID，真实目标0，取消其所有未执行调整，按原行付款证据 `lineCash=C0+Σ该行PAID有符号调整` 创建待追回lineCash（0不创建）。此处来源失效同时失去该来源未执行人工加减资格；人工已实际执行也进入真实现金追回，但不改历史。校验各行lineCash≥0且ΣlineCash等于commissionCashExposure；不猜缺付款证据。其他仍有效源因配额变化，继续按原率和人工分离公式重算。统一锁内计划计算全部旧/新系统调整，既有反转helper改为调用此计划，避免两套追回叠加。

退款等导致有效源份额变0而源未反转：父单仍存在且原人工调整保留，按费用系统目标处理，不把它误当来源反转的整单取消。

### 9.6 共享事务与幂等触发

不新建事件平台。现有核销Create/Reverse、商业/内部对冲Confirm/Reverse在自己的共享事务中完成资金分摊，再直接调用同一 `ReconcileCommissionEconomics(txCtx, org, actor, orderIDs, trigger)`；来源反转事件先写active/reversed_at，再在提交前按新的完整投影生成一次计划，失败整体回滚。CORRECTION_SETTLEMENT通常目标不变，但仍复核投影；真实退款PAYABLE核销也会触发，不仅RECEIVABLE核销。

```go
type CommissionEconomicTrigger struct {
    Kind string // FEE_CORRECTION / VERIFICATION_CREATE / VERIFICATION_REVERSE /
                // NETTING_CONFIRM / NETTING_REVERSE
    ID uuid.UUID
    Version uint64 // 已完成该动作后的实体version
}
type CommissionSourceQuota struct {
    SourceType string
    SourceID, AllocationID, BillLineID, EconomicRootID uuid.UUID
    SettlementPartyID uuid.UUID
    Currency, BaseCurrency string
    EffectiveAt time.Time
    NativeNetAmount, NativeQuota, BaseQuota decimal.Decimal
}
type CommissionEconomicOrderProjection struct {
    OrderID uuid.UUID
    FeeContentVersion uint64
    BaseCurrency string
    Receivable, Payable decimal.Decimal
    Quotas []CommissionSourceQuota
    Fingerprint string
}
```

核心已定义的 `FeeCorrectionCommissionRepo.PreviewCorrectionImpact(ctx,org,order,request,currentFees,projectedFees)` 返回逐行target/current/difference及Fingerprint；`ReconcileCorrectionImpact(ctx,org,order,actor,request,impact)` 在共享事务锁内重算后执行净计划。提成worker给普通补录一个同算法adapter，不再保留旧仅成本边际算法。内部方法建议 `LoadCommissionEconomicProjection`（调用唯一行清偿投影）、`PlanCommissionEconomicReconciliation`（纯Go）、`ApplyCommissionEconomicReconciliation`（Ent/审计）。

新调整FEE_CORRECTION关联申请；ECONOMIC_REALLOCATION关联触发的source_verification_id或source_netting_id，含真实退款PAYABLE核销。幂等键为sha256(kind+id+version+commissionID+orderID)加短前缀，不溢出128字符；来源字段数据库CHECK与真实类型对应。普通动作重试先复用既有幂等资金事实，不能重复执行改变目标；最终调整本身唯一键兜底。旧“parent/order/source_verification_id唯一”仅对原反核销来源适用，新事件可发生create与reverse不同版本，必须修改为按新幂等键去重。

锁序：需员工资格快照的初次生成/确认先User；费用审批先授权User→Membership→Role。随后先解析受影响全部Order，包括本资金事件账单的订单及同根来源涉及的父单所有订单，按UUID一次FOR UPDATE；再申请（费用更正时）、Cashflow（核销时）、当前来源、Bill按UUID、BillLine/费用按UUID、Commission父按UUID、CommissionLine按UUID、Adjustment按UUID。不得在持Bill/Cashflow/Commission后追加Order，也不得在持Order后为重算原快照再锁员工User。现有核销Create的Cashflow→Bill入口必须移到先Order；NettingConfirm不能先锁来源再回头锁Order。确认提成已有Order→来源→Bill路径沿用相同序。授权与经营范围锁内重读，所有repo经Data.client(txCtx)。

使用Order锁稳定整个根及共享父单目标，不需新增可写配额状态/version镜像。预读定位与锁内源集合不一致即409重新开始，不能增量追加未排序订单。费用、请求、整票批准、调整作废/新建、来源状态、审计/outbox在同一事务，失败不留半笔资金或半笔提成。

### 9.7 最薄持久字段和实施回归

沿用core新增字段：adjustment的source_fee_correction_request_id/source_netting_id/superseded_by_fee_correction_request_id及FEE_CORRECTION、ECONOMIC_REALLOCATION；line的fee_content_version_snapshot、realization_snapshot JSON。realization_snapshot保存本次9.3全部参与配额证据与经济根/币种/原币/base/排序/指纹；原JSON不用于猜当前费用版本。更正申请commission_impact_snapshot固化计划及封顶。真实清偿投影用既有allocation、confirmed/created/reversed_at重放，不新建通用事件或分摊表。

正式迁移从原费用/账单/有效与反转分摊确定性生成经济根和原证据；只生成可证明数据，缺失则明确UNAVAILABLE及原因码，禁运行时“缺字段按当前比例猜”。开发样例同步新字段，不重置库。初次计提/工作台/月申所有调用共用新投影和原规则解析，来源候选不把内部对冲、成本追回当新提成。

必须验收Q1=25、全收优惠90→80、共享账A更正后后续收款仅归B、退款及退款反转恢复、原来源有退款依赖时反转拒绝、两来源额度500+400、后续超额源0、换对象/币种旧贡献不跨桶、连续补录跨零/封顶恢复仅实际差、CONFIRMED/PAID/OFFSET_WITH_PARENT四种执行证据、手工-80不抹除、源反转单一追回、来源回款后旧父单净调整、同刻稳定顺序/8位尾差及共享事务竞态。
