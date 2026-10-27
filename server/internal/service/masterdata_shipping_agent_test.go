package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	accessv1 "github.com/roncin/roncin-go-admin/server/api/access/v1"
	v1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type shippingAgentAllRepoStub struct {
	biz.MasterDataRepo
	kind    biz.MasterDataKind
	enabled *bool
	calls   int
}

func (s *shippingAgentAllRepoStub) ListAll(_ context.Context, kind biz.MasterDataKind, enabled *bool) ([]*biz.MasterDataItem, error) {
	s.kind, s.enabled = kind, enabled
	s.calls++
	items := make([]*biz.MasterDataItem, 205)
	for i := range items {
		items[i] = &biz.MasterDataItem{ID: uuid.New(), Kind: biz.MasterDataKindShippingAgent, Code: "AGENT", Name: "船代", Enabled: i%2 == 0}
	}
	return items, nil
}

func TestMasterDataShippingAgentListAllService(t *testing.T) {
	stub := &shippingAgentAllRepoStub{}
	svc := NewMasterDataService(biz.NewMasterDataUsecase(stub), nil, nil, nil)
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New(), Kind: biz.OrganizationKindCompany}})
	result, err := svc.ListAllItems(ctx, &v1.ListAllItemsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_SHIPPING_AGENT})
	if err != nil || len(result.Data) != 205 || result.Total != 205 || stub.kind != biz.MasterDataKindShippingAgent || stub.enabled != nil || result.Data[0].Kind != v1.MasterDataKind_MASTER_DATA_KIND_SHIPPING_AGENT {
		t.Fatalf("全量转换错误: %+v %v", result, err)
	}
	disabled := false
	if _, err := svc.ListAllItems(ctx, &v1.ListAllItemsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_SHIPPING_AGENT, Enabled: &disabled}); err != nil || stub.enabled == nil || *stub.enabled {
		t.Fatalf("停用筛选丢失: %v", err)
	}
	for _, kind := range []v1.MasterDataKind{v1.MasterDataKind_MASTER_DATA_KIND_UNSPECIFIED, v1.MasterDataKind(999)} {
		before := stub.calls
		if _, err := svc.ListAllItems(ctx, &v1.ListAllItemsRequest{Kind: kind}); !errors.Is(err, biz.ErrMasterDataInvalidKind) || stub.calls != before {
			t.Fatalf("非法kind应拒绝: %v", err)
		}
	}
	if _, err := svc.ListAllItems(t.Context(), &v1.ListAllItemsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_SHIPPING_AGENT}); err == nil {
		t.Fatal("无会话应拒绝")
	}
	method := v1.File_masterdata_v1_masterdata_proto.Services().ByName("MasterDataService").Methods().ByName("ListAllItems")
	rule := proto.GetExtension(method.Options().(*descriptorpb.MethodOptions), accessv1.E_Rule).(*accessv1.Rule)
	if rule.GetMode() != accessv1.AccessMode_ACCESS_MODE_PERMISSION || rule.GetPermission() != "system.master_data.item.read" {
		t.Fatalf("全量维护接口权限错误: %+v", rule)
	}
	fields := (&v1.ListAllItemsRequest{}).ProtoReflect().Descriptor().Fields()
	if fields.ByName("page") != nil || fields.ByName("page_size") != nil {
		t.Fatal("全量接口不应存在分页参数")
	}
}
