package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type backgroundTaskRepoStub struct {
	enqueuedOrgID uuid.UUID
	enqueuedTask  *BackgroundTask

	claimOrgID uuid.UUID
	claimKinds []BackgroundTaskKind
	claimLease time.Duration
	claimNow   time.Time
	claimTask  *BackgroundTask
	claimError error

	completeOrgID      uuid.UUID
	completeID         uuid.UUID
	completeLeaseToken string

	failOrgID        uuid.UUID
	failID           uuid.UUID
	failLeaseToken   string
	failErrorMessage string
	failNextRunAt    time.Time

	getOrgID uuid.UUID
	getID    uuid.UUID

	listOrgID   uuid.UUID
	listOptions BackgroundTaskListOptions
	listResult  *BackgroundTaskList

	requeueOrgID uuid.UUID
	requeueID    uuid.UUID
	requeueNow   time.Time
	requeueAudit *AuditEvent
	requeueTask  *BackgroundTask
}

func (s *backgroundTaskRepoStub) Enqueue(_ context.Context, organizationID uuid.UUID, task *BackgroundTask) (*BackgroundTask, error) {
	s.enqueuedOrgID = organizationID
	s.enqueuedTask = task
	task.ID = uuid.New()
	return task, nil
}

func (s *backgroundTaskRepoStub) Claim(_ context.Context, organizationID uuid.UUID, kinds []BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*BackgroundTask, error) {
	s.claimOrgID = organizationID
	s.claimKinds = kinds
	s.claimLease = leaseDuration
	s.claimNow = now
	return &BackgroundTask{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Status:         BackgroundTaskStatusRunning,
	}, nil
}

func (s *backgroundTaskRepoStub) ClaimAny(_ context.Context, kinds []BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*BackgroundTask, error) {
	s.claimKinds = kinds
	s.claimLease = leaseDuration
	s.claimNow = now
	if s.claimError != nil {
		return nil, s.claimError
	}
	if s.claimTask != nil {
		return s.claimTask, nil
	}
	return &BackgroundTask{ID: uuid.New(), OrganizationID: uuid.New(), Status: BackgroundTaskStatusRunning}, nil
}

func (s *backgroundTaskRepoStub) Complete(_ context.Context, organizationID, id uuid.UUID, leaseToken string) (*BackgroundTask, error) {
	s.completeOrgID = organizationID
	s.completeID = id
	s.completeLeaseToken = leaseToken
	return &BackgroundTask{
		ID:             id,
		OrganizationID: organizationID,
		Status:         BackgroundTaskStatusSucceeded,
	}, nil
}

func (s *backgroundTaskRepoStub) Fail(_ context.Context, organizationID, id uuid.UUID, leaseToken, errorMessage string, nextRunAt time.Time) (*BackgroundTask, error) {
	s.failOrgID = organizationID
	s.failID = id
	s.failLeaseToken = leaseToken
	s.failErrorMessage = errorMessage
	s.failNextRunAt = nextRunAt
	return &BackgroundTask{
		ID:             id,
		OrganizationID: organizationID,
		Status:         BackgroundTaskStatusFailed,
		LastError:      &errorMessage,
	}, nil
}

func (s *backgroundTaskRepoStub) Get(_ context.Context, organizationID, id uuid.UUID) (*BackgroundTask, error) {
	s.getOrgID = organizationID
	s.getID = id
	return &BackgroundTask{
		ID:             id,
		OrganizationID: organizationID,
	}, nil
}

func (s *backgroundTaskRepoStub) List(_ context.Context, organizationID uuid.UUID, options BackgroundTaskListOptions) (*BackgroundTaskList, error) {
	s.listOrgID = organizationID
	s.listOptions = options
	if s.listResult != nil {
		return s.listResult, nil
	}
	return &BackgroundTaskList{
		Items:    []*BackgroundTask{},
		Total:    0,
		Page:     options.Page,
		PageSize: options.PageSize,
	}, nil
}

