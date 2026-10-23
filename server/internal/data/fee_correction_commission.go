package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	"github.com/shopspring/decimal"
)

type feeCorrectionCommissionRepo struct{ data *Data }

func NewFeeCorrectionCommissionRepo(data *Data) biz.FeeCorrectionCommissionRepo {
	return &feeCorrectionCommissionRepo{data: data}
}

type commissionEconomicPlan struct {
	Parent                                              *ent.FinanceCommission
	Line                                                *ent.FinanceCommissionLine
	Rows                                                []*ent.FinanceCommissionAdjustment
	Projection                                          *biz.CommissionEconomicOrderProjection
	Original, Current, Target, Desired, Rebuild, Excess decimal.Decimal
	Changed                                             bool
	CancelIDs                                           []uuid.UUID
}

func commissionSystemSource(s adjustment.SourceType) bool {
	return s == adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT || s == adjustment.SourceTypeFEE_CORRECTION || s == adjustment.SourceTypeECONOMIC_REALLOCATION || s == adjustment.SourceTypeFEE_MUTATION
}
func commissionSignedAdjustment(row *ent.FinanceCommissionAdjustment) (decimal.Decimal, error) {
	v, err := decimalOf(row.Amount)
	if row.Direction == adjustment.DirectionDECREASE {
		v = v.Neg()
	}
	return v, err
}
func planCommissionEconomicRows(ctx context.Context, client *ent.Client, org, orderID uuid.UUID, projection *biz.CommissionEconomicOrderProjection, lock bool) ([]*commissionEconomicPlan, error) {
	lq := client.FinanceCommissionLine.Query().Where(lineent.OrganizationIDEQ(org), lineent.OrderIDEQ(orderID), lineent.HasCommissionWith(commission.StatusIn(commission.StatusCONFIRMED, commission.StatusPAID)))
	lines, err := lq.All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(lines))
	byParent := make(map[uuid.UUID]*ent.FinanceCommissionLine)
	for _, l := range lines {
		ids = append(ids, l.CommissionID)
		byParent[l.CommissionID] = l
	}
	pq := client.FinanceCommission.Query().Where(commission.IDIn(ids...)).Order(commission.ByID())
	if lock {
		pq.ForUpdate()
	}
	parents, err := pq.All(ctx)
	if err != nil {
		return nil, err
	}
	aq := client.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDIn(ids...)).Order(adjustment.ByID())
	if lock {
		aq.ForUpdate()
	}
	rows, err := aq.All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*commissionEconomicPlan, 0, len(parents))
	for _, parent := range parents {
		// 失效来源已由独立真实现金追回计划承接，费用更正不能恢复它。
		if parent.VerificationID != nil {
			source, err := client.FinanceVerification.Get(ctx, *parent.VerificationID)
			if err != nil {
				return nil, err
			}
			if string(source.Status) != "ACTIVE" {
				continue
			}
		} else if parent.NettingID != nil {
			source, err := client.FinanceNetting.Get(ctx, *parent.NettingID)
			if err != nil {
				return nil, err
			}
			if string(source.Status) != "CONFIRMED" {
				continue
			}
		} else {
			return nil, biz.ErrCommissionSource
		}
		l := byParent[parent.ID]
		if l.SnapshotStatus == nil || *l.SnapshotStatus != lineent.SnapshotStatusREADY || l.TotalReceivableSnapshot == nil || l.TotalPayableSnapshot == nil || parent.CalculationVersion != biz.CommissionCalculationVersion {
			return nil, biz.ErrCommissionSnapshotUnavailable
		}
		if l.BaseCurrency != projection.BaseCurrency {
			return nil, biz.ErrCommissionBaseCurrencyMismatch
		}
		original, err := decimalOf(l.CommissionAmount)
		if err != nil {
			return nil, err
		}
		rate, err := decimalOf(l.RatePercent)
		if err != nil {
			return nil, err
		}
		sourceKind := "VERIFICATION"
		sourceID := uuid.Nil
		if parent.VerificationID != nil {
			sourceID = *parent.VerificationID
		} else if parent.NettingID != nil {
			sourceKind = "NETTING"
			sourceID = *parent.NettingID
		}
		target, err := biz.CommissionEconomicTarget(parent.CalculationVersion, biz.CommissionCalculationBasis(l.CalculationBasis), projection.SourceRevenue(sourceKind, sourceID), projection.Receivable, projection.Payable, rate)
		if err != nil {
			return nil, err
		}
		parentBase, err := decimalOf(parent.CommissionAmount)
		if err != nil {
			return nil, err
		}
		plan := &commissionEconomicPlan{Parent: parent, Line: l, Projection: projection, Original: original, Target: target, Current: original}
		paid, pending, lineOther, parentOther, lineDraft, parentDraft := decimal.Zero, decimal.Zero, decimal.Zero, parentBase, decimal.Zero, decimal.Zero
		for _, row := range rows {
			if row.CommissionID != parent.ID {
				continue
			}
			plan.Rows = append(plan.Rows, row)
			value, err := commissionSignedAdjustment(row)
			if err != nil {
				return nil, err
			}
			if row.Status == adjustment.StatusDRAFT && row.Direction == adjustment.DirectionDECREASE {
				parentDraft = parentDraft.Add(value.Abs())
				if row.OrderID == orderID {
					lineDraft = lineDraft.Add(value.Abs())
				}
				continue
			}
			if row.Status != adjustment.StatusCONFIRMED && row.Status != adjustment.StatusPAID {
				continue
			}
			if row.OrderID == orderID {
				plan.Current = plan.Current.Add(value)
			}
			system := commissionSystemSource(row.SourceType)
			if row.OrderID == orderID && system {
				if row.Status == adjustment.StatusPAID {
					paid = paid.Add(value)
				} else {
					pending = pending.Add(value)
					plan.CancelIDs = append(plan.CancelIDs, row.ID)
				}
			} else {
				parentOther = parentOther.Add(value)
				if row.OrderID == orderID {
					lineOther = lineOther.Add(value)
				}
			}
		}
		plan.Desired, plan.Rebuild, plan.Excess = biz.PlanCommissionSystemAdjustment(original, target, paid, pending, lineOther, parentOther, lineDraft, parentDraft)
		plan.Changed = !plan.Desired.Equal(paid.Add(pending))
		if !plan.Changed {
			plan.CancelIDs = nil
		}
		if parent.Status == commission.StatusPAID {
			if _, err := commissionCashExposure(parent, plan.Rows); err != nil {
				return nil, err
			}
		} else {
			for _, row := range plan.Rows {
				if row.Status == adjustment.StatusPAID {
					return nil, biz.ErrCommissionPaymentEvidence
				}
			}
		}
		result = append(result, plan)
	}
	return result, nil
}
func plansImpact(plans []*commissionEconomicPlan) *biz.FeeCorrectionCommissionImpact {
	impact := &biz.FeeCorrectionCommissionImpact{Lines: make([]*biz.FeeCorrectionCommissionLineImpact, 0, len(plans))}
	for _, p := range plans {
		systemNow := decimal.Zero
		for _, a := range p.Rows {
			if a.OrderID == p.Line.OrderID && commissionSystemSource(a.SourceType) && (a.Status == adjustment.StatusCONFIRMED || a.Status == adjustment.StatusPAID) {
				v, _ := commissionSignedAdjustment(a)
				systemNow = systemNow.Add(v)
			}
		}
		difference := p.Desired.Sub(systemNow)
		impact.Lines = append(impact.Lines, &biz.FeeCorrectionCommissionLineImpact{CommissionID: p.Parent.ID, CommissionNo: p.Parent.CommissionNo, OrderID: p.Line.OrderID, EmployeeID: p.Parent.EmployeeID, EmployeeName: p.Parent.EmployeeName, BaseCurrency: p.Parent.BaseCurrency, OriginalAmount: p.Original, CurrentAmount: p.Current, TargetAmount: p.Current.Add(difference), Difference: difference})
	}
	raw, _ := json.Marshal(plans)
	digest := sha256.Sum256(raw)
	impact.Fingerprint = hex.EncodeToString(digest[:])
	return impact
}
func (r *feeCorrectionCommissionRepo) PreviewCorrectionImpact(ctx context.Context, org, orderID uuid.UUID, request *biz.OrderFeeCorrectionRequest, current, projected []*biz.OrderFee) (*biz.FeeCorrectionCommissionImpact, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	projected = correctionProjectionFees(request, projected)
	projection, err := loadCommissionEconomicProjection(ctx, client, orderID, projected)
	if err != nil {
		return nil, err
	}
	_, locked := transactionFromContext(ctx)
	plans, err := planCommissionEconomicRows(ctx, client, org, orderID, projection, locked)
	if err != nil {
		return nil, err
	}
	return plansImpact(plans), nil
}
func correctionProjectionFees(request *biz.OrderFeeCorrectionRequest, fees []*biz.OrderFee) []*biz.OrderFee {
	out := make([]*biz.OrderFee, 0, len(fees))
	for _, f := range fees {
		v := *f
		if request.After != nil && v.ID == request.After.ID {
			v.SourceFeeID = request.SourceFeeID
			v.CorrectionRequestID = &request.ID
			v.EconomicEffect = biz.OrderFeeEconomicIncrease
			v.SupersededByRequestID = nil
		}
		out = append(out, &v)
	}
	return out
}
func (r *feeCorrectionCommissionRepo) ReconcileCorrectionImpact(ctx context.Context, org, orderID, actor uuid.UUID, request *biz.OrderFeeCorrectionRequest, expected *biz.FeeCorrectionCommissionImpact) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	projection, err := loadCommissionEconomicProjection(ctx, client, orderID, nil)
	if err != nil {
		return err
	}
	plans, err := planCommissionEconomicRows(ctx, client, org, orderID, projection, true)
	if err != nil {
		return err
	}
	actual := plansImpact(plans)
	a, _ := json.Marshal(actual.Lines)
	b, _ := json.Marshal(expected.Lines)
	if string(a) != string(b) {
		return biz.ErrFeeCorrectionConflict
	}
	return applyCommissionEconomicPlans(ctx, client, org, actor, biz.CommissionEconomicTrigger{Kind: "FEE_CORRECTION", ID: request.ID, Version: request.Version}, plans)
}
func economicAdjustmentKey(trigger biz.CommissionEconomicTrigger, parent, order uuid.UUID) string {
	raw := fmt.Sprintf("%s:%s:%d:%s:%s", trigger.Kind, trigger.ID, trigger.Version, parent, order)
	digest := sha256.Sum256([]byte(raw))
	return "econ:" + hex.EncodeToString(digest[:])
}
func applyCommissionEconomicPlans(ctx context.Context, client *ent.Client, org, actor uuid.UUID, trigger biz.CommissionEconomicTrigger, plans []*commissionEconomicPlan) error {
	now := time.Now().UTC()
	for _, p := range plans {
		if !p.Changed {
			continue
		}
		if len(p.CancelIDs) > 0 {
			u := client.FinanceCommissionAdjustment.Update().Where(adjustment.IDIn(p.CancelIDs...), adjustment.StatusEQ(adjustment.StatusCONFIRMED)).SetStatus(adjustment.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason("经济费用变化后按净额重建系统调整").AddVersion(1)
			if trigger.Kind == "FEE_CORRECTION" {
				u.SetSupersededByFeeCorrectionRequestID(trigger.ID)
			}
			if _, err := u.Save(ctx); err != nil {
				return err
			}
		}
		if p.Rebuild.IsZero() {
			if err := client.FinanceCommission.UpdateOneID(p.Parent.ID).AddVersion(1).Exec(ctx); err != nil {
				return err
			}
			if err := auditCommissionEconomicPlan(ctx, client, org, actor, trigger, p); err != nil {
				return err
			}
			continue
		}
		sequence := p.Parent.AdjustmentSequence + 1
		direction := adjustment.DirectionINCREASE
		if p.Rebuild.IsNegative() {
			direction = adjustment.DirectionDECREASE
		}
		key := economicAdjustmentKey(trigger, p.Parent.ID, p.Line.OrderID)
		exists, err := client.FinanceCommissionAdjustment.Query().Where(adjustment.OrganizationIDEQ(org), adjustment.IdempotencyKeyEQ(key)).Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			return biz.ErrFeeCorrectionConflict
		}
		builder := client.FinanceCommissionAdjustment.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(org).SetCommissionID(p.Parent.ID).SetOrderID(p.Line.OrderID).SetAdjustmentNo(fmt.Sprintf("%s-ADJ%03d", p.Parent.CommissionNo, sequence)).SetIdempotencyKey(key).SetCommissionNo(p.Parent.CommissionNo).SetOrderNo(p.Line.OrderNo).SetEmployeeID(p.Parent.EmployeeID).SetEmployeeName(p.Parent.EmployeeName).SetDirection(direction).SetStatus(adjustment.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor).SetBaseCurrency(p.Parent.BaseCurrency).SetAmount(p.Rebuild.Abs().StringFixed(8)).SetReason("费用及真实回款变化的系统净额调整").SetVersion(1)
		switch trigger.Kind {
		case "FEE_CORRECTION":
			builder.SetSourceType(adjustment.SourceTypeFEE_CORRECTION).SetSourceFeeCorrectionRequestID(trigger.ID)
		case "FEE_MUTATION":
			builder.SetSourceType(adjustment.SourceTypeFEE_MUTATION)
		case "FEE_SUPPLEMENT":
			builder.SetSourceType(adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT).SetSourceFeeSupplementRequestID(trigger.ID)
		default:
			builder.SetSourceType(adjustment.SourceTypeECONOMIC_REALLOCATION)
			if trigger.Kind == "VERIFICATION_CREATE" || trigger.Kind == "VERIFICATION_REVERSE" {
				builder.SetSourceVerificationID(trigger.ID)
			} else if trigger.Kind == "NETTING_CREATE" || trigger.Kind == "NETTING_REVERSE" {
				builder.SetSourceNettingID(trigger.ID)
			}
		}
		if _, err := builder.Save(ctx); err != nil {
			return err
		}
		if err := client.FinanceCommission.UpdateOneID(p.Parent.ID).SetAdjustmentSequence(sequence).AddVersion(1).Exec(ctx); err != nil {
			return err
		}
		if err := auditCommissionEconomicPlan(ctx, client, org, actor, trigger, p); err != nil {
			return err
		}
	}
	return nil
}

