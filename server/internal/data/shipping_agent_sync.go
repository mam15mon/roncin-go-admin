package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
)

const ShippingAgentSnapshotSource = "HGJ_SHIPPING_AGENTS_20261027"

// ShippingAgentSyncRecord 保留来源身份；外部整数 ID 只用于生成固定内部导入码。
type ShippingAgentSyncRecord struct {
	SourceID  int64  `json:"id"`
	Name      string `json:"value"`
	NameEN    string `json:"enValue"`
	SortOrder int    `json:"order"`
}

func (r ShippingAgentSyncRecord) Code() string { return fmt.Sprintf("HGJ-%d", r.SourceID) }

// ApplyShippingAgents 只补缺；唯一索引解决并发重复导入，已有行从不被更新或停用。
func (s *IndustryReferenceSyncStore) ApplyShippingAgents(ctx context.Context, rows []ShippingAgentSyncRecord) (int, error) {
	created := 0
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		builders := make([]*ent.MasterDataItemCreate, 0, len(rows))
		ids := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			id := uuid.Must(uuid.NewV7())
			ids = append(ids, id)
			builders = append(builders, tx.MasterDataItem.Create().SetID(id).
				SetKind(masterdataitem.KindShippingAgent).SetCode(row.Code()).
				SetName(row.Name).SetNameEn(row.NameEN).SetSortOrder(row.SortOrder).
				SetSource(ShippingAgentSnapshotSource).SetEnabled(true))
		}
		if len(builders) == 0 {
			return nil
		}
		if err := tx.MasterDataItem.CreateBulk(builders...).OnConflictColumns(masterdataitem.FieldKind, masterdataitem.FieldCode).DoNothing().Exec(ctx); err != nil {
			return fmt.Errorf("补齐船代公共主数据失败: %w", err)
		}
		var err error
		created, err = tx.MasterDataItem.Query().Where(masterdataitem.IDIn(ids...)).Count(ctx)
		return err
	})
	if err != nil {
		return 0, err
	}
	return created, nil
}
