package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/wire"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewAuthRepo, NewAuditRepo, NewPartnerRepo, NewPartnerAccountRepo, NewPartnerContractRepo, NewPartnerSettlementRuleRepo, NewPartnerAttachmentRepo, NewPartnerShippingPresetRepo, NewAdminRepo, NewMasterDataRepo, NewIndustryReferenceRepo, NewReferenceDataRepo, NewOrderConfigRepo, NewMilestoneConfigRepo, NewOrderRepo, NewOrderMilestoneRepo, NewOrderAttachmentRepo, NewOrderPersonnelRepo, NewOrderContainerRepo, NewOrderCargoItemRepo, NewOrderShippingDocumentRepo, NewOrderReleasePodRepo, NewOrderAbnormalCaseRepo, NewBackgroundTaskRepo)

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	db    *ent.Client
	sqlDB *sql.DB
}

// Ping verifies that the primary database is reachable.
func (d *Data) Ping(ctx context.Context) error {
	return d.sqlDB.PingContext(ctx)
}

// NewData opens the database client and returns it with a cleanup function.
func NewData(c *conf.Data) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetDriver() != dialect.Postgres {
		return nil, nil, fmt.Errorf("unsupported database driver %q: expected postgres", dc.GetDriver())
	}
	if dc.GetSource() == "" {
		return nil, nil, fmt.Errorf("database source is required")
	}
	sqlDB, err := sql.Open("pgx", dc.GetSource())
	if err != nil {
		return nil, nil, err
	}
	if dc.GetMaxOpenConnections() > 0 {
		sqlDB.SetMaxOpenConns(int(dc.GetMaxOpenConnections()))
	}
	if dc.GetMaxIdleConnections() > 0 {
		sqlDB.SetMaxIdleConns(int(dc.GetMaxIdleConnections()))
	}
	if dc.GetConnectionMaxLifetime() != nil {
		sqlDB.SetConnMaxLifetime(dc.GetConnectionMaxLifetime().AsDuration())
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	db := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	if dc.GetDebug() {
		db = db.Debug()
	}
	// Auto migration is a convenience for local development. In production,
	// apply schema changes as a separate reviewed step instead.
	if dc.GetAutoMigrate() {
		if err := db.Schema.Create(context.Background()); err != nil {
			db.Close()
			return nil, nil, err
		}
	}
	cleanup := func() {
		log.Info("closing the data resources")
		if err := db.Close(); err != nil {
			log.Error("failed closing the database", "err", err)
		}
	}
	return &Data{db: db, sqlDB: sqlDB}, cleanup, nil
}
