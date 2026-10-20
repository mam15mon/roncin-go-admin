package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	cargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	containerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	requestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	sharedallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
	"github.com/shopspring/decimal"
)

func (r *orderFeeRepo) QuantitySource(ctx context.Context, organizationID, orderID, billingUnitID uuid.UUID, source string) (*biz.FeeQuantitySource, error) {
	// 先在组织谓词内确认订单，单位和所有子资源只在授权订单上下文读取。
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := client.BillingUnit.Query().Where(billingunitent.IDEQ(billingUnitID)).Only(ctx)
	if ent.IsNotFound(err) {
		return &biz.FeeQuantitySource{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := &biz.FeeQuantitySource{UnitCode: unit.Code, UnitEnabled: unit.Enabled, QuantityMustBeInteger: unit.QuantityMustBeInteger, ContainerUnit: unit.IsContainerUnit}
	if !unit.Enabled {
		return result, nil
	}
	// 在数据库 numeric 上聚合并直接扫描十进制字符串；不能先加载 Ent float64 再求和。
	var totals []struct {
		Count  int     `json:"count"`
		Weight *string `json:"weight"`
		Volume *string `json:"volume"`
	}
	err = client.OrderCargoItem.Query().Where(cargoent.OrganizationIDEQ(organizationID), cargoent.OrderIDEQ(orderID)).Aggregate(
		ent.As(ent.Count(), "count"), ent.As(ent.Sum(cargoent.FieldGrossWeightKg), "weight"), ent.As(ent.Sum(cargoent.FieldVolumeCbm), "volume"),
	).Scan(ctx, &totals)
	if err != nil {
		return nil, err
	}
	if len(totals) > 0 {
		result.CargoCount = totals[0].Count
		if totals[0].Weight != nil {
			result.GrossWeightKG, err = decimal.NewFromString(*totals[0].Weight)
			if err != nil {
				return nil, fmt.Errorf("读取订单毛重: %w", err)
			}
		}
		if totals[0].Volume != nil {
			result.VolumeCBM, err = decimal.NewFromString(*totals[0].Volume)
			if err != nil {
				return nil, fmt.Errorf("读取订单体积: %w", err)
			}
		}
	}
	if !unit.IsContainerUnit && unit.Code != "CONT" {
		return result, nil
	}
	var specID uuid.UUID
	if unit.Code != "CONT" {
		spec, err := client.MasterDataItem.Query().Where(masterdataent.KindEQ(masterdataent.KindContainerSpec), masterdataent.CodeEQ(unit.Code)).Only(ctx)
		if ent.IsNotFound(err) {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		specID, result.ContainerSpecMatched = spec.ID, true
	}
	if source == "PLANNED" {
		query := client.OrderContainerRequest.Query().Where(requestent.OrderIDEQ(orderID))
		if specID != uuid.Nil {
			query.Where(requestent.ContainerSpecIDEQ(specID))
		}
		requests, err := query.All(ctx)
		if err != nil {
			return nil, err
		}
		for _, request := range requests {
			result.ContainerQuantity += request.Quantity
		}
	} else {
		// 共享箱归属不是整箱所有权，禁止将参与箱数直接当作本票独占箱量。
		result.HasSharedContainers, err = client.SeaSharedContainerAllocation.Query().Where(sharedallocationent.OrganizationIDEQ(organizationID), sharedallocationent.OrderIDEQ(orderID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		query := client.OrderContainer.Query().Where(containerent.OrganizationIDEQ(organizationID), containerent.OrderIDEQ(orderID))
		if specID != uuid.Nil {
			query.Where(containerent.ContainerSpecIDEQ(specID))
		}
		result.ContainerQuantity, err = query.Count(ctx)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
