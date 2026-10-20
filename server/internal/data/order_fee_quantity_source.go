package data

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	awcent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airweightconfirmation"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	"github.com/shopspring/decimal"
)

func feeQuantitySelectionMap(s *biz.FeeQuantitySelection) map[string]string {
	if s == nil {
		return map[string]string{"mode": "MANUAL"}
	}
	result := map[string]string{"mode": s.Mode, "reason": s.Reason, "adopted_quantity": s.AdoptedQuantity, "unit": s.Unit, "reference_quantity": s.ReferenceQuantity, "source": s.Source, "confirmed_at": s.ConfirmedAt}
	if s.ConfirmationID != uuid.Nil {
		result["confirmation_id"] = s.ConfirmationID.String()
		result["confirmation_version"] = strconv.FormatUint(s.ConfirmationVersion, 10)
	}
	return result
}
func feeQuantitySelectionFromMap(data map[string]string) (*biz.FeeQuantitySelection, error) {
	if len(data) == 0 {
		return &biz.FeeQuantitySelection{Mode: "MANUAL"}, nil
	}
	result := &biz.FeeQuantitySelection{Mode: data["mode"], Reason: data["reason"], AdoptedQuantity: data["adopted_quantity"], Unit: data["unit"], ReferenceQuantity: data["reference_quantity"], Source: data["source"], ConfirmedAt: data["confirmed_at"]}
	if data["confirmation_id"] != "" {
		id, err := uuid.Parse(data["confirmation_id"])
		if err != nil {
			return nil, err
		}
		result.ConfirmationID = id
		version, err := strconv.ParseUint(data["confirmation_version"], 10, 64)
		if err != nil {
			return nil, err
		}
		result.ConfirmationVersion = version
	}
	return result, nil
}
func airWeightInputForConfirmation(c *ent.AirWeightConfirmation) (*biz.AirWeightInput, error) {
	n, err := strconv.ParseInt(c.Snapshot["bubble_share_numerator"], 10, 32)
	if err != nil {
		return nil, err
	}
	d, err := strconv.ParseInt(c.Snapshot["bubble_share_denominator"], 10, 32)
	if err != nil {
		return nil, err
	}
	return &biz.AirWeightInput{Scope: string(c.Scope), TargetID: c.TargetID, Source: c.Source, VolumetricDivisor: c.Snapshot["volumetric_divisor"], BubbleShareNumerator: int32(n), BubbleShareDenominator: int32(d)}, nil
}

// validateFeeQuantitySelection 在订单已加 FOR UPDATE 后执行，与确认/货量更新串行。
func validateFeeQuantitySelection(ctx context.Context, client *ent.Client, org, orderID uuid.UUID, fee *biz.OrderFee, unit *ent.BillingUnit) error {
	selection, err := biz.NormalizeFeeQuantitySelection(fee.QuantitySource)
	if err != nil {
		return err
	}
	selection.AdoptedQuantity = fee.Quantity.String()
	selection.Unit = unit.Code
	if selection.Mode == "MANUAL" {
		fee.QuantitySource = selection
		return nil
	}
	if unit.Code != "KG" {
		return biz.ErrAirWeightInvalid
	}
	row, err := client.AirWeightConfirmation.Query().Where(awcent.IDEQ(selection.ConfirmationID), awcent.OrganizationIDEQ(org), awcent.ScopeEQ(awcent.ScopeORDER), awcent.TargetIDEQ(orderID), awcent.VersionEQ(selection.ConfirmationVersion)).Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrAirWeightNotFound, nil)
	}
	weight, err := decimal.NewFromString(row.ConfirmedWeightKg)
	if err != nil {
		return err
	}
	selection.ReferenceQuantity = weight.String()
	selection.Source = row.Source
	selection.ConfirmedAt = row.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	if selection.Mode == "DIRECT" {
		current, err := airCurrentWeight(ctx, client, org, "ORDER", orderID, row.Source)
		if err != nil {
			return err
		}
		if current == nil || current.ID != row.ID {
			return biz.ErrAirWeightConflict
		}
		input, err := airWeightInputForConfirmation(row)
		if err != nil {
			return err
		}
		facts, err := airWeightFacts(ctx, client, org, input)
		if err != nil {
			return err
		}
		preview, err := biz.BuildAirWeightPreview(input, facts, airWeightToBiz(row))
		if err != nil {
			return err
		}
		if preview.ConfirmationStale || !fee.Quantity.Equal(weight) {
			return biz.ErrAirWeightConflict
		}
	}
	fee.QuantitySource = selection
	return nil
}
func (r *orderFeeRepo) ConfirmedQuantityPreview(ctx context.Context, org, orderID, unitID uuid.UUID, source string) (*biz.FeeQuantityPreview, error) {
	if err := r.order(ctx, org, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FeeQuantityPreview{SourceLabel: map[string]string{"PLANNED": "本票计划货量的确认计费重", "ACTUAL": "本票实际货物的确认计费重"}[source], Inputs: []biz.FeeQuantityInput{}}
	unit, err := client.BillingUnit.Query().Where(billingunitent.IDEQ(unitID)).Only(ctx)
	if ent.IsNotFound(err) {
		result.Reason = "计费单位不存在"
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if !unit.Enabled || unit.Code != "KG" || (source != "ACTUAL" && source != "PLANNED") {
		result.Reason = "本票确认计费重仅支持明确的计划/实际来源及启用的 KG 单位"
		return result, nil
	}
	current, err := airCurrentWeight(ctx, client, org, "ORDER", orderID, source)
	if err != nil {
		return nil, err
	}
	if current == nil {
		result.Reason = "本票尚未确认计费重，不会回退为毛重"
		return result, nil
	}
	input, err := airWeightInputForConfirmation(current)
	if err != nil {
		return nil, err
	}
	facts, err := airWeightFacts(ctx, client, org, input)
	if err != nil {
		return nil, err
	}
	preview, err := biz.BuildAirWeightPreview(input, facts, airWeightToBiz(current))
	if err != nil {
		return nil, err
	}
	if preview.ConfirmationStale {
		result.Reason = "确认计费重的依据已变化，请重新确认后带入"
		return result, nil
	}
	quantity, err := decimal.NewFromString(current.ConfirmedWeightKg)
	if err != nil {
		return nil, err
	}
	if err := biz.ValidateFeeQuantityForUnit(quantity, unit.QuantityMustBeInteger); err != nil {
		result.Reason = "当前计费单位要求整数，请手工确定数量"
		return result, nil
	}
	result.Available = true
	result.Quantity = quantity.String()
	result.Formula = "采用本票当前已确认计费重（KG），不包含整主单重量"
	result.Inputs = []biz.FeeQuantityInput{{Label: "确认计费重", Value: quantity.String(), Unit: "KG"}, {Label: "确认版本", Value: strconv.FormatUint(current.Version, 10), Unit: ""}}
	result.QuantitySource = &biz.FeeQuantitySelection{Mode: "DIRECT", ConfirmationID: current.ID, ConfirmationVersion: current.Version, AdoptedQuantity: quantity.String(), Unit: "KG", ReferenceQuantity: quantity.String(), Source: source, ConfirmedAt: current.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
	return result, nil
}

func attachFeeQuantityAudit(audit *biz.AuditEvent, s *biz.FeeQuantitySelection) {
	if audit == nil || s == nil {
		return
	}
	if audit.Details == nil {
		audit.Details = map[string]string{}
	}
	for key, value := range feeQuantitySelectionMap(s) {
		audit.Details["fee.quantity_source."+key] = value
	}
}
