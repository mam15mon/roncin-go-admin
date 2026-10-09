package biz

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"sort"
	"strings"
	"time"
)

var (
	ErrDingTalkDirectoryBusy      = errors.Conflict("DINGTALK_DIRECTORY_BUSY", "本企业已有通讯录同步任务正在执行")
	ErrDingTalkDirectoryFailed    = errors.BadRequest("DINGTALK_DIRECTORY_FAILED", "通讯录读取失败，请检查钉钉应用通讯录权限")
	ErrDingTalkIdentityConflict   = errors.Conflict("DINGTALK_IDENTITY_CONFLICT", "钉钉身份与现有账号冲突，请管理员核对")
	ErrDingTalkAssignmentConflict = errors.Conflict("DINGTALK_ASSIGNMENT_CONFLICT", "人员已被开通或状态已变更，请刷新后重试")
	ErrDingTalkProfileReadOnly    = errors.BadRequest("DINGTALK_PROFILE_READ_ONLY", "钉钉基础资料请在钉钉修改后同步")
)

type DingTalkDepartment struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type DingTalkDirectoryPerson struct {
	UserID, UnionID, Name    string
	Email, AvatarURL, Mobile *string
	Departments              []DingTalkDepartment
}
type DingTalkDirectoryFailure struct {
	UserID       string `json:"userId"`
	DepartmentID int64  `json:"departmentId"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
}
type DingTalkDirectorySnapshot struct {
	CorpID          string
	ScopeSummary    string
	FullScope       bool
	AvailableFields []string
	People          []DingTalkDirectoryPerson
	Failures        []DingTalkDirectoryFailure
}
type DingTalkDirectorySync struct {
	TaskID                                                              uuid.UUID
	Status                                                              BackgroundTaskStatus
	CreatedAt                                                           time.Time
	CompletedAt                                                         *time.Time
	CorpID, ScopeSummary                                                string
	FullScope                                                           bool
	AvailableFields                                                     []string
	TotalCount, CreatedCount, UpdatedCount, UnchangedCount, FailedCount int
	Failures                                                            []DingTalkDirectoryFailure
	FailureTotal                                                        int
	Page, PageSize                                                      int
	LastError                                                           *string
}
type DingTalkDirectoryGateway interface {
	Enabled() bool
	EnterpriseID() string
	ReadDirectory(context.Context) (*DingTalkDirectorySnapshot, error)
}
type DingTalkDirectoryRepo interface {
	Start(context.Context, uuid.UUID, uuid.UUID, string) (*DingTalkDirectorySync, error)
	Get(context.Context, uuid.UUID, uuid.UUID, int, int) (*DingTalkDirectorySync, error)
	ApplyPerson(context.Context, *BackgroundTask, *DingTalkDirectoryPerson, []string) (string, error)
	Finish(context.Context, *BackgroundTask, *DingTalkDirectorySync, string) error
}
type DingTalkDirectoryUsecase struct {
	repo    DingTalkDirectoryRepo
	gateway DingTalkDirectoryGateway
	tasks   *BackgroundTaskUsecase
}

func NewDingTalkDirectoryUsecase(repo DingTalkDirectoryRepo, gateway DingTalkDirectoryGateway, tasks *BackgroundTaskUsecase) *DingTalkDirectoryUsecase {
	return &DingTalkDirectoryUsecase{repo: repo, gateway: gateway, tasks: tasks}
}
func (uc *DingTalkDirectoryUsecase) Enabled() bool { return uc.gateway.Enabled() }
func (uc *DingTalkDirectoryUsecase) Start(ctx context.Context, org, actor uuid.UUID) (*DingTalkDirectorySync, error) {
	if !IsSystemWorkspace(ctx) {
		return nil, ErrPermissionDenied
	}
	if !uc.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	if org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.Start(ctx, org, actor, uc.gateway.EnterpriseID())
}

// taskID 为空时读取最近一项同步，用于刷新页面后恢复运行状态。
func (uc *DingTalkDirectoryUsecase) Get(ctx context.Context, org, taskID uuid.UUID, page, pageSize int) (*DingTalkDirectorySync, error) {
	if !IsSystemWorkspace(ctx) {
		return nil, ErrPermissionDenied
	}
	if org == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.Get(ctx, org, taskID, page, pageSize)
}
func (uc *DingTalkDirectoryUsecase) ProcessNext(ctx context.Context, lease time.Duration) error {
	task, err := uc.tasks.ClaimAny(ctx, []BackgroundTaskKind{BackgroundTaskKindDingTalkDirectorySync}, lease)
	if err != nil {
		return err
	}
	// 外部请求与人员逐条事务均在租约期限内结束；完成记录使用单独有界上下文。
	remaining := time.Until(*task.LeaseExpiresAt) - 5*time.Second
	runCtx, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()
	result := &DingTalkDirectorySync{TaskID: task.ID}
	snapshot, err := uc.gateway.ReadDirectory(runCtx)
	finishError := ""
	if err != nil {
		finishError = "通讯录读取失败，请检查应用权限或稍后重新同步"
	} else {
		result.CorpID = snapshot.CorpID
		result.ScopeSummary = snapshot.ScopeSummary
		result.FullScope = snapshot.FullScope
		result.AvailableFields = snapshot.AvailableFields
		result.Failures = append(result.Failures, snapshot.Failures...)
		byID := map[string]*DingTalkDirectoryPerson{}
		conflicted := map[string]bool{}
		for _, person := range snapshot.People {
			p := person
			p.UserID = strings.TrimSpace(p.UserID)
			p.UnionID = strings.TrimSpace(p.UnionID)
			if conflicted[p.UserID] {
				continue
			}
			if existing, ok := byID[p.UserID]; ok {
				if existing.UnionID != p.UnionID {
					result.Failures = append(result.Failures, DingTalkDirectoryFailure{UserID: p.UserID, Reason: "DINGTALK_IDENTITY_CONFLICT", Message: "同一人员身份资料不一致"})
					delete(byID, p.UserID)
					conflicted[p.UserID] = true
					continue
				}
				existing.Departments = mergeDingTalkDepartments(existing.Departments, p.Departments)
				continue
			}
			byID[p.UserID] = &p
		}
		ids := make([]string, 0, len(byID))
		for id := range byID {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		result.TotalCount = len(ids) + len(conflicted)
		for _, id := range ids {
			if runCtx.Err() != nil {
				finishError = "同步执行超时，请重新同步"
				break
			}
			outcome, applyErr := uc.repo.ApplyPerson(runCtx, task, byID[id], snapshot.AvailableFields)
			if applyErr != nil {
				reason, message := "DINGTALK_PERSON_SYNC_FAILED", "人员同步失败，请稍后重试"
				if errors.FromError(applyErr).Reason == ErrDingTalkIdentityConflict.Reason {
					reason, message = ErrDingTalkIdentityConflict.Reason, ErrDingTalkIdentityConflict.Message
				}
				result.Failures = append(result.Failures, DingTalkDirectoryFailure{UserID: id, Reason: reason, Message: message})
				continue
			}
			switch outcome {
			case "created":
				result.CreatedCount++
			case "updated":
				result.UpdatedCount++
			default:
				result.UnchangedCount++
			}
		}
	}
	result.FailedCount = len(result.Failures)
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer finishCancel()
	return uc.repo.Finish(finishCtx, task, result, finishError)
}
func mergeDingTalkDepartments(a, b []DingTalkDepartment) []DingTalkDepartment {
	byID := map[int64]DingTalkDepartment{}
	for _, d := range append(a, b...) {
		byID[d.ID] = d
	}
	result := make([]DingTalkDepartment, 0, len(byID))
	for _, d := range byID {
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

type DingTalkAssignmentResult struct {
	UserID          uuid.UUID
	Success         bool
	Reason, Message string
}

func (uc *AdminUsecase) BatchAuthorizeDingTalkUsers(ctx context.Context, sourceOrg, targetOrg, actor uuid.UUID, userIDs, roleIDs []uuid.UUID) ([]*DingTalkAssignmentResult, error) {
	if !IsSystemWorkspace(ctx) {
		return nil, ErrPermissionDenied
	}
	if sourceOrg == uuid.Nil || targetOrg == uuid.Nil || actor == uuid.Nil || len(userIDs) == 0 || len(userIDs) > MaxListPageSize || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	for _, id := range userIDs {
		if id == uuid.Nil {
			return nil, ErrAdminInvalidArgument
		}
	}
	seen := map[uuid.UUID]bool{}
	result := make([]*DingTalkAssignmentResult, 0, len(userIDs))
	for _, id := range userIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		_, err := uc.AuthorizeDingTalkUser(ctx, sourceOrg, targetOrg, actor, &AdminUser{ID: id}, roleIDs)
		item := &DingTalkAssignmentResult{UserID: id, Success: err == nil, Message: "开通成功"}
		if err != nil {
			e := errors.FromError(err)
			item.Reason = e.Reason
			item.Message = e.Message
			if e.Code >= 500 {
				item.Reason = "DINGTALK_ASSIGNMENT_FAILED"
				item.Message = "开通失败，请稍后重试"
			}
		}
		result = append(result, item)
	}
	return result, nil
}
