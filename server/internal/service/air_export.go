package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// AirExportService 只转换 DTO、校验目标公司权限并调用空运用例。
type AirExportService struct {
	v1.UnimplementedAirExportServiceServer
	usecase *biz.AirExportUsecase
}

func NewAirExportService(uc *biz.AirExportUsecase) *AirExportService {
	return &AirExportService{usecase: uc}
}
func airPrincipal(ctx context.Context, writable bool) (*biz.Principal, error) {
	p, e := biz.RequirePrincipal(ctx)
	if e != nil {
		return nil, e
	}
	op := access.OrderRead
	if writable {
		op = access.OrderUpdate
	}
	if !canOperateOrderInCurrentOrganization(p, biz.OrderBusinessAE, op, writable) {
		return nil, biz.ErrPermissionDenied
	}
	return p, nil
}
func airPagination(page, size int32) biz.SelectorListOptions {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	return biz.SelectorListOptions{Page: int(page), PageSize: int(size)}
}
func (s *AirExportService) ListAirMasterWaybills(ctx context.Context, req *v1.ListAirMasterWaybillsRequest) (*v1.ListAirMasterWaybillsResponse, error) {
	p, e := airPrincipal(ctx, false)
	if e != nil {
		return nil, e
	}
	o := airPagination(req.GetPage(), req.GetPageSize())
	o.Keyword = req.GetKeyword()
	rows, total, e := s.usecase.List(ctx, p.Organization.ID, o)
	if e != nil {
		return nil, e
	}
	out := &v1.ListAirMasterWaybillsResponse{Total: int32(total)}
	for _, m := range rows {
		out.Data = append(out.Data, airMasterToAPI(m))
	}
	return ok(ctx, out), nil
}
func (s *AirExportService) GetAirMasterWaybill(ctx context.Context, req *v1.GetAirMasterWaybillRequest) (*v1.GetAirMasterWaybillResponse, error) {
	p, e := airPrincipal(ctx, false)
	if e != nil {
		return nil, e
	}
	id, e := uuid.Parse(req.GetId())
	if e != nil {
		return nil, biz.ErrAirWaybillNotFound
	}
	result, e := s.usecase.Get(ctx, p.Organization.ID, id, airPagination(req.GetPage(), req.GetPageSize()))
	if e != nil {
		return nil, e
	}
	return ok(ctx, airDetailToAPI(result)), nil
}
func (s *AirExportService) PreviewAirMasterWaybillUpdate(ctx context.Context, req *v1.PreviewAirMasterWaybillUpdateRequest) (*v1.PreviewAirMasterWaybillUpdateResponse, error) {
	p, e := airPrincipal(ctx, true)
	if e != nil {
		return nil, e
	}
	id, e := uuid.Parse(req.GetId())
	if e != nil {
		return nil, biz.ErrAirWaybillNotFound
	}
	in, e := airMasterFromAPI(req.GetInput())
	if e != nil {
		return nil, e
	}
	v, e := s.usecase.Preview(ctx, p.Organization.ID, id, req.GetExpectedVersion(), in)
	if e != nil {
		return nil, e
	}
	out := &v1.PreviewAirMasterWaybillUpdateResponse{InputFingerprint: v.InputFingerprint, BlockedReasons: v.BlockedReasons, CanExecute: v.CanExecute}
	for _, m := range v.Members {
		out.Members = append(out.Members, airMemberToAPI(m))
	}
	return ok(ctx, out), nil
}
func (s *AirExportService) UpdateAirMasterWaybill(ctx context.Context, req *v1.UpdateAirMasterWaybillRequest) (*v1.UpdateAirMasterWaybillResponse, error) {
	p, e := airPrincipal(ctx, true)
	if e != nil {
		return nil, e
	}
	id, e := uuid.Parse(req.GetId())
	if e != nil {
		return nil, biz.ErrAirWaybillNotFound
	}
	in, e := airMasterFromAPI(req.GetInput())
	if e != nil {
		return nil, e
	}
	if e = s.usecase.Update(ctx, p.Organization.ID, p.UserID, id, req.GetExpectedVersion(), in, req.GetInputFingerprint(), req.GetReason()); e != nil {
		return nil, e
	}
	result, e := s.usecase.Get(ctx, p.Organization.ID, id, airPagination(1, 20))
	if e != nil {
		return nil, e
	}
	v := airDetailToAPI(result)
	return ok(ctx, &v1.UpdateAirMasterWaybillResponse{Data: v.Data, Members: v.Members, Total: v.Total, Totals: v.Totals, History: v.History, BlockedReasons: v.BlockedReasons, HistoryTotal: v.HistoryTotal}), nil
}
func (s *AirExportService) LinkAirMasterWaybill(ctx context.Context, req *v1.LinkAirMasterWaybillRequest) (*v1.LinkAirMasterWaybillResponse, error) {
	p, e := airPrincipal(ctx, true)
	if e != nil {
		return nil, e
	}
	if !canOperateOrderInCurrentOrganization(p, biz.OrderBusinessAE, access.OrderTransition, true) {
		return nil, biz.ErrPermissionDenied
	}
	id, e := uuid.Parse(req.GetId())
	if e != nil {
		return nil, biz.ErrAirWaybillNotFound
	}
	order, e := uuid.Parse(req.GetOrderId())
	if e != nil {
		return nil, biz.ErrOrderNotFound
	}
	if e = s.usecase.Link(ctx, p.Organization.ID, p.UserID, id, order, req.GetExpectedMasterVersion(), req.GetExpectedOrderVersion(), req.GetHouseNo(), req.GetReason()); e != nil {
		return nil, e
	}
	result, e := s.usecase.Get(ctx, p.Organization.ID, id, airPagination(1, 20))
	if e != nil {
		return nil, e
	}
	v := airDetailToAPI(result)
	return ok(ctx, &v1.LinkAirMasterWaybillResponse{Data: v.Data, Members: v.Members, Total: v.Total, Totals: v.Totals, History: v.History, BlockedReasons: v.BlockedReasons, HistoryTotal: v.HistoryTotal}), nil
}
func airMasterFromAPI(in *v1.AirMasterWaybillInput) (*biz.AirMasterWaybillInput, error) {
	if in == nil {
		return nil, biz.ErrAirWaybillInvalid
	}
	a, e := uuid.Parse(in.GetAirlineId())
	if e != nil {
		return nil, biz.ErrAirWaybillInvalid
	}
	o, e := uuid.Parse(in.GetOriginAirportId())
	if e != nil {
		return nil, biz.ErrAirWaybillInvalid
	}
	d, e := uuid.Parse(in.GetDestinationAirportId())
	if e != nil {
		return nil, biz.ErrAirWaybillInvalid
	}
	return &biz.AirMasterWaybillInput{MasterNo: in.GetMasterNo(), AirlineID: a, OriginAirportID: o, DestinationAirportID: d, FlightNo: in.GetFlightNo(), ETD: in.GetEtd(), ETA: in.GetEta(), ActualDepartureAt: in.GetActualDepartureAt(), ActualArrivalAt: in.GetActualArrivalAt(), DocumentCutoffAt: in.GetDocumentCutoffAt(), CustomsCutoffAt: in.GetCustomsCutoffAt(), Status: in.GetStatus(), DeclaredPackages: optionalInt32ToInt(in.DeclaredPackages), DeclaredGrossWeightKg: in.DeclaredGrossWeightKg, DeclaredVolumeCbm: in.DeclaredVolumeCbm, Notes: in.GetNotes()}, nil
}
func airMasterInputToAPI(in *biz.AirMasterWaybillInput) *v1.AirMasterWaybillInput {
	if in == nil {
		return nil
	}
	return &v1.AirMasterWaybillInput{MasterNo: in.MasterNo, AirlineId: in.AirlineID.String(), OriginAirportId: in.OriginAirportID.String(), DestinationAirportId: in.DestinationAirportID.String(), FlightNo: in.FlightNo, Etd: in.ETD, Eta: in.ETA, ActualDepartureAt: in.ActualDepartureAt, ActualArrivalAt: in.ActualArrivalAt, DocumentCutoffAt: in.DocumentCutoffAt, CustomsCutoffAt: in.CustomsCutoffAt, Status: in.Status, DeclaredPackages: intToInt32Ptr(in.DeclaredPackages), DeclaredGrossWeightKg: in.DeclaredGrossWeightKg, DeclaredVolumeCbm: in.DeclaredVolumeCbm, Notes: in.Notes}
}
func airMasterToAPI(m *biz.AirMasterWaybill) *v1.AirMasterWaybill {
	if m == nil {
		return nil
	}
	return &v1.AirMasterWaybill{Id: m.ID.String(), OrganizationId: m.OrganizationID.String(), Version: m.Version, Content: airMasterInputToAPI(&m.Content), AirlineName: m.AirlineName, OriginAirportName: m.OriginAirportName, DestinationAirportName: m.DestinationAirportName, MemberCount: int32(m.MemberCount), CreatedAt: m.CreatedAt.Format(time.RFC3339), UpdatedAt: m.UpdatedAt.Format(time.RFC3339)}
}
func airMemberToAPI(m *biz.AirMasterMember) *v1.AirMasterMember {
	return &v1.AirMasterMember{OrderId: m.OrderID.String(), OrderNo: m.OrderNo, CustomerName: m.CustomerName, HouseNo: m.HouseNo, Packages: intToInt32Ptr(m.Packages), PackageUnit: m.PackageUnit, GrossWeightKg: m.GrossWeightKg, VolumeCbm: m.VolumeCbm, FlowStatus: m.FlowStatus, TerminationStatus: m.TerminationStatus, ClosureStatus: m.ClosureStatus, Locked: m.Locked, OrderVersion: m.OrderVersion, HouseStatus: m.HouseStatus, ConfirmedChargeableWeightKg: m.ConfirmedChargeableWeightKg, ConfirmationStale: m.ConfirmationStale}
}
func airDetailToAPI(v *biz.AirMasterDetail) *v1.GetAirMasterWaybillResponse {
	out := &v1.GetAirMasterWaybillResponse{Data: airMasterToAPI(v.Master), Total: int32(v.Total), BlockedReasons: v.BlockedReasons, HistoryTotal: int32(v.HistoryTotal), Totals: &v1.AirMasterTotals{GrossWeightKg: v.Totals.GrossWeightKg, VolumeCbm: v.Totals.VolumeCbm, Complete: v.Totals.Complete, MemberCount: int32(v.Totals.MemberCount)}}
	for _, m := range v.Members {
		out.Members = append(out.Members, airMemberToAPI(m))
	}
	units := make([]string, 0, len(v.Totals.PackagesByUnit))
	for u := range v.Totals.PackagesByUnit {
		units = append(units, u)
	}
	sort.Strings(units)
	for _, u := range units {
		out.Totals.PackagesByUnit = append(out.Totals.PackagesByUnit, &v1.AirPackageTotal{Unit: u, Packages: int32(v.Totals.PackagesByUnit[u])})
	}
	for _, h := range v.History {
		out.History = append(out.History, &v1.AirMasterHistory{Id: h.ID.String(), Version: h.Version, Action: h.Action, Reason: h.Reason, ActorId: h.ActorID.String(), OccurredAt: h.OccurredAt.Format(time.RFC3339), Content: airMasterInputToAPI(&h.Content)})
	}
	return out
}
func airOrderInputFromAPI(in *v1.AirOrderInput) (*biz.AirOrderInput, error) {
	if in == nil {
		return nil, nil
	}
	out := &biz.AirOrderInput{ExpectedMasterVersion: in.GetExpectedMasterVersion(), ExpectedLinkVersion: in.GetExpectedLinkVersion(), ChangeReason: in.GetChangeReason(), HouseNo: in.GetHouseNo(), HouseContent: in.GetHouseContent(), HouseStatus: in.GetHouseStatus()}
	var e error
	if in.MasterWaybillId != nil {
		out.MasterWaybillID, e = parseOptionalUUIDPointer(in.MasterWaybillId)
		if e != nil {
			return nil, e
		}
	}
	if in.Master != nil {
		out.Master, e = airMasterFromAPI(in.Master)
		if e != nil {
			return nil, e
		}
	}
	return out, nil
}
func airOrderToAPI(in *biz.AirOrderInfo) *v1.AirOrderInfo {
	if in == nil {
		return nil
	}
	return &v1.AirOrderInfo{Master: airMasterToAPI(in.Master), HouseNo: in.HouseNo, HouseStatus: in.HouseStatus, HouseVersion: in.HouseVersion, HouseContent: in.HouseContent, HouseId: in.HouseID.String(), LinkId: in.LinkID.String(), LinkVersion: in.LinkVersion, Active: in.Active}
}

