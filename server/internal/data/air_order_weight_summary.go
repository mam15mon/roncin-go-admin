package data

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	awcent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airweightconfirmation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	cargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type airPlannedWeightRow struct {
	ID     uuid.UUID `json:"id"`
	Weight *string   `json:"total_gross_weight_kg"`
	Volume *string   `json:"total_volume_cbm"`
}
type airCargoWeightRow struct {
	ID            uuid.UUID `json:"id"`
	OrderID       uuid.UUID `json:"order_id"`
	Version       uint64    `json:"version"`
	GrossWeightKG string    `json:"gross_weight_kg"`
	VolumeCBM     string    `json:"volume_cbm"`
}

// 两种读取路径共用货量与指纹构造，列表不能另算过期规则。
func airPlannedWeightFacts(rows []airPlannedWeightRow, expected int, identity []any) (*biz.AirWeightFacts, error) {
	facts := &biz.AirWeightFacts{}
	weight, volume := decimal.Zero, decimal.Zero
	complete := len(rows) == expected
	for _, row := range rows {
		identity = append(identity, row.ID.String(), row.Weight, row.Volume)
		if row.Weight == nil || row.Volume == nil {
			complete = false
			continue
		}
		g, err := decimal.NewFromString(*row.Weight)
		if err != nil {
			return nil, err
		}
		v, err := decimal.NewFromString(*row.Volume)
		if err != nil {
			return nil, err
		}
		if !g.IsPositive() || !v.IsPositive() {
			complete = false
		}
		weight = weight.Add(g)
		volume = volume.Add(v)
	}
	facts.HasCargo = complete
	facts.GrossWeightKG = weight.String()
	facts.VolumeCBM = volume.String()
	serialized, _ := json.Marshal(identity)
	facts.Identity = string(serialized)
	return facts, nil
}
func airCargoWeightFacts(rows []airCargoWeightRow, ids []uuid.UUID, identity []any) (*biz.AirWeightFacts, error) {
	facts := &biz.AirWeightFacts{}
	weight, volume := decimal.Zero, decimal.Zero
	coveredOrders := map[uuid.UUID]bool{}
	for _, row := range rows {
		coveredOrders[row.OrderID] = true
		g, e := decimal.NewFromString(row.GrossWeightKG)
		if e != nil {
			return nil, e
		}
		v, e := decimal.NewFromString(row.VolumeCBM)
		if e != nil {
			return nil, e
		}
		weight = weight.Add(g)
		volume = volume.Add(v)
		identity = append(identity, row.ID.String(), row.Version, row.GrossWeightKG, row.VolumeCBM)
	}
	facts.HasCargo = len(rows) > 0 && len(coveredOrders) == len(ids)
	facts.GrossWeightKG = weight.String()
	facts.VolumeCBM = volume.String()
	serialized, _ := json.Marshal(identity)
	facts.Identity = string(serialized)
	return facts, nil
}

// attachAirOrderWeightSummaries 对已授权订单按公司批量投影两个独立来源的最新确认。
// 摘要不选择费用采用值；ORDER/MASTER 与 PLANNED/ACTUAL 不相互替代。
func attachAirOrderWeightSummaries(ctx context.Context, client *ent.Client, orders []*biz.Order) error {
	byOrganization := map[uuid.UUID][]*biz.Order{}
	for _, order := range orders {
		if order.AirExport != nil {
			byOrganization[order.OrganizationID] = append(byOrganization[order.OrganizationID], order)
		}
	}
	for organizationID, rows := range byOrganization {
		if err := attachOrganizationAirOrderWeights(ctx, client, organizationID, rows); err != nil {
			return err
		}
	}
	return nil
}

func attachOrganizationAirOrderWeights(ctx context.Context, client *ent.Client, organizationID uuid.UUID, orders []*biz.Order) error {
	ids := make([]uuid.UUID, 0, len(orders))
	byID := map[uuid.UUID]*biz.Order{}
	for _, order := range orders {
		ids = append(ids, order.ID)
		byID[order.ID] = order
	}
	var latest []struct {
		TargetID uuid.UUID `json:"target_id"`
		Source   string    `json:"source"`
		Version  uint64    `json:"version"`
	}
	err := client.AirWeightConfirmation.Query().Where(awcent.OrganizationIDEQ(organizationID), awcent.ScopeEQ(awcent.ScopeORDER), awcent.TargetIDIn(ids...), awcent.SourceIn("PLANNED", "ACTUAL")).GroupBy(awcent.FieldTargetID, awcent.FieldSource).Aggregate(ent.As(ent.Max(awcent.FieldVersion), "version")).Scan(ctx, &latest)
	if err != nil || len(latest) == 0 {
		return err
	}
	predicates := make([]predicate.AirWeightConfirmation, 0, len(latest))
	for _, row := range latest {
		predicates = append(predicates, awcent.And(awcent.TargetIDEQ(row.TargetID), awcent.SourceEQ(row.Source), awcent.VersionEQ(row.Version)))
	}
	confirmations, err := client.AirWeightConfirmation.Query().Where(awcent.OrganizationIDEQ(organizationID), awcent.ScopeEQ(awcent.ScopeORDER), awcent.Or(predicates...)).Order(awcent.BySource()).All(ctx)
	if err != nil {
		return err
	}
	var planned []airPlannedWeightRow
	if err := client.Order.Query().Where(orderent.OrganizationIDEQ(organizationID), orderent.IDIn(ids...)).Order(orderent.ByID()).Select(orderent.FieldID, orderent.FieldTotalGrossWeightKg, orderent.FieldTotalVolumeCbm).Scan(ctx, &planned); err != nil {
		return err
	}
	planByID := map[uuid.UUID]airPlannedWeightRow{}
	for _, row := range planned {
		planByID[row.ID] = row
	}
	var cargo []airCargoWeightRow
	if err := client.OrderCargoItem.Query().Where(cargoent.OrganizationIDEQ(organizationID), cargoent.OrderIDIn(ids...)).Order(cargoent.ByID()).Select(cargoent.FieldID, cargoent.FieldOrderID, cargoent.FieldVersion, cargoent.FieldGrossWeightKg, cargoent.FieldVolumeCbm).Scan(ctx, &cargo); err != nil {
		return err
	}
	cargoByOrder := map[uuid.UUID][]airCargoWeightRow{}
	for _, row := range cargo {
		cargoByOrder[row.OrderID] = append(cargoByOrder[row.OrderID], row)
	}
	for _, confirmation := range confirmations {
		order := byID[confirmation.TargetID]
		if order == nil {
			return biz.ErrAirWaybillConflict
		}
		input, err := airWeightInputForConfirmation(confirmation)
		if err != nil {
			return err
		}
		var facts *biz.AirWeightFacts
		if input.Source == "PLANNED" {
			row, exists := planByID[order.ID]
			if !exists {
				return biz.ErrAirWaybillConflict
			}
			facts, err = airPlannedWeightFacts([]airPlannedWeightRow{row}, 1, []any{})
		} else {
			facts, err = airCargoWeightFacts(cargoByOrder[order.ID], []uuid.UUID{order.ID}, []any{})
		}
		if err != nil {
			return err
		}
		current := airWeightToBiz(confirmation)
		preview, err := biz.BuildAirWeightPreview(input, facts, current)
		if err != nil {
			return err
		}
		order.AirExport.WeightSummaries = append(order.AirExport.WeightSummaries, &biz.AirOrderWeightSummary{Source: confirmation.Source, ConfirmedWeightKG: current.ConfirmedWeightKG, ConfirmationStale: preview.ConfirmationStale, ConfirmedAt: current.ConfirmedAt})
	}
	return nil
}
