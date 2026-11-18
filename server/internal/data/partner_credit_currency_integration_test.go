package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
)

func TestPartnerCreditCurrencyPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	for _, code := range []string{"CNY", "USD"} {
		exists, err := data.db.Currency.Query().Where(currencyent.CodeEQ(code)).Exist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			if _, err := data.db.Currency.Create().SetCode(code).SetName(code).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, baseCurrency := range []string{"CNY", "USD"} {
		t.Run(baseCurrency, func(t *testing.T) {
			f := newCreditPartnerFixture(t, data)
			if err := data.db.Organization.UpdateOneID(f.organizationID).SetBaseCurrency(baseCurrency).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			f.baseCurrency = baseCurrency
			partnerID, roleID := f.newCreditPartner("币种客户")
			settlementCurrency := "USD"
			if baseCurrency == "USD" {
				settlementCurrency = "CNY"
			}
			amount := int64(100000)
			input := &biz.PartnerSettlementRule{CreditLimitMinor: &amount, CreditCurrency: &baseCurrency}
			audit := func() *biz.AuditEvent {
				return &biz.AuditEvent{OrganizationID: &f.organizationID, Action: "partner.settlement_rule.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{}}
			}
			repo := NewPartnerSettlementRuleRepo(data)
			invalid := *input
			invalid.CreditCurrency = &settlementCurrency
			if _, err := repo.Create(ctx, f.organizationID, partnerID, biz.PartnerRoleCustomer, &invalid, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("外币额度必须拒绝，实际 %v", err)
			}
			if count, err := data.db.PartnerSettlementRule.Query().Where(partnersettlementruleent.PartnerRoleIDEQ(roleID)).Count(ctx); err != nil || count != 0 {
				t.Fatalf("失败创建不得留下规则: count=%d err=%v", count, err)
			}
			created, err := repo.Create(ctx, f.organizationID, partnerID, biz.PartnerRoleCustomer, input, audit())
			if err != nil {
				t.Fatalf("客户可配所属公司本币额度: %v", err)
			}
			if _, err := repo.Update(ctx, f.organizationID, partnerID, biz.PartnerRoleCustomer, created.ID, &invalid, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("外币额度更新必须拒绝: %v", err)
			}
			partnerRepo := NewPartnerRepo(data)
			partner, err := partnerRepo.Get(ctx, f.organizationID, partnerID)
			if err != nil {
				t.Fatal(err)
			}
			originalName := partner.LegalName
			partner.LegalName = "不得落库的新名称"
			partner.NormalizedName = partner.LegalName
			partner.Roles[0].SettlementRule = &invalid
			if _, err := partnerRepo.Update(ctx, f.organizationID, partnerID, partner, biz.PartnerSettlementRuleWriteAccess{Create: true, Update: true}, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("内嵌结算规则同样拒绝: %v", err)
			}
			if _, err := partnerRepo.Import(ctx, f.organizationID, biz.PartnerImportUpsert, []*biz.Partner{partner}, biz.PartnerSettlementRuleWriteAccess{Create: true, Update: true}, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("导入更新内嵌结算规则同样拒绝: %v", err)
			}
			stored, err := data.db.Partner.Get(ctx, partnerID)
			if err != nil || stored.LegalName != originalName {
				t.Fatalf("内嵌失败必须回滚档案修改: %+v %v", stored, err)
			}
			beforeCount, err := data.db.Partner.Query().Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			newPartner := *partner
			newPartner.Code = "INVALID-CURRENCY-" + uuid.NewString()[:8]
			newPartner.LegalName = newPartner.Code
			newPartner.NormalizedName = newPartner.Code
			if _, err := partnerRepo.Create(ctx, f.organizationID, &newPartner, biz.PartnerSettlementRuleWriteAccess{Create: true, Update: true}, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("新建档案的内嵌结算规则同样拒绝: %v", err)
			}
			if _, err := partnerRepo.Import(ctx, f.organizationID, biz.PartnerImportUpsert, []*biz.Partner{&newPartner}, biz.PartnerSettlementRuleWriteAccess{Create: true, Update: true}, audit()); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("导入新建内嵌结算规则同样拒绝: %v", err)
			}
			if count, err := data.db.Partner.Query().Count(ctx); err != nil || count != beforeCount {
				t.Fatalf("拒绝内嵌新建不得留下档案 count=%d err=%v", count, err)
			}
			f.newConfirmedBill("currency", partnerID, originalName, financebillent.DirectionRECEIVABLE, financebillent.StatusCONFIRMED, "2000")
			summaries, err := f.repo.GetPartnerCreditSummaries(ctx, f.organizationID, []uuid.UUID{partnerID})
			if err != nil || !biz.PartnerCreditExceeded(summaries[partnerID]) {
				t.Fatalf("本币欠款2000超过本币额度1000: %+v %v", summaries, err)
			}
			// 唯一约定的异常币种必须显式报错，维护入口仍可读取原值。
			if err := data.db.PartnerSettlementRule.UpdateOneID(created.ID).SetCreditCurrency(settlementCurrency).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := f.repo.GetPartnerCreditSummaries(ctx, f.organizationID, []uuid.UUID{partnerID}); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("异常币种不可参与比较: %v", err)
			}
			if items, err := repo.List(ctx, f.organizationID, partnerID, biz.PartnerRoleCustomer); err != nil || len(items) != 1 || items[0].CreditCurrency == nil || *items[0].CreditCurrency != settlementCurrency {
				t.Fatalf("管理入口须保留异常原值: %+v %v", items, err)
			}
			if err := data.db.PartnerSettlementRule.UpdateOneID(created.ID).ClearCreditCurrency().Exec(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := f.repo.GetPartnerCreditSummaries(ctx, f.organizationID, []uuid.UUID{partnerID}); err != biz.ErrPartnerCreditCurrencyMismatch {
				t.Fatalf("正额度缺币种不得解释成本币: %v", err)
			}

		})
	}
}