func (s *backgroundTaskRepoStub) Requeue(_ context.Context, organizationID, id uuid.UUID, now time.Time, audit *AuditEvent) (*BackgroundTask, error) {
	s.requeueOrgID = organizationID
	s.requeueID = id
	s.requeueNow = now
	s.requeueAudit = audit
	if s.requeueTask != nil {
		return s.requeueTask, nil
	}
	return &BackgroundTask{
		ID:             id,
		OrganizationID: organizationID,
		Kind:           BackgroundTaskKindMasterDataImport,
		Status:         BackgroundTaskStatusPending,
		Attempts:       0,
		MaxAttempts:    3,
		NextRunAt:      now,
	}, nil
}

var _ BackgroundTaskRepo = (*backgroundTaskRepoStub)(nil)

func TestBackgroundTaskKindValid(t *testing.T) {
	validKinds := []BackgroundTaskKind{
		BackgroundTaskKindMasterDataImport,
		BackgroundTaskKindUnlocodeImport,
		BackgroundTaskKindOrderReminder,
		BackgroundTaskKindIntegration,
		BackgroundTaskKindDingTalkNotice,
	}
	for _, k := range validKinds {
		if !k.Valid() {
			t.Fatalf("expected kind %s to be valid", k)
		}
	}

	invalidKinds := []BackgroundTaskKind{
		"",
		"INVALID",
		"UNKNOWN",
		"master_data_import",
	}
	for _, k := range invalidKinds {
		if k.Valid() {
			t.Fatalf("expected kind %s to be invalid", k)
		}
	}
}

func TestBackgroundTaskStatusValid(t *testing.T) {
	validStatuses := []BackgroundTaskStatus{
		BackgroundTaskStatusPending,
		BackgroundTaskStatusRunning,
		BackgroundTaskStatusSucceeded,
		BackgroundTaskStatusFailed,
		BackgroundTaskStatusDeadLetter,
	}
	for _, s := range validStatuses {
		if !s.Valid() {
			t.Fatalf("expected status %s to be valid", s)
		}
	}

	invalidStatuses := []BackgroundTaskStatus{
		"",
		"INVALID",
		"UNKNOWN",
		"pending",
	}
	for _, s := range invalidStatuses {
		if s.Valid() {
			t.Fatalf("expected status %s to be invalid", s)
		}
	}
}

func TestBackgroundTaskEnqueueValidation(t *testing.T) {
	repo := &backgroundTaskRepoStub{}
	uc := NewBackgroundTaskUsecase(repo)
	orgID := uuid.New()

	// organizationID is nil
	if _, err := uc.Enqueue(context.Background(), uuid.Nil, &BackgroundTask{
		Kind:           BackgroundTaskKindMasterDataImport,
		IdempotencyKey: "k-1",
	}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// input is nil
	if _, err := uc.Enqueue(context.Background(), orgID, nil); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil input, got %v", err)
	}

	// Kind is invalid
	if _, err := uc.Enqueue(context.Background(), orgID, &BackgroundTask{
		Kind:           BackgroundTaskKind("INVALID_KIND"),
		IdempotencyKey: "k-1",
	}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for invalid kind, got %v", err)
	}

	// IdempotencyKey is empty or whitespace
	for _, key := range []string{"", "   ", "\t\n"} {
		if _, err := uc.Enqueue(context.Background(), orgID, &BackgroundTask{
			Kind:           BackgroundTaskKindMasterDataImport,
			IdempotencyKey: key,
		}); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for empty idempotency key %q, got %v", key, err)
		}
	}

	// IdempotencyKey exceeds 128 runes
	if _, err := uc.Enqueue(context.Background(), orgID, &BackgroundTask{
		Kind:           BackgroundTaskKindMasterDataImport,
		IdempotencyKey: strings.Repeat("a", 129),
	}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for idempotency key > 128 runes, got %v", err)
	}

	// MaxAttempts out of range (< 1 or > 10)
	for _, maxAttempts := range []int{-1, -10, 11, 100} {
		if _, err := uc.Enqueue(context.Background(), orgID, &BackgroundTask{
			Kind:           BackgroundTaskKindMasterDataImport,
			IdempotencyKey: "k-1",
			MaxAttempts:    maxAttempts,
		}); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for maxAttempts=%d, got %v", maxAttempts, err)
		}
	}
}

