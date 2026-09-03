package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type mockSeaOrderChangeRepoForService struct {
	actions     *biz.SeaOrderChangeActions
	splitCtx    *biz.SeaOrderSplitContext
	previewRes  *biz.SeaOrderSplitPreview
	splitEvt    *biz.SeaOrderSplitEvent
	reasPreview *biz.SeaOrderReassignmentPreview
	reasEvt     *biz.SeaOrderReassignmentEvent
	events      []*biz.SeaOrderChangeEventSummary
	totalEvents int32
	eventDetail *biz.SeaOrderChangeEventDetail
}

func (m *mockSeaOrderChangeRepoForService) GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderChangeActions, error) {
	return m.actions, nil
}
func (m *mockSeaOrderChangeRepoForService) GetSplitContext(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderSplitContext, error) {
	return m.splitCtx, nil
}
func (m *mockSeaOrderChangeRepoForService) PreviewSplit(ctx context.Context, organizationID uuid.UUID, input *biz.SeaOrderSplitInput) (*biz.SeaOrderSplitPreview, error) {
	return m.previewRes, nil
}
func (m *mockSeaOrderChangeRepoForService) ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderSplitInput, audit *biz.AuditEvent) (*biz.SeaOrderSplitEvent, error) {
	return m.splitEvt, nil
}
func (m *mockSeaOrderChangeRepoForService) PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *biz.SeaOrderReassignmentInput) (*biz.SeaOrderReassignmentPreview, error) {
	return m.reasPreview, nil
}
func (m *mockSeaOrderChangeRepoForService) ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderReassignmentInput, audit *biz.AuditEvent) (*biz.SeaOrderReassignmentEvent, error) {
	return m.reasEvt, nil
}
func (m *mockSeaOrderChangeRepoForService) ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*biz.SeaOrderChangeEventSummary, int32, error) {
	return m.events, m.totalEvents, nil
}
func (m *mockSeaOrderChangeRepoForService) GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*biz.SeaOrderChangeEventDetail, error) {
	return m.eventDetail, nil
}
func (m *mockSeaOrderChangeRepoForService) GetSplitEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.SeaOrderSplitEvent, error) {
	return nil, nil
}
func (m *mockSeaOrderChangeRepoForService) GetSplitEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*biz.SeaOrderSplitEvent, error) {
	return m.splitEvt, nil
}
func (m *mockSeaOrderChangeRepoForService) GetReassignmentEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.SeaOrderReassignmentEvent, error) {
	return nil, nil
}
func (m *mockSeaOrderChangeRepoForService) GetReassignmentEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*biz.SeaOrderReassignmentEvent, error) {
	return m.reasEvt, nil
}

type mockTransactorForService struct{}

func (m *mockTransactorForService) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func strPtr(s string) *string {
	return &s
}

