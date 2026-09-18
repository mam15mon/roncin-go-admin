package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	orderservicetypeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/shopspring/decimal"
)

// financialLockEvidenceVersion 标识财务锁证据规范编码算法版本；编码语义变化时
// 必须递增，旧申请按其固化版本与哈希复核，避免跨版本误判。
const financialLockEvidenceVersion = "FIN_LOCK_EVIDENCE_V1"

type orderFeeSupplementRepo struct {
	data *Data
}

func NewOrderFeeSupplementRepo(data *Data) biz.OrderFeeSupplementRequestRepo {
	return &orderFeeSupplementRepo{data: data}
}

// financialLockEvidenceComponents 是财务锁证据的参与事实集合：与财务锁净额
// 口径（financeCommissionLockNetAmount / financeLockedOrderPredicate）完全同源，
// 仅统计参与净额的 CONFIRMED/PAID 提成行与调整行。
type financialLockEvidenceComponents struct {
	lines       []*ent.FinanceCommissionLine
	adjustments []*ent.FinanceCommissionAdjustment
}

// hash 按版本化规范编码计算证据哈希：组件类型 + 主键 + 方向 + 符号化 8 位金额，
// 排序后编码；CONFIRMED 与 PAID 统一编码为 ACTIVE（状态不进入编码），同一提成
// 或调整在 CONFIRMED 与 PAID 之间流转不改变证据。
func (c *financialLockEvidenceComponents) hash() (string, error) {
	parts := []string{"fin-lock-evidence/" + strings.ToLower(strings.TrimPrefix(financialLockEvidenceVersion, "FIN_LOCK_EVIDENCE_"))}
	for _, line := range c.lines {
		amount, err := decimalOf(line.CommissionAmount)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("COMMISSION_LINE|%s|ACTIVE|%s", line.ID, amount.StringFixed(8)))
	}
	for _, adjustment := range c.adjustments {
		amount, err := decimalOf(adjustment.Amount)
		if err != nil {
			return "", err
		}
		sign := "+"
		if adjustment.Direction == commissionadjustmentent.DirectionDECREASE {
			sign = "-"
		}
		parts = append(parts, fmt.Sprintf("COMMISSION_ADJUSTMENT|%s|%s|%s%s", adjustment.ID, adjustment.Direction, sign, amount.StringFixed(8)))
	}
	sort.Strings(parts)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:]), nil
}

// netAmount 复用台账财务锁净额的 Go 口径，禁止另写一套锁定公式。
func (c *financialLockEvidenceComponents) netAmount() (decimal.Decimal, error) {
	return financeCommissionLockNetAmount(c.lines, c.adjustments)
}

