package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type transportUpdateRepoForService struct {
	mockSeaOrderChangeRepoForService
	input *biz.SeaTransportExecutionUpdateCommand
}

func (repo *transportUpdateRepoForService) ExecuteTransportExecutionUpdate(_ context.Context, _, _ uuid.UUID, input *biz.SeaTransportExecutionUpdateCommand, _ *biz.AuditEvent) (*biz.SeaTransportExecutionUpdateResult, error) {
	repo.input = input
	return &biz.SeaTransportExecutionUpdateResult{
		TransportExecution: &biz.SeaTransportExecution{ID: uuid.New(), Version: 4},
		VersionID:          uuid.New(),
	}, nil
}

func TestExecuteSeaTransportExecutionUpdateWithoutExternalConfirmation(t *testing.T) {
	repo := &transportUpdateRepoForService{}
	service := NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(repo, &mockTransactorForService{}))
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New()},
	})
	orderID := uuid.New()
	response, err := service.ExecuteSeaTransportExecutionUpdate(ctx, &v1.ExecuteSeaTransportExecutionUpdateRequest{
		OrderId: orderID.String(), ExpectedTransportExecutionVersion: 3,
		Input:  &v1.SeaTransportExecutionUpdateInput{VoyageNo: "V002"},
		Reason: "调整船期", IdempotencyKey: "transport-update-without-confirmation",
	})
	if err != nil {
		t.Fatalf("无外部确认的共享航程请求应成功: %v", err)
	}
	if response.TransportExecution == nil || response.VersionId == "" || repo.input == nil || repo.input.OrderID != orderID || repo.input.Input.VoyageNo != "V002" {
		t.Fatalf("共享航程请求或响应映射错误: response=%+v, input=%+v", response, repo.input)
	}
}
