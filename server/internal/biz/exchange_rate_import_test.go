package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type exchangeRateImportRepoStub struct {
	ExchangeRateRepo
	context          *ExchangeRateContext
	inspectionErrors map[int][]string
	created          *ExchangeRateImportBatch
}

func (stub *exchangeRateImportRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return stub.context, nil
}

func (stub *exchangeRateImportRepoStub) InspectImport(context.Context, uuid.UUID, []*ExchangeRateImportRow) (map[int][]string, error) {
	return stub.inspectionErrors, nil
}

func (stub *exchangeRateImportRepoStub) CreateImportPreview(_ context.Context, batch *ExchangeRateImportBatch, _ *AuditEvent) (*ExchangeRateImportBatch, error) {
	stub.created = batch
	return batch, nil
}

func TestNormalizeExchangeRateImportRowsSupportsSecondPrecision(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{{
		RowNumber: 2, FromCurrency: " usd ", ToCurrency: "cny",
		ARRate: "7.3", APRate: "7.1", Rate: "7.2", EffectiveFrom: "2026-08-27 09:30:01",
	}}, "CNY")
	if len(rows) != 1 || rows[0].Status != ExchangeRateImportRowValid || rows[0].SettingID == uuid.Nil {
		t.Fatalf("合法秒级汇率行应通过规范化: %#v", rows)
	}
	// 周内时刻归一化为自然周窗口，双轨点差与基准价按 8 位小数固化。
	if rows[0].FromCurrency != "USD" || rows[0].ARRate != "7.30000000" || rows[0].APRate != "7.10000000" || rows[0].Rate != "7.20000000" {
		t.Fatalf("汇率导入行规范化结果不正确: %#v", rows[0])
	}
	if rows[0].EffectiveFrom != "2026-08-24T00:00:00+08:00" || rows[0].EffectiveTo == nil || *rows[0].EffectiveTo != "2026-08-30T23:59:59+08:00" {
		t.Fatalf("生效周归一化结果不正确: %#v", rows[0])
	}
}

