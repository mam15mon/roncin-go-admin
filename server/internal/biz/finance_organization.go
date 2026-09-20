package biz

import "github.com/google/uuid"

// validFinanceOrganizationIDs 严格校验财务数据范围组织集合：非空、无 Nil UUID、
// 无重复。cashflow/invoice/verification/settlement 共用。
// 注意与 validFinanceBillOrganizationIDs 的宽松口径（允许重复）不同，两者语义
// 差异是历史契约，勿在提取重构时顺手统一。
func validFinanceOrganizationIDs(organizationIDs []uuid.UUID) bool {
	if len(organizationIDs) == 0 {
		return false
	}
	seen := make(map[uuid.UUID]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return false
		}
		if _, exists := seen[organizationID]; exists {
			return false
		}
		seen[organizationID] = struct{}{}
	}
	return true
}
