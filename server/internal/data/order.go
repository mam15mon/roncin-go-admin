package data

import (
	"context"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airportent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordercargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargocategory"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	orderconsolidationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderconsolidation"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	orderserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	entpredicate "github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
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

func (r *orderRepo) Find(ctx context.Context, id uuid.UUID) (*biz.Order, error) {
	item, err := withOrderEdges(r.data.db.Order.Query().Where(orderent.IDEQ(id))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	return orderToBiz(item), nil
}

func (r *orderRepo) List(ctx context.Context, organizationIDs []uuid.UUID, options biz.OrderListOptions) (*biz.OrderList, error) {
	query := r.data.db.Order.Query().Where(orderent.OrganizationIDIn(organizationIDs...))
	if options.Keyword != "" {
		query.Where(orderent.Or(orderent.OrderNoContainsFold(options.Keyword), orderent.VesselVoyageContainsFold(options.Keyword), orderent.GoodsDescriptionContainsFold(options.Keyword)))
	}
	if options.NumberKeyword != "" {
		switch options.NumberType {
		case biz.OrderNumberFilterOrder:
			query.Where(orderent.OrderNoContainsFold(options.NumberKeyword))
		case biz.OrderNumberFilterMaster:
			query.Where(orderent.HasShippingDocumentsWith(
				ordershippingdocumentent.HasConsolidationWith(orderconsolidationent.MasterNoContainsFold(options.NumberKeyword)),
			))
		case biz.OrderNumberFilterConsolidatedMaster:
			query.Where(orderConsolidatedMasterContainsFold(options.NumberKeyword))
		}
	}
	if options.CreatedAtRange.From != nil {
		query.Where(orderent.CreatedAtGTE(*options.CreatedAtRange.From))
	}
	if options.CreatedAtRange.ToExclusive != nil {
		query.Where(orderent.CreatedAtLT(*options.CreatedAtRange.ToExclusive))
	}
	applyOrderStringDateRange(query, orderent.FieldEtd, options.ETDRange)
	applyOrderStringDateRange(query, orderent.FieldEta, options.ETARange)
	if options.StatusTimeRange.From != nil || options.StatusTimeRange.ToExclusive != nil {
		predicates := make([]entpredicate.OrderLifecycleEvent, 0, 2)
		if options.StatusTimeRange.From != nil {
			predicates = append(predicates, orderlifecycleeventent.ChangedAtGTE(*options.StatusTimeRange.From))
		}
		if options.StatusTimeRange.ToExclusive != nil {
			predicates = append(predicates, orderlifecycleeventent.ChangedAtLT(*options.StatusTimeRange.ToExclusive))
		}
		query.Where(orderent.HasLifecycleEventsWith(predicates...))
	}
	if options.LockedAtRange.From != nil {
		query.Where(orderent.LockedAtGTE(*options.LockedAtRange.From))
	}
	if options.LockedAtRange.ToExclusive != nil {
		query.Where(orderent.LockedAtLT(*options.LockedAtRange.ToExclusive))
	}
	if options.FlowStatus != "" {
		query.Where(orderent.FlowStatusEQ(orderent.FlowStatus(options.FlowStatus)))
	}
	if options.TerminationStatus != "" {
		query.Where(orderent.TerminationStatusEQ(orderent.TerminationStatus(options.TerminationStatus)))
	}
	if options.ClosureStatus != "" {
		query.Where(orderent.ClosureStatusEQ(orderent.ClosureStatus(options.ClosureStatus)))
	}
	if options.HasActiveException != nil {
		activeException := orderent.HasAbnormalCasesWith(orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE))
		if *options.HasActiveException {
			query.Where(activeException)
		} else {
			query.Where(orderent.Not(activeException))
		}
	}
	if options.BusinessType != "" {
		query.Where(orderent.BusinessTypeEQ(orderent.BusinessType(options.BusinessType)))
	} else {
		businessTypes := make([]orderent.BusinessType, 0, len(options.BusinessTypes))
		for _, businessType := range options.BusinessTypes {
			businessTypes = append(businessTypes, orderent.BusinessType(businessType))
		}
		query.Where(orderent.BusinessTypeIn(businessTypes...))
	}
	if options.CustomerID != nil {
		query.Where(orderent.CustomerIDEQ(*options.CustomerID))
	}
	if options.OriginLocationID != nil {
		query.Where(orderent.OriginLocationIDEQ(*options.OriginLocationID))
	}
	if options.DestinationLocationID != nil {
		query.Where(orderent.DestinationLocationIDEQ(*options.DestinationLocationID))
	}
	if options.CarrierID != nil {
		query.Where(orderent.CarrierIDEQ(*options.CarrierID))
	}
	if options.ConsigneeShortName != "" {
		query.Where(orderent.ConsigneeShortNameContainsFold(options.ConsigneeShortName))
	}
	if options.ShipperShortName != "" {
		query.Where(orderent.ShipperShortNameContainsFold(options.ShipperShortName))
	}
	applyOrderPersonnelFilter(query, orderpersonnelent.RoleOPERATOR, options.Operator)
	applyOrderPersonnelFilter(query, orderpersonnelent.RoleSALES, options.Sales)
	applyOrderPersonnelFilter(query, orderpersonnelent.RoleCUSTOMER_SERVICE, options.CustomerService)
	applyOrderPersonnelFilter(query, orderpersonnelent.RoleCREATOR, options.Creator)
	if options.IsLocked != nil {
		if *options.IsLocked {
			query.Where(orderent.LockedAtNotNil())
		} else {
			query.Where(orderent.LockedAtIsNil())
		}
	}
	if options.IsShared != nil {
		query.Where(orderent.IsSharedEQ(*options.IsShared))
	}
	if len(options.Tags) > 0 {
		if options.TagMatchMode == biz.OrderTagMatchExactAnd {
			for _, tag := range options.Tags {
				tag := tag
				query.Where(entpredicate.Order(func(selector *entsql.Selector) {
					selector.Where(sqljson.ValueContains(selector.C(orderent.FieldTags), tag))
				}))
			}
		} else {
			tags := append([]string(nil), options.Tags...)
			query.Where(entpredicate.Order(func(selector *entsql.Selector) {
				predicates := make([]*entsql.Predicate, 0, len(tags))
				for _, tag := range tags {
					predicates = append(predicates, sqljson.StringContains(selector.C(orderent.FieldTags), tag))
				}
				selector.Where(entsql.Or(predicates...))
			}))
		}
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

func orderConsolidatedMasterContainsFold(keyword string) entpredicate.Order {
	return entpredicate.Order(func(selector *entsql.Selector) {
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			builder.WriteString(`EXISTS (SELECT 1 FROM "order_shipping_documents" AS "filter_document" JOIN "order_consolidations" AS "filter_consolidation" ON "filter_consolidation"."id" = "filter_document"."consolidation_id" WHERE "filter_document"."order_id" = `).
				Ident(selector.C(orderent.FieldID)).
				WriteString(` AND LOWER("filter_consolidation"."master_no") LIKE `).
				Arg("%" + strings.ToLower(keyword) + "%").
				WriteString(` GROUP BY "filter_consolidation"."id" HAVING COUNT(DISTINCT "filter_document"."order_id") > 1)`)
		}))
	})
}

