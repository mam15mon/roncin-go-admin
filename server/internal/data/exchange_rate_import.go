package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	importent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangerateimportbatch"
	exchangerateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
)

func (r *exchangeRateRepo) InspectImport(ctx context.Context, ownerOrganizationID uuid.UUID, rows []*biz.ExchangeRateImportRow) (map[int][]string, error) {
	errorsByRow := make(map[int][]string)
	validCurrencies, err := r.enabledExchangeRateCurrencies(ctx, rows)
	if err != nil {
		return nil, err
	}
	existing, err := r.data.db.ExchangeRateSetting.Query().Where(exchangerateent.OrganizationIDEQ(ownerOrganizationID), exchangerateent.IsActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil || row.Status != biz.ExchangeRateImportRowValid {
			continue
		}
		if !validCurrencies[row.FromCurrency] || !validCurrencies[row.ToCurrency] {
			errorsByRow[row.RowNumber] = append(errorsByRow[row.RowNumber], "原币或本币不是启用的 ISO 币种")
			continue
		}
		if exchangeRateImportOverlapsExisting(row, existing) {
			errorsByRow[row.RowNumber] = append(errorsByRow[row.RowNumber], "生效区间与现有启用汇率重叠")
		}
	}
	return errorsByRow, nil
}

func (r *exchangeRateRepo) CreateImportPreview(ctx context.Context, batch *biz.ExchangeRateImportBatch, audit *biz.AuditEvent) (*biz.ExchangeRateImportBatch, error) {
	rows, err := json.Marshal(batch.Rows)
	if err != nil {
		return nil, err
	}
	var item *ent.ExchangeRateImportBatch
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		item, saveErr = tx.ExchangeRateImportBatch.Create().
			SetID(batch.ID).
			SetOrganizationID(batch.OrganizationID).
			SetOwnerOrganizationID(batch.OwnerOrganizationID).
			SetCreatedBy(batch.CreatedBy).
			SetFileName(batch.FileName).
			SetFileChecksum(batch.FileChecksum).
			SetTemplateVersion(batch.TemplateVersion).
			SetStatus(importent.Status(batch.Status)).
			SetPreviewTokenHash(batch.PreviewTokenHash).
			SetExpiresAt(batch.ExpiresAt).
			SetTotalCount(batch.TotalCount).
			SetValidCount(batch.ValidCount).
			SetInvalidCount(batch.InvalidCount).
			SetRows(rows).
			Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return exchangeRateImportBatchToBiz(item)
}

func (r *exchangeRateRepo) GetImport(ctx context.Context, organizationID, id uuid.UUID) (*biz.ExchangeRateImportBatch, error) {
	item, err := r.data.db.ExchangeRateImportBatch.Query().Where(importent.IDEQ(id), importent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrExchangeRateImportNotFound, nil)
	}
	return exchangeRateImportBatchToBiz(item)
}

