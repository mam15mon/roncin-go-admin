package data

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybill"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	awcent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airweightconfirmation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	cargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	"github.com/shopspring/decimal"
)

type airWeightRepo struct{ data *Data }

func NewAirWeightRepo(data *Data) biz.AirWeightRepo { return &airWeightRepo{data: data} }

func airWeightPermission(ctx context.Context, org uuid.UUID, write bool) error {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	op := access.OrderRead
	if write {
		op = access.OrderUpdate
	}
	if !principal.CanAccessOrganizationForPermission(access.OrderPermission(access.OrderBusinessAE, op), org, write) {
		return biz.ErrPermissionDenied
	}
	return nil
}
func canonicalAirWeight(value string) string {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return value
	}
	return d.String()
}
func airWeightToBiz(row *ent.AirWeightConfirmation) *biz.AirWeightConfirmation {
	if row == nil {
		return nil
	}
	return &biz.AirWeightConfirmation{ID: row.ID, Scope: string(row.Scope), TargetID: row.TargetID, Source: row.Source, Version: row.Version, InputFingerprint: row.InputFingerprint, ConfirmedWeightKG: canonicalAirWeight(row.ConfirmedWeightKg), Reason: row.Reason, ConfirmedBy: row.ConfirmedBy, ConfirmedAt: row.CreatedAt, Snapshot: row.Snapshot}
}
func airCurrentWeight(ctx context.Context, client *ent.Client, org uuid.UUID, scope string, id uuid.UUID, source string) (*ent.AirWeightConfirmation, error) {
	row, err := client.AirWeightConfirmation.Query().Where(awcent.OrganizationIDEQ(org), awcent.ScopeEQ(awcent.Scope(scope)), awcent.TargetIDEQ(id), awcent.SourceEQ(source)).Order(ent.Desc(awcent.FieldVersion)).First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return row, err
}
func airWeightFacts(ctx context.Context, client *ent.Client, org uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightFacts, error) {
	ids := []uuid.UUID{input.TargetID}
	identity := []any{}
	if input.Scope == "MASTER" {
		master, err := client.AirMasterWaybill.Query().Where(masterent.IDEQ(input.TargetID), masterent.OrganizationIDEQ(org)).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrAirWeightNotFound, nil)
		}
		links, err := client.AirMasterWaybillOrderLink.Query().Where(linkent.OrganizationIDEQ(org), linkent.MasterWaybillIDEQ(input.TargetID), linkent.StatusEQ("ACTIVE")).Order(linkent.ByOrderID()).All(ctx)
		if err != nil {
			return nil, err
		}
		ids = nil
		identity = append(identity, master.ID.String())
		for _, link := range links {
			ids = append(ids, link.OrderID)
			identity = append(identity, link.ID.String(), link.Version)
		}
	} else {
		exists, err := client.Order.Query().Where(orderent.IDEQ(input.TargetID), orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessTypeAE)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, biz.ErrAirWeightNotFound
		}
	}
	facts := &biz.AirWeightFacts{GrossWeightKG: "0", VolumeCBM: "0"}
	if len(ids) == 0 {
		serialized, _ := json.Marshal(identity)
		facts.Identity = string(serialized)
		return facts, nil
	}

	if input.Source == "PLANNED" {
		var rows []struct {
			ID     uuid.UUID `json:"id"`
			Weight *string   `json:"total_gross_weight_kg"`
			Volume *string   `json:"total_volume_cbm"`
		}
		err := client.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.IDIn(ids...), orderent.BusinessTypeEQ(orderent.BusinessTypeAE)).Order(orderent.ByID()).Select(orderent.FieldID, orderent.FieldTotalGrossWeightKg, orderent.FieldTotalVolumeCbm).Scan(ctx, &rows)
		if err != nil {
			return nil, err
		}
		weight, volume := decimal.Zero, decimal.Zero
		complete := len(rows) == len(ids)
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
	// numeric 直接扫描字符串，避免先转 float64 再参与重量计算。
	var rows []struct {
		ID            uuid.UUID `json:"id"`
		OrderID       uuid.UUID `json:"order_id"`
		Version       uint64    `json:"version"`
		GrossWeightKG string    `json:"gross_weight_kg"`
		VolumeCBM     string    `json:"volume_cbm"`
	}
	err := client.OrderCargoItem.Query().Where(cargoent.OrganizationIDEQ(org), cargoent.OrderIDIn(ids...)).Order(cargoent.ByID()).Select(cargoent.FieldID, cargoent.FieldOrderID, cargoent.FieldVersion, cargoent.FieldGrossWeightKg, cargoent.FieldVolumeCbm).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
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
func (r *airWeightRepo) Preview(ctx context.Context, org uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightPreview, error) {
	if err := airWeightPermission(ctx, org, false); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	facts, err := airWeightFacts(ctx, client, org, input)
	if err != nil {
		return nil, err
	}
	row, err := airCurrentWeight(ctx, client, org, input.Scope, input.TargetID, input.Source)
	if err != nil {
		return nil, err
	}
	return biz.BuildAirWeightPreview(input, facts, airWeightToBiz(row))
}
func (r *airWeightRepo) Confirm(ctx context.Context, org, actor uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightConfirmation, error) {
	if err := airWeightPermission(ctx, org, true); err != nil {
		return nil, err
	}
	var result *ent.AirWeightConfirmation
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if input.Scope == "MASTER" {
			locked, err := lockAirMasterMembers(ctx, tx, org, input.TargetID, nil)
			if err != nil {
				return err
			}
			if err := ensureAirMembersEditable(locked); err != nil {
				return err
			}
			for _, member := range locked.members {
				if err := ensureOrderBusinessEditable(ctx, tx, member); err != nil {
					return err
				}
			}
		} else {
			order, err := tx.Order.Query().Where(orderent.IDEQ(input.TargetID), orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessTypeAE)).ForUpdate().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrAirWeightNotFound, nil)
			}
			if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
				return err
			}
		}
		facts, err := airWeightFacts(ctx, tx.Client(), org, input)
		if err != nil {
			return err
		}
		current, err := airCurrentWeight(ctx, tx.Client(), org, input.Scope, input.TargetID, input.Source)
		if err != nil {
			return err
		}
		version := uint64(0)
		if current != nil {
			version = current.Version
		}
		if version != input.ExpectedVersion {
			return biz.ErrAirWeightConflict
		}
		preview, err := biz.BuildAirWeightPreview(input, facts, airWeightToBiz(current))
		if err != nil {
			return err
		}
		if preview.InputFingerprint != input.InputFingerprint {
			return biz.ErrAirWeightConflict
		}
		confirmed, err := decimal.NewFromString(input.ConfirmedWeightKG)
		if err != nil {
			return biz.ErrAirWeightInvalid
		}
		suggestion, _ := decimal.NewFromString(preview.SuggestedChargeableWeightKG)
		if (!preview.Available || !confirmed.Equal(suggestion)) && input.Reason == "" {
			return biz.ErrAirWeightReasonRequired
		}
		result, err = tx.AirWeightConfirmation.Create().SetOrganizationID(org).SetScope(awcent.Scope(input.Scope)).SetTargetID(input.TargetID).SetSource(input.Source).SetVersion(version + 1).SetInputFingerprint(preview.InputFingerprint).SetSnapshot(preview.Snapshot()).SetConfirmedWeightKg(input.ConfirmedWeightKG).SetReason(input.Reason).SetConfirmedBy(actor).Save(ctx)
		if ent.IsConstraintError(err) {
			return biz.ErrAirWeightConflict
		}
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "air.weight.confirm", Result: "success", Details: map[string]string{"scope": input.Scope, "target.id": input.TargetID.String(), "confirmation.id": result.ID.String(), "version": strconv.FormatUint(result.Version, 10), "weight.kg": input.ConfirmedWeightKG, "reason": input.Reason}})
	})
	if err != nil {
		return nil, err
	}
	return airWeightToBiz(result), nil
}
func (r *airWeightRepo) List(ctx context.Context, org uuid.UUID, scope string, id uuid.UUID, page, size int) ([]*biz.AirWeightConfirmation, int, error) {
	if err := airWeightPermission(ctx, org, false); err != nil {
		return nil, 0, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	if _, err := airWeightFacts(ctx, client, org, &biz.AirWeightInput{Scope: scope, TargetID: id}); err != nil {
		return nil, 0, err
	}
	query := client.AirWeightConfirmation.Query().Where(awcent.OrganizationIDEQ(org), awcent.ScopeEQ(awcent.Scope(scope)), awcent.TargetIDEQ(id))
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(ent.Desc(awcent.FieldCreatedAt), ent.Desc(awcent.FieldVersion)).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.AirWeightConfirmation, 0, len(rows))
	for _, row := range rows {
		result = append(result, airWeightToBiz(row))
	}
	return result, total, nil
}

// attachAirMemberWeights 只投影成员本票已确认重量，主单不通过分单相加形成计费重。
func attachAirMemberWeights(ctx context.Context, client *ent.Client, org uuid.UUID, members []*biz.AirMasterMember) error {
	for _, member := range members {
		current, err := airCurrentWeight(ctx, client, org, "ORDER", member.OrderID, "PLANNED")
		if err != nil {
			return err
		}
		if current == nil {
			continue
		}
		input, err := airWeightInputForConfirmation(current)
		if err != nil {
			return err
		}
		facts, err := airWeightFacts(ctx, client, org, input)
		if err != nil {
			return err
		}
		preview, err := biz.BuildAirWeightPreview(input, facts, airWeightToBiz(current))
		if err != nil {
			return err
		}
		member.ConfirmedChargeableWeightKg = current.ConfirmedWeightKg
		member.ConfirmationStale = preview.ConfirmationStale
	}
	return nil
}
