package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

// Create 幂等创建申请：同组织同幂等键同指纹返回既有申请，同键不同指纹返回
// 稳定幂等冲突；组织级唯一索引兜底并发重试。
func (r *orderFeeSupplementRepo) Create(ctx context.Context, request *biz.OrderFeeSupplementRequest, audit *biz.AuditEvent) (*biz.OrderFeeSupplementRequest, error) {
	var result *biz.OrderFeeSupplementRequest
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.OrderFeeSupplementRequest.Query().
			Where(orderfeesupplementent.OrganizationIDEQ(request.OrganizationID), orderfeesupplementent.IdempotencyKeyEQ(request.IdempotencyKey)).
			Only(ctx)
		if queryErr == nil {
			return reuseOrCreateResult(existing, request, &result)
		}
		if !ent.IsNotFound(queryErr) {
			return queryErr
		}
		builder := tx.OrderFeeSupplementRequest.Create().
			SetID(request.ID).
			SetOrganizationID(request.OrganizationID).
			SetOrderID(request.OrderID).
			SetLockBasis(orderfeesupplementent.LockBasis(request.LockBasis)).
			SetIdempotencyKey(request.IdempotencyKey).
			SetRequestFingerprint(request.RequestFingerprint).
			SetDirection(orderfeesupplementent.Direction(request.Fee.Direction)).
			SetFeeCode(request.Fee.FeeCode).
			SetFeeName(request.Fee.FeeName).
			SetSettlementPartyID(request.Fee.SettlementPartyID).
			SetBillingUnit(request.Fee.BillingUnit).
			SetQuantity(request.Fee.Quantity.StringFixed(4)).
			SetUnitPrice(request.Fee.UnitPrice.StringFixed(4)).
			SetTotalAmount(request.Fee.TotalAmount.StringFixed(8)).
			SetTaxInclusive(request.Fee.TaxInclusive).
			SetNetAmount(request.Fee.NetAmount.StringFixed(8)).
			SetTaxAmount(request.Fee.TaxAmount.StringFixed(8)).
			SetCurrency(request.Fee.Currency).
			SetExchangeRate(request.Fee.ExchangeRate.StringFixed(8)).
			SetExchangeRateSource(orderfeesupplementent.ExchangeRateSource(request.Fee.ExchangeRateSource)).
			SetExchangeRateDate(request.Fee.ExchangeRateDate).
			SetBaseCurrency(request.Fee.BaseCurrency).
			SetBaseCurrencyAmount(request.Fee.BaseCurrencyAmount.StringFixed(8)).
			SetExpenseDate(request.Fee.ExpenseDate).
			SetReason(request.Reason).
			SetRequestedBy(request.RequestedBy).
			SetRequestedAt(request.RequestedAt).
			SetStatus(orderfeesupplementent.StatusPENDING).
			SetVersion(request.Version)
		if request.BusinessLockGeneration != nil {
			builder.SetBusinessLockGeneration(*request.BusinessLockGeneration)
		}
		if request.FinancialLockEvidenceVersion != nil {
			builder.SetFinancialLockEvidenceVersion(*request.FinancialLockEvidenceVersion).
				SetFinancialLockEvidenceHash(*request.FinancialLockEvidenceHash).
				SetFinancialLockNetAmountSnapshot(request.FinancialLockNetAmount.StringFixed(8))
		}
		if request.Fee.FeeSettingID != nil {
			builder.SetFeeSettingID(*request.Fee.FeeSettingID)
		}
		if request.Fee.FeeNameEN != nil {
			builder.SetFeeNameEn(*request.Fee.FeeNameEN)
		}
		if request.Fee.BillingUnitID != nil {
			builder.SetBillingUnitID(*request.Fee.BillingUnitID)
		}
		if request.Fee.TaxRate != nil {
			builder.SetTaxRate(request.Fee.TaxRate.StringFixed(2))
		}
		if request.Fee.TaxableServiceName != nil {
			builder.SetTaxableServiceName(*request.Fee.TaxableServiceName)
		}
		if request.Fee.ExchangeRateSettingID != nil {
			builder.SetExchangeRateSettingID(*request.Fee.ExchangeRateSettingID)
		}
		if request.Fee.Note != nil {
			builder.SetNote(*request.Fee.Note)
		}
		if _, createErr := builder.Save(ctx); createErr != nil {
			if ent.IsConstraintError(createErr) {
				// 并发重试命中组织级幂等唯一索引：仅同指纹语义重放返回原申请。
				retry, retryErr := tx.OrderFeeSupplementRequest.Query().
					Where(orderfeesupplementent.OrganizationIDEQ(request.OrganizationID), orderfeesupplementent.IdempotencyKeyEQ(request.IdempotencyKey)).
					Only(ctx)
				if retryErr == nil {
					return reuseOrCreateResult(retry, request, &result)
				}
				return biz.ErrFeeSupplementIdempotencyConflict
			}
			return createErr
		}
		result = request
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// reuseOrCreateResult 判定幂等命中结果：同指纹返回原申请，不同指纹返回稳定
// 幂等冲突。
func reuseOrCreateResult(existing *ent.OrderFeeSupplementRequest, request *biz.OrderFeeSupplementRequest, result **biz.OrderFeeSupplementRequest) error {
	if existing.RequestFingerprint != request.RequestFingerprint {
		return biz.ErrFeeSupplementIdempotencyConflict
	}
	converted, err := supplementRequestToBiz(existing)
	if err != nil {
		return err
	}
	*result = converted
	return nil
}

func (r *orderFeeSupplementRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderFeeSupplementRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(id), orderfeesupplementent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	return supplementRequestToBiz(item)
}

func (r *orderFeeSupplementRepo) ListByOrder(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderFeeSupplementRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.OrganizationIDEQ(organizationID), orderfeesupplementent.OrderIDEQ(orderID)).
		WithRequestedByUser().
		WithDecidedByUser().
		Order(orderfeesupplementent.ByRequestedAt(), orderfeesupplementent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	partyIDs := make([]uuid.UUID, 0, len(items))
	partySeen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if _, exists := partySeen[item.SettlementPartyID]; !exists {
			partySeen[item.SettlementPartyID] = struct{}{}
			partyIDs = append(partyIDs, item.SettlementPartyID)
		}
	}
	partyNames := make(map[uuid.UUID]string, len(partyIDs))
	if len(partyIDs) > 0 {
		parties, partyErr := client.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(partyIDs...)).All(ctx)
		if partyErr != nil {
			return nil, partyErr
		}
		for _, party := range parties {
			partyNames[party.ID] = party.LegalName
		}
	}
	result := make([]*biz.OrderFeeSupplementRequest, 0, len(items))
	for _, item := range items {
		converted, convertErr := supplementRequestToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		converted.Fee.SettlementPartyName = partyNames[item.SettlementPartyID]
		result = append(result, converted)
	}
	// 列表按发起时间倒序展示（最新申请在前）。
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].RequestedAt.Equal(result[j].RequestedAt) {
			return result[i].RequestedAt.After(result[j].RequestedAt)
		}
		return result[i].ID.String() > result[j].ID.String()
	})
	return result, nil
}

