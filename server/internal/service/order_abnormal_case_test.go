package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type abnormalCaseServiceRepoStub struct {
	biz.OrderAbnormalCaseRepo
	organizationID, orderID, actorID uuid.UUID
	result                           *biz.OrderAbnormalCase
}

func (s *abnormalCaseServiceRepoStub) Mark(_ context.Context, organizationID, orderID, actorID, kindID, newID uuid.UUID, _ *biz.AuditEvent) (*biz.OrderAbnormalCase, error) {
	s.organizationID, s.orderID, s.actorID = organizationID, orderID, actorID
	s.result.ID, s.result.AbnormalCaseID = newID, kindID
	return s.result, nil
}
func (s *abnormalCaseServiceRepoStub) Resolve(_ context.Context, organizationID, orderID, actorID, id uuid.UUID, _ *biz.AuditEvent) (*biz.OrderAbnormalCase, error) {
	s.organizationID, s.orderID, s.actorID = organizationID, orderID, actorID
	s.result.ID = id
	return s.result, nil
}

// 操作者必须来自会话，不能与路径订单 ID 互换；名称由仓储投影原样传给面板。
func TestOrderAbnormalCaseServiceActorAndOrder(t *testing.T) {
	org, orderID, actor, kindID, recordID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: actor, Organization: biz.Organization{ID: org}})
	stub := &abnormalCaseServiceRepoStub{result: &biz.OrderAbnormalCase{OrderID: orderID, AbnormalCaseName: "海关查验", MarkedAt: time.Now(), MarkedBy: actor}}
	service := NewOrderAbnormalCaseService(biz.NewOrderAbnormalCaseUsecase(stub))
	marked, err := service.MarkAbnormalCase(ctx, &v1.MarkAbnormalCaseRequest{OrderId: orderID.String(), AbnormalCaseId: kindID.String()})
	if err != nil || stub.organizationID != org || stub.orderID != orderID || stub.actorID != actor || marked.Data.AbnormalCaseName != "海关查验" {
		t.Fatalf("登记参数或名称错误: %+v %+v %v", stub, marked, err)
	}
	resolved, err := service.ResolveAbnormalCase(ctx, &v1.ResolveAbnormalCaseRequest{OrderId: orderID.String(), Id: recordID.String()})
	if err != nil || stub.organizationID != org || stub.orderID != orderID || stub.actorID != actor || resolved.Data.AbnormalCaseName != "海关查验" {
		t.Fatalf("解决参数或名称错误: %+v %+v %v", stub, resolved, err)
	}
}
