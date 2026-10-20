package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
)

func TestPartnerContractNumberAllocationPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().SetCode("CT-" + suffix).SetName("合同取号测试组织-" + suffix).SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := data.WithTx(ctx, func(tx *ent.Tx) error { return CreateDefaultNumberRules(ctx, tx, organization.ID) }); err != nil {
		t.Fatal(err)
	}
	partner, err := data.db.Partner.Create().SetOrganizationID(organization.ID).SetLegalName("合同取号测试客户").SetNormalizedName("合同取号测试客户").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := data.db.User.Create().SetDisplayName("合同取号测试用户").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	usecase := biz.NewPartnerContractUsecase(NewPartnerContractRepo(data))
	start := time.Now().UTC().Truncate(24 * time.Hour)
	input := &biz.PartnerContract{Name: "年度合同", Status: biz.PartnerContractPending, StartDate: start, EndDate: start.AddDate(1, 0, 0)}
	first, err := usecase.Create(ctx, organization.ID, actor.ID, partner.ID, input)
	if err != nil {
		t.Fatalf("首次合同创建失败: %v", err)
	}
	second, err := usecase.Create(ctx, organization.ID, actor.ID, partner.ID, input)
	if err != nil {
		t.Fatalf("第二次合同创建失败: %v", err)
	}
	if !strings.HasPrefix(first.ContractNo, "CT") || len(first.ContractNo) < 7 || second.ContractNo[:len(second.ContractNo)-5] != first.ContractNo[:len(first.ContractNo)-5] || second.ContractNo[len(second.ContractNo)-5:] != "00002" {
		t.Fatalf("合同编号未按规则连续生成: %q、%q", first.ContractNo, second.ContractNo)
	}
	if _, err := usecase.Create(ctx, organization.ID, actor.ID, uuid.New(), input); err != biz.ErrPartnerContractInvalidArgument {
		t.Fatalf("跨组织/不存在的伙伴错误 = %v", err)
	}
	third, err := usecase.Create(ctx, organization.ID, actor.ID, partner.ID, input)
	if err != nil || third.ContractNo[len(third.ContractNo)-5:] != "00003" {
		t.Fatalf("无效伙伴不得占用流水: 合同 = %#v, 错误 = %v", third, err)
	}
	if _, err := data.db.NumberRule.Update().Where(numberrule.OrganizationIDEQ(organization.ID), numberrule.DocumentTypeEQ(numberrule.DocumentTypeContract)).SetEnabled(false).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := usecase.Create(ctx, organization.ID, actor.ID, partner.ID, input); err != biz.ErrNumberRuleNotFound {
		t.Fatalf("禁用规则后错误 = %v，期望 ErrNumberRuleNotFound", err)
	}
}