func (r *exchangeRateRepo) ConfirmImport(ctx context.Context, organizationID, ownerOrganizationID, actorID uuid.UUID, previewTokenHash, idempotencyKey string, now time.Time, audit *biz.AuditEvent) (*biz.ExchangeRateImportBatch, error) {
	preview, err := r.data.db.ExchangeRateImportBatch.Query().Where(
		importent.OrganizationIDEQ(organizationID), importent.OwnerOrganizationIDEQ(ownerOrganizationID),
		importent.CreatedByEQ(actorID), importent.PreviewTokenHashEQ(previewTokenHash),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrExchangeRateImportNotFound, nil)
	}
	if preview.Status == importent.StatusIMPORTED {
		if preview.IdempotencyKey != nil && *preview.IdempotencyKey == idempotencyKey {
			return exchangeRateImportBatchToBiz(preview)
		}
		return nil, biz.ErrExchangeRateImportIdempotencyConflict
	}
	if preview.Status != importent.StatusPREVIEW_READY || preview.InvalidCount != 0 {
		return nil, biz.ErrExchangeRateImportInvalid
	}
	if !preview.ExpiresAt.After(now) {
		return nil, biz.ErrExchangeRateImportExpired
	}
	rows, err := exchangeRateImportRowsFromJSON(preview.Rows)
	if err != nil {
		return nil, err
	}
	lockKeys := exchangeRateImportLockKeys(ownerOrganizationID, rows)
	connection, err := r.data.sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	locked := make([]string, 0, len(lockKeys))
	for _, key := range lockKeys {
		if _, err = connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext($1))", key); err != nil {
			unlockExchangeRateImportKeys(connection, locked)
			return nil, err
		}
		locked = append(locked, key)
	}
	defer unlockExchangeRateImportKeys(connection, locked)

	var updated *ent.ExchangeRateImportBatch
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.ExchangeRateImportBatch.Query().Where(importent.IDEQ(preview.ID), importent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		if current.Status == importent.StatusIMPORTED {
			if current.IdempotencyKey != nil && *current.IdempotencyKey == idempotencyKey {
				updated = current
				return nil
			}
			return biz.ErrExchangeRateImportIdempotencyConflict
		}
		if current.Status != importent.StatusPREVIEW_READY || current.InvalidCount != 0 {
			return biz.ErrExchangeRateImportInvalid
		}
		if !current.ExpiresAt.After(now) {
			return biz.ErrExchangeRateImportExpired
		}
		idempotencyUsed, queryErr := tx.ExchangeRateImportBatch.Query().Where(importent.OrganizationIDEQ(organizationID), importent.IDNEQ(current.ID), importent.IdempotencyKeyEQ(idempotencyKey)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if idempotencyUsed {
			return biz.ErrExchangeRateImportIdempotencyConflict
		}
		rows, queryErr = exchangeRateImportRowsFromJSON(current.Rows)
		if queryErr != nil {
			return queryErr
		}
		if len(rows) != current.TotalCount || current.ValidCount != current.TotalCount {
			return biz.ErrExchangeRateImportStale
		}
		if validateErr := validateExchangeRateImportRowsInTx(ctx, tx, ownerOrganizationID, rows); validateErr != nil {
			return validateErr
		}
		builders := make([]*ent.ExchangeRateSettingCreate, 0, len(rows))
		for _, row := range rows {
			effectiveFrom, parseErr := parseExchangeRateStorageTime(row.EffectiveFrom)
			if parseErr != nil {
				return biz.ErrExchangeRateImportStale
			}
			builder := tx.ExchangeRateSetting.Create().SetID(row.SettingID).SetOrganizationID(ownerOrganizationID).
				SetRateType(exchangerateent.RateType(row.RateType)).SetFromCurrency(row.FromCurrency).SetToCurrency(row.ToCurrency).
				SetEffectiveFrom(effectiveFrom).SetReceivableRate(row.ReceivableRate).SetPayableRate(row.PayableRate).SetIsActive(true)
			if row.EffectiveTo != nil {
				effectiveTo, parseErr := parseExchangeRateStorageTime(*row.EffectiveTo)
				if parseErr != nil {
					return biz.ErrExchangeRateImportStale
				}
				builder.SetEffectiveTo(effectiveTo)
			}
			builders = append(builders, builder)
		}
		if _, saveErr := tx.ExchangeRateSetting.CreateBulk(builders...).Save(ctx); saveErr != nil {
			return biz.ErrExchangeRateImportStale
		}
		var saveErr error
		updated, saveErr = tx.ExchangeRateImportBatch.UpdateOneID(current.ID).
			SetStatus(importent.StatusIMPORTED).
			SetIdempotencyKey(idempotencyKey).
			SetImportedCount(len(rows)).
			SetImportedAt(now).
			SetImportedBy(actorID).
			Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrExchangeRateImportIdempotencyConflict)
		}
		audit.ResourceID = current.ID.String()
		audit.Details = map[string]string{"exchange_rate_import.file_checksum": current.FileChecksum, "exchange_rate_import.imported_count": fmt.Sprintf("%d", len(rows))}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return exchangeRateImportBatchToBiz(updated)
}

func (r *exchangeRateRepo) enabledExchangeRateCurrencies(ctx context.Context, rows []*biz.ExchangeRateImportRow) (map[string]bool, error) {
	codes := make(map[string]struct{})
	for _, row := range rows {
		if row != nil {
			codes[row.FromCurrency], codes[row.ToCurrency] = struct{}{}, struct{}{}
		}
	}
	values := make([]string, 0, len(codes))
	for code := range codes {
		if code != "" {
			values = append(values, code)
		}
	}
	items, err := r.data.db.Currency.Query().Where(currencyent.CodeIn(values...), currencyent.EnabledEQ(true)).Select(currencyent.FieldCode).Strings(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(items))
	for _, code := range items {
		result[code] = true
	}
	return result, nil
}