func TestBackgroundTaskEnqueueDefaultsAndSuccess(t *testing.T) {
	fixedNow := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	repo := &backgroundTaskRepoStub{}
	uc := &BackgroundTaskUsecase{
		repo: repo,
		now:  func() time.Time { return fixedNow },
	}
	orgID := uuid.New()

	// MaxAttempts == 0 defaults to 3; NextRunAt zero value defaults to uc.now()
	task, err := uc.Enqueue(context.Background(), orgID, &BackgroundTask{
		Kind:           BackgroundTaskKindMasterDataImport,
		IdempotencyKey: "  import-key-001  ",
		MaxAttempts:    0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatalf("expected non-nil task returned")
	}
	if repo.enqueuedOrgID != orgID {
		t.Fatalf("expected repo enqueuedOrgID %v, got %v", orgID, repo.enqueuedOrgID)
	}
	if repo.enqueuedTask == nil {
		t.Fatalf("expected repo enqueuedTask to be populated")
	}
	if repo.enqueuedTask.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %v, got %v", orgID, repo.enqueuedTask.OrganizationID)
	}
	if repo.enqueuedTask.Kind != BackgroundTaskKindMasterDataImport {
		t.Fatalf("expected Kind %v, got %v", BackgroundTaskKindMasterDataImport, repo.enqueuedTask.Kind)
	}
	if repo.enqueuedTask.IdempotencyKey != "import-key-001" {
		t.Fatalf("expected trimmed IdempotencyKey 'import-key-001', got %q", repo.enqueuedTask.IdempotencyKey)
	}
	if repo.enqueuedTask.Status != BackgroundTaskStatusPending {
		t.Fatalf("expected Status PENDING, got %v", repo.enqueuedTask.Status)
	}
	if repo.enqueuedTask.Attempts != 0 {
		t.Fatalf("expected Attempts 0, got %d", repo.enqueuedTask.Attempts)
	}
	if repo.enqueuedTask.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts default to 3, got %d", repo.enqueuedTask.MaxAttempts)
	}
	if !repo.enqueuedTask.NextRunAt.Equal(fixedNow) {
		t.Fatalf("expected NextRunAt default to fixedNow %v, got %v", fixedNow, repo.enqueuedTask.NextRunAt)
	}

	// Explicit MaxAttempts and NextRunAt
	customRunAt := fixedNow.Add(10 * time.Minute)
	_, err = uc.Enqueue(context.Background(), orgID, &BackgroundTask{
		Kind:           BackgroundTaskKindUnlocodeImport,
		IdempotencyKey: "unlocode-key",
		MaxAttempts:    5,
		NextRunAt:      customRunAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.enqueuedTask.MaxAttempts != 5 {
		t.Fatalf("expected MaxAttempts 5, got %d", repo.enqueuedTask.MaxAttempts)
	}
	if !repo.enqueuedTask.NextRunAt.Equal(customRunAt) {
		t.Fatalf("expected NextRunAt %v, got %v", customRunAt, repo.enqueuedTask.NextRunAt)
	}
}

func TestBackgroundTaskClaimValidation(t *testing.T) {
	fixedNow := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	repo := &backgroundTaskRepoStub{}
	uc := &BackgroundTaskUsecase{
		repo: repo,
		now:  func() time.Time { return fixedNow },
	}
	orgID := uuid.New()

	// organizationID is nil
	if _, err := uc.Claim(context.Background(), uuid.Nil, []BackgroundTaskKind{BackgroundTaskKindMasterDataImport}, time.Minute); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// leaseDuration <= 0
	for _, lease := range []time.Duration{0, -time.Second, -time.Minute} {
		if _, err := uc.Claim(context.Background(), orgID, []BackgroundTaskKind{BackgroundTaskKindMasterDataImport}, lease); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for leaseDuration=%v, got %v", lease, err)
		}
	}

	// kinds contains invalid kind
	if _, err := uc.Claim(context.Background(), orgID, []BackgroundTaskKind{BackgroundTaskKindMasterDataImport, BackgroundTaskKind("INVALID")}, time.Minute); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for invalid kind in kinds, got %v", err)
	}

	// Success with valid kinds
	kinds := []BackgroundTaskKind{BackgroundTaskKindMasterDataImport, BackgroundTaskKindOrderReminder}
	task, err := uc.Claim(context.Background(), orgID, kinds, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatalf("expected non-nil task returned")
	}
	if repo.claimOrgID != orgID {
		t.Fatalf("expected repo claimOrgID %v, got %v", orgID, repo.claimOrgID)
	}
	if len(repo.claimKinds) != len(kinds) || repo.claimKinds[0] != kinds[0] || repo.claimKinds[1] != kinds[1] {
		t.Fatalf("expected repo claimKinds %v, got %v", kinds, repo.claimKinds)
	}
	if repo.claimLease != 30*time.Second {
		t.Fatalf("expected repo claimLease 30s, got %v", repo.claimLease)
	}
	if !repo.claimNow.Equal(fixedNow) {
		t.Fatalf("expected repo claimNow %v, got %v", fixedNow, repo.claimNow)
	}
}

