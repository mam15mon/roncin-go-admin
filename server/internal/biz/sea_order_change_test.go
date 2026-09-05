package biz

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockSeaOrderChangeRepo struct {
	getActionsFunc           func(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderChangeActions, error)
	getSplitCtxFunc          func(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderSplitContext, error)
	previewSplitFunc         func(ctx context.Context, organizationID uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitPreview, error)
	executeSplitFunc         func(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderSplitInput, audit *AuditEvent) (*SeaOrderSplitEvent, error)
	getSplitEventByIdempFunc func(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderSplitEvent, error)
	previewReasFunc          func(ctx context.Context, organizationID uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error)
	executeReasFunc          func(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderReassignmentInput, audit *AuditEvent) (*SeaOrderReassignmentEvent, error)
	getReasEventByIdempFunc  func(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderReassignmentEvent, error)
	getSplitEventFunc        func(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderSplitEvent, error)
	getReassignmentEventFunc func(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderReassignmentEvent, error)
	listEventsFunc           func(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*SeaOrderChangeEventSummary, int32, error)
	getEventFunc             func(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*SeaOrderChangeEventDetail, error)
}

func (m *mockSeaOrderChangeRepo) GetSplitEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderSplitEvent, error) {
	if m.getSplitEventByIdempFunc != nil {
		return m.getSplitEventByIdempFunc(ctx, organizationID, idempotencyKey)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) GetSplitEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderSplitEvent, error) {
	if m.getSplitEventFunc != nil {
		return m.getSplitEventFunc(ctx, organizationID, orderID, eventID)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) GetReassignmentEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderReassignmentEvent, error) {
	if m.getReasEventByIdempFunc != nil {
		return m.getReasEventByIdempFunc(ctx, organizationID, idempotencyKey)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) GetReassignmentEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderReassignmentEvent, error) {
	if m.getReassignmentEventFunc != nil {
		return m.getReassignmentEventFunc(ctx, organizationID, orderID, eventID)
	}
	return nil, nil
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (m *mockSeaOrderChangeRepo) GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderChangeActions, error) {
	if m.getActionsFunc != nil {
		return m.getActionsFunc(ctx, organizationID, orderID)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) GetSplitContext(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderSplitContext, error) {
	if m.getSplitCtxFunc != nil {
		return m.getSplitCtxFunc(ctx, organizationID, orderID)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) PreviewSplit(ctx context.Context, organizationID uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitPreview, error) {
	if m.previewSplitFunc != nil {
		return m.previewSplitFunc(ctx, organizationID, input)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderSplitInput, audit *AuditEvent) (*SeaOrderSplitEvent, error) {
	if m.executeSplitFunc != nil {
		return m.executeSplitFunc(ctx, organizationID, actorID, input, audit)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error) {
	if m.previewReasFunc != nil {
		return m.previewReasFunc(ctx, organizationID, input)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderReassignmentInput, audit *AuditEvent) (*SeaOrderReassignmentEvent, error) {
	if m.executeReasFunc != nil {
		return m.executeReasFunc(ctx, organizationID, actorID, input, audit)
	}
	return nil, nil
}

func (m *mockSeaOrderChangeRepo) ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*SeaOrderChangeEventSummary, int32, error) {
	if m.listEventsFunc != nil {
		return m.listEventsFunc(ctx, organizationID, orderID, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockSeaOrderChangeRepo) GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*SeaOrderChangeEventDetail, error) {
	if m.getEventFunc != nil {
		return m.getEventFunc(ctx, organizationID, orderID, eventID, eventType)
	}
	return nil, nil
}

func TestSeaOrderChangeUsecase_ErrorCodes(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrSeaOrderSplitInvalidArgument, "SEA_ORDER_SPLIT_INVALID_ARGUMENT"},
		{ErrSeaOrderSplitBlocked, "SEA_ORDER_SPLIT_BLOCKED"},
		{ErrSeaOrderSplitConservationFailed, "SEA_ORDER_SPLIT_CONSERVATION_FAILED"},
		{ErrSeaOrderSplitEntityCrossesResults, "SEA_ORDER_SPLIT_ENTITY_CROSSES_RESULTS"},
		{ErrSeaOrderSplitVersionConflict, "SEA_ORDER_SPLIT_VERSION_CONFLICT"},
		{ErrSeaOrderSplitIdempotencyConflict, "SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT"},
		{ErrSeaOrderReassignmentInvalidArgument, "SEA_ORDER_REASSIGNMENT_INVALID_ARGUMENT"},
		{ErrSeaOrderReassignmentBlocked, "SEA_ORDER_REASSIGNMENT_BLOCKED"},
		{ErrSeaOrderReassignmentTargetConflict, "SEA_ORDER_REASSIGNMENT_TARGET_CONFLICT"},
		{ErrSeaOrderReassignmentVersionConflict, "SEA_ORDER_REASSIGNMENT_VERSION_CONFLICT"},
		{ErrSeaOrderReassignmentIdempotencyConflict, "SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT"},
	}

	for _, tc := range tests {
		if tc.err == nil {
			t.Fatalf("expected non-nil error")
		}
		reason := errors.Reason(tc.err)
		if reason != tc.expected {
			t.Errorf("error reason mismatch: got %v, expected %v", reason, tc.expected)
		}
	}
}

func TestSeaOrderChangeUsecase_GetChangeActions(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	orderID := uuid.New()

	called := false
	repo := &mockSeaOrderChangeRepo{
		getActionsFunc: func(ctx context.Context, oid, orderIDVal uuid.UUID) (*SeaOrderChangeActions, error) {
			called = true
			if oid != orgID || orderIDVal != orderID {
				t.Fatalf("IDs mismatch: got %v/%v, want %v/%v", oid, orderIDVal, orgID, orderID)
			}
			return &SeaOrderChangeActions{
				CanSplit:    true,
				CanReassign: true,
			}, nil
		},
	}

	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})
	actions, err := uc.GetChangeActions(ctx, orgID, orderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("repo.GetChangeActions was not called")
	}
	if !actions.CanSplit || !actions.CanReassign {
		t.Errorf("actions result mismatch: %+v", actions)
	}

	// 零值校验
	if _, err := uc.GetChangeActions(ctx, uuid.Nil, orderID); err != ErrSeaOrderSplitInvalidArgument {
		t.Fatalf("expected ErrSeaOrderSplitInvalidArgument on nil org uuid, got %v", err)
	}
	if _, err := uc.GetChangeActions(ctx, orgID, uuid.Nil); err != ErrSeaOrderSplitInvalidArgument {
		t.Fatalf("expected ErrSeaOrderSplitInvalidArgument on nil order uuid, got %v", err)
	}
}

func TestSeaOrderChangeUsecase_PreviewAndExecuteSplit(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	targetKey := "res-origin"
	previewCalls := 0

	repo := &mockSeaOrderChangeRepo{
		previewSplitFunc: func(ctx context.Context, oid uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitPreview, error) {
			previewCalls++
			return &SeaOrderSplitPreview{
				IsValid:            true,
				ConservationPassed: true,
				Baseline: SeaOrderSplitQuantitySummary{
					PackageCount:  100,
					GrossWeightKg: decimal.NewFromFloat(500.25),
					VolumeCbm:     decimal.NewFromFloat(10.5),
				},
				Allocated: SeaOrderSplitQuantitySummary{
					PackageCount:  100,
					GrossWeightKg: decimal.NewFromFloat(500.25),
					VolumeCbm:     decimal.NewFromFloat(10.5),
				},
				Remaining: SeaOrderSplitQuantitySummary{
					PackageCount:  0,
					GrossWeightKg: decimal.Zero,
					VolumeCbm:     decimal.Zero,
				},
			}, nil
		},
		getSplitEventFunc: func(ctx context.Context, oid, oId, eventID uuid.UUID) (*SeaOrderSplitEvent, error) {
			return &SeaOrderSplitEvent{
				ID:            eventID,
				SourceOrderID: orderID,
				SourceOrderNo: "SE20260903001",
				Results: []*SeaOrderSplitResult{
					{OrderID: orderID, OrderNo: "SE20260903001", ResultRole: ResultRoleOriginal, ClientResultKey: "res-origin"},
					{OrderID: uuid.New(), OrderNo: "SE20260903002", ResultRole: ResultRoleCreated, ClientResultKey: "res-new-1"},
				},
			}, nil
		},
		executeSplitFunc: func(ctx context.Context, oid, aid uuid.UUID, input *SeaOrderSplitInput, audit *AuditEvent) (*SeaOrderSplitEvent, error) {
			if input.IdempotencyKey == "" {
				return nil, ErrSeaOrderSplitInvalidArgument
			}
			return &SeaOrderSplitEvent{
				ID:            uuid.New(),
				SourceOrderID: orderID,
				SourceOrderNo: "SE20260903001",
				Results: []*SeaOrderSplitResult{
					{OrderID: orderID, OrderNo: "SE20260903001", ResultRole: ResultRoleOriginal, ClientResultKey: "res-origin"},
					{OrderID: uuid.New(), OrderNo: "SE20260903002", ResultRole: ResultRoleCreated, ClientResultKey: "res-new-1"},
				},
			}, nil
		},
	}

	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})

	// Preview 验证
	input := &SeaOrderSplitInput{
		OrderID: orderID,
		Targets: []*SeaOrderSplitTargetInput{
			{ClientTargetKey: targetKey, TargetType: SplitTargetTypeCurrent},
		},
		Results: []*SeaOrderSplitResultInput{
			{ClientResultKey: "res-origin", ResultRole: ResultRoleOriginal, ClientTargetKey: targetKey},
			{ClientResultKey: "res-new-1", ResultRole: ResultRoleCreated, ClientTargetKey: targetKey},
		},
		ExpectedVersions: &SeaOrderSplitExpectedVersions{
			OrderVersion:      1,
			LinkVersion:       1,
			AllocationVersion: 1,
		},
	}

	preview, err := uc.PreviewSplit(ctx, orgID, input)
	if err != nil {
		t.Fatalf("PreviewSplit error: %v", err)
	}
	if !preview.ConservationPassed || !preview.IsValid {
		t.Errorf("expected conservation passed and valid, got %+v", preview)
	}
	if preview.Remaining.PackageCount != 0 || !preview.Remaining.GrossWeightKg.IsZero() || !preview.Remaining.VolumeCbm.IsZero() {
		t.Errorf("expected 0 remaining, got %+v", preview.Remaining)
	}

	// 缺失结果校验
	invalidInput := &SeaOrderSplitInput{OrderID: uuid.Nil}
	if _, err := uc.PreviewSplit(ctx, orgID, invalidInput); err != ErrSeaOrderSplitInvalidArgument {
		t.Fatalf("expected ErrSeaOrderSplitInvalidArgument, got %v", err)
	}

	// 版本 0 校验
	zeroVerInput := *input
	zeroVerInput.IdempotencyKey = "idemp-test-zero"
	zeroVerInput.RequestFingerprint = "fp-zero"
	zeroVerInput.ExpectedVersions = &SeaOrderSplitExpectedVersions{OrderVersion: 0, LinkVersion: 1, AllocationVersion: 1}
	if _, err := uc.ExecuteSplit(ctx, orgID, actorID, &zeroVerInput); err != ErrSeaOrderSplitInvalidArgument {
		t.Fatalf("expected ErrSeaOrderSplitInvalidArgument on version 0, got %v", err)
	}

	// Execute 验证
	input.IdempotencyKey = "idemp-test-123"
	input.RequestFingerprint = "fp-split-123"
	execRes, err := uc.ExecuteSplit(ctx, orgID, actorID, input)
	if err != nil {
		t.Fatalf("ExecuteSplit error: %v", err)
	}
	if len(execRes.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(execRes.Results))
	}
	if previewCalls != 1 {
		t.Fatalf("ExecuteSplit 不应调用有状态 Preview，Preview 调用次数 = %d，期望仅显式预览的 1 次", previewCalls)
	}
}

