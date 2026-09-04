package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func parseRequiredUUID(value string) (uuid.UUID, error) {
	result, err := uuid.Parse(value)
	if err != nil || result == uuid.Nil {
		return uuid.Nil, biz.ErrSeaDocumentInvalidArgument
	}
	return result, nil
}

func listDocumentPage(page, pageSize int32) (int, int, error) {
	return listPageValues(page, pageSize, biz.ErrSeaDocumentInvalidArgument)
}

func (s *SeaDocumentService) ListSeaMasterBillVersions(ctx context.Context, req *v1.ListSeaMasterBillVersionsRequest) (*v1.ListSeaMasterBillVersionsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := parseRequiredUUID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listDocumentPage(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	rows, total, err := s.changeUsecase.ListMasterBillVersions(ctx, principal.Organization.ID, orderID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ListSeaMasterBillVersionsResponse{Data: seaDocumentVersionsToAPI(rows), Total: int32(total)}), nil
}

func (s *SeaDocumentService) ListSeaHouseBillVersions(ctx context.Context, req *v1.ListSeaHouseBillVersionsRequest) (*v1.ListSeaHouseBillVersionsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := parseRequiredUUID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	houseID, err := parseRequiredUUID(req.GetHouseBillId())
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listDocumentPage(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	rows, total, err := s.changeUsecase.ListHouseBillVersions(ctx, principal.Organization.ID, orderID, houseID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ListSeaHouseBillVersionsResponse{Data: seaDocumentVersionsToAPI(rows), Total: int32(total)}), nil
}

func (s *SeaDocumentService) GetSeaDocumentVersion(ctx context.Context, req *v1.GetSeaDocumentVersionRequest) (*v1.GetSeaDocumentVersionResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := parseRequiredUUID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	versionID, err := parseRequiredUUID(req.GetVersionId())
	if err != nil {
		return nil, err
	}
	row, err := s.changeUsecase.GetDocumentVersion(ctx, principal.Organization.ID, orderID, versionID, seaDocumentTypeFromAPI(req.GetDocumentType()))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetSeaDocumentVersionResponse{Data: seaDocumentVersionToAPI(row)}), nil
}

func (s *SeaDocumentService) ListSeaDocumentEvents(ctx context.Context, req *v1.ListSeaDocumentEventsRequest) (*v1.ListSeaDocumentEventsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := parseRequiredUUID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listDocumentPage(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	rows, total, err := s.changeUsecase.ListDocumentEvents(ctx, principal.Organization.ID, orderID, page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.SeaDocumentEvent, 0, len(rows))
	for _, row := range rows {
		data = append(data, seaDocumentEventToAPI(row))
	}
	return ok(ctx, &v1.ListSeaDocumentEventsResponse{Data: data, Total: int32(total)}), nil
}

func amendmentCommandFromAPI(orderID string, documentType v1.SeaDocumentType, documentID string, expectedOrderVersion, expectedDocumentVersion uint64, currentVersionID, reason, key string, input *v1.SeaDocumentAmendmentInput) (*biz.SeaDocumentAmendmentCommand, error) {
	orderUUID, err := parseRequiredUUID(orderID)
	if err != nil {
		return nil, err
	}
	docUUID, err := parseRequiredUUID(documentID)
	if err != nil {
		return nil, err
	}
	versionUUID, err := parseRequiredUUID(currentVersionID)
	if err != nil {
		return nil, err
	}
	result := &biz.SeaDocumentAmendmentCommand{OrderID: orderUUID, DocumentType: seaDocumentTypeFromAPI(documentType), DocumentID: docUUID, ExpectedOrderVersion: expectedOrderVersion, ExpectedDocumentVersion: expectedDocumentVersion, ExpectedCurrentVersionID: versionUUID, Reason: reason, IdempotencyKey: key, Input: &biz.SeaDocumentAmendmentInput{}}
	if input != nil {
		result.Input.MasterBillContent = seaBillContentFromAPI(input.GetMasterBillContent())
		if input.HouseBill != nil {
			result.Input.HouseBill, err = seaHouseBillInputFromAPI(input.HouseBill)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func voidCommandFromAPI(orderID string, documentType v1.SeaDocumentType, documentID string, expectedOrderVersion, expectedDocumentVersion uint64, currentVersionID, reason, key string) (*biz.SeaDocumentVoidCommand, error) {
	orderUUID, err := parseRequiredUUID(orderID)
	if err != nil {
		return nil, err
	}
	docUUID, err := parseRequiredUUID(documentID)
	if err != nil {
		return nil, err
	}
	versionUUID, err := parseRequiredUUID(currentVersionID)
	if err != nil {
		return nil, err
	}
	return &biz.SeaDocumentVoidCommand{OrderID: orderUUID, DocumentType: seaDocumentTypeFromAPI(documentType), DocumentID: docUUID, ExpectedOrderVersion: expectedOrderVersion, ExpectedDocumentVersion: expectedDocumentVersion, ExpectedCurrentVersionID: versionUUID, Reason: reason, IdempotencyKey: key}, nil
}

func switchCommandFromAPI(orderID, oldHouseID string, expectedOrderVersion, expectedHouseVersion uint64, currentVersionID, reason string, surrenderInfo *string, key string, newHouse *v1.SeaHouseBillInput) (*biz.SeaHouseBillSwitchCommand, error) {
	orderUUID, err := parseRequiredUUID(orderID)
	if err != nil {
		return nil, err
	}
	oldUUID, err := parseRequiredUUID(oldHouseID)
	if err != nil {
		return nil, err
	}
	versionUUID, err := parseRequiredUUID(currentVersionID)
	if err != nil {
		return nil, err
	}
	hb, err := seaHouseBillInputFromAPI(newHouse)
	if err != nil {
		return nil, err
	}
	return &biz.SeaHouseBillSwitchCommand{OrderID: orderUUID, OldHouseBillID: oldUUID, ExpectedOrderVersion: expectedOrderVersion, ExpectedHouseBillVersion: expectedHouseVersion, ExpectedCurrentVersionID: versionUUID, Reason: reason, SurrenderInfo: surrenderInfo, IdempotencyKey: key, NewHouseBill: hb}, nil
}

func changeAudit(principal *biz.Principal) *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &principal.Organization.ID, UserID: &principal.UserID, Result: "success"}
}

func (s *SeaDocumentService) PreviewSeaDocumentAmendment(ctx context.Context, req *v1.PreviewSeaDocumentAmendmentRequest) (*v1.PreviewSeaDocumentAmendmentResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := amendmentCommandFromAPI(req.GetOrderId(), req.GetDocumentType(), req.GetDocumentId(), req.GetExpectedOrderVersion(), req.GetExpectedDocumentVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), "", req.GetInput())
	if err != nil {
		return nil, err
	}
	preview, err := s.changeUsecase.PreviewAmendment(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewSeaDocumentAmendmentResponse{Data: seaDocumentChangePreviewToAPI(preview)}), nil
}

func (s *SeaDocumentService) ExecuteSeaDocumentAmendment(ctx context.Context, req *v1.ExecuteSeaDocumentAmendmentRequest) (*v1.ExecuteSeaDocumentAmendmentResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := amendmentCommandFromAPI(req.GetOrderId(), req.GetDocumentType(), req.GetDocumentId(), req.GetExpectedOrderVersion(), req.GetExpectedDocumentVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), req.GetIdempotencyKey(), req.GetInput())
	if err != nil {
		return nil, err
	}
	row, err := s.changeUsecase.ExecuteAmendment(ctx, principal.Organization.ID, principal.UserID, input, changeAudit(principal))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ExecuteSeaDocumentAmendmentResponse{Data: seaDocumentVersionToAPI(row)}), nil
}