func TestSeaOrderChangeService_MappingsAndEndpoints(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	hblID := uuid.New()
	feeID := uuid.New()
	eventID := uuid.New()
	transportExecutionID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: actorID, Organization: biz.Organization{ID: orgID}})

	mockRepo := &mockSeaOrderChangeRepoForService{
		actions: &biz.SeaOrderChangeActions{
			CanSplit:               true,
			CanReassign:            false,
			ReassignBlockedReasons: []string{"已提交装船"},
		},
		splitCtx: &biz.SeaOrderSplitContext{
			OrderID:      orderID,
			OrderNo:      "SE20260903001",
			BusinessType: "SE",
			ShipmentType: "FCL",
			CurrentMasterBill: &biz.SeaMasterBillSummary{
				MasterBillID:              uuid.New(),
				MasterNo:                  "MBL001",
				TransportExecutionID:      transportExecutionID,
				TransportExecutionVersion: 7,
			},
			HouseBills: []*biz.SeaOrderSplitHouseBillItem{
				{ID: hblID, HouseNo: "HBL001", Status: "DRAFT", Version: 1},
			},
			DraftFees: []*biz.SeaOrderSplitDraftFeeItem{
				{
					ID:          feeID,
					FeeName:     "海运费",
					Direction:   "RECEIVABLE",
					Currency:    "USD",
					TotalAmount: decimal.RequireFromString("1500.00"),
				},
			},
		},
		previewRes: &biz.SeaOrderSplitPreview{
			IsValid:            true,
			ConservationPassed: true,
			Baseline: biz.SeaOrderSplitQuantitySummary{
				PackageCount:  100,
				GrossWeightKg: decimal.RequireFromString("2000.000"),
				VolumeCbm:     decimal.RequireFromString("15.000000"),
			},
			Allocated: biz.SeaOrderSplitQuantitySummary{
				PackageCount:  100,
				GrossWeightKg: decimal.RequireFromString("2000.000"),
				VolumeCbm:     decimal.RequireFromString("15.000000"),
			},
			Remaining: biz.SeaOrderSplitQuantitySummary{
				PackageCount:  0,
				GrossWeightKg: decimal.Zero,
				VolumeCbm:     decimal.Zero,
			},
			Results: []*biz.SeaOrderSplitPreviewResultItem{
				{
					ClientResultKey: "res-origin",
					ResultRole:      "ORIGINAL",
					PackageCount:    60,
					GrossWeightKg:   decimal.RequireFromString("1200.000"),
					VolumeCbm:       decimal.RequireFromString("9.000000"),
				},
			},
		},
		splitEvt: &biz.SeaOrderSplitEvent{
			ID:            eventID,
			SourceOrderID: orderID,
			SourceOrderNo: "SE20260903001",
			Results: []*biz.SeaOrderSplitResult{
				{
					OrderID:         orderID,
					OrderNo:         "SE20260903001",
					ResultRole:      "ORIGINAL",
					ClientResultKey: "res-origin",
				},
				{
					OrderID:         uuid.New(),
					OrderNo:         "SE20260903002",
					ResultRole:      "CREATED",
					ClientResultKey: "res-new-1",
				},
			},
		},
		reasPreview: &biz.SeaOrderReassignmentPreview{
			IsValid:           true,
			TargetMemberCount: 2,
			Differences: []*biz.VoyageDifference{
				{FieldName: "vessel_name", Label: "船名", CurrentValue: "SHIP A", TargetValue: "SHIP B", IsDifferent: true},
			},
		},
		reasEvt: &biz.SeaOrderReassignmentEvent{
			ID:      eventID,
			OrderID: orderID,
			OrderNo: "SE20260903001",
		},
		events: []*biz.SeaOrderChangeEventSummary{
			{
				ID:        eventID,
				EventType: "SPLIT",
				CreatedAt: time.Now(),
			},
		},
		totalEvents: 1,
		eventDetail: &biz.SeaOrderChangeEventDetail{
			ID:        eventID,
			EventType: "SPLIT",
		},
	}

	uc := biz.NewSeaOrderChangeUsecase(mockRepo, &mockTransactorForService{})
	svc := NewSeaOrderChangeService(uc)

	// 1. GetSeaOrderChangeActions
	actionsResp, err := svc.GetSeaOrderChangeActions(ctx, &v1.GetSeaOrderChangeActionsRequest{OrderId: orderID.String()})
	if err != nil {
		t.Fatalf("GetSeaOrderChangeActions error: %v", err)
	}
	if actionsResp.Data == nil || !actionsResp.Data.CanSplit {
		t.Errorf("actions response mismatch: %+v", actionsResp)
	}

	// 2. GetSeaOrderSplitContext
	splitCtxResp, err := svc.GetSeaOrderSplitContext(ctx, &v1.GetSeaOrderSplitContextRequest{OrderId: orderID.String()})
	if err != nil {
		t.Fatalf("GetSeaOrderSplitContext error: %v", err)
	}
	if splitCtxResp.Data == nil || splitCtxResp.Data.OrderNo != "SE20260903001" {
		t.Errorf("split context mismatch: %+v", splitCtxResp)
	}
	if splitCtxResp.Data.CurrentMasterBill == nil ||
		splitCtxResp.Data.CurrentMasterBill.TransportExecutionId != transportExecutionID.String() ||
		splitCtxResp.Data.CurrentMasterBill.TransportExecutionVersion != 7 {
		t.Errorf("split context missing transport execution version: %+v", splitCtxResp.Data.CurrentMasterBill)
	}

	// 3. PreviewSeaOrderSplit
	previewResp, err := svc.PreviewSeaOrderSplit(ctx, &v1.PreviewSeaOrderSplitRequest{
		OrderId: orderID.String(),
		Targets: []*v1.SeaOrderSplitTargetInput{
			{ClientTargetKey: "res-origin", TargetType: "CURRENT"},
		},
		Results: []*v1.SeaOrderSplitResultInput{
			{ClientResultKey: "res-origin", ResultRole: "ORIGINAL", ClientTargetKey: "res-origin"},
		},
	})
	if err != nil {
		t.Fatalf("PreviewSeaOrderSplit error: %v", err)
	}
	if previewResp.Data == nil || !previewResp.Data.ConservationPassed || previewResp.Data.Baseline == nil {
		t.Errorf("preview response mismatch: %+v", previewResp)
	}

	// 4. ExecuteSeaOrderSplit - 版本 0 拦截
	_, err = svc.ExecuteSeaOrderSplit(ctx, &v1.ExecuteSeaOrderSplitRequest{
		OrderId:            orderID.String(),
		IdempotencyKey:     "idemp-001",
		RequestFingerprint: "fp-001",
		Targets: []*v1.SeaOrderSplitTargetInput{
			{ClientTargetKey: "res-origin", TargetType: "CURRENT"},
		},
		Results: []*v1.SeaOrderSplitResultInput{
			{ClientResultKey: "res-origin", ResultRole: "ORIGINAL", ClientTargetKey: "res-origin"},
			{ClientResultKey: "res-new-1", ResultRole: "CREATED", ClientTargetKey: "res-origin"},
		},
		ExpectedVersions: &v1.SeaOrderSplitExpectedVersions{
			OrderVersion: 0,
		},
	})
	if err == nil {
		t.Fatal("expected error on expected order version 0")
	}

	// 4.1 ExecuteSeaOrderSplit - 正常执行
	execResp, err := svc.ExecuteSeaOrderSplit(ctx, &v1.ExecuteSeaOrderSplitRequest{
		OrderId:            orderID.String(),
		IdempotencyKey:     "idemp-001",
		RequestFingerprint: "fp-001",
		Targets: []*v1.SeaOrderSplitTargetInput{
			{ClientTargetKey: "res-origin", TargetType: "CURRENT"},
		},
		Results: []*v1.SeaOrderSplitResultInput{
			{ClientResultKey: "res-origin", ResultRole: "ORIGINAL", ClientTargetKey: "res-origin"},
			{ClientResultKey: "res-new-1", ResultRole: "CREATED", ClientTargetKey: "res-origin"},
		},
		ExpectedVersions: &v1.SeaOrderSplitExpectedVersions{
			OrderVersion:      1,
			LinkVersion:       1,
			AllocationVersion: 1,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteSeaOrderSplit error: %v", err)
	}
	if execResp.Data == nil || len(execResp.Data.CreatedOrders) != 1 {
		t.Errorf("execute response mismatch: %+v", execResp)
	}

	// 5. PreviewSeaOrderReassignment
	reasPreviewResp, err := svc.PreviewSeaOrderReassignment(ctx, &v1.PreviewSeaOrderReassignmentRequest{
		OrderId: orderID.String(),
		Target: &v1.SeaOrderReassignmentTargetInput{
			TargetType: "NEW",
			MasterNo:   strPtr("MBL888"),
		},
	})
	if err != nil {
		t.Fatalf("PreviewSeaOrderReassignment error: %v", err)
	}
	if reasPreviewResp.Data == nil || len(reasPreviewResp.Data.Differences) != 1 {
		t.Errorf("reassignment preview mismatch: %+v", reasPreviewResp)
	}

	// 6. ExecuteSeaOrderReassignment - 版本 0 拦截
	_, err = svc.ExecuteSeaOrderReassignment(ctx, &v1.ExecuteSeaOrderReassignmentRequest{
		OrderId:              orderID.String(),
		IdempotencyKey:       "reas-001",
		RequestFingerprint:   "fp-reas-001",
		Reason:               "船公司跳港",
		ResponsibilityType:   "CARRIER",
		ExpectedOrderVersion: 0,
		ExpectedLinkVersion:  1,
		Target: &v1.SeaOrderReassignmentTargetInput{
			TargetType: "NEW",
			MasterNo:   strPtr("MBL888"),
		},
	})
	if err == nil {
		t.Fatal("expected error on expected order version 0")
	}

	// 6.1 ExecuteSeaOrderReassignment - 正常执行
	reasExecResp, err := svc.ExecuteSeaOrderReassignment(ctx, &v1.ExecuteSeaOrderReassignmentRequest{
		OrderId:              orderID.String(),
		IdempotencyKey:       "reas-001",
		RequestFingerprint:   "fp-reas-001",
		Reason:               "船公司跳港",
		ResponsibilityType:   "CARRIER",
		ExpectedOrderVersion: 1,
		ExpectedLinkVersion:  1,
		Target: &v1.SeaOrderReassignmentTargetInput{
			TargetType: "NEW",
			MasterNo:   strPtr("MBL888"),
		},
	})
	if err != nil {
		t.Fatalf("ExecuteSeaOrderReassignment error: %v", err)
	}
	if reasExecResp.Data == nil || reasExecResp.Data.OrderNo != "SE20260903001" {
		t.Errorf("reassignment execute mismatch: %+v", reasExecResp)
	}

	// 7. ListSeaOrderChangeEvents
	eventsResp, err := svc.ListSeaOrderChangeEvents(ctx, &v1.ListSeaOrderChangeEventsRequest{
		OrderId:  orderID.String(),
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListSeaOrderChangeEvents error: %v", err)
	}
	if len(eventsResp.Data) != 1 || eventsResp.Total != 1 {
		t.Errorf("list events mismatch: %+v", eventsResp)
	}

	// 8. GetSeaOrderChangeEvent
	eventDetailResp, err := svc.GetSeaOrderChangeEvent(ctx, &v1.GetSeaOrderChangeEventRequest{
		OrderId:   orderID.String(),
		EventId:   eventID.String(),
		EventType: "SPLIT",
	})
	if err != nil {
		t.Fatalf("GetSeaOrderChangeEvent error: %v", err)
	}
	if eventDetailResp.Data == nil || eventDetailResp.Data.Id != eventID.String() {
		t.Errorf("event detail mismatch: %+v", eventDetailResp)
	}
}

func TestSeaOrderChangeService_ExecuteSplitRequiresReassignPermission(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       actorID,
		Organization: biz.Organization{ID: orgID},
	})
	svc := NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(&mockSeaOrderChangeRepoForService{}, &mockTransactorForService{}))

	_, err := svc.ExecuteSeaOrderSplit(ctx, &v1.ExecuteSeaOrderSplitRequest{
		OrderId:            orderID.String(),
		IdempotencyKey:     "split-combined-permission",
		RequestFingerprint: "split-combined-permission-fingerprint",
		Targets: []*v1.SeaOrderSplitTargetInput{
			{ClientTargetKey: "current", TargetType: biz.SplitTargetTypeCurrent},
			{ClientTargetKey: "new", TargetType: biz.SplitTargetTypeNew},
		},
		Results: []*v1.SeaOrderSplitResultInput{
			{ClientResultKey: "original", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "current"},
			{ClientResultKey: "created", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "new"},
		},
		ExpectedVersions: &v1.SeaOrderSplitExpectedVersions{OrderVersion: 1, LinkVersion: 1, AllocationVersion: 1},
	})
	if err != biz.ErrPermissionDenied {
		t.Fatalf("组合拆票缺少整体改配权限应被拒绝，实际错误: %v", err)
	}

	permission := access.OrderPermission(access.OrderBusinessSE, access.OrderReassign)
	ctx = biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       actorID,
		Organization: biz.Organization{ID: orgID},
		Permissions:  []string{permission},
		RoleScopes:   []biz.RoleScope{{RoleCode: "operator", DataScope: biz.DataScopeSelf}},
		RolePermissions: map[string]map[string]struct{}{
			"operator": {permission: {}},
		},
	})
	_, err = svc.ExecuteSeaOrderSplit(ctx, &v1.ExecuteSeaOrderSplitRequest{
		OrderId:            orderID.String(),
		IdempotencyKey:     "split-combined-self-scope",
		RequestFingerprint: "split-combined-self-scope-fingerprint",
		Targets: []*v1.SeaOrderSplitTargetInput{
			{ClientTargetKey: "current", TargetType: biz.SplitTargetTypeCurrent},
			{ClientTargetKey: "new", TargetType: biz.SplitTargetTypeNew},
		},
		Results: []*v1.SeaOrderSplitResultInput{
			{ClientResultKey: "original", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "current"},
			{ClientResultKey: "created", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "new"},
		},
		ExpectedVersions: &v1.SeaOrderSplitExpectedVersions{OrderVersion: 1, LinkVersion: 1, AllocationVersion: 1},
	})
	if err != biz.ErrPermissionDenied {
		t.Fatalf("组合拆票的整体改配权限必须具备组织范围，实际错误: %v", err)
	}
}

