package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

// Save 单事务整表保存单侧费用的新增与修改。
//
// 顺序：订单行锁（含审核锁与业务门禁）→ 目标费用行按主键升序 FOR UPDATE → 新增行幂等键重读 →
// 回放判定 → 逐行完整校验（全部通过才写）→ 手工汇率币种集合按「提交后整单集合」事务内重读判定 →
// 写入 → 订单费用内容版本只 +1 一次 → 提成经济重算只调用一次 → 逐行审计。
// 任一行失败整体回滚；回调内的业务错误原样外传。
func (r *orderFeeRepo) Save(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderFeeSaveInput, audits map[string]*biz.AuditEvent) (*biz.OrderFeeSaveResult, error) {
	var feeContentVersion uint64
	replayed := false
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}

		updateIDs := make([]uuid.UUID, 0, len(input.Rows))
		addKeys := make([]string, 0, len(input.Rows))
		for _, row := range input.Rows {
			if row.FeeID != nil {
				updateIDs = append(updateIDs, *row.FeeID)
			} else {
				addKeys = append(addKeys, row.Fee.IdempotencyKey)
			}
		}

		locked := make(map[uuid.UUID]*ent.OrderFee, len(updateIDs))
		if len(updateIDs) > 0 {
			items, queryErr := tx.OrderFee.Query().
				Where(orderfeeent.OrderIDEQ(orderID), orderfeeent.IDIn(updateIDs...)).
				WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).
				Order(orderfeeent.ByID()).ForUpdate().All(ctx)
			if queryErr != nil {
				return queryErr
			}
			for _, item := range items {
				locked[item.ID] = item
			}
		}
		existingAdds := make(map[string]*ent.OrderFee, len(addKeys))
		if len(addKeys) > 0 {
			items, queryErr := tx.OrderFee.Query().
				Where(orderfeeent.OrderIDEQ(orderID), orderfeeent.IdempotencyKeyIn(addKeys...)).
				WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).All(ctx)
			if queryErr != nil {
				return queryErr
			}
			for _, item := range items {
				existingAdds[item.IdempotencyKey] = item
			}
		}

		// 回放与逐行状态复核：applied 表示该行已处于本次提交的目标内容，fresh 表示可正常写入。
		var failures []biz.OrderFeeSaveRowFailure
		fail := func(clientRowID string, err error) error {
			failure, isRowFailure := biz.NewOrderFeeSaveRowFailure(clientRowID, err)
			if !isRowFailure {
				return err
			}
			failures = append(failures, failure)
			return nil
		}
		applied := 0
		for _, row := range input.Rows {
			if row.FeeID == nil {
				item, exists := existingAdds[row.Fee.IdempotencyKey]
				if !exists {
					continue
				}
				existing, convErr := orderFeeToBiz(item)
				if convErr != nil {
					return convErr
				}
				if biz.SameOrderFeeIntent(existing, row.Fee) {
					applied++
					continue
				}
				if err := fail(row.ClientRowID, biz.ErrOrderFeeIdempotencyConflict); err != nil {
					return err
				}
				continue
			}
			item, found := locked[*row.FeeID]
			if !found {
				if err := fail(row.ClientRowID, biz.ErrOrderFeeNotFound); err != nil {
					return err
				}
				continue
			}
			existing, convErr := orderFeeToBiz(item)
			if convErr != nil {
				return convErr
			}
			if item.Version == row.ExpectedVersion+1 && biz.SameOrderFeeIntent(existing, row.Fee) {
				applied++
				continue
			}
			var rowErr error
			switch {
			case item.SupersededByRequestID != nil || item.EconomicEffect == orderfeeent.EconomicEffectDECREASE:
				rowErr = biz.ErrFeeCorrectionProtected
			case string(item.Direction) != string(row.Fee.Direction):
				rowErr = biz.ErrOrderFeeInvalidArgument
			case len(item.Edges.FinanceBillLines) > 0:
				rowErr = biz.ErrOrderFeeBillOccupied
			case item.Version != row.ExpectedVersion:
				rowErr = biz.ErrOrderFeeVersionConflict
			}
			if rowErr != nil {
				if err := fail(row.ClientRowID, rowErr); err != nil {
					return err
				}
			}
		}
		if applied > 0 && applied == len(input.Rows) {
			replayed = true
			order, getErr := tx.Order.Get(ctx, orderID)
			if getErr != nil {
				return getErr
			}
			feeContentVersion = order.FeeContentVersion
			return nil
		}
		if applied > 0 {
			// 部分行已处于目标内容：有具体行冲突（意图不一致、版本变化等）时报该行，否则报半应用。
			if len(failures) > 0 {
				return biz.NewOrderFeeSaveFailure(failures)
			}
			return biz.ErrOrderFeeSavePartiallyApplied
		}

		// 引用校验：在锁内用 tx 读取，通过状态复核的行才继续（避免对已失败行重复报错）。
		failedRows := make(map[string]struct{}, len(failures))
		for _, failure := range failures {
			failedRows[failure.ClientRowID] = struct{}{}
		}
		for _, row := range input.Rows {
			if _, bad := failedRows[row.ClientRowID]; bad {
				continue
			}
			var refErr error
			if row.FeeID == nil {
				refErr = validateSaveAddRefs(ctx, tx, organizationID, orderID, row.Fee)
			} else {
				refErr = validateSaveUpdateRefs(ctx, tx, organizationID, orderID, locked[*row.FeeID], row.Fee)
			}
			if refErr != nil {
				if err := fail(row.ClientRowID, refErr); err != nil {
					return err
				}
				failedRows[row.ClientRowID] = struct{}{}
			}
		}

		// 手工汇率覆盖：币种集合按「本次提交后的整单费用集合」在事务内重读判定。
		hasOverride := false
		for _, row := range input.Rows {
			if row.ExchangeRateOverridden {
				hasOverride = true
				break
			}
		}
		if hasOverride {
			currentItems, queryErr := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(orderID), currentOrderFeePredicate()).All(ctx)
			if queryErr != nil {
				return queryErr
			}
			existing := make([]*biz.OrderFee, 0, len(currentItems))
			for _, item := range currentItems {
				existing = append(existing, &biz.OrderFee{ID: item.ID, Direction: biz.OrderFeeDirection(item.Direction), Currency: item.Currency})
			}
			submitted := make([]*biz.OrderFee, 0, len(input.Rows))
			for _, row := range input.Rows {
				id := row.Fee.ID
				if row.FeeID != nil {
					id = *row.FeeID
				}
				submitted = append(submitted, &biz.OrderFee{ID: id, Direction: row.Fee.Direction, Currency: row.Fee.Currency})
			}
			if !biz.OrderFeeCurrencySetsMatchAfterSave(existing, submitted) {
				for _, row := range input.Rows {
					if !row.ExchangeRateOverridden {
						continue
					}
					if _, bad := failedRows[row.ClientRowID]; bad {
						continue
					}
					if err := fail(row.ClientRowID, biz.ErrOrderFeeExchangeRateOverrideMismatch); err != nil {
						return err
					}
				}
			}
		}
		if len(failures) > 0 {
			return biz.NewOrderFeeSaveFailure(failures)
		}

		// 全部校验通过后才写入：先更新（锁已按主键升序取得），再新增。
		for _, row := range input.Rows {
			if row.FeeID != nil {
				if _, saveErr := newOrderFeeUpdate(tx, locked[*row.FeeID], row.Fee).Save(ctx); saveErr != nil {
					return saveErr
				}
				continue
			}
			if _, saveErr := newOrderFeeCreate(tx, orderID, row.Fee).Save(ctx); saveErr != nil {
				return mapEntConstraint(saveErr, "orderfee_order_id_idempotency_key", biz.ErrOrderFeeIdempotencyConflict)
			}
		}
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		for _, row := range input.Rows {
			attachFeeQuantityAudit(audits[row.ClientRowID], row.Fee.QuantitySource)
		}
		if err := reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audits[input.Rows[0].ClientRowID]); err != nil {
			return err
		}
		for _, row := range input.Rows {
			if err := writeAudit(ctx, tx.AuditLog, audits[row.ClientRowID]); err != nil {
				return err
			}
		}
		order, getErr := tx.Order.Get(ctx, orderID)
		if getErr != nil {
			return getErr
		}
		feeContentVersion = order.FeeContentVersion
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 事务提交后以普通上下文重读完整业务响应（含逐费能力），再取本侧费用。
	all, listErr := r.List(ctx, organizationID, orderID)
	if listErr != nil {
		return nil, listErr
	}
	fees := make([]*biz.OrderFee, 0, len(all))
	for _, fee := range all {
		if fee.Direction == input.Direction {
			fees = append(fees, fee)
		}
	}
	return &biz.OrderFeeSaveResult{Fees: fees, FeeContentVersion: feeContentVersion, Replayed: replayed}, nil
}