func (s *SeaDocumentService) PreviewSeaDocumentVoid(ctx context.Context, req *v1.PreviewSeaDocumentVoidRequest) (*v1.PreviewSeaDocumentVoidResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := voidCommandFromAPI(req.GetOrderId(), req.GetDocumentType(), req.GetDocumentId(), req.GetExpectedOrderVersion(), req.GetExpectedDocumentVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), "")
	if err != nil {
		return nil, err
	}
	preview, err := s.changeUsecase.PreviewVoid(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewSeaDocumentVoidResponse{Data: seaDocumentVoidPreviewToAPI(preview)}), nil
}

func (s *SeaDocumentService) ExecuteSeaDocumentVoid(ctx context.Context, req *v1.ExecuteSeaDocumentVoidRequest) (*v1.ExecuteSeaDocumentVoidResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := voidCommandFromAPI(req.GetOrderId(), req.GetDocumentType(), req.GetDocumentId(), req.GetExpectedOrderVersion(), req.GetExpectedDocumentVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), req.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	event, err := s.changeUsecase.ExecuteVoid(ctx, principal.Organization.ID, principal.UserID, input, changeAudit(principal))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ExecuteSeaDocumentVoidResponse{Data: seaDocumentEventToAPI(event)}), nil
}

