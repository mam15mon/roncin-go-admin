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

func airWeightPermission(ctx context.Context, client *ent.Client, org uuid.UUID, scope string, id uuid.UUID, write bool) error {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	kind, err := airTargetBusinessType(ctx, client, org, scope, id)
	if err != nil {
		return err
	}
	op := access.OrderRead
	if write {
		op = access.OrderUpdate
	}
	if !principal.CanAccessOrganizationForPermission(access.OrderPermission(access.OrderBusinessType(kind), op), org, write) {
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
	kind, err := airTargetBusinessType(ctx, client, org, input.Scope, input.TargetID)
	if err != nil {
		return nil, err
	}
	ids := []uuid.UUID{input.TargetID}
	identity := []any{}
	if input.Scope == "MASTER" {
		master, err := client.AirMasterWaybill.Query().Where(masterent.IDEQ(input.TargetID), masterent.OrganizationIDEQ(org), masterent.BusinessTypeIn("AE", "AI")).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrAirWeightNotFound, nil)
		}
		links, err := client.AirMasterWaybillOrderLink.Query().Where(linkent.MasterWaybillIDEQ(input.TargetID), linkent.StatusEQ("ACTIVE")).Order(linkent.ByOrderID()).All(ctx)
		if err != nil {
			return nil, err
		}
		ids = nil
		identity = append(identity, master.ID.String())
		for _, link := range links {
			if link.OrganizationID != org {
				return nil, biz.ErrAirWaybillConflict
			}
			ids = append(ids, link.OrderID)
			identity = append(identity, link.ID.String(), link.DocumentMode)
		}
	} else {
		exists, err := client.Order.Query().Where(orderent.IDEQ(input.TargetID), orderent.OrganizationIDEQ(org), orderent.BusinessTypeIn(orderent.BusinessTypeAE, orderent.BusinessTypeAI)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, biz.ErrAirWeightNotFound
		}
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessType(kind))).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) != len(ids) {
		return nil, biz.ErrAirWaybillConflict
	}
	for _, o := range orders {
		if kind.IsImport() != (o.TradeDirection == orderent.TradeDirectionImport) {
			return nil, biz.ErrAirWaybillConflict
		}
	}
	facts := &biz.AirWeightFacts{GrossWeightKG: "0", VolumeCBM: "0"}
	if len(ids) == 0 {
		serialized, _ := json.Marshal(identity)
		facts.Identity = string(serialized)
		return facts, nil
	}

	if input.Source == "PLANNED" {
		var rows []airPlannedWeightRow
		err := client.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.IDIn(ids...), orderent.BusinessTypeIn(orderent.BusinessTypeAE, orderent.BusinessTypeAI)).Order(orderent.ByID()).Select(orderent.FieldID, orderent.FieldTotalGrossWeightKg, orderent.FieldTotalVolumeCbm).Scan(ctx, &rows)
		if err != nil {
			return nil, err
		}
		return airPlannedWeightFacts(rows, len(ids), identity)
	}
	// numeric 直接扫描字符串，避免先转 float64 再参与重量计算。
	var rows []airCargoWeightRow
	err = client.OrderCargoItem.Query().Where(cargoent.OrganizationIDEQ(org), cargoent.OrderIDIn(ids...)).Order(cargoent.ByID()).Select(cargoent.FieldID, cargoent.FieldOrderID, cargoent.FieldVersion, cargoent.FieldGrossWeightKg, cargoent.FieldVolumeCbm).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return airCargoWeightFacts(rows, ids, identity)
}
func (r *airWeightRepo) Preview(ctx context.Context, org uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightPreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if err := airWeightPermission(ctx, client, org, input.Scope, input.TargetID, false); err != nil {
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
	var result *ent.AirWeightConfirmation
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := airWeightPermission(ctx, tx.Client(), org, input.Scope, input.TargetID, true); err != nil {
			return err
		}
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
			order, err := tx.Order.Query().Where(orderent.IDEQ(input.TargetID), orderent.OrganizationIDEQ(org), orderent.BusinessTypeIn(orderent.BusinessTypeAE, orderent.BusinessTypeAI)).ForUpdate().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrAirWeightNotFound, nil)
			}
			if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
				return err
			}
		}
		if err := airWeightPermission(ctx, tx.Client(), org, input.Scope, input.TargetID, true); err != nil {
			return err
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
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	if err := airWeightPermission(ctx, client, org, scope, id, false); err != nil {
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
