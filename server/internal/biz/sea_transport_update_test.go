package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestExecuteTransportExecutionUpdateWithoutExternalConfirmation(t *testing.T) {
	organizationID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()
	result := &SeaTransportExecutionUpdateResult{VersionID: uuid.New()}
	calls := 0
	repo := &mockSeaOrderChangeRepo{
		executeTransportExecutionUpdateFunc: func(_ context.Context, org, actor uuid.UUID, input *SeaTransportExecutionUpdateCommand, audit *AuditEvent) (*SeaTransportExecutionUpdateResult, error) {
			calls++
			if org != organizationID || actor != actorID || input.OrderID != orderID || input.Reason != "船期调整" || input.Input.VoyageNo != "V002" {
				t.Fatalf("共享航程调整参数错误: %+v", input)
			}
			if audit.Action != "order.sea.transport_execution.update" || audit.Details["reason"] != "船期调整" {
				t.Fatalf("删除外部确认后仍须记录操作审计: %+v", audit)
			}
			return result, nil
		},
	}
	usecase := NewSeaOrderChangeUsecase(repo, &mockTransactor{})
	command := SeaTransportExecutionUpdateCommand{
		OrderID: orderID, ExpectedTransportExecutionVersion: 3,
		Input:  &SeaTransportExecutionUpdateInput{VoyageNo: " V002 "},
		Reason: " 船期调整 ", IdempotencyKey: "voyage-update-without-confirmation",
	}
	got, err := usecase.ExecuteTransportExecutionUpdate(context.Background(), organizationID, actorID, &command)
	if err != nil || got != result || calls != 1 {
		t.Fatalf("无需外部确认应能执行共享航程调整: result=%+v, err=%v, calls=%d", got, err, calls)
	}

	for _, test := range []struct {
		name   string
		mutate func(*SeaTransportExecutionUpdateCommand)
	}{
		{"缺少原因", func(input *SeaTransportExecutionUpdateCommand) { input.Reason = " " }},
		{"缺少版本", func(input *SeaTransportExecutionUpdateCommand) { input.ExpectedTransportExecutionVersion = 0 }},
		{"缺少幂等键", func(input *SeaTransportExecutionUpdateCommand) { input.IdempotencyKey = " " }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := command
			test.mutate(&invalid)
			if _, err := usecase.ExecuteTransportExecutionUpdate(context.Background(), organizationID, actorID, &invalid); err != ErrSeaOrderReassignmentInvalidArgument {
				t.Fatalf("应保留必要参数校验: %v", err)
			}
		})
	}
	if calls != 1 {
		t.Fatalf("无效请求不得进入仓储: calls=%d", calls)
	}
}
