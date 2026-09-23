package biz

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/shopspring/decimal"
)

// OrderFeeSupplementUsecase 锁单后应付费用补录申请用例：负责状态机、锁依据
// 复核、边际影响计算、双层封顶与共享事务编排。普通费用写入口的锁门禁不因本
// 用例放宽；补录审批承担费用真实性确认，生成 UNBILLED 费用。
type OrderFeeSupplementUsecase struct {
	repo       OrderFeeSupplementRequestRepo
	fee        *OrderFeeUsecase
	transactor Transactor
	now        func() time.Time
}

func NewOrderFeeSupplementUsecase(repo OrderFeeSupplementRequestRepo, fee *OrderFeeUsecase, transactor Transactor) *OrderFeeSupplementUsecase {
	return &OrderFeeSupplementUsecase{repo: repo, fee: fee, transactor: transactor, now: time.Now}
}

// OrderFeeSupplementCreateInput 是创建补录申请的领域入参；锁类型与锁依据由
// 服务端在 Order 行锁内判定固化，客户端提供的锁事实一律被忽略。
type OrderFeeSupplementCreateInput struct {
	Direction            OrderFeeDirection
	FeeSettingID         uuid.UUID
	SettlementPartyID    uuid.UUID
	BillingUnitID        uuid.UUID
	Quantity             decimal.Decimal
	UnitPrice            decimal.Decimal
	Currency             string
	ExpenseDate          string
	Note                 *string
	ExchangeRateOverride *decimal.Decimal
	TaxInclusive         bool
	Reason               string
	IdempotencyKey       string
}

// OrderFeeSupplementApproveResult 是审批通过的结果投影。
type OrderFeeSupplementApproveResult struct {
	Request     *OrderFeeSupplementRequest
	Fee         *OrderFee
	Suggestions []OrderFeeSupplementDecreaseSuggestion
}

// OrderFeeSupplementApprovalPreview 是审批时点的本币费用毛利估算。
type OrderFeeSupplementApprovalPreview struct {
	BaseCurrency                                           string
	CurrentReceivable, CurrentPayable, CurrentProfit       decimal.Decimal
	CurrentProfitRate                                      *decimal.Decimal
	SupplementCost                                         decimal.Decimal
	ProjectedReceivable, ProjectedPayable, ProjectedProfit decimal.Decimal
	ProjectedProfitRate                                    *decimal.Decimal
	ProfitChange                                           decimal.Decimal
}