func TestSeaOrderChangeUsecase_PreviewAndExecuteReassignment(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	partnerID := uuid.New()

	repo := &mockSeaOrderChangeRepo{
		previewReasFunc: func(ctx context.Context, oid uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error) {
			return &SeaOrderReassignmentPreview{
				IsValid:           true,
				TargetMemberCount: 1,
				Differences: []*VoyageDifference{
					{FieldName: "master_no", Label: "提单号", CurrentValue: "OLDMBL", TargetValue: "NEWMBL", IsDifferent: true},
				},
			}, nil
		},
		getReassignmentEventFunc: func(ctx context.Context, oid, oId, eventID uuid.UUID) (*SeaOrderReassignmentEvent, error) {
			return &SeaOrderReassignmentEvent{
				ID:      eventID,
				OrderID: orderID,
				OrderNo: "SE20260903001",
			}, nil
		},
		executeReasFunc: func(ctx context.Context, oid, aid uuid.UUID, input *SeaOrderReassignmentInput, audit *AuditEvent) (*SeaOrderReassignmentEvent, error) {
			return &SeaOrderReassignmentEvent{
				ID:      uuid.New(),
				OrderID: orderID,
				OrderNo: "SE20260903001",
			}, nil
		},
	}

	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})

	input := &SeaOrderReassignmentInput{
		OrderID: orderID,
		Target: &SeaOrderReassignmentTargetInput{
			TargetType:      "NEW",
			MasterNo:        "NEWMBL",
			IssuerPartnerID: &partnerID,
		},
		Reason:               "客户要求改配",
		ResponsibilityType:   "CUSTOMER",
		IdempotencyKey:       "reas-idemp-123",
		RequestFingerprint:   "fp-reas-123",
		ExpectedOrderVersion: 1,
		ExpectedLinkVersion:  1,
	}

	preview, err := uc.PreviewReassignment(ctx, orgID, input)
	if err != nil {
		t.Fatalf("PreviewReassignment error: %v", err)
	}
	if len(preview.Differences) != 1 || !preview.Differences[0].IsDifferent {
		t.Errorf("expected differences detected, got %+v", preview.Differences)
	}

	// 版本 0 校验
	zeroVerInput := *input
	zeroVerInput.ExpectedOrderVersion = 0
	if _, err := uc.ExecuteReassignment(ctx, orgID, actorID, &zeroVerInput); err != ErrSeaOrderReassignmentInvalidArgument {
		t.Fatalf("expected ErrSeaOrderReassignmentInvalidArgument on version 0, got %v", err)
	}

	res, err := uc.ExecuteReassignment(ctx, orgID, actorID, input)
	if err != nil {
		t.Fatalf("ExecuteReassignment error: %v", err)
	}
	if res.OrderNo != "SE20260903001" {
		t.Errorf("expected SE20260903001, got %s", res.OrderNo)
	}

	// 缺少原因与责任归属
	invalidInput := &SeaOrderReassignmentInput{
		OrderID:              orderID,
		Target:               input.Target,
		ExpectedOrderVersion: 1,
		ExpectedLinkVersion:  1,
	}
	if _, err := uc.ExecuteReassignment(ctx, orgID, actorID, invalidInput); err != ErrSeaOrderReassignmentInvalidArgument {
		t.Fatalf("expected ErrSeaOrderReassignmentInvalidArgument, got %v", err)
	}
}

