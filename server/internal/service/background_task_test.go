package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	taskv1 "github.com/roncin/roncin-go-admin/server/api/task/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestBackgroundTaskKindRoundTrip(t *testing.T) {
	kinds := []biz.BackgroundTaskKind{
		biz.BackgroundTaskKindMasterDataImport,
		biz.BackgroundTaskKindUnlocodeImport,
		biz.BackgroundTaskKindOrderReminder,
		biz.BackgroundTaskKindIntegration,
		biz.BackgroundTaskKindDingTalkNotice,
	}
	for _, kind := range kinds {
		apiKind, err := backgroundTaskKindFromAPI(backgroundTaskKindToAPI(kind))
		if err != nil {
			t.Fatalf("unexpected error for kind %s: %v", kind, err)
		}
		if apiKind != kind {
			t.Fatalf("expected kind %s after round trip, got %s", kind, apiKind)
		}
	}
	if _, err := backgroundTaskKindFromAPI(taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_UNSPECIFIED); err != biz.ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for unspecified kind, got %v", err)
	}
}

func TestBackgroundTaskStatusRoundTrip(t *testing.T) {
	statuses := []biz.BackgroundTaskStatus{
		biz.BackgroundTaskStatusPending,
		biz.BackgroundTaskStatusRunning,
		biz.BackgroundTaskStatusSucceeded,
		biz.BackgroundTaskStatusFailed,
		biz.BackgroundTaskStatusDeadLetter,
	}
	for _, status := range statuses {
		apiStatus, err := backgroundTaskStatusFromAPI(backgroundTaskStatusToAPI(status))
		if err != nil {
			t.Fatalf("unexpected error for status %s: %v", status, err)
		}
		if apiStatus != status {
			t.Fatalf("expected status %s after round trip, got %s", status, apiStatus)
		}
	}
	if _, err := backgroundTaskStatusFromAPI(taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_UNSPECIFIED); err != biz.ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected ErrBackgroundTaskInvalidArgument for unspecified status, got %v", err)
	}
}

func TestBackgroundTaskToAPIOmitsLeaseFields(t *testing.T) {
	nextRunAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC)
	lastError := "import failed"
	value := &biz.BackgroundTask{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Kind:           biz.BackgroundTaskKindMasterDataImport,
		IdempotencyKey: "import-key-001",
		Status:         biz.BackgroundTaskStatusDeadLetter,
		Attempts:       3,
		MaxAttempts:    3,
		NextRunAt:      nextRunAt,
		LeaseToken:     &lastError,
		LastError:      &lastError,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	apiTask := backgroundTaskToAPI(value)
	if apiTask.GetKind() != taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT {
		t.Fatalf("expected master data import kind, got %v", apiTask.GetKind())
	}
	if apiTask.GetStatus() != taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_DEAD_LETTER {
		t.Fatalf("expected dead letter status, got %v", apiTask.GetStatus())
	}
	if apiTask.GetAttempts() != 3 || apiTask.GetMaxAttempts() != 3 {
		t.Fatalf("expected attempts 3/3, got %d/%d", apiTask.GetAttempts(), apiTask.GetMaxAttempts())
	}
	if apiTask.GetLastError() != lastError {
		t.Fatalf("expected last error %q, got %q", lastError, apiTask.GetLastError())
	}
	if apiTask.GetNextRunAt() != nextRunAt.Format(time.RFC3339) {
		t.Fatalf("expected next run at %s, got %s", nextRunAt.Format(time.RFC3339), apiTask.GetNextRunAt())
	}
}

func TestBackgroundTaskPageValues(t *testing.T) {
	page, pageSize, err := backgroundTaskPageValues(0, 0)
	if err != nil || page != 1 || pageSize != 20 {
		t.Fatalf("expected defaults 1/20, got %d/%d err %v", page, pageSize, err)
	}
	page, pageSize, err = backgroundTaskPageValues(2, 50)
	if err != nil || page != 2 || pageSize != 50 {
		t.Fatalf("expected 2/50, got %d/%d err %v", page, pageSize, err)
	}
	if _, _, err := backgroundTaskPageValues(-1, 20); err != biz.ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected invalid argument for negative page, got %v", err)
	}
	page, pageSize, err = backgroundTaskPageValues(1, biz.MaxListPageSize)
	if err != nil || page != 1 || pageSize != biz.MaxListPageSize {
		t.Fatalf("expected maximum page size, got %d/%d err %v", page, pageSize, err)
	}
	if _, _, err := backgroundTaskPageValues(1, biz.MaxListPageSize+1); err != biz.ErrBackgroundTaskInvalidArgument {
		t.Fatalf("expected invalid argument for oversized page size, got %v", err)
	}
}
