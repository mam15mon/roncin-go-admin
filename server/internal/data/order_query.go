package data

import (
	"context"
	"sort"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordertaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	entpredicate "github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

func (r *orderRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := withOrderEdges(client.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID))).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	return orderToBiz(item), nil
}

func (r *orderRepo) FindAuthorized(ctx context.Context, id uuid.UUID, scopes []biz.OrderOrganizationScope) (*biz.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := withOrderEdges(client.Order.Query().Where(orderent.IDEQ(id), orderOrganizationScopePredicate(scopes))).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	return orderToBiz(item), nil
}

func (r *orderRepo) List(ctx context.Context, scopes []biz.OrderOrganizationScope, options biz.OrderListOptions) (*biz.OrderList, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Order.Query().Where(orderOrganizationScopePredicate(scopes))
	if options.Keyword != "" {
		query.Where(orderent.Or(orderent.OrderNoContainsFold(options.Keyword), orderent.VesselVoyageContainsFold(options.Keyword), orderent.GoodsDescriptionContainsFold(options.Keyword)))
	}
	if options.NumberKeyword != "" {
		switch options.NumberType {
		case biz.OrderNumberFilterOrder:
			query.Where(orderent.OrderNoContainsFold(options.NumberKeyword))
		case biz.OrderNumberFilterMaster:
			query.Where(orderent.HasSeaMasterBillLinksWith(
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
				seamasterbillorderlink.HasMasterBillWith(seamasterbill.MasterNoContainsFold(options.NumberKeyword)),
			))
		case biz.OrderNumberFilterConsolidatedMaster:
			query.Where(orderConsolidatedMasterContainsFold(options.NumberKeyword))
		case biz.OrderNumberFilterCustomerReference:
			query.Where(orderent.CustomerReferenceNoContainsFold(options.NumberKeyword))
		case biz.OrderNumberFilterBooking:
			query.Where(orderent.BookingNoContainsFold(options.NumberKeyword))
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
	if options.ShippingLineID != nil {
		query.Where(orderent.ShippingLineIDEQ(*options.ShippingLineID))
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
	if len(options.TagIDs) > 0 {
		query.Where(orderent.HasEnterpriseTagLinksWith(ordertaglinkent.TagResourceIDIn(options.TagIDs...)))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.Order, error) {
		return withOrderEdges(query).Order(orderent.ByCreatedAt(entsql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(orderToBiz))
}

func orderOrganizationScopePredicate(scopes []biz.OrderOrganizationScope) entpredicate.Order {
	predicates := make([]entpredicate.Order, 0, len(scopes))
	for _, scope := range scopes {
		predicates = append(predicates, orderent.And(
			orderent.BusinessTypeEQ(orderent.BusinessType(scope.BusinessType)),
			orderent.OrganizationIDIn(scope.OrganizationIDs...),
		))
	}
	return orderent.Or(predicates...)
}

func orderConsolidatedMasterContainsFold(keyword string) entpredicate.Order {
	return entpredicate.Order(func(selector *entsql.Selector) {
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			builder.WriteString(`EXISTS (SELECT 1 FROM "sea_master_bill_order_links" AS "filter_link" JOIN "sea_master_bills" AS "filter_mbl" ON "filter_mbl"."id" = "filter_link"."master_bill_id" WHERE "filter_link"."order_id" = `).
				Ident(selector.C(orderent.FieldID)).
				WriteString(` AND "filter_link"."status" = 'ACTIVE' AND LOWER("filter_mbl"."master_no") LIKE `).
				Arg("%" + strings.ToLower(keyword) + "%").
				WriteString(` AND (SELECT COUNT(*) FROM "sea_master_bill_order_links" AS "sub_link" WHERE "sub_link"."master_bill_id" = "filter_mbl"."id" AND "sub_link"."status" = 'ACTIVE') > 1)`)
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
	query.Where(orderent.HasPersonnelWith(
		orderpersonnelent.RoleEQ(role),
		orderpersonnelent.UserIDEQ(*filter.UserID),
	))
}

func (r *orderRepo) FindReferenceDuplicate(ctx context.Context, organizationID uuid.UUID, check biz.OrderReferenceCheck) (*biz.OrderReferenceMatch, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Order.Query().Where(orderent.OrganizationIDEQ(organizationID))
	switch check.ReferenceType {
	case biz.OrderReferenceCustomer:
		query.Where(
			orderent.CustomerIDEQ(*check.CustomerID),
			orderent.CustomerReferenceNoEqualFold(check.ReferenceNo),
		)
	case biz.OrderReferenceInternal:
		query.Where(orderent.InternalReferenceNoEqualFold(check.ReferenceNo))
	case biz.OrderReferenceBooking:
		query.Where(orderent.BookingNoEqualFold(check.ReferenceNo))
	default:
		return nil, biz.ErrOrderInvalidArgument
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
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	return client.Order.Query().Where(
		orderent.IDEQ(orderID),
		orderent.OrganizationIDEQ(organizationID),
		orderent.HasContainers(),
	).Exist(ctx)
}

func (r *orderRepo) ListConsolidationSummaries(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderConsolidationSummary, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	current, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if current.ShipmentType == nil || *current.ShipmentType != orderent.ShipmentTypeLCL {
		return nil, biz.ErrOrderConsolidationShipmentType
	}
	activeLinks, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(orderID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithMasterBill(func(query *ent.SeaMasterBillQuery) {
			query.Where(seamasterbill.OrganizationIDEQ(organizationID))
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(activeLinks) == 0 {
		return []*biz.OrderConsolidationSummary{}, nil
	}
	mblIDs := make([]uuid.UUID, 0, len(activeLinks))
	mblMap := make(map[uuid.UUID]*ent.SeaMasterBill, len(activeLinks))
	for _, link := range activeLinks {
		if link.Edges.MasterBill != nil {
			mblIDs = append(mblIDs, link.MasterBillID)
			mblMap[link.MasterBillID] = link.Edges.MasterBill
		}
	}
	if len(mblIDs) == 0 {
		return []*biz.OrderConsolidationSummary{}, nil
	}

	// 成员集合以 Link ACTIVE 定位，成员 Order 额外要求 LCL 且终止维度 ACTIVE：
	// 退关订单不计入件重尺、成员数与 house numbers。HBL 从当前真相源
	// SeaHouseBill 读取，且必须属于该成员当前活动 MBL，不再依赖旧
	// OrderShippingDocument。
	allLinks, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.MasterBillIDIn(mblIDs...),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithOrder(func(oq *ent.OrderQuery) {
			oq.Where(
				orderent.ShipmentTypeEQ(orderent.ShipmentTypeLCL),
				orderent.TerminationStatusEQ(orderent.TerminationStatusACTIVE),
			).
				WithCargoItems().
				WithSeaHouseBills(func(hq *ent.SeaHouseBillQuery) {
					hq.Where(
						seahousebill.OrganizationIDEQ(organizationID),
						seahousebill.MasterBillIDIn(mblIDs...),
						seahousebill.StatusNEQ(seahousebill.StatusVOIDED),
					)
				})
		}).
		Order(seamasterbillorderlink.ByStartedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.OrderConsolidationSummary, 0, len(mblIDs))
	for _, mblID := range mblIDs {
		mbl := mblMap[mblID]
		summary := &biz.OrderConsolidationSummary{ConsolidationID: mbl.ID, MasterNo: mbl.MasterNo}
		members := make(map[uuid.UUID]*biz.OrderConsolidationMember)
		for _, link := range allLinks {
			if link.MasterBillID != mblID || link.Edges.Order == nil {
				continue
			}
			orderItem := link.Edges.Order
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
				// 仅统计属于当前活动 MBL 的有效 HBL，按 house no 稳定排序并去重。
				houseSeen := make(map[string]struct{}, len(orderItem.Edges.SeaHouseBills))
				for _, hb := range orderItem.Edges.SeaHouseBills {
					if hb.MasterBillID != mblID {
						continue
					}
					if _, dup := houseSeen[hb.HouseNo]; dup {
						continue
					}
					houseSeen[hb.HouseNo] = struct{}{}
					member.HouseNos = append(member.HouseNos, hb.HouseNo)
				}
				sort.Strings(member.HouseNos)
				members[orderItem.ID] = member
				summary.Members = append(summary.Members, member)
			}
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
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	organizations, err := client.Organization.Query().
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
	membershipScope := []entpredicate.Membership{
		membershipent.OrganizationIDIn(organizationIDs...),
		membershipent.EnabledEQ(true),
		membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)),
	}
	query := client.User.Query().Where(
		userent.EnabledEQ(true),
		userent.HasMembershipsWith(membershipScope...),
	)
	if options.Keyword != "" {
		query.Where(userent.Or(
			userent.UsernameContainsFold(options.Keyword),
			userent.DisplayNameContainsFold(options.Keyword),
			userent.SearchKeywordsContainsFold(options.Keyword),
			userent.HasMembershipsWith(
				membershipent.OrganizationIDIn(organizationIDs...),
				membershipent.EnabledEQ(true),
				membershipent.HasOrganizationWith(
					organizationent.EnabledEQ(true),
					organizationent.Or(organizationent.CodeContainsFold(options.Keyword), organizationent.NameContainsFold(options.Keyword), organizationent.SearchKeywordsContainsFold(options.Keyword)),
				),
			),
		))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.User, error) {
		return query.Order(userent.ByDisplayName(), userent.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(func(item *ent.User) *biz.OrderPersonnelOption {
		return &biz.OrderPersonnelOption{UserID: item.ID, DisplayName: item.DisplayName}
	}))
}

func (r *orderRepo) ListSameBatchOrders(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.SameBatchOrderSummary, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	currentOrder, err := client.Order.Query().
		Where(
			orderent.IDEQ(orderID),
			orderent.OrganizationIDEQ(organizationID),
		).
		WithSeaMasterBillLinks(func(q *ent.SeaMasterBillOrderLinkQuery) {
			q.Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			)
		}).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}

	matchSourcesByOrderID := make(map[uuid.UUID]map[string]struct{})
	recordMatch := func(matchedID uuid.UUID, source string) {
		if matchedID == orderID {
			return
		}
		sources := matchSourcesByOrderID[matchedID]
		if sources == nil {
			sources = make(map[string]struct{})
			matchSourcesByOrderID[matchedID] = sources
		}
		sources[source] = struct{}{}
	}

	// 1. 客户业务号：org + customer_id + customer_reference_no (非空且 customer_id 有效)
	customerRef := strings.TrimSpace(currentOrder.CustomerReferenceNo)
	if customerRef != "" && currentOrder.CustomerID != uuid.Nil {
		matchedOrders, err := client.Order.Query().
			Where(
				orderent.OrganizationIDEQ(organizationID),
				orderent.CustomerIDEQ(currentOrder.CustomerID),
				orderent.CustomerReferenceNoEQ(customerRef),
				orderent.IDNEQ(orderID),
			).
			Select(orderent.FieldID).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range matchedOrders {
			recordMatch(o.ID, "CUSTOMER_REFERENCE")
		}
	}

	// 2. Booking No：org + booking_no (非空)
	bookingNo := strings.TrimSpace(currentOrder.BookingNo)
	if bookingNo != "" {
		matchedOrders, err := client.Order.Query().
			Where(
				orderent.OrganizationIDEQ(organizationID),
				orderent.BookingNoEQ(bookingNo),
				orderent.IDNEQ(orderID),
			).
			Select(orderent.FieldID).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range matchedOrders {
			recordMatch(o.ID, "BOOKING")
		}
	}

	// 3. 活动 MBL：org + master_bill_id (通过活动 Link 查找同 MBL 的其他订单)
	var activeMblIDs []uuid.UUID
	for _, link := range currentOrder.Edges.SeaMasterBillLinks {
		if link.Status == seamasterbillorderlink.StatusACTIVE && link.MasterBillID != uuid.Nil {
			activeMblIDs = append(activeMblIDs, link.MasterBillID)
		}
	}
	if len(activeMblIDs) > 0 {
		matchedLinks, err := client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.MasterBillIDIn(activeMblIDs...),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
				seamasterbillorderlink.OrderIDNEQ(orderID),
				seamasterbillorderlink.HasOrderWith(orderent.OrganizationIDEQ(organizationID)),
			).
			Select(seamasterbillorderlink.FieldOrderID).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range matchedLinks {
			recordMatch(l.OrderID, "MASTER")
		}
	}

	if len(matchSourcesByOrderID) == 0 {
		return []*biz.SameBatchOrderSummary{}, nil
	}

	allMatchedIDs := make([]uuid.UUID, 0, len(matchSourcesByOrderID))
	for id := range matchSourcesByOrderID {
		allMatchedIDs = append(allMatchedIDs, id)
	}

	orders, err := client.Order.Query().
		Where(
			orderent.OrganizationIDEQ(organizationID),
			orderent.IDIn(allMatchedIDs...),
		).
		WithSeaMasterBillLinks(func(q *ent.SeaMasterBillOrderLinkQuery) {
			q.Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
				WithMasterBill(func(mq *ent.SeaMasterBillQuery) {
					mq.Where(seamasterbill.OrganizationIDEQ(organizationID))
				})
		}).
		WithSeaHouseBills(func(q *ent.SeaHouseBillQuery) {
			q.Where(seahousebill.StatusNEQ(seahousebill.StatusVOIDED))
		}).
		Order(orderent.ByCreatedAt(entsql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.SameBatchOrderSummary, 0, len(orders))
	for _, o := range orders {
		var masterNo string
		for _, link := range o.Edges.SeaMasterBillLinks {
			if link.Status == seamasterbillorderlink.StatusACTIVE && link.Edges.MasterBill != nil {
				masterNo = link.Edges.MasterBill.MasterNo
				break
			}
		}
		var houseNo string
		for _, hb := range o.Edges.SeaHouseBills {
			if hb.Status != seahousebill.StatusVOIDED {
				houseNo = hb.HouseNo
				break
			}
		}
		sourcesSet := matchSourcesByOrderID[o.ID]
		var matchSources []string
		for _, s := range []string{"CUSTOMER_REFERENCE", "BOOKING", "MASTER"} {
			if _, ok := sourcesSet[s]; ok {
				matchSources = append(matchSources, s)
			}
		}
		var customerID *uuid.UUID
		if o.CustomerID != uuid.Nil {
			customerID = &o.CustomerID
		}
		result = append(result, &biz.SameBatchOrderSummary{
			OrderID:             o.ID,
			OrderNo:             o.OrderNo,
			CustomerID:          customerID,
			CustomerReferenceNo: o.CustomerReferenceNo,
			BookingNo:           o.BookingNo,
			MasterNo:            masterNo,
			HouseNo:             houseNo,
			FlowStatus:          biz.OrderFlowStatus(o.FlowStatus),
			MatchSources:        matchSources,
			TotalPackages:       o.TotalPackages,
			TotalGrossWeightKg:  o.TotalGrossWeightKg,
			TotalVolumeCbm:      o.TotalVolumeCbm,
			CreatedAt:           o.CreatedAt,
		})
	}
	return result, nil
}
