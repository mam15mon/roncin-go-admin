package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	resourceimageent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceimage"
	objectdeletionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/objectstoragedeletion"
)

type enterpriseImageDeletionFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	resourceID     uuid.UUID
	actorID        uuid.UUID
	objectKey      string
	suffix         string
}

type invalidAuditResultEnterpriseResourceRepo struct {
	biz.EnterpriseResourceRepo
}

func (r *invalidAuditResultEnterpriseResourceRepo) Delete(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	audit.Result = "invalid"
	return r.EnterpriseResourceRepo.Delete(ctx, organizationID, id, audit)
}

type deletionIntegrationStorage struct {
	deleteOrgID uuid.UUID
	deleteKey   string
}

func (s *deletionIntegrationStorage) Enabled() bool { return true }

func (s *deletionIntegrationStorage) PrepareUpload(context.Context, uuid.UUID, string, string, int64, string) (*biz.EnterpriseImageUpload, error) {
	return nil, nil
}

func (s *deletionIntegrationStorage) VerifyUpload(context.Context, uuid.UUID, *biz.EnterpriseResourceImage) error {
	return nil
}

func (s *deletionIntegrationStorage) PresignGet(context.Context, uuid.UUID, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (s *deletionIntegrationStorage) Delete(_ context.Context, organizationID uuid.UUID, objectKey string) error {
	s.deleteOrgID = organizationID
	s.deleteKey = objectKey
	return nil
}

func TestEnterpriseImageDeleteRegistersDeletionTaskPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             source,
		AutoMigrate:        true,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	t.Run("删除图片资源同事务登记删除任务并提交后执行", func(t *testing.T) {
		fixture := newEnterpriseImageDeletionFixture(t, data)
		repo := NewEnterpriseResourceRepo(data)

		if err := repo.Delete(context.Background(), fixture.organizationID, fixture.resourceID, fixture.deletionAudit()); err != nil {
			t.Fatalf("删除图片资源: %v", err)
		}
		fixture.requireResourceDeleted()
		task := fixture.requireDeletionTask(biz.BackgroundTaskStatusPending)

		storage := &deletionIntegrationStorage{}
		usecase := biz.NewObjectDeletionUsecase(biz.NewBackgroundTaskUsecase(NewBackgroundTaskRepo(data)), NewObjectDeletionRepo(data), storage)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := usecase.ProcessNext(ctx, 30*time.Second); err != nil {
			t.Fatalf("对象删除 Worker 执行任务: %v", err)
		}
		if storage.deleteOrgID != fixture.organizationID || storage.deleteKey != fixture.objectKey {
			t.Fatalf("Worker 删除参数错误: organizationID=%s objectKey=%q", storage.deleteOrgID, storage.deleteKey)
		}
		if refreshed, err := data.db.BackgroundTask.Get(context.Background(), task.ID); err != nil || refreshed.Status != backgroundtaskent.StatusSUCCEEDED {
			t.Fatalf("任务完成状态 = %#v，期望 SUCCEEDED，error=%v", refreshed, err)
		}
		payloadCount, err := data.db.ObjectStorageDeletion.Query().Where(objectdeletionent.BackgroundTaskIDEQ(task.ID)).Count(context.Background())
		if err != nil || payloadCount != 1 {
			t.Fatalf("任务完成后明细应保留: count=%d，期望 1，error=%v", payloadCount, err)
		}
	})

	t.Run("审计失败回滚资源删除与删除任务", func(t *testing.T) {
		fixture := newEnterpriseImageDeletionFixture(t, data)
		repo := &invalidAuditResultEnterpriseResourceRepo{EnterpriseResourceRepo: NewEnterpriseResourceRepo(data)}

		if err := repo.Delete(context.Background(), fixture.organizationID, fixture.resourceID, fixture.deletionAudit()); err == nil {
			t.Fatal("审计结果非法时删除资源未失败")
		}
		fixture.requireRolledBackState()
	})
}

func newEnterpriseImageDeletionFixture(t *testing.T, data *Data) *enterpriseImageDeletionFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("IMG-DEL-" + suffix).
		SetName("图片删除集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	actor, err := data.db.User.Create().
		SetDisplayName("图片删除测试用户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户: %v", err)
	}
	objectKey := "enterprise-resources/" + organization.ID.String() + "/" + uuid.NewString() + ".jpg"
	resource, err := data.db.EnterpriseResource.Create().
		SetOrganizationID(organization.ID).
		SetResourceType(resourceent.ResourceTypeIMAGE).
		SetShortName("图片删除测试资源-" + suffix).
		SetCreatedBy(actor.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试图片资源: %v", err)
	}
	if _, err = data.db.EnterpriseResourceImage.Create().
		SetResourceID(resource.ID).
		SetFileName("fixture.jpg").
		SetMimeType("image/jpeg").
		SetFileSize(1024).
		SetObjectKey(objectKey).
		SetChecksum(strings.Repeat("a", 64)).
		SetUploadedBy(actor.ID).
		Save(ctx); err != nil {
		t.Fatalf("创建测试图片元数据: %v", err)
	}
	fixture := &enterpriseImageDeletionFixture{
		t: t, data: data, organizationID: organization.ID, resourceID: resource.ID,
		actorID: actor.ID, objectKey: objectKey, suffix: suffix,
	}
	t.Cleanup(fixture.cleanup)
	return fixture
}