func TestSeaOrderChangeUsecase_ListAndGetEvents(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	orderID := uuid.New()
	eventID := uuid.New()

	repo := &mockSeaOrderChangeRepo{
		listEventsFunc: func(ctx context.Context, oid, orderIDVal uuid.UUID, page, pageSize int32) ([]*SeaOrderChangeEventSummary, int32, error) {
			return []*SeaOrderChangeEventSummary{
				{
					ID:        eventID,
					EventType: "SPLIT",
					CreatedAt: time.Now(),
				},
			}, 1, nil
		},
		getEventFunc: func(ctx context.Context, oid, orderIDVal, eid uuid.UUID, eventType string) (*SeaOrderChangeEventDetail, error) {
			return &SeaOrderChangeEventDetail{
				ID:        eid,
				EventType: eventType,
			}, nil
		},
	}

	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})

	events, total, err := uc.ListChangeEvents(ctx, orgID, orderID, 1, 10)
	if err != nil {
		t.Fatalf("ListChangeEvents error: %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Errorf("expected 1 event, got %d, total %d", len(events), total)
	}

	detail, err := uc.GetChangeEvent(ctx, orgID, orderID, eventID, "SPLIT")
	if err != nil {
		t.Fatalf("GetChangeEvent error: %v", err)
	}
	if detail.ID != eventID {
		t.Errorf("expected eventID %v, got %v", eventID, detail.ID)
	}
}

