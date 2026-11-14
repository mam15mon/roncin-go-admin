package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerprofile"
)

func TestVerificationCandidatesSearchPaginationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	t.Run("两类未结清候选超过200条仍可独立翻页和搜索", func(t *testing.T) {
		f := newFeeLedgerPostgresFixture(t, data)
		ctx := context.Background()
		for index := range 205 {
			key := fmt.Sprintf("page-%03d", index)
			f.createConfirmedBill(key, "10", "2026-09-01")
			f.createConfirmedCashflow(key, "10", "2026-09-01")
		}
		if _, err := data.db.Partner.UpdateOneID(f.partnerID).SetCode("CAND-CODE-" + f.suffix).SetLegalName("青岛远航").SetNormalizedName("青岛远航").Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.PartnerProfile.Create().SetPartnerID(f.partnerID).SetNameEn("OceanFixture").Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.PartnerAlias.Create().SetPartnerID(f.partnerID).SetAliasName("测试分支").SetNormalizedAliasName("测试分支").Save(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := data.db.PartnerAlias.Delete().Where(partneralias.PartnerIDEQ(f.partnerID)).Exec(ctx); err != nil {
				t.Error(err)
			}
			if _, err := data.db.PartnerProfile.Delete().Where(partnerprofile.PartnerIDEQ(f.partnerID)).Exec(ctx); err != nil {
				t.Error(err)
			}
		})
		usecase := biz.NewVerificationUsecase(NewVerificationRepo(data), nil, nil, nil)
		filter := biz.VerificationCreationCandidateFilter{Page: 1, PageSize: 200, Direction: biz.OrderFeeReceivable, SettlementPartyID: f.partnerID, Currency: "CNY"}
		cashPage1, err := usecase.ListCashflowCandidates(ctx, f.organizationID, filter)
		if err != nil || cashPage1.Total != 205 || len(cashPage1.Items) != 200 {
			t.Fatalf("资金第一页: %+v err=%v", cashPage1, err)
		}
		billPage1, err := usecase.ListBillCandidates(ctx, f.organizationID, filter)
		if err != nil || billPage1.Total != 205 || len(billPage1.Items) != 200 {
			t.Fatalf("账单第一页: %+v err=%v", billPage1, err)
		}
		filter.Page = 2
		cashPage2, err := usecase.ListCashflowCandidates(ctx, f.organizationID, filter)
		if err != nil || cashPage2.Total != 205 || len(cashPage2.Items) != 5 {
			t.Fatalf("资金第二页: %+v err=%v", cashPage2, err)
		}
		billPage2, err := usecase.ListBillCandidates(ctx, f.organizationID, filter)
		if err != nil || billPage2.Total != 205 || len(billPage2.Items) != 5 {
			t.Fatalf("账单第二页: %+v err=%v", billPage2, err)
		}
		cashIDs, billIDs := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
		for _, item := range cashPage1.Items {
			cashIDs[item.ID] = true
		}
		for _, item := range billPage1.Items {
			billIDs[item.ID] = true
		}
		for _, item := range cashPage2.Items {
			if cashIDs[item.ID] {
				t.Fatal("资金跨页重复")
			}
		}
		for _, item := range billPage2.Items {
			if billIDs[item.ID] {
				t.Fatal("账单跨页重复")
			}
		}
		filter.Page = 1
		for _, keyword := range []string{"page-204", "CAND-CODE", "青岛远航", "qingdaoyuanhang", "qdyh", "OceanFixture", "测试分支", "ceshifenzhi", "csfz"} {
			filter.Keyword = keyword
			cashResult, cashErr := usecase.ListCashflowCandidates(ctx, f.organizationID, filter)
			billResult, billErr := usecase.ListBillCandidates(ctx, f.organizationID, filter)
			want := int64(205)
			if keyword == "page-204" {
				want = 1
			}
			if cashErr != nil || billErr != nil || cashResult.Total != want || billResult.Total != want {
				t.Fatalf("关键词%s: cash=%+v bill=%+v err=%v/%v", keyword, cashResult, billResult, cashErr, billErr)
			}
		}
		filter.Keyword = ""
		foreignCash, err := usecase.ListCashflowCandidates(ctx, uuid.New(), filter)
		if err != nil || foreignCash.Total != 0 {
			t.Fatalf("资金公司隔离失败: %+v %v", foreignCash, err)
		}
		filter.SettlementPartyID = uuid.New()
		foreignBill, err := usecase.ListBillCandidates(ctx, f.organizationID, filter)
		if err != nil || foreignBill.Total != 0 {
			t.Fatalf("账单结算对象隔离失败: %+v %v", foreignBill, err)
		}
	})
}
