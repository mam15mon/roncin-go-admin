package server

import (
	"log/slog"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport/http"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	v1 "github.com/roncin/roncin-go-admin/server/api/todo/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"github.com/roncin/roncin-go-admin/server/internal/service"

	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, auth *service.AuthService, todo *service.TodoService, authUsecase *biz.AuthUsecase, policy *biz.SessionPolicy, logger *slog.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			requestmeta.Middleware(),
			Authorization(authUsecase, policy),
			requestmeta.Logging(logger),
			validate.Validator(func(req any) error {
				if msg, ok := req.(proto.Message); ok {
					if err := fieldbehavior.ValidateRequiredFields(msg); err != nil {
						return err
					}
				}
				return nil
			}),
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
	authv1.RegisterAuthServiceHTTPServer(srv, auth)
	v1.RegisterTodoServiceHTTPServer(srv, todo)
	return srv
}