func TestSeaOrderChangeUsecase_ExecuteCandidateRequiresAllVersions(t *testing.T) {
	uc := NewSeaOrderChangeUsecase(&mockSeaOrderChangeRepo{}, &mockTransactor{})
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	candidateID := uuid.New()
	candidateVersion := uint64(1)
	candidateTEID := uuid.New()
	candidateTEVersion := uint64(2)
	issuerID := uuid.New()

	splitInput := &SeaOrderSplitInput{
		OrderID:            orderID,
		IdempotencyKey:     "split-candidate-versions",
		RequestFingerprint: "split-candidate-versions-fingerprint",
		Targets: []*SeaOrderSplitTargetInput{
			{ClientTargetKey: "current", TargetType: SplitTargetTypeCurrent},
			{
				ClientTargetKey:    "candidate",
				TargetType:         SplitTargetTypeCandidate,
				CandidateID:        &candidateID,
				CandidateVersion:   &candidateVersion,
				CandidateTEID:      &candidateTEID,
				CandidateTEVersion: &candidateTEVersion,
				IssuerPartnerID:    &issuerID,
			},
		},
		Results: []*SeaOrderSplitResultInput{
			{ClientResultKey: "original", ResultRole: ResultRoleOriginal, ClientTargetKey: "current"},
			{ClientResultKey: "created", ResultRole: ResultRoleCreated, ClientTargetKey: "candidate"},
		},
		ExpectedVersions: &SeaOrderSplitExpectedVersions{
			OrderVersion:         1,
			LinkVersion:          1,
			AllocationVersion:    1,
			CandidateMBLVersions: map[uuid.UUID]uint64{candidateID: candidateVersion},
		},
	}
	if _, err := uc.ExecuteSplit(context.Background(), orgID, actorID, splitInput); err != ErrSeaOrderSplitInvalidArgument {
		t.Fatalf("拆票缺少候选运输执行版本应被边界拦截，实际错误: %v", err)
	}

	reassignInput := &SeaOrderReassignmentInput{
		OrderID:            orderID,
		IdempotencyKey:     "reassign-candidate-versions",
		RequestFingerprint: "reassign-candidate-versions-fingerprint",
		Target: &SeaOrderReassignmentTargetInput{
			TargetType:         SplitTargetTypeCandidate,
			CandidateID:        &candidateID,
			CandidateVersion:   &candidateVersion,
			CandidateTEID:      &candidateTEID,
			CandidateTEVersion: &candidateTEVersion,
			IssuerPartnerID:    &issuerID,
		},
		Reason:                      "改配测试",
		ResponsibilityType:          ResponsibilityTypeOwnCompany,
		ExpectedOrderVersion:        1,
		ExpectedLinkVersion:         1,
		ExpectedCandidateMBLVersion: &candidateVersion,
	}
	if _, err := uc.ExecuteReassignment(context.Background(), orgID, actorID, reassignInput); err != ErrSeaOrderReassignmentInvalidArgument {
		t.Fatalf("改配缺少候选运输执行版本应被边界拦截，实际错误: %v", err)
	}

	wrongCandidateVersion := candidateVersion + 1
	reassignInput.ExpectedCandidateTEVersion = &candidateTEVersion
	reassignInput.Target.CandidateVersion = &wrongCandidateVersion
	if _, err := uc.ExecuteReassignment(context.Background(), orgID, actorID, reassignInput); err != ErrSeaOrderReassignmentInvalidArgument {
		t.Fatalf("改配目标版本与顶层预期版本不一致应被边界拦截，实际错误: %v", err)
	}
}