func validateExchangeRateImportRowsInTx(ctx context.Context, tx *ent.Tx, ownerOrganizationID uuid.UUID, rows []*biz.ExchangeRateImportRow) error {
	codes := make(map[string]struct{})
	for _, row := range rows {
		if row == nil || row.Status != biz.ExchangeRateImportRowValid || row.SettingID == uuid.Nil {
			return biz.ErrExchangeRateImportStale
		}
		codes[row.FromCurrency], codes[row.ToCurrency] = struct{}{}, struct{}{}
	}
	currencyCodes := make([]string, 0, len(codes))
	for code := range codes {
		currencyCodes = append(currencyCodes, code)
	}
	count, err := tx.Currency.Query().Where(currencyent.CodeIn(currencyCodes...), currencyent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(currencyCodes) {
		return biz.ErrExchangeRateImportStale
	}
	for _, row := range rows {
		effectiveFrom, _ := parseExchangeRateStorageTime(row.EffectiveFrom)
		query := tx.ExchangeRateSetting.Query().Where(
			exchangerateent.OrganizationIDEQ(ownerOrganizationID), exchangerateent.RateTypeEQ(exchangerateent.RateType(row.RateType)),
			exchangerateent.FromCurrencyEQ(row.FromCurrency), exchangerateent.ToCurrencyEQ(row.ToCurrency), exchangerateent.IsActiveEQ(true),
			exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(effectiveFrom)),
		)
		if row.EffectiveTo != nil {
			effectiveTo, _ := parseExchangeRateStorageTime(*row.EffectiveTo)
			query.Where(exchangerateent.EffectiveFromLT(effectiveTo))
		}
		overlap, queryErr := query.Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if overlap {
			return biz.ErrExchangeRateImportStale
		}
	}
	return nil
}

func exchangeRateImportOverlapsExisting(row *biz.ExchangeRateImportRow, existing []*ent.ExchangeRateSetting) bool {
	rowFrom, err := parseExchangeRateStorageTime(row.EffectiveFrom)
	if err != nil {
		return false
	}
	var rowTo *time.Time
	if row.EffectiveTo != nil {
		value, parseErr := parseExchangeRateStorageTime(*row.EffectiveTo)
		if parseErr != nil {
			return false
		}
		rowTo = &value
	}
	for _, current := range existing {
		if string(current.RateType) != row.RateType || current.FromCurrency != row.FromCurrency || current.ToCurrency != row.ToCurrency {
			continue
		}
		rowBeforeCurrentEnd := current.EffectiveTo == nil || rowFrom.Before(*current.EffectiveTo)
		currentBeforeRowEnd := rowTo == nil || current.EffectiveFrom.Before(*rowTo)
		if rowBeforeCurrentEnd && currentBeforeRowEnd {
			return true
		}
	}
	return false
}

func exchangeRateImportLockKeys(ownerOrganizationID uuid.UUID, rows []*biz.ExchangeRateImportRow) []string {
	set := make(map[string]struct{})
	for _, row := range rows {
		if row != nil {
			key := fmt.Sprintf("exchange-rate:%s:%s:%s:%s", ownerOrganizationID, row.RateType, row.FromCurrency, row.ToCurrency)
			set[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func unlockExchangeRateImportKeys(connection *sql.Conn, keys []string) {
	for index := len(keys) - 1; index >= 0; index-- {
		_, _ = connection.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", keys[index])
	}
}

func exchangeRateImportRowsFromJSON(raw json.RawMessage) ([]*biz.ExchangeRateImportRow, error) {
	var rows []*biz.ExchangeRateImportRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func exchangeRateImportBatchToBiz(item *ent.ExchangeRateImportBatch) (*biz.ExchangeRateImportBatch, error) {
	rows, err := exchangeRateImportRowsFromJSON(item.Rows)
	if err != nil {
		return nil, err
	}
	return &biz.ExchangeRateImportBatch{
		ID: item.ID, OrganizationID: item.OrganizationID, OwnerOrganizationID: item.OwnerOrganizationID, CreatedBy: item.CreatedBy,
		FileName: item.FileName, FileChecksum: item.FileChecksum, TemplateVersion: item.TemplateVersion, Status: string(item.Status), PreviewTokenHash: item.PreviewTokenHash,
		ExpiresAt: item.ExpiresAt, IdempotencyKey: item.IdempotencyKey, TotalCount: item.TotalCount, ValidCount: item.ValidCount,
		InvalidCount: item.InvalidCount, ImportedCount: item.ImportedCount, Rows: rows, ImportedAt: item.ImportedAt, ImportedBy: item.ImportedBy,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}
