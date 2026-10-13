package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	correction "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

func (r *commissionRepo) GetAdjustmentByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommissionAdjustment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().WithOrganization().WithCommission(commissionAdjustmentParentProjection).Where(adjustment.OrganizationIDEQ(org), adjustment.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return commissionAdjustmentToBiz(x)
}

func (r *commissionRepo) GetAdjustmentScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommissionAdjustment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDIn(organizationIDs...)).WithOrganization().WithCommission(commissionAdjustmentParentProjection).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
	}
	return commissionAdjustmentToBiz(x)
}

// commissionAdjustmentListPredicates 构造调整列表筛选谓词：关键字覆盖订单号、
// 提成号与调整号；状态、来源与员工过滤显式命中。
func commissionAdjustmentListPredicates(organizationIDs []uuid.UUID, f biz.CommissionAdjustmentFilter) []predicate.FinanceCommissionAdjustment {
	p := []predicate.FinanceCommissionAdjustment{adjustment.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		p = append(p, adjustment.Or(
			adjustment.OrderNoContainsFold(f.Keyword),
			adjustment.CommissionNoContainsFold(f.Keyword),
			adjustment.AdjustmentNoContainsFold(f.Keyword),
		))
	}
	if f.Status != "" {
		p = append(p, adjustment.StatusEQ(adjustment.Status(f.Status)))
	}
	if f.SourceType != "" {
		p = append(p, adjustment.SourceTypeEQ(adjustment.SourceType(f.SourceType)))
	}
	if f.EmployeeID != uuid.Nil {
		p = append(p, adjustment.EmployeeIDEQ(f.EmployeeID))
	}
	return p
}

