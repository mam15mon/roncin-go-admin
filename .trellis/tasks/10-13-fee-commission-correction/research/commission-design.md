# 研究：费用更正的提成重算、执行分离与来源额度

- 研究问题：成本补录 1,000→100 如何联动原 CONFIRMED/PAID 提成及已确认/已执行冲减；连续补录封顶、首次计提、应收优惠、部分回款与更换结算对象如何避免重复计提或恢复。
- 范围：仓库内部
- 日期：2026-10-13
- 状态：技术候选设计；部分回款下优惠的计提口径已由主会话向用户提问，尚未得到回答。不得将下面推荐算法视为已批准业务合同。

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

部分收款有两种不同业务结果，原代码不能替用户决定。主会话已提问：原 R=1,000/K=400/收500/10%=30，优惠后 R=800，按实际已收500与新总应收比例重新分摊成本，目标25；保持原来源50%覆盖比例则实现收入400，目标20。推荐前者，更符合“回款是回款”，但必须等待答复。

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
| 原R1000/K400/收500/10%，优惠R800 | 用户待答：实际回款口径25，固定50%覆盖口径20。 |
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

- 唯一最高价值用户问题是部分回款优惠后的计提口径；问题已由主会话提出，当前未答。不得把25或20写成已确认验收，也不能悄悄删掉应收优惠用例。
- 实际回款配额路线覆盖更广，但必须包括真实资金事件与来源反转的联动；只在费用批准算一次不足以闭环。经济配额跨来源的分摊规则是候选技术实现，应随用户口径写入design，不是现有代码合同。
- 已付原行仅有逐订单realized收入；没有原费用级贡献和经济根、修订快照。新字段若历史不可可信还原必须明确UNAVAILABLE，不能用当前账单/当前汇率兜底。
- 现有数据库没有 EXECUTED 状态，执行完成为 PAID；无新状态必要。
- 本研究只读代码并写本文件；没有改产品/规格、运行测试或提交Git。