func (f *enterpriseImageDeletionFixture) deletionAudit() *biz.AuditEvent {
	return &biz.AuditEvent{
		OrganizationID: &f.organizationID,
		UserID:         &f.actorID,
		Action:         "enterprise_resource.delete",
		Result:         "success",
		Details:        map[string]string{"resource.type": "IMAGE"},
	}
}

func (f *enterpriseImageDeletionFixture) requireResourceDeleted() {
	f.t.Helper()
	ctx := context.Background()
	resourceCount, err := f.data.db.EnterpriseResource.Query().Where(resourceent.IDEQ(f.resourceID)).Count(ctx)
	if err != nil || resourceCount != 0 {
		f.t.Fatalf("已删除资源数 = %d，期望 0，error=%v", resourceCount, err)
	}
	imageCount, err := f.data.db.EnterpriseResourceImage.Query().Count(ctx)
	if err != nil || imageCount != 0 {
		f.t.Fatalf("已删除图片元数据数 = %d，期望 0，error=%v", imageCount, err)
	}
}

func (f *enterpriseImageDeletionFixture) requireDeletionTask(status biz.BackgroundTaskStatus) *biz.BackgroundTask {
	f.t.Helper()
	ctx := context.Background()
	tasks, err := f.data.db.BackgroundTask.Query().Where(backgroundtaskent.OrganizationIDEQ(f.organizationID)).All(ctx)
	if err != nil || len(tasks) != 1 {
		f.t.Fatalf("删除任务数 = %d，期望 1，error=%v", len(tasks), err)
	}
	task := tasks[0]
	if task.Kind != backgroundtaskent.KindOBJECT_STORAGE_DELETION {
		f.t.Fatalf("任务类型 = %s，期望 OBJECT_STORAGE_DELETION", task.Kind)
	}
	if task.IdempotencyKey != f.objectKey {
		f.t.Fatalf("任务幂等键 = %q，期望对象键 %q", task.IdempotencyKey, f.objectKey)
	}
	if task.Status != backgroundtaskent.Status(status) {
		f.t.Fatalf("任务状态 = %s，期望 %s", task.Status, status)
	}
	if task.MaxAttempts != 5 {
		f.t.Fatalf("任务最大尝试次数 = %d，期望 5", task.MaxAttempts)
	}
	payload, err := f.data.db.ObjectStorageDeletion.Query().Where(objectdeletionent.BackgroundTaskIDEQ(task.ID)).Only(ctx)
	if err != nil || payload.ObjectKey != f.objectKey {
		f.t.Fatalf("任务明细 = %#v，期望对象键 %q，error=%v", payload, f.objectKey, err)
	}
	return &biz.BackgroundTask{ID: task.ID, OrganizationID: task.OrganizationID, Kind: biz.BackgroundTaskKindObjectStorageDelete}
}

func (f *enterpriseImageDeletionFixture) requireRolledBackState() {
	f.t.Helper()
	ctx := context.Background()
	resourceCount, err := f.data.db.EnterpriseResource.Query().Where(resourceent.IDEQ(f.resourceID)).Count(ctx)
	if err != nil || resourceCount != 1 {
		f.t.Fatalf("回滚后资源数 = %d，期望 1，error=%v", resourceCount, err)
	}
	taskCount, err := f.data.db.BackgroundTask.Query().Where(backgroundtaskent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || taskCount != 0 {
		f.t.Fatalf("回滚后删除任务数 = %d，期望 0，error=%v", taskCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, err)
	}
}

func (f *enterpriseImageDeletionFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "对象删除任务明细", run: func() error {
			_, err := f.data.db.ObjectStorageDeletion.Delete().Where(objectdeletionent.HasBackgroundTaskWith(backgroundtaskent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "后台任务", run: func() error {
			_, err := f.data.db.BackgroundTask.Delete().Where(backgroundtaskent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "审计日志", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "图片元数据", run: func() error {
			_, err := f.data.db.EnterpriseResourceImage.Delete().Where(resourceimageent.HasResourceWith(resourceent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "企业资源", run: func() error {
			_, err := f.data.db.EnterpriseResource.Delete().Where(resourceent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "用户", run: func() error {
			return f.data.db.User.DeleteOneID(f.actorID).Exec(ctx)
		}},
		{name: "组织", run: func() error {
			return f.data.db.Organization.DeleteOneID(f.organizationID).Exec(ctx)
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理%s失败: %v", step.name, err)
			return
		}
	}
}
