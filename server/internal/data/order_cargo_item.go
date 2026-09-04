package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	seacargoallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type orderCargoItemRepo struct {
	data *Data
}

func NewOrderCargoItemRepo(data *Data) biz.OrderCargoItemRepo {
	return &orderCargoItemRepo{data: data}
}

func (r *orderCargoItemRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		return mapEntError(err, biz.ErrOrderCargoItemNotFound, nil)
	}
	return nil
}

func (r *orderCargoItemRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderCargoItem.Query().
		Where(ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(organizationID)).
		Order(ordercargoitement.ByCreatedAt(), ordercargoitement.ByCargoName()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderCargoItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderCargoItemToBiz(item))
	}
	return result, nil
}

func (r *orderCargoItemRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderCargoItem, audit *biz.AuditEvent) (*biz.OrderCargoItem, error) {
	var created *ent.OrderCargoItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁订单
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		// 如果是海运单票，检查分配状态门禁
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

		builder := tx.OrderCargoItem.Create().
			SetID(input.ID).
			SetOrganizationID(organizationID).
			SetOrderID(orderID).
			SetCargoName(input.CargoName).
			SetPackageCount(input.PackageCount).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm).
			SetVersion(1)
		if input.NetWeightKg != nil {
			builder.SetNetWeightKg(*input.NetWeightKg)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		var saveErr error
		created, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["cargo_item.version.new"] = "1"
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(created), nil
}

func (r *orderCargoItemRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, input *biz.OrderCargoItem, audit *biz.AuditEvent) (*biz.OrderCargoItem, error) {
	var updated *ent.OrderCargoItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
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
					Where(seacargoallocationent.OrganizationIDEQ(organizationID), seacargoallocationent.OrderIDEQ(orderID), seacargoallocationent.CargoItemIDEQ(id)).
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
				if allocPkg > int32(input.PackageCount) {
					excess := allocPkg - int32(input.PackageCount)
					return biz.NewErrAllocationExceeded(
						"cargo_item", id.String(), input.CargoName, "package_count",
						fmt.Sprintf("%d", input.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", excess),
						&id, nil, nil,
					)
				}
				newWeight := decimal.NewFromFloat(input.GrossWeightKg)
				if allocWeight.GreaterThan(newWeight) {
					excess := allocWeight.Sub(newWeight)
					return biz.NewErrAllocationExceeded(
						"cargo_item", id.String(), input.CargoName, "gross_weight_kg",
						newWeight.StringFixed(3), allocWeight.StringFixed(3), excess.StringFixed(3),
						&id, nil, nil,
					)
				}
				newVol := decimal.NewFromFloat(input.VolumeCbm)
				if allocVol.GreaterThan(newVol) {
					excess := allocVol.Sub(newVol)
					return biz.NewErrAllocationExceeded(
						"cargo_item", id.String(), input.CargoName, "volume_cbm",
						newVol.StringFixed(6), allocVol.StringFixed(6), excess.StringFixed(6),
						&id, nil, nil,
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
		item, queryErr := tx.OrderCargoItem.Query().
			Where(ordercargoitement.IDEQ(id), ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(organizationID)).
			ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrOrderCargoItemConflict
		}

		builder := tx.OrderCargoItem.UpdateOne(item).
			SetCargoName(input.CargoName).
			SetPackageCount(input.PackageCount).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm).
			SetVersion(item.Version + 1)
		if input.NetWeightKg != nil {
			builder.SetNetWeightKg(*input.NetWeightKg)
		} else {
			builder.ClearNetWeightKg()
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		} else {
			builder.ClearNote()
		}
		var saveErr error
		updated, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["cargo_item.version.old"] = fmt.Sprintf("%d", item.Version)
		audit.Details["cargo_item.version.new"] = fmt.Sprintf("%d", item.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(updated), nil
}

func (r *orderCargoItemRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
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
					Where(seacargoallocationent.OrganizationIDEQ(organizationID), seacargoallocationent.OrderIDEQ(orderID), seacargoallocationent.CargoItemIDEQ(id)).
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
		item, queryErr := tx.OrderCargoItem.Query().
			Where(ordercargoitement.IDEQ(id), ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(organizationID)).
			ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrOrderCargoItemConflict
		}

		deleteErr := tx.OrderCargoItem.DeleteOne(item).Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["cargo_item.version.old"] = fmt.Sprintf("%d", item.Version)
		audit.Details["cargo_item.version.new"] = "deleted"
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderCargoItemToBiz(item *ent.OrderCargoItem) *biz.OrderCargoItem {
	result := &biz.OrderCargoItem{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		OrderID:        item.OrderID,
		CargoName:      item.CargoName,
		PackageCount:   item.PackageCount,
		GrossWeightKg:  item.GrossWeightKg,
		VolumeCbm:      item.VolumeCbm,
		Version:        item.Version,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.NetWeightKg > 0 {
		v := item.NetWeightKg
		result.NetWeightKg = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}

var _ biz.OrderCargoItemRepo = (*orderCargoItemRepo)(nil)
