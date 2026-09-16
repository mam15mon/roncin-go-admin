package biz

import (
	"strings"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
)

func TestSeaHouseBillBatchNoDuplicateError(t *testing.T) {
	t.Run("携带冲突分单号的中文提示且 reason 稳定", func(t *testing.T) {
		err := SeaHouseBillBatchNoDuplicateError([]string{"HBL-001", "HBL/002"})
		if kratoserrors.FromError(err).Reason != ErrSeaHouseBillBatchNoDuplicate.Reason {
			t.Fatalf("动态批次排重错误 reason = %s，期望 %s", kratoserrors.FromError(err).Reason, ErrSeaHouseBillBatchNoDuplicate.Reason)
		}
		if !kratoserrors.IsConflict(err) {
			t.Fatalf("批次排重错误应为 409 冲突: %v", err)
		}
		if message := kratoserrors.FromError(err).Message; !strings.Contains(message, "HBL-001") || !strings.Contains(message, "HBL/002") {
			t.Fatalf("错误消息应包含冲突分单号: %s", message)
		}
	})

	t.Run("空冲突清单回退通用文案", func(t *testing.T) {
		err := SeaHouseBillBatchNoDuplicateError(nil)
		if kratoserrors.FromError(err).Message != ErrSeaHouseBillBatchNoDuplicate.Message {
			t.Fatalf("空清单应回退通用文案: %s", kratoserrors.FromError(err).Message)
		}
	})
}

func TestSeaMasterBillBatchRuleErrors(t *testing.T) {
	t.Run("批次强制分单错误为 400", func(t *testing.T) {
		if !kratoserrors.IsBadRequest(ErrSeaOrderBatchRequiresHouse) {
			t.Fatalf("ErrSeaOrderBatchRequiresHouse 应为 400")
		}
		if kratoserrors.FromError(ErrSeaOrderBatchRequiresHouse).Reason != "SEA_ORDER_BATCH_REQUIRES_HOUSE" {
			t.Fatalf("ErrSeaOrderBatchRequiresHouse reason 异常")
		}
	})

	t.Run("直单禁拼与成员退出错误为 409", func(t *testing.T) {
		for _, err := range []*kratoserrors.Error{ErrSeaMasterBillBatchDirectBlocked, ErrSeaDocumentBatchMemberExitBlocked} {
			if !kratoserrors.IsConflict(err) {
				t.Fatalf("%s 应为 409 冲突", kratoserrors.FromError(err).Reason)
			}
		}
	})
}
