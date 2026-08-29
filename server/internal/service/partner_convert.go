package service

import (
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func partnerImportModeFromAPI(value v1.PartnerImportMode) biz.PartnerImportMode {
	switch value {
	case v1.PartnerImportMode_PARTNER_IMPORT_MODE_CREATE_ONLY:
		return biz.PartnerImportCreateOnly
	case v1.PartnerImportMode_PARTNER_IMPORT_MODE_UPSERT:
		return biz.PartnerImportUpsert
	default:
		return ""
	}
}

func partnerAuditLogToAPI(value *biz.PartnerAuditLog) *v1.PartnerAuditLog {
	result := &v1.PartnerAuditLog{
		Id: value.Log.ID.String(), UserDisplayName: value.UserDisplayName, Action: value.Log.Action,
		Result: value.Log.Result, TraceId: value.Log.TraceID, Details: value.Log.Details, CreatedAt: value.Log.CreatedAt.Format(time.RFC3339),
	}
	if value.Log.UserID != nil {
		userID := value.Log.UserID.String()
		result.UserId = &userID
	}
	return result
}

func partnerRoleTypeFromAPI(value v1.PartnerRoleType) biz.PartnerRoleType {
	switch value {
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER:
		return biz.PartnerRoleCustomer
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER:
		return biz.PartnerRoleSupplier
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_FOREIGN_AGENT:
		return biz.PartnerRoleForeignAgent
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_CARRIER:
		return biz.PartnerRoleCarrier
	default:
		return ""
	}
}

func partnerRoleTypeToAPI(value biz.PartnerRoleType) v1.PartnerRoleType {
	switch value {
	case biz.PartnerRoleCustomer:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER
	case biz.PartnerRoleSupplier:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER
	case biz.PartnerRoleForeignAgent:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_FOREIGN_AGENT
	case biz.PartnerRoleCarrier:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_CARRIER
	default:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_UNSPECIFIED
	}
}

func partnerCustomerTypeFromAPI(value v1.PartnerCustomerType) biz.PartnerCustomerType {
	switch value {
	case v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_DIRECT:
		return biz.PartnerCustomerDirect
	case v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_PEER:
		return biz.PartnerCustomerPeer
	default:
		return ""
	}
}

func partnerCustomerTypeToAPI(value biz.PartnerCustomerType) v1.PartnerCustomerType {
	switch value {
	case biz.PartnerCustomerDirect:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_DIRECT
	case biz.PartnerCustomerPeer:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_PEER
	default:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_UNSPECIFIED
	}
}

func partnerBusinessTypeFromAPI(value v1.PartnerBusinessType) biz.PartnerBusinessType {
	switch value {
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SE:
		return biz.PartnerBusinessSE
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SI:
		return biz.PartnerBusinessSI
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AE:
		return biz.PartnerBusinessAE
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AI:
		return biz.PartnerBusinessAI
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_LAND:
		return biz.PartnerBusinessLand
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_RAIL:
		return biz.PartnerBusinessRail
	default:
		return ""
	}
}

func partnerBusinessTypeToAPI(value biz.PartnerBusinessType) v1.PartnerBusinessType {
	switch value {
	case biz.PartnerBusinessSE:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SE
	case biz.PartnerBusinessSI:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SI
	case biz.PartnerBusinessAE:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AE
	case biz.PartnerBusinessAI:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AI
	case biz.PartnerBusinessLand:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_LAND
	case biz.PartnerBusinessRail:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_RAIL
	default:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_UNSPECIFIED
	}
}

func partnerAssignmentRoleFromAPI(value v1.PartnerAssignmentRole) biz.PartnerAssignmentRole {
	switch value {
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CREATOR:
		return biz.PartnerAssignmentCreator
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_OPERATOR:
		return biz.PartnerAssignmentOperator
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_SALES:
		return biz.PartnerAssignmentSales
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE:
		return biz.PartnerAssignmentCustomerService
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_FINANCE:
		return biz.PartnerAssignmentFinance
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_COMMERCIAL:
		return biz.PartnerAssignmentCommercial
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT:
		return biz.PartnerAssignmentInternalContact
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_DOCUMENT:
		return biz.PartnerAssignmentDocument
	default:
		return ""
	}
}

func partnerAssignmentRoleToAPI(value biz.PartnerAssignmentRole) v1.PartnerAssignmentRole {
	switch value {
	case biz.PartnerAssignmentCreator:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CREATOR
	case biz.PartnerAssignmentOperator:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_OPERATOR
	case biz.PartnerAssignmentSales:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_SALES
	case biz.PartnerAssignmentCustomerService:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE
	case biz.PartnerAssignmentFinance:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_FINANCE
	case biz.PartnerAssignmentCommercial:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_COMMERCIAL
	case biz.PartnerAssignmentInternalContact:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT
	case biz.PartnerAssignmentDocument:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_DOCUMENT
	default:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_UNSPECIFIED
	}
}

func partnerRolesFromAPI(items []*v1.PartnerRoleInput) []*biz.PartnerRole {
	roles := make([]*biz.PartnerRole, 0, len(items))
	for _, item := range items {
		if item == nil {
			roles = append(roles, nil)
			continue
		}
		roles = append(roles, &biz.PartnerRole{
			Type: partnerRoleTypeFromAPI(item.GetType()), Enabled: item.GetEnabled(),
			SettlementRule: partnerSettlementRuleFromAPI(item.GetSettlementRule()),
		})
	}
	return roles
}

func partnerContactsFromAPI(items []*v1.PartnerContactInput) []*biz.PartnerContact {
	contacts := make([]*biz.PartnerContact, 0, len(items))
	for _, item := range items {
		if item == nil {
			contacts = append(contacts, nil)
			continue
		}
		contacts = append(contacts, &biz.PartnerContact{
			Name: item.GetName(), Phone: item.GetPhone(), Email: item.GetEmail(), Note: item.GetNote(), IsPrimary: item.GetIsPrimary(),
		})
	}
	return contacts
}

func partnerAliasesFromAPI(items []*v1.PartnerAliasInput) []*biz.PartnerAlias {
	aliases := make([]*biz.PartnerAlias, 0, len(items))
	for _, item := range items {
		if item == nil {
			aliases = append(aliases, nil)
			continue
		}
		aliases = append(aliases, &biz.PartnerAlias{AliasName: item.GetAliasName(), SortOrder: int(item.GetSortOrder())})
	}
	return aliases
}

func partnerProfileFromAPI(value *v1.PartnerProfile) *biz.PartnerProfile {
	if value == nil {
		return nil
	}
	customerTypes := make([]biz.PartnerCustomerType, 0, len(value.GetCustomerTypes()))
	for _, item := range value.GetCustomerTypes() {
		customerTypes = append(customerTypes, partnerCustomerTypeFromAPI(item))
	}
	businessTypes := make([]biz.PartnerBusinessType, 0, len(value.GetBusinessTypes()))
	for _, item := range value.GetBusinessTypes() {
		businessTypes = append(businessTypes, partnerBusinessTypeFromAPI(item))
	}
	return &biz.PartnerProfile{
		NameEN: value.GetNameEn(), AddressEN: value.GetAddressEn(), CountryCode: value.GetCountryCode(),
		ProvinceCode: value.GetProvinceCode(), CityCode: value.GetCityCode(), DistrictCode: value.GetDistrictCode(),
		AddressDetail: value.GetAddressDetail(), Nature: value.GetNature(), DevelopmentMethod: value.GetDevelopmentMethod(),
		CustomerTypes: customerTypes, BusinessTypes: businessTypes, Remark: value.GetRemark(),
	}
}

func partnerAssignmentsFromAPI(items []*v1.PartnerAssignmentInput) []*biz.PartnerAssignment {
	result := make([]*biz.PartnerAssignment, 0, len(items))
	for _, item := range items {
		if item == nil {
			result = append(result, nil)
			continue
		}
		userID, _ := uuid.Parse(item.GetUserId())
		organizationID, _ := uuid.Parse(item.GetOrganizationId())
		result = append(result, &biz.PartnerAssignment{
			Role: partnerAssignmentRoleFromAPI(item.GetRole()), UserID: userID, OrganizationID: organizationID,
		})
	}
	return result
}

func partnerToAPI(value *biz.Partner) *v1.Partner {
	roles := make([]*v1.PartnerRole, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, &v1.PartnerRole{
			Type: partnerRoleTypeToAPI(role.Type), Enabled: role.Enabled, Blacklisted: role.Blacklisted,
			BlacklistReason: role.BlacklistReason, BlacklistedAt: formatOptionalTime(role.BlacklistedAt), BlacklistedBy: formatOptionalUUID(role.BlacklistedBy),
			SettlementRule: partnerSettlementRuleToAPI(role.SettlementRule),
		})
	}
	contacts := make([]*v1.PartnerContact, 0, len(value.Contacts))
	for _, contact := range value.Contacts {
		contacts = append(contacts, &v1.PartnerContact{
			Id: contact.ID.String(), Name: contact.Name, Phone: contact.Phone, Email: contact.Email, Note: contact.Note,
			IsPrimary: contact.IsPrimary, CreatedAt: contact.CreatedAt.Format(time.RFC3339), UpdatedAt: contact.UpdatedAt.Format(time.RFC3339),
		})
	}
	aliases := make([]*v1.PartnerAlias, 0, len(value.Aliases))
	for _, alias := range value.Aliases {
		aliases = append(aliases, &v1.PartnerAlias{
			Id: alias.ID.String(), AliasName: alias.AliasName, SortOrder: int32(alias.SortOrder),
			CreatedAt: alias.CreatedAt.Format(time.RFC3339), UpdatedAt: alias.UpdatedAt.Format(time.RFC3339),
		})
	}
	return &v1.Partner{
		Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Code: value.Code, LegalName: value.LegalName,
		UnifiedSocialCreditCode: value.UnifiedSocialCreditCode, RegisteredAddress: value.RegisteredAddress, Enabled: value.Enabled,
		Roles: roles, Contacts: contacts, Aliases: aliases,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
		Profile: partnerProfileToAPI(value.Profile), Assignments: partnerAssignmentsToAPI(value.Assignments),
	}
}

