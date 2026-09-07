package data

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

type seaSharedContainerRepo struct{ data *Data }

func NewSeaSharedContainerRepo(data *Data) biz.SeaSharedContainerRepo {
	return &seaSharedContainerRepo{data: data}
}

func (r *seaSharedContainerRepo) List(ctx context.Context, organizationID, executionID uuid.UUID, keyword string, page, pageSize int) ([]*biz.SeaSharedContainer, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	if err := validateSharedExecution(ctx, client, organizationID, executionID, false); err != nil {
		return nil, 0, err
	}
	query := client.SeaSharedContainer.Query().Where(
		seasharedcontainerent.OrganizationIDEQ(organizationID),
		seasharedcontainerent.TransportExecutionIDEQ(executionID),
	)
	if keyword != "" {
		query.Where(seasharedcontainerent.Or(
			seasharedcontainerent.ContainerNoContainsFold(keyword),
			seasharedcontainerent.SealNoContainsFold(keyword),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(seasharedcontainerent.ByContainerNo(), seasharedcontainerent.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*biz.SeaSharedContainer, 0, len(rows))
	for _, row := range rows {
		item, err := r.getByEntity(ctx, client, row)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *seaSharedContainerRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.SeaSharedContainer, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	row, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
	}
	return r.getByEntity(ctx, client, row)
}

func (r *seaSharedContainerRepo) ListCandidates(ctx context.Context, organizationID, executionID uuid.UUID, keyword string, page, pageSize int) ([]*biz.SeaSharedContainerCandidateOrder, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	if err := validateSharedExecution(ctx, client, organizationID, executionID, false); err != nil {
		return nil, 0, err
	}
	query := client.Order.Query().Where(
		orderent.OrganizationIDEQ(organizationID),
		orderent.BusinessTypeEQ(orderent.BusinessTypeSE),
		orderent.HasSeaMasterBillLinksWith(
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.TransportExecutionIDEQ(executionID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			seamasterbillorderlinkent.DocumentStructureEQ(seamasterbillorderlinkent.DocumentStructureHOUSE),
		),
	)
	if keyword != "" {
		query.Where(orderent.Or(
			orderent.OrderNoContainsFold(keyword),
			orderent.CustomerReferenceNoContainsFold(keyword),
			orderent.BookingNoContainsFold(keyword),
			orderent.HasSeaHouseBillsWith(
				seahousebillent.OrganizationIDEQ(organizationID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
				seahousebillent.HouseNoContainsFold(keyword),
			),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	orders, err := query.Order(orderent.ByOrderNo(), orderent.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.SeaSharedContainerCandidateOrder, 0, len(orders))
	for _, order := range orders {
		link, err := client.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.OrderIDEQ(order.ID),
			seamasterbillorderlinkent.TransportExecutionIDEQ(executionID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			seamasterbillorderlinkent.DocumentStructureEQ(seamasterbillorderlinkent.DocumentStructureHOUSE),
		).Only(ctx)
		if err != nil {
			return nil, 0, err
		}
		houseBill, err := client.SeaHouseBill.Query().Where(
			seahousebillent.OrganizationIDEQ(organizationID),
			seahousebillent.OrderIDEQ(order.ID),
			seahousebillent.MasterBillIDEQ(link.MasterBillID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
		).Only(ctx)
		if err != nil {
			return nil, 0, mapEntError(err, biz.ErrSeaSharedContainerInvalidReference, nil)
		}
		cargoRows, err := client.OrderCargoItem.Query().Where(ordercargoitement.OrganizationIDEQ(organizationID), ordercargoitement.OrderIDEQ(order.ID)).Order(ordercargoitement.ByID()).All(ctx)
		if err != nil {
			return nil, 0, err
		}
		candidate := &biz.SeaSharedContainerCandidateOrder{OrderID: order.ID, OrderNo: order.OrderNo, HouseBillID: houseBill.ID, HouseNo: houseBill.HouseNo, OrderVersion: order.Version, LinkVersion: link.Version, HouseBillVersion: houseBill.Version}
		for _, cargo := range cargoRows {
			candidate.CargoItems = append(candidate.CargoItems, &biz.SeaSharedContainerCandidateCargoItem{ID: cargo.ID, CargoName: cargo.CargoName, PackageCount: int32(cargo.PackageCount), GrossWeightKg: decimal.NewFromFloat(cargo.GrossWeightKg), VolumeCbm: decimal.NewFromFloat(cargo.VolumeCbm), Version: cargo.Version})
		}
		result = append(result, candidate)
	}
	return result, total, nil
}

func (r *seaSharedContainerRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.SeaSharedContainer, audit *biz.AuditEvent) (*biz.SeaSharedContainer, error) {
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		if err := validateSharedExecution(txCtx, client, organizationID, input.TransportExecutionID, true); err != nil {
			return err
		}
		if err := validateExecutionHasHouseOrders(txCtx, client, organizationID, input.TransportExecutionID); err != nil {
			return err
		}
		if err := validateSharedContainerSpec(txCtx, client, organizationID, input.ContainerSpecID); err != nil {
			return err
		}
		_, err = client.SeaSharedContainer.Create().SetID(input.ID).SetOrganizationID(organizationID).SetTransportExecutionID(input.TransportExecutionID).SetContainerNo(input.ContainerNo).SetContainerSpecID(input.ContainerSpecID).SetNillableSealNo(input.SealNo).SetPackageCount(int(input.PackageCount)).SetGrossWeightKg(input.GrossWeightKg.StringFixed(3)).SetVolumeCbm(input.VolumeCbm.StringFixed(6)).SetStatus(seasharedcontainerent.StatusDRAFT).SetNillableNote(input.Note).SetVersion(1).Save(txCtx)
		if err != nil {
			return mapEntConstraint(err, "sea_shared_container_execution_no", biz.ErrSeaSharedContainerExists)
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, input.ID)
}

func (r *seaSharedContainerRepo) Update(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, input *biz.SeaSharedContainer, audit *biz.AuditEvent) (*biz.SeaSharedContainer, error) {
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		located, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
		}
		executionIDs := sortedUUIDs([]uuid.UUID{located.TransportExecutionID, input.TransportExecutionID})
		executions, err := client.SeaTransportExecution.Query().Where(seatransportexecutionent.IDIn(executionIDs...), seatransportexecutionent.OrganizationIDEQ(organizationID)).Order(seatransportexecutionent.ByID()).ForUpdate().All(txCtx)
		if err != nil {
			return err
		}
		if len(executions) != len(executionIDs) {
			return biz.ErrSeaSharedContainerInvalidReference
		}
		if err := validateExecutionHasHouseOrders(txCtx, client, organizationID, input.TransportExecutionID); err != nil {
			return err
		}
		if err := validateSharedContainerSpec(txCtx, client, organizationID, input.ContainerSpecID); err != nil {
			return err
		}
		container, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
		}
		if container.Version != expectedVersion {
			return biz.ErrSeaSharedContainerConflict
		}
		if container.Status != seasharedcontainerent.StatusDRAFT {
			return biz.ErrSeaSharedContainerStatusConflict
		}
		allocations, err := client.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.SharedContainerIDEQ(id), seasharedcontainerallocationent.OrganizationIDEQ(organizationID)).Order(seasharedcontainerallocationent.ByID()).ForUpdate().All(txCtx)
		if err != nil {
			return err
		}
		if len(allocations) > 0 && container.TransportExecutionID != input.TransportExecutionID {
			return biz.ErrSeaSharedContainerInvalidReference
		}
		allocated := biz.SeaSharedQuantity{}
		for _, allocation := range allocations {
			weight, err := decimal.NewFromString(allocation.GrossWeightKg)
			if err != nil {
				return err
			}
			volume, err := decimal.NewFromString(allocation.VolumeCbm)
			if err != nil {
				return err
			}
			allocated = allocated.Add(biz.SeaSharedQuantity{PackageCount: int32(allocation.PackageCount), GrossWeightKg: weight, VolumeCbm: volume})
		}
		if allocated.Exceeds(biz.SeaSharedQuantity{PackageCount: input.PackageCount, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}) {
			return biz.ErrSeaSharedContainerExceeded
		}
		_, err = client.SeaSharedContainer.UpdateOne(container).SetTransportExecutionID(input.TransportExecutionID).SetContainerNo(input.ContainerNo).SetContainerSpecID(input.ContainerSpecID).ClearSealNo().SetNillableSealNo(input.SealNo).SetPackageCount(int(input.PackageCount)).SetGrossWeightKg(input.GrossWeightKg.StringFixed(3)).SetVolumeCbm(input.VolumeCbm.StringFixed(6)).ClearNote().SetNillableNote(input.Note).SetVersion(container.Version + 1).Save(txCtx)
		if err != nil {
			return mapEntConstraint(err, "sea_shared_container_execution_no", biz.ErrSeaSharedContainerExists)
		}
		addVersionAudit(audit, container.Version, container.Version+1)
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *seaSharedContainerRepo) Delete(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) error {
	return r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		container, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
		}
		if container.Version != expectedVersion {
			return biz.ErrSeaSharedContainerConflict
		}
		if container.Status != seasharedcontainerent.StatusDRAFT {
			return biz.ErrSeaSharedContainerStatusConflict
		}
		allocations, err := client.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.SharedContainerIDEQ(id), seasharedcontainerallocationent.OrganizationIDEQ(organizationID)).Order(seasharedcontainerallocationent.ByID()).ForUpdate().All(txCtx)
		if err != nil {
			return err
		}
		if len(allocations) > 0 {
			return biz.ErrSeaSharedContainerStatusConflict
		}
		if err := client.SeaSharedContainer.DeleteOne(container).Exec(txCtx); err != nil {
			return err
		}
		addVersionAudit(audit, container.Version, 0)
		return writeAudit(txCtx, client.AuditLog, audit)
	})
}

func (r *seaSharedContainerRepo) SaveDraft(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, inputs []*biz.SeaSharedContainerAllocationInput, audit *biz.AuditEvent) (*biz.SeaSharedContainer, error) {
	err := r.mutateAllocations(ctx, organizationID, id, expectedVersion, inputs, false, uuid.Nil, audit)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

// Confirm 确认共享箱：inputs 非 nil 时按该输入在同一事务内保存并严格守恒确认，
// 为 nil 时按当前已保存分配确认。
func (r *seaSharedContainerRepo) Confirm(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, inputs []*biz.SeaSharedContainerAllocationInput, audit *biz.AuditEvent) (*biz.SeaSharedContainer, error) {
	if inputs == nil {
		current, err := r.Get(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
		inputs = make([]*biz.SeaSharedContainerAllocationInput, 0, len(current.Allocations))
		for _, allocation := range current.Allocations {
			// Confirm 会在事务内重新读取并锁定真实版本；零值表示不信任外部缓存，只接受当前值。
			inputs = append(inputs, &biz.SeaSharedContainerAllocationInput{OrderID: allocation.OrderID, HouseBillID: allocation.HouseBillID, CargoItemID: allocation.CargoItemID, PackageCount: allocation.PackageCount, GrossWeightKg: allocation.GrossWeightKg, VolumeCbm: allocation.VolumeCbm})
		}
	}
	if err := r.mutateAllocations(ctx, organizationID, id, expectedVersion, inputs, true, actorID, audit); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *seaSharedContainerRepo) Withdraw(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) (*biz.SeaSharedContainer, error) {
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		container, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
		}
		if container.Version != expectedVersion {
			return biz.ErrSeaSharedContainerConflict
		}
		if container.Status != seasharedcontainerent.StatusCONFIRMED {
			return biz.ErrSeaSharedContainerStatusConflict
		}
		_, err = client.SeaSharedContainer.UpdateOne(container).SetStatus(seasharedcontainerent.StatusDRAFT).ClearConfirmedAt().ClearConfirmedBy().SetVersion(container.Version + 1).Save(txCtx)
		if err != nil {
			return err
		}
		addVersionAudit(audit, container.Version, container.Version+1)
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *seaSharedContainerRepo) mutateAllocations(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, inputs []*biz.SeaSharedContainerAllocationInput, confirm bool, actorID uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		return r.data.WithTx(txCtx, func(tx *ent.Tx) error {
			client := tx.Client()
			located, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(id), seasharedcontainerent.OrganizationIDEQ(organizationID)).Only(txCtx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaSharedContainerNotFound, nil)
			}
			existingTargetAllocations, err := client.SeaSharedContainerAllocation.Query().Where(
				seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
				seasharedcontainerallocationent.SharedContainerIDEQ(id),
			).All(txCtx)
			if err != nil {
				return err
			}
			orderIDs, cargoIDs, houseBillIDs := allocationReferenceIDs(inputs)
			for _, allocation := range existingTargetAllocations {
				orderIDs = append(orderIDs, allocation.OrderID)
				cargoIDs = append(cargoIDs, allocation.CargoItemID)
				houseBillIDs = append(houseBillIDs, allocation.HouseBillID)
			}
			orderIDs, cargoIDs, houseBillIDs = sortedUUIDs(orderIDs), sortedUUIDs(cargoIDs), sortedUUIDs(houseBillIDs)
			orders, err := lockSharedOrders(txCtx, tx, organizationID, orderIDs)
			if err != nil {
				return err
			}
			linksByOrder, err := locateSharedLinks(txCtx, client, organizationID, orderIDs)
			if err != nil {
				return err
			}
			masterBillIDs := make([]uuid.UUID, 0, len(linksByOrder))
			linkIDs := make([]uuid.UUID, 0, len(linksByOrder))
			for _, link := range linksByOrder {
				masterBillIDs = append(masterBillIDs, link.MasterBillID)
				linkIDs = append(linkIDs, link.ID)
			}
			masterBillIDs = sortedUUIDs(masterBillIDs)
			linkIDs = sortedUUIDs(linkIDs)
			if len(masterBillIDs) > 0 {
				locked, err := client.SeaMasterBill.Query().Where(seamasterbillent.IDIn(masterBillIDs...), seamasterbillent.OrganizationIDEQ(organizationID)).Order(seamasterbillent.ByID()).ForUpdate().All(txCtx)
				if err != nil {
					return err
				}
				if len(locked) != len(masterBillIDs) {
					return biz.ErrSeaSharedContainerInvalidReference
				}
			}
			lockedLinks := make(map[uuid.UUID]*ent.SeaMasterBillOrderLink, len(linkIDs))
			if len(linkIDs) > 0 {
				rows, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.IDIn(linkIDs...), seamasterbillorderlinkent.OrganizationIDEQ(organizationID)).Order(seamasterbillorderlinkent.ByID()).ForUpdate().All(txCtx)
				if err != nil {
					return err
				}
				if len(rows) != len(linkIDs) {
					return biz.ErrSeaSharedContainerInvalidReference
				}
				for _, row := range rows {
					lockedLinks[row.OrderID] = row
				}
			}
			execution, err := client.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(located.TransportExecutionID), seatransportexecutionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaSharedContainerInvalidReference, nil)
			}
			_ = execution
			houseBills := make(map[uuid.UUID]*ent.SeaHouseBill, len(houseBillIDs))
			if len(houseBillIDs) > 0 {
				rows, err := client.SeaHouseBill.Query().Where(seahousebillent.IDIn(houseBillIDs...), seahousebillent.OrganizationIDEQ(organizationID)).Order(seahousebillent.ByID()).ForUpdate().All(txCtx)
				if err != nil {
					return err
				}
				if len(rows) != len(houseBillIDs) {
					return biz.ErrSeaSharedContainerInvalidReference
				}
				for _, row := range rows {
					houseBills[row.ID] = row
				}
			}
			cargoRows := make(map[uuid.UUID]*ent.OrderCargoItem, len(cargoIDs))
			if len(cargoIDs) > 0 {
				rows, err := client.OrderCargoItem.Query().Where(ordercargoitement.IDIn(cargoIDs...), ordercargoitement.OrganizationIDEQ(organizationID)).Order(ordercargoitement.ByID()).ForUpdate().All(txCtx)
				if err != nil {
					return err
				}
				if len(rows) != len(cargoIDs) {
					return biz.ErrSeaSharedContainerInvalidReference
				}
				for _, row := range rows {
					cargoRows[row.ID] = row
				}
			}
			containerIDs := make([]uuid.UUID, 0)
			rowsOnExecution, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.OrganizationIDEQ(organizationID), seasharedcontainerent.TransportExecutionIDEQ(located.TransportExecutionID)).All(txCtx)
			if err != nil {
				return err
			}
			for _, row := range rowsOnExecution {
				containerIDs = append(containerIDs, row.ID)
			}
			if len(cargoIDs) > 0 {
				otherAllocations, err := client.SeaSharedContainerAllocation.Query().Where(
					seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
					seasharedcontainerallocationent.CargoItemIDIn(cargoIDs...),
				).All(txCtx)
				if err != nil {
					return err
				}
				for _, allocation := range otherAllocations {
					containerIDs = append(containerIDs, allocation.SharedContainerID)
				}
			}
			containerIDs = sortedUUIDs(containerIDs)
			containers, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDIn(containerIDs...), seasharedcontainerent.OrganizationIDEQ(organizationID)).Order(seasharedcontainerent.ByID()).ForUpdate().All(txCtx)
			if err != nil {
				return err
			}
			var target *ent.SeaSharedContainer
			for _, container := range containers {
				if container.ID == id {
					target = container
				}
			}
			if target == nil {
				return biz.ErrSeaSharedContainerNotFound
			}
			if target.Version != expectedVersion {
				return biz.ErrSeaSharedContainerConflict
			}
			if target.Status != seasharedcontainerent.StatusDRAFT {
				return biz.ErrSeaSharedContainerStatusConflict
			}
			var allAllocations []*ent.SeaSharedContainerAllocation
			if len(containerIDs) > 0 {
				allAllocations, err = client.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.OrganizationIDEQ(organizationID), seasharedcontainerallocationent.SharedContainerIDIn(containerIDs...)).Order(seasharedcontainerallocationent.ByID()).ForUpdate().All(txCtx)
				if err != nil {
					return err
				}
			}
			cargoBaselines := make(map[uuid.UUID]biz.SeaSharedQuantity, len(cargoRows))
			cargoAllocated := make(map[uuid.UUID]biz.SeaSharedQuantity, len(cargoRows))
			for id, cargo := range cargoRows {
				cargoBaselines[id] = biz.SeaSharedQuantity{PackageCount: int32(cargo.PackageCount), GrossWeightKg: decimal.NewFromFloat(cargo.GrossWeightKg), VolumeCbm: decimal.NewFromFloat(cargo.VolumeCbm)}
			}
			for _, allocation := range allAllocations {
				if allocation.SharedContainerID == id {
					continue
				}
				if _, involved := cargoBaselines[allocation.CargoItemID]; !involved {
					continue
				}
				weight, err := decimal.NewFromString(allocation.GrossWeightKg)
				if err != nil {
					return err
				}
				volume, err := decimal.NewFromString(allocation.VolumeCbm)
				if err != nil {
					return err
				}
				cargoAllocated[allocation.CargoItemID] = cargoAllocated[allocation.CargoItemID].Add(biz.SeaSharedQuantity{PackageCount: int32(allocation.PackageCount), GrossWeightKg: weight, VolumeCbm: volume})
			}
			valueInputs := make([]biz.SeaSharedContainerAllocationInput, 0, len(inputs))
			for _, input := range inputs {
				order := orders[input.OrderID]
				link := lockedLinks[input.OrderID]
				houseBill := houseBills[input.HouseBillID]
				cargo := cargoRows[input.CargoItemID]
				if order == nil || link == nil || houseBill == nil || cargo == nil || order.BusinessType != orderent.BusinessTypeSE || link.Status != seamasterbillorderlinkent.StatusACTIVE || link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE || link.TransportExecutionID != target.TransportExecutionID || houseBill.OrderID != order.ID || houseBill.MasterBillID != link.MasterBillID || houseBill.Status == seahousebillent.StatusVOIDED || cargo.OrderID != order.ID {
					return biz.ErrSeaSharedContainerInvalidReference
				}
				if !confirm && (order.Version != input.ExpectedOrderVersion || link.Version != input.ExpectedLinkVersion || houseBill.Version != input.ExpectedHouseBillVersion || cargo.Version != input.ExpectedCargoItemVersion) {
					return biz.ErrSeaSharedContainerConflict
				}
				quantity := biz.SeaSharedQuantity{PackageCount: input.PackageCount, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}
				cargoAllocated[input.CargoItemID] = cargoAllocated[input.CargoItemID].Add(quantity)
				valueInputs = append(valueInputs, *input)
			}
			targetWeight, err := decimal.NewFromString(target.GrossWeightKg)
			if err != nil {
				return err
			}
			targetVolume, err := decimal.NewFromString(target.VolumeCbm)
			if err != nil {
				return err
			}
			containerQuantity := biz.SeaSharedQuantity{PackageCount: int32(target.PackageCount), GrossWeightKg: targetWeight, VolumeCbm: targetVolume}
			if err := biz.ValidateSeaSharedConservation(containerQuantity, valueInputs, cargoBaselines, cargoAllocated, confirm); err != nil {
				return err
			}
			for _, allocation := range allAllocations {
				if allocation.SharedContainerID == id {
					if err := client.SeaSharedContainerAllocation.DeleteOne(allocation).Exec(txCtx); err != nil {
						return err
					}
				}
			}
			for _, input := range inputs {
				_, err := client.SeaSharedContainerAllocation.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(organizationID).SetSharedContainerID(id).SetOrderID(input.OrderID).SetHouseBillID(input.HouseBillID).SetCargoItemID(input.CargoItemID).SetPackageCount(int(input.PackageCount)).SetGrossWeightKg(input.GrossWeightKg.StringFixed(3)).SetVolumeCbm(input.VolumeCbm.StringFixed(6)).SetVersion(1).Save(txCtx)
				if err != nil {
					return mapEntConstraint(err, "sea_shared_cntr_alloc_unique", biz.ErrSeaSharedContainerInvalidArgument)
				}
			}
			update := client.SeaSharedContainer.UpdateOne(target).SetVersion(target.Version + 1)
			if confirm {
				now := time.Now().UTC()
				update.SetStatus(seasharedcontainerent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actorID)
			} else {
				update.SetStatus(seasharedcontainerent.StatusDRAFT).ClearConfirmedAt().ClearConfirmedBy()
			}
			if _, err := update.Save(txCtx); err != nil {
				return err
			}
			addVersionAudit(audit, target.Version, target.Version+1)
			return writeAudit(txCtx, client.AuditLog, audit)
		})
	})
}