func applyOrderStringDateRange(query *ent.OrderQuery, fieldName string, dateRange biz.OrderDateRange) {
	if dateRange.From == nil && dateRange.ToExclusive == nil {
		return
	}
	query.Where(entpredicate.Order(func(selector *entsql.Selector) {
		if dateRange.From != nil {
			selector.Where(entsql.GTE(selector.C(fieldName), dateRange.From.Format("2006-01-02")))
		}
		if dateRange.ToExclusive != nil {
			selector.Where(entsql.LT(selector.C(fieldName), dateRange.ToExclusive.Format("2006-01-02")))
		}
	}))
}

func applyOrderPersonnelFilter(query *ent.OrderQuery, role orderpersonnelent.Role, filter biz.OrderPersonnelFilter) {
	if filter.UserID == nil {
		return
	}
	predicates := []entpredicate.OrderPersonnel{
		orderpersonnelent.RoleEQ(role),
		orderpersonnelent.UserIDEQ(*filter.UserID),
	}
	if filter.OrganizationID != nil {
		predicates = append(predicates, orderpersonnelent.OrganizationIDEQ(*filter.OrganizationID))
	}
	query.Where(orderent.HasPersonnelWith(predicates...))
}

func (r *orderRepo) FindReferenceDuplicate(ctx context.Context, organizationID uuid.UUID, check biz.OrderReferenceCheck) (*biz.OrderReferenceMatch, error) {
	query := r.data.db.Order.Query().Where(orderent.OrganizationIDEQ(organizationID))
	if check.ReferenceType == biz.OrderReferenceCustomer {
		query.Where(
			orderent.CustomerIDEQ(*check.CustomerID),
			orderent.CustomerReferenceNoEqualFold(check.ReferenceNo),
		)
	} else {
		query.Where(orderent.InternalReferenceNoEqualFold(check.ReferenceNo))
	}
	if check.ExcludeOrderID != nil {
		query.Where(orderent.IDNEQ(*check.ExcludeOrderID))
	}
	item, err := query.Order(orderent.ByCreatedAt(entsql.OrderDesc())).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &biz.OrderReferenceMatch{OrderID: item.ID, OrderNo: item.OrderNo}, nil
}

func (r *orderRepo) HasContainers(ctx context.Context, organizationID, orderID uuid.UUID) (bool, error) {
	return r.data.db.Order.Query().Where(
		orderent.IDEQ(orderID),
		orderent.OrganizationIDEQ(organizationID),
		orderent.HasContainers(),
	).Exist(ctx)
}

