package data

import (
	"context"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargocategory"
	orderserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	statustemplateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/statustemplate"
	statustemplateitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/statustemplateitem"
)

type orderRepo struct{ data *Data }

func NewOrderRepo(data *Data) biz.OrderRepo { return &orderRepo{data: data} }

func (r *orderRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.Order, error) {
	item, err := withOrderEdges(r.data.db.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	return orderToBiz(item), nil
}

func (r *orderRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.OrderListOptions) (*biz.OrderList, error) {
	query := r.data.db.Order.Query().Where(orderent.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(orderent.Or(orderent.OrderNoContainsFold(options.Keyword), orderent.VesselVoyageContainsFold(options.Keyword), orderent.GoodsDescriptionContainsFold(options.Keyword)))
	}
	if options.Status != "" {
		query.Where(orderent.StatusEQ(options.Status))
	}
	if options.BusinessType != "" {
		query.Where(orderent.BusinessTypeEQ(orderent.BusinessType(options.BusinessType)))
	}
	if options.CustomerID != nil {
		query.Where(orderent.CustomerIDEQ(*options.CustomerID))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := withOrderEdges(query).
		Order(orderent.ByCreatedAt(entsql.OrderDesc())).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Order, 0, len(items))
	for _, item := range items {
		result = append(result, orderToBiz(item))
	}
	return &biz.OrderList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *orderRepo) Create(ctx context.Context, organizationID, actorID uuid.UUID, number string, input *biz.Order) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateOrderReferences(ctx, tx, organizationID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	create := tx.Order.Create().
		SetOrganizationID(organizationID).
		SetOrderNo(number).
		SetCustomerID(input.CustomerID).
		SetNillableCarrierID(input.CarrierID).
		SetNillableBookingAgentID(input.BookingAgentID).
		SetBusinessType(orderent.BusinessType(input.BusinessType)).
		SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
		SetTradeTerm(orderent.TradeTerm(input.TradeTerm)).
		SetPaymentTerm(orderent.PaymentTerm(input.PaymentTerm)).
		SetNillableShipmentType(orderShipmentTypeToEnt(input.ShipmentType)).
		SetNillableContainerOwnership(orderContainerOwnershipToEnt(input.ContainerOwnership)).
		SetNillableShipmentMode(orderShipmentModeToEnt(input.ShipmentMode)).
		SetStatus("DRAFT").
		SetStatusTemplateID(input.StatusTemplateID).
		SetNillableOriginLocationID(input.OriginLocationID).
		SetNillableDestinationLocationID(input.DestinationLocationID).
		SetNillableDischargeLocationID(input.DischargeLocationID).
		SetNillableTransitLocationID(input.TransitLocationID).
		SetVesselVoyage(input.VesselVoyage).
		SetEtd(input.ETD).
		SetEta(input.ETA).
		SetSiCutoff(input.SICutoff).
		SetDocCutoff(input.DocCutoff).
		SetCustomsCutoff(input.CustomsCutoff).
		SetVgmCutoff(input.VGMCutoff).
		SetGoodsDescription(input.GoodsDescription).
		SetNillableTotalPackages(input.TotalPackages).
		SetTotalPackageUnit(input.TotalPackageUnit).
		SetSpecialRequirements(input.SpecialRequirements).
		SetOrderDate(input.OrderDate).
		SetNotes(input.Notes)
	created, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapOrderConstraint(err)
	}
	if err := replaceOrderSelections(ctx, tx, created.ID, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderStatusLog.Create().SetOrderID(created.ID).SetToStatus("DRAFT").SetAction("create").SetOperatorID(actorID).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, created.ID)
}