func TestSeaOrderChangeUsecase_IdempotencyRecoveryPropagatesLookupErrors(t *testing.T) {
	lookupErr := stderrors.New("idempotency lookup failed")
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()

	splitLookups := 0
	splitRepo := &mockSeaOrderChangeRepo{
		getSplitEventByIdempFunc: func(context.Context, uuid.UUID, string) (*SeaOrderSplitEvent, error) {
			splitLookups++
			if splitLookups == 3 {
				return nil, lookupErr
			}
			return nil, nil
		},
		previewSplitFunc: func(context.Context, uuid.UUID, *SeaOrderSplitInput) (*SeaOrderSplitPreview, error) {
			return &SeaOrderSplitPreview{IsValid: true, ConservationPassed: true}, nil
		},
		executeSplitFunc: func(context.Context, uuid.UUID, uuid.UUID, *SeaOrderSplitInput, *AuditEvent) (*SeaOrderSplitEvent, error) {
			return nil, ErrSeaOrderSplitIdempotencyConflict
		},
	}
	splitUC := NewSeaOrderChangeUsecase(splitRepo, &mockTransactor{})
	_, err := splitUC.ExecuteSplit(context.Background(), orgID, actorID, &SeaOrderSplitInput{
		OrderID:            orderID,
		IdempotencyKey:     "split-lookup-error",
		RequestFingerprint: "split-lookup-error-fingerprint",
		Targets: []*SeaOrderSplitTargetInput{
			{ClientTargetKey: "current", TargetType: SplitTargetTypeCurrent},
		},
		Results: []*SeaOrderSplitResultInput{
			{ClientResultKey: "original", ResultRole: ResultRoleOriginal, ClientTargetKey: "current"},
			{ClientResultKey: "created", ResultRole: ResultRoleCreated, ClientTargetKey: "current"},
		},
		ExpectedVersions: &SeaOrderSplitExpectedVersions{OrderVersion: 1, LinkVersion: 1, AllocationVersion: 1},
	})
	if !stderrors.Is(err, lookupErr) {
		t.Fatalf("split recovery must propagate lookup error, got %v", err)
	}

	reassignLookups := 0
	reassignRepo := &mockSeaOrderChangeRepo{
		getReasEventByIdempFunc: func(context.Context, uuid.UUID, string) (*SeaOrderReassignmentEvent, error) {
			reassignLookups++
			if reassignLookups == 3 {
				return nil, lookupErr
			}
			return nil, nil
		},
		previewReasFunc: func(context.Context, uuid.UUID, *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error) {
			return &SeaOrderReassignmentPreview{IsValid: true}, nil
		},
		executeReasFunc: func(context.Context, uuid.UUID, uuid.UUID, *SeaOrderReassignmentInput, *AuditEvent) (*SeaOrderReassignmentEvent, error) {
			return nil, ErrSeaOrderReassignmentIdempotencyConflict
		},
	}
	reassignUC := NewSeaOrderChangeUsecase(reassignRepo, &mockTransactor{})
	_, err = reassignUC.ExecuteReassignment(context.Background(), orgID, actorID, &SeaOrderReassignmentInput{
		OrderID:              orderID,
		IdempotencyKey:       "reassign-lookup-error",
		RequestFingerprint:   "reassign-lookup-error-fingerprint",
		Reason:               "船期调整",
		ResponsibilityType:   ResponsibilityTypeCarrier,
		Target:               &SeaOrderReassignmentTargetInput{TargetType: SplitTargetTypeNew, MasterNo: "NEWMBL001"},
		ExpectedOrderVersion: 1,
		ExpectedLinkVersion:  1,
	})
	if !stderrors.Is(err, lookupErr) {
		t.Fatalf("reassignment recovery must propagate lookup error, got %v", err)
	}
}

