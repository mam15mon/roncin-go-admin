package biz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FeeQuantitySource 保留当前订单来源数据；货量直接承接数据库 numeric 聚合，避免浮点换算。
type FeeQuantitySource struct {
	UnitCode              string
	UnitEnabled           bool
	QuantityMustBeInteger bool
	ContainerUnit         bool
	ContainerSpecMatched  bool
	ContainerQuantity     int
	HasSharedContainers   bool
	CargoCount            int
	GrossWeightKG         decimal.Decimal
	VolumeCBM             decimal.Decimal
}

type FeeQuantityInput struct{ Label, Value, Unit string }
type FeeQuantityPreview struct {
	Available   bool
	Quantity    string
	SourceLabel string
	Formula     string
	Inputs      []FeeQuantityInput
	Reason      string
}

func (uc *OrderFeeUsecase) PreviewFeeQuantity(ctx context.Context, organizationID, orderID, billingUnitID uuid.UUID, source string) (*FeeQuantityPreview, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || billingUnitID == uuid.Nil || (source != "PLANNED" && source != "ACTUAL") {
		return nil, ErrOrderFeeInvalidArgument
	}
	data, err := uc.repo.QuantitySource(ctx, organizationID, orderID, billingUnitID, source)
	if err != nil {
		return nil, err
	}
	return buildFeeQuantityPreview(data, source), nil
}

func buildFeeQuantityPreview(data *FeeQuantitySource, source string) *FeeQuantityPreview {
	result := &FeeQuantityPreview{SourceLabel: "本订单实际货物明细", Inputs: []FeeQuantityInput{}}
	if data == nil || !data.UnitEnabled {
		result.Reason = "计费单位不存在或已停用，请选择有效单位"
		return result
	}
	var quantity decimal.Decimal
	switch data.UnitCode {
	case "CBM", "KG", "TON", "RT":
		if source != "ACTUAL" {
			result.Reason = "货量仅支持本订单实际货物明细，请选择实际登记来源"
			return result
		}
		if data.CargoCount == 0 {
			result.Reason = "本订单尚未登记货物明细，请先补充货物数据或手工录入数量"
			return result
		}
		weight, volume := data.GrossWeightKG, data.VolumeCBM
		switch data.UnitCode {
		case "CBM":
			quantity = volume
			result.Inputs = []FeeQuantityInput{{"总体积", volume.String(), "CBM"}}
			result.Formula = fmt.Sprintf("总体积 = %s CBM", volume.String())
		case "KG":
			quantity = weight
			result.Inputs = []FeeQuantityInput{{"总毛重", weight.String(), "KG"}}
			result.Formula = fmt.Sprintf("总毛重 = %s KG（非空运计费重）", weight.String())
		case "TON":
			quantity = weight.Div(decimal.NewFromInt(1000))
			result.Inputs = []FeeQuantityInput{{"总毛重", weight.String(), "KG"}}
			result.Formula = fmt.Sprintf("总毛重 %s KG ÷ 1000 = %s 吨", weight.String(), quantity.String())
		case "RT":
			tons := weight.Div(decimal.NewFromInt(1000))
			quantity = decimal.Max(volume, tons)
			result.Inputs = []FeeQuantityInput{{"总体积", volume.String(), "CBM"}, {"总毛重", weight.String(), "KG"}}
			result.Formula = fmt.Sprintf("按 1 CBM : 1000 KG，max(%s CBM, %s KG ÷ 1000) = %s RT；请确认该费用适用此规则", volume.String(), weight.String(), quantity.String())
		}
	default:
		if !data.ContainerUnit && data.UnitCode != "CONT" {
			result.Reason = "该单位暂不支持从订单计算数量，请手工录入"
			return result
		}
		result.SourceLabel = "本订单实际登记箱量"
		if source == "PLANNED" {
			result.SourceLabel = "本订单订舱计划箱量"
		}
		if data.UnitCode != "CONT" && !data.ContainerSpecMatched {
			result.Reason = "计费单位代码无法精确对应标准箱型，请核对箱型或手工录入数量"
			return result
		}
		if source == "ACTUAL" && data.HasSharedContainers {
			result.Reason = "本订单关联共享箱，整箱数量无法准确归属当前订单，请按费用约定手工录入"
			return result
		}
		quantity = decimal.NewFromInt(int64(data.ContainerQuantity))
		label := "自然箱"
		if data.UnitCode != "CONT" {
			label = data.UnitCode
		}
		result.Inputs = []FeeQuantityInput{{label + "箱量", quantity.String(), "箱"}}
		result.Formula = fmt.Sprintf("%s：%s × %s 箱", result.SourceLabel, label, quantity.String())
	}
	if !quantity.IsPositive() {
		result.Reason = "所选来源没有可用的正数数量，请补充数据或手工录入"
		return result
	}
	if !quantity.Equal(quantity.Truncate(4)) {
		result.Reason = "计算结果超过费用数量允许的四位小数，请按业务约定手工确定数量；系统不会自动舍入"
		return result
	}
	if !quantityOrPricePattern.MatchString(quantity.String()) {
		result.Reason = "计算结果超过费用数量上限，请核对来源或手工录入"
		return result
	}
	if ValidateFeeQuantityForUnit(quantity, data.QuantityMustBeInteger) != nil {
		result.Reason = "当前单位要求整数，但来源计算结果含小数，请按业务约定手工确定数量"
		return result
	}
	result.Available = true
	result.Quantity = quantity.String()
	return result
}
