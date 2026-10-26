package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
)

// 分页候选复用现有 List 检索：代码、中文、英文、全拼、首字母都只返回启用且同类的项。
func TestMasterDataSearchOptionsPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	suffix := uuid.NewString()[:8]
	target := d.db.MasterDataItem.Create().SetKind(masterdataitem.KindAbnormalCase).SetCode("INSPECTION-" + suffix).SetName("星海查验").SetNameEn("Customs Fixture Inspection " + suffix).SetEnabled(true).SaveX(ctx)
	d.db.MasterDataItem.Create().SetKind(masterdataitem.KindAbnormalCase).SetCode("DISABLED-" + suffix).SetName("星海查验").SetNameEn("Customs Fixture Inspection " + suffix).SetEnabled(false).SaveX(ctx)
	d.db.MasterDataItem.Create().SetKind(masterdataitem.KindCargoCategory).SetCode("CARGO-" + suffix).SetName("星海查验").SetNameEn("Customs Fixture Inspection " + suffix).SetEnabled(true).SaveX(ctx)
	usecase := biz.NewMasterDataUsecase(NewMasterDataRepo(d))
	enabled := true
	for _, keyword := range []string{"INSPECTION-" + suffix, "星海查验", "Customs Fixture Inspection " + suffix, "xinghaichayan", "XHCY"} {
		result, err := usecase.List(ctx, uuid.New(), biz.MasterDataListOptions{Kind: biz.MasterDataKindAbnormalCase, Keyword: keyword, Page: 1, PageSize: 20, Enabled: &enabled})
		if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != target.ID {
			t.Fatalf("启用分类候选搜索错误 keyword=%q result=%+v err=%v", keyword, result, err)
		}
	}
	// 同名结果按代码顺序稳定分页，总数只包含启用同类项。
	second := d.db.MasterDataItem.Create().SetKind(masterdataitem.KindAbnormalCase).SetCode("ZZZ-" + suffix).SetName("星海查验").SetEnabled(true).SaveX(ctx)
	for i, want := range []uuid.UUID{target.ID, second.ID} {
		result, err := usecase.List(ctx, uuid.New(), biz.MasterDataListOptions{Kind: biz.MasterDataKindAbnormalCase, Keyword: "星海查验", Page: i + 1, PageSize: 1, Enabled: &enabled})
		if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].ID != want {
			t.Fatalf("分页候选错误 page=%d result=%+v err=%v", i+1, result, err)
		}
	}
}