func TestBackgroundTaskCompleteValidation(t *testing.T) {
	repo := &backgroundTaskRepoStub{}
	uc := NewBackgroundTaskUsecase(repo)
	orgID := uuid.New()
	id := uuid.New()
	validToken := "lease-token-123"

	// organizationID is nil
	if _, err := uc.Complete(context.Background(), uuid.Nil, id, validToken); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// id is nil
	if _, err := uc.Complete(context.Background(), orgID, uuid.Nil, validToken); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil id, got %v", err)
	}

	// leaseToken is empty or whitespace
	for _, token := range []string{"", "   ", "\t\n"} {
		if _, err := uc.Complete(context.Background(), orgID, id, token); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for empty leaseToken %q, got %v", token, err)
		}
	}

	// leaseToken > 128 runes
	if _, err := uc.Complete(context.Background(), orgID, id, strings.Repeat("t", 129)); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for leaseToken > 128 runes, got %v", err)
	}

	// Success
	task, err := uc.Complete(context.Background(), orgID, id, "  token-ok  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatalf("expected non-nil task returned")
	}
	if repo.completeOrgID != orgID {
		t.Fatalf("expected repo completeOrgID %v, got %v", orgID, repo.completeOrgID)
	}
	if repo.completeID != id {
		t.Fatalf("expected repo completeID %v, got %v", id, repo.completeID)
	}
	if repo.completeLeaseToken != "token-ok" {
		t.Fatalf("expected trimmed completeLeaseToken 'token-ok', got %q", repo.completeLeaseToken)
	}
}