func (r *seaSharedContainerRepo) getByEntity(ctx context.Context, client *ent.Client, row *ent.SeaSharedContainer) (*biz.SeaSharedContainer, error) {
	container, err := client.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(row.ID), seasharedcontainerent.OrganizationIDEQ(row.OrganizationID)).WithConfirmedByUser().Only(ctx)
	if err != nil {
		return nil, err
	}
	spec, err := client.MasterDataItem.Query().Where(masterdataitement.IDEQ(container.ContainerSpecID), masterdataitement.OrganizationIDEQ(container.OrganizationID), masterdataitement.KindEQ(masterdataitement.KindContainerSpec)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaSharedContainerInvalidReference, nil)
	}
	allocRows, err := client.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.OrganizationIDEQ(container.OrganizationID), seasharedcontainerallocationent.SharedContainerIDEQ(container.ID)).WithOrder().WithHouseBill().WithCargoItem().Order(seasharedcontainerallocationent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	allocations := make([]*biz.SeaSharedContainerAllocation, 0, len(allocRows))
	cargoTotals := make(map[uuid.UUID]biz.SeaSharedQuantity)
	cargoIDs := make([]uuid.UUID, 0, len(allocRows))
	for _, allocation := range allocRows {
		weight, err := decimal.NewFromString(allocation.GrossWeightKg)
		if err != nil {
			return nil, err
		}
		volume, err := decimal.NewFromString(allocation.VolumeCbm)
		if err != nil {
			return nil, err
		}
		item := &biz.SeaSharedContainerAllocation{ID: allocation.ID, OrganizationID: allocation.OrganizationID, SharedContainerID: allocation.SharedContainerID, OrderID: allocation.OrderID, HouseBillID: allocation.HouseBillID, CargoItemID: allocation.CargoItemID, PackageCount: int32(allocation.PackageCount), GrossWeightKg: weight, VolumeCbm: volume, Version: allocation.Version, CreatedAt: allocation.CreatedAt, UpdatedAt: allocation.UpdatedAt}
		if allocation.Edges.Order != nil {
			item.OrderNo = allocation.Edges.Order.OrderNo
			item.OrderVersion = allocation.Edges.Order.Version
		}
		if allocation.Edges.HouseBill != nil {
			item.HouseNo = allocation.Edges.HouseBill.HouseNo
			item.HouseBillVersion = allocation.Edges.HouseBill.Version
		}
		if allocation.Edges.CargoItem != nil {
			item.CargoName = allocation.Edges.CargoItem.CargoName
			item.CargoItemVersion = allocation.Edges.CargoItem.Version
		}
		link, err := client.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(container.OrganizationID),
			seamasterbillorderlinkent.OrderIDEQ(allocation.OrderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaSharedContainerInvalidReference, nil)
		}
		item.LinkVersion = link.Version
		allocations = append(allocations, item)
		cargoIDs = append(cargoIDs, allocation.CargoItemID)
	}
	cargoBalanced := true
	for _, cargoID := range sortedUUIDs(cargoIDs) {
		rows, err := client.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.OrganizationIDEQ(container.OrganizationID), seasharedcontainerallocationent.CargoItemIDEQ(cargoID)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, allocation := range rows {
			weight, err := decimal.NewFromString(allocation.GrossWeightKg)
			if err != nil {
				return nil, err
			}
			volume, err := decimal.NewFromString(allocation.VolumeCbm)
			if err != nil {
				return nil, err
			}
			cargoTotals[cargoID] = cargoTotals[cargoID].Add(biz.SeaSharedQuantity{PackageCount: int32(allocation.PackageCount), GrossWeightKg: weight, VolumeCbm: volume})
		}
		cargo, err := client.OrderCargoItem.Query().Where(ordercargoitement.IDEQ(cargoID), ordercargoitement.OrganizationIDEQ(container.OrganizationID)).Only(ctx)
		if err != nil {
			return nil, err
		}
		baseline := biz.SeaSharedQuantity{PackageCount: int32(cargo.PackageCount), GrossWeightKg: decimal.NewFromFloat(cargo.GrossWeightKg), VolumeCbm: decimal.NewFromFloat(cargo.VolumeCbm)}
		if !cargoTotals[cargoID].Equal(baseline) {
			cargoBalanced = false
		}
	}
	containerWeight, err := decimal.NewFromString(container.GrossWeightKg)
	if err != nil {
		return nil, err
	}
	containerVolume, err := decimal.NewFromString(container.VolumeCbm)
	if err != nil {
		return nil, err
	}
	result := &biz.SeaSharedContainer{ID: container.ID, OrganizationID: container.OrganizationID, TransportExecutionID: container.TransportExecutionID, ContainerNo: container.ContainerNo, ContainerSpecID: container.ContainerSpecID, ContainerSpecName: spec.Name, PackageCount: int32(container.PackageCount), GrossWeightKg: containerWeight, VolumeCbm: containerVolume, Status: biz.SeaSharedContainerStatus(container.Status), Version: container.Version, Allocations: allocations, CreatedAt: container.CreatedAt, UpdatedAt: container.UpdatedAt}
	if container.SealNo != nil {
		value := *container.SealNo
		result.SealNo = &value
	}
	if container.Note != nil {
		value := *container.Note
		result.Note = &value
	}
	if container.ConfirmedAt != nil {
		value := *container.ConfirmedAt
		result.ConfirmedAt = &value
	}
	if container.ConfirmedBy != nil {
		value := *container.ConfirmedBy
		result.ConfirmedBy = &value
	}
	if container.Edges.ConfirmedByUser != nil {
		result.ConfirmedByName = container.Edges.ConfirmedByUser.DisplayName
	}
	result.Progress = biz.CalculateSeaSharedContainerProgress(biz.SeaSharedQuantity{PackageCount: result.PackageCount, GrossWeightKg: result.GrossWeightKg, VolumeCbm: result.VolumeCbm}, allocations, cargoBalanced)
	return result, nil
}