func (r *orderRepo) UpdateDraft(ctx context.Context, organizationID, id uuid.UUID, expectedStatus string, input *biz.Order) (*biz.Order, error) {
	if expectedStatus != "DRAFT" {
		return nil, biz.ErrOrderStatusConflict
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Status != expectedStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if existing.BusinessType != orderent.BusinessType(input.BusinessType) || existing.StatusTemplateID != input.StatusTemplateID {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusTemplate
	}
	if err := validateOrderReferences(ctx, tx, organizationID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := existing.Update().
		SetCustomerID(input.CustomerID).
		SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
		SetTradeTerm(orderent.TradeTerm(input.TradeTerm)).
		SetPaymentTerm(orderent.PaymentTerm(input.PaymentTerm)).
		SetVesselVoyage(input.VesselVoyage).
		SetEtd(input.ETD).
		SetEta(input.ETA).
		SetSiCutoff(input.SICutoff).
		SetDocCutoff(input.DocCutoff).
		SetCustomsCutoff(input.CustomsCutoff).
		SetVgmCutoff(input.VGMCutoff).
		SetGoodsDescription(input.GoodsDescription).
		SetTotalPackageUnit(input.TotalPackageUnit).
		SetSpecialRequirements(input.SpecialRequirements).
		SetOrderDate(input.OrderDate).
		SetNotes(input.Notes)
	setOrderOptionalReferences(update, input)
	if input.TotalPackages == nil {
		update.ClearTotalPackages()
	} else {
		update.SetTotalPackages(*input.TotalPackages)
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, mapOrderConstraint(err)
	}
	if err := replaceOrderSelections(ctx, tx, id, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionStatus(ctx context.Context, organizationID, id uuid.UUID, expectedStatus, targetStatus, reason string, actorID uuid.UUID, event *biz.OrderStatusChangedEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Status != expectedStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	valid, err := tx.StatusTemplateItem.Query().Where(statustemplateitement.TemplateIDEQ(existing.StatusTemplateID), statustemplateitement.CodeEQ(targetStatus), statustemplateitement.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if !valid {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusInvalid
	}
	if _, err := existing.Update().SetStatus(targetStatus).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderStatusLog.Create().SetOrderID(id).SetFromStatus(expectedStatus).SetToStatus(targetStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, event.AuditEvent()); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func validateOrderReferences(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, input *biz.Order) error {
	if err := validatePartnerRole(ctx, tx, organizationID, input.CustomerID, partnerroleent.RoleTypeCustomer); err != nil {
		return biz.ErrOrderCustomerInvalid
	}
	if input.CarrierID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.CarrierID, partnerroleent.RoleTypeCarrier); err != nil {
			return biz.ErrOrderInvalidArgument
		}
	}
	if input.BookingAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.BookingAgentID, partnerroleent.RoleTypeAgent); err != nil {
			return biz.ErrOrderInvalidArgument
		}
	}
	validTemplate, err := tx.StatusTemplate.Query().Where(
		statustemplateent.IDEQ(input.StatusTemplateID),
		statustemplateent.OrganizationIDEQ(organizationID),
		statustemplateent.BusinessTypeEQ(statustemplateent.BusinessType(input.BusinessType)),
		statustemplateent.PublishedAtNotNil(),
		statustemplateent.EnabledEQ(true),
		statustemplateent.HasItemsWith(statustemplateitement.CodeEQ("DRAFT"), statustemplateitement.EnabledEQ(true)),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if !validTemplate {
		return biz.ErrOrderStatusTemplate
	}
	if err := validateMasterDataIDs(ctx, tx, organizationID, input.ServiceTypeIDs, masterdataent.KindServiceType); err != nil {
		return err
	}
	if err := validateMasterDataIDs(ctx, tx, organizationID, input.CargoCategoryIDs, masterdataent.KindCargoCategory); err != nil {
		return err
	}
	locationIDs := nonNilUUIDs(input.OriginLocationID, input.DestinationLocationID, input.DischargeLocationID, input.TransitLocationID)
	locationKind := masterdataent.KindRegion
	if input.BusinessType == biz.OrderBusinessSE || input.BusinessType == biz.OrderBusinessSI {
		locationKind = masterdataent.KindPort
	} else if input.BusinessType == biz.OrderBusinessAE || input.BusinessType == biz.OrderBusinessAI {
		locationKind = masterdataent.KindAirport
	}
	return validateMasterDataIDs(ctx, tx, organizationID, locationIDs, locationKind)
}

func validatePartnerRole(ctx context.Context, tx *ent.Tx, organizationID, partnerID uuid.UUID, roleType partnerroleent.RoleType) error {
	exists, err := tx.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(partnerID),
		partnerroleent.RoleTypeEQ(roleType),
		partnerroleent.EnabledEQ(true),
		partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderInvalidArgument
	}
	return nil
}

func validateMasterDataIDs(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, ids []uuid.UUID, kind masterdataent.Kind) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := tx.MasterDataItem.Query().Where(masterdataent.IDIn(ids...), masterdataent.OrganizationIDEQ(organizationID), masterdataent.KindEQ(kind), masterdataent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return biz.ErrOrderInvalidArgument
	}
	return nil
}

func replaceOrderSelections(ctx context.Context, tx *ent.Tx, orderID uuid.UUID, serviceTypeIDs, cargoCategoryIDs []uuid.UUID) error {
	if _, err := tx.OrderServiceType.Delete().Where(orderserviceent.OrderIDEQ(orderID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.OrderCargoCategory.Delete().Where(ordercargoent.OrderIDEQ(orderID)).Exec(ctx); err != nil {
		return err
	}
	for _, id := range serviceTypeIDs {
		if _, err := tx.OrderServiceType.Create().SetOrderID(orderID).SetMasterDataItemID(id).Save(ctx); err != nil {
			return err
		}
	}
	for _, id := range cargoCategoryIDs {
		if _, err := tx.OrderCargoCategory.Create().SetOrderID(orderID).SetMasterDataItemID(id).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func withOrderEdges(query *ent.OrderQuery) *ent.OrderQuery {
	return query.
		WithServiceTypes(func(q *ent.OrderServiceTypeQuery) { q.Order(orderserviceent.ByCreatedAt()) }).
		WithCargoCategories(func(q *ent.OrderCargoCategoryQuery) { q.Order(ordercargoent.ByCreatedAt()) })
}

func orderToBiz(item *ent.Order) *biz.Order {
	result := &biz.Order{
		ID: item.ID, OrganizationID: item.OrganizationID, OrderNo: item.OrderNo, CustomerID: item.CustomerID,
		CarrierID: item.CarrierID, BookingAgentID: item.BookingAgentID, BusinessType: biz.OrderBusinessType(item.BusinessType),
		TradeDirection: biz.OrderTradeDirection(item.TradeDirection), TradeTerm: biz.OrderTradeTerm(item.TradeTerm), PaymentTerm: biz.OrderPaymentTerm(item.PaymentTerm),
		Status: item.Status, StatusTemplateID: item.StatusTemplateID, OriginLocationID: item.OriginLocationID, DestinationLocationID: item.DestinationLocationID,
		DischargeLocationID: item.DischargeLocationID, TransitLocationID: item.TransitLocationID, VesselVoyage: item.VesselVoyage, ETD: item.Etd, ETA: item.Eta,
		SICutoff: item.SiCutoff, DocCutoff: item.DocCutoff, CustomsCutoff: item.CustomsCutoff, VGMCutoff: item.VgmCutoff,
		GoodsDescription: item.GoodsDescription, TotalPackages: item.TotalPackages, TotalPackageUnit: item.TotalPackageUnit,
		SpecialRequirements: item.SpecialRequirements, OrderDate: item.OrderDate, Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.ShipmentType != nil {
		value := biz.OrderShipmentType(*item.ShipmentType)
		result.ShipmentType = &value
	}
	if item.ContainerOwnership != nil {
		value := biz.OrderContainerOwnership(*item.ContainerOwnership)
		result.ContainerOwnership = &value
	}
	if item.ShipmentMode != nil {
		value := biz.OrderShipmentMode(*item.ShipmentMode)
		result.ShipmentMode = &value
	}
	result.ServiceTypeIDs = make([]uuid.UUID, 0, len(item.Edges.ServiceTypes))
	for _, link := range item.Edges.ServiceTypes {
		result.ServiceTypeIDs = append(result.ServiceTypeIDs, link.MasterDataItemID)
	}
	result.CargoCategoryIDs = make([]uuid.UUID, 0, len(item.Edges.CargoCategories))
	for _, link := range item.Edges.CargoCategories {
		result.CargoCategoryIDs = append(result.CargoCategoryIDs, link.MasterDataItemID)
	}
	return result
}

func setOrderOptionalReferences(update *ent.OrderUpdateOne, input *biz.Order) {
	if input.CarrierID == nil {
		update.ClearCarrierID()
	} else {
		update.SetCarrierID(*input.CarrierID)
	}
	if input.BookingAgentID == nil {
		update.ClearBookingAgentID()
	} else {
		update.SetBookingAgentID(*input.BookingAgentID)
	}
	if input.ShipmentType == nil {
		update.ClearShipmentType()
	} else {
		update.SetShipmentType(orderent.ShipmentType(*input.ShipmentType))
	}
	if input.ContainerOwnership == nil {
		update.ClearContainerOwnership()
	} else {
		update.SetContainerOwnership(orderent.ContainerOwnership(*input.ContainerOwnership))
	}
	if input.ShipmentMode == nil {
		update.ClearShipmentMode()
	} else {
		update.SetShipmentMode(orderent.ShipmentMode(*input.ShipmentMode))
	}
	if input.OriginLocationID == nil {
		update.ClearOriginLocationID()
	} else {
		update.SetOriginLocationID(*input.OriginLocationID)
	}
	if input.DestinationLocationID == nil {
		update.ClearDestinationLocationID()
	} else {
		update.SetDestinationLocationID(*input.DestinationLocationID)
	}
	if input.DischargeLocationID == nil {
		update.ClearDischargeLocationID()
	} else {
		update.SetDischargeLocationID(*input.DischargeLocationID)
	}
	if input.TransitLocationID == nil {
		update.ClearTransitLocationID()
	} else {
		update.SetTransitLocationID(*input.TransitLocationID)
	}
}

func orderShipmentTypeToEnt(value *biz.OrderShipmentType) *orderent.ShipmentType {
	if value == nil {
		return nil
	}
	result := orderent.ShipmentType(*value)
	return &result
}

func orderContainerOwnershipToEnt(value *biz.OrderContainerOwnership) *orderent.ContainerOwnership {
	if value == nil {
		return nil
	}
	result := orderent.ContainerOwnership(*value)
	return &result
}

func orderShipmentModeToEnt(value *biz.OrderShipmentMode) *orderent.ShipmentMode {
	if value == nil {
		return nil
	}
	result := orderent.ShipmentMode(*value)
	return &result
}

func nonNilUUIDs(values ...*uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func mapOrderConstraint(err error) error {
	if ent.IsConstraintError(err) && strings.Contains(err.Error(), "order_organization_id_order_no") {
		return biz.ErrOrderNumberExists
	}
	return err
}

var _ biz.OrderRepo = (*orderRepo)(nil)
