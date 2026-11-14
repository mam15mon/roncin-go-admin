package data

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// orderRateDayOf 把订单行投影为 biz.OrderRateDay 所需的最小订单对象。
func orderRateDayOf(o *ent.Order) (string, error) {
	return biz.OrderRateDay(&biz.Order{TradeDirection: biz.OrderTradeDirection(o.TradeDirection), ETD: o.Etd, ETA: o.Eta})
}

// orderRateDayChanged 判断订单从 before 到 after 的费用取汇率日（出口 ETD、进口 ETA 所在业务日）是否变化。
// 任一侧缺失或非法视为变化，让联动在新取值日不可用时显式失败，而不是静默保留旧汇率；
// 两侧都缺失（费用本就无取值日）视为未变化。只改非基准日期，或同一业务日内改时分，均不触发。
func orderRateDayChanged(before, after *ent.Order) bool {
	beforeDay, beforeErr := orderRateDayOf(before)
	afterDay, afterErr := orderRateDayOf(after)
	if beforeErr != nil && afterErr != nil {
		return false
	}
	if beforeErr != nil || afterErr != nil {
		return true
	}
	return beforeDay != afterDay
}

// refreshFeeRatesForOrders 在订单取汇率日变化后，于同一事务内按新取值日重取可更新费用的周汇率。
//
// 前提：调用方已在本事务内按主键升序 FOR UPDATE 锁定 orderIDs 对应订单、通过内容门禁，
// 且订单取汇率日期列已写入。联动只再锁费用行（订单 → 费用行升序），不锁账单、发票、核销与对冲：
// 这些写路径均先锁 Order，持有订单锁期间不会提交新的保护事实（见 research/a1-verification.md）。
//
// 可更新费用：来源不是 MANUAL（SYSTEM/WEEKLY/BOC_SYNC/DERIVED/INHERITED_LAST_WEEK 视为系统汇率）、
// 当前有效（biz.IsCurrentFee）、锁内无受保护财务事实。已建账但无保护事实的费用同样更新，
// 账单行汇率与折本币快照不动，也不改其他字段。
//
// 先解析全部目标汇率，任一缺失立即返回专用错误，此时尚未写入任何费用；调用方须让整个事务回滚，
// 使日期与费用同成败。不需更新的费用不要求其汇率存在。
// 写入：汇率、取值日、来源、设置 ID、折本币金额（原币金额不变）、费用 version+1、
// 订单 fee_content_version+1，随后经 reconcileOrdinaryFeeEconomics 重算提成（保留已付事实）并写审计。
func refreshFeeRatesForOrders(ctx context.Context, tx *ent.Tx, organizationID, actorID uuid.UUID, orderIDs []uuid.UUID) error {
	orderIDs = sortAndDeduplicateUUIDs(orderIDs)
	if len(orderIDs) == 0 {
		return nil
	}
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(organizationID), orderent.IDIn(orderIDs...)).Order(orderent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(orderIDs) {
		return biz.ErrOrderNotFound
	}
	fees, err := tx.OrderFee.Query().
		Where(orderfeeent.OrderIDIn(orderIDs...)).
		Order(orderfeeent.ByOrderID(), orderfeeent.ByID()).
		ForUpdate().
		All(ctx)
	if err != nil {
		return err
	}

	type feeTarget struct {
		fee      *ent.OrderFee
		rateDay  string
		resolved biz.ResolvedRate
	}
	type rateKey struct {
		direction orderfeeent.Direction
		currency  string
		rateDay   string
	}
	cache := make(map[rateKey]biz.ResolvedRate)
	targets := make(map[uuid.UUID][]*feeTarget, len(orders))
	orderByID := make(map[uuid.UUID]*ent.Order, len(orders))
	for _, o := range orders {
		orderByID[o.ID] = o
	}
	// 先对全部可更新费用解析目标汇率，任一缺失即整体失败；此阶段不写任何费用。
	for _, fee := range fees {
		if fee.ExchangeRateSource == orderfeeent.ExchangeRateSourceMANUAL ||
			fee.SupersededByRequestID != nil || fee.EconomicEffect == orderfeeent.EconomicEffectDECREASE {
			continue
		}
		protected, err := feeHasProtectedFinanceFact(ctx, tx.Client(), fee.ID)
		if err != nil {
			return err
		}
		if protected {
			continue
		}
		order := orderByID[fee.OrderID]
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
		rateDay, err := orderRateDayOf(order)
		if err != nil {
			return err
		}
		key := rateKey{direction: fee.Direction, currency: fee.Currency, rateDay: rateDay}
		resolved, ok := cache[key]
		if !ok {
			resolved, err = resolveRateWithClient(ctx, tx.Client(), organizationID, biz.OrderFeeDirection(fee.Direction), fee.Currency, rateDay)
			if err != nil {
				return err
			}
			cache[key] = resolved
		}
		targets[fee.OrderID] = append(targets[fee.OrderID], &feeTarget{fee: fee, rateDay: rateDay, resolved: resolved})
	}

	// 提成经济重算会写入关联父单涉及的全部订单：联动不得在已持订单锁之后再补锁其他订单（会破坏
	// 升序锁序），因此要求提成关联闭包完全落在调用方已锁定的订单集合内，否则 fail-closed 返回冲突。
	if len(targets) > 0 {
		closure, err := commissionEconomicExpandOrderIDs(ctx, tx.Client(), organizationID, orderIDs)
		if err != nil {
			return err
		}
		if len(closure) != len(orderIDs) {
			return biz.ErrOrderStatusConflict
		}
	}

	for _, orderID := range orderIDs {
		var changedFeeIDs []string
		for _, target := range targets[orderID] {
			fee := target.fee
			currentRate, err := decimalOf(fee.ExchangeRate)
			if err != nil {
				return err
			}
			sameSetting := (fee.ExchangeRateSettingID == nil && target.resolved.SettingID == nil) ||
				(fee.ExchangeRateSettingID != nil && target.resolved.SettingID != nil && *fee.ExchangeRateSettingID == *target.resolved.SettingID)
			if currentRate.Equal(target.resolved.Rate) && string(fee.ExchangeRateSource) == target.resolved.Source &&
				fee.ExchangeRateDate == target.rateDay && sameSetting {
				continue
			}
			total, err := decimalOf(fee.TotalAmount)
			if err != nil {
				return err
			}
			update := tx.OrderFee.UpdateOne(fee).
				SetExchangeRate(target.resolved.Rate.StringFixed(8)).
				SetExchangeRateSource(orderfeeent.ExchangeRateSource(target.resolved.Source)).
				SetExchangeRateDate(target.rateDay).
				SetBaseCurrencyAmount(total.Mul(target.resolved.Rate).RoundBank(8).StringFixed(8)).
				SetVersion(fee.Version + 1)
			if target.resolved.SettingID != nil {
				update.SetExchangeRateSettingID(*target.resolved.SettingID)
			} else {
				update.ClearExchangeRateSettingID()
			}
			if err := update.Exec(ctx); err != nil {
				return err
			}
			changedFeeIDs = append(changedFeeIDs, fee.ID.String())
		}
		if len(changedFeeIDs) == 0 {
			continue
		}
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		sort.Strings(changedFeeIDs)
		audit := &biz.AuditEvent{
			OrganizationID: &organizationID,
			UserID:         &actorID,
			Action:         "order.fee.exchange_rate_refresh",
			Result:         "success",
			ResourceType:   "order",
			ResourceID:     orderID.String(),
			Details: map[string]string{
				"order.id":  orderID.String(),
				"order.no":  orderByID[orderID].OrderNo,
				"rate_day":  targets[orderID][0].rateDay,
				"fee.ids":   strings.Join(changedFeeIDs, ","),
				"fee.count": decimal.NewFromInt(int64(len(changedFeeIDs))).String(),
				"trigger":   "ORDER_RATE_DAY_CHANGED",
			},
		}
		if err := reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit); err != nil {
			return err
		}
		if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
			return err
		}
	}
	return nil
}

