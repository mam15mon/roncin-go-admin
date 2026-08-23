package server

import (
	"log/slog"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	masterdatav1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	taskv1 "github.com/roncin/roncin-go-admin/server/api/task/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"github.com/roncin/roncin-go-admin/server/internal/service"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, auth *service.AuthService, partner *service.PartnerService, admin *service.AdminService, masterData *service.MasterDataService, order *service.OrderService, milestones *service.OrderMilestoneService, orderAttachment *service.OrderAttachmentService, orderPersonnel *service.OrderPersonnelService, backgroundTask *service.BackgroundTaskService, orderContainer *service.OrderContainerService, orderCargoItem *service.OrderCargoItemService, shippingDocument *service.OrderShippingDocumentService, abnormalCase *service.OrderAbnormalCaseService, releasePod *service.OrderReleasePodService, authUsecase *biz.AuthUsecase, orderUsecase *biz.OrderUsecase, policy *biz.SessionPolicy, logger *slog.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			requestmeta.Middleware(),
			Authorization(authUsecase, policy, orderUsecase),
			requestmeta.Logging(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	authv1.RegisterAuthServiceServer(srv, auth)
	partnerv1.RegisterPartnerServiceServer(srv, partner)
	adminv1.RegisterAdminServiceServer(srv, admin)
	masterdatav1.RegisterMasterDataServiceServer(srv, masterData)
	orderv1.RegisterOrderServiceServer(srv, order)
	orderv1.RegisterOrderMilestoneServiceServer(srv, milestones)
	orderv1.RegisterOrderAttachmentServiceServer(srv, orderAttachment)
	orderv1.RegisterOrderPersonnelServiceServer(srv, orderPersonnel)
	taskv1.RegisterBackgroundTaskServiceServer(srv, backgroundTask)
	orderv1.RegisterOrderContainerServiceServer(srv, orderContainer)
	orderv1.RegisterOrderCargoItemServiceServer(srv, orderCargoItem)
	orderv1.RegisterOrderShippingDocumentServiceServer(srv, shippingDocument)
	orderv1.RegisterOrderAbnormalCaseServiceServer(srv, abnormalCase)
	orderv1.RegisterOrderReleasePodServiceServer(srv, releasePod)
	return srv
}