// 零净额重建同样记录取消来源、计算依据与超额，保留完整更正轨迹。
func auditCommissionEconomicPlan(ctx context.Context, client *ent.Client, org, actor uuid.UUID, trigger biz.CommissionEconomicTrigger, p *commissionEconomicPlan) error {
	cancelled, err := json.Marshal(p.CancelIDs)
	if err != nil {
		return err
	}
	details := map[string]string{"trigger.kind": trigger.Kind, "trigger.id": trigger.ID.String(), "order.id": p.Line.OrderID.String(), "original": p.Original.StringFixed(8), "target": p.Target.StringFixed(8), "desired_system": p.Desired.StringFixed(8), "new_adjustment": p.Rebuild.StringFixed(8), "cancelled_adjustments": string(cancelled), "capped_excess": p.Excess.StringFixed(8), "projection": p.Projection.Fingerprint}
	return writeAudit(ctx, client.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "finance.commission.economic_reconcile", Result: "success", ResourceType: "finance_commission", ResourceID: p.Parent.ID.String(), Details: details})
}

var _ biz.FeeCorrectionCommissionRepo = (*feeCorrectionCommissionRepo)(nil)

// reconcileOrdinaryFeeEconomics 将普通费用变更纳入同一经济净额算法，保留已发放事实并生成差额。
// 调用方已按关联提成父单的订单闭包固定顺序锁定所有订单。
func reconcileOrdinaryFeeEconomics(ctx context.Context, client *ent.Client, org, orderID uuid.UUID, audit *biz.AuditEvent) error {
	if audit == nil || audit.UserID == nil {
		return biz.ErrFeeCorrectionInvalid
	}
	projection, err := loadCommissionEconomicProjection(ctx, client, orderID, nil)
	if err != nil {
		return err
	}
	plans, err := planCommissionEconomicRows(ctx, client, org, orderID, projection, true)
	if err != nil {
		return err
	}
	order, err := client.Order.Get(ctx, orderID)
	if err != nil {
		return err
	}
	return applyCommissionEconomicPlans(ctx, client, org, *audit.UserID, biz.CommissionEconomicTrigger{Kind: "FEE_MUTATION", ID: orderID, Version: order.FeeContentVersion}, plans)
}
