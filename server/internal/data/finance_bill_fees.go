package data

import (
	"context"
	"sort"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// ListBillFeeCandidates 返回可加入账单的未建账费用。SQL 条件与 biz.ValidateBillFeeForAttach 同口径：
// 同公司、需要建账、当前有效（未被替代、经济效果为 INCREASE，故收付方向即费用方向）、无有效账单占用、
// 同结算单位、同币种与本币、更正来源与账单一致，并排除待审更正的源费用。关键字覆盖费用代码、中英文名称、拼音与订单号，别名由所属公司的费用设置补充。
func (r *financeBillRepo) ListBillFeeCandidates(ctx context.Context, bill *biz.FinanceBill, filter biz.FinanceBillFeeCandidateFilter) (*biz.FinanceBillCreationCandidateResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.OrderFee{
		orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(bill.OrganizationID)),
		orderfeeent.AccountingRequiredEQ(true),
		orderfeeent.SupersededByRequestIDIsNil(),
		orderfeeent.EconomicEffectEQ(orderfeeent.EconomicEffectINCREASE),
		orderfeeent.DirectionEQ(orderfeeent.Direction(bill.Direction)),
		orderfeeent.SettlementPartyIDEQ(bill.SettlementPartyID),
		orderfeeent.CurrencyEQ(bill.Currency),
		orderfeeent.BaseCurrencyEQ(bill.BaseCurrency),
		orderfeeent.Not(orderfeeent.HasFinanceBillLinesWith(effectiveBillLinePredicate())),
		orderfeeent.Not(orderfeeent.HasTargetedCorrectionRequestsWith(correctionent.StatusEQ(correctionent.StatusPENDING))),
	}
	if bill.CorrectionRequestID == nil {
		predicates = append(predicates, orderfeeent.SourceFeeIDIsNil())
	} else {
		predicates = append(predicates, orderfeeent.SourceFeeIDNotNil(), orderfeeent.CorrectionRequestIDEQ(*bill.CorrectionRequestID))
	}
	if filter.Keyword != "" {
		predicates = append(predicates, financeBillFeeKeywordPredicate(bill.OrganizationID, filter.Keyword))
	}
	query := client.OrderFee.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.WithSettlementParty().WithOrder().WithFinanceBillLines(effectiveBillLineFilter).
		Order(orderfeeent.ByCreatedAt(entsql.OrderDesc()), orderfeeent.ByID()).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBillCreationCandidateResult{Items: make([]*biz.FinanceBillableFee, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		converted, convertErr := financeBillableFeeFromEnt(item)
		if convertErr != nil {
			return nil, convertErr
		}
		converted.CanBill = true
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

// financeBillFeeKeywordPredicate 共用于建账与账单追加候选：按费用名称快照检索，
// 同公司费目仅补足别名，不能代替更正费用的自定义名称或借其他公司的字典扩大匹配。
func financeBillFeeKeywordPredicate(organizationID uuid.UUID, keyword string) predicate.OrderFee {
	return orderfeeent.Or(
		orderfeeent.FeeCodeContainsFold(keyword),
		orderfeeent.FeeNameContainsFold(keyword),
		orderfeeent.FeeNameEnContainsFold(keyword),
		orderfeeent.SearchKeywordsContainsFold(keyword),
		orderfeeent.HasOrderWith(orderent.OrderNoContainsFold(keyword)),
		orderfeeent.HasFeeSettingWith(
			feesettingent.OrganizationIDEQ(organizationID),
			feesettingent.Or(feesettingent.AliasNameContainsFold(keyword), feesettingent.SearchKeywordsContainsFold(keyword)),
		),
	)
}

func financeBillableFeeFromEnt(item *ent.OrderFee) (*biz.FinanceBillableFee, error) {
	fee, err := orderFeeToBiz(item)
	if err != nil {
		return nil, err
	}
	businessOrder, err := item.Edges.OrderOrErr()
	if err != nil {
		return nil, err
	}
	isCasual := false
	if party, partyErr := item.Edges.SettlementPartyOrErr(); partyErr == nil && party != nil {
		isCasual = party.IsCasual
	}
	return &biz.FinanceBillableFee{OrderLocked: businessOrder.LockedAt != nil, Fee: fee, OrganizationID: businessOrder.OrganizationID, OrderNo: businessOrder.OrderNo, BusinessType: string(businessOrder.BusinessType), SettlementPartyIsCasual: isCasual}, nil
}

// UpdateFees 为账单增加与移除费用，固定锁序（design 3.2）：
//  1. 无锁预读账单有效行所属订单与增删费用所属订单，取并集；
//  2. lockBillEconomicOrders 展开提成父订单闭包、升序 FOR UPDATE 并复核闭包；
//  3. 账单 FOR UPDATE，校验版本与状态，复核有效行集合未漂移；
//  4. 锁内读取费用锁定事实（发票、核销、已确认对冲、草稿对冲），命中即拒绝；
//  5. 增删费用按主键升序 FOR UPDATE，再锁账单有效行；
//  6. 逐项复核后写入：停用移除行、插入新行、按账单汇率重算头与行、fee_count、预计开票金额、账单与费用 version+1；
//  7. 提成经济重算与审计。
//
// 与发票、核销、对冲确认、取消账单、费用汇率联动等写路径一样先锁订单再锁账单，不引入反向等待。
func (r *financeBillRepo) UpdateFees(ctx context.Context, organizationIDs []uuid.UUID, input biz.UpdateFinanceBillFeesInput, refresh func(context.Context, *biz.FinanceBill) error, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		head, err := client.FinanceBill.Query().Where(financebillent.IDEQ(input.ID), financeBillOrganizationScopePredicate(organizationIDs)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		organizationID := head.OrganizationID
		changedFeeIDs := uniqueSortedUUIDs(append(append([]uuid.UUID{}, input.AddFeeIDs...), input.RemoveFeeIDs...))
		preLines, err := client.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(input.ID), financebilllineent.ActiveEQ(true)).All(ctx)
		if err != nil {
			return err
		}
		preFees, err := client.OrderFee.Query().Where(orderfeeent.IDIn(changedFeeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).All(ctx)
		if err != nil {
			return err
		}
		// 跨公司或不存在的费用在预读阶段即不可见，按费用不合法拒绝，不进入加锁。
		if len(preFees) != len(changedFeeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		seeds := make([]uuid.UUID, 0, len(preLines)+len(preFees))
		for _, line := range preLines {
			seeds = append(seeds, line.OrderID)
		}
		feeOrders := make(map[uuid.UUID]uuid.UUID, len(preFees))
		for _, fee := range preFees {
			seeds = append(seeds, fee.OrderID)
			feeOrders[fee.ID] = fee.OrderID
		}
		if err := lockBillEconomicOrders(ctx, client, []uuid.UUID{organizationID}, uniqueSortedUUIDs(seeds)); err != nil {
			return err
		}
		item, err := client.FinanceBill.Query().Where(financebillent.IDEQ(input.ID), financebillent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		if item.Version != input.ExpectedVersion {
			return biz.ErrFinanceBillVersionConflict
		}
		if item.Status != financebillent.StatusCONFIRMED {
			return biz.ErrFinanceBillInvalidTransition
		}
		facts, err := billFeeLockFacts(ctx, client, item.ID)
		if err != nil {
			return err
		}
		if len(facts) > 0 {
			return biz.FinanceBillFeesLockedError(facts)
		}
		fees, err := client.OrderFee.Query().
			Where(orderfeeent.IDIn(changedFeeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).
			Order(orderfeeent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(fees) != len(changedFeeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		for _, fee := range fees {
			// 加锁后费用所属订单不得变化，否则所持订单锁不能覆盖该费用。
			if feeOrders[fee.ID] != fee.OrderID {
				return biz.ErrFinanceBillVersionConflict
			}
		}
		lines, err := client.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(item.ID), financebilllineent.ActiveEQ(true)).
			WithOrder().WithOrderFee().Order(financebilllineent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if !sameBillLineOrders(preLines, lines) {
			return biz.ErrFinanceBillVersionConflict
		}
		item.Edges.Lines = lines
		bill, err := financeBillToBiz(item)
		if err != nil {
			return err
		}
		// 账单行快照按创建时间排序决定尾差归属；新行追加在末尾。
		sort.SliceStable(lines, func(i, j int) bool {
			if lines[i].CreatedAt.Equal(lines[j].CreatedAt) {
				return lines[i].ID.String() < lines[j].ID.String()
			}
			return lines[i].CreatedAt.Before(lines[j].CreatedAt)
		})
		removeSet := make(map[uuid.UUID]bool, len(input.RemoveFeeIDs))
		for _, id := range input.RemoveFeeIDs {
			removeSet[id] = true
		}
		lineFeeIDs := make([]uuid.UUID, 0, len(lines))
		removedLineIDs := make([]uuid.UUID, 0, len(input.RemoveFeeIDs))
		kept := make([]*biz.FinanceBillLine, 0, len(lines)+len(input.AddFeeIDs))
		storedLines := make(map[uuid.UUID]struct{}, len(lines))
		for _, line := range lines {
			lineFeeIDs = append(lineFeeIDs, line.OrderFeeID)
			if removeSet[line.OrderFeeID] {
				removedLineIDs = append(removedLineIDs, line.ID)
				continue
			}
			converted, convertErr := financeBillLineToBiz(line)
			if convertErr != nil {
				return convertErr
			}
			storedLines[line.ID] = struct{}{}
			kept = append(kept, converted)
		}
		// 移除目标必须是本账单当前有效行。
		if len(removedLineIDs) != len(input.RemoveFeeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		// 账单含已被更正替代的来源费用时属于受保护历史，与取消、更新口径一致。
		if superseded, queryErr := client.OrderFee.Query().Where(orderfeeent.IDIn(lineFeeIDs...), orderfeeent.SupersededByRequestIDNotNil()).Exist(ctx); queryErr != nil {
			return queryErr
		} else if superseded {
			return biz.ErrFeeCorrectionProtected
		}
		if len(input.AddFeeIDs) > 0 {
			added, addErr := loadBillFeeAdditions(ctx, client, organizationID, input.AddFeeIDs)
			if addErr != nil {
				return addErr
			}
			for _, candidate := range added {
				if validationErr := biz.ValidateBillFeeForAttach(bill, candidate); validationErr != nil {
					return validationErr
				}
				kept = append(kept, biz.NewFinanceBillLineFromFee(bill.ID, candidate))
			}
		}
		if len(kept) == 0 {
			return biz.ErrFinanceBillLastFee
		}
		if refresh == nil {
			return biz.ErrFinanceBillInvalidArgument
		}
		bill.Lines = kept
		if err := refresh(ctx, bill); err != nil {
			return err
		}
		if err := biz.RecomputeFinanceBillAmounts(bill, kept); err != nil {
			return err
		}
		newVersion := item.Version + 1
		if len(removedLineIDs) > 0 {
			affected, updateErr := client.FinanceBillLine.Update().Where(financebilllineent.IDIn(removedLineIDs...), financebilllineent.ActiveEQ(true)).SetActive(false).Save(ctx)
			if updateErr != nil {
				return updateErr
			}
			if affected != len(removedLineIDs) {
				return biz.ErrFinanceBillVersionConflict
			}
		}
		builders := make([]*ent.FinanceBillLineCreate, 0, len(input.AddFeeIDs))
		for _, line := range bill.Lines {
			if _, exists := storedLines[line.ID]; exists {
				if _, updateErr := client.FinanceBillLine.UpdateOneID(line.ID).SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).Save(ctx); updateErr != nil {
					return updateErr
				}
				continue
			}
			builders = append(builders, client.FinanceBillLine.Create().
				SetID(line.ID).SetBillID(bill.ID).SetOrderFeeID(line.OrderFeeID).SetOrderID(line.OrderID).
				SetOrderNo(line.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).SetQuantity(line.Quantity.StringFixed(4)).SetUnitPrice(line.UnitPrice.StringFixed(4)).
				SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetNillableTaxRate(financeDecimalString(line.TaxRate, 4)).SetCurrency(line.Currency).
				SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrency(line.BaseCurrency).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).
				SetActive(true))
		}
		if len(builders) > 0 {
			if _, createErr := client.FinanceBillLine.CreateBulk(builders...).Save(ctx); createErr != nil {
				// 有效费用部分唯一索引兜底：同一费用不能同时处于两张账单的有效行。
				return mapEntError(createErr, nil, biz.ErrFinanceBillFeeInvalid)
			}
		}
		update := client.FinanceBill.UpdateOneID(bill.ID).
			SetTotalAmount(bill.TotalAmount.StringFixed(8)).SetNetAmount(bill.NetAmount.StringFixed(8)).SetTaxAmount(bill.TaxAmount.StringFixed(8)).
			SetBaseCurrencyAmount(bill.BaseCurrencyAmount.StringFixed(8)).SetFeeCount(bill.FeeCount).
			SetBillDate(bill.BillDate).SetNillableDueDate(bill.DueDate).
			SetExchangeRate(bill.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(bill.ExchangeRateSource)).
			SetExchangeRateDate(bill.ExchangeRateDate).SetNillableExchangeRateSettingID(bill.ExchangeRateSettingID).SetVersion(newVersion)
		if bill.EstimatedInvoiceAmount != nil {
			update.SetEstimatedInvoiceAmount(bill.EstimatedInvoiceAmount.StringFixed(8))
		}
		if _, err := update.Save(ctx); err != nil {
			return err
		}
		affected, err := client.OrderFee.Update().Where(orderfeeent.IDIn(changedFeeIDs...)).AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if affected != len(changedFeeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		orderIDs := make([]uuid.UUID, 0, len(lines)+len(fees))
		for _, line := range lines {
			orderIDs = append(orderIDs, line.OrderID)
		}
		for _, fee := range fees {
			orderIDs = append(orderIDs, fee.OrderID)
		}
		if err := reconcileBillEconomics(ctx, client, organizationID, orderIDs, billEconomicActor(audit), biz.CommissionEconomicTrigger{Kind: "BILL_FEE_UPDATE", ID: bill.ID, Version: newVersion}); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// loadBillFeeAdditions 在费用行已锁定后读取待加入费用的权威事实：有效账单占用以显式查询复核
// （与建账同口径），待审更正源费用拒绝。
func loadBillFeeAdditions(ctx context.Context, client *ent.Client, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	items, err := client.OrderFee.Query().
		Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).
		WithSettlementParty().WithOrder().WithFinanceBillLines(effectiveBillLineFilter).Order(orderfeeent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(feeIDs) {
		return nil, biz.ErrFinanceBillFeeInvalid
	}
	occupied, err := client.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDIn(feeIDs...), effectiveBillLinePredicate()).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if occupied {
		return nil, biz.ErrFinanceBillFeeInvalid
	}
	pending, err := pendingCorrectionSourceFees(ctx, client, feeIDs)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FinanceBillableFee, 0, len(items))
	for _, item := range items {
		if pending[item.ID] {
			return nil, biz.ErrFeeCorrectionPending
		}
		converted, convertErr := financeBillableFeeFromEnt(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result = append(result, converted)
	}
	return result, nil
}

// sameBillLineOrders 判断加锁前后账单有效行集合及其订单归属是否一致。
func sameBillLineOrders(before, after []*ent.FinanceBillLine) bool {
	if len(before) != len(after) {
		return false
	}
	original := make(map[uuid.UUID]uuid.UUID, len(before))
	for _, line := range before {
		original[line.ID] = line.OrderID
	}
	for _, line := range after {
		orderID, exists := original[line.ID]
		if !exists || orderID != line.OrderID {
			return false
		}
	}
	return true
}
