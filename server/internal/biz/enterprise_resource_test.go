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
	inputs := []*EnterpriseResource{
		{ResourceType: EnterpriseResourceShipperType, ShortName: "发货人一", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: "甲公司", BusinessCode: "ABC", CountryCode: "CN"}},
		{ResourceType: EnterpriseResourceShipperType, ShortName: "发货人二", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: "乙公司", BusinessCode: "abc", CountryCode: "CN"}},
	}
	_, errors := validateEnterpriseResourceImport(inputs, EnterpriseResourceShipperType)
	if errors[0] != nil || errors[1] != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("导入批次内业务代码判重结果错误: %v", errors)
	}
}

func TestPreviewEnterpriseResourceImportRejectsDuplicateCompanyName(t *testing.T) {
	inputs := []*EnterpriseResource{
		{ResourceType: EnterpriseResourceConsigneeType, ShortName: "收货人一", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: "甲公司", BusinessCode: "A001", CountryCode: "CN"}},
		{ResourceType: EnterpriseResourceConsigneeType, ShortName: "收货人二", Enabled: true, Party: &EnterpriseResourceParty{CompanyName: " 甲公司 ", BusinessCode: "A002", CountryCode: "CN"}},
	}
	_, errors := validateEnterpriseResourceImport(inputs, EnterpriseResourceConsigneeType)
	if errors[0] != nil || errors[1] != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("导入批次内企业名称判重结果错误: %v", errors)
	}
}

func TestEnterpriseResourceBatchRejectsEmptyTargets(t *testing.T) {
	usecase := &EnterpriseResourceUsecase{}
	_, err := usecase.BatchPartners(t.Context(), uuid.New(), uuid.New(), nil, []uuid.UUID{uuid.New()}, true)
	if err != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("空资源集合应被拒绝，实际为 %v", err)
	}
}

func TestNormalizeEnterpriseResourceValidatesImageTypeAndSize(t *testing.T) {
	valid := &EnterpriseResource{
		ResourceType: EnterpriseResourceImageType,
		ShortName:    "装箱照片",
		Enabled:      true,
		Image: &EnterpriseResourceImage{
			FileName: "packing.png", MIMEType: "image/png", FileSize: EnterpriseImageMaxFileSize,
			ObjectKey: "enterprise-resources/example/packing.png", Checksum: "checksum",
		},
	}
	if _, err := normalizeEnterpriseResource(valid); err != nil {
		t.Fatalf("合法图片应通过校验: %v", err)
	}

	invalidMIME := *valid
	invalidMIME.Image = &EnterpriseResourceImage{FileName: "document.pdf", MIMEType: "application/pdf", FileSize: 1024, ObjectKey: "object", Checksum: "checksum"}
	if _, err := normalizeEnterpriseResource(&invalidMIME); err != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("非图片 MIME 应被拒绝，实际为 %v", err)
	}

	oversized := *valid
	oversized.Image = &EnterpriseResourceImage{FileName: "large.png", MIMEType: "image/png", FileSize: EnterpriseImageMaxFileSize + 1, ObjectKey: "object", Checksum: "checksum"}
	if _, err := normalizeEnterpriseResource(&oversized); err != ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("超限图片应被拒绝，实际为 %v", err)
	}
}

func TestNormalizeEnterpriseResourceBuildsCompactPartyDisplayContent(t *testing.T) {
	result, err := normalizeEnterpriseResource(&EnterpriseResource{
		ResourceType: EnterpriseResourceShipperType,
		ShortName:    "测试发货人",
		Enabled:      true,
		Party: &EnterpriseResourceParty{
			CompanyName:  " 测试企业 ",
			CountryCode:  "cn",
			ContactPhone: " 13800000000 ",
		},
	})
	if err != nil {
		t.Fatalf("规范化单证主体失败: %v", err)
	}
	if result.Party.DisplayContent != "测试企业\n13800000000" {
		t.Fatalf("自动展示内容存在多余空行: %q", result.Party.DisplayContent)
	}
}

func TestEnterpriseResourceBatchAuditHasNoZeroResourceID(t *testing.T) {
	audit := enterpriseResourceAudit(uuid.New(), uuid.New(), "enterprise_resource.import", uuid.Nil, EnterpriseResourceShipperType)
	if audit.ResourceID != "" {
		t.Fatalf("批量审计不应写入零 UUID: %q", audit.ResourceID)
	}
}