func TestSeaOrderSplit_TargetAndResultValidation(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	partnerID := uuid.New()
	candMBLID := uuid.New()
	candTEID := uuid.New()
	u64 := func(v uint64) *uint64 { return &v }

	repo := &mockSeaOrderChangeRepo{
		previewSplitFunc: func(ctx context.Context, org uuid.UUID, in *SeaOrderSplitInput) (*SeaOrderSplitPreview, error) {
			return &SeaOrderSplitPreview{IsValid: true, ConservationPassed: true}, nil
		},
		executeSplitFunc: func(ctx context.Context, org, act uuid.UUID, in *SeaOrderSplitInput, audit *AuditEvent) (*SeaOrderSplitEvent, error) {
			return &SeaOrderSplitEvent{ID: uuid.New(), SourceOrderID: in.OrderID}, nil
		},
		getSplitEventByIdempFunc: func(ctx context.Context, org uuid.UUID, key string) (*SeaOrderSplitEvent, error) {
			return nil, nil
		},
		getSplitEventFunc: func(ctx context.Context, org, ord, eid uuid.UUID) (*SeaOrderSplitEvent, error) {
			return &SeaOrderSplitEvent{ID: eid, SourceOrderID: ord}, nil
		},
	}
	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})

	validExpected := &SeaOrderSplitExpectedVersions{
		OrderVersion:      1,
		LinkVersion:       1,
		AllocationVersion: 1,
		CandidateMBLVersions: map[uuid.UUID]uint64{
			candMBLID: 2,
		},
		CandidateTEVersions: map[uuid.UUID]uint64{
			candTEID: 3,
		},
	}

	testCases := []struct {
		name       string
		targets    []*SeaOrderSplitTargetInput
		results    []*SeaOrderSplitResultInput
		expected   *SeaOrderSplitExpectedVersions
		wantReason string
	}{
		{
			name: "client_target_key为空",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "", TargetType: SplitTargetTypeCurrent},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: ""},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: ""},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "client_target_key重复",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: SplitTargetTypeCurrent},
				{ClientTargetKey: "t-1", TargetType: SplitTargetTypeCurrent},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-1"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "target_type未知非法",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: "OTHER"},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-1"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "result引用的client_target_key未在targets定义",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: SplitTargetTypeCurrent},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "undefined-target"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "CURRENT夹带候选MBL_ID",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: SplitTargetTypeCurrent, CandidateID: &candMBLID},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-1"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "CURRENT夹带MasterNo",
			targets: []*SeaOrderSplitTargetInput{
				{ClientTargetKey: "t-1", TargetType: SplitTargetTypeCurrent, MasterNo: "MBL123"},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-1"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-1"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "CANDIDATE缺失TE版本",
			targets: []*SeaOrderSplitTargetInput{
				{
					ClientTargetKey:  "t-cand",
					TargetType:       SplitTargetTypeCandidate,
					CandidateID:      &candMBLID,
					CandidateVersion: u64(2),
					CandidateTEID:    &candTEID,
					IssuerPartnerID:  &partnerID,
				},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-cand"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-cand"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "CANDIDATE与ExpectedVersions不一致",
			targets: []*SeaOrderSplitTargetInput{
				{
					ClientTargetKey:    "t-cand",
					TargetType:         SplitTargetTypeCandidate,
					CandidateID:        &candMBLID,
					CandidateVersion:   u64(2),
					CandidateTEID:      &candTEID,
					CandidateTEVersion: u64(999),
					IssuerPartnerID:    &partnerID,
				},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-cand"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-cand"},
			},
			expected:   validExpected,
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "NEW目标夹带CandidateID",
			targets: []*SeaOrderSplitTargetInput{
				{
					ClientTargetKey: "t-new",
					TargetType:      SplitTargetTypeNew,
					CandidateID:     &candMBLID,
					MasterNo:        "NEWSPLIT999",
					IssuerPartnerID: &partnerID,
				},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-new"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-new"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "NEW目标缺少IssuerPartnerID",
			targets: []*SeaOrderSplitTargetInput{
				{
					ClientTargetKey: "t-new",
					TargetType:      SplitTargetTypeNew,
					MasterNo:        "NEWSPLIT999",
				},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-new"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-new"},
			},
			wantReason: "SEA_ORDER_SPLIT_INVALID_ARGUMENT",
		},
		{
			name: "NEW目标MasterNo包含非法连字符",
			targets: []*SeaOrderSplitTargetInput{
				{
					ClientTargetKey: "t-new",
					TargetType:      SplitTargetTypeNew,
					MasterNo:        "NEW-MBL-INVALID",
					IssuerPartnerID: &partnerID,
				},
			},
			results: []*SeaOrderSplitResultInput{
				{ClientResultKey: "orig", ResultRole: ResultRoleOriginal, ClientTargetKey: "t-new"},
				{ClientResultKey: "new1", ResultRole: ResultRoleCreated, ClientTargetKey: "t-new"},
			},
			wantReason: "SEA_MASTER_BILL_INVALID_ARGUMENT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp := tc.expected
			if exp == nil {
				exp = validExpected
			}
			previewInput := &SeaOrderSplitInput{
				OrderID:          orderID,
				Targets:          tc.targets,
				Results:          tc.results,
				ExpectedVersions: exp,
			}
			_, pErr := uc.PreviewSplit(ctx, orgID, previewInput)
			if pErr == nil {
				t.Fatalf("PreviewSplit expected error but got nil")
			}
			if reason := errors.Reason(pErr); reason != tc.wantReason {
				t.Errorf("PreviewSplit expected reason %q, got %q (%v)", tc.wantReason, reason, pErr)
			}

			execInput := &SeaOrderSplitInput{
				OrderID:            orderID,
				IdempotencyKey:     "idemp-" + uuid.NewString(),
				RequestFingerprint: "fp-" + uuid.NewString(),
				Targets:            tc.targets,
				Results:            tc.results,
				ExpectedVersions:   exp,
			}
			_, eErr := uc.ExecuteSplit(ctx, orgID, actorID, execInput)
			if eErr == nil {
				t.Fatalf("ExecuteSplit expected error but got nil")
			}
			if reason := errors.Reason(eErr); reason != tc.wantReason {
				t.Errorf("ExecuteSplit expected reason %q, got %q (%v)", tc.wantReason, reason, eErr)
			}
		})
	}
}