func (s *SeaDocumentService) PreviewSeaHouseBillSwitch(ctx context.Context, req *v1.PreviewSeaHouseBillSwitchRequest) (*v1.PreviewSeaHouseBillSwitchResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := switchCommandFromAPI(req.GetOrderId(), req.GetOldHouseBillId(), req.GetExpectedOrderVersion(), req.GetExpectedHouseBillVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), req.SurrenderInfo, "", req.GetNewHouseBill())
	if err != nil {
		return nil, err
	}
	preview, err := s.changeUsecase.PreviewSwitch(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewSeaHouseBillSwitchResponse{Data: seaHouseBillSwitchPreviewToAPI(preview)}), nil
}

func (s *SeaDocumentService) ExecuteSeaHouseBillSwitch(ctx context.Context, req *v1.ExecuteSeaHouseBillSwitchRequest) (*v1.ExecuteSeaHouseBillSwitchResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := switchCommandFromAPI(req.GetOrderId(), req.GetOldHouseBillId(), req.GetExpectedOrderVersion(), req.GetExpectedHouseBillVersion(), req.GetExpectedCurrentVersionId(), req.GetReason(), req.SurrenderInfo, req.GetIdempotencyKey(), req.GetNewHouseBill())
	if err != nil {
		return nil, err
	}
	result, err := s.changeUsecase.ExecuteSwitch(ctx, principal.Organization.ID, principal.UserID, input, changeAudit(principal))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ExecuteSeaHouseBillSwitchResponse{Data: seaDocumentEventToAPI(result.Event), NewHouseBill: seaHouseBillToAPI(result.NewHouseBill)}), nil
}

