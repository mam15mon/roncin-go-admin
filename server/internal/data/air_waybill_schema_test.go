package data

import (
	"testing"

	entmigrate "github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

func TestAirWaybillSchemaModeAndJSONContract(t *testing.T) {
	links := entmigrate.AirMasterWaybillOrderLinksTable
	if links.Annotation == nil || links.Annotation.Checks["air_links_document_mode_check"] != "document_mode IN ('DIRECT','HOUSE')" {
		t.Fatal("Ent未声明航空模式CHECK")
	}
	found := false
	for _, idx := range links.Indexes {
		if idx.Name == "idx_air_links_active_direct_master" {
			found = true
			if !idx.Unique || idx.Annotation == nil || idx.Annotation.Where != "status = 'ACTIVE' AND document_mode = 'DIRECT'" {
				t.Fatal("活动DIRECT索引谓词错误")
			}
		}
	}
	if !found {
		t.Fatal("缺少活动DIRECT主单唯一索引")
	}
	for _, table := range []string{"air_house_waybills", "air_house_waybill_versions", "air_master_waybills"} {
		found := false
		for _, tb := range entmigrate.Tables {
			if tb.Name != table {
				continue
			}
			for _, col := range tb.Columns {
				if col.Name == "content_hash" {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("缺少%s正文哈希列", table)
		}
	}
}