func TestSeaOrderReassignment_TargetTypeAndFieldValidation(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	candMBLID := uuid.New()
	candTEID := uuid.New()
	issuerID := uuid.New()
	candMBLVer := uint64(2)
	candTEVer := uint64(3)
	u64 := func(v uint64) *uint64 { return &v }

	repo := &mockSeaOrderChangeRepo{
		previewReasFunc: func(ctx context.Context, oid uuid.UUID, in *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error) {
			return &SeaOrderReassignmentPreview{IsValid: true}, nil
		},
		executeReasFunc: func(ctx context.Context, oid, aid uuid.UUID, in *SeaOrderReassignmentInput, audit *AuditEvent) (*SeaOrderReassignmentEvent, error) {
			return &SeaOrderReassignmentEvent{ID: uuid.New(), OrderID: in.OrderID}, nil
		},
		getReassignmentEventFunc: func(ctx context.Context, oid, oId, eventID uuid.UUID) (*SeaOrderReassignmentEvent, error) {
			return &SeaOrderReassignmentEvent{ID: eventID, OrderID: oId}, nil
		},
	}
	uc := NewSeaOrderChangeUsecase(repo, &mockTransactor{})

	testCases := []struct {
		name                 string
		target               *SeaOrderReassignmentTargetInput
		expectedCandidateMBL *uint64
		expectedCandidateTE  *uint64
		wantError            bool
	}{
		{
			name:      "未知target_type被阻断",
			target:    &SeaOrderReassignmentTargetInput{TargetType: "UNKNOWN", MasterNo: "MBL123"},
			wantError: true,
		},
		{
			name:      "CURRENT_target_type在改配中被阻断",
			target:    &SeaOrderReassignmentTargetInput{TargetType: SplitTargetTypeCurrent, MasterNo: "MBL123"},
			wantError: true,
		},
		{
			name:      "空target_type被阻断",
			target:    &SeaOrderReassignmentTargetInput{TargetType: "", MasterNo: "MBL123"},
			wantError: true,
		},
		{
			name: "NEW夹带CandidateID被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:  SplitTargetTypeNew,
				MasterNo:    "NEWMBL01",
				CandidateID: &candMBLID,
			},
			wantError: true,
		},
		{
			name: "NEW夹带CandidateVersion被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:       SplitTargetTypeNew,
				MasterNo:         "NEWMBL01",
				CandidateVersion: u64(1),
			},
			wantError: true,
		},
		{
			name: "NEW夹带CandidateTEID被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:    SplitTargetTypeNew,
				MasterNo:      "NEWMBL01",
				CandidateTEID: &candTEID,
			},
			wantError: true,
		},
		{
			name: "NEW夹带CandidateTEVersion被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:         SplitTargetTypeNew,
				MasterNo:           "NEWMBL01",
				CandidateTEVersion: u64(1),
			},
			wantError: true,
		},
		{
			name: "CANDIDATE缺失CandidateID被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:         SplitTargetTypeCandidate,
				CandidateVersion:   u64(candMBLVer),
				CandidateTEID:      &candTEID,
				CandidateTEVersion: u64(candTEVer),
				IssuerPartnerID:    &issuerID,
			},
			expectedCandidateMBL: u64(candMBLVer),
			expectedCandidateTE:  u64(candTEVer),
			wantError:            true,
		},
		{
			name: "CANDIDATE缺失IssuerPartnerID被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:         SplitTargetTypeCandidate,
				CandidateID:        &candMBLID,
				CandidateVersion:   u64(candMBLVer),
				CandidateTEID:      &candTEID,
				CandidateTEVersion: u64(candTEVer),
			},
			expectedCandidateMBL: u64(candMBLVer),
			expectedCandidateTE:  u64(candTEVer),
			wantError:            true,
		},
		{
			name: "CANDIDATE缺失CandidateTEVersion被阻断",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:       SplitTargetTypeCandidate,
				CandidateID:      &candMBLID,
				CandidateVersion: u64(candMBLVer),
				CandidateTEID:    &candTEID,
				IssuerPartnerID:  &issuerID,
			},
			expectedCandidateMBL: u64(candMBLVer),
			expectedCandidateTE:  u64(candTEVer),
			wantError:            true,
		},
		{
			name: "合法NEW目标校验成功",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:      SplitTargetTypeNew,
				MasterNo:        "NEWMBL001",
				IssuerPartnerID: &issuerID,
			},
			wantError: false,
		},
		{
			name: "合法CANDIDATE目标校验成功",
			target: &SeaOrderReassignmentTargetInput{
				TargetType:         SplitTargetTypeCandidate,
				CandidateID:        &candMBLID,
				CandidateVersion:   u64(candMBLVer),
				CandidateTEID:      &candTEID,
				CandidateTEVersion: u64(candTEVer),
				IssuerPartnerID:    &issuerID,
			},
			expectedCandidateMBL: u64(candMBLVer),
			expectedCandidateTE:  u64(candTEVer),
			wantError:            false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			previewInput := &SeaOrderReassignmentInput{
				OrderID: orderID,
				Target:  tc.target,
			}
			_, pErr := uc.PreviewReassignment(ctx, orgID, previewInput)
			if tc.wantError && pErr != ErrSeaOrderReassignmentInvalidArgument {
				t.Fatalf("PreviewReassignment expected ErrSeaOrderReassignmentInvalidArgument, got %v", pErr)
			}
			if !tc.wantError && pErr != nil {
				t.Fatalf("PreviewReassignment unexpected error: %v", pErr)
			}

			execInput := &SeaOrderReassignmentInput{
				OrderID:                     orderID,
				IdempotencyKey:              "idemp-" + uuid.NewString(),
				RequestFingerprint:          "fp-" + uuid.NewString(),
				Target:                      tc.target,
				Reason:                      "测试改配",
				ResponsibilityType:          ResponsibilityTypeCarrier,
				ExpectedOrderVersion:        1,
				ExpectedLinkVersion:         1,
				ExpectedCandidateMBLVersion: tc.expectedCandidateMBL,
				ExpectedCandidateTEVersion:  tc.expectedCandidateTE,
			}
			_, eErr := uc.ExecuteReassignment(ctx, orgID, actorID, execInput)
			if tc.wantError && eErr != ErrSeaOrderReassignmentInvalidArgument {
				t.Fatalf("ExecuteReassignment expected ErrSeaOrderReassignmentInvalidArgument, got %v", eErr)
			}
			if !tc.wantError && eErr != nil {
				t.Fatalf("ExecuteReassignment unexpected error: %v", eErr)
			}
		})
	}
}