// validateSaveAddRefs 在订单锁内复核新增行引用的计费单位、结算单位与币种，口径同 Add。
func validateSaveAddRefs(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, fee *biz.OrderFee) error {
	if fee.BillingUnitID == nil {
		return biz.ErrOrderFeeBillingUnitInvalid
	}
	unit, unitErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(*fee.BillingUnitID)).ForShare().Only(ctx)
	if unitErr != nil {
		return mapEntError(unitErr, biz.ErrOrderFeeBillingUnitInvalid, nil)
	}
	if !unit.Enabled {
		return biz.ErrOrderFeeBillingUnitInvalid
	}
	if err := validateFeeQuantitySelection(ctx, tx.Client(), organizationID, orderID, fee, unit); err != nil {
		return err
	}
	if err := biz.ValidateFeeQuantityForUnit(fee.Quantity, unit.QuantityMustBeInteger); err != nil {
		return err
	}
	return validateSavePartyAndCurrency(ctx, tx, organizationID, fee)
}

// validateSaveUpdateRefs 复核修改行引用，口径同 Update：数量来源未变化时保留历史快照。
func validateSaveUpdateRefs(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, item *ent.OrderFee, fee *biz.OrderFee) error {
	if fee.BillingUnitID == nil {
		return biz.ErrOrderFeeBillingUnitInvalid
	}
	unit, unitErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(*fee.BillingUnitID)).ForShare().Only(ctx)
	if unitErr != nil {
		return mapEntError(unitErr, biz.ErrOrderFeeBillingUnitInvalid, nil)
	}
	existingSelection, selectionErr := feeQuantitySelectionFromMap(item.QuantitySource)
	if selectionErr != nil {
		return selectionErr
	}
	existingQuantity, parseErr := decimalOf(item.Quantity)
	if parseErr != nil {
		return parseErr
	}
	unchangedUnit := item.BillingUnitID != nil && *item.BillingUnitID == *fee.BillingUnitID
	if existingQuantity.Equal(fee.Quantity) && unchangedUnit && biz.FeeQuantitySelectionsEqual(existingSelection, fee.QuantitySource) {
		fee.QuantitySource = existingSelection
	} else if err := validateFeeQuantitySelection(ctx, tx.Client(), organizationID, orderID, fee, unit); err != nil {
		return err
	}
	if !existingQuantity.Equal(fee.Quantity) || !unchangedUnit {
		if err := biz.ValidateFeeQuantityForUnit(fee.Quantity, unit.QuantityMustBeInteger); err != nil {
			return err
		}
	}
	return validateSavePartyAndCurrency(ctx, tx, organizationID, fee)
}

func validateSavePartyAndCurrency(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, fee *biz.OrderFee) error {
	_, queryErr := tx.Partner.Query().Where(partnerent.IDEQ(fee.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
	if queryErr != nil {
		return mapEntError(queryErr, biz.ErrOrderFeePartyInvalid, nil)
	}
	validCurrency, currencyErr := tx.Currency.Query().Where(currencyent.CodeEQ(fee.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
	if currencyErr != nil {
		return currencyErr
	}
	if !validCurrency {
		return biz.ErrOrderFeeCurrencyInvalid
	}
	return nil
}