// syncMemberOrderDatesAndRefreshFeeRates 把共享航程/主单的新 ETD、ETA 写回已被调用方按主键升序锁定的成员订单列
// （订单、费用、报表读同一值；订单 version+1 让打开的编辑表单感知变化），再对取汇率日变化的成员订单
// 逐票联动重取费用汇率。逐票判定：各订单按自身方向取 ETD 或 ETA，未变化的成员不触发。
// 任一成员联动失败（如缺目标周汇率）整体返回错误，调用方事务回滚，主单/航程日期与成员订单列都不落库。
func syncMemberOrderDatesAndRefreshFeeRates(ctx context.Context, tx *ent.Tx, organizationID, actorID uuid.UUID, members []*ent.Order, etd, eta string) error {
	var changed []uuid.UUID
	for _, member := range members {
		if member.Etd == etd && member.Eta == eta {
			continue
		}
		update := tx.Order.UpdateOne(member).SetVersion(member.Version + 1)
		if etd != "" {
			update.SetEtd(etd)
		} else {
			update.ClearEtd()
		}
		if eta != "" {
			update.SetEta(eta)
		} else {
			update.ClearEta()
		}
		updated, err := update.Save(ctx)
		if err != nil {
			return err
		}
		if orderRateDayChanged(member, updated) {
			changed = append(changed, member.ID)
		}
	}
	return refreshFeeRatesForOrders(ctx, tx, organizationID, actorID, changed)
}
