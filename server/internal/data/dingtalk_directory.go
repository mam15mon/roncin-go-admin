package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	taskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	syncent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkdirectorysync"
	orgent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"reflect"
	"strings"
	"time"
)

type dingTalkDirectoryRepo struct{ data *Data }

func NewDingTalkDirectoryRepo(data *Data) biz.DingTalkDirectoryRepo {
	return &dingTalkDirectoryRepo{data: data}
}
func (r *dingTalkDirectoryRepo) Start(ctx context.Context, org, actor uuid.UUID, corp string) (*biz.DingTalkDirectorySync, error) {
	var task *ent.BackgroundTask
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.Organization.Query().Where(orgent.IDEQ(org), orgent.KindEQ(orgent.KindSystem), orgent.EnabledEQ(true)).ForShare().Only(ctx); err != nil {
			return mapEntError(err, biz.ErrPermissionDenied, nil)
		}
		var err error
		task, err = tx.BackgroundTask.Create().SetOrganizationID(org).SetKind(taskent.KindDINGTALK_DIRECTORY_SYNC).SetIdempotencyKey(uuid.NewString()).SetMaxAttempts(1).Save(ctx)
		if err != nil {
			return err
		}
		_, err = tx.DingTalkDirectorySync.Create().SetBackgroundTaskID(task.ID).SetActorID(actor).SetCorpID(corp).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrDingTalkDirectoryBusy)
		}
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "admin.dingtalk.directory.sync", Result: "success", Details: map[string]string{"background_task.id": task.ID.String()}})
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, task.ID, 1, 20)
}
func (r *dingTalkDirectoryRepo) Get(ctx context.Context, org, taskID uuid.UUID, page, size int) (*biz.DingTalkDirectorySync, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.DingTalkDirectorySync.Query().Where(syncent.HasBackgroundTaskWith(taskent.OrganizationIDEQ(org))).WithBackgroundTask()
	if taskID != uuid.Nil {
		query = query.Where(syncent.BackgroundTaskIDEQ(taskID))
	}
	record, err := query.Order(ent.Desc(syncent.FieldCreatedAt), ent.Desc(syncent.FieldID)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) && taskID == uuid.Nil {
			return nil, nil
		}
		return nil, mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
	}
	task := record.Edges.BackgroundTask
	result := &biz.DingTalkDirectorySync{TaskID: record.BackgroundTaskID, Status: biz.BackgroundTaskStatus(task.Status), CreatedAt: record.CreatedAt, CompletedAt: record.CompletedAt, CorpID: record.CorpID, ScopeSummary: record.ScopeSummary, FullScope: record.FullScope, AvailableFields: record.AvailableFields, TotalCount: record.TotalCount, CreatedCount: record.CreatedCount, UpdatedCount: record.UpdatedCount, UnchangedCount: record.UnchangedCount, FailedCount: record.FailedCount, FailureTotal: len(record.Failures), Page: page, PageSize: size, LastError: task.LastError}
	start := (page - 1) * size
	if start > len(record.Failures) {
		start = len(record.Failures)
	}
	end := start + size
	if end > len(record.Failures) {
		end = len(record.Failures)
	}
	for _, f := range record.Failures[start:end] {
		result.Failures = append(result.Failures, biz.DingTalkDirectoryFailure{UserID: f.UserID, UserName: f.UserName, DepartmentID: f.DepartmentID, DepartmentName: f.DepartmentName, Reason: f.Reason, Message: f.Message})
	}
	return result, nil
}
func lockDirectoryLease(ctx context.Context, tx *ent.Tx, task *biz.BackgroundTask) (*ent.BackgroundTask, error) {
	record, err := tx.BackgroundTask.Query().Where(taskent.IDEQ(task.ID), taskent.KindEQ(taskent.KindDINGTALK_DIRECTORY_SYNC)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
	}
	if record.Status != taskent.StatusRUNNING || record.LeaseToken == nil || task.LeaseToken == nil || *record.LeaseToken != *task.LeaseToken || record.LeaseExpiresAt == nil || !record.LeaseExpiresAt.After(time.Now()) {
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}
	return record, nil
}
func (r *dingTalkDirectoryRepo) ApplyPerson(ctx context.Context, task *biz.BackgroundTask, p *biz.DingTalkDirectoryPerson, fields []string) (string, error) {
	if p == nil || strings.TrimSpace(p.UserID) == "" || strings.TrimSpace(p.UnionID) == "" || strings.TrimSpace(p.Name) == "" {
		return "", biz.ErrDingTalkIdentityConflict
	}
	outcome := "unchanged"
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := lockDirectoryLease(ctx, tx, task); err != nil {
			return err
		}
		accounts, err := tx.User.Query().Where(userent.Or(userent.DingtalkUseridEQ(p.UserID), userent.DingtalkUnionidEQ(p.UnionID))).Order(userent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(accounts) > 1 {
			return biz.ErrDingTalkIdentityConflict
		}
		depts := make([]schema.DirectoryDepartment, 0, len(p.Departments))
		for _, d := range p.Departments {
			depts = append(depts, schema.DirectoryDepartment{ID: d.ID, Name: d.Name})
		}
		allowed := map[string]bool{}
		for _, field := range fields {
			allowed[field] = true
		}
		// 未确认授权的敏感字段清空，已授权空值由源字段显式决定。
		email, mobile, avatar := p.Email, p.Mobile, p.AvatarURL
		if !allowed["email"] || email != nil && *email == "" {
			email = nil
		}
		if !allowed["mobile"] {
			mobile = nil
		}
		if !allowed["avatar_url"] {
			avatar = nil
		}
		if len(accounts) == 0 {
			_, err := tx.User.Create().SetDisplayName(p.Name).SetDingtalkName(p.Name).SetDingtalkUnionid(p.UnionID).SetDingtalkUserid(p.UserID).SetEnabled(false).SetPendingAssignment(true).SetDingtalkDepartments(depts).SetDingtalkLastSyncedAt(time.Now().UTC()).SetNillableEmail(email).SetNillableDingtalkMobile(mobile).SetNillableAvatarURL(avatar).Save(ctx)
			outcome = "created"
			return mapEntError(err, nil, biz.ErrDingTalkIdentityConflict)
		}
		a := accounts[0]
		if (a.DingtalkUserid != nil && *a.DingtalkUserid != p.UserID) || (a.DingtalkUnionid != nil && *a.DingtalkUnionid != p.UnionID) {
			return biz.ErrDingTalkIdentityConflict
		}
		if a.DisplayName != p.Name || a.DingtalkName == nil || *a.DingtalkName != p.Name || !reflect.DeepEqual(a.DingtalkDepartments, depts) || !reflect.DeepEqual(a.Email, email) || !reflect.DeepEqual(a.DingtalkMobile, mobile) || !reflect.DeepEqual(a.AvatarURL, avatar) {
			outcome = "updated"
		}
		update := tx.User.UpdateOne(a).SetDisplayName(p.Name).SetDingtalkName(p.Name).SetDingtalkUnionid(p.UnionID).SetDingtalkUserid(p.UserID).SetDingtalkDepartments(depts).SetDingtalkLastSyncedAt(time.Now().UTC())
		if email == nil {
			update.ClearEmail()
		} else {
			update.SetEmail(*email)
		}
		if mobile == nil {
			update.ClearDingtalkMobile()
		} else {
			update.SetDingtalkMobile(*mobile)
		}
		if avatar == nil {
			update.ClearAvatarURL()
		} else {
			update.SetAvatarURL(*avatar)
		}
		_, err = update.Save(ctx)
		return mapEntError(err, nil, biz.ErrDingTalkIdentityConflict)
	})
	return outcome, err
}
func (r *dingTalkDirectoryRepo) Finish(ctx context.Context, task *biz.BackgroundTask, result *biz.DingTalkDirectorySync, message string) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		record, err := lockDirectoryLease(ctx, tx, task)
		if err != nil {
			return err
		}
		failures := make([]schema.DirectoryFailure, 0, len(result.Failures))
		for _, f := range result.Failures {
			failures = append(failures, schema.DirectoryFailure{UserID: f.UserID, UserName: f.UserName, DepartmentID: f.DepartmentID, DepartmentName: f.DepartmentName, Reason: f.Reason, Message: f.Message})
		}
		_, err = tx.DingTalkDirectorySync.Update().Where(syncent.BackgroundTaskIDEQ(task.ID)).SetActive(false).SetScopeSummary(result.ScopeSummary).SetFullScope(result.FullScope).SetAvailableFields(result.AvailableFields).SetTotalCount(result.TotalCount).SetCreatedCount(result.CreatedCount).SetUpdatedCount(result.UpdatedCount).SetUnchangedCount(result.UnchangedCount).SetFailedCount(result.FailedCount).SetFailures(failures).SetCompletedAt(time.Now().UTC()).Save(ctx)
		if err != nil {
			return err
		}
		update := tx.BackgroundTask.UpdateOne(record).ClearLeaseToken().ClearLeaseExpiresAt()
		if message != "" {
			update.SetStatus(taskent.StatusDEAD_LETTER).SetLastError(message).SetAttempts(record.MaxAttempts)
		} else {
			update.SetStatus(taskent.StatusSUCCEEDED).ClearLastError()
		}
		_, err = update.Save(ctx)
		return err
	})
}
