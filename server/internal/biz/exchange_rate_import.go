package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrExchangeRateImportFileInvalid         = errors.BadRequest("EXCHANGE_RATE_IMPORT_FILE_INVALID", "汇率导入文件不合法")
	ErrExchangeRateImportEmpty               = errors.BadRequest("EXCHANGE_RATE_IMPORT_EMPTY", "汇率导入文件没有数据")
	ErrExchangeRateImportTooManyRows         = errors.BadRequest("EXCHANGE_RATE_IMPORT_TOO_MANY_ROWS", "汇率导入最多支持 500 条数据")
	ErrExchangeRateImportNotFound            = errors.NotFound("EXCHANGE_RATE_IMPORT_NOT_FOUND", "汇率导入预检不存在")
	ErrExchangeRateImportInvalid             = errors.Conflict("EXCHANGE_RATE_IMPORT_INVALID", "汇率导入预检存在错误，不能确认")
	ErrExchangeRateImportExpired             = errors.Conflict("EXCHANGE_RATE_IMPORT_EXPIRED", "汇率导入预检已过期，请重新上传")
	ErrExchangeRateImportIdempotencyConflict = errors.Conflict("EXCHANGE_RATE_IMPORT_IDEMPOTENCY_CONFLICT", "汇率导入幂等键已被其他请求使用")
	ErrExchangeRateImportStale               = errors.Conflict("EXCHANGE_RATE_IMPORT_STALE", "汇率数据已变化，请重新上传预检")
)

const (
	ExchangeRateImportTemplateVersion = 3
	ExchangeRateImportMaxRows         = 500
	ExchangeRateImportPreviewTTL      = 30 * time.Minute
	ExchangeRateImportPreviewReady    = "PREVIEW_READY"
	ExchangeRateImportPreviewInvalid  = "PREVIEW_INVALID"
	ExchangeRateImportImported        = "IMPORTED"
	ExchangeRateImportRowValid        = "VALID"
	ExchangeRateImportRowInvalid      = "INVALID"
)

type ExchangeRateImportRow struct {
	SettingID     uuid.UUID `json:"settingId"`
	RowNumber     int       `json:"rowNumber"`
	FromCurrency  string    `json:"fromCurrency"`
	ToCurrency    string    `json:"toCurrency"`
	ARRate        string    `json:"arRate"`
	APRate        string    `json:"apRate"`
	Rate          string    `json:"rate"`
	EffectiveFrom string    `json:"effectiveFrom"`
	EffectiveTo   *string   `json:"effectiveTo,omitempty"`
	Status        string    `json:"status"`
	Errors        []string  `json:"errors"`
}

type ExchangeRateImportBatch struct {
	ID, OrganizationID, OwnerOrganizationID, CreatedBy  uuid.UUID
	FileName, FileChecksum, Status, PreviewTokenHash    string
	TemplateVersion                                     int
	ExpiresAt                                           time.Time
	IdempotencyKey                                      *string
	TotalCount, ValidCount, InvalidCount, ImportedCount int
	Rows                                                []*ExchangeRateImportRow
	ImportedAt                                          *time.Time
	ImportedBy                                          *uuid.UUID
	CreatedAt, UpdatedAt                                time.Time
}

type PreviewExchangeRateImportInput struct {
	FileName, FileChecksum string
	TemplateVersion        int
	Rows                   []*ExchangeRateImportRow
}

