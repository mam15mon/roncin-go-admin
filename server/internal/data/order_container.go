package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	seacargoallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type orderContainerRepo struct {
	data *Data
}

func NewOrderContainerRepo(data *Data) biz.OrderContainerRepo {
	return &orderContainerRepo{data: data}
}

func (r *orderContainerRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderContainerNotFound, nil)
	}
	return item, nil
}

func validateOrderSupportsContainers(item *ent.Order) error {
	if item.ShipmentType == nil || *item.ShipmentType != orderent.ShipmentTypeFCL {
		return biz.ErrOrderContainerShipmentType
	}
	return nil
}

func (r *orderContainerRepo) validateContainerSpec(ctx context.Context, organizationID, specID uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	count, err := client.MasterDataItem.Query().
		Where(
			masterdataitement.IDEQ(specID),
			masterdataitement.OrganizationIDEQ(organizationID),
			masterdataitement.KindEQ(masterdataitement.KindContainerSpec),
			masterdataitement.EnabledEQ(true),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return biz.ErrOrderContainerSpecInvalid
	}
	return nil
}

func (r *orderContainerRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderContainer, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderContainer.Query().
		Where(ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(organizationID)).
		Order(ordercontainerent.ByContainerNo()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderContainer, 0, len(items))
	for _, item := range items {
		result = append(result, orderContainerToBiz(item))
	}
	return result, nil
}