func TestNormalizeExchangeRateImportRowsMarksSameWeekDuplicates(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{
		{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.30", APRate: "7.10", Rate: "7.2", EffectiveFrom: "2026-08-25 09:00:00"},
		{RowNumber: 3, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.31", APRate: "7.11", Rate: "7.3", EffectiveFrom: "2026-08-27 11:59:59"},
	}, "CNY")
	// 同一自然周的两行归一化后为同周重复，整批标记不可确认。
	if rows[0].Status != ExchangeRateImportRowInvalid || rows[1].Status != ExchangeRateImportRowInvalid || len(rows[0].Errors) == 0 || len(rows[1].Errors) == 0 {
		t.Fatalf("同周重复的两行都必须标记错误: %#v", rows)
	}
}

func TestNormalizeExchangeRateImportRowsAllowsAdjacentWeeks(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{
		{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.30", APRate: "7.10", EffectiveFrom: "2026-08-25 09:00:00"},
		{RowNumber: 3, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.31", APRate: "7.11", EffectiveFrom: "2026-09-01 09:00:00"},
	}, "CNY")
	if rows[0].Status != ExchangeRateImportRowValid || rows[1].Status != ExchangeRateImportRowValid {
		t.Fatalf("相邻自然周行不应冲突: %#v", rows)
	}
}

func TestNormalizeExchangeRateImportRowsRequiresDualRates(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{
		{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", APRate: "7.10", EffectiveFrom: "2026-08-25 09:00:00"},
	}, "CNY")
	if rows[0].Status != ExchangeRateImportRowInvalid || len(rows[0].Errors) == 0 {
		t.Fatalf("缺少应收汇率的行必须标记错误: %#v", rows[0])
	}
}

func TestNormalizeExchangeRateImportRowsRejectsWrongBaseCurrency(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{{
		RowNumber: 2, FromCurrency: "USD", ToCurrency: "EUR",
		ARRate: "7.3", APRate: "7.1", EffectiveFrom: "2026-08-27 09:00:00",
	}}, "CNY")
	if rows[0].Status != ExchangeRateImportRowInvalid || len(rows[0].Errors) == 0 {
		t.Fatalf("非组织本币必须标记错误: %#v", rows[0])
	}
}

func TestPreviewExchangeRateImportPersistsNormalizedSnapshot(t *testing.T) {
	organizationID, actorID := uuid.New(), uuid.New()
	repo := &exchangeRateImportRepoStub{context: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo, nil)
	batch, token, err := usecase.PreviewImport(context.Background(), organizationID, actorID, PreviewExchangeRateImportInput{
		FileName: "汇率.xlsx", FileChecksum: strings.Repeat("a", 64), TemplateVersion: ExchangeRateImportTemplateVersion,
		Rows: []*ExchangeRateImportRow{{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.3", APRate: "7.1", Rate: "7.2", EffectiveFrom: "2026-08-27 09:30:01"}},
	})
	if err != nil {
		t.Fatalf("汇率导入预检失败: %v", err)
	}
	if token == "" || batch.Status != ExchangeRateImportPreviewReady || batch.ValidCount != 1 || batch.InvalidCount != 0 || batch.PreviewTokenHash == token || repo.created != batch {
		t.Fatalf("汇率预检批次不完整: %#v", batch)
	}
}

func TestPreviewExchangeRateImportMarksDatabaseConflict(t *testing.T) {
	organizationID := uuid.New()
	repo := &exchangeRateImportRepoStub{
		context:          &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		inspectionErrors: map[int][]string{2: {"原币或本币不是启用的 ISO 币种"}},
	}
	usecase := NewExchangeRateUsecase(repo, nil)
	batch, _, err := usecase.PreviewImport(context.Background(), organizationID, uuid.New(), PreviewExchangeRateImportInput{
		FileName: "汇率.xlsx", FileChecksum: strings.Repeat("b", 64), TemplateVersion: ExchangeRateImportTemplateVersion,
		Rows: []*ExchangeRateImportRow{{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.3", APRate: "7.1", EffectiveFrom: "2026-08-27 09:30:01"}},
	})
	if err != nil {
		t.Fatalf("存在业务错误的文件也应返回预检结果: %v", err)
	}
	if batch.Status != ExchangeRateImportPreviewInvalid || batch.InvalidCount != 1 || batch.Rows[0].Status != ExchangeRateImportRowInvalid {
		t.Fatalf("数据库冲突未写入预检行: %#v", batch)
	}
}

// TestPreviewExchangeRateImportAllowsBranchOrganization 验证导入按当前组织落地：
// 分公司导入不再被总部门禁拦截（总部导基线行、分公司导本组织行由确认导入按
// 组织身份判定）。
func TestPreviewExchangeRateImportAllowsBranchOrganization(t *testing.T) {
	branchID, ownerID := uuid.New(), uuid.New()
	repo := &exchangeRateImportRepoStub{context: &ExchangeRateContext{OwnerOrganizationID: ownerID, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo, nil)
	batch, _, err := usecase.PreviewImport(context.Background(), branchID, uuid.New(), PreviewExchangeRateImportInput{
		FileName: "汇率.xlsx", FileChecksum: strings.Repeat("c", 64), TemplateVersion: ExchangeRateImportTemplateVersion,
		Rows: []*ExchangeRateImportRow{{RowNumber: 2, FromCurrency: "USD", ToCurrency: "CNY", ARRate: "7.3", APRate: "7.1", EffectiveFrom: "2026-08-27 09:30:01"}},
	})
	if err != nil {
		t.Fatalf("分支机构导入汇率不应被拒绝: %v", err)
	}
	if batch == nil || batch.OrganizationID != branchID || repo.created == nil {
		t.Fatalf("分支机构导入应创建本组织预检批次: %#v", batch)
	}
}

func exchangeRateStringPointer(value string) *string { return &value }