func validateSharedExecution(ctx context.Context, client *ent.Client, organizationID, executionID uuid.UUID, lock bool) error {
	query := client.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(executionID), seatransportexecutionent.OrganizationIDEQ(organizationID))
	if lock {
		query.ForUpdate()
	}
	if _, err := query.Only(ctx); err != nil {
		return mapEntError(err, biz.ErrSeaSharedContainerInvalidReference, nil)
	}
	return nil
}

func validateSharedContainerSpec(ctx context.Context, client *ent.Client, organizationID, specID uuid.UUID) error {
	_, err := client.MasterDataItem.Query().Where(masterdataitement.IDEQ(specID), masterdataitement.OrganizationIDEQ(organizationID), masterdataitement.KindEQ(masterdataitement.KindContainerSpec), masterdataitement.EnabledEQ(true)).ForShare().Only(ctx)
	if ent.IsNotFound(err) {
		return biz.ErrOrderContainerSpecInvalid
	}
	return err
}

func validateExecutionHasHouseOrders(ctx context.Context, client *ent.Client, organizationID, executionID uuid.UUID) error {
	count, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
		seamasterbillorderlinkent.TransportExecutionIDEQ(executionID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		seamasterbillorderlinkent.DocumentStructureEQ(seamasterbillorderlinkent.DocumentStructureHOUSE),
	).Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return biz.ErrSeaSharedContainerInvalidReference
	}
	return nil
}

