package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/platform/telemetry"
	"github.com/roncin/roncin-go-admin/server/internal/server"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "roncin-server"
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server, notifications *server.NotificationWorker, objectDeletions *server.ObjectDeletionWorker) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
			notifications,
			objectDeletions,
		),
	)
}

func newRuntimeConfig(path string) config.Config {
	return config.New(
		config.WithSource(
			file.NewSource(path),
			env.NewSource(),
		),
		config.WithResolveActualTypes(true),
	)
}

func parseLogLevel(value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("invalid logging level %q: %w", value, err)
	}
	return level, nil
}

func newLogger(level slog.Level) *slog.Logger {
	return log.NewLogger(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     level,
		}),
		log.WithExtractor(tracing.TraceAttrs),
	).With(
		slog.String("service.id", id),
		slog.String("service.name", Name),
		slog.String("service.version", Version),
	)
}

func main() {
	flag.Parse()
	c := newRuntimeConfig(flagconf)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	logLevel, err := parseLogLevel(bc.GetLogging().GetLevel())
	if err != nil {
		panic(err)
	}
	logger := newLogger(logLevel)
	log.SetDefault(logger)
	shutdownTelemetry, err := telemetry.Setup(context.Background(), bc.Telemetry, Name, Version)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			logger.Error("shutdown telemetry", slog.Any("error", err))
		}
	}()

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Security, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
