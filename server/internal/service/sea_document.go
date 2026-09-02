package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type SeaDocumentService struct {
	v1.UnimplementedSeaDocumentServiceServer
	usecase *biz.SeaDocumentUsecase
}

func NewSeaDocumentService(usecase *biz.SeaDocumentUsecase) *SeaDocumentService {
	return &SeaDocumentService{usecase: usecase}
}

var _ v1.SeaDocumentServiceServer = (*SeaDocumentService)(nil)

func (s *SeaDocumentService) GetSeaOrderDocuments(ctx context.Context, req *v1.GetSeaOrderDocumentsRequest) (*v1.GetSeaOrderDocumentsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}

	docs, err := s.usecase.GetSeaOrderDocuments(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.GetSeaOrderDocumentsResponse{
		Data: seaOrderDocumentsToAPI(docs),
	}), nil
}

func (s *SeaDocumentService) MarkSeaOrderDirect(ctx context.Context, req *v1.MarkSeaOrderDirectRequest) (*v1.MarkSeaOrderDirectResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	docs, err := s.usecase.MarkSeaOrderDirect(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedLinkVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.MarkSeaOrderDirectResponse{
		Data: seaOrderDocumentsToAPI(docs),
	}), nil
}

func (s *SeaDocumentService) CancelSeaOrderDirect(ctx context.Context, req *v1.CancelSeaOrderDirectRequest) (*v1.CancelSeaOrderDirectResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	docs, err := s.usecase.CancelSeaOrderDirect(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedLinkVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.CancelSeaOrderDirectResponse{
		Data: seaOrderDocumentsToAPI(docs),
	}), nil
}