func TestSeaOrderChangeService_CandidateTargetMappingKeepsAllIdentifiersAndVersions(t *testing.T) {
	orderID := uuid.New()
	candidateID := uuid.New()
	candidateVersion := uint64(7)
	candidateTEID := uuid.New()
	candidateTEVersion := uint64(9)
	issuerID := uuid.New()

	splitInput, err := mapSplitInput(
		orderID.String(),
		nil,
		[]*v1.SeaOrderSplitTargetInput{
			{
				ClientTargetKey:    "candidate",
				TargetType:         biz.SplitTargetTypeCandidate,
				CandidateId:        strPtr(candidateID.String()),
				CandidateVersion:   &candidateVersion,
				CandidateTeId:      strPtr(candidateTEID.String()),
				CandidateTeVersion: &candidateTEVersion,
				IssuerPartnerId:    strPtr(issuerID.String()),
			},
		},
		nil,
		&v1.SeaOrderSplitExpectedVersions{
			OrderVersion:         1,
			LinkVersion:          2,
			AllocationVersion:    3,
			CandidateMblVersions: map[string]uint64{candidateID.String(): candidateVersion},
			CandidateTeVersions:  map[string]uint64{candidateTEID.String(): candidateTEVersion},
		},
	)
	if err != nil {
		t.Fatalf("mapSplitInput error: %v", err)
	}
	if len(splitInput.Targets) != 1 {
		t.Fatalf("expected one split target, got %d", len(splitInput.Targets))
	}
	target := splitInput.Targets[0]
	if target.CandidateID == nil || *target.CandidateID != candidateID ||
		target.CandidateVersion == nil || *target.CandidateVersion != candidateVersion ||
		target.CandidateTEID == nil || *target.CandidateTEID != candidateTEID ||
		target.CandidateTEVersion == nil || *target.CandidateTEVersion != candidateTEVersion {
		t.Fatalf("candidate split target mapping lost identifier/version: %+v", target)
	}
	if splitInput.ExpectedVersions.CandidateMBLVersions[candidateID] != candidateVersion ||
		splitInput.ExpectedVersions.CandidateTEVersions[candidateTEID] != candidateTEVersion {
		t.Fatalf("candidate expected version mapping mismatch: %+v", splitInput.ExpectedVersions)
	}

	reassignTarget, err := mapReassignTarget(&v1.SeaOrderReassignmentTargetInput{
		TargetType:         biz.SplitTargetTypeCandidate,
		CandidateId:        strPtr(candidateID.String()),
		CandidateVersion:   &candidateVersion,
		CandidateTeId:      strPtr(candidateTEID.String()),
		CandidateTeVersion: &candidateTEVersion,
		IssuerPartnerId:    strPtr(issuerID.String()),
	})
	if err != nil {
		t.Fatalf("mapReassignTarget error: %v", err)
	}
	if reassignTarget.CandidateID == nil || *reassignTarget.CandidateID != candidateID ||
		reassignTarget.CandidateVersion == nil || *reassignTarget.CandidateVersion != candidateVersion ||
		reassignTarget.CandidateTEID == nil || *reassignTarget.CandidateTEID != candidateTEID ||
		reassignTarget.CandidateTEVersion == nil || *reassignTarget.CandidateTEVersion != candidateTEVersion {
		t.Fatalf("candidate reassignment target mapping lost identifier/version: %+v", reassignTarget)
	}
}

