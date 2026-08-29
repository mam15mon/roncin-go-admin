package biz

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeEnterpriseResourceAcceptsIndependentAddress(t *testing.T) {
	input := &EnterpriseResource{
		ResourceType: EnterpriseResourceAddressType,
		ShortName:    " 临时仓库 ",
		Enabled:      true,
		Address:      &EnterpriseResourceAddress{CountryCode: "cn", AddressDetail: " 浦东新区 "},
	}
	result, err := normalizeEnterpriseResource(input)
	if err != nil {
		t.Fatalf("规范化独立地址失败: %v", err)
	}
	if result.ShortName != "临时仓库" || result.Address.CountryCode != "CN" || result.Address.AddressDetail != "浦东新区" {
		t.Fatalf("规范化结果不符合预期: %+v", result)
	}
	if result.PartnerIDs != nil {
		t.Fatalf("独立地址不应自动生成企业关联: %v", result.PartnerIDs)
	}
}

func TestNormalizeEnterpriseResourceRejectsMismatchedDetail(t *testing.T) {
	_, err := normalizeEnterpriseResource(&EnterpriseResource{
		ResourceType: EnterpriseResourceRemarkType,
		ShortName:    "错误载荷",
		Address:      &EnterpriseResourceAddress{CountryCode: "CN", AddressDetail: "测试"},
	})
	if err != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("分类与详情不匹配时应返回参数错误，实际为 %v", err)
	}
}

func TestPreviewEnterpriseResourceImportRejectsDuplicateBusinessCode(t *testing.T) {
	usecase := &EnterpriseResourceUsecase{}
	inputs := []*EnterpriseResource{
		{ResourceType: EnterpriseResourceShipperType, ShortName: "发货人一", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: "甲公司", BusinessCode: "ABC", CountryCode: "CN"}},
		{ResourceType: EnterpriseResourceShipperType, ShortName: "发货人二", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: "乙公司", BusinessCode: "abc", CountryCode: "CN"}},
	}
	_, errors := usecase.PreviewImport(inputs, EnterpriseResourceShipperType)
	if errors[0] != nil || errors[1] != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("导入批次内业务代码判重结果错误: %v", errors)
	}
}

func TestEnterpriseResourceBatchRejectsEmptyTargets(t *testing.T) {
	usecase := &EnterpriseResourceUsecase{}
	_, err := usecase.BatchPartners(t.Context(), uuid.New(), uuid.New(), nil, []uuid.UUID{uuid.New()}, true)
	if err != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("空资源集合应被拒绝，实际为 %v", err)
	}
}