func seaDocumentTypeFromAPI(v v1.SeaDocumentType) biz.SeaDocumentType {
	switch v {
	case v1.SeaDocumentType_SEA_DOCUMENT_TYPE_MASTER_BILL:
		return biz.SeaDocumentTypeMasterBill
	case v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL:
		return biz.SeaDocumentTypeHouseBill
	default:
		return ""
	}
}
func seaDocumentTypeToAPI(v biz.SeaDocumentType) v1.SeaDocumentType {
	if v == biz.SeaDocumentTypeMasterBill {
		return v1.SeaDocumentType_SEA_DOCUMENT_TYPE_MASTER_BILL
	}
	if v == biz.SeaDocumentTypeHouseBill {
		return v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL
	}
	return v1.SeaDocumentType_SEA_DOCUMENT_TYPE_UNSPECIFIED
}
func seaVersionSourceToAPI(v string) v1.SeaDocumentVersionSource {
	switch v {
	case biz.VersionSourceOrderLock:
		return v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_ORDER_LOCK
	case biz.VersionSourceAmendment:
		return v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_AMENDMENT
	case biz.VersionSourceSwitch:
		return v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_SWITCH
	case biz.VersionSourceVoid:
		return v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_VOID
	default:
		return v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_UNSPECIFIED
	}
}
func seaDocumentVersionToAPI(v *biz.SeaDocumentVersion) *v1.SeaDocumentVersion {
	if v == nil {
		return nil
	}
	result := &v1.SeaDocumentVersion{Id: v.ID.String(), DocumentType: seaDocumentTypeToAPI(v.DocumentType), DocumentId: v.DocumentID.String(), OrderId: v.OrderID.String(), MasterBillId: v.MasterBillID.String(), VersionNo: v.VersionNo, SourceEntityVersion: v.SourceEntityVersion, DocumentNo: v.DocumentNo, NormalizedDocumentNo: v.NormalizedDocumentNo, Status: v.Status, Source: seaVersionSourceToAPI(v.Source), Reason: v.Reason, IssuerSource: seaHouseBillIssuerSourceToAPI(v.IssuerSource), VesselName: v.VesselName, VoyageNo: v.VoyageNo, Note: v.Note, Content: seaBillContentToAPI(v.Content), CreatedAt: v.CreatedAt.Format(time.RFC3339)}
	if v.IssuerPartnerID != nil {
		s := v.IssuerPartnerID.String()
		result.IssuerPartnerId = &s
	}
	if v.IssuerOrganizationID != nil {
		s := v.IssuerOrganizationID.String()
		result.IssuerOrganizationId = &s
	}
	if v.TransportExecutionID != nil {
		s := v.TransportExecutionID.String()
		result.TransportExecutionId = &s
	}
	if v.ETD != nil {
		s := v.ETD.Format(time.RFC3339)
		result.Etd = &s
	}
	if v.ETA != nil {
		s := v.ETA.Format(time.RFC3339)
		result.Eta = &s
	}
	if v.CreatedBy != nil {
		s := v.CreatedBy.String()
		result.CreatedBy = &s
	}
	return result
}
func seaDocumentVersionsToAPI(rows []*biz.SeaDocumentVersion) []*v1.SeaDocumentVersion {
	result := make([]*v1.SeaDocumentVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, seaDocumentVersionToAPI(row))
	}
	return result
}
func differencesToAPI(rows []*biz.SeaDocumentFieldDifference) []*v1.SeaDocumentFieldDifference {
	result := make([]*v1.SeaDocumentFieldDifference, 0, len(rows))
	for _, r := range rows {
		result = append(result, &v1.SeaDocumentFieldDifference{Field: r.Field, Label: r.Label, BeforeValue: r.BeforeValue, AfterValue: r.AfterValue})
	}
	return result
}
func impactsToAPI(rows []*biz.SeaDocumentDownstreamImpact) []*v1.SeaDocumentDownstreamImpact {
	result := make([]*v1.SeaDocumentDownstreamImpact, 0, len(rows))
	for _, r := range rows {
		result = append(result, &v1.SeaDocumentDownstreamImpact{FactType: r.FactType, ReferenceId: r.ReferenceID, ReferenceNo: r.ReferenceNo, Message: r.Message, BlocksExecution: r.BlocksExecution})
	}
	return result
}
func seaDocumentChangePreviewToAPI(v *biz.SeaDocumentChangePreview) *v1.SeaDocumentAmendmentPreview {
	if v == nil {
		return nil
	}
	return &v1.SeaDocumentAmendmentPreview{BaseVersion: seaDocumentVersionToAPI(v.BaseVersion), Differences: differencesToAPI(v.Differences), Impacts: impactsToAPI(v.Impacts), Executable: v.Executable}
}
func seaDocumentVoidPreviewToAPI(v *biz.SeaDocumentChangePreview) *v1.SeaDocumentVoidPreview {
	if v == nil {
		return nil
	}
	return &v1.SeaDocumentVoidPreview{BaseVersion: seaDocumentVersionToAPI(v.BaseVersion), Differences: differencesToAPI(v.Differences), Impacts: impactsToAPI(v.Impacts), Executable: v.Executable}
}
func seaHouseBillSwitchPreviewToAPI(v *biz.SeaDocumentChangePreview) *v1.SeaHouseBillSwitchPreview {
	if v == nil {
		return nil
	}
	return &v1.SeaHouseBillSwitchPreview{BaseVersion: seaDocumentVersionToAPI(v.BaseVersion), Differences: differencesToAPI(v.Differences), Impacts: impactsToAPI(v.Impacts), Executable: v.Executable}
}
func seaDocumentEventToAPI(v *biz.SeaDocumentEvent) *v1.SeaDocumentEvent {
	if v == nil {
		return nil
	}
	eventType := v1.SeaDocumentEventType_SEA_DOCUMENT_EVENT_TYPE_UNSPECIFIED
	switch v.EventType {
	case biz.SeaDocumentEventTypeAmendment:
		eventType = v1.SeaDocumentEventType_SEA_DOCUMENT_EVENT_TYPE_AMENDMENT
	case biz.SeaDocumentEventTypeVoid:
		eventType = v1.SeaDocumentEventType_SEA_DOCUMENT_EVENT_TYPE_VOID
	case biz.SeaDocumentEventTypeSwitch:
		eventType = v1.SeaDocumentEventType_SEA_DOCUMENT_EVENT_TYPE_SWITCH
	}
	result := &v1.SeaDocumentEvent{Id: v.ID.String(), EventType: eventType, DocumentType: seaDocumentTypeToAPI(v.DocumentType), DocumentNo: v.DocumentNo, OldHouseNo: v.OldHouseNo, NewHouseNo: v.NewHouseNo, Reason: v.Reason, ImpactSummary: v.ImpactSummary, SurrenderInfo: v.SurrenderInfo, CreatedAt: v.CreatedAt.Format(time.RFC3339)}
	setUUID := func(target **string, id *uuid.UUID) {
		if id != nil {
			s := id.String()
			*target = &s
		}
	}
	setUUID(&result.DocumentId, v.DocumentID)
	setUUID(&result.PreviousVersionId, v.PreviousVersionID)
	setUUID(&result.ResultVersionId, v.ResultVersionID)
	setUUID(&result.OldHouseBillId, v.OldHouseBillID)
	setUUID(&result.NewHouseBillId, v.NewHouseBillID)
	setUUID(&result.ChainId, v.ChainID)
	setUUID(&result.CreatedBy, v.CreatedBy)
	if v.Sequence != nil {
		x := int32(*v.Sequence)
		result.Sequence = &x
	}
	return result
}