func TestSeaOrderChangeService_ListPaginationBoundaries(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       actorID,
		Organization: biz.Organization{ID: orgID},
	})
	svc := NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(&mockSeaOrderChangeRepoForService{}, &mockTransactorForService{}))

	for _, tc := range []struct {
		name     string
		page     int32
		pageSize int32
		wantErr  bool
	}{
		{name: "zero values use defaults", page: 0, pageSize: 0},
		{name: "maximum page size", page: 1, pageSize: 200},
		{name: "page size over maximum", page: 1, pageSize: 201, wantErr: true},
		{name: "negative page", page: -1, pageSize: 20, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ListSeaOrderChangeEvents(ctx, &v1.ListSeaOrderChangeEventsRequest{
				OrderId:  orderID.String(),
				Page:     tc.page,
				PageSize: tc.pageSize,
			})
			if tc.wantErr && err != biz.ErrSeaOrderSplitInvalidArgument {
				t.Fatalf("expected invalid argument, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSeaOrderChangeService_MalformedSplitRequestsBlocked(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       actorID,
		Organization: biz.Organization{ID: orgID},
	})
	svc := NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(&mockSeaOrderChangeRepoForService{
		previewRes: &biz.SeaOrderSplitPreview{IsValid: true, ConservationPassed: true},
		splitEvt:   &biz.SeaOrderSplitEvent{ID: uuid.New(), SourceOrderID: orderID},
	}, &mockTransactorForService{}))

	testCases := []struct {
		name    string
		targets []*v1.SeaOrderSplitTargetInput
		results []*v1.SeaOrderSplitResultInput
	}{
		{
			name: "result引用的client_target_key未定义",
			targets: []*v1.SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			results: []*v1.SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "unmapped-key"},
			},
		},
		{
			name: "targets存在重复client_target_key",
			targets: []*v1.SeaOrderSplitTargetInput{
				{ClientTargetKey: "dup", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "dup", TargetType: biz.SplitTargetTypeCurrent},
			},
			results: []*v1.SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "dup"},
				{ClientResultKey: "new1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "dup"},
			},
		},
		{
			name: "target_type未知非法",
			targets: []*v1.SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: "INVALID_TYPE"},
			},
			results: []*v1.SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "t-1"},
			},
		},
		{
			name: "CURRENT夹带MasterNo",
			targets: []*v1.SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: biz.SplitTargetTypeCurrent, MasterNo: strPtr("MBL999")},
			},
			results: []*v1.SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "t-1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, pErr := svc.PreviewSeaOrderSplit(ctx, &v1.PreviewSeaOrderSplitRequest{
				OrderId: orderID.String(),
				Targets: tc.targets,
				Results: tc.results,
			})
			if pErr != biz.ErrSeaOrderSplitInvalidArgument {
				t.Fatalf("PreviewSeaOrderSplit expected ErrSeaOrderSplitInvalidArgument, got %v", pErr)
			}

			_, eErr := svc.ExecuteSeaOrderSplit(ctx, &v1.ExecuteSeaOrderSplitRequest{
				OrderId:            orderID.String(),
				IdempotencyKey:     "idemp-" + uuid.NewString(),
				RequestFingerprint: "fp-" + uuid.NewString(),
				Targets:            tc.targets,
				Results:            tc.results,
				ExpectedVersions:   &v1.SeaOrderSplitExpectedVersions{OrderVersion: 1, LinkVersion: 1, AllocationVersion: 1},
			})
			if eErr != biz.ErrSeaOrderSplitInvalidArgument {
				t.Fatalf("ExecuteSeaOrderSplit expected ErrSeaOrderSplitInvalidArgument, got %v", eErr)
			}
		})
	}
}

