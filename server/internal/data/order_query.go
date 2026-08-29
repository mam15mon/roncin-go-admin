package data

import (
	"context"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderconsolidationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderconsolidation"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	entpredicate "github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

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