func (r *orderRepo) ListConsolidationSummaries(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderConsolidationSummary, error) {
	current, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if current.ShipmentType == nil || *current.ShipmentType != orderent.ShipmentTypeLCL {
		return nil, biz.ErrOrderConsolidationShipmentType
	}
	documents, err := r.data.db.OrderShippingDocument.Query().Where(ordershippingdocumentent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, err
	}
	consolidationIDs := make([]uuid.UUID, 0, len(documents))
	seen := make(map[uuid.UUID]struct{}, len(documents))
	for _, document := range documents {
		if _, exists := seen[document.ConsolidationID]; !exists {
			seen[document.ConsolidationID] = struct{}{}
			consolidationIDs = append(consolidationIDs, document.ConsolidationID)
		}
	}
	if len(consolidationIDs) == 0 {
		return []*biz.OrderConsolidationSummary{}, nil
	}
	items, err := r.data.db.OrderConsolidation.Query().Where(
		orderconsolidationent.IDIn(consolidationIDs...),
		orderconsolidationent.OrganizationIDEQ(organizationID),
	).WithShippingDocuments(func(query *ent.OrderShippingDocumentQuery) {
		query.Where(ordershippingdocumentent.HasOrderWith(orderent.ShipmentTypeEQ(orderent.ShipmentTypeLCL))).
			WithOrder(func(orderQuery *ent.OrderQuery) { orderQuery.WithCargoItems() }).
			Order(ordershippingdocumentent.ByCreatedAt())
	}).Order(orderconsolidationent.ByMasterNo()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderConsolidationSummary, 0, len(items))
	for _, item := range items {
		summary := &biz.OrderConsolidationSummary{ConsolidationID: item.ID, MasterNo: item.MasterNo}
		members := make(map[uuid.UUID]*biz.OrderConsolidationMember)
		for _, document := range item.Edges.ShippingDocuments {
			orderItem := document.Edges.Order
			member := members[orderItem.ID]
			if member == nil {
				member = &biz.OrderConsolidationMember{OrderID: orderItem.ID, OrderNo: orderItem.OrderNo, CustomerReferenceNo: orderItem.CustomerReferenceNo}
				if orderItem.TotalPackages != nil {
					member.Entrusted.Packages = *orderItem.TotalPackages
				}
				if orderItem.TotalGrossWeightKg != nil {
					member.Entrusted.GrossWeightKg = *orderItem.TotalGrossWeightKg
				}
				if orderItem.TotalVolumeCbm != nil {
					member.Entrusted.VolumeCbm = *orderItem.TotalVolumeCbm
				}
				for _, cargo := range orderItem.Edges.CargoItems {
					member.Actual.Packages += cargo.PackageCount
					member.Actual.GrossWeightKg += cargo.GrossWeightKg
					member.Actual.VolumeCbm += cargo.VolumeCbm
				}
				members[orderItem.ID] = member
				summary.Members = append(summary.Members, member)
			}
			member.HouseNos = append(member.HouseNos, document.HouseNo)
		}
		for _, member := range summary.Members {
			summary.Entrusted.Packages += member.Entrusted.Packages
			summary.Entrusted.GrossWeightKg += member.Entrusted.GrossWeightKg
			summary.Entrusted.VolumeCbm += member.Entrusted.VolumeCbm
			summary.Actual.Packages += member.Actual.Packages
			summary.Actual.GrossWeightKg += member.Actual.GrossWeightKg
			summary.Actual.VolumeCbm += member.Actual.VolumeCbm
		}
		result = append(result, summary)
	}
	return result, nil
}

func (r *orderRepo) ListPersonnelOptions(ctx context.Context, organizationID uuid.UUID, options biz.SelectorListOptions) (*biz.PagedList[*biz.OrderPersonnelOption], error) {
	organizations, err := r.data.db.Organization.Query().
		Select(organizationent.FieldID, organizationent.FieldParentID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	parentByID := make(map[uuid.UUID]*uuid.UUID, len(organizations))
	organizationIDs := make([]uuid.UUID, 0, len(organizations))
	for _, organization := range organizations {
		parentByID[organization.ID] = organization.ParentID
	}
	for _, organization := range organizations {
		if organizationWithinRoot(parentByID, organizationID, organization.ID) {
			organizationIDs = append(organizationIDs, organization.ID)
		}
	}
	query := r.data.db.Membership.Query().Where(
		membershipent.OrganizationIDIn(organizationIDs...),
		membershipent.EnabledEQ(true),
		membershipent.HasUserWith(userent.EnabledEQ(true)),
		membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)),
	)
	if options.Keyword != "" {
		query.Where(membershipent.Or(
			membershipent.HasUserWith(userent.Or(userent.UsernameContainsFold(options.Keyword), userent.DisplayNameContainsFold(options.Keyword), userent.SearchKeywordsContainsFold(options.Keyword))),
			membershipent.HasOrganizationWith(organizationent.Or(organizationent.CodeContainsFold(options.Keyword), organizationent.NameContainsFold(options.Keyword), organizationent.SearchKeywordsContainsFold(options.Keyword))),
		))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	memberships, err := query.WithUser().WithOrganization().
		Order(membershipent.ByUserField(userent.FieldDisplayName), membershipent.ByOrganizationField(organizationent.FieldName)).
		Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderPersonnelOption, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, &biz.OrderPersonnelOption{
			UserID: membership.UserID, DisplayName: membership.Edges.User.DisplayName,
			OrganizationID: membership.OrganizationID, OrganizationName: membership.Edges.Organization.Name,
		})
	}
	return &biz.PagedList[*biz.OrderPersonnelOption]{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
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
		SetCustomerReferenceNo(input.CustomerReferenceNo).
		SetInternalReferenceNo(input.InternalReferenceNo).
		SetShipperShortName(input.ShipperShortName).
		SetConsigneeShortName(input.ConsigneeShortName).
		SetTags(input.Tags).
		SetNillableCarrierID(input.CarrierID).
		SetNillableBookingAgentID(input.BookingAgentID).
		SetNillableForeignAgentID(input.ForeignAgentID).
		SetNillableShippingAgentID(input.ShippingAgentID).
		SetContractNo(input.ContractNo).
		SetCargoValue(input.CargoValue).
		SetInsurancePremium(input.InsurancePremium).
		SetUnNumber(input.UNNumber).
		SetHazardClass(input.HazardClass).
		SetFactoryName(input.FactoryName).
		SetCargoReadyAt(input.CargoReadyAt).
		SetLoadingTerms(input.LoadingTerms).
		SetDeclarationCutoffAt(input.DeclarationCutoffAt).
		SetReceivedAt(input.ReceivedAt).
		SetBusinessType(orderent.BusinessType(input.BusinessType)).
		SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
		SetTradeTerm(orderent.TradeTerm(input.TradeTerm)).
		SetPaymentTerm(orderent.PaymentTerm(input.PaymentTerm)).
		SetNillableShipmentType(orderShipmentTypeToEnt(input.ShipmentType)).
		SetNillableContainerOwnership(orderContainerOwnershipToEnt(input.ContainerOwnership)).
		SetNillableShipmentMode(orderShipmentModeToEnt(input.ShipmentMode)).
		SetFlowStatus(orderent.FlowStatusDRAFT).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
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
		SetNillableTotalGrossWeightKg(input.TotalGrossWeightKg).
		SetNillableTotalVolumeCbm(input.TotalVolumeCbm).
		SetTotalPackageUnit(input.TotalPackageUnit).
		SetSpecialRequirements(input.SpecialRequirements).
		SetOrderDate(input.OrderDate).
		SetNotes(input.Notes).
		SetBookingNotes(input.BookingNotes).
		SetAllocationNotes(input.AllocationNotes).
		SetOperationNotes(input.OperationNotes)
	create.SetNillableCargoCurrency(nonEmptyStringPointer(input.CargoCurrency))
	create.SetNillableInsuranceCurrency(nonEmptyStringPointer(input.InsuranceCurrency))
	created, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapOrderConstraint(err)
	}
	if err := replaceOrderSelections(ctx, tx, created.ID, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderShippingDocuments(ctx, tx, organizationID, input.BusinessType, created.ID, input.ShippingDocuments); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderContainerRequests(ctx, tx, organizationID, created.ID, input.ContainerRequests); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(created.ID).SetDimension(orderlifecycleeventent.DimensionFLOW).SetToStatus("DRAFT").SetAction("create").SetOperatorID(actorID).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	personnel := make([]*biz.OrderPersonnel, 0, len(input.PersonnelAssignments)+1)
	personnel = append(personnel, &biz.OrderPersonnel{UserID: actorID, OrganizationID: organizationID, Role: biz.OrderPersonnelRoleCreator})
	personnel = append(personnel, input.PersonnelAssignments...)
	if err := createOrderPersonnel(ctx, tx, organizationID, created.ID, created.OrderNo, personnel); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := snapshotOrderCommissionAttributions(ctx, tx, organizationID, created.ID, input.CustomerID, created.CreatedAt); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, created.ID)
}