func (s *SeaDocumentService) AddSeaHouseBill(ctx context.Context, req *v1.AddSeaHouseBillRequest) (*v1.AddSeaHouseBillResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	input, err := seaHouseBillInputFromAPI(req.GetHouseBill())
	if err != nil {
		return nil, err
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	hb, err := s.usecase.AddSeaHouseBill(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedLinkVersion(), input, audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.AddSeaHouseBillResponse{
		Data: seaHouseBillToAPI(hb),
	}), nil
}

func (s *SeaDocumentService) UpdateSeaHouseBill(ctx context.Context, req *v1.UpdateSeaHouseBillRequest) (*v1.UpdateSeaHouseBillResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	hbID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	input, err := seaHouseBillInputFromAPI(req.GetHouseBill())
	if err != nil {
		return nil, err
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	hb, err := s.usecase.UpdateSeaHouseBill(ctx, principal.Organization.ID, principal.UserID, orderID, hbID, req.GetExpectedVersion(), req.GetExpectedLinkVersion(), input, audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.UpdateSeaHouseBillResponse{
		Data: seaHouseBillToAPI(hb),
	}), nil
}

func (s *SeaDocumentService) RemoveSeaHouseBill(ctx context.Context, req *v1.RemoveSeaHouseBillRequest) (*v1.RemoveSeaHouseBillResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	hbID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	if err := s.usecase.RemoveSeaHouseBill(ctx, principal.Organization.ID, principal.UserID, orderID, hbID, req.GetExpectedVersion(), req.GetExpectedLinkVersion(), req.GetReturnToUndetermined(), audit); err != nil {
		return nil, err
	}

	return ok(ctx, &v1.RemoveSeaHouseBillResponse{}), nil
}

func (s *SeaDocumentService) UpdateSeaMasterBillContent(ctx context.Context, req *v1.UpdateSeaMasterBillContentRequest) (*v1.UpdateSeaMasterBillContentResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	content := seaBillContentFromAPI(req.GetContent())

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	detail, err := s.usecase.UpdateSeaMasterBillContent(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedMblVersion(), content, audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.UpdateSeaMasterBillContentResponse{
		Data: seaMasterBillDetailToAPI(detail),
	}), nil
}

func seaDocumentStructureToAPI(s biz.SeaDocumentStructure) v1.SeaDocumentStructure {
	switch s {
	case biz.SeaDocumentStructureUndetermined:
		return v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_UNDETERMINED
	case biz.SeaDocumentStructureDirect:
		return v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_DIRECT
	case biz.SeaDocumentStructureHouse:
		return v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_HOUSE
	default:
		return v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_UNSPECIFIED
	}
}

func seaDocumentStructureFromAPI(s v1.SeaDocumentStructure) biz.SeaDocumentStructure {
	switch s {
	case v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_UNDETERMINED:
		return biz.SeaDocumentStructureUndetermined
	case v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_DIRECT:
		return biz.SeaDocumentStructureDirect
	case v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_HOUSE:
		return biz.SeaDocumentStructureHouse
	default:
		return ""
	}
}

func seaDocumentActionToAPI(a biz.SeaDocumentAction) v1.SeaDocumentAction {
	switch a {
	case biz.SeaDocumentActionMarkDirect:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_MARK_DIRECT
	case biz.SeaDocumentActionCancelDirect:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_CANCEL_DIRECT
	case biz.SeaDocumentActionAddHouseBill:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_ADD_HOUSE_BILL
	case biz.SeaDocumentActionUpdateHouseBill:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_UPDATE_HOUSE_BILL
	case biz.SeaDocumentActionRemoveHouseBill:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_REMOVE_HOUSE_BILL
	case biz.SeaDocumentActionUpdateMasterBillContent:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_UPDATE_MASTER_BILL_CONTENT
	default:
		return v1.SeaDocumentAction_SEA_DOCUMENT_ACTION_UNSPECIFIED
	}
}

func seaHouseBillIssuerSourceToAPI(s biz.SeaHouseBillIssuerSource) v1.SeaHouseBillIssuerSource {
	switch s {
	case biz.SeaHouseBillIssuerSourceSelfOrganization:
		return v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
	case biz.SeaHouseBillIssuerSourceCustomerPartner:
		return v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
	case biz.SeaHouseBillIssuerSourceOtherPartner:
		return v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
	default:
		return v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_UNSPECIFIED
	}
}

func seaHouseBillIssuerSourceFromAPI(s v1.SeaHouseBillIssuerSource) biz.SeaHouseBillIssuerSource {
	switch s {
	case v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION:
		return biz.SeaHouseBillIssuerSourceSelfOrganization
	case v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER:
		return biz.SeaHouseBillIssuerSourceCustomerPartner
	case v1.SeaHouseBillIssuerSource_SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER:
		return biz.SeaHouseBillIssuerSourceOtherPartner
	default:
		return ""
	}
}

func seaHouseBillStatusToAPI(s biz.SeaHouseBillStatus) v1.SeaHouseBillStatus {
	switch s {
	case biz.SeaHouseBillStatusDraft:
		return v1.SeaHouseBillStatus_SEA_HOUSE_BILL_STATUS_DRAFT
	case biz.SeaHouseBillStatusConfirmed:
		return v1.SeaHouseBillStatus_SEA_HOUSE_BILL_STATUS_CONFIRMED
	case biz.SeaHouseBillStatusReleased:
		return v1.SeaHouseBillStatus_SEA_HOUSE_BILL_STATUS_RELEASED
	default:
		return v1.SeaHouseBillStatus_SEA_HOUSE_BILL_STATUS_UNSPECIFIED
	}
}

func seaBillContentToAPI(c *biz.SeaBillContent) *v1.SeaBillContent {
	if c == nil {
		return nil
	}
	return &v1.SeaBillContent{
		ShipperText:           c.ShipperText,
		ConsigneeText:         c.ConsigneeText,
		NotifyPartyText:       c.NotifyPartyText,
		SecondNotifyPartyText: c.SecondNotifyPartyText,
		MarksText:             c.MarksText,
		GoodsDescriptionText:  c.GoodsDescriptionText,
		PackageCount:          c.PackageCount,
		PackageUnit:           c.PackageUnit,
		GrossWeightKg:         c.GrossWeightKg,
		VolumeCbm:             c.VolumeCbm,
		FreightTerms:          c.FreightTerms,
		TransportTerms:        c.TransportTerms,
		BillForm:              c.BillForm,
		ReleaseType:           c.ReleaseType,
		Clauses:               c.Clauses,
	}
}

func seaBillContentFromAPI(c *v1.SeaBillContent) *biz.SeaBillContent {
	if c == nil {
		return nil
	}
	return &biz.SeaBillContent{
		ShipperText:           c.ShipperText,
		ConsigneeText:         c.ConsigneeText,
		NotifyPartyText:       c.NotifyPartyText,
		SecondNotifyPartyText: c.SecondNotifyPartyText,
		MarksText:             c.MarksText,
		GoodsDescriptionText:  c.GoodsDescriptionText,
		PackageCount:          c.PackageCount,
		PackageUnit:           c.PackageUnit,
		GrossWeightKg:         c.GrossWeightKg,
		VolumeCbm:             c.VolumeCbm,
		FreightTerms:          c.FreightTerms,
		TransportTerms:        c.TransportTerms,
		BillForm:              c.BillForm,
		ReleaseType:           c.ReleaseType,
		Clauses:               c.Clauses,
	}
}

func seaHouseBillToAPI(hb *biz.SeaHouseBill) *v1.SeaHouseBill {
	if hb == nil {
		return nil
	}
	var orgID, partnerID *string
	if hb.IssuerOrganizationID != nil {
		s := hb.IssuerOrganizationID.String()
		orgID = &s
	}
	if hb.IssuerPartnerID != nil {
		s := hb.IssuerPartnerID.String()
		partnerID = &s
	}
	var orgName, partnerName *string
	if hb.IssuerOrganizationName != "" {
		orgName = &hb.IssuerOrganizationName
	}
	if hb.IssuerPartnerName != "" {
		partnerName = &hb.IssuerPartnerName
	}

	return &v1.SeaHouseBill{
		Id:                     hb.ID.String(),
		OrganizationId:         hb.OrganizationID.String(),
		OrderId:                hb.OrderID.String(),
		MasterBillId:           hb.MasterBillID.String(),
		HouseNo:                hb.HouseNo,
		IssuerSource:           seaHouseBillIssuerSourceToAPI(hb.IssuerSource),
		IssuerOrganizationId:   orgID,
		IssuerOrganizationName: orgName,
		IssuerPartnerId:        partnerID,
		IssuerPartnerName:      partnerName,
		Status:                 seaHouseBillStatusToAPI(hb.Status),
		Version:                hb.Version,
		Note:                   hb.Note,
		Content:                seaBillContentToAPI(hb.Content),
		CreatedAt:              hb.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              hb.UpdatedAt.Format(time.RFC3339),
	}
}

func seaHouseBillInputFromAPI(in *v1.SeaHouseBillInput) (*biz.SeaHouseBillInput, error) {
	if in == nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	var id *uuid.UUID
	if in.Id != nil && *in.Id != "" {
		parsed, err := uuid.Parse(*in.Id)
		if err != nil {
			return nil, biz.ErrSeaHouseBillInvalidArgument
		}
		id = &parsed
	}
	var partnerID *uuid.UUID
	if in.IssuerPartnerId != nil && *in.IssuerPartnerId != "" {
		parsed, err := uuid.Parse(*in.IssuerPartnerId)
		if err != nil {
			return nil, biz.ErrSeaHouseBillInvalidArgument
		}
		partnerID = &parsed
	}
	return &biz.SeaHouseBillInput{
		ID:              id,
		HouseNo:         in.GetHouseNo(),
		IssuerSource:    seaHouseBillIssuerSourceFromAPI(in.GetIssuerSource()),
		IssuerPartnerID: partnerID,
		Note:            in.Note,
		Content:         seaBillContentFromAPI(in.GetContent()),
		ExpectedVersion: in.ExpectedVersion,
	}, nil
}

func seaMasterBillDetailToAPI(m *biz.SeaMasterBillDetail) *v1.SeaMasterBillDetail {
	if m == nil {
		return nil
	}
	var partnerName *string
	if m.IssuerPartnerName != "" {
		partnerName = &m.IssuerPartnerName
	}
	return &v1.SeaMasterBillDetail{
		Id:                m.ID.String(),
		MasterNo:          m.MasterNo,
		IssuerPartnerId:   m.IssuerPartnerID.String(),
		IssuerPartnerName: partnerName,
		Status:            m.Status,
		Version:           m.Version,
		Content:           seaBillContentToAPI(m.Content),
		MemberCount:       int32(m.MemberCount),
	}
}

func seaOrderDocumentsToAPI(d *biz.SeaOrderDocuments) *v1.SeaOrderDocuments {
	if d == nil {
		return nil
	}
	hbs := make([]*v1.SeaHouseBill, 0, len(d.HouseBills))
	for _, hb := range d.HouseBills {
		hbs = append(hbs, seaHouseBillToAPI(hb))
	}
	actions := make([]v1.SeaDocumentAction, 0, len(d.AllowedActions))
	for _, a := range d.AllowedActions {
		actions = append(actions, seaDocumentActionToAPI(a))
	}
	return &v1.SeaOrderDocuments{
		OrderId:           d.OrderID.String(),
		DocumentStructure: seaDocumentStructureToAPI(d.DocumentStructure),
		LinkVersion:       d.LinkVersion,
		MasterBill:        seaMasterBillDetailToAPI(d.MasterBill),
		HouseBills:        hbs,
		AllowedActions:    actions,
	}
}