func (s *AirExportService) ListAirLinkCandidates(ctx context.Context, req *v1.ListAirLinkCandidatesRequest) (*v1.ListAirLinkCandidatesResponse, error) {
	p, e := airPrincipal(ctx, true)
	if e != nil {
		return nil, e
	}
	id, e := uuid.Parse(req.GetId())
	if e != nil {
		return nil, biz.ErrAirWaybillNotFound
	}
	o := airPagination(req.GetPage(), req.GetPageSize())
	o.Keyword = req.GetKeyword()
	rows, total, e := s.usecase.LinkCandidates(ctx, p.Organization.ID, id, o)
	if e != nil {
		return nil, e
	}
	out := &v1.ListAirLinkCandidatesResponse{Total: int32(total)}
	for _, row := range rows {
		out.Data = append(out.Data, airMemberToAPI(row))
	}
	return ok(ctx, out), nil
}

func (s *AirExportService) ListAirHouseWaybillVersions(ctx context.Context, req *v1.ListAirHouseWaybillVersionsRequest) (*v1.ListAirHouseWaybillVersionsResponse, error) {
	p, e := airPrincipal(ctx, false)
	if e != nil {
		return nil, e
	}
	id, e := uuid.Parse(req.GetOrderId())
	if e != nil {
		return nil, biz.ErrOrderNotFound
	}
	rows, total, e := s.usecase.HouseVersions(ctx, p.Organization.ID, id, airPagination(req.GetPage(), req.GetPageSize()))
	if e != nil {
		return nil, e
	}
	out := &v1.ListAirHouseWaybillVersionsResponse{Total: int32(total)}
	for _, v := range rows {
		out.Data = append(out.Data, &v1.AirHouseWaybillVersion{Id: v.ID.String(), Version: v.Version, HouseNo: v.HouseNo, Status: v.Status, Content: v.Content, ActorId: v.ActorID.String(), Reason: v.Reason, OccurredAt: v.OccurredAt.Format(time.RFC3339)})
	}
	return ok(ctx, out), nil
}
