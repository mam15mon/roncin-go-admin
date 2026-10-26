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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type masterDataOptionsRepoStub struct {
	biz.MasterDataRepo
	organizationID uuid.UUID
	options        biz.MasterDataListOptions
	calls          int
}

func (s *masterDataOptionsRepoStub) List(_ context.Context, organizationID uuid.UUID, options biz.MasterDataListOptions) (*biz.MasterDataList, error) {
	s.organizationID, s.options = organizationID, options
	s.calls++
	return &biz.MasterDataList{Items: []*biz.MasterDataItem{{ID: uuid.New(), Kind: biz.MasterDataKindAbnormalCase, Code: "INSPECTION", Name: "海关查验", Enabled: true}}, Total: 1, Page: options.Page, PageSize: options.PageSize}, nil
}

func TestMasterDataSearchOptionsService(t *testing.T) {
	org := uuid.New()
	// 普通公司员工无任何主数据管理读取权限，也能读取公共启用候选。
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}})
	stub := &masterDataOptionsRepoStub{}
	service := NewMasterDataService(biz.NewMasterDataUsecase(stub), nil, nil, nil)
	result, err := service.SearchOptions(ctx, &v1.SearchOptionsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE, Keyword: "  haiguan  "})
	if err != nil {
		t.Fatal(err)
	}
	if stub.organizationID != org || stub.options.Kind != biz.MasterDataKindAbnormalCase || stub.options.Keyword != "haiguan" || stub.options.Enabled == nil || !*stub.options.Enabled || result.Page != 1 || result.PageSize != 20 || result.Total != 1 || len(result.Data) != 1 || result.Data[0].Name != "海关查验" {
		t.Fatalf("公共候选范围/投影不符: options=%+v result=%+v", stub.options, result)
	}
	for _, kind := range []v1.MasterDataKind{v1.MasterDataKind_MASTER_DATA_KIND_UNSPECIFIED, v1.MasterDataKind(6), v1.MasterDataKind(999)} {
		before := stub.calls
		if _, err := service.SearchOptions(ctx, &v1.SearchOptionsRequest{Kind: kind}); !errors.Is(err, biz.ErrMasterDataInvalidKind) || stub.calls != before {
			t.Fatalf("非法kind未拒绝: %v %v", kind, err)
		}
	}
	for _, value := range [][2]int32{{0, 20}, {-1, 20}, {1, 0}, {1, -1}, {1, 201}} {
		before := stub.calls
		if _, err := service.SearchOptions(ctx, &v1.SearchOptionsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE, Page: proto.Int32(value[0]), PageSize: proto.Int32(value[1])}); !errors.Is(err, biz.ErrMasterDataInvalidArgument) || stub.calls != before {
			t.Fatalf("非法分页未拒绝: %v %v", value, err)
		}
	}
	for _, size := range []int32{1, 200} {
		result, err := service.SearchOptions(ctx, &v1.SearchOptionsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE, Page: proto.Int32(2), PageSize: proto.Int32(size)})
		if err != nil || result.Page != 2 || result.PageSize != size {
			t.Fatalf("合法分页被拒绝: %+v %v", result, err)
		}
	}
	if _, err := service.SearchOptions(t.Context(), &v1.SearchOptionsRequest{Kind: v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE}); err == nil {
		t.Fatal("无会话应拒绝")
	}
}

// 候选搜索和全量候选仅要求登录，管理列表仍保留主数据读取权限。
func TestMasterDataSearchOptionsAccessContract(t *testing.T) {
	descriptor := v1.File_masterdata_v1_masterdata_proto.Services().ByName("MasterDataService")
	for _, name := range []string{"ListOptions", "SearchOptions"} {
		method := descriptor.Methods().ByName(protoreflect.Name(name))
		options := method.Options().(*descriptorpb.MethodOptions)
		rule := proto.GetExtension(options, accessv1.E_Rule).(*accessv1.Rule)
		if rule.GetMode() != accessv1.AccessMode_ACCESS_MODE_AUTHENTICATED || rule.GetPermission() != "" {
			t.Fatalf("公共候选增加了管理权限: %s %+v", name, rule)
		}
	}
	options := descriptor.Methods().ByName("ListItems").Options().(*descriptorpb.MethodOptions)
	rule := proto.GetExtension(options, accessv1.E_Rule).(*accessv1.Rule)
	if rule.GetMode() != accessv1.AccessMode_ACCESS_MODE_PERMISSION || rule.GetPermission() != "system.master_data.item.read" {
		t.Fatalf("管理列表权限被改变: %+v", rule)
	}
}