func (r *orderFeeSupplementRepo) LockForDecision(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderFeeSupplementRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(id), orderfeesupplementent.OrganizationIDEQ(organizationID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	return supplementRequestToBiz(item)
}

// SaveDecision 在申请行锁内写入终态与决策字段：以「version + PENDING」条件
// 更新保证状态机唯一迁移，竞争失败方影响行数为零并返回状态冲突。
func (r *orderFeeSupplementRepo) SaveDecision(ctx context.Context, request *biz.OrderFeeSupplementRequest) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if !request.Status.Terminal() {
		return biz.ErrFeeSupplementTransition
	}
	previousVersion := request.Version - 1
	if previousVersion == 0 {
		previousVersion = request.Version
	}
	update := client.OrderFeeSupplementRequest.Update().
		Where(
			orderfeesupplementent.IDEQ(request.ID),
			orderfeesupplementent.VersionEQ(previousVersion),
			orderfeesupplementent.StatusEQ(orderfeesupplementent.StatusPENDING),
		).
		SetStatus(orderfeesupplementent.Status(request.Status)).
		SetVersion(request.Version)
	if request.DecidedBy != nil {
		update.SetDecidedBy(*request.DecidedBy)
	}
	if request.DecidedAt != nil {
		update.SetDecidedAt(*request.DecidedAt)
	}
	if request.DecisionReason != nil {
		update.SetDecisionReason(*request.DecisionReason)
	}
	affected, saveErr := update.Save(ctx)
	if saveErr != nil {
		return saveErr
	}
	if affected == 0 {
		return biz.ErrFeeSupplementTransition
	}
	return nil
}

func (r *orderFeeSupplementRepo) SaveAudit(ctx context.Context, event *biz.AuditEvent) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	return writeAudit(ctx, client.AuditLog, event)
}

// ---------------------------------------------------------------------------
// 审批：费用事实重解析、提成上下文加锁、费用与建议创建
// ---------------------------------------------------------------------------