func (r *orderContainerRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderContainer, audit *biz.AuditEvent) (*biz.OrderContainer, error) {
	item, err := r.order(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if err := validateOrderSupportsContainers(item); err != nil {
		return nil, err
	}
	if err := r.validateContainerSpec(ctx, organizationID, input.ContainerSpecID); err != nil {
		return nil, err
	}

	var created *ent.OrderContainer
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁订单
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		if order.BusinessType == orderent.BusinessTypeSE {
			activeLink, linkErr := lockActiveSeaCargoAllocationLink(ctx, tx, organizationID, orderID)
			if linkErr != nil {
				return linkErr
			}
			if activeLink != nil {
				if activeLink.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
					return biz.ErrSeaCargoAllocationStatusConflict
				}
				if activeLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if _, err := tx.SeaMasterBillOrderLink.UpdateOne(activeLink).
						SetCargoAllocationVersion(activeLink.CargoAllocationVersion + 1).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}

		builder := tx.OrderContainer.Create().
			SetID(input.ID).
			SetOrganizationID(organizationID).
			SetOrderID(orderID).
			SetContainerNo(input.ContainerNo).
			SetContainerSpecID(input.ContainerSpecID).
			SetPackageCount(int(input.PackageCount)).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm).
			SetVersion(1)
		if input.SealNo != nil {
			builder.SetSealNo(*input.SealNo)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		var saveErr error
		created, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordercontainer_order_id_container_no", biz.ErrOrderContainerExists)
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["container.version.new"] = "1"
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderContainerToBiz(created), nil
}

func (r *orderContainerRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, input *biz.OrderContainer, audit *biz.AuditEvent) (*biz.OrderContainer, error) {
	item, err := r.order(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if err := validateOrderSupportsContainers(item); err != nil {
		return nil, err
	}
	if err := r.validateContainerSpec(ctx, organizationID, input.ContainerSpecID); err != nil {
		return nil, err
	}

	var updated *ent.OrderContainer
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		if order.BusinessType == orderent.BusinessTypeSE {
			activeLink, linkErr := lockActiveSeaCargoAllocationLink(ctx, tx, organizationID, orderID)
			if linkErr != nil {
				return linkErr
			}
			if activeLink != nil {
				if activeLink.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
					return biz.ErrSeaCargoAllocationStatusConflict
				}

				// 检查草稿分配是否会超出修改后的值
				allocs, aErr := tx.SeaCargoAllocation.Query().
					Where(seacargoallocationent.OrganizationIDEQ(organizationID), seacargoallocationent.OrderIDEQ(orderID), seacargoallocationent.ContainerIDEQ(id)).
					All(ctx)
				if aErr != nil {
					return aErr
				}
				var allocPkg int32
				allocWeight := decimal.Zero
				allocVol := decimal.Zero
				for _, a := range allocs {
					allocPkg += int32(a.PackageCount)
					w, _ := decimal.NewFromString(a.GrossWeightKg)
					v, _ := decimal.NewFromString(a.VolumeCbm)
					allocWeight = allocWeight.Add(w)
					allocVol = allocVol.Add(v)
				}
				if allocPkg > input.PackageCount {
					excess := allocPkg - input.PackageCount
					return biz.NewErrAllocationExceeded(
						"container", id.String(), input.ContainerNo, "package_count",
						fmt.Sprintf("%d", input.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", excess),
						nil, nil, &id,
					)
				}
				newWeight := decimal.NewFromFloat(input.GrossWeightKg)
				if allocWeight.GreaterThan(newWeight) {
					excess := allocWeight.Sub(newWeight)
					return biz.NewErrAllocationExceeded(
						"container", id.String(), input.ContainerNo, "gross_weight_kg",
						newWeight.StringFixed(3), allocWeight.StringFixed(3), excess.StringFixed(3),
						nil, nil, &id,
					)
				}
				newVol := decimal.NewFromFloat(input.VolumeCbm)
				if allocVol.GreaterThan(newVol) {
					excess := allocVol.Sub(newVol)
					return biz.NewErrAllocationExceeded(
						"container", id.String(), input.ContainerNo, "volume_cbm",
						newVol.StringFixed(6), allocVol.StringFixed(6), excess.StringFixed(6),
						nil, nil, &id,
					)
				}

				if activeLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if _, err := tx.SeaMasterBillOrderLink.UpdateOne(activeLink).
						SetCargoAllocationVersion(activeLink.CargoAllocationVersion + 1).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}
		cItem, queryErr := tx.OrderContainer.Query().
			Where(ordercontainerent.IDEQ(id), ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(organizationID)).
			ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		if cItem.Version != expectedVersion {
			return biz.ErrOrderContainerConflict
		}

		builder := tx.OrderContainer.UpdateOne(cItem).
			SetContainerNo(input.ContainerNo).
			SetContainerSpecID(input.ContainerSpecID).
			SetPackageCount(int(input.PackageCount)).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm).
			SetVersion(cItem.Version + 1)
		if input.SealNo != nil {
			builder.SetSealNo(*input.SealNo)
		} else {
			builder.ClearSealNo()
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		} else {
			builder.ClearNote()
		}
		var saveErr error
		updated, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordercontainer_order_id_container_no", biz.ErrOrderContainerExists)
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["container.version.old"] = fmt.Sprintf("%d", cItem.Version)
		audit.Details["container.version.new"] = fmt.Sprintf("%d", cItem.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderContainerToBiz(updated), nil
}

func (r *orderContainerRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		if order.BusinessType == orderent.BusinessTypeSE {
			activeLink, linkErr := lockActiveSeaCargoAllocationLink(ctx, tx, organizationID, orderID)
			if linkErr != nil {
				return linkErr
			}
			if activeLink != nil {
				if activeLink.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
					return biz.ErrSeaCargoAllocationStatusConflict
				}
				hasAlloc, aErr := tx.SeaCargoAllocation.Query().
					Where(seacargoallocationent.OrganizationIDEQ(organizationID), seacargoallocationent.OrderIDEQ(orderID), seacargoallocationent.ContainerIDEQ(id)).
					Exist(ctx)
				if aErr != nil {
					return aErr
				}
				if hasAlloc {
					return biz.ErrSeaCargoAllocationInvalidReference
				}
				if activeLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if _, err := tx.SeaMasterBillOrderLink.UpdateOne(activeLink).
						SetCargoAllocationVersion(activeLink.CargoAllocationVersion + 1).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}
		item, queryErr := tx.OrderContainer.Query().
			Where(ordercontainerent.IDEQ(id), ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(organizationID)).
			ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrOrderContainerConflict
		}

		deleteErr := tx.OrderContainer.DeleteOne(item).Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["container.version.old"] = fmt.Sprintf("%d", item.Version)
		audit.Details["container.version.new"] = "deleted"
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderContainerToBiz(item *ent.OrderContainer) *biz.OrderContainer {
	result := &biz.OrderContainer{
		ID:              item.ID,
		OrganizationID:  item.OrganizationID,
		OrderID:         item.OrderID,
		ContainerNo:     item.ContainerNo,
		ContainerSpecID: item.ContainerSpecID,
		PackageCount:    int32(item.PackageCount),
		GrossWeightKg:   item.GrossWeightKg,
		VolumeCbm:       item.VolumeCbm,
		Version:         item.Version,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
	if item.SealNo != "" {
		v := item.SealNo
		result.SealNo = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}

var _ biz.OrderContainerRepo = (*orderContainerRepo)(nil)
