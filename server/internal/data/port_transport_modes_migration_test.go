package data

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func TestPortTransportModesMigrationPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261109110000")
	defer cleanup()
	ctx := context.Background()
	port := d.db.Port.Create().SetUnLocode("CNSHA").SetNameEn("Port of Shanghai").SetNameZh("上海港").SetCountryCode("CN").SetTransportModes([]string{"maritime"}).SetSource("manual").SaveX(ctx)
	standard := d.db.Port.Create().SetUnLocode("CNSHG").SetNameEn("Shanghai Pt").SetCountryCode("CN").SetTransportModes([]string{"SEA", "RAIL"}).SaveX(ctx)
	standard = d.db.Port.GetX(ctx, standard.ID)
	for range 2 {
		if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
	}
	got := d.db.Port.GetX(ctx, port.ID)
	if !reflect.DeepEqual(got.TransportModes, []string{"SEA"}) || got.UnLocode != port.UnLocode || got.Source != port.Source || got.NameEn != port.NameEn {
		t.Fatalf("运输类型应修正且保留港口身份与资料: %+v", got)
	}
	got = d.db.Port.GetX(ctx, standard.ID)
	if !reflect.DeepEqual(got.TransportModes, standard.TransportModes) || !got.UpdatedAt.Equal(standard.UpdatedAt) {
		t.Fatal("标准港口不应被修改")
	}
}