// ListAdjustmentsScoped 服务端分页读取提成调整，默认 created_at 倒序、主键倒序
// 兜底稳定排序；组织范围显式传入，由调用方按 commission.read 权限解析。
func (r *commissionRepo) ListAdjustmentsScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionAdjustmentFilter) (*biz.CommissionAdjustmentListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	q := client.FinanceCommissionAdjustment.Query().Where(commissionAdjustmentListPredicates(organizationIDs, f)...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.WithOrganization().WithCommission(commissionAdjustmentParentProjection).
		Order(adjustment.ByCreatedAt(entsql.OrderDesc()), adjustment.ByID(entsql.OrderDesc())).
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionAdjustmentListResult{Items: make([]*biz.FinanceCommissionAdjustment, 0, len(xs)), Total: int64(total), Page: f.Page, PageSize: f.PageSize}
	for _, x := range xs {
		converted, convertErr := commissionAdjustmentToBiz(x)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

// GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源最小详情。查询谓词同时
// 限定调整 ID、employee_id = 当前用户、LOCKED_FEE_SUPPLEMENT 来源，以及「调整
// 所属组织启用且当前用户存在启用成员关系」；任一不满足统一映射为调整不存在，
// 不泄露他人调整的记录事实。
func (r *commissionRepo) GetMyFeeSupplementAdjustmentSource(ctx context.Context, userID, id uuid.UUID) (*biz.MyFeeSupplementAdjustmentSource, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().Where(
		adjustment.IDEQ(id),
		adjustment.EmployeeIDEQ(userID),
		adjustment.SourceTypeIn(adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT, adjustment.SourceTypeFEE_CORRECTION, adjustment.SourceTypeECONOMIC_REALLOCATION),
		adjustment.HasOrganizationWith(
			organizationent.EnabledEQ(true),
			organizationent.HasMembershipsWith(membership.UserIDEQ(userID), membership.EnabledEQ(true)),
		),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
	}
	if x.SourceType != adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT {
		amount, err := decimalOf(x.Amount)
		if err != nil {
			return nil, err
		}
		result := &biz.MyFeeSupplementAdjustmentSource{AdjustmentID: x.ID, AdjustmentNo: x.AdjustmentNo, OrderNo: x.OrderNo, CommissionNo: x.CommissionNo, Status: biz.CommissionStatus(x.Status), SuggestedAmount: amount, BaseCurrency: x.BaseCurrency, CreatedAt: x.CreatedAt, SourceType: biz.CommissionAdjustmentSourceType(x.SourceType), Direction: biz.CommissionAdjustmentDirection(x.Direction), SupplementReason: x.Reason}
		if x.SourceFeeCorrectionRequestID != nil {
			request, err := client.OrderFeeCorrectionRequest.Get(ctx, *x.SourceFeeCorrectionRequestID)
			if err != nil {
				return nil, biz.ErrCommissionAdjustmentNotFound
			}
			result.SourceReferenceID = x.SourceFeeCorrectionRequestID
			result.SupplementReason = request.Reason
			var f biz.OrderFee
			raw := request.AfterSnapshot
			if len(raw) == 0 {
				raw = request.BeforeSnapshot
			}
			if err := json.Unmarshal(raw, &f); err != nil {
				return nil, biz.ErrCommissionSnapshotUnavailable
			}
			result.FeeCode = f.FeeCode
			result.FeeName = f.FeeName
			result.FeeCurrency = f.Currency
			result.FeeTotalAmount = f.TotalAmount
			result.FeeBaseCurrency = f.BaseCurrency
			result.FeeBaseCurrencyAmount = f.BaseCurrencyAmount
			result.FeeExpenseDate = f.ExpenseDate
		} else if x.SourceVerificationID != nil {
			v, err := client.FinanceVerification.Get(ctx, *x.SourceVerificationID)
			if err != nil {
				return nil, biz.ErrCommissionAdjustmentNotFound
			}
			result.SourceReferenceID = x.SourceVerificationID
			result.SourceNo = &v.VerificationNo
		} else if x.SourceNettingID != nil {
			n, err := client.FinanceNetting.Get(ctx, *x.SourceNettingID)
			if err != nil {
				return nil, biz.ErrCommissionAdjustmentNotFound
			}
			result.SourceReferenceID = x.SourceNettingID
			result.SourceNo = &n.NettingNo
		}
		return result, nil
	}
	requestID := uuid.Nil
	if x.SourceFeeSupplementRequestID != nil {
		requestID = *x.SourceFeeSupplementRequestID
	}
	if requestID == uuid.Nil {
		return nil, biz.ErrCommissionAdjustmentNotFound
	}
	request, requestErr := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(requestID)).
		Only(ctx)
	if requestErr != nil {
		return nil, biz.ErrCommissionAdjustmentNotFound
	}
	amount, parseErr := decimalOf(x.Amount)
	if parseErr != nil {
		return nil, parseErr
	}
	feeTotal, parseErr := decimalOf(request.TotalAmount)
	if parseErr != nil {
		return nil, parseErr
	}
	feeBase, parseErr := decimalOf(request.BaseCurrencyAmount)
	if parseErr != nil {
		return nil, parseErr
	}
	return &biz.MyFeeSupplementAdjustmentSource{
		AdjustmentID: x.ID, AdjustmentNo: x.AdjustmentNo, OrderNo: x.OrderNo, CommissionNo: x.CommissionNo,
		Status: biz.CommissionStatus(x.Status), SuggestedAmount: amount, BaseCurrency: x.BaseCurrency, CreatedAt: x.CreatedAt, SourceType: biz.CommissionAdjustmentSourceType(x.SourceType), Direction: biz.CommissionAdjustmentDirection(x.Direction), SourceReferenceID: x.SourceFeeSupplementRequestID,
		FeeCode: request.FeeCode, FeeName: request.FeeName, FeeCurrency: request.Currency, FeeTotalAmount: feeTotal,
		FeeBaseCurrency: request.BaseCurrency, FeeBaseCurrencyAmount: feeBase, FeeExpenseDate: request.ExpenseDate,
		SupplementReason: request.Reason,
	}, nil
}

func (r *commissionRepo) CreateAdjustment(ctx context.Context, org, actor uuid.UUID, item *biz.FinanceCommissionAdjustment, audit *biz.AuditEvent) (*biz.FinanceCommissionAdjustment, error) {
	var created *ent.FinanceCommissionAdjustment
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		lines, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(item.CommissionID)).All(ctx)
		if err != nil {
			return err
		}
		if err := lockOrdersSortedForFinance(ctx, tx, orderUUIDsFromLines(lines)); err != nil {
			return err
		}
		if err := ensureCommissionOrdersReady(ctx, tx.Client(), orderUUIDsFromLines(lines), false); err != nil {
			return err
		}
		parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(item.CommissionID), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if parent.Status != commission.StatusCONFIRMED && parent.Status != commission.StatusPAID {
			return biz.ErrCommissionAdjustmentTransition
		}
		line, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(parent.ID), commissionline.OrderIDEQ(item.OrderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentInvalid, nil)
		}
		sequence := parent.AdjustmentSequence + 1
		item.AdjustmentNo = fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)
		item.CommissionNo, item.OrderNo = parent.CommissionNo, line.OrderNo
		item.EmployeeID, item.EmployeeName = parent.EmployeeID, parent.EmployeeName
		item.BaseCurrency = parent.BaseCurrency
		created, err = tx.FinanceCommissionAdjustment.Create().
			SetID(item.ID).SetOrganizationID(org).SetCommissionID(parent.ID).SetOrderID(line.OrderID).
			SetAdjustmentNo(item.AdjustmentNo).SetIdempotencyKey(item.IdempotencyKey).
			SetCommissionNo(parent.CommissionNo).SetOrderNo(line.OrderNo).
			SetEmployeeID(parent.EmployeeID).SetEmployeeName(parent.EmployeeName).
			SetSourceType(adjustment.SourceType(item.SourceType)).SetDirection(adjustment.Direction(item.Direction)).SetStatus(adjustment.StatusDRAFT).
			SetBaseCurrency(parent.BaseCurrency).SetAmount(item.Amount.StringFixed(8)).SetReason(item.Reason).
			SetNillableNote(item.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrCommissionAdjustmentInvalid)
		}
		if _, err = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).SetVersion(parent.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	loaded, err := r.GetAdjustmentScoped(ctx, []uuid.UUID{org}, created.ID)
	return loaded, err
}

