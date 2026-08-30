package syncrunner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

type Options struct {
	Apply            bool
	Source           string
	Release          string
	OrganizationCode string
}

func Run(task func(context.Context) error) {
	if err := task(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func OpenStore() (*data.IndustryReferenceSyncStore, func(), error) {
	databaseSource := strings.TrimSpace(os.Getenv("DATABASE_SOURCE"))
	if databaseSource == "" {
		return nil, nil, errors.New("DATABASE_SOURCE 不能为空")
	}
	storage, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: databaseSource}}, slog.Default())
	if err != nil {
		return nil, nil, err
	}
	return data.NewIndustryReferenceSyncStore(storage), cleanup, nil
}
