package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

func TestOrderDraftRequiresValidScheduleBeforeWriting(t *testing.T) {
	for _, operation := range []string{"创建", "更新草稿"} {
		t.Run(operation, func(t *testing.T) {
			for _, testCase := range []struct {
				name    string
				etd     string
				eta     string
				message string
			}{
				{name: "缺少ETD", eta: "2026-09-25T00:00:00Z", message: "请填写 ETD（预计离港时间）"},
				{name: "缺少ETA", etd: "2026-09-10T00:00:00Z", message: "请填写 ETA（预计到港时间）"},
				{name: "两者均缺少", message: "请填写 ETD（预计离港时间）"},
				{name: "ETD仅空白", etd: " \t ", eta: "2026-09-25T00:00:00Z", message: "请填写 ETD（预计离港时间）"},
				{name: "ETA仅空白", etd: "2026-09-10T00:00:00Z", eta: " \t ", message: "请填写 ETA（预计到港时间）"},
				{name: "ETD格式无效", etd: "2026-09-10", eta: "2026-09-25T00:00:00Z", message: "ETD（预计离港时间）格式不合法"},
				{name: "ETA日期无效", etd: "2026-09-10T00:00:00Z", eta: "2026-09-31T00:00:00Z", message: "ETA（预计到港时间）格式不合法"},
				{name: "ETA早于ETD", etd: "2026-09-10T00:00:00Z", eta: "2026-09-09T23:59:59Z", message: "ETA（预计到港时间）不能早于 ETD（预计离港时间）"},
				{name: "按实际时刻拒绝跨时区倒序", etd: "2026-09-10T00:00:00Z", eta: "2026-09-10T07:59:59+08:00", message: "ETA（预计到港时间）不能早于 ETD（预计离港时间）"},
				{name: "相同时间允许", etd: "2026-09-10T00:00:00Z", eta: "2026-09-10T00:00:00Z"},
				{name: "跨时区相同时刻允许", etd: "2026-09-10T00:00:00Z", eta: "2026-09-10T08:00:00+08:00"},
				{name: "有效船期并去空白", etd: " 2026-09-10T00:00:00Z ", eta: " 2026-09-25T00:00:00Z "},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					organizationID := uuid.New()
					input := idempotentCreateInput(organizationID)
					input.IdempotencyKey = ""
					input.ETD, input.ETA = testCase.etd, testCase.eta
					repo := &orderRepoStub{}
					usecase := NewOrderUsecase(repo, nil, &seaMasterBillRepoStub{}, nil, newReminderModeCreditControl(), nil)
					var saved *Order
					var err error
					if operation == "创建" {
						saved, err = usecase.Create(context.Background(), organizationID, uuid.New(), input)
					} else {
						input.SeaDocumentInput = nil
						saved, err = usecase.UpdateDraft(context.Background(), organizationID, uuid.New(), uuid.New(), 1, input)
					}
					if testCase.message != "" {
						if err == nil || errors.Code(err) != 400 || errors.Reason(err) != "ORDER_INVALID_ARGUMENT" || errors.FromError(err).Message != testCase.message {
							t.Fatalf("船期校验错误 = %v，期望 400 ORDER_INVALID_ARGUMENT: %s", err, testCase.message)
						}
						if saved != nil || repo.created != nil || repo.createdAudit != nil || repo.updatedAudit != nil {
							t.Fatal("船期不合法时不得写入订单仓储")
						}
						return
					}
					if err != nil || saved == nil {
						t.Fatalf("有效船期保存失败: %v", err)
					}
					if saved.ETD != strings.TrimSpace(testCase.etd) || saved.ETA != strings.TrimSpace(testCase.eta) {
						t.Fatalf("船期未正确保存或去空白: ETD=%q，ETA=%q", saved.ETD, saved.ETA)
					}
					if operation == "创建" && repo.createdAudit == nil || operation == "更新草稿" && repo.updatedAudit == nil {
						t.Fatal("有效船期应进入对应的仓储写入路径")
					}
				})
			}
		})
	}
}
