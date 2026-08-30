package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type OrderPersonnelService struct {
	v1.UnimplementedOrderPersonnelServiceServer
	usecase *biz.OrderPersonnelUsecase
}

func NewOrderPersonnelService(usecase *biz.OrderPersonnelUsecase) *OrderPersonnelService {
	return &OrderPersonnelService{usecase: usecase}
}

func (s *OrderPersonnelService) ListPersonnel(ctx context.Context, request *v1.ListPersonnelRequest) (*v1.ListPersonnelResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderPersonnel, 0, len(items))
	for _, item := range items {
		data = append(data, orderPersonnelToAPI(item))
	}
	return okList(ctx, &v1.ListPersonnelResponse{
		Data: data,
	}), nil
}

func (s *OrderPersonnelService) AssignPersonnel(ctx context.Context, request *v1.AssignPersonnelRequest) (*v1.AssignPersonnelResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	userID, err := uuid.Parse(request.GetUserId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	memberOrganizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	role, err := protoRoleToBiz(request.GetRole())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Assign(ctx, principal.Organization.ID, principal.UserID, orderID, userID, memberOrganizationID, role)
	if err != nil {
		return nil, err
	}
	return orderPersonnelResponse(ctx, created), nil
}

func (s *OrderPersonnelService) RemovePersonnel(ctx context.Context, request *v1.RemovePersonnelRequest) (*v1.RemovePersonnelResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderPersonnelInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemovePersonnelResponse{}), nil
}

func orderPersonnelResponse(ctx context.Context, value *biz.OrderPersonnel) *v1.AssignPersonnelResponse {
	return ok(ctx, &v1.AssignPersonnelResponse{
		Data: orderPersonnelToAPI(value),
	})

}

func orderPersonnelToAPI(value *biz.OrderPersonnel) *v1.OrderPersonnel {
	return &v1.OrderPersonnel{
		Id:             value.ID.String(),
		OrderId:        value.OrderID.String(),
		UserId:         value.UserID.String(),
		OrganizationId: value.OrganizationID.String(),
		Role:           bizRoleToProto(value.Role),
		AssignedAt:     value.AssignedAt.UTC().Format(time.RFC3339),
		CreatedAt:      value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      value.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func protoRoleToBiz(role v1.OrderPersonnelRole) (biz.OrderPersonnelRole, error) {
	switch role {
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CREATOR:
		return biz.OrderPersonnelRoleCreator, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_OPERATOR:
		return biz.OrderPersonnelRoleOperator, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_SALES:
		return biz.OrderPersonnelRoleSales, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE:
		return biz.OrderPersonnelRoleCustomerService, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_DOCUMENT:
		return biz.OrderPersonnelRoleDocument, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_COMMERCIAL:
		return biz.OrderPersonnelRoleCommercial, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE:
		return biz.OrderPersonnelRoleAssociate, nil
	case v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE2:
		return biz.OrderPersonnelRoleAssociate2, nil
	default:
		return "", biz.ErrOrderPersonnelInvalidArgument
	}
}

func bizRoleToProto(role biz.OrderPersonnelRole) v1.OrderPersonnelRole {
	switch role {
	case biz.OrderPersonnelRoleCreator:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CREATOR
	case biz.OrderPersonnelRoleOperator:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_OPERATOR
	case biz.OrderPersonnelRoleSales:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_SALES
	case biz.OrderPersonnelRoleCustomerService:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE
	case biz.OrderPersonnelRoleDocument:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_DOCUMENT
	case biz.OrderPersonnelRoleCommercial:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_COMMERCIAL
	case biz.OrderPersonnelRoleAssociate:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE
	case biz.OrderPersonnelRoleAssociate2:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE2
	default:
		return v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_UNSPECIFIED
	}
}

var _ v1.OrderPersonnelServiceServer = (*OrderPersonnelService)(nil)
