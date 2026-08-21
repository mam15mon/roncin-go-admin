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

var _ BackgroundTaskRepo = (*backgroundTaskRepoStub)(nil)

func TestBackgroundTaskKindValid(t *testing.T) {
	validKinds := []BackgroundTaskKind{
		BackgroundTaskKindMasterDataImport,
		BackgroundTaskKindUnlocodeImport,
		BackgroundTaskKindOrderReminder,
		BackgroundTaskKindIntegration,
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
