package service

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *AdminService) BatchAuthorizeDingTalkUsers(ctx context.Context, request *v1.BatchAuthorizeDingTalkUsersRequest) (*v1.BatchAuthorizeDingTalkUsersResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	target, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	// 限制原始载荷规模，不能通过重复 ID 绕过批次上限。
	if len(request.GetUserIds()) == 0 || len(request.GetUserIds()) > biz.MaxListPageSize {
		return nil, biz.ErrAdminInvalidArgument
	}
	users, err := parseUniqueUUIDValues(request.GetUserIds(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	roles, err := parseUniqueUUIDValues(request.GetRoleIds(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	results, err := s.usecase.BatchAuthorizeDingTalkUsers(ctx, principal.Organization.ID, target, principal.UserID, users, roles)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.DingTalkAssignmentResult, 0, len(results))
	for _, result := range results {
		data = append(data, &v1.DingTalkAssignmentResult{UserId: result.UserID.String(), Success: result.Success, Reason: result.Reason, Message: result.Message})
	}
	return ok(ctx, &v1.BatchAuthorizeDingTalkUsersResponse{Data: data}), nil
}

func (s *AdminService) StartDingTalkDirectorySync(ctx context.Context, _ *v1.StartDingTalkDirectorySyncRequest) (*v1.StartDingTalkDirectorySyncResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.dingTalkDirectory.Start(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.StartDingTalkDirectorySyncResponse{Data: dingTalkDirectorySyncToAPI(result)}), nil
}

func (s *AdminService) GetLatestDingTalkDirectorySync(ctx context.Context, request *v1.GetLatestDingTalkDirectorySyncRequest) (*v1.GetLatestDingTalkDirectorySyncResponse, error) {
	result, err := s.getDingTalkDirectorySync(ctx, uuid.Nil, request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetLatestDingTalkDirectorySyncResponse{Data: dingTalkDirectorySyncToAPI(result)}), nil
}

func (s *AdminService) GetDingTalkDirectorySync(ctx context.Context, request *v1.GetDingTalkDirectorySyncRequest) (*v1.GetDingTalkDirectorySyncResponse, error) {
	taskID, err := uuid.Parse(request.GetTaskId())
	if err != nil || taskID == uuid.Nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	result, err := s.getDingTalkDirectorySync(ctx, taskID, request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetDingTalkDirectorySyncResponse{Data: dingTalkDirectorySyncToAPI(result)}), nil
}

func (s *AdminService) getDingTalkDirectorySync(ctx context.Context, taskID uuid.UUID, rawPage, rawPageSize int32) (*biz.DingTalkDirectorySync, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(rawPage, rawPageSize, biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	return s.dingTalkDirectory.Get(ctx, principal.Organization.ID, taskID, page, pageSize)
}

func dingTalkDirectorySyncToAPI(value *biz.DingTalkDirectorySync) *v1.DingTalkDirectorySync {
	if value == nil {
		return nil
	}
	result := &v1.DingTalkDirectorySync{TaskId: value.TaskID.String(), Status: string(value.Status), StartedAt: value.CreatedAt.Format(time.RFC3339), Total: int32(value.TotalCount), Created: int32(value.CreatedCount), Updated: int32(value.UpdatedCount), Unchanged: int32(value.UnchangedCount), Failed: int32(value.FailedCount), ScopeSummary: value.ScopeSummary, FullScope: value.FullScope, AvailableFields: value.AvailableFields, FailureTotal: int32(value.FailureTotal), Page: int32(value.Page), PageSize: int32(value.PageSize), LastError: value.LastError}
	if value.CompletedAt != nil {
		completed := value.CompletedAt.Format(time.RFC3339)
		result.CompletedAt = &completed
	}
	for _, failure := range value.Failures {
		result.Failures = append(result.Failures, &v1.DingTalkDirectorySyncFailure{UserId: failure.UserID, DepartmentId: strconv.FormatInt(failure.DepartmentID, 10), Reason: failure.Reason, Message: failure.Message})
	}
	return result
}
