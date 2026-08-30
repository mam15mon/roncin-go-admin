package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type objectDeletionRepoStub struct {
	deletion *ObjectDeletion
	err      error
}

func (s *objectDeletionRepoStub) FindByTaskID(context.Context, uuid.UUID) (*ObjectDeletion, error) {
	return s.deletion, s.err
}

type enterpriseImageStorageStub struct {
	enabled     bool
	deleteOrgID uuid.UUID
	deleteKey   string
	deleteErr   error
}

func (s *enterpriseImageStorageStub) Enabled() bool { return s.enabled }

func (s *enterpriseImageStorageStub) PrepareUpload(context.Context, uuid.UUID, string, string, int64, string) (*EnterpriseImageUpload, error) {
	return nil, nil
}

func (s *enterpriseImageStorageStub) VerifyUpload(context.Context, uuid.UUID, *EnterpriseResourceImage) error {
	return nil
}

func (s *enterpriseImageStorageStub) PresignGet(context.Context, uuid.UUID, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (s *enterpriseImageStorageStub) Delete(_ context.Context, organizationID uuid.UUID, objectKey string) error {
	s.deleteOrgID = organizationID
	s.deleteKey = objectKey
	return s.deleteErr
}

func TestObjectDeletionProcessNextCompletesTask(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	objectKey := "enterprise-resources/" + organizationID.String() + "/" + uuid.NewString() + ".jpg"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindObjectStorageDelete,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	storage := &enterpriseImageStorageStub{enabled: true}
	usecase := NewObjectDeletionUsecase(NewBackgroundTaskUsecase(taskRepo), &objectDeletionRepoStub{
		deletion: &ObjectDeletion{ObjectKey: objectKey},
	}, storage)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if storage.deleteOrgID != organizationID || storage.deleteKey != objectKey {
		t.Fatalf("对象删除参数错误: organizationID=%s objectKey=%q", storage.deleteOrgID, storage.deleteKey)
	}
	if taskRepo.completeOrgID != organizationID || taskRepo.completeID != taskID || taskRepo.completeLeaseToken != leaseToken {
		t.Fatalf("任务完成参数错误: %#v", taskRepo)
	}
	if taskRepo.failID != uuid.Nil {
		t.Fatalf("成功路径不应记录失败状态: %#v", taskRepo)
	}
}

func TestObjectDeletionProcessNextFailsOnStorageError(t *testing.T) {
	organizationID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: uuid.New(), OrganizationID: organizationID, Kind: BackgroundTaskKindObjectStorageDelete,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	storage := &enterpriseImageStorageStub{enabled: true, deleteErr: errors.New("对象存储不可用")}
	usecase := NewObjectDeletionUsecase(NewBackgroundTaskUsecase(taskRepo), &objectDeletionRepoStub{
		deletion: &ObjectDeletion{ObjectKey: "enterprise-resources/" + organizationID.String() + "/key.jpg"},
	}, storage)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err == nil {
		t.Fatal("对象删除失败时 ProcessNext 应返回错误")
	}
	if taskRepo.failOrgID != organizationID || taskRepo.failLeaseToken != leaseToken {
		t.Fatalf("失败状态记录参数错误: %#v", taskRepo)
	}
	if taskRepo.failErrorMessage == "" || taskRepo.failNextRunAt.IsZero() {
		t.Fatalf("失败状态应包含错误信息与重试时间: %#v", taskRepo)
	}
	if taskRepo.completeID != uuid.Nil {
		t.Fatalf("删除失败不应完成任务: %#v", taskRepo)
	}
}

func TestObjectDeletionProcessNextRejectsIncompletePayload(t *testing.T) {
	organizationID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: uuid.New(), OrganizationID: organizationID, Kind: BackgroundTaskKindObjectStorageDelete,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	storage := &enterpriseImageStorageStub{enabled: true}
	usecase := NewObjectDeletionUsecase(NewBackgroundTaskUsecase(taskRepo), &objectDeletionRepoStub{
		deletion: &ObjectDeletion{ObjectKey: "  "},
	}, storage)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err == nil {
		t.Fatal("对象键缺失时 ProcessNext 应返回错误")
	}
	if storage.deleteKey != "" {
		t.Fatalf("对象键缺失时不应调用对象存储: %#v", storage)
	}
	if taskRepo.failID == uuid.Nil {
		t.Fatalf("对象键缺失应记录失败状态: %#v", taskRepo)
	}
}