func lockSharedOrders(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*ent.Order, error) {
	result := make(map[uuid.UUID]*ent.Order, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := tx.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDEQ(organizationID)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(ids) {
		return nil, biz.ErrSeaSharedContainerInvalidReference
	}
	for _, row := range rows {
		if err := ensureOrderBusinessEditable(ctx, tx, row); err != nil {
			return nil, err
		}
		result[row.ID] = row
	}
	return result, nil
}

func locateSharedLinks(ctx context.Context, client *ent.Client, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*ent.SeaMasterBillOrderLink, error) {
	result := make(map[uuid.UUID]*ent.SeaMasterBillOrderLink, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}
	rows, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(organizationID), seamasterbillorderlinkent.OrderIDIn(orderIDs...), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(orderIDs) {
		return nil, biz.ErrSeaSharedContainerInvalidReference
	}
	for _, row := range rows {
		result[row.OrderID] = row
	}
	return result, nil
}

func allocationReferenceIDs(inputs []*biz.SeaSharedContainerAllocationInput) ([]uuid.UUID, []uuid.UUID, []uuid.UUID) {
	orders := make([]uuid.UUID, 0, len(inputs))
	cargo := make([]uuid.UUID, 0, len(inputs))
	houseBills := make([]uuid.UUID, 0, len(inputs))
	for _, input := range inputs {
		orders = append(orders, input.OrderID)
		cargo = append(cargo, input.CargoItemID)
		houseBills = append(houseBills, input.HouseBillID)
	}
	return sortedUUIDs(orders), sortedUUIDs(cargo), sortedUUIDs(houseBills)
}

func sortedUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return strings.Compare(result[i].String(), result[j].String()) < 0 })
	return result
}

func addVersionAudit(audit *biz.AuditEvent, oldVersion, newVersion uint64) {
	if audit.Details == nil {
		audit.Details = make(map[string]string)
	}
	audit.Details["shared_container.version.old"] = fmt.Sprintf("%d", oldVersion)
	if newVersion == 0 {
		audit.Details["shared_container.version.new"] = "deleted"
	} else {
		audit.Details["shared_container.version.new"] = fmt.Sprintf("%d", newVersion)
	}
}

var _ biz.SeaSharedContainerRepo = (*seaSharedContainerRepo)(nil)