func TestSeaOrderChangeService_ReassignmentTargetTypeAndFieldValidation(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	candMBLID := uuid.New().String()
	candTEID := uuid.New().String()
	issuerID := uuid.New().String()
	u64 := func(v uint64) *uint64 { return &v }

	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       actorID,
		Organization: biz.Organization{ID: orgID},
	})
	svc := NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(&mockSeaOrderChangeRepoForService{
		reasPreview: &biz.SeaOrderReassignmentPreview{IsValid: true},
		reasEvt:     &biz.SeaOrderReassignmentEvent{ID: uuid.New(), OrderID: orderID},
	}, &mockTransactorForService{}))

	testCases := []struct {
		name                 string
		target               *v1.SeaOrderReassignmentTargetInput
		expectedCandidateMBL *uint64
		expectedCandidateTE  *uint64
	}{
		{
			name: "未知target_type被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType: "UNKNOWN",
				MasterNo:   strPtr("MBL123"),
			},
		},
		{
			name: "CURRENT_target_type在改配中被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType: "CURRENT",
				MasterNo:   strPtr("MBL123"),
			},
		},
		{
			name: "空target_type被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType: "",
				MasterNo:   strPtr("MBL123"),
			},
		},
		{
			name: "NEW夹带CandidateId被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:  "NEW",
				MasterNo:    strPtr("NEWMBL01"),
				CandidateId: &candMBLID,
			},
		},
		{
			name: "NEW夹带CandidateVersion被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:       "NEW",
				MasterNo:         strPtr("NEWMBL01"),
				CandidateVersion: u64(1),
			},
		},
		{
			name: "NEW夹带CandidateTeId被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:    "NEW",
				MasterNo:      strPtr("NEWMBL01"),
				CandidateTeId: &candTEID,
			},
		},
		{
			name: "NEW夹带CandidateTeVersion被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:         "NEW",
				MasterNo:           strPtr("NEWMBL01"),
				CandidateTeVersion: u64(1),
			},
		},
		{
			name: "CANDIDATE缺失CandidateId被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:         "CANDIDATE",
				CandidateVersion:   u64(2),
				CandidateTeId:      &candTEID,
				CandidateTeVersion: u64(3),
				IssuerPartnerId:    &issuerID,
			},
			expectedCandidateMBL: u64(2),
			expectedCandidateTE:  u64(3),
		},
		{
			name: "CANDIDATE缺失IssuerPartnerId被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:         "CANDIDATE",
				CandidateId:        &candMBLID,
				CandidateVersion:   u64(2),
				CandidateTeId:      &candTEID,
				CandidateTeVersion: u64(3),
			},
			expectedCandidateMBL: u64(2),
			expectedCandidateTE:  u64(3),
		},
		{
			name: "CANDIDATE缺失CandidateTeVersion被阻断",
			target: &v1.SeaOrderReassignmentTargetInput{
				TargetType:       "CANDIDATE",
				CandidateId:      &candMBLID,
				CandidateVersion: u64(2),
				CandidateTeId:    &candTEID,
				IssuerPartnerId:  &issuerID,
			},
			expectedCandidateMBL: u64(2),
			expectedCandidateTE:  u64(3),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, pErr := svc.PreviewSeaOrderReassignment(ctx, &v1.PreviewSeaOrderReassignmentRequest{
				OrderId: orderID.String(),
				Target:  tc.target,
			})
			if pErr != biz.ErrSeaOrderReassignmentInvalidArgument {
				t.Fatalf("PreviewSeaOrderReassignment expected ErrSeaOrderReassignmentInvalidArgument, got %v", pErr)
			}

			_, eErr := svc.ExecuteSeaOrderReassignment(ctx, &v1.ExecuteSeaOrderReassignmentRequest{
				OrderId:                     orderID.String(),
				IdempotencyKey:              "idemp-" + uuid.NewString(),
				RequestFingerprint:          "fp-" + uuid.NewString(),
				Target:                      tc.target,
				Reason:                      "测试改配",
				ResponsibilityType:          "CARRIER",
				ExpectedOrderVersion:        1,
				ExpectedLinkVersion:         1,
				ExpectedCandidateMblVersion: tc.expectedCandidateMBL,
				ExpectedCandidateTeVersion:  tc.expectedCandidateTE,
			})
			if eErr != biz.ErrSeaOrderReassignmentInvalidArgument {
				t.Fatalf("ExecuteSeaOrderReassignment expected ErrSeaOrderReassignmentInvalidArgument, got %v", eErr)
			}
		})
	}
}
