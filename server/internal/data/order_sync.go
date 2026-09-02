package data

import (
	"context"
	"errors"

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
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

func validateOrderReferences(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, input *biz.Order) error {
	if err := validatePartnerRole(ctx, tx, organizationID, input.CustomerID, partnerroleent.RoleTypeCustomer); err != nil {
		if errors.Is(err, biz.ErrOrderInvalidArgument) {
			return biz.ErrOrderCustomerInvalid
		}
		return err
	}
	if input.CarrierID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.CarrierID, partnerroleent.RoleTypeCarrier); err != nil {
			return err
		}
	}
	if input.BookingAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.BookingAgentID, partnerroleent.RoleTypeSupplier); err != nil {
			return err
		}
	}
	if input.ForeignAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.ForeignAgentID, partnerroleent.RoleTypeForeignAgent); err != nil {
			return err
		}
	}
	if input.ShippingAgentID != nil {
		if err := validatePartnerRole(ctx, tx, organizationID, *input.ShippingAgentID, partnerroleent.RoleTypeSupplier); err != nil {
			return err
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
			return mapEntError(err, biz.ErrOrderPersonnelUserInvalid, nil)
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