func (uc *ExchangeRateUsecase) PreviewImport(ctx context.Context, organizationID, actorID uuid.UUID, input PreviewExchangeRateImportInput) (*ExchangeRateImportBatch, string, error) {
	input.FileName = strings.TrimSpace(input.FileName)
	input.FileChecksum = strings.ToLower(strings.TrimSpace(input.FileChecksum))
	if organizationID == uuid.Nil || actorID == uuid.Nil || input.FileName == "" || utf8.RuneCountInString(input.FileName) > 255 || len(input.FileChecksum) != 64 || input.TemplateVersion != ExchangeRateImportTemplateVersion {
		return nil, "", ErrExchangeRateImportFileInvalid
	}
	if len(input.Rows) == 0 {
		return nil, "", ErrExchangeRateImportEmpty
	}
	if len(input.Rows) > ExchangeRateImportMaxRows {
		return nil, "", ErrExchangeRateImportTooManyRows
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, "", err
	}
	// 汇率导入按当前组织落地：总部落 NULL 基线行，分公司落本组织行。
	rows := normalizeExchangeRateImportRows(input.Rows, rateContext.BaseCurrency)
	inspectionErrors, err := uc.repo.InspectImport(ctx, rateContext.OwnerOrganizationID, rows)
	if err != nil {
		return nil, "", err
	}
	for _, row := range rows {
		if messages := inspectionErrors[row.RowNumber]; len(messages) > 0 {
			row.Errors = appendUniqueStrings(row.Errors, messages...)
			row.Status = ExchangeRateImportRowInvalid
		}
	}
	token, tokenHash, err := newExchangeRateImportPreviewToken()
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	batch := &ExchangeRateImportBatch{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, OwnerOrganizationID: rateContext.OwnerOrganizationID,
		CreatedBy: actorID, FileName: input.FileName, FileChecksum: input.FileChecksum,
		TemplateVersion: input.TemplateVersion, Status: ExchangeRateImportPreviewReady,
		PreviewTokenHash: tokenHash, ExpiresAt: now.Add(ExchangeRateImportPreviewTTL), Rows: rows,
	}
	batch.TotalCount = len(rows)
	for _, row := range rows {
		if row.Status == ExchangeRateImportRowValid {
			batch.ValidCount++
		} else {
			batch.InvalidCount++
		}
	}
	if batch.InvalidCount > 0 {
		batch.Status = ExchangeRateImportPreviewInvalid
	}
	created, err := uc.repo.CreateImportPreview(ctx, batch, exchangeRateImportAudit(organizationID, actorID, batch.ID, "finance.exchange_rate.import.preview", input.FileChecksum))
	if err != nil {
		return nil, "", err
	}
	return created, token, nil
}

