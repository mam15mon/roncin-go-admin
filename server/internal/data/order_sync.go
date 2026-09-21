package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airportent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	ordercargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargocategory"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	orderserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

func validateOrderReferences(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, input *biz.Order, existing *ent.Order) error {
	if err := validateOrderPartnerReferences(ctx, tx, organizationID, input, existing); err != nil {
		return err
	}
	if input.ShippingLineID != nil {
		predicates := []predicate.ShippingLine{
			shippinglineent.IDEQ(*input.ShippingLineID),
		}
		if existing == nil || existing.ShippingLineID == nil || *existing.ShippingLineID != *input.ShippingLineID {
			predicates = append(predicates, shippinglineent.EnabledEQ(true))
		}
		exists, err := tx.ShippingLine.Query().Where(predicates...).ForShare().Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
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
	if err := validateMasterDataIDs(ctx, tx, organizationID, input.ServiceTypeIDs, masterdataent.KindChargeCategory); err != nil {
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

type orderPartnerReference struct {
	partnerID      uuid.UUID
	roleType       partnerroleent.RoleType
	checkBlacklist bool
	invalidErr     error
}

func validateOrderPartnerReferences(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, input *biz.Order, existing *ent.Order) error {
	references := []orderPartnerReference{{
		partnerID: input.CustomerID, roleType: partnerroleent.RoleTypeCustomer,
		checkBlacklist: existing == nil || existing.CustomerID != input.CustomerID,
		invalidErr:     biz.ErrOrderCustomerInvalid,
	}}
	appendOptional := func(inputID, existingID *uuid.UUID, roleType partnerroleent.RoleType) {
		if inputID == nil {
			return
		}
		references = append(references, orderPartnerReference{
			partnerID: *inputID, roleType: roleType,
			checkBlacklist: existing == nil || existingID == nil || *existingID != *inputID,
			invalidErr:     biz.ErrOrderInvalidArgument,
		})
	}
	var existingBookingAgentID, existingForeignAgentID, existingShippingAgentID *uuid.UUID
	if existing != nil {
		existingBookingAgentID = existing.BookingAgentID
		existingForeignAgentID = existing.ForeignAgentID
		existingShippingAgentID = existing.ShippingAgentID
	}
	appendOptional(input.BookingAgentID, existingBookingAgentID, partnerroleent.RoleTypeSupplier)
	appendOptional(input.ForeignAgentID, existingForeignAgentID, partnerroleent.RoleTypeForeignAgent)
	appendOptional(input.ShippingAgentID, existingShippingAgentID, partnerroleent.RoleTypeSupplier)

	partnerIDsByValue := make(map[uuid.UUID]struct{}, len(references))
	for _, reference := range references {
		partnerIDsByValue[reference.partnerID] = struct{}{}
	}
	partnerIDs := make([]uuid.UUID, 0, len(partnerIDsByValue))
	for partnerID := range partnerIDsByValue {
		partnerIDs = append(partnerIDs, partnerID)
	}
	sort.Slice(partnerIDs, func(i, j int) bool { return partnerIDs[i].String() < partnerIDs[j].String() })
	lockedPartners, err := tx.Partner.Query().Where(
		partnerent.IDIn(partnerIDs...),
		partnerent.OrganizationIDEQ(organizationID),
		partnerent.EnabledEQ(true),
	).Order(partnerent.ByID()).ForShare().All(ctx)
	if err != nil {
		return err
	}
	if len(lockedPartners) != len(partnerIDs) {
		for _, reference := range references {
			found := false
			for _, partner := range lockedPartners {
				if partner.ID == reference.partnerID {
					found = true
					break
				}
			}
			if !found {
				return reference.invalidErr
			}
		}
	}

	roles, err := tx.PartnerRole.Query().Where(
		partnerroleent.PartnerIDIn(partnerIDs...),
		partnerroleent.EnabledEQ(true),
	).All(ctx)
	if err != nil {
		return err
	}
	rolesByKey := make(map[string]*ent.PartnerRole, len(roles))
	for _, role := range roles {
		rolesByKey[role.PartnerID.String()+":"+string(role.RoleType)] = role
	}
	for _, reference := range references {
		role := rolesByKey[reference.partnerID.String()+":"+string(reference.roleType)]
		if role == nil {
			return reference.invalidErr
		}
		if reference.checkBlacklist && role.Blacklisted {
			return biz.NewOrderPartnerRoleBlacklisted(biz.PartnerRoleType(reference.roleType))
		}
	}
	return nil
}

func validateMasterDataIDs(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, ids []uuid.UUID, kind masterdataent.Kind) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := tx.MasterDataItem.Query().Where(masterdataent.IDIn(ids...), masterdataent.KindEQ(kind), masterdataent.EnabledEQ(true)).Count(ctx)
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
	subtreeOrganizationIDs, err := companyPersonnelOrganizationIDs(ctx, tx.Client(), rootOrganizationID)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		user, err := tx.User.Query().Where(
			userent.IDEQ(assignment.UserID),
			userent.EnabledEQ(true),
			userent.HasMembershipsWith(
				membershipent.OrganizationIDIn(subtreeOrganizationIDs...),
				membershipent.EnabledEQ(true),
			),
		).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderPersonnelUserInvalid, nil)
		}
		if _, err := tx.OrderPersonnel.Create().
			SetOrderID(orderID).
			SetUserID(assignment.UserID).
			SetOrganizationID(rootOrganizationID).
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
	if businessType == biz.OrderBusinessSE && len(inputs) > 0 {
		return biz.ErrSeaShippingDocumentsDeprecated
	}
	existing, err := tx.OrderShippingDocument.Query().Where(ordershippingdocumentent.OrderIDEQ(orderID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	remaining := make(map[uuid.UUID]*ent.OrderShippingDocument, len(existing))
	for _, item := range existing {
		remaining[item.ID] = item
	}
	for _, input := range inputs {
		if input.ID == uuid.Nil {
			builder := tx.OrderShippingDocument.Create().SetID(uuid.Must(uuid.NewV7())).SetOrderID(orderID).SetHouseNo(input.HouseNo).SetStatus(ordershippingdocumentent.StatusDRAFT)
			setShippingDocumentOptionalFieldsOnCreate(builder, input)
			if _, err := builder.Save(ctx); err != nil {
				return mapEntConstraint(err, "ordershippingdocument_order_id_house_no", biz.ErrOrderShippingDocumentExists)
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
		builder := item.Update().SetHouseNo(input.HouseNo)
		setShippingDocumentOptionalFieldsOnUpdate(builder, input)
		if _, err := builder.Save(ctx); err != nil {
			return mapEntConstraint(err, "ordershippingdocument_order_id_house_no", biz.ErrOrderShippingDocumentExists)
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

func syncOrderContainerRequests(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, inputs []*biz.OrderContainerRequest) error {
	for _, input := range inputs {
		exists, err := tx.MasterDataItem.Query().Where(
			masterdataent.IDEQ(input.ContainerSpecID),
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
