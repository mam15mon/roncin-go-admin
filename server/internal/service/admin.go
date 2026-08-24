package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

type AdminService struct {
	v1.UnimplementedAdminServiceServer
	usecase *biz.AdminUsecase
}

func NewAdminService(usecase *biz.AdminUsecase) *AdminService {
	return &AdminService{usecase: usecase}
}

func (s *AdminService) ListOrganizations(ctx context.Context, _ *v1.ListOrganizationsRequest) (*v1.ListOrganizationsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.usecase.ListOrganizations(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminOrganization, 0, len(items))
	for _, item := range items {
		data = append(data, organizationToAPI(item))
	}
	return &v1.ListOrganizationsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) CreateOrganization(ctx context.Context, request *v1.CreateOrganizationRequest) (*v1.CreateOrganizationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.GetParentId() == "" {
		return nil, biz.ErrAdminOrganizationParentRequired
	}
	parentID, err := uuid.Parse(request.GetParentId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	created, err := s.usecase.CreateOrganization(ctx, principal.UserID, &biz.AdminOrganization{Code: request.GetCode(), Name: request.GetName(), Kind: organizationKindFromAPI(request.GetKind()), ParentID: &parentID, Enabled: true, BaseCurrency: request.GetBaseCurrency()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateOrganizationResponse{Success: true, Code: 0, Message: "OK", Data: organizationToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) UpdateOrganization(ctx context.Context, request *v1.UpdateOrganizationRequest) (*v1.UpdateOrganizationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	organizationID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	updated, err := s.usecase.UpdateOrganization(ctx, principal.UserID, principal.Organization.ID, organizationID, request.GetName(), request.GetEnabled(), request.GetBaseCurrency())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateOrganizationResponse{Success: true, Code: 0, Message: "OK", Data: organizationToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ListUsers(ctx context.Context, request *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := adminPageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	list, err := s.usecase.ListUsers(ctx, principal.Organization.ID, biz.AdminUserListOptions{Page: page, PageSize: pageSize, Keyword: request.GetKeyword()})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminUser, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, userToAPI(item))
	}
	return &v1.ListUsersResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) CreateUser(ctx context.Context, request *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := parseUUIDs(request.GetRoleIds())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	created, err := s.usecase.CreateUser(ctx, principal.Organization.ID, principal.UserID, &biz.AdminUser{Username: request.GetUsername(), DisplayName: request.GetDisplayName(), Email: optionalString(request.GetEmail(), request.Email != nil), Enabled: true}, request.GetPassword(), roles)
	if err != nil {
		return nil, err
	}
	return &v1.CreateUserResponse{Success: true, Code: 0, Message: "OK", Data: userToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, request *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	roles, err := parseUUIDs(request.GetRoleIds())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	updated, err := s.usecase.UpdateUser(ctx, principal.Organization.ID, principal.UserID, userID, &biz.AdminUser{ID: userID, DisplayName: request.GetDisplayName(), Email: optionalString(request.GetEmail(), request.Email != nil), Enabled: request.GetEnabled()}, roles)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateUserResponse{Success: true, Code: 0, Message: "OK", Data: userToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) AuthorizeWeComUser(ctx context.Context, request *v1.AuthorizeWeComUserRequest) (*v1.AuthorizeWeComUserResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	targetOrganizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	roles, err := parseUUIDs(request.GetRoleIds())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	authorized, err := s.usecase.AuthorizeWeComUser(ctx, principal.Organization.ID, targetOrganizationID, principal.UserID, &biz.AdminUser{ID: userID, DisplayName: request.GetDisplayName(), Email: optionalString(request.GetEmail(), request.Email != nil)}, roles)
	if err != nil {
		return nil, err
	}
	return &v1.AuthorizeWeComUserResponse{Success: true, Code: 0, Message: "OK", Data: userToAPI(authorized), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) AuthorizeDingTalkUser(ctx context.Context, request *v1.AuthorizeDingTalkUserRequest) (*v1.AuthorizeDingTalkUserResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	targetOrganizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	roles, err := parseUUIDs(request.GetRoleIds())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	authorized, err := s.usecase.AuthorizeDingTalkUser(ctx, principal.Organization.ID, targetOrganizationID, principal.UserID, &biz.AdminUser{ID: userID, DisplayName: request.GetDisplayName(), Email: optionalString(request.GetEmail(), request.Email != nil)}, roles)
	if err != nil {
		return nil, err
	}
	return &v1.AuthorizeDingTalkUserResponse{Success: true, Code: 0, Message: "OK", Data: userToAPI(authorized), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ResetUserPassword(ctx context.Context, request *v1.ResetUserPasswordRequest) (*v1.ResetUserPasswordResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	if err := s.usecase.ResetUserPassword(ctx, principal.Organization.ID, principal.UserID, userID, request.GetPassword()); err != nil {
		return nil, err
	}
	return &v1.ResetUserPasswordResponse{Success: true, Code: 0, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ListRoles(ctx context.Context, _ *v1.ListRolesRequest) (*v1.ListRolesResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.usecase.ListRoles(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminRole, 0, len(items))
	for _, item := range items {
		data = append(data, roleToAPI(item))
	}
	return &v1.ListRolesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ListOrganizationRoles(ctx context.Context, request *v1.ListOrganizationRolesRequest) (*v1.ListOrganizationRolesResponse, error) {
	organizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	items, err := s.usecase.ListRoles(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminRole, 0, len(items))
	for _, item := range items {
		data = append(data, roleToAPI(item))
	}
	return &v1.ListOrganizationRolesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) CreateRole(ctx context.Context, request *v1.CreateRoleRequest) (*v1.CreateRoleResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	accesses, err := orderOrganizationAccessesFromAPI(request.GetOrderOrganizationAccesses())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.CreateRole(ctx, principal.Organization.ID, principal.UserID, &biz.AdminRole{Code: request.GetCode(), Name: request.GetName(), DataScope: dataScopeFromAPI(request.GetDataScope()), Enabled: true, OrderOrganizationAccesses: accesses}, request.GetPermissionKeys())
	if err != nil {
		return nil, err
	}
	return &v1.CreateRoleResponse{Success: true, Code: 0, Message: "OK", Data: roleToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) UpdateRole(ctx context.Context, request *v1.UpdateRoleRequest) (*v1.UpdateRoleResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	roleID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	accesses, err := orderOrganizationAccessesFromAPI(request.GetOrderOrganizationAccesses())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.UpdateRole(ctx, principal.Organization.ID, principal.UserID, roleID, &biz.AdminRole{ID: roleID, Name: request.GetName(), DataScope: dataScopeFromAPI(request.GetDataScope()), Enabled: request.GetEnabled(), OrderOrganizationAccesses: accesses}, request.GetPermissionKeys())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateRoleResponse{Success: true, Code: 0, Message: "OK", Data: roleToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ListPermissions(ctx context.Context, _ *v1.ListPermissionsRequest) (*v1.ListPermissionsResponse, error) {
	items, err := s.usecase.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminPermission, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.AdminPermission{Key: item.Key, Name: item.Name, Group: item.Group, Description: item.Description})
	}
	return &v1.ListPermissionsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AdminService) ListAuditLogs(ctx context.Context, request *v1.ListAuditLogsRequest) (*v1.ListAuditLogsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := adminPageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	var userID *uuid.UUID
	if value := request.GetUserId(); value != "" {
		parsed, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return nil, biz.ErrAdminInvalidArgument
		}
		userID = &parsed
	}
	startTime, err := parseAuditTime(request.GetStartTime())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	endTime, err := parseAuditTime(request.GetEndTime())
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	list, err := s.usecase.ListAuditLogs(ctx, principal.Organization.ID, biz.AdminAuditLogListOptions{
		Page: page, PageSize: pageSize, Action: request.GetAction(), UserID: userID, StartTime: startTime, EndTime: endTime,
		ResourceType: request.GetResourceType(), ResourceID: request.GetResourceId(),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminAuditLog, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, auditLogToAPI(item))
	}
	return &v1.ListAuditLogsResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func requirePrincipal(ctx context.Context) (*biz.Principal, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	return principal, nil
}

func adminPageValues(page, pageSize int32) (int, int, error) {
	pageValue := int(page)
	if pageValue == 0 {
		pageValue = 1
	}
	pageSizeValue := int(pageSize)
	if pageSizeValue == 0 {
		pageSizeValue = 20
	}
	if pageValue < 1 || pageSizeValue < 1 || pageSizeValue > 100 {
		return 0, 0, biz.ErrAdminInvalidArgument
	}
	return pageValue, pageSizeValue, nil
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[parsed]; exists {
			continue
		}
		seen[parsed] = struct{}{}
		result = append(result, parsed)
	}
	return result, nil
}

func optionalString(value string, present bool) *string {
	if !present {
		return nil
	}
	return &value
}

func parseAuditTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func dataScopeFromAPI(value v1.DataScope) biz.DataScope {
	switch value {
	case v1.DataScope_DATA_SCOPE_ALL:
		return biz.DataScopeAll
	case v1.DataScope_DATA_SCOPE_ORGANIZATION:
		return biz.DataScopeOrganization
	case v1.DataScope_DATA_SCOPE_ORGANIZATION_TREE:
		return biz.DataScopeOrganizationTree
	case v1.DataScope_DATA_SCOPE_SELF:
		return biz.DataScopeSelf
	default:
		return ""
	}
}

func dataScopeToAPI(value biz.DataScope) v1.DataScope {
	switch value {
	case biz.DataScopeAll:
		return v1.DataScope_DATA_SCOPE_ALL
	case biz.DataScopeOrganization:
		return v1.DataScope_DATA_SCOPE_ORGANIZATION
	case biz.DataScopeOrganizationTree:
		return v1.DataScope_DATA_SCOPE_ORGANIZATION_TREE
	case biz.DataScopeSelf:
		return v1.DataScope_DATA_SCOPE_SELF
	default:
		return v1.DataScope_DATA_SCOPE_UNSPECIFIED
	}
}

func organizationToAPI(value *biz.AdminOrganization) *v1.AdminOrganization {
	return &v1.AdminOrganization{Id: value.ID.String(), Code: value.Code, Name: value.Name, Kind: organizationKindToAPI(value.Kind), ParentId: uuidString(value.ParentID), Enabled: value.Enabled, BaseCurrency: value.BaseCurrency}
}

func organizationKindFromAPI(value v1.OrganizationKind) biz.OrganizationKind {
	switch value {
	case v1.OrganizationKind_ORGANIZATION_KIND_COMPANY:
		return biz.OrganizationKindCompany
	case v1.OrganizationKind_ORGANIZATION_KIND_DEPARTMENT:
		return biz.OrganizationKindDepartment
	case v1.OrganizationKind_ORGANIZATION_KIND_TEAM:
		return biz.OrganizationKindTeam
	default:
		return ""
	}
}

func organizationKindToAPI(value biz.OrganizationKind) v1.OrganizationKind {
	switch value {
	case biz.OrganizationKindHeadquarters:
		return v1.OrganizationKind_ORGANIZATION_KIND_HEADQUARTERS
	case biz.OrganizationKindCompany:
		return v1.OrganizationKind_ORGANIZATION_KIND_COMPANY
	case biz.OrganizationKindDepartment:
		return v1.OrganizationKind_ORGANIZATION_KIND_DEPARTMENT
	case biz.OrganizationKindTeam:
		return v1.OrganizationKind_ORGANIZATION_KIND_TEAM
	default:
		return v1.OrganizationKind_ORGANIZATION_KIND_UNSPECIFIED
	}
}

func userToAPI(value *biz.AdminUser) *v1.AdminUser {
	return &v1.AdminUser{Id: value.ID.String(), Username: value.Username, DisplayName: value.DisplayName, Email: value.Email, WecomUserid: value.WeComUserID, WecomName: value.WeComName, DingtalkUnionid: value.DingTalkUnionID, DingtalkName: value.DingTalkName, Enabled: value.Enabled, RoleIds: uuidStrings(value.RoleIDs), RoleCodes: value.RoleCodes, CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339)}
}

func roleToAPI(value *biz.AdminRole) *v1.AdminRole {
	return &v1.AdminRole{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Code: value.Code, Name: value.Name, DataScope: dataScopeToAPI(value.DataScope), Enabled: value.Enabled, PermissionKeys: value.PermissionKeys, OrderOrganizationAccesses: orderOrganizationAccessesToAPI(value.OrderOrganizationAccesses), CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339)}
}

func orderOrganizationAccessesFromAPI(values []*v1.OrderOrganizationAccess) ([]biz.OrderOrganizationAccess, error) {
	result := make([]biz.OrderOrganizationAccess, 0, len(values))
	for _, value := range values {
		organizationID, err := uuid.Parse(value.GetOrganizationId())
		if err != nil {
			return nil, biz.ErrAdminInvalidArgument
		}
		result = append(result, biz.OrderOrganizationAccess{OrganizationID: organizationID, Writable: value.GetWritable()})
	}
	return result, nil
}

func orderOrganizationAccessesToAPI(values []biz.OrderOrganizationAccess) []*v1.OrderOrganizationAccess {
	result := make([]*v1.OrderOrganizationAccess, 0, len(values))
	for _, value := range values {
		result = append(result, &v1.OrderOrganizationAccess{OrganizationId: value.OrganizationID.String(), Writable: value.Writable})
	}
	return result
}

func auditLogToAPI(value *biz.AuditLog) *v1.AdminAuditLog {
	return &v1.AdminAuditLog{Id: value.ID.String(), OrganizationId: uuidString(value.OrganizationID), UserId: uuidString(value.UserID), Action: value.Action, ResourceType: value.ResourceType, ResourceId: value.ResourceID, Result: value.Result, RequestId: value.RequestID, TraceId: value.TraceID, IpAddress: value.IPAddress, Details: value.Details, CreatedAt: value.CreatedAt.Format(time.RFC3339)}
}

func uuidString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	result := value.String()
	return &result
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}

var _ v1.AdminServiceServer = (*AdminService)(nil)