func (r *commissionRepo) TransitionAdjustment(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommissionAdjustment, error) {
	var updated *ent.FinanceCommissionAdjustment
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 调整状态迁移会改变订单财务锁证据集合与双层余额；先只读定位目标调整，
		// 再按 Order → 提成父单 → 调整固定锁序取得行锁，避免补录审批复核证据后
		// 被并发改写。
		pre, preErr := tx.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDEQ(org)).Only(ctx)
		if preErr != nil {
			return mapEntError(preErr, biz.ErrCommissionAdjustmentNotFound, nil)
		}
		preLines, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(pre.CommissionID)).All(ctx)
		if err != nil {
			return err
		}
		if lockErr := lockOrdersSortedForFinance(ctx, tx, orderUUIDsFromLines(preLines)); lockErr != nil {
			return lockErr
		}
		parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(pre.CommissionID), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		x, err := tx.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
		}
		if x.Version != version {
			return biz.ErrCommissionAdjustmentTransition
		}
		if target == biz.CommissionConfirmed || target == biz.CommissionPaid {
			if err := ensureCommissionOrdersReady(ctx, tx.Client(), orderUUIDsFromLines(preLines), false); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		update := tx.FinanceCommissionAdjustment.UpdateOne(x).SetVersion(version + 1)
		switch target {
		case biz.CommissionConfirmed:
			if commissionSystemSource(x.SourceType) {
				return biz.ErrCommissionAdjustmentTransition
			}
			if x.Status != adjustment.StatusDRAFT || (parent.Status != commission.StatusCONFIRMED && parent.Status != commission.StatusPAID) {
				return biz.ErrCommissionAdjustmentTransition
			}
			if x.Direction == adjustment.DirectionDECREASE {
				currentAmount, parseErr := decimalOf(x.Amount)
				if parseErr != nil {
					return parseErr
				}
				// 订单行层有效余额：以 FinanceCommissionLine.commission_amount 为起点，
				// 只汇总同 commission_id + order_id 的 CONFIRMED/PAID 符号化调整
				//（不含其他 DRAFT）。确认在锁内重算，不静默缩小建议金额；
				// 同父单其他订单行的正余额不得替当前订单行兜底。
				line, lineErr := tx.FinanceCommissionLine.Query().
					Where(commissionline.CommissionIDEQ(parent.ID), commissionline.OrderIDEQ(x.OrderID)).
					Only(ctx)
				if lineErr != nil {
					return mapEntError(lineErr, biz.ErrCommissionAdjustmentInvalid, nil)
				}
				lineEffective, parseErr := decimalOf(line.CommissionAmount)
				if parseErr != nil {
					return parseErr
				}
				lineAdjustments, queryErr := tx.FinanceCommissionAdjustment.Query().Where(
					adjustment.CommissionIDEQ(parent.ID),
					adjustment.OrderIDEQ(x.OrderID),
					adjustment.IDNEQ(x.ID),
					adjustment.StatusIn(adjustment.StatusCONFIRMED, adjustment.StatusPAID),
				).All(ctx)
				if queryErr != nil {
					return queryErr
				}
				for _, old := range lineAdjustments {
					amount, amountErr := decimalOf(old.Amount)
					if amountErr != nil {
						return amountErr
					}
					if old.Direction == adjustment.DirectionDECREASE {
						lineEffective = lineEffective.Sub(amount)
					} else {
						lineEffective = lineEffective.Add(amount)
					}
				}
				if lineEffective.Sub(currentAmount).IsNegative() {
					return biz.ErrCommissionAdjustmentExceeds
				}
				// 父单层有效余额：以 FinanceCommission.commission_amount 为起点，
				// 汇总父单全部 CONFIRMED/PAID 符号化调整（不含其他 DRAFT）。
				active, queryErr := tx.FinanceCommissionAdjustment.Query().Where(
					adjustment.CommissionIDEQ(parent.ID), adjustment.IDNEQ(x.ID),
					adjustment.StatusIn(adjustment.StatusCONFIRMED, adjustment.StatusPAID),
				).All(ctx)
				if queryErr != nil {
					return queryErr
				}
				effective, parseErr := decimalOf(parent.CommissionAmount)
				if parseErr != nil {
					return parseErr
				}
				for _, old := range active {
					amount, amountErr := decimalOf(old.Amount)
					if amountErr != nil {
						return amountErr
					}
					if old.Direction == adjustment.DirectionDECREASE {
						effective = effective.Sub(amount)
					} else {
						effective = effective.Add(amount)
					}
				}
				if effective.Sub(currentAmount).IsNegative() {
					return biz.ErrCommissionAdjustmentExceeds
				}
			}
			update.SetStatus(adjustment.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
		case biz.CommissionPaid:
			if x.Status != adjustment.StatusCONFIRMED || parent.Status != commission.StatusPAID || x.ExecutionMethod != nil {
				return biz.ErrCommissionAdjustmentTransition
			}
			executed, err := tx.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(parent.ID), adjustment.StatusEQ(adjustment.StatusPAID)).Order(adjustment.ByID()).ForUpdate().All(ctx)
			if err != nil {
				return err
			}
			cash, err := commissionCashExposure(parent, executed)
			if err != nil {
				return err
			}
			currentAmount, err := decimalOf(x.Amount)
			if err != nil {
				return err
			}
			if x.Direction == adjustment.DirectionDECREASE && cash.LessThan(currentAmount) {
				return biz.ErrCommissionAdjustmentExceeds
			}
			update.SetStatus(adjustment.StatusPAID).SetPaidAt(now).SetPaidBy(actor).SetExecutionMethod(adjustment.ExecutionMethodSEPARATE)
		case biz.CommissionCancelled:
			if commissionSystemSource(x.SourceType) {
				// 补录批准直接确认系统冲减，任何状态均不可通过通用接口忽略。
				return biz.ErrCommissionAdjustmentCancelNotAllowed
			}
			if x.Status != adjustment.StatusDRAFT && x.Status != adjustment.StatusCONFIRMED {
				return biz.ErrCommissionAdjustmentTransition
			}
			update.SetStatus(adjustment.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
		default:
			return biz.ErrCommissionAdjustmentInvalid
		}
		updated, err = update.Save(ctx)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	loaded, err := r.GetAdjustmentScoped(ctx, []uuid.UUID{org}, updated.ID)
	return loaded, err
}

func commissionAdjustmentToBiz(x *ent.FinanceCommissionAdjustment) (*biz.FinanceCommissionAdjustment, error) {
	amount, err := decimalOf(x.Amount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommissionAdjustment{
		ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID,
		AdjustmentNo: x.AdjustmentNo, IdempotencyKey: x.IdempotencyKey, CommissionNo: x.CommissionNo,
		OrderNo: x.OrderNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName,
		Direction: biz.CommissionAdjustmentDirection(x.Direction), SourceType: biz.CommissionAdjustmentSourceType(x.SourceType), SourceVerificationID: x.SourceVerificationID, SourceFeeSupplementRequestID: x.SourceFeeSupplementRequestID, SourceFeeCorrectionRequestID: x.SourceFeeCorrectionRequestID, SourceNettingID: x.SourceNettingID, SupersededByFeeCorrectionRequestID: x.SupersededByFeeCorrectionRequestID, Status: biz.CommissionStatus(x.Status),
		BaseCurrency: x.BaseCurrency, Amount: amount, Reason: x.Reason, Note: x.Note, Version: x.Version,
		ConfirmedAt: x.ConfirmedAt, ConfirmedBy: x.ConfirmedBy, PaidAt: x.PaidAt, PaidBy: x.PaidBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy,
		ExecutionMethod: executionMethodText(x.ExecutionMethod), ParentPaymentID: x.ParentPaymentID,
		CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt,
	}
	var rows []*ent.FinanceCommissionAdjustment
	if x.Edges.Commission != nil {
		rows = x.Edges.Commission.Edges.Adjustments
	}
	populateAdjustmentExecutionCapability(result, x, x.Edges.Commission, rows)
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	return result, nil
}

func executionMethodText(value *adjustment.ExecutionMethod) *string {
	if value == nil {
		return nil
	}
	s := string(*value)
	return &s
}

func commissionAdjustmentParentProjection(q *ent.FinanceCommissionQuery) {
	q.WithAdjustments().WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.WithOrder(func(q *ent.OrderQuery) {
			q.WithFeeCorrectionRequests(func(q *ent.OrderFeeCorrectionRequestQuery) { q.Where(correction.StatusEQ(correction.StatusPENDING)) })
			q.WithFeeSupplementRequests(func(q *ent.OrderFeeSupplementRequestQuery) {
				q.Where(orderfeesupplementent.StatusEQ(orderfeesupplementent.StatusPENDING))
			})
		})
	})
}
