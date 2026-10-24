package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
)

// loadFeeApplicability 读取订单的异常案例适用范围，供费用项适用性复核使用；
// 与 orderFeeRepo.loadApplicability 同一口径，仅以显式客户端参数支持事务内调用。
func loadFeeApplicability(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*orderFeeApplicability, error) {
	exists, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrOrderFeeNotFound
	}
	abnormalCases, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(orderID), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &orderFeeApplicability{abnormalCaseIDs: make(map[uuid.UUID]struct{}, len(abnormalCases))}
	for _, abnormalCase := range abnormalCases {
		result.abnormalCaseIDs[abnormalCase.AbnormalCaseID] = struct{}{}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 专用作废：能力投影与 design 5.1 六步事务
// ---------------------------------------------------------------------------

// supplementCancelBlockReason 是专用作废阻断的稳定投影。
type supplementCancelBlockReason struct {
	code   string
	reason string
}

// evaluateCancelCapability 按专用作废前置条件只读评估能力：仅限 APPROVED 申请、
// 无活动账单行、无待审或更晚有效补录，且未产生系统冲减。列表与命令复用。
func uniqueSortedUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}