func snapshotOrderCommissionAttributions(ctx context.Context, tx *ent.Tx, organizationID, orderID, customerID uuid.UUID, attributedAt time.Time) error {
	assignments, err := tx.PartnerAssignment.Query().Where(
		partnerassignmentent.OrganizationIDEQ(organizationID),
		partnerassignmentent.PartnerIDEQ(customerID),
		partnerassignmentent.RoleIn(partnerassignmentent.RoleSALES, partnerassignmentent.RoleOPERATOR, partnerassignmentent.RoleCUSTOMER_SERVICE),
	).WithUser().Order(partnerassignmentent.ByRole(), partnerassignmentent.BySortOrder()).All(ctx)
	if err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}
	builders := make([]*ent.OrderCommissionAttributionCreate, 0, len(assignments))
	seenAttributions := make(map[string]struct{}, len(assignments))
	for _, item := range assignments {
		role := ordercommissionattributionent.PersonnelRole(item.Role)
		key := item.UserID.String() + ":" + string(role)
		if _, exists := seenAttributions[key]; exists {
			continue
		}
		employee, edgeErr := item.Edges.UserOrErr()
		if edgeErr != nil {
			return edgeErr
		}
		builders = append(builders, tx.OrderCommissionAttribution.Create().
			SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(organizationID).SetOrderID(orderID).SetCustomerID(customerID).
			SetSourceAssignmentID(item.ID).SetEmployeeID(item.UserID).SetEmployeeName(employee.DisplayName).
			SetPersonnelRole(role).SetAttributedAt(attributedAt))
		seenAttributions[key] = struct{}{}
	}
	if len(builders) == 0 {
		return nil
	}
	_, err = tx.OrderCommissionAttribution.CreateBulk(builders...).Save(ctx)
	return err
}

