package data

import (
	"context"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestReferenceDataListAdministrativeRegionsFullList(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewReferenceDataRepo(data)
	seeds := []struct {
		code   string
		name   string
		level  int
		parent string
	}{
		{code: "110000000000", name: "北京市", level: 1},
		{code: "130000000000", name: "河北省", level: 1},
		{code: "130100000000", name: "石家庄市", level: 2, parent: "130000000000"},
	}
	for _, seed := range seeds {
		create := data.db.AdministrativeRegion.Create().
			SetCode(seed.code).SetName(seed.name).SetLevel(seed.level)
		if seed.parent != "" {
			create = create.SetParentCode(seed.parent)
		}
		if _, err := create.Save(ctx); err != nil {
			t.Fatalf("创建区划 %s 失败: %v", seed.code, err)
		}
	}

	// 1) 全量语义：0/0 一次返回全部，按代码升序，不分页
	all, err := repo.ListAdministrativeRegions(ctx, biz.AdministrativeRegionQuery{})
	if err != nil {
		t.Fatalf("ListAdministrativeRegions(full) error = %v", err)
	}
	if len(all.Items) != len(seeds) || all.Total != len(seeds) {
		t.Fatalf("full list = %d items (total %d), want %d", len(all.Items), all.Total, len(seeds))
	}
	if all.Items[0].Code != "110000000000" || all.Items[2].Code != "130100000000" {
		t.Fatalf("full list order wrong: %s .. %s", all.Items[0].Code, all.Items[2].Code)
	}
	if all.Page != 1 || all.PageSize != len(seeds) {
		t.Fatalf("full list page/pageSize = %d/%d, want 1/%d", all.Page, all.PageSize, len(seeds))
	}

	// 2) 全量分支同样应用 keyword 过滤
	filtered, err := repo.ListAdministrativeRegions(ctx, biz.AdministrativeRegionQuery{Keyword: "石家庄"})
	if err != nil {
		t.Fatalf("ListAdministrativeRegions(full+keyword) error = %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].Name != "石家庄市" {
		t.Fatalf("full+keyword list = %#v, want 石家庄市 only", filtered.Items)
	}

	// 3) 显式分页语义保持不变
	paged, err := repo.ListAdministrativeRegions(ctx, biz.AdministrativeRegionQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListAdministrativeRegions(paged) error = %v", err)
	}
	if len(paged.Items) != 2 || paged.Total != len(seeds) || paged.PageSize != 2 {
		t.Fatalf("paged list = %d items (total %d, pageSize %d), want 2/%d/2", len(paged.Items), paged.Total, paged.PageSize, len(seeds))
	}
}