func TestBackgroundTaskFailValidation(t *testing.T) {
	repo := &backgroundTaskRepoStub{}
	uc := NewBackgroundTaskUsecase(repo)
	orgID := uuid.New()
	id := uuid.New()
	validToken := "lease-token-123"
	validNextRunAt := time.Now().Add(5 * time.Minute)

	// organizationID is nil
	if _, err := uc.Fail(context.Background(), uuid.Nil, id, validToken, "err", validNextRunAt); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// id is nil
	if _, err := uc.Fail(context.Background(), orgID, uuid.Nil, validToken, "err", validNextRunAt); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil id, got %v", err)
	}

	// leaseToken is empty or whitespace
	for _, token := range []string{"", "   ", "\t\n"} {
		if _, err := uc.Fail(context.Background(), orgID, id, token, "err", validNextRunAt); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for empty leaseToken %q, got %v", token, err)
		}
	}

	// leaseToken > 128 runes
	if _, err := uc.Fail(context.Background(), orgID, id, strings.Repeat("t", 129), "err", validNextRunAt); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for leaseToken > 128 runes, got %v", err)
	}

	// errorMessage > 2000 runes
	if _, err := uc.Fail(context.Background(), orgID, id, validToken, strings.Repeat("e", 2001), validNextRunAt); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for errorMessage > 2000 runes, got %v", err)
	}

	// nextRunAt is zero
	if _, err := uc.Fail(context.Background(), orgID, id, validToken, "err", time.Time{}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for zero nextRunAt, got %v", err)
	}

	// Success
	task, err := uc.Fail(context.Background(), orgID, id, "  token-fail  ", "  failed reason  ", validNextRunAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatalf("expected non-nil task returned")
	}
	if repo.failOrgID != orgID {
		t.Fatalf("expected repo failOrgID %v, got %v", orgID, repo.failOrgID)
	}
	if repo.failID != id {
		t.Fatalf("expected repo failID %v, got %v", id, repo.failID)
	}
	if repo.failLeaseToken != "token-fail" {
		t.Fatalf("expected trimmed failLeaseToken 'token-fail', got %q", repo.failLeaseToken)
	}
	if repo.failErrorMessage != "failed reason" {
		t.Fatalf("expected trimmed failErrorMessage 'failed reason', got %q", repo.failErrorMessage)
	}
	if !repo.failNextRunAt.Equal(validNextRunAt) {
		t.Fatalf("expected repo failNextRunAt %v, got %v", validNextRunAt, repo.failNextRunAt)
	}
}

func TestBackgroundTaskGetValidation(t *testing.T) {
	repo := &backgroundTaskRepoStub{}
	uc := NewBackgroundTaskUsecase(repo)
	orgID := uuid.New()
	id := uuid.New()

	// organizationID is nil
	if _, err := uc.Get(context.Background(), uuid.Nil, id); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// id is nil
	if _, err := uc.Get(context.Background(), orgID, uuid.Nil); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil id, got %v", err)
	}

	// Success
	task, err := uc.Get(context.Background(), orgID, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task == nil {
		t.Fatalf("expected non-nil task returned")
	}
	if repo.getOrgID != orgID {
		t.Fatalf("expected repo getOrgID %v, got %v", orgID, repo.getOrgID)
	}
	if repo.getID != id {
		t.Fatalf("expected repo getID %v, got %v", id, repo.getID)
	}
}

