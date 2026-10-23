package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// lockOrdersForFinanceReversal 按主键顺序串行化订单与财务回退，不阻止锁定订单回退账务。
func lockOrdersForFinanceReversal(ctx context.Context, tx *ent.Tx, orderIDs []uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}
	_, err := tx.Order.Query().
		Where(orderent.IDIn(orderIDs...)).
		Order(ent.Asc(orderent.FieldID)).
		ForUpdate().
		All(ctx)
	return err
}