// loadFinancialLockEvidence 在当前事务客户端内读取参与净额的提成行与调整行。
func loadFinancialLockEvidence(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*financialLockEvidenceComponents, error) {
	lines, err := client.FinanceCommissionLine.Query().Where(
		commissionlineent.OrganizationIDEQ(organizationID),
		commissionlineent.OrderIDEQ(orderID),
		commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
	).WithCommission().All(ctx)
	if err != nil {
		return nil, err
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().Where(
		commissionadjustmentent.OrganizationIDEQ(organizationID),
		commissionadjustmentent.OrderIDEQ(orderID),
		commissionadjustmentent.StatusIn(commissionadjustmentent.StatusCONFIRMED, commissionadjustmentent.StatusPAID),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	return &financialLockEvidenceComponents{lines: lines, adjustments: adjustments}, nil
}

// lockOrderForSupplementInternal 在事务内锁定订单行并计算当前锁事实。补录是
// 业务锁与提成净额财务锁的受控例外：这里不执行普通费用写入口的内容门禁与
// 财务锁拒绝，但财务证据计算必须复用净额口径并在 Order 行锁内读取。
func lockOrderForSupplementInternal(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementLockEvidence, error) {
	order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return nil, err
	}
	components, err := loadFinancialLockEvidence(ctx, tx.Client(), organizationID, orderID)
	if err != nil {
		return nil, err
	}
	hash, err := components.hash()
	if err != nil {
		return nil, err
	}
	net, err := components.netAmount()
	if err != nil {
		return nil, err
	}
	return &biz.OrderFeeSupplementLockEvidence{
		OrderNo:                  order.OrderNo,
		BusinessType:             string(businessType),
		BusinessLocked:           order.LockedAt != nil,
		BusinessLockGeneration:   order.LockGeneration,
		FinancialLocked:          net.Sign() > 0,
		FinancialEvidenceVersion: financialLockEvidenceVersion,
		FinancialEvidenceHash:    hash,
		FinancialNetAmount:       net,
	}, nil
}

func (r *orderFeeSupplementRepo) LockOrderForSupplement(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementLockEvidence, error) {
	var evidence *biz.OrderFeeSupplementLockEvidence
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		result, lockErr := lockOrderForSupplementInternal(ctx, tx, organizationID, orderID)
		if lockErr != nil {
			return lockErr
		}
		evidence = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	return evidence, nil
}

func (r *orderFeeSupplementRepo) GetOrderRef(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementOrderRef, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	return &biz.OrderFeeSupplementOrderRef{ID: order.ID, OrganizationID: order.OrganizationID, OrderNo: order.OrderNo, BusinessType: string(order.BusinessType)}, nil
}

func (r *orderFeeSupplementRepo) ListLockGrantApprovers(ctx context.Context, organizationID, orderID uuid.UUID) ([]biz.OrderFeeSupplementApprover, error) {
	ref, err := r.GetOrderRef(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	businessType, err := orderAccessBusinessType(orderent.BusinessType(ref.BusinessType))
	if err != nil {
		return nil, err
	}
	var result []biz.OrderFeeSupplementApprover
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		candidates, candidateErr := queryQualifiedBusinessLockCandidates(ctx, tx.Client(), organizationID, businessType)
		if candidateErr != nil {
			return candidateErr
		}
		result = make([]biz.OrderFeeSupplementApprover, 0, len(candidates))
		for _, candidate := range candidates {
			result = append(result, biz.OrderFeeSupplementApprover{UserID: candidate.UserID, DisplayName: candidate.DisplayName, DingTalkUserID: candidate.DingTalkUserIDSnapshot})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// hasRealtimeLockGrantWithClient 复用订单直接解锁资格的唯一契约：bootstrap
// admin 显式具备资格；普通用户按统一 lock grant 谓词判定。
func hasRealtimeLockGrantWithClient(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, businessType access.OrderBusinessType, isBootstrapAdmin bool) (bool, error) {
	if isBootstrapAdmin {
		return true, nil
	}
	return isUserQualifiedBusinessLockRole(ctx, client, organizationID, userID, businessType)
}

func (r *orderFeeSupplementRepo) HasRealtimeLockGrant(ctx context.Context, organizationID, orderID, userID uuid.UUID, isBootstrapAdmin bool) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return false, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return false, err
	}
	return hasRealtimeLockGrantWithClient(ctx, client, organizationID, userID, businessType, isBootstrapAdmin)
}

// ---------------------------------------------------------------------------
// 申请持久化
// ---------------------------------------------------------------------------

func supplementSnapshotFromEnt(item *ent.OrderFeeSupplementRequest) (biz.OrderFeeSupplementFeeSnapshot, error) {
	quantity, err := decimalOf(item.Quantity)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	unitPrice, err := decimalOf(item.UnitPrice)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	netAmount, err := decimalOf(item.NetAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	taxAmount, err := decimalOf(item.TaxAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	exchangeRate, err := decimalOf(item.ExchangeRate)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	baseCurrencyAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	snapshot := biz.OrderFeeSupplementFeeSnapshot{
		Direction:             biz.OrderFeeDirection(item.Direction),
		FeeSettingID:          item.FeeSettingID,
		FeeCode:               item.FeeCode,
		FeeName:               item.FeeName,
		FeeNameEN:             item.FeeNameEn,
		SettlementPartyID:     item.SettlementPartyID,
		BillingUnitID:         item.BillingUnitID,
		BillingUnit:           item.BillingUnit,
		TaxableServiceName:    item.TaxableServiceName,
		Quantity:              quantity,
		UnitPrice:             unitPrice,
		TotalAmount:           totalAmount,
		TaxInclusive:          item.TaxInclusive,
		NetAmount:             netAmount,
		TaxAmount:             taxAmount,
		Currency:              item.Currency,
		ExchangeRate:          exchangeRate,
		ExchangeRateSource:    string(item.ExchangeRateSource),
		ExchangeRateDate:      item.ExchangeRateDate,
		ExchangeRateSettingID: item.ExchangeRateSettingID,
		BaseCurrency:          item.BaseCurrency,
		BaseCurrencyAmount:    baseCurrencyAmount,
		ExpenseDate:           item.ExpenseDate,
	}
	if item.TaxRate != nil {
		taxRate, parseErr := decimalOf(*item.TaxRate)
		if parseErr != nil {
			return biz.OrderFeeSupplementFeeSnapshot{}, parseErr
		}
		snapshot.TaxRate = &taxRate
	}
	if item.Note != "" {
		note := item.Note
		snapshot.Note = &note
	}
	return snapshot, nil
}

func supplementRequestToBiz(item *ent.OrderFeeSupplementRequest) (*biz.OrderFeeSupplementRequest, error) {
	snapshot, err := supplementSnapshotFromEnt(item)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeSupplementRequest{
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		OrderID:            item.OrderID,
		LockBasis:          biz.OrderFeeSupplementLockBasis(item.LockBasis),
		IdempotencyKey:     item.IdempotencyKey,
		RequestFingerprint: item.RequestFingerprint,
		Fee:                snapshot,
		Reason:             item.Reason,
		RequestedBy:        item.RequestedBy,
		RequestedAt:        item.RequestedAt,
		Status:             biz.OrderFeeSupplementStatus(item.Status),
		Version:            item.Version,
		DecidedBy:          item.DecidedBy,
		DecidedAt:          item.DecidedAt,
		DecisionReason:     item.DecisionReason,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
	if item.BusinessLockGeneration != nil {
		generation := *item.BusinessLockGeneration
		result.BusinessLockGeneration = &generation
	}
	if item.FinancialLockEvidenceVersion != nil {
		version := *item.FinancialLockEvidenceVersion
		result.FinancialLockEvidenceVersion = &version
	}
	if item.FinancialLockEvidenceHash != nil {
		hash := *item.FinancialLockEvidenceHash
		result.FinancialLockEvidenceHash = &hash
	}
	if item.FinancialLockNetAmountSnapshot != nil {
		net, parseErr := decimalOf(*item.FinancialLockNetAmountSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.FinancialLockNetAmount = &net
	}
	return result, nil
}

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
		Order(orderfeesupplementent.ByRequestedAt(), orderfeesupplementent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderFeeSupplementRequest, 0, len(items))
	for _, item := range items {
		converted, convertErr := supplementRequestToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
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

// ResolveFeeFactsForApproval 在审批事务的锁内复核费用快照引用：结算对象、费用
// 项、计费单位与币种必须仍然有效。汇率与金额的重解析由用例层复用现有解析与
// 计算入口完成，本方法只做引用校验并返回快照原值。
func (r *orderFeeSupplementRepo) ResolveFeeFactsForApproval(ctx context.Context, organizationID, orderID uuid.UUID, snapshot *biz.OrderFeeSupplementFeeSnapshot) (*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	applicability, err := loadFeeApplicability(ctx, client, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if snapshot.FeeSettingID != nil {
		feeSetting, settingErr := client.FeeSetting.Query().
			Where(feesettingent.IDEQ(*snapshot.FeeSettingID), feesettingent.Or(feesettingent.OrganizationIDEQ(organizationID), feesettingent.OrganizationIDIsNil()), feesettingent.EnabledEQ(true)).
			Only(ctx)
		if settingErr != nil || !feeSettingApplies(feeSetting, applicability) {
			return nil, biz.ErrOrderFeeSettingInvalid
		}
	}
	if snapshot.BillingUnitID != nil {
		billingUnit, unitErr := client.BillingUnit.Query().Where(billingunitent.IDEQ(*snapshot.BillingUnitID), billingunitent.EnabledEQ(true)).Only(ctx)
		if unitErr != nil || !billingUnit.Enabled {
			return nil, biz.ErrOrderFeeBillingUnitInvalid
		}
	}
	party, partyErr := client.Partner.Query().Where(partnerent.IDEQ(snapshot.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
	if partyErr != nil {
		return nil, biz.ErrOrderFeePartyInvalid
	}
	validCurrency, currencyErr := client.Currency.Query().Where(currencyent.CodeEQ(snapshot.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
	if currencyErr != nil {
		return nil, currencyErr
	}
	if !validCurrency {
		return nil, biz.ErrOrderFeeCurrencyInvalid
	}
	fee := &biz.OrderFee{
		OrderID:               orderID,
		Direction:             snapshot.Direction,
		FeeSettingID:          snapshot.FeeSettingID,
		FeeCode:               snapshot.FeeCode,
		FeeName:               snapshot.FeeName,
		FeeNameEN:             snapshot.FeeNameEN,
		SettlementPartyID:     snapshot.SettlementPartyID,
		SettlementPartyName:   party.LegalName,
		BillingUnitID:         snapshot.BillingUnitID,
		BillingUnit:           snapshot.BillingUnit,
		TaxRate:               snapshot.TaxRate,
		TaxableServiceName:    snapshot.TaxableServiceName,
		Quantity:              snapshot.Quantity,
		UnitPrice:             snapshot.UnitPrice,
		TotalAmount:           snapshot.TotalAmount,
		TaxInclusive:          snapshot.TaxInclusive,
		Currency:              snapshot.Currency,
		ExchangeRate:          snapshot.ExchangeRate,
		ExchangeRateSource:    snapshot.ExchangeRateSource,
		ExchangeRateDate:      snapshot.ExchangeRateDate,
		ExchangeRateSettingID: snapshot.ExchangeRateSettingID,
		BaseCurrency:          snapshot.BaseCurrency,
		BaseCurrencyAmount:    snapshot.BaseCurrencyAmount,
		ExpenseDate:           snapshot.ExpenseDate,
		Note:                  snapshot.Note,
	}
	return fee, nil
}

// LockCommissionImpactContext 按父单 UUID 升序 FOR UPDATE 锁定包含目标订单的
// CONFIRMED/PAID 提成父单、目标订单行与全部相关调整，并固化边际影响计算上下文。
func (r *orderFeeSupplementRepo) LockCommissionImpactContext(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementImpactContext, error) {
	var result *biz.OrderFeeSupplementImpactContext
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		// 此前已审批且尚未作废的补录应付合计：已作废不进基线。
		priorFees, err := client.OrderFee.Query().Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.SupplementRequestIDNotNil(),
			orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		).All(ctx)
		if err != nil {
			return err
		}
		priorBase := decimal.Zero
		for _, item := range priorFees {
			amount, parseErr := decimalOf(item.BaseCurrencyAmount)
			if parseErr != nil {
				return parseErr
			}
			priorBase = priorBase.Add(amount)
		}
		lines, err := client.FinanceCommissionLine.Query().Where(
			commissionlineent.OrganizationIDEQ(organizationID),
			commissionlineent.OrderIDEQ(orderID),
			commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
		).All(ctx)
		if err != nil {
			return err
		}
		commissionIDs := make([]uuid.UUID, 0, len(lines))
		for _, line := range lines {
			commissionIDs = append(commissionIDs, line.CommissionID)
		}
		// 多张父单按 UUID 升序一次性加锁，固定加锁顺序防死锁。
		sort.Slice(commissionIDs, func(i, j int) bool { return commissionIDs[i].String() < commissionIDs[j].String() })
		result = &biz.OrderFeeSupplementImpactContext{PriorSupplementBaseAmount: priorBase.Round(8), Lines: make([]*biz.OrderFeeSupplementCommissionLine, 0, len(commissionIDs))}
		lineByCommission := make(map[uuid.UUID]*ent.FinanceCommissionLine, len(lines))
		for _, line := range lines {
			lineByCommission[line.CommissionID] = line
		}
		for _, commissionID := range commissionIDs {
			parent, parentErr := tx.FinanceCommission.Query().Where(commissionent.IDEQ(commissionID), commissionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
			if parentErr != nil {
				return parentErr
			}
			line := lineByCommission[commissionID]
			adjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().Where(
				commissionadjustmentent.CommissionIDEQ(commissionID),
			).Order(commissionadjustmentent.ByID()).ForUpdate().All(ctx)
			if adjustmentErr != nil {
				return adjustmentErr
			}
			contextLine, convertErr := commissionImpactLineToBiz(parent, line, adjustments)
			if convertErr != nil {
				return convertErr
			}
			result.Lines = append(result.Lines, contextLine)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// commissionImpactLineToBiz 把锁定的父单、订单行与调整聚合为边际计算上下文行：
// 双层余额按「CONFIRMED/PAID 符号化合计 − 其他 DRAFT DECREASE 预留」统计，
// DRAFT INCREASE 不提前扩张余额。
func commissionImpactLineToBiz(parent *ent.FinanceCommission, line *ent.FinanceCommissionLine, adjustments []*ent.FinanceCommissionAdjustment) (*biz.OrderFeeSupplementCommissionLine, error) {
	parentAmount, err := decimalOf(parent.CommissionAmount)
	if err != nil {
		return nil, err
	}
	ratePercent, err := decimalOf(line.RatePercent)
	if err != nil {
		return nil, err
	}
	lineAmount, err := decimalOf(line.CommissionAmount)
	if err != nil {
		return nil, err
	}
	realizedRevenue, err := decimalOf(line.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeSupplementCommissionLine{
		CommissionID:           parent.ID,
		CommissionNo:           parent.CommissionNo,
		CommissionStatus:       string(parent.Status),
		CommissionAmount:       parentAmount,
		AdjustmentSequence:     parent.AdjustmentSequence,
		EmployeeID:             parent.EmployeeID,
		EmployeeName:           parent.EmployeeName,
		ParentBaseCurrency:     parent.BaseCurrency,
		LineID:                 line.ID,
		OrderID:                line.OrderID,
		OrderNo:                line.OrderNo,
		CalculationVersion:     parent.CalculationVersion,
		CalculationBasis:       biz.CommissionCalculationBasis(line.CalculationBasis),
		CalculationRatePercent: ratePercent,
		RealizedRevenue:        realizedRevenue,
		LineCommissionAmount:   lineAmount,
		LineBaseCurrency:       line.BaseCurrency,
		SnapshotStatus:         snapshotStatusOrEmpty(line.SnapshotStatus),
	}
	if line.TotalReceivableSnapshot != nil {
		receivable, parseErr := decimalOf(*line.TotalReceivableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalReceivableSnapshot = &receivable
	}
	if line.TotalPayableSnapshot != nil {
		payable, parseErr := decimalOf(*line.TotalPayableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalPayableSnapshot = &payable
	}
	parentEffective := parentAmount
	lineEffective := lineAmount
	parentDraftDecrease := decimal.Zero
	lineDraftDecrease := decimal.Zero
	for _, adjustment := range adjustments {
		amount, parseErr := decimalOf(adjustment.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		isDecrease := adjustment.Direction == commissionadjustmentent.DirectionDECREASE
		switch adjustment.Status {
		case commissionadjustmentent.StatusCONFIRMED, commissionadjustmentent.StatusPAID:
			if isDecrease {
				parentEffective = parentEffective.Sub(amount)
				if adjustment.OrderID == line.OrderID {
					lineEffective = lineEffective.Sub(amount)
				}
			} else {
				parentEffective = parentEffective.Add(amount)
				if adjustment.OrderID == line.OrderID {
					lineEffective = lineEffective.Add(amount)
				}
			}
		case commissionadjustmentent.StatusDRAFT:
			if isDecrease {
				parentDraftDecrease = parentDraftDecrease.Add(amount)
				if adjustment.OrderID == line.OrderID {
					lineDraftDecrease = lineDraftDecrease.Add(amount)
				}
			}
		}
	}
	result.LineEffective = lineEffective.Round(8)
	result.ParentEffective = parentEffective.Round(8)
	result.LineDraftDecrease = lineDraftDecrease.Round(8)
	result.ParentDraftDecrease = parentDraftDecrease.Round(8)
	return result, nil
}

// CreateApprovedFee 在审批事务内创建与申请一对一关联的 CONFIRMED 补录费用：
// 幂等键由申请 ID 派生，supplement_request_id 反向关联；唯一约束兜底并发重复。
func (r *orderFeeSupplementRepo) CreateApprovedFee(ctx context.Context, organizationID, requestID uuid.UUID, fee *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	var created *ent.OrderFee
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.OrderFee.Create().
			SetID(fee.ID).
			SetOrderID(fee.OrderID).
			SetIdempotencyKey(fee.IdempotencyKey).
			SetDirection(orderfeeent.Direction(fee.Direction)).
			SetStatus(orderfeeent.StatusCONFIRMED).
			SetFeeCode(fee.FeeCode).
			SetFeeName(fee.FeeName).
			SetSettlementPartyID(fee.SettlementPartyID).
			SetBillingUnit(fee.BillingUnit).
			SetQuantity(fee.Quantity.StringFixed(4)).
			SetUnitPrice(fee.UnitPrice.StringFixed(4)).
			SetTotalAmount(fee.TotalAmount.StringFixed(8)).
			SetTaxInclusive(fee.TaxInclusive).
			SetNetAmount(fee.NetAmount.StringFixed(8)).
			SetTaxAmount(fee.TaxAmount.StringFixed(8)).
			SetCurrency(fee.Currency).
			SetExchangeRate(fee.ExchangeRate.StringFixed(8)).
			SetExchangeRateSource(orderfeeent.ExchangeRateSource(fee.ExchangeRateSource)).
			SetExchangeRateDate(fee.ExchangeRateDate).
			SetBaseCurrency(fee.BaseCurrency).
			SetBaseCurrencyAmount(fee.BaseCurrencyAmount.StringFixed(8)).
			SetExpenseDate(fee.ExpenseDate).
			SetSupplementRequestID(requestID).
			SetVersion(1)
		if fee.FeeSettingID != nil {
			builder.SetFeeSettingID(*fee.FeeSettingID)
		}
		if fee.FeeNameEN != nil {
			builder.SetFeeNameEn(*fee.FeeNameEN)
		}
		if fee.BillingUnitID != nil {
			builder.SetBillingUnitID(*fee.BillingUnitID)
		}
		if fee.TaxRate != nil {
			builder.SetTaxRate(fee.TaxRate.StringFixed(2))
		}
		if fee.TaxableServiceName != nil {
			builder.SetTaxableServiceName(*fee.TaxableServiceName)
		}
		if fee.ExchangeRateSettingID != nil {
			builder.SetExchangeRateSettingID(*fee.ExchangeRateSettingID)
		}
		if fee.Note != nil {
			builder.SetNote(*fee.Note)
		}
		item, saveErr := builder.Save(ctx)
		if saveErr != nil {
			if ent.IsConstraintError(saveErr) {
				return biz.ErrOrderFeeIdempotencyConflict
			}
			return saveErr
		}
		created = item
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	loaded, err := client.OrderFee.Query().Where(orderfeeent.IDEQ(created.ID)).WithSettlementParty().Only(ctx)
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(loaded)
}

// CreateDecreaseSuggestion 在审批事务内创建 DECREASE+DRAFT+LOCKED_FEE_SUPPLEMENT
// 调整：父单行内分配调整号并递增序号；来源关联由数据库 CHECK 强制指向补录申请。
func (r *orderFeeSupplementRepo) CreateDecreaseSuggestion(ctx context.Context, organizationID, requestID uuid.UUID, suggestion *biz.OrderFeeSupplementDecreaseSuggestion, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		parent, err := tx.FinanceCommission.Query().Where(commissionent.IDEQ(suggestion.CommissionID), commissionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if parent.Status != commissionent.StatusCONFIRMED && parent.Status != commissionent.StatusPAID {
			return biz.ErrCommissionAdjustmentTransition
		}
		line, err := tx.FinanceCommissionLine.Query().Where(commissionlineent.CommissionIDEQ(parent.ID), commissionlineent.OrderIDEQ(suggestion.OrderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentInvalid, nil)
		}
		sequence := parent.AdjustmentSequence + 1
		idempotencyKey := fmt.Sprintf("lfs:%s:%s:%s", requestID, suggestion.CommissionID, suggestion.OrderID)
		if _, err = tx.FinanceCommissionAdjustment.Create().
			SetID(suggestion.AdjustmentID).
			SetOrganizationID(organizationID).
			SetCommissionID(parent.ID).
			SetOrderID(line.OrderID).
			SetAdjustmentNo(fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)).
			SetIdempotencyKey(idempotencyKey).
			SetCommissionNo(parent.CommissionNo).
			SetOrderNo(line.OrderNo).
			SetEmployeeID(parent.EmployeeID).
			SetEmployeeName(parent.EmployeeName).
			SetSourceType(commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT).
			SetSourceFeeSupplementRequestID(requestID).
			SetDirection(commissionadjustmentent.DirectionDECREASE).
			SetStatus(commissionadjustmentent.StatusDRAFT).
			SetBaseCurrency(parent.BaseCurrency).
			SetAmount(suggestion.Amount.StringFixed(8)).
			SetReason(suggestion.Reason).
			SetVersion(1).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrFeeSupplementTransition
			}
			return err
		}
		if _, err = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ---------------------------------------------------------------------------
// 通知入队
// ---------------------------------------------------------------------------

// EnqueueApprovalPendingNotifications 在创建申请的同一事务内为审批人逐人入队
// 待审批通知：任务 ID 与幂等键按「申请 + 审批人」确定性生成，BackgroundTask 与
// NotificationDelivery 均以唯一约束幂等收敛；未绑定钉钉的审批人跳过投递明细，
// 审批资格仍以接口实时校验为准。
func (r *orderFeeSupplementRepo) EnqueueApprovalPendingNotifications(ctx context.Context, organizationID, requestID uuid.UUID, orderNo string, totalAmount decimal.Decimal, currency string, approvers []biz.OrderFeeSupplementApprover) error {
	if len(approvers) == 0 {
		return nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	summary := clampNotificationBytes(fmt.Sprintf("订单 %s 补录应付 %s %s", orderNo, totalAmount.StringFixed(8), currency), 256)
	referenceCode := clampNotificationBytes(orderNo, 64)
	if summary == "" || referenceCode == "" {
		return nil
	}
	now := time.Now()
	for _, approver := range approvers {
		if strings.TrimSpace(approver.DingTalkUserID) == "" {
			continue
		}
		taskID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte("fee-supplement-pending:"+requestID.String()+":"+approver.UserID.String()))
		if err := client.BackgroundTask.Create().
			SetID(taskID).
			SetOrganizationID(organizationID).
			SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
			SetIdempotencyKey("dingtalk-notice:" + taskID.String()).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetMaxAttempts(5).
			SetNextRunAt(now).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
		if err := client.NotificationDelivery.Create().
			SetBackgroundTaskID(taskID).
			SetRecipientUserID(approver.UserID).
			SetChannel(notificationent.ChannelDINGTALK).
			SetTemplate(notificationent.TemplateFEE_SUPPLEMENT_APPROVAL_PENDING).
			SetResourceType("FEE_SUPPLEMENT_REQUEST").
			SetResourceID(requestID).
			SetReferenceCode(referenceCode).
			SetParameter(summary).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// EnqueueDecreaseSuggestedNotifications 在审批事务内为实际生成冲减建议的员工
// 逐人入队知情通知：任务 ID 与幂等键按「调整 + 员工」确定性生成；未产生建议
// 的员工不通知。
func (r *orderFeeSupplementRepo) EnqueueDecreaseSuggestedNotifications(ctx context.Context, organizationID uuid.UUID, orderNo string, suggestions []biz.OrderFeeSupplementDecreaseSuggestion) error {
	if len(suggestions) == 0 {
		return nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	referenceCode := clampNotificationBytes(orderNo, 64)
	if referenceCode == "" {
		return nil
	}
	now := time.Now()
	for _, suggestion := range suggestions {
		summary := clampNotificationBytes(fmt.Sprintf("冲减建议 %s %s", suggestion.Amount.StringFixed(8), suggestion.BaseCurrency), 256)
		taskID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte("commission-decrease-suggested:"+suggestion.AdjustmentID.String()+":"+suggestion.EmployeeID.String()))
		if err := client.BackgroundTask.Create().
			SetID(taskID).
			SetOrganizationID(organizationID).
			SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
			SetIdempotencyKey("dingtalk-notice:" + taskID.String()).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetMaxAttempts(5).
			SetNextRunAt(now).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
		if err := client.NotificationDelivery.Create().
			SetBackgroundTaskID(taskID).
			SetRecipientUserID(suggestion.EmployeeID).
			SetChannel(notificationent.ChannelDINGTALK).
			SetTemplate(notificationent.TemplateCOMMISSION_DECREASE_SUGGESTED).
			SetResourceType("COMMISSION_ADJUSTMENT").
			SetResourceID(suggestion.AdjustmentID).
			SetReferenceCode(referenceCode).
			SetParameter(summary).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// loadFeeApplicability 读取订单的服务类型与异常案例适用范围，供费用项适用性
// 复核使用；与 orderFeeRepo.loadApplicability 同一口径，仅以显式客户端参数
// 支持事务内调用。
func loadFeeApplicability(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*orderFeeApplicability, error) {
	exists, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrOrderFeeNotFound
	}
	serviceTypes, err := client.OrderServiceType.Query().Where(orderservicetypeent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, err
	}
	abnormalCases, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(orderID), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &orderFeeApplicability{serviceTypeIDs: make(map[uuid.UUID]struct{}, len(serviceTypes)), abnormalCaseIDs: make(map[uuid.UUID]struct{}, len(abnormalCases))}
	for _, serviceType := range serviceTypes {
		result.serviceTypeIDs[serviceType.MasterDataItemID] = struct{}{}
	}
	for _, abnormalCase := range abnormalCases {
		result.abnormalCaseIDs[abnormalCase.AbnormalCaseID] = struct{}{}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 专用作废：能力投影与 design 5.1 六步事务
// ---------------------------------------------------------------------------

// supplementCancelBlockReason 是专用作废阻断的稳定投影。
type supplementCancelBlockReason struct {
	code   string
	reason string
}

// evaluateCancelCapability 按专用作废前置条件只读评估能力：仅限 APPROVED 申请、
// CONFIRMED、无活动账单行、无更晚生效且未作废的补录，且关联调整 confirmed_at/
// paid_at 从未写入、当前全为 DRAFT/CANCELLED。列表投影与专用作废命令复用本实现。
func evaluateCancelCapability(ctx context.Context, client *ent.Client, organizationID, orderID, requestID uuid.UUID) (*biz.OrderFeeSupplementCancelCapability, error) {
	result := &biz.OrderFeeSupplementCancelCapability{}
	request, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(requestID), orderfeesupplementent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	if request.OrderID != orderID {
		return nil, biz.ErrFeeSupplementNotFound
	}
	if request.Status != orderfeesupplementent.StatusAPPROVED {
		return result, nil
	}
	fee, err := client.OrderFee.Query().Where(orderfeeent.SupplementRequestIDEQ(requestID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return result, nil
		}
		return nil, err
	}
	result.FeeID = &fee.ID
	result.FeeStatus = string(fee.Status)
	blocked := func(code, reason string) (*biz.OrderFeeSupplementCancelCapability, error) {
		result.Cancellable = false
		result.BlockReasonCode = code
		result.BlockReason = reason
		return result, nil
	}
	if fee.Status == orderfeeent.StatusCANCELLED {
		return blocked("FEE_ALREADY_CANCELLED", "补录费用已作废")
	}
	activeLine, err := client.FinanceBillLine.Query().
		Where(financebilllineent.OrderFeeIDEQ(fee.ID), financebilllineent.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if activeLine || fee.Status == orderfeeent.StatusBILLED {
		return blocked("FEE_BILLED", "补录费用已建账，需先按现有财务链路取消账单")
	}
	if fee.Status != orderfeeent.StatusCONFIRMED {
		return blocked("FEE_STATUS_INVALID", "仅 CONFIRMED 状态的补录费用可专用作废")
	}
	// 更晚生效且未作废的补录存在时，必须按批准时间倒序逐笔作废，避免撤销早期
	// 成本后让后续边际冲减失真。比较键为批准时间 + 申请 ID。
	decidedAt := time.Time{}
	if request.DecidedAt != nil {
		decidedAt = *request.DecidedAt
	}
	laterEffective, err := client.OrderFee.Query().Where(
		orderfeeent.OrderIDEQ(orderID),
		orderfeeent.SupplementRequestIDNotNil(),
		orderfeeent.SupplementRequestIDNEQ(requestID),
		orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		orderfeeent.HasSupplementRequestWith(
			orderfeesupplementent.StatusEQ(orderfeesupplementent.StatusAPPROVED),
			orderfeesupplementent.Or(
				orderfeesupplementent.DecidedAtGT(decidedAt),
				orderfeesupplementent.And(
					orderfeesupplementent.DecidedAtEQ(decidedAt),
					orderfeesupplementent.IDGT(requestID),
				),
			),
		),
	).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if laterEffective {
		return blocked("LATER_SUPPLEMENT_EXISTS", "存在更晚生效的有效补录，请按批准时间倒序逐笔作废")
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().
		Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(requestID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, adjustment := range adjustments {
		if adjustment.ConfirmedAt != nil || adjustment.PaidAt != nil ||
			adjustment.Status == commissionadjustmentent.StatusCONFIRMED || adjustment.Status == commissionadjustmentent.StatusPAID {
			return blocked("ADJUSTMENT_CONFIRMED", "关联冲减建议已确认或已扣回，不允许直接作废补录费用")
		}
	}
	result.Cancellable = true
	return result, nil
}

func (r *orderFeeSupplementRepo) CancelCapability(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*biz.OrderFeeSupplementCancelCapability, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return evaluateCancelCapability(ctx, client, organizationID, orderID, requestID)
}

// CancelApprovedFee 执行专用作废事务，固定锁序为 Order → 补录申请/费用 → 按
// UUID 排序的提成父单 → 关联调整（design 5.1 六步）。任何一步失败整体回滚；
// 该事务不删除任何事实，也不把系统作废伪装成财务主动忽略建议。
func (r *orderFeeSupplementRepo) CancelApprovedFee(ctx context.Context, input *biz.OrderFeeSupplementCancelInput) (*biz.OrderFeeSupplementCancelResult, error) {
	var result *biz.OrderFeeSupplementCancelResult
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		// 1. Order FOR UPDATE 后实时复核直接解锁资格；订单只作互斥点，不调用
		// 会禁止锁后专用动作的普通费用内容门禁。
		order, orderErr := tx.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
		if orderErr != nil {
			return mapEntError(orderErr, biz.ErrFeeSupplementNotFound, nil)
		}
		if order.ID != input.OrderID {
			return biz.ErrFeeSupplementNotFound
		}
		businessType, typeErr := orderAccessBusinessType(order.BusinessType)
		if typeErr != nil {
			return typeErr
		}
		qualified, grantErr := hasRealtimeLockGrantWithClient(ctx, client, input.OrganizationID, input.OperatorID, businessType, input.IsBootstrapAdmin)
		if grantErr != nil {
			return grantErr
		}
		if !qualified {
			return biz.ErrPermissionDenied
		}
		// 2. 锁定 APPROVED 申请和生成费用，比较费用版本，确认状态与来源匹配。
		request, requestErr := tx.OrderFeeSupplementRequest.Query().
			Where(orderfeesupplementent.IDEQ(input.RequestID), orderfeesupplementent.OrganizationIDEQ(input.OrganizationID)).
			ForUpdate().Only(ctx)
		if requestErr != nil {
			return mapEntError(requestErr, biz.ErrFeeSupplementNotFound, nil)
		}
		if request.OrderID != input.OrderID {
			return biz.ErrFeeSupplementNotFound
		}
		if request.Status != orderfeesupplementent.StatusAPPROVED {
			return biz.ErrFeeSupplementCancelBlocked
		}
		fee, feeErr := tx.OrderFee.Query().Where(orderfeeent.SupplementRequestIDEQ(input.RequestID)).ForUpdate().Only(ctx)
		if feeErr != nil {
			return mapEntError(feeErr, biz.ErrFeeSupplementCancelBlocked, nil)
		}
		if fee.OrderID != input.OrderID {
			return biz.ErrFeeSupplementCancelBlocked
		}
		if fee.Version != input.FeeExpectedVersion {
			return biz.ErrOrderFeeVersionConflict
		}
		blocked, capabilityErr := evaluateCancelCapability(ctx, client, input.OrganizationID, input.OrderID, input.RequestID)
		if capabilityErr != nil {
			return capabilityErr
		}
		if !blocked.Cancellable {
			return newFeeSupplementCancelBlockedError(blocked.BlockReason)
		}
		// 4. 按 design §5.1 固定锁序：先只读定位关联调整并按父单 UUID 升序锁定
		// 全部提成父单，再按 ID 锁定关联调整，与审批/确认路径保持同一顺序，
		// 避免多订单父单并发下形成反向锁序死锁。任一调整曾进入 CONFIRMED 或
		// PAID 时已在能力评估中整体拒绝。
		linkedAdjustments, linkedErr := tx.FinanceCommissionAdjustment.Query().
			Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(input.RequestID)).
			All(ctx)
		if linkedErr != nil {
			return linkedErr
		}
		commissionIDs := make([]uuid.UUID, 0, len(linkedAdjustments))
		for _, adjustment := range linkedAdjustments {
			commissionIDs = append(commissionIDs, adjustment.CommissionID)
		}
		for _, commissionID := range uniqueSortedUUIDs(commissionIDs) {
			if _, parentErr := tx.FinanceCommission.Query().Where(commissionent.IDEQ(commissionID), commissionent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx); parentErr != nil {
				return parentErr
			}
		}
		adjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().
			Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(input.RequestID)).
			Order(commissionadjustmentent.ByID()).ForUpdate().All(ctx)
		if adjustmentErr != nil {
			return adjustmentErr
		}
		// 5. 将关联 DRAFT 调整转为 CANCELLED，再将费用转为 CANCELLED；APPROVED
		// 申请保持终态不变，生成费用已作废由列表投影呈现。
		cancelledIDs := make([]uuid.UUID, 0, len(adjustments))
		now := time.Now().UTC()
		for _, adjustment := range adjustments {
			if adjustment.Status != commissionadjustmentent.StatusDRAFT {
				continue
			}
			if _, updateErr := tx.FinanceCommissionAdjustment.UpdateOne(adjustment).
				SetStatus(commissionadjustmentent.StatusCANCELLED).
				SetCancelledAt(now).
				SetCancelledBy(input.OperatorID).
				SetCancellationReason(input.Reason).
				SetVersion(adjustment.Version + 1).
				Save(ctx); updateErr != nil {
				return updateErr
			}
			cancelledIDs = append(cancelledIDs, adjustment.ID)
		}
		if _, feeUpdateErr := tx.OrderFee.UpdateOne(fee).
			SetStatus(orderfeeent.StatusCANCELLED).
			SetVersion(fee.Version + 1).
			SetCancelledAt(now).
			SetCancelledBy(input.OperatorID).
			SetCancellationReason(input.Reason).
			Save(ctx); feeUpdateErr != nil {
			return feeUpdateErr
		}
		// 6. 写作废人、时间、原因、费用前一版本与被取消调整 ID 审计后提交。
		if input.Audit != nil {
			if input.Audit.Details == nil {
				input.Audit.Details = make(map[string]string)
			}
			input.Audit.Details["fee.id"] = fee.ID.String()
			input.Audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(fee.Version)).String()
			input.Audit.Details["fee.previous_status"] = string(fee.Status)
			cancelledIDTexts := make([]string, 0, len(cancelledIDs))
			for _, id := range cancelledIDs {
				cancelledIDTexts = append(cancelledIDTexts, id.String())
			}
			input.Audit.Details["cancelled_adjustment_ids"] = strings.Join(cancelledIDTexts, ",")
			if writeErr := writeAudit(ctx, tx.AuditLog, input.Audit); writeErr != nil {
				return writeErr
			}
		}
		loaded, loadErr := tx.OrderFee.Query().Where(orderfeeent.IDEQ(fee.ID)).WithSettlementParty().Only(ctx)
		if loadErr != nil {
			return loadErr
		}
		converted, convertErr := orderFeeToBiz(loaded)
		if convertErr != nil {
			return convertErr
		}
		result = &biz.OrderFeeSupplementCancelResult{Fee: converted, CancelledAdjustmentIDs: cancelledIDs}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// newFeeSupplementCancelBlockedError 以具体阻断原因构造稳定错误。
func newFeeSupplementCancelBlockedError(reason string) error {
	if reason == "" {
		return biz.ErrFeeSupplementCancelBlocked
	}
	return kratoserrors.Conflict("FEE_SUPPLEMENT_CANCEL_BLOCKED", reason)
}

func uniqueSortedUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

var _ biz.OrderFeeSupplementRequestRepo = (*orderFeeSupplementRepo)(nil)