func TestBackgroundTaskUsecaseList(t *testing.T) {
	repo := &backgroundTaskRepoStub{}
	uc := NewBackgroundTaskUsecase(repo)
	orgID := uuid.New()

	// organizationID is nil
	if _, err := uc.List(context.Background(), uuid.Nil, BackgroundTaskListOptions{Page: 1, PageSize: 20}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// invalid pagination (page < 1, pageSize < 1, pageSize > 200)
	for _, opt := range []BackgroundTaskListOptions{
		{Page: 0, PageSize: 20},
		{Page: -1, PageSize: 20},
		{Page: 1, PageSize: 0},
		{Page: 1, PageSize: -5},
		{Page: 1, PageSize: MaxListPageSize + 1},
	} {
		if _, err := uc.List(context.Background(), orgID, opt); err != ErrBackgroundTaskInvalidArgument {
			t.Fatalf("expected ErrBackgroundTaskInvalidArgument for invalid pagination %+v, got %v", opt, err)
		}
	}

	// invalid status enum
	invalidStatus := BackgroundTaskStatus("INVALID_STATUS")
	if _, err := uc.List(context.Background(), orgID, BackgroundTaskListOptions{Page: 1, PageSize: 20, Status: &invalidStatus}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for invalid status, got %v", err)
	}

	// invalid kind enum
	invalidKind := BackgroundTaskKind("INVALID_KIND")
	if _, err := uc.List(context.Background(), orgID, BackgroundTaskListOptions{Page: 1, PageSize: 20, Kind: &invalidKind}); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for invalid kind, got %v", err)
	}

	// success with valid options and pass-through assertion
	validStatus := BackgroundTaskStatusFailed
	validKind := BackgroundTaskKindMasterDataImport
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now
	opt := BackgroundTaskListOptions{
		Page:      2,
		PageSize:  50,
		Status:    &validStatus,
		Kind:      &validKind,
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	expectedItem := &BackgroundTask{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           validKind,
		Status:         validStatus,
	}
	repo.listResult = &BackgroundTaskList{
		Items:    []*BackgroundTask{expectedItem},
		Total:    1,
		Page:     2,
		PageSize: 50,
	}

	res, err := uc.List(context.Background(), orgID, opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Total != 1 || len(res.Items) != 1 || res.Page != 2 || res.PageSize != 50 {
		t.Fatalf("unexpected result: %#v", res)
	}
	if repo.listOrgID != orgID {
		t.Fatalf("expected repo.listOrgID %v, got %v", orgID, repo.listOrgID)
	}
	if repo.listOptions.Page != 2 || repo.listOptions.PageSize != 50 || *repo.listOptions.Status != validStatus || *repo.listOptions.Kind != validKind || !repo.listOptions.StartTime.Equal(startTime) || !repo.listOptions.EndTime.Equal(endTime) {
		t.Fatalf("expected repo.listOptions %+v, got %+v", opt, repo.listOptions)
	}
}

func TestBackgroundTaskUsecaseRequeue(t *testing.T) {
	fixedNow := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	repo := &backgroundTaskRepoStub{}
	uc := &BackgroundTaskUsecase{
		repo: repo,
		now:  func() time.Time { return fixedNow },
	}
	orgID := uuid.New()
	actorID := uuid.New()
	taskID := uuid.New()

	// organizationID is nil
	if _, err := uc.Requeue(context.Background(), uuid.Nil, actorID, taskID); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil orgID, got %v", err)
	}

	// actorID is nil
	if _, err := uc.Requeue(context.Background(), orgID, uuid.Nil, taskID); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil actorID, got %v", err)
	}

	// taskID is nil
	if _, err := uc.Requeue(context.Background(), orgID, actorID, uuid.Nil); err != ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for nil taskID, got %v", err)
	}

	// Success
	requeuedTask := &BackgroundTask{
		ID:             taskID,
		OrganizationID: orgID,
		Kind:           BackgroundTaskKindUnlocodeImport,
		Status:         BackgroundTaskStatusPending,
		Attempts:       0,
		MaxAttempts:    5,
		NextRunAt:      fixedNow,
	}
	repo.requeueTask = requeuedTask

	result, err := uc.Requeue(context.Background(), orgID, actorID, taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != requeuedTask {
		t.Fatalf("expected result %v, got %v", requeuedTask, result)
	}
	if repo.requeueOrgID != orgID {
		t.Fatalf("expected repo.requeueOrgID %v, got %v", orgID, repo.requeueOrgID)
	}
	if repo.requeueID != taskID {
		t.Fatalf("expected repo.requeueID %v, got %v", taskID, repo.requeueID)
	}
	if !repo.requeueNow.Equal(fixedNow) {
		t.Fatalf("expected repo.requeueNow %v, got %v", fixedNow, repo.requeueNow)
	}

	// Assert audit passed to repo
	if repo.requeueAudit == nil {
		t.Fatalf("expected non-nil requeueAudit passed to repo")
	}
	event := repo.requeueAudit
	if event.Action != "background_task.requeue" {
		t.Fatalf("expected audit Action 'background_task.requeue', got %s", event.Action)
	}
	if event.Result != "success" {
		t.Fatalf("expected audit Result 'success', got %s", event.Result)
	}
	if event.OrganizationID == nil || *event.OrganizationID != orgID {
		t.Fatalf("expected audit OrganizationID %v, got %v", orgID, event.OrganizationID)
	}
	if event.UserID == nil || *event.UserID != actorID {
		t.Fatalf("expected audit UserID %v, got %v", actorID, event.UserID)
	}
	if event.Details == nil {
		t.Fatalf("expected non-nil audit Details")
	}
	if event.Details["background_task.id"] != taskID.String() {
		t.Fatalf("expected audit Details['background_task.id'] == %q, got %q", taskID.String(), event.Details["background_task.id"])
	}
}
