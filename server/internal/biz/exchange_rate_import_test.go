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
		RowNumber: 2, RateType: "账单汇率", FromCurrency: " usd ", ToCurrency: "cny",
		ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:30:01", EffectiveTo: exchangeRateStringPointer("2026-08-27 18:00:00"),
	}}, "CNY")
	if len(rows) != 1 || rows[0].Status != ExchangeRateImportRowValid || rows[0].SettingID == uuid.Nil {
		t.Fatalf("合法秒级汇率行应通过规范化: %#v", rows)
	}
	if rows[0].RateType != BillRateType || rows[0].FromCurrency != "USD" || rows[0].EffectiveFrom != "2026-08-27T09:30:01+08:00" || rows[0].EffectiveTo == nil || *rows[0].EffectiveTo != "2026-08-27T18:00:00+08:00" {
		t.Fatalf("汇率导入行规范化结果不正确: %#v", rows[0])
	}
}

func TestNormalizeExchangeRateImportRowsMarksAllInternalOverlaps(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{
		{RowNumber: 2, RateType: "开票汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:00:00", EffectiveTo: exchangeRateStringPointer("2026-08-27 12:00:00")},
		{RowNumber: 3, RateType: "开票汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.3", PayableRate: "7.2", EffectiveFrom: "2026-08-27 11:59:59", EffectiveTo: exchangeRateStringPointer("2026-08-27 15:00:00")},
	}, "CNY")
	if rows[0].Status != ExchangeRateImportRowInvalid || rows[1].Status != ExchangeRateImportRowInvalid || len(rows[0].Errors) == 0 || len(rows[1].Errors) == 0 {
		t.Fatalf("重叠的两行都必须标记错误: %#v", rows)
	}
}

func TestNormalizeExchangeRateImportRowsAllowsAdjacentIntervals(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{
		{RowNumber: 2, RateType: "核销汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:00:00", EffectiveTo: exchangeRateStringPointer("2026-08-27 12:00:00")},
		{RowNumber: 3, RateType: "核销汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.3", PayableRate: "7.2", EffectiveFrom: "2026-08-27 12:00:00", EffectiveTo: exchangeRateStringPointer("2026-08-27 15:00:00")},
	}, "CNY")
	if rows[0].Status != ExchangeRateImportRowValid || rows[1].Status != ExchangeRateImportRowValid {
		t.Fatalf("左闭右开相邻区间不应冲突: %#v", rows)
	}
}

func TestNormalizeExchangeRateImportRowsRejectsWrongBaseCurrency(t *testing.T) {
	rows := normalizeExchangeRateImportRows([]*ExchangeRateImportRow{{
		RowNumber: 2, RateType: "结算汇率", FromCurrency: "USD", ToCurrency: "EUR",
		ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:00:00",
	}}, "CNY")
	if rows[0].Status != ExchangeRateImportRowInvalid || len(rows[0].Errors) == 0 {
		t.Fatalf("非组织本币必须标记错误: %#v", rows[0])
	}
}

func TestPreviewExchangeRateImportPersistsNormalizedSnapshot(t *testing.T) {
	organizationID, ownerID, actorID := uuid.New(), uuid.New(), uuid.New()
	repo := &exchangeRateImportRepoStub{context: &ExchangeRateContext{OwnerOrganizationID: ownerID, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo)
	batch, token, err := usecase.PreviewImport(context.Background(), organizationID, actorID, PreviewExchangeRateImportInput{
		FileName: "汇率.xlsx", FileChecksum: strings.Repeat("a", 64), TemplateVersion: ExchangeRateImportTemplateVersion,
		Rows: []*ExchangeRateImportRow{{RowNumber: 2, RateType: "账单汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:30:01"}},
	})
	if err != nil {
		t.Fatalf("汇率导入预检失败: %v", err)
	}
	if token == "" || batch.Status != ExchangeRateImportPreviewReady || batch.ValidCount != 1 || batch.InvalidCount != 0 || batch.PreviewTokenHash == token || repo.created != batch {
		t.Fatalf("汇率预检批次不完整: %#v", batch)
	}
}

func TestPreviewExchangeRateImportMarksDatabaseConflict(t *testing.T) {
	repo := &exchangeRateImportRepoStub{
		context:          &ExchangeRateContext{OwnerOrganizationID: uuid.New(), BaseCurrency: "CNY"},
		inspectionErrors: map[int][]string{2: {"生效区间与现有启用汇率重叠"}},
	}
	usecase := NewExchangeRateUsecase(repo)
	batch, _, err := usecase.PreviewImport(context.Background(), uuid.New(), uuid.New(), PreviewExchangeRateImportInput{
		FileName: "汇率.xlsx", FileChecksum: strings.Repeat("b", 64), TemplateVersion: ExchangeRateImportTemplateVersion,
		Rows: []*ExchangeRateImportRow{{RowNumber: 2, RateType: "账单汇率", FromCurrency: "USD", ToCurrency: "CNY", ReceivableRate: "7.2", PayableRate: "7.1", EffectiveFrom: "2026-08-27 09:30:01"}},
	})
	if err != nil {
		t.Fatalf("存在业务错误的文件也应返回预检结果: %v", err)
	}
	if batch.Status != ExchangeRateImportPreviewInvalid || batch.InvalidCount != 1 || batch.Rows[0].Status != ExchangeRateImportRowInvalid {
		t.Fatalf("数据库冲突未写入预检行: %#v", batch)
	}
}

func exchangeRateStringPointer(value string) *string { return &value }
