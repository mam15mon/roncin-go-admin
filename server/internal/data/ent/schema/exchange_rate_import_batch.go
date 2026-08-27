package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateImportBatch 保存 Excel 预检快照和确认导入结果。
// 原始文件不落库，只保存摘要和规范化后的行数据。
type ExchangeRateImportBatch struct{ ent.Schema }

func (ExchangeRateImportBatch) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateImportBatch) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("owner_organization_id", uuid.Nil).Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
		field.String("file_name").NotEmpty().MaxLen(255).Immutable(),
		field.String("file_checksum").NotEmpty().MaxLen(64).Immutable(),
		field.Int("template_version").Positive().Immutable(),
		field.Enum("status").Values("PREVIEW_READY", "PREVIEW_INVALID", "IMPORTED"),
		field.String("preview_token_hash").NotEmpty().MaxLen(64).Immutable(),
		field.Time("expires_at").Immutable(),
		field.String("idempotency_key").Optional().Nillable().MaxLen(128),
		field.Int("total_count").NonNegative().Immutable(),
		field.Int("valid_count").NonNegative().Immutable(),
		field.Int("invalid_count").NonNegative().Immutable(),
		field.Int("imported_count").NonNegative().Default(0),
		field.JSON("rows", json.RawMessage{}).Immutable(),
		field.Time("imported_at").Optional().Nillable(),
		field.UUID("imported_by", uuid.Nil).Optional().Nillable(),
	}
}

func (ExchangeRateImportBatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("preview_token_hash").Unique().StorageKey("exchange_rate_import_preview_token"),
		index.Fields("organization_id", "id").StorageKey("exchange_rate_import_org_id"),
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("exchange_rate_import_idempotency"),
		index.Fields("organization_id", "created_at").StorageKey("exchange_rate_import_created_at"),
	}
}
