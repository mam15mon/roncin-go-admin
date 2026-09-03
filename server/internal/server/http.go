package server

import (
	"log/slog"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	enterpriseresourcev1 "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	masterdatav1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	taskv1 "github.com/roncin/roncin-go-admin/server/api/task/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"github.com/roncin/roncin-go-admin/server/internal/service"
	"github.com/roncin/roncin-go-admin/server/internal/webassets"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, enterpriseResource *service.EnterpriseResourceService, auth *service.AuthService, partner *service.PartnerService, admin *service.AdminService, masterData *service.MasterDataService, order *service.OrderService, orderTag *service.OrderTagService, milestones *service.OrderMilestoneService, orderAttachment *service.OrderAttachmentService, orderPersonnel *service.OrderPersonnelService, backgroundTask *service.BackgroundTaskService, orderContainer *service.OrderContainerService, orderCargoItem *service.OrderCargoItemService, shippingDocument *service.OrderShippingDocumentService, seaDocument *service.SeaDocumentService, seaCargoAllocation *service.SeaCargoAllocationService, seaOrderChange *service.SeaOrderChangeService, abnormalCase *service.OrderAbnormalCaseService, releasePod *service.OrderReleasePodService, exchangeRate *service.ExchangeRateService, feeCatalog *service.FeeCatalogService, orderFee *service.OrderFeeService, settlement *service.SettlementService, authUsecase *biz.AuthUsecase, orderUsecase *biz.OrderUsecase, policy *biz.SessionPolicy, readiness ReadinessChecker, logger *slog.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.ResponseEncoder(encodeResponse),
		http.Middleware(
			Recovery(logger),
			tracing.Server(),
			requestmeta.Middleware(),
			requestmeta.Metrics(),
			requestmeta.Logging(logger),
			Authorization(authUsecase, policy, orderUsecase),
			RequiredFieldsValidator(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	opts = append(opts, http.ErrorEncoder(encodeError))
	srv := http.NewServer(opts...)
	enterpriseresourcev1.RegisterEnterpriseResourceServiceHTTPServer(srv, enterpriseResource)
	authv1.RegisterAuthServiceHTTPServer(srv, auth)
	partnerv1.RegisterPartnerServiceHTTPServer(srv, partner)
	adminv1.RegisterAdminServiceHTTPServer(srv, admin)
	masterdatav1.RegisterMasterDataServiceHTTPServer(srv, masterData)
	orderv1.RegisterOrderServiceHTTPServer(srv, order)
	orderv1.RegisterOrderTagServiceHTTPServer(srv, orderTag)
	orderv1.RegisterOrderMilestoneServiceHTTPServer(srv, milestones)
	orderv1.RegisterOrderAttachmentServiceHTTPServer(srv, orderAttachment)
	orderv1.RegisterOrderPersonnelServiceHTTPServer(srv, orderPersonnel)
	taskv1.RegisterBackgroundTaskServiceHTTPServer(srv, backgroundTask)
	orderv1.RegisterOrderContainerServiceHTTPServer(srv, orderContainer)
	orderv1.RegisterOrderCargoItemServiceHTTPServer(srv, orderCargoItem)
	orderv1.RegisterOrderShippingDocumentServiceHTTPServer(srv, shippingDocument)
	orderv1.RegisterSeaDocumentServiceHTTPServer(srv, seaDocument)
	orderv1.RegisterSeaCargoAllocationServiceHTTPServer(srv, seaCargoAllocation)
	orderv1.RegisterSeaOrderChangeServiceHTTPServer(srv, seaOrderChange)
	orderv1.RegisterOrderAbnormalCaseServiceHTTPServer(srv, abnormalCase)
	orderv1.RegisterOrderReleasePodServiceHTTPServer(srv, releasePod)
	financev1.RegisterExchangeRateServiceHTTPServer(srv, exchangeRate)
	financev1.RegisterFeeCatalogServiceHTTPServer(srv, feeCatalog)
	orderv1.RegisterOrderFeeServiceHTTPServer(srv, orderFee)
	financev1.RegisterSettlementServiceHTTPServer(srv, settlement)
	registerHealthHandlers(srv, readiness)
	srv.Handle("/metrics", promhttp.Handler())
	srv.HandlePrefix("/", webassets.Handler())
	return srv
}