func (uc *ExchangeRateUsecase) ConfirmImport(ctx context.Context, organizationID, actorID uuid.UUID, previewToken, idempotencyKey string) (*ExchangeRateImportBatch, error) {
	previewToken = strings.TrimSpace(previewToken)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if organizationID == uuid.Nil || actorID == uuid.Nil || previewToken == "" || idempotencyKey == "" || utf8.RuneCountInString(idempotencyKey) > 128 {
		return nil, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	tokenHash := hashExchangeRateImportPreviewToken(previewToken)
	audit := exchangeRateImportAudit(organizationID, actorID, uuid.Nil, "finance.exchange_rate.import.confirm", "")
	return uc.repo.ConfirmImport(ctx, organizationID, rateContext.OwnerOrganizationID, actorID, tokenHash, idempotencyKey, time.Now().UTC(), audit)
}

func (uc *ExchangeRateUsecase) GetImport(ctx context.Context, organizationID, id uuid.UUID) (*ExchangeRateImportBatch, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	return uc.repo.GetImport(ctx, organizationID, id)
}

func normalizeExchangeRateImportRows(input []*ExchangeRateImportRow, baseCurrency string) []*ExchangeRateImportRow {
	rows := make([]*ExchangeRateImportRow, 0, len(input))
	for _, source := range input {
		if source == nil {
			continue
		}
		row := *source
		row.Errors = append([]string(nil), source.Errors...)
		row.FromCurrency = strings.ToUpper(strings.TrimSpace(row.FromCurrency))
		row.ToCurrency = strings.ToUpper(strings.TrimSpace(row.ToCurrency))
		row.ARRate = strings.TrimSpace(row.ARRate)
		row.APRate = strings.TrimSpace(row.APRate)
		row.Rate = strings.TrimSpace(row.Rate)
		row.EffectiveFrom = normalizeExchangeRateImportTime(row.EffectiveFrom)
		if row.EffectiveTo != nil {
			value := normalizeExchangeRateImportTime(*row.EffectiveTo)
			if value == "" {
				row.Errors = appendUniqueStrings(row.Errors, "生效结束时间格式不合法")
			} else {
				row.EffectiveTo = &value
			}
		}
		arRate, arErr := decimal.NewFromString(row.ARRate)
		if arErr != nil {
			row.Errors = appendUniqueStrings(row.Errors, "应收汇率（现汇卖出价）格式不合法")
		}
		apRate, apErr := decimal.NewFromString(row.APRate)
		if apErr != nil {
			row.Errors = appendUniqueStrings(row.Errors, "应付汇率（现汇买入价）格式不合法")
		}
		rate := decimal.Zero
		if row.Rate != "" {
			parsed, rateErr := decimal.NewFromString(row.Rate)
			if rateErr != nil {
				row.Errors = appendUniqueStrings(row.Errors, "基准汇率格式不合法")
			} else {
				rate = parsed
			}
		}
		if row.EffectiveFrom == "" {
			row.Errors = appendUniqueStrings(row.Errors, "生效开始时间格式不合法")
		}
		if row.ToCurrency != baseCurrency {
			row.Errors = appendUniqueStrings(row.Errors, "本币必须是当前组织本币 "+baseCurrency)
		}
		if arErr == nil && apErr == nil && row.EffectiveFrom != "" {
			setting := &ExchangeRateSetting{FromCurrency: row.FromCurrency, ToCurrency: row.ToCurrency, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, ARRate: &arRate, APRate: &apRate, Rate: rate}
			if normalized, err := normalizeExchangeRateSetting(setting); err != nil {
				row.Errors = appendUniqueStrings(row.Errors, "汇率字段或生效周不合法")
			} else {
				row.SettingID = uuid.Must(uuid.NewV7())
				row.FromCurrency, row.ToCurrency = normalized.FromCurrency, normalized.ToCurrency
				row.ARRate, row.APRate = normalized.ARRate.StringFixed(8), normalized.APRate.StringFixed(8)
				row.Rate = normalized.Rate.StringFixed(8)
				row.EffectiveFrom, row.EffectiveTo = normalized.EffectiveFrom, normalized.EffectiveTo
			}
		}
		row.Status = ExchangeRateImportRowValid
		if len(row.Errors) > 0 {
			row.Status = ExchangeRateImportRowInvalid
		}
		rows = append(rows, &row)
	}
	markExchangeRateImportInternalOverlaps(rows)
	return rows
}

func markExchangeRateImportInternalOverlaps(rows []*ExchangeRateImportRow) {
	for i := 0; i < len(rows); i++ {
		left := rows[i]
		if left.SettingID == uuid.Nil {
			continue
		}
		for j := i + 1; j < len(rows); j++ {
			right := rows[j]
			if right.SettingID == uuid.Nil || left.FromCurrency != right.FromCurrency || left.ToCurrency != right.ToCurrency {
				continue
			}
			if exchangeRateImportIntervalsOverlap(left, right) {
				left.Errors = appendUniqueStrings(left.Errors, "与 Excel 第 "+strconv.Itoa(right.RowNumber)+" 行生效区间重叠")
				right.Errors = appendUniqueStrings(right.Errors, "与 Excel 第 "+strconv.Itoa(left.RowNumber)+" 行生效区间重叠")
				left.Status, right.Status = ExchangeRateImportRowInvalid, ExchangeRateImportRowInvalid
			}
		}
	}
}

func exchangeRateImportIntervalsOverlap(left, right *ExchangeRateImportRow) bool {
	_, leftFrom, _ := normalizeExchangeRateTimestamp(left.EffectiveFrom)
	_, rightFrom, _ := normalizeExchangeRateTimestamp(right.EffectiveFrom)
	leftBeforeRightEnd := right.EffectiveTo == nil
	if right.EffectiveTo != nil {
		_, rightTo, _ := normalizeExchangeRateTimestamp(*right.EffectiveTo)
		leftBeforeRightEnd = leftFrom.Before(rightTo)
	}
	rightBeforeLeftEnd := left.EffectiveTo == nil
	if left.EffectiveTo != nil {
		_, leftTo, _ := normalizeExchangeRateTimestamp(*left.EffectiveTo)
		rightBeforeLeftEnd = rightFrom.Before(leftTo)
	}
	return leftBeforeRightEnd && rightBeforeLeftEnd
}

func normalizeExchangeRateImportTime(value string) string {
	value = strings.TrimSpace(value)
	if normalized, _, valid := normalizeExchangeRateTimestamp(value); valid {
		return normalized
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, exchangeRateBusinessLocation)
	if err != nil {
		return ""
	}
	return parsed.Format(time.RFC3339)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func newExchangeRateImportPreviewToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashExchangeRateImportPreviewToken(token), nil
}

func hashExchangeRateImportPreviewToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func exchangeRateImportAudit(organizationID, actorID, id uuid.UUID, action, checksum string) *AuditEvent {
	details := map[string]string{}
	if checksum != "" {
		details["exchange_rate_import.file_checksum"] = checksum
	}
	resourceID := ""
	if id != uuid.Nil {
		resourceID = id.String()
	}
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "exchange_rate_import_batch", ResourceID: resourceID, Details: details}
}