func partnerProfileToAPI(value *biz.PartnerProfile) *v1.PartnerProfile {
	if value == nil {
		return nil
	}
	customerTypes := make([]v1.PartnerCustomerType, 0, len(value.CustomerTypes))
	for _, item := range value.CustomerTypes {
		customerTypes = append(customerTypes, partnerCustomerTypeToAPI(item))
	}
	businessTypes := make([]v1.PartnerBusinessType, 0, len(value.BusinessTypes))
	for _, item := range value.BusinessTypes {
		businessTypes = append(businessTypes, partnerBusinessTypeToAPI(item))
	}
	return &v1.PartnerProfile{
		NameEn: value.NameEN, AddressEn: value.AddressEN, CountryCode: value.CountryCode,
		ProvinceCode: value.ProvinceCode, CityCode: value.CityCode, DistrictCode: value.DistrictCode,
		AddressDetail: value.AddressDetail, Nature: value.Nature, DevelopmentMethod: value.DevelopmentMethod,
		CustomerTypes: customerTypes, BusinessTypes: businessTypes, Remark: value.Remark,
	}
}

func partnerAssignmentsToAPI(items []*biz.PartnerAssignment) []*v1.PartnerAssignment {
	result := make([]*v1.PartnerAssignment, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.PartnerAssignment{
			Id: item.ID.String(), Role: partnerAssignmentRoleToAPI(item.Role), UserId: item.UserID.String(),
			OrganizationId: item.OrganizationID.String(), CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
			SortOrder: int32(item.SortOrder),
		})
	}
	return result
}