// PreviewApproval 只读复核审批资格和锁依据，按审批相同的费用事实解析链估算毛利。
func (uc *OrderFeeSupplementUsecase) PreviewApproval(ctx context.Context, caller *Principal, organizationID, orderID, requestID uuid.UUID, expectedVersion uint64) (*OrderFeeSupplementApprovalPreview, error) {
	if caller == nil || !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || requestID == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	request, err := uc.repo.Get(ctx, organizationID, requestID)
	if err != nil {
		return nil, err
	}
	if request.OrderID != orderID {
		return nil, ErrFeeSupplementNotFound
	}
	if request.Version != expectedVersion || request.Status != OrderFeeSupplementPending {
		return nil, ErrFeeSupplementTransition
	}
	qualified, err := uc.repo.HasRealtimeLockGrant(ctx, organizationID, orderID, caller.UserID, caller.IsBootstrapAdmin)
	if err != nil {
		return nil, err
	}
	if !qualified {
		return nil, ErrPermissionDenied
	}
	evidence, err := uc.repo.ReadLockEvidence(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if err = verifyLockBasis(request, evidence); err != nil {
		return nil, err
	}
	fee, err := uc.repo.ResolveFeeFactsForApproval(ctx, organizationID, orderID, &request.Fee)
	if err != nil {
		return nil, err
	}
	if request.Fee.ExchangeRateSource != ExchangeRateSourceManual {
		if err = uc.fee.resolveExchangeRate(ctx, organizationID, orderID, fee, false); err != nil {
			return nil, err
		}
	}
	if err = uc.fee.calculateAmounts(ctx, organizationID, fee); err != nil {
		return nil, err
	}
	fees, err := uc.fee.List(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	return calculateSupplementApprovalPreview(fees, fee)
}

func calculateSupplementApprovalPreview(fees []*OrderFee, fee *OrderFee) (*OrderFeeSupplementApprovalPreview, error) {
	if fee == nil || fee.BaseCurrency == "" || fee.Direction != OrderFeePayable || fee.BaseCurrencyAmount.Sign() <= 0 {
		return nil, ErrOrderFeeInvalidArgument
	}
	result := &OrderFeeSupplementApprovalPreview{BaseCurrency: fee.BaseCurrency, SupplementCost: fee.BaseCurrencyAmount}
	for _, current := range fees {
		if current == nil {
			return nil, ErrOrderFeeInvalidArgument
		}
		if current.Status == OrderFeeCancelled {
			continue
		}
		if current.BaseCurrency != result.BaseCurrency {
			return nil, ErrOrderFeeInvalidArgument
		}
		switch current.Direction {
		case OrderFeeReceivable:
			result.CurrentReceivable = result.CurrentReceivable.Add(current.BaseCurrencyAmount)
		case OrderFeePayable:
			result.CurrentPayable = result.CurrentPayable.Add(current.BaseCurrencyAmount)
		default:
			return nil, ErrOrderFeeInvalidArgument
		}
	}
	result.CurrentProfit = result.CurrentReceivable.Sub(result.CurrentPayable)
	result.ProjectedReceivable = result.CurrentReceivable
	result.ProjectedPayable = result.CurrentPayable.Add(result.SupplementCost)
	result.ProjectedProfit = result.ProjectedReceivable.Sub(result.ProjectedPayable)
	result.ProfitChange = result.ProjectedProfit.Sub(result.CurrentProfit)
	if result.CurrentReceivable.Sign() != 0 {
		currentRate := result.CurrentProfit.DivRound(result.CurrentReceivable, 6).Mul(decimal.NewFromInt(100))
		projectedRate := result.ProjectedProfit.DivRound(result.ProjectedReceivable, 6).Mul(decimal.NewFromInt(100))
		result.CurrentProfitRate = &currentRate
		result.ProjectedProfitRate = &projectedRate
	}
	return result, nil
}

// OrderFeeSupplementListView 是补录申请列表的逐行授权结果。
type OrderFeeSupplementListView struct {
	Items []*OrderFeeSupplementRequestView
	Total int
	Page  int
	Size  int
}

// OrderFeeSupplementRequestView 在申请领域对象之上追加当前调用人的能力投影。
type OrderFeeSupplementRequestView struct {
	Request           *OrderFeeSupplementRequest
	CanApprove        bool
	CanWithdraw       bool
	CanCancel         bool
	CancelBlockReason string
	CancelBlockCode   string
	FeeID             *uuid.UUID
	FeeStatus         string
	ApproverAvailable bool
}

// validSupplementCaller 校验调用人与组织参数。
func validSupplementCaller(caller *Principal, organizationID, orderID uuid.UUID) bool {
	return caller != nil && caller.UserID != uuid.Nil && organizationID != uuid.Nil && orderID != uuid.Nil
}

// Create 在 Order 行锁内判定业务锁与财务锁，二者均不存在时拒绝并引导普通新增；
// 仅业务锁、仅财务锁和双锁分别固化为 BUSINESS、FINANCIAL、BOTH。财务锁依据
// 固化版本化证据三元组与当时净额；同一事务写入申请、审计与逐审批人的待审批
// 通知，找不到任何当前有效审批人时零写入。
func (uc *OrderFeeSupplementUsecase) Create(ctx context.Context, caller *Principal, organizationID, orderID uuid.UUID, input *OrderFeeSupplementCreateInput, canOverrideExchangeRate bool) (*OrderFeeSupplementRequest, error) {
	if !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || input == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	// 双重校验第一层：补录仅接受正数应付成本，应收方向在领域边界拒绝。
	if input.Direction != OrderFeePayable {
		return nil, ErrFeeSupplementInvalidArgument
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.Reason == "" || utf8.RuneCountInString(input.Reason) > 500 ||
		input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	shell := &OrderFee{
		IdempotencyKey:       input.IdempotencyKey,
		Direction:            input.Direction,
		FeeSettingID:         &input.FeeSettingID,
		SettlementPartyID:    input.SettlementPartyID,
		BillingUnitID:        &input.BillingUnitID,
		Quantity:             input.Quantity,
		UnitPrice:            input.UnitPrice,
		Currency:             input.Currency,
		ExpenseDate:          input.ExpenseDate,
		Note:                 input.Note,
		TaxInclusive:         input.TaxInclusive,
		ExchangeRateOverride: input.ExchangeRateOverride,
	}
	normalized, err := normalizeOrderFee(shell)
	if err != nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	if uc.transactor == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	var created *OrderFeeSupplementRequest
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Order 行锁内判定锁事实；普通费用写入口门禁不适用于本受控入口。
		evidence, lockErr := uc.repo.LockOrderForSupplement(txCtx, organizationID, orderID)
		if lockErr != nil {
			return lockErr
		}
		basis, lockGen, evidenceVersion, evidenceHash, netAmount := classifySupplementLockBasis(evidence)
		if basis == "" {
			return ErrFeeSupplementNotApplicable
		}
		approvers, approverErr := uc.repo.ListLockGrantApprovers(txCtx, organizationID, orderID)
		if approverErr != nil {
			return approverErr
		}
		if len(approvers) == 0 {
			return ErrFeeSupplementApproverUnavailable
		}
		// 锁内解析费用项、计费单位、结算对象、币种与费用发生日汇率，固化快照。
		fee := &OrderFee{
			IdempotencyKey:       normalized.IdempotencyKey,
			Direction:            normalized.Direction,
			FeeSettingID:         normalized.FeeSettingID,
			SettlementPartyID:    normalized.SettlementPartyID,
			BillingUnitID:        normalized.BillingUnitID,
			Quantity:             normalized.Quantity,
			UnitPrice:            normalized.UnitPrice,
			TotalAmount:          normalized.TotalAmount,
			Currency:             normalized.Currency,
			ExpenseDate:          normalized.ExpenseDate,
			Note:                 normalized.Note,
			TaxInclusive:         normalized.TaxInclusive,
			ExchangeRateOverride: normalized.ExchangeRateOverride,
		}
		if catalogErr := uc.fee.resolveCatalog(txCtx, organizationID, orderID, fee, true, false); catalogErr != nil {
			return catalogErr
		}
		if rateErr := uc.fee.resolveExchangeRate(txCtx, organizationID, orderID, fee, canOverrideExchangeRate); rateErr != nil {
			return rateErr
		}
		if amountErr := uc.fee.calculateAmounts(txCtx, organizationID, fee); amountErr != nil {
			return amountErr
		}
		snapshot := feeSnapshotFromOrderFee(fee)
		// 双重校验第二层：快照必须为正数应付。
		if validateErr := snapshot.Validate(); validateErr != nil {
			return validateErr
		}
		fingerprint := BuildOrderFeeSupplementFingerprint(orderID, snapshot, input.Reason)
		request := &OrderFeeSupplementRequest{
			ID:                           uuid.Must(uuid.NewV7()),
			OrganizationID:               organizationID,
			OrderID:                      orderID,
			LockBasis:                    basis,
			BusinessLockGeneration:       lockGen,
			FinancialLockEvidenceVersion: evidenceVersion,
			FinancialLockEvidenceHash:    evidenceHash,
			FinancialLockNetAmount:       netAmount,
			IdempotencyKey:               input.IdempotencyKey,
			RequestFingerprint:           fingerprint,
			Fee:                          snapshot,
			Reason:                       input.Reason,
			RequestedBy:                  caller.UserID,
			RequestedAt:                  uc.now().UTC(),
			Status:                       OrderFeeSupplementPending,
			Version:                      1,
		}
		if validateErr := request.Validate(); validateErr != nil {
			return validateErr
		}
		result, createErr := uc.repo.Create(txCtx, request, uc.createAudit(organizationID, caller.UserID, request, evidence, approvers))
		if createErr != nil {
			return createErr
		}
		if result.ID == request.ID {
			if notifyErr := uc.repo.EnqueueApprovalPendingNotifications(txCtx, organizationID, request.ID, evidence.OrderNo, fee.TotalAmount, fee.Currency, approvers); notifyErr != nil {
				return notifyErr
			}
			created = result
			return nil
		}
		// 幂等命中：同键同指纹的语义重放返回原申请。
		created = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// classifySupplementLockBasis 依据锁事实收敛 lock_basis 与固化字段：仅业务锁、
// 仅财务锁、双锁；二者均不存在时返回空 basis（调用方拒绝补录）。
func classifySupplementLockBasis(evidence *OrderFeeSupplementLockEvidence) (OrderFeeSupplementLockBasis, *uint64, *string, *string, *decimal.Decimal) {
	business := evidence.BusinessLocked
	financial := evidence.FinancialLocked && evidence.FinancialNetAmount.Sign() > 0
	switch {
	case business && financial:
		generation := evidence.BusinessLockGeneration
		version := evidence.FinancialEvidenceVersion
		hash := evidence.FinancialEvidenceHash
		net := evidence.FinancialNetAmount
		return SupplementLockBasisBoth, &generation, &version, &hash, &net
	case business:
		generation := evidence.BusinessLockGeneration
		return SupplementLockBasisBusiness, &generation, nil, nil, nil
	case financial:
		version := evidence.FinancialEvidenceVersion
		hash := evidence.FinancialEvidenceHash
		net := evidence.FinancialNetAmount
		return SupplementLockBasisFinancial, nil, &version, &hash, &net
	default:
		return "", nil, nil, nil, nil
	}
}

func feeSnapshotFromOrderFee(fee *OrderFee) OrderFeeSupplementFeeSnapshot {
	return OrderFeeSupplementFeeSnapshot{
		Direction:             fee.Direction,
		FeeSettingID:          fee.FeeSettingID,
		FeeCode:               fee.FeeCode,
		FeeName:               fee.FeeName,
		FeeNameEN:             fee.FeeNameEN,
		SettlementPartyID:     fee.SettlementPartyID,
		BillingUnitID:         fee.BillingUnitID,
		BillingUnit:           fee.BillingUnit,
		TaxRate:               fee.TaxRate,
		TaxableServiceName:    fee.TaxableServiceName,
		Quantity:              fee.Quantity,
		UnitPrice:             fee.UnitPrice,
		TotalAmount:           fee.TotalAmount,
		TaxInclusive:          fee.TaxInclusive,
		NetAmount:             fee.NetAmount,
		TaxAmount:             fee.TaxAmount,
		Currency:              fee.Currency,
		ExchangeRate:          fee.ExchangeRate,
		ExchangeRateSource:    fee.ExchangeRateSource,
		ExchangeRateDate:      fee.ExchangeRateDate,
		ExchangeRateSettingID: fee.ExchangeRateSettingID,
		BaseCurrency:          fee.BaseCurrency,
		BaseCurrencyAmount:    fee.BaseCurrencyAmount,
		ExpenseDate:           fee.ExpenseDate,
		Note:                  fee.Note,
	}
}

func (uc *OrderFeeSupplementUsecase) createAudit(organizationID, actorID uuid.UUID, request *OrderFeeSupplementRequest, evidence *OrderFeeSupplementLockEvidence, approvers []OrderFeeSupplementApprover) *AuditEvent {
	// 审批人资格快照（含未绑定钉钉者）写入创建审计，保证「提交时谁有资格」
	// 存在持久痕迹；通知收件人投影与审批资格仍以实时校验为准。
	approverIDs := make([]string, 0, len(approvers))
	for _, approver := range approvers {
		approverIDs = append(approverIDs, approver.UserID.String())
	}
	sort.Strings(approverIDs)
	details := map[string]string{
		"approver_count":      strconv.Itoa(len(approverIDs)),
		"approver_snapshot":   strings.Join(approverIDs, ","),
		"request.id":          request.ID.String(),
		"order.id":            request.OrderID.String(),
		"lock_basis":          string(request.LockBasis),
		"fee.code":            request.Fee.FeeCode,
		"fee.amount":          request.Fee.TotalAmount.StringFixed(8),
		"fee.currency":        request.Fee.Currency,
		"fee.expense_date":    request.Fee.ExpenseDate,
		"fingerprint.version": feeSupplementFingerprintVersion,
		"fingerprint":         request.RequestFingerprint,
		"idempotency_key":     request.IdempotencyKey,
		"business_locked":     boolToFlag(evidence.BusinessLocked),
		"financial_locked":    boolToFlag(evidence.FinancialLocked),
	}
	if request.BusinessLockGeneration != nil {
		details["business_lock_generation"] = decimal.NewFromUint64(*request.BusinessLockGeneration).String()
	}
	if request.FinancialLockEvidenceVersion != nil {
		details["financial_lock_evidence_version"] = *request.FinancialLockEvidenceVersion
		details["financial_lock_evidence_hash"] = *request.FinancialLockEvidenceHash
		details["financial_lock_net_amount"] = request.FinancialLockNetAmount.StringFixed(8)
	}
	return &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee_supplement.create",
		Result:         "success",
		ResourceType:   "order_fee_supplement_request",
		ResourceID:     request.ID.String(),
		Details:        details,
	}
}

// verifyLockBasis 按申请固化的 lock_basis 复核原始依据：BUSINESS 要求同一业务
// 锁代次；FINANCIAL 要求净额大于零且同版本证据哈希一致；BOTH 任一匹配即可。
// 原始依据全部失效时返回 LOCK_BASIS_CHANGED，并按当前锁状态分别提示重新申请
// 或改走普通费用新增；提交后新出现的锁不得自动承接旧申请。
func verifyLockBasis(request *OrderFeeSupplementRequest, evidence *OrderFeeSupplementLockEvidence) error {
	businessMatched := request.BusinessLockGeneration != nil && evidence.BusinessMatched(*request.BusinessLockGeneration)
	financialMatched := request.FinancialLockEvidenceVersion != nil && request.FinancialLockEvidenceHash != nil &&
		evidence.FinancialMatched(*request.FinancialLockEvidenceVersion, *request.FinancialLockEvidenceHash)
	switch request.LockBasis {
	case SupplementLockBasisBusiness:
		if businessMatched {
			return nil
		}
	case SupplementLockBasisFinancial:
		if financialMatched {
			return nil
		}
	case SupplementLockBasisBoth:
		if businessMatched || financialMatched {
			return nil
		}
	default:
		return ErrFeeSupplementInvalidArgument
	}
	if evidence.BusinessLocked || evidence.FinancialLocked {
		return newLockBasisChangedError(LockBasisChangedRecreateSupplement, "申请提交时的锁依据已全部失效，订单当前存在新的锁，请重新发起补录申请")
	}
	return newLockBasisChangedError(LockBasisChangedUseNormalFeeEntry, "申请提交时的锁依据已全部失效，订单当前已无任何锁，请改走普通费用新增入口")
}

// Approve 按 design 第 5 节九步固定顺序执行：Order 行锁 → 申请行锁校验版本与
// 状态 → 实时审批资格 → 锁依据复核 → 锁内重解析费用事实 → 按父单 UUID 升序
// 锁定提成上下文 → 边际计算与双层封顶 → 创建费用与建议 → 申请终态、审计与
// 逐员工通知。任何一步失败整体回滚，不允许部分成功。
func (uc *OrderFeeSupplementUsecase) Approve(ctx context.Context, caller *Principal, organizationID, orderID, requestID uuid.UUID, expectedVersion uint64) (*OrderFeeSupplementApproveResult, error) {
	if !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || requestID == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	if uc.transactor == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	var result *OrderFeeSupplementApproveResult
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. FOR UPDATE 锁定订单。
		evidence, lockErr := uc.repo.LockOrderForSupplement(txCtx, organizationID, orderID)
		if lockErr != nil {
			return lockErr
		}
		// 2. 锁定申请行并校验版本与状态。
		request, requestErr := uc.repo.LockForDecision(txCtx, organizationID, requestID)
		if requestErr != nil {
			return requestErr
		}
		if request.OrderID != orderID {
			return ErrFeeSupplementNotFound
		}
		if request.Version != expectedVersion || request.Status != OrderFeeSupplementPending {
			return ErrFeeSupplementTransition
		}
		// 3. 实时校验审批人 lock grant 与组织范围。
		qualified, grantErr := uc.repo.HasRealtimeLockGrant(txCtx, organizationID, orderID, caller.UserID, caller.IsBootstrapAdmin)
		if grantErr != nil {
			return grantErr
		}
		if !qualified {
			return ErrPermissionDenied
		}
		// 2b. 按固化 lock_basis 复核原始锁依据。
		if basisErr := verifyLockBasis(request, evidence); basisErr != nil {
			return basisErr
		}
		// 4. 锁内重新解析结算对象、费用项、币种与费用发生日汇率。
		fee, feeErr := uc.repo.ResolveFeeFactsForApproval(txCtx, organizationID, orderID, &request.Fee)
		if feeErr != nil {
			return feeErr
		}
		// 费用发生日汇率按现有解析链在锁内重新固化；申请时手工覆盖的快照保留
		// 原汇率事实，不用审批日或订单月份静默替代。
		if request.Fee.ExchangeRateSource != ExchangeRateSourceManual {
			if rateErr := uc.fee.resolveExchangeRate(txCtx, organizationID, orderID, fee, false); rateErr != nil {
				return rateErr
			}
		}
		if amountErr := uc.fee.calculateAmounts(txCtx, organizationID, fee); amountErr != nil {
			return amountErr
		}
		fee.ID = uuid.Must(uuid.NewV7())
		fee.Status = OrderFeeUnbilled
		fee.Version = 1
		fee.IdempotencyKey = "fee-supplement:" + request.ID.String()
		// 5. 按父单 UUID 升序锁定提成父单、订单行与调整。
		impact, impactErr := uc.repo.LockCommissionImpactContext(txCtx, organizationID, orderID)
		if impactErr != nil {
			return impactErr
		}
		// 6. 按 calculation_version 路由成本敏感性并计算边际差额与双层封顶。
		suggestions, auditDetails, computeErr := uc.computeSuggestions(impact, fee, request.Reason)
		if computeErr != nil {
			return computeErr
		}
		// 7. 创建 UNBILLED 费用与 DECREASE+DRAFT 调整。
		approveAudit := uc.approveAudit(organizationID, caller.UserID, request, evidence, auditDetails)
		createdFee, feeCreateErr := uc.repo.CreateApprovedFee(txCtx, organizationID, request.ID, fee, approveAudit)
		if feeCreateErr != nil {
			return feeCreateErr
		}
		for _, suggestion := range suggestions {
			if suggestErr := uc.repo.CreateDecreaseSuggestion(txCtx, organizationID, request.ID, &suggestion, commissionAdjustmentAudit(organizationID, caller.UserID, suggestion.AdjustmentID, "finance.commission_adjustment.create")); suggestErr != nil {
				return suggestErr
			}
		}
		// 8. 更新申请为 APPROVED。
		now := uc.now().UTC()
		request.Status = OrderFeeSupplementApproved
		request.DecidedBy = &caller.UserID
		request.DecidedAt = &now
		request.Version = request.Version + 1
		if decisionErr := uc.repo.SaveDecision(txCtx, request); decisionErr != nil {
			return decisionErr
		}
		// 9. 仅为实际生成冲减建议的员工逐人入队知情通知。
		if notifyErr := uc.repo.EnqueueDecreaseSuggestedNotifications(txCtx, organizationID, evidence.OrderNo, suggestions); notifyErr != nil {
			return notifyErr
		}
		result = &OrderFeeSupplementApproveResult{Request: request, Fee: createdFee, Suggestions: suggestions}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// computeSuggestions 对每条原提成订单行独立计算边际影响：按 calculation_version
// 路由成本敏感性（未知版本失败关闭）；成本敏感行要求 READY 快照且本位币与新
// 费用一致；此前未作废补录进入 before 基线；建议金额按订单行与父单双层封顶。
func (uc *OrderFeeSupplementUsecase) computeSuggestions(impact *OrderFeeSupplementImpactContext, fee *OrderFee, supplementReason string) ([]OrderFeeSupplementDecreaseSuggestion, map[string]string, error) {
	auditDetails := make(map[string]string)
	suggestions := make([]OrderFeeSupplementDecreaseSuggestion, 0, len(impact.Lines))
	for _, line := range impact.Lines {
		sensitive, sensitivityErr := CommissionSupplementCostSensitivity(line.CalculationVersion, line.CalculationBasis)
		if sensitivityErr != nil {
			return nil, nil, sensitivityErr
		}
		if !sensitive {
			// REALIZED_REVENUE 等明确不受应付成本影响的行跳过分母与币种阻断。
			continue
		}
		// 注意全空存量行 SnapshotStatus 为空，不得放行。
		if line.SnapshotStatus != "READY" || line.TotalReceivableSnapshot == nil || line.TotalPayableSnapshot == nil {
			return nil, nil, ErrCommissionSnapshotUnavailable
		}
		if line.LineBaseCurrency != fee.BaseCurrency {
			return nil, nil, ErrCommissionBaseCurrencyMismatch
		}
		impactResult := ComputeSupplementMarginalImpact(
			line.RealizedRevenue, *line.TotalReceivableSnapshot, *line.TotalPayableSnapshot,
			impact.PriorSupplementBaseAmount, fee.BaseCurrencyAmount, line.CalculationRatePercent, line.CalculationBasis,
		)
		lineKey := "impact." + line.CommissionID.String() + "." + line.OrderID.String()
		auditDetails[lineKey+".before"] = impactResult.AmountBefore.StringFixed(8)
		auditDetails[lineKey+".after"] = impactResult.AmountAfter.StringFixed(8)
		auditDetails[lineKey+".marginal"] = impactResult.Delta.StringFixed(8)
		if impactResult.Delta.Sign() <= 0 {
			continue
		}
		lineAvailable := line.LineEffective.Sub(line.LineDraftDecrease)
		parentAvailable := line.ParentEffective.Sub(line.ParentDraftDecrease)
		suggested, excess := CapSupplementSuggestion(impactResult.Delta, lineAvailable, parentAvailable)
		auditDetails[lineKey+".line_available"] = lineAvailable.StringFixed(8)
		auditDetails[lineKey+".parent_available"] = parentAvailable.StringFixed(8)
		auditDetails[lineKey+".excess"] = excess.StringFixed(8)
		if suggested.Sign() <= 0 {
			continue
		}
		suggestions = append(suggestions, OrderFeeSupplementDecreaseSuggestion{
			AdjustmentID:  uuid.Must(uuid.NewV7()),
			CommissionID:  line.CommissionID,
			CommissionNo:  line.CommissionNo,
			OrderID:       line.OrderID,
			OrderNo:       line.OrderNo,
			EmployeeID:    line.EmployeeID,
			EmployeeName:  line.EmployeeName,
			BaseCurrency:  fee.BaseCurrency,
			Amount:        suggested,
			Reason:        clampRunes("锁后费用补录自动冲减："+supplementReason, 500),
			AmountBefore:  impactResult.AmountBefore,
			AmountAfter:   impactResult.AmountAfter,
			MarginalDelta: impactResult.Delta,
			ExcessAmount:  excess,
		})
	}
	return suggestions, auditDetails, nil
}

// clampRunes 按字符数上限安全截断文本。
func clampRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func (uc *OrderFeeSupplementUsecase) approveAudit(organizationID, actorID uuid.UUID, request *OrderFeeSupplementRequest, evidence *OrderFeeSupplementLockEvidence, computeDetails map[string]string) *AuditEvent {
	details := map[string]string{
		"request.id":          request.ID.String(),
		"request.version":     decimal.NewFromUint64(request.Version).String(),
		"order.id":            request.OrderID.String(),
		"lock_basis":          string(request.LockBasis),
		"business_locked":     boolToFlag(evidence.BusinessLocked),
		"financial_locked":    boolToFlag(evidence.FinancialLocked),
		"fingerprint.version": feeSupplementFingerprintVersion,
		"fingerprint":         request.RequestFingerprint,
	}
	if request.FinancialLockEvidenceVersion != nil {
		details["financial_lock_evidence_version"] = *request.FinancialLockEvidenceVersion
		details["financial_lock_evidence_hash"] = *request.FinancialLockEvidenceHash
		details["financial_lock_net_amount_snapshot"] = request.FinancialLockNetAmount.StringFixed(8)
	}
	for key, value := range computeDetails {
		details[key] = value
	}
	return &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee_supplement.approve",
		Result:         "success",
		ResourceType:   "order_fee_supplement_request",
		ResourceID:     request.ID.String(),
		Details:        details,
	}
}

// Reject 驳回补录申请：申请行锁 + 实时资格，只写申请终态与审计，不产生费用
// 或调整。
func (uc *OrderFeeSupplementUsecase) Reject(ctx context.Context, caller *Principal, organizationID, orderID, requestID uuid.UUID, expectedVersion uint64, reason *string) (*OrderFeeSupplementRequest, error) {
	if !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || requestID == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	normalizedReason := ""
	if reason != nil {
		normalizedReason = strings.TrimSpace(*reason)
		if utf8.RuneCountInString(normalizedReason) > 500 {
			return nil, ErrFeeSupplementInvalidArgument
		}
	}
	if uc.transactor == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	var updated *OrderFeeSupplementRequest
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if _, lockErr := uc.repo.LockOrderForSupplement(txCtx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		request, requestErr := uc.repo.LockForDecision(txCtx, organizationID, requestID)
		if requestErr != nil {
			return requestErr
		}
		if request.OrderID != orderID {
			return ErrFeeSupplementNotFound
		}
		if request.Version != expectedVersion || request.Status != OrderFeeSupplementPending {
			return ErrFeeSupplementTransition
		}
		qualified, grantErr := uc.repo.HasRealtimeLockGrant(txCtx, organizationID, orderID, caller.UserID, caller.IsBootstrapAdmin)
		if grantErr != nil {
			return grantErr
		}
		if !qualified {
			return ErrPermissionDenied
		}
		now := uc.now().UTC()
		request.Status = OrderFeeSupplementRejected
		request.DecidedBy = &caller.UserID
		request.DecidedAt = &now
		if normalizedReason != "" {
			request.DecisionReason = &normalizedReason
		}
		request.Version = request.Version + 1
		if saveErr := uc.repo.SaveDecision(txCtx, request); saveErr != nil {
			return saveErr
		}
		updated = request
		return uc.repo.SaveAudit(txCtx, &AuditEvent{
			OrganizationID: &organizationID,
			UserID:         &caller.UserID,
			Action:         "order.fee_supplement.reject",
			Result:         "success",
			ResourceType:   "order_fee_supplement_request",
			ResourceID:     request.ID.String(),
			Details: map[string]string{
				"request.id":      request.ID.String(),
				"request.version": decimal.NewFromUint64(request.Version).String(),
				"order.id":        request.OrderID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Withdraw 仅发起人可撤回本人仍处于 PENDING 的申请；与审批并发时在同一申请行
// 锁内竞争，只有先提交的一方成功，失败方返回状态冲突；撤回不产生费用或调整。
func (uc *OrderFeeSupplementUsecase) Withdraw(ctx context.Context, caller *Principal, organizationID, orderID, requestID uuid.UUID, expectedVersion uint64) (*OrderFeeSupplementRequest, error) {
	if !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || requestID == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	if uc.transactor == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	var updated *OrderFeeSupplementRequest
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if _, lockErr := uc.repo.LockOrderForSupplement(txCtx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		request, requestErr := uc.repo.LockForDecision(txCtx, organizationID, requestID)
		if requestErr != nil {
			return requestErr
		}
		if request.OrderID != orderID || request.RequestedBy != caller.UserID {
			// 非发起人不可见：稳定返回不存在，不泄露申请事实。
			return ErrFeeSupplementNotFound
		}
		if request.Version != expectedVersion || request.Status != OrderFeeSupplementPending {
			return ErrFeeSupplementTransition
		}
		now := uc.now().UTC()
		request.Status = OrderFeeSupplementWithdrawn
		request.DecidedAt = &now
		request.Version = request.Version + 1
		if saveErr := uc.repo.SaveDecision(txCtx, request); saveErr != nil {
			return saveErr
		}
		updated = request
		return uc.repo.SaveAudit(txCtx, &AuditEvent{
			OrganizationID: &organizationID,
			UserID:         &caller.UserID,
			Action:         "order.fee_supplement.withdraw",
			Result:         "success",
			ResourceType:   "order_fee_supplement_request",
			ResourceID:     request.ID.String(),
			Details: map[string]string{
				"request.id":       request.ID.String(),
				"previous_version": decimal.NewFromUint64(expectedVersion).String(),
				"order.id":         request.OrderID.String(),
				"previous_status":  string(OrderFeeSupplementPending),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// CancelApprovedFee 专用作废已批准补录生成的费用：操作者必须实时具备与补录
// 审批相同的直接解锁资格；事务内固定锁序与全部阻断校验由仓储实现（design 5.1）。
func (uc *OrderFeeSupplementUsecase) CancelApprovedFee(ctx context.Context, caller *Principal, organizationID, orderID, requestID uuid.UUID, feeExpectedVersion uint64, reason string) (*OrderFeeSupplementCancelResult, error) {
	if !caller.CanOperateBusiness() || caller.Organization.ID != organizationID {
		return nil, ErrOperatingCompanyRequired
	}
	if !validSupplementCaller(caller, organizationID, orderID) || requestID == uuid.Nil || feeExpectedVersion == 0 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFeeSupplementInvalidArgument
	}
	if uc.transactor == nil {
		return nil, ErrFeeSupplementInvalidArgument
	}
	var result *OrderFeeSupplementCancelResult
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		cancelResult, cancelErr := uc.repo.CancelApprovedFee(txCtx, &OrderFeeSupplementCancelInput{
			OrganizationID:     organizationID,
			OrderID:            orderID,
			RequestID:          requestID,
			OperatorID:         caller.UserID,
			IsBootstrapAdmin:   caller.IsBootstrapAdmin,
			FeeExpectedVersion: feeExpectedVersion,
			Reason:             reason,
			Audit:              uc.cancelAudit(organizationID, caller.UserID, requestID, orderID, reason),
		})
		if cancelErr != nil {
			return cancelErr
		}
		result = cancelResult
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *OrderFeeSupplementUsecase) cancelAudit(organizationID, operatorID, requestID, orderID uuid.UUID, reason string) *AuditEvent {
	return &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &operatorID,
		Action:         "order.fee_supplement.cancel_fee",
		Result:         "success",
		ResourceType:   "order_fee_supplement_request",
		ResourceID:     requestID.String(),
		Details: map[string]string{
			"request.id": requestID.String(),
			"order.id":   orderID.String(),
			"reason":     reason,
		},
	}
}

// List 订单维度分页读取补录申请，逐行执行「fee.read 或发起人本人或实时
// lock grant」授权，只返回调用人可见的申请及其能力投影，不泄露无权申请。
func (uc *OrderFeeSupplementUsecase) List(ctx context.Context, caller *Principal, organizationID, orderID uuid.UUID, page, pageSize int) (*OrderFeeSupplementListView, error) {
	if !validSupplementCaller(caller, organizationID, orderID) || !ValidListPagination(page, pageSize) {
		return nil, ErrFeeSupplementInvalidArgument
	}
	orderRef, err := uc.repo.GetOrderRef(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	// fee.read 逐行授权：按订单业务类型解析调用人的组织范围；不持有该权限时
	// 仅本人申请或实时具备 lock grant 的行可见。
	hasFeeRead := false
	if businessType := access.OrderBusinessType(orderRef.BusinessType); businessType.Valid() {
		permission := access.OrderPermission(businessType, access.OrderFeeRead)
		hasFeeRead = caller.CanAccessOrganizationForPermission(permission, organizationID, false)
	}
	grant, err := uc.repo.HasRealtimeLockGrant(ctx, organizationID, orderID, caller.UserID, caller.IsBootstrapAdmin)
	if err != nil {
		return nil, err
	}
	rows, err := uc.repo.ListByOrder(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	visible := make([]*OrderFeeSupplementRequest, 0, len(rows))
	for _, row := range rows {
		if hasFeeRead || grant || row.RequestedBy == caller.UserID {
			visible = append(visible, row)
		}
	}
	result := &OrderFeeSupplementListView{Items: make([]*OrderFeeSupplementRequestView, 0, len(visible)), Total: len(visible), Page: page, Size: pageSize}
	start := (page - 1) * pageSize
	if start >= len(visible) {
		return result, nil
	}
	end := start + pageSize
	if end > len(visible) {
		end = len(visible)
	}
	for _, row := range visible[start:end] {
		view := &OrderFeeSupplementRequestView{
			Request:           row,
			CanApprove:        caller.CanOperateBusiness() && caller.Organization.ID == organizationID && row.Status == OrderFeeSupplementPending && grant,
			CanWithdraw:       caller.CanOperateBusiness() && caller.Organization.ID == organizationID && row.Status == OrderFeeSupplementPending && row.RequestedBy == caller.UserID,
			ApproverAvailable: grant,
		}
		if row.Status == OrderFeeSupplementApproved {
			capability, capabilityErr := uc.repo.CancelCapability(ctx, organizationID, orderID, row.ID)
			if capabilityErr != nil {
				return nil, capabilityErr
			}
			view.FeeID = capability.FeeID
			view.FeeStatus = capability.FeeStatus
			view.CanCancel = caller.CanOperateBusiness() && caller.Organization.ID == organizationID && capability.Cancellable && grant
			view.CancelBlockCode = capability.BlockReasonCode
			view.CancelBlockReason = capability.BlockReason
		}
		result.Items = append(result.Items, view)
	}
	return result, nil
}