func (r *orderRepo) UpdateDraft(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, input *biz.Order) (*biz.Order, error) {
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
	if existing.Version != expectedVersion {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if existing.FlowStatus != orderent.FlowStatusDRAFT || existing.TerminationStatus != orderent.TerminationStatusACTIVE || existing.ClosureStatus != orderent.ClosureStatusOPEN {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if existing.BusinessType != orderent.BusinessType(input.BusinessType) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderBusinessUnsupported
	}
	if err := validateOrderReferences(ctx, tx, organizationID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := existing.Update().
		SetVersion(existing.Version + 1).
		SetCustomerID(input.CustomerID).
		SetCustomerReferenceNo(input.CustomerReferenceNo).
		SetInternalReferenceNo(input.InternalReferenceNo).
		SetShipperShortName(input.ShipperShortName).
		SetConsigneeShortName(input.ConsigneeShortName).
		SetTags(input.Tags).
		SetContractNo(input.ContractNo).
		SetCargoValue(input.CargoValue).
		SetInsurancePremium(input.InsurancePremium).
		SetUnNumber(input.UNNumber).
		SetHazardClass(input.HazardClass).
		SetFactoryName(input.FactoryName).
		SetCargoReadyAt(input.CargoReadyAt).
		SetLoadingTerms(input.LoadingTerms).
		SetDeclarationCutoffAt(input.DeclarationCutoffAt).
		SetReceivedAt(input.ReceivedAt).
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
		SetNotes(input.Notes).
		SetBookingNotes(input.BookingNotes).
		SetAllocationNotes(input.AllocationNotes).
		SetOperationNotes(input.OperationNotes)
	setOrderOptionalReferences(update, input)
	setOrderOptionalAmounts(update, input)
	if input.TotalPackages == nil {
		update.ClearTotalPackages()
	} else {
		update.SetTotalPackages(*input.TotalPackages)
	}
	if input.TotalGrossWeightKg == nil {
		update.ClearTotalGrossWeightKg()
	} else {
		update.SetTotalGrossWeightKg(*input.TotalGrossWeightKg)
	}
	if input.TotalVolumeCbm == nil {
		update.ClearTotalVolumeCbm()
	} else {
		update.SetTotalVolumeCbm(*input.TotalVolumeCbm)
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, mapOrderConstraint(err)
	}
	if err := replaceOrderSelections(ctx, tx, id, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderShippingDocuments(ctx, tx, organizationID, input.BusinessType, id, input.ShippingDocuments); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderContainerRequests(ctx, tx, organizationID, id, input.ContainerRequests); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if existing.CustomerID != input.CustomerID {
		if _, err := tx.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrderIDEQ(id)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if err := snapshotOrderCommissionAttributions(ctx, tx, organizationID, id, input.CustomerID, existing.CreatedAt); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionStatus(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, targetStatus biz.OrderFlowStatus, reason string, actorID uuid.UUID, event *biz.OrderStatusChangedEvent) (*biz.Order, error) {
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
	if existing.Version != expectedVersion || biz.OrderFlowStatus(existing.FlowStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if _, err := existing.Update().SetFlowStatus(orderent.FlowStatus(targetStatus)).SetVersion(existing.Version + 1).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionFLOW).SetFromStatus(string(event.FromStatus)).SetToStatus(string(targetStatus)).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
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

func (r *orderRepo) TransitionTermination(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderTerminationStatus, terminationType *biz.OrderTerminationType, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
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
	if existing.Version != expectedVersion || string(existing.TerminationStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	update := existing.Update().SetTerminationStatus(orderent.TerminationStatus(target)).SetVersion(existing.Version + 1)
	if target == biz.OrderTerminationActive {
		update.ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy()
	} else {
		update.SetTerminationType(orderent.TerminationType(*terminationType)).SetTerminationReason(reason)
		if target == biz.OrderTerminationTerminated {
			update.SetTerminatedAt(event.OccurredAt).SetTerminatedBy(actorID)
		} else {
			update.ClearTerminatedAt().ClearTerminatedBy()
		}
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionTERMINATION).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
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

func (r *orderRepo) ClosureReadiness(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderClosureReadiness, error) {
	item, err := r.data.db.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	hasActiveException, err := r.data.db.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	hasUnbilledFees, err := r.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.OrderClosureReadiness{FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), HasActiveException: hasActiveException, HasUnbilledOrderFees: hasUnbilledFees}, nil
}

func (r *orderRepo) TransitionClosure(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderClosureStatus, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
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
	if existing.Version != expectedVersion || string(existing.ClosureStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if target == biz.OrderClosureClosed {
		flowFinished := biz.OrderFlowStatus(existing.FlowStatus) == biz.OrderFlowDocumentReleased
		terminated := biz.OrderTerminationStatus(existing.TerminationStatus) == biz.OrderTerminationTerminated
		if !flowFinished && !terminated {
			_ = tx.Rollback()
			return nil, biz.ErrOrderClosureBlocked
		}
		hasActiveException, err := tx.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		hasUnbilledFees, err := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if hasActiveException || hasUnbilledFees {
			_ = tx.Rollback()
			return nil, biz.ErrOrderClosureBlocked
		}
	}
	update := existing.Update().SetClosureStatus(orderent.ClosureStatus(target)).SetVersion(existing.Version + 1)
	if target == biz.OrderClosureClosed {
		update.SetClosureReason(reason).SetClosedAt(event.OccurredAt).SetClosedBy(actorID)
	} else {
		update.ClearClosureReason().ClearClosedAt().ClearClosedBy()
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionCLOSURE).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
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
		if err := validatePartnerRole(ctx, tx, organizationID, *input.BookingAgentID, partnerroleent.RoleTypeSupplier); err != nil {
			return biz.ErrOrderInvalidArgument
		}
	}
	if input.ForeignAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.ForeignAgentID, partnerroleent.RoleTypeForeignAgent); err != nil {
			return biz.ErrOrderInvalidArgument
		}
	}
	if input.ShippingAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.ShippingAgentID, partnerroleent.RoleTypeSupplier); err != nil {
			return biz.ErrOrderInvalidArgument
		}
	}
	if input.CargoCurrency != "" {
		validCurrency, err := tx.Currency.Query().Where(
			currencyent.CodeEQ(input.CargoCurrency),
			currencyent.EnabledEQ(true),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !validCurrency {
			return biz.ErrOrderInvalidArgument
		}
	}
	if input.InsuranceCurrency != "" {
		validCurrency, err := tx.Currency.Query().Where(
			currencyent.CodeEQ(input.InsuranceCurrency),
			currencyent.EnabledEQ(true),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !validCurrency {
			return biz.ErrOrderInvalidArgument
		}
	}
	if err := validateMasterDataIDs(ctx, tx, organizationID, input.ServiceTypeIDs, masterdataent.KindServiceType); err != nil {
		return err
	}
	if err := validateMasterDataIDs(ctx, tx, organizationID, input.CargoCategoryIDs, masterdataent.KindCargoCategory); err != nil {
		return err
	}
	locationIDs := nonNilUUIDs(input.OriginLocationID, input.DestinationLocationID, input.DischargeLocationID, input.TransitLocationID)
	if input.BusinessType == biz.OrderBusinessSE || input.BusinessType == biz.OrderBusinessSI {
		return validatePortIDs(ctx, tx, organizationID, locationIDs)
	} else if input.BusinessType == biz.OrderBusinessAE || input.BusinessType == biz.OrderBusinessAI {
		return validateAirportIDs(ctx, tx, organizationID, locationIDs)
	}
	return validateMasterDataIDs(ctx, tx, organizationID, locationIDs, masterdataent.KindRegion)
}

func validatePortIDs(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := tx.Port.Query().Where(portent.IDIn(ids...), portent.OrganizationIDEQ(organizationID), portent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return biz.ErrOrderInvalidArgument
	}
	return nil
}

func validateAirportIDs(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := tx.Airport.Query().Where(airportent.IDIn(ids...), airportent.OrganizationIDEQ(organizationID), airportent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return biz.ErrOrderInvalidArgument
	}
	return nil
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

func createOrderPersonnel(ctx context.Context, tx *ent.Tx, rootOrganizationID, orderID uuid.UUID, orderNo string, assignments []*biz.OrderPersonnel) error {
	organizations, err := tx.Organization.Query().Select(organizationent.FieldID, organizationent.FieldParentID).All(ctx)
	if err != nil {
		return err
	}
	parentByID := make(map[uuid.UUID]*uuid.UUID, len(organizations))
	for _, organization := range organizations {
		parentByID[organization.ID] = organization.ParentID
	}
	for _, assignment := range assignments {
		if !organizationWithinRoot(parentByID, rootOrganizationID, assignment.OrganizationID) {
			return biz.ErrOrderPersonnelUserInvalid
		}
		membership, err := tx.Membership.Query().Where(
			membershipent.UserIDEQ(assignment.UserID),
			membershipent.OrganizationIDEQ(assignment.OrganizationID),
			membershipent.EnabledEQ(true),
			membershipent.HasUserWith(userent.EnabledEQ(true)),
		).WithUser().Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrOrderPersonnelUserInvalid
			}
			return err
		}
		user, err := membership.Edges.UserOrErr()
		if err != nil {
			return err
		}
		if _, err := tx.OrderPersonnel.Create().
			SetOrderID(orderID).
			SetUserID(assignment.UserID).
			SetOrganizationID(assignment.OrganizationID).
			SetRole(orderpersonnelent.Role(assignment.Role)).
			Save(ctx); err != nil {
			return err
		}
		if err := enqueueOrderPersonnelNotification(ctx, tx, rootOrganizationID, orderID, orderNo, assignment.Role, user, assignment.Notification); err != nil {
			return err
		}
	}
	return nil
}

func syncOrderShippingDocuments(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, businessType biz.OrderBusinessType, orderID uuid.UUID, inputs []*biz.OrderShippingDocument) error {
	existing, err := tx.OrderShippingDocument.Query().Where(ordershippingdocumentent.OrderIDEQ(orderID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	remaining := make(map[uuid.UUID]*ent.OrderShippingDocument, len(existing))
	for _, item := range existing {
		remaining[item.ID] = item
	}
	for _, input := range inputs {
		consolidation, err := resolveOrderConsolidation(ctx, tx, organizationID, businessType, input)
		if err != nil {
			return err
		}
		if input.ID == uuid.Nil {
			builder := tx.OrderShippingDocument.Create().SetID(uuid.Must(uuid.NewV7())).SetOrderID(orderID).SetConsolidationID(consolidation.ID).SetHouseNo(input.HouseNo).SetStatus(ordershippingdocumentent.StatusDRAFT)
			setShippingDocumentOptionalFieldsOnCreate(builder, input)
			if _, err := builder.Save(ctx); err != nil {
				return mapShippingDocumentConstraint(err)
			}
			continue
		}
		item, ok := remaining[input.ID]
		if !ok {
			return biz.ErrOrderShippingDocumentNotFound
		}
		if item.Status == ordershippingdocumentent.StatusRELEASED {
			return biz.ErrOrderShippingDocumentInvalidStatus
		}
		builder := item.Update().SetConsolidationID(consolidation.ID).SetHouseNo(input.HouseNo)
		setShippingDocumentOptionalFieldsOnUpdate(builder, input)
		if _, err := builder.Save(ctx); err != nil {
			return mapShippingDocumentConstraint(err)
		}
		delete(remaining, input.ID)
	}
	for _, item := range remaining {
		if item.Status == ordershippingdocumentent.StatusRELEASED {
			return biz.ErrOrderShippingDocumentInvalidStatus
		}
		if err := tx.OrderShippingDocument.DeleteOneID(item.ID).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func resolveOrderConsolidation(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, businessType biz.OrderBusinessType, input *biz.OrderShippingDocument) (*ent.OrderConsolidation, error) {
	normalizedMasterNo := strings.ToLower(input.MasterNo)
	consolidation, err := tx.OrderConsolidation.Query().Where(
		orderconsolidationent.OrganizationIDEQ(organizationID),
		orderconsolidationent.BusinessTypeEQ(orderconsolidationent.BusinessType(businessType)),
		orderconsolidationent.NormalizedMasterNoEQ(normalizedMasterNo),
	).Only(ctx)
	if err == nil {
		builder := consolidation.Update()
		changed := false
		if input.MasterDocumentType != nil {
			builder.SetDocumentType(*input.MasterDocumentType)
			changed = true
		}
		if input.MasterReleaseMethod != nil {
			builder.SetReleaseMethod(*input.MasterReleaseMethod)
			changed = true
		}
		if changed {
			return builder.Save(ctx)
		}
		return consolidation, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	builder := tx.OrderConsolidation.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetBusinessType(orderconsolidationent.BusinessType(businessType)).
		SetMasterNo(input.MasterNo).
		SetNormalizedMasterNo(normalizedMasterNo)
	if input.MasterDocumentType != nil {
		builder.SetDocumentType(*input.MasterDocumentType)
	}
	if input.MasterReleaseMethod != nil {
		builder.SetReleaseMethod(*input.MasterReleaseMethod)
	}
	return builder.Save(ctx)
}

func setShippingDocumentOptionalFieldsOnCreate(builder *ent.OrderShippingDocumentCreate, input *biz.OrderShippingDocument) {
	if input.ReleaseType != nil {
		builder.SetReleaseType(*input.ReleaseType)
	}
	if input.Note != nil {
		builder.SetNote(*input.Note)
	}
}

func setShippingDocumentOptionalFieldsOnUpdate(builder *ent.OrderShippingDocumentUpdateOne, input *biz.OrderShippingDocument) {
	if input.ReleaseType == nil {
		builder.ClearReleaseType()
	} else {
		builder.SetReleaseType(*input.ReleaseType)
	}
	if input.Note == nil {
		builder.ClearNote()
	} else {
		builder.SetNote(*input.Note)
	}
}

func mapShippingDocumentConstraint(err error) error {
	if ent.IsConstraintError(err) && strings.Contains(err.Error(), "ordershippingdocument_order_id_house_no") {
		return biz.ErrOrderShippingDocumentExists
	}
	return err
}

func syncOrderContainerRequests(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, inputs []*biz.OrderContainerRequest) error {
	for _, input := range inputs {
		exists, err := tx.MasterDataItem.Query().Where(
			masterdataent.IDEQ(input.ContainerSpecID),
			masterdataent.OrganizationIDEQ(organizationID),
			masterdataent.KindEQ(masterdataent.KindContainerSpec),
			masterdataent.EnabledEQ(true),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrOrderContainerSpecInvalid
		}
	}
	existing, err := tx.OrderContainerRequest.Query().Where(ordercontainerrequestent.OrderIDEQ(orderID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	remaining := make(map[uuid.UUID]*ent.OrderContainerRequest, len(existing))
	for _, item := range existing {
		remaining[item.ID] = item
	}
	for _, input := range inputs {
		if input.ID == uuid.Nil {
			if _, err := tx.OrderContainerRequest.Create().SetID(uuid.Must(uuid.NewV7())).SetOrderID(orderID).SetContainerSpecID(input.ContainerSpecID).SetQuantity(input.Quantity).Save(ctx); err != nil {
				return err
			}
			continue
		}
		item, ok := remaining[input.ID]
		if !ok {
			return biz.ErrOrderInvalidArgument
		}
		if _, err := item.Update().SetContainerSpecID(input.ContainerSpecID).SetQuantity(input.Quantity).Save(ctx); err != nil {
			return err
		}
		delete(remaining, input.ID)
	}
	for id := range remaining {
		if err := tx.OrderContainerRequest.DeleteOneID(id).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func withOrderEdges(query *ent.OrderQuery) *ent.OrderQuery {
	return query.
		WithOrganization().
		WithServiceTypes(func(q *ent.OrderServiceTypeQuery) { q.Order(orderserviceent.ByCreatedAt()) }).
		WithCargoCategories(func(q *ent.OrderCargoCategoryQuery) { q.Order(ordercargoent.ByCreatedAt()) }).
		WithShippingDocuments(func(q *ent.OrderShippingDocumentQuery) {
			q.WithConsolidation().Order(ordershippingdocumentent.ByCreatedAt())
		}).
		WithContainerRequests(func(q *ent.OrderContainerRequestQuery) { q.Order(ordercontainerrequestent.ByCreatedAt()) }).
		WithAbnormalCases(func(q *ent.OrderAbnormalCaseQuery) {
			q.Where(orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE))
		})
}

func orderToBiz(item *ent.Order) *biz.Order {
	result := &biz.Order{
		ID: item.ID, OrganizationID: item.OrganizationID, OrganizationName: item.Edges.Organization.Name, OrderNo: item.OrderNo, CustomerID: item.CustomerID,
		CarrierID: item.CarrierID, BookingAgentID: item.BookingAgentID, ForeignAgentID: item.ForeignAgentID, ShippingAgentID: item.ShippingAgentID, BusinessType: biz.OrderBusinessType(item.BusinessType),
		CustomerReferenceNo: item.CustomerReferenceNo, InternalReferenceNo: item.InternalReferenceNo, ContractNo: item.ContractNo, CargoValue: item.CargoValue, CargoCurrency: item.CargoCurrency,
		ShipperShortName: item.ShipperShortName, ConsigneeShortName: item.ConsigneeShortName, LockedAt: item.LockedAt, IsShared: item.IsShared, Tags: append([]string(nil), item.Tags...),
		InsurancePremium: item.InsurancePremium, InsuranceCurrency: item.InsuranceCurrency, UNNumber: item.UnNumber, HazardClass: item.HazardClass, FactoryName: item.FactoryName, CargoReadyAt: item.CargoReadyAt, LoadingTerms: item.LoadingTerms,
		DeclarationCutoffAt: item.DeclarationCutoffAt, ReceivedAt: item.ReceivedAt,
		TradeDirection: biz.OrderTradeDirection(item.TradeDirection), TradeTerm: biz.OrderTradeTerm(item.TradeTerm), PaymentTerm: biz.OrderPaymentTerm(item.PaymentTerm),
		FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), TerminationReason: orderOptionalStringValue(item.TerminationReason),
		TerminatedAt: item.TerminatedAt, TerminatedBy: item.TerminatedBy, ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), ClosureReason: orderOptionalStringValue(item.ClosureReason),
		ClosedAt: item.ClosedAt, ClosedBy: item.ClosedBy, Version: item.Version, HasActiveException: len(item.Edges.AbnormalCases) > 0, ActiveExceptionCount: len(item.Edges.AbnormalCases),
		OriginLocationID: item.OriginLocationID, DestinationLocationID: item.DestinationLocationID,
		DischargeLocationID: item.DischargeLocationID, TransitLocationID: item.TransitLocationID, VesselVoyage: item.VesselVoyage, ETD: item.Etd, ETA: item.Eta,
		SICutoff: item.SiCutoff, DocCutoff: item.DocCutoff, CustomsCutoff: item.CustomsCutoff, VGMCutoff: item.VgmCutoff,
		GoodsDescription: item.GoodsDescription, TotalPackages: item.TotalPackages, TotalGrossWeightKg: item.TotalGrossWeightKg, TotalVolumeCbm: item.TotalVolumeCbm, TotalPackageUnit: item.TotalPackageUnit,
		SpecialRequirements: item.SpecialRequirements, OrderDate: item.OrderDate, Notes: item.Notes,
		BookingNotes: item.BookingNotes, AllocationNotes: item.AllocationNotes, OperationNotes: item.OperationNotes,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.ShipmentType != nil {
		value := biz.OrderShipmentType(*item.ShipmentType)
		result.ShipmentType = &value
	}
	if item.TerminationType != nil {
		value := biz.OrderTerminationType(*item.TerminationType)
		result.TerminationType = &value
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
	result.ShippingDocuments = make([]*biz.OrderShippingDocument, 0, len(item.Edges.ShippingDocuments))
	for _, document := range item.Edges.ShippingDocuments {
		result.ShippingDocuments = append(result.ShippingDocuments, orderShippingDocumentToBiz(document))
	}
	result.ContainerRequests = make([]*biz.OrderContainerRequest, 0, len(item.Edges.ContainerRequests))
	for _, request := range item.Edges.ContainerRequests {
		result.ContainerRequests = append(result.ContainerRequests, &biz.OrderContainerRequest{
			ID: request.ID, OrderID: request.OrderID, ContainerSpecID: request.ContainerSpecID,
			Quantity: request.Quantity, CreatedAt: request.CreatedAt, UpdatedAt: request.UpdatedAt,
		})
	}
	result.AllowedActions = orderAllowedActions(result)
	return result
}

func orderAllowedActions(order *biz.Order) []biz.OrderAllowedAction {
	if order.ClosureStatus == biz.OrderClosureClosed {
		return []biz.OrderAllowedAction{biz.OrderActionReopen}
	}
	result := make([]biz.OrderAllowedAction, 0, 4)
	if order.TerminationStatus == biz.OrderTerminationActive {
		if order.FlowStatus == biz.OrderFlowDraft {
			result = append(result, biz.OrderActionEdit)
		}
		if order.FlowStatus != biz.OrderFlowDocumentReleased {
			result = append(result, biz.OrderActionTransitionFlow)
		}
		result = append(result, biz.OrderActionStartTermination)
	} else if order.TerminationStatus == biz.OrderTerminationTerminating {
		result = append(result, biz.OrderActionCompleteTermination, biz.OrderActionCancelTermination)
	} else {
		result = append(result, biz.OrderActionCancelTermination)
	}
	if !order.HasActiveException && (order.FlowStatus == biz.OrderFlowDocumentReleased || order.TerminationStatus == biz.OrderTerminationTerminated) {
		result = append(result, biz.OrderActionClose)
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
	if input.ForeignAgentID == nil {
		update.ClearForeignAgentID()
	} else {
		update.SetForeignAgentID(*input.ForeignAgentID)
	}
	if input.ShippingAgentID == nil {
		update.ClearShippingAgentID()
	} else {
		update.SetShippingAgentID(*input.ShippingAgentID)
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

func setOrderOptionalAmounts(update *ent.OrderUpdateOne, input *biz.Order) {
	if input.CargoCurrency == "" {
		update.ClearCargoCurrency()
	} else {
		update.SetCargoCurrency(input.CargoCurrency)
	}
	if input.InsuranceCurrency == "" {
		update.ClearInsuranceCurrency()
	} else {
		update.SetInsuranceCurrency(input.InsuranceCurrency)
	}
}

func nonEmptyStringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func orderOptionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
