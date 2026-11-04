package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCompanyNameEnCreateAndUpdateValidation(t *testing.T) {
	cases := []struct {
		name    string
		kind    OrganizationKind
		value   string
		want    string
		invalid bool
	}{
		{name: "保持正式大小写和内部空格", kind: OrganizationKindCompany, value: "  Roncin Example CO., LTD  ", want: "Roncin Example CO., LTD"},
		{name: "可空", kind: OrganizationKindCompany, value: "  ", want: ""},
		{name: "200字符", kind: OrganizationKindCompany, value: strings.Repeat("中", 200), want: strings.Repeat("中", 200)},
		{name: "超过200字符", kind: OrganizationKindCompany, value: strings.Repeat("A", 201), invalid: true},
		{name: "内部换行", kind: OrganizationKindCompany, value: "Company\nName", invalid: true},
		{name: "Unicode换行", kind: OrganizationKindCompany, value: "Company\u2028Name", invalid: true},
		{name: "Unicode段落", kind: OrganizationKindCompany, value: "Company\u2029Name", invalid: true},
		{name: "内部Tab", kind: OrganizationKindCompany, value: "Company\tName", invalid: true},
		{name: "控制字符", kind: OrganizationKindCompany, value: "Company\x00Name", invalid: true},
		{name: "部门拒绝", kind: OrganizationKindDepartment, value: "Department", invalid: true},
		{name: "团队拒绝", kind: OrganizationKindTeam, value: "Team", invalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.New()
			parent := uuid.New()
			repo := &adminRepoStub{organization: &AdminOrganization{ID: id, Kind: tc.kind, BaseCurrency: "CNY"}}
			uc := NewAdminUsecase(repo, nil)
			input := &AdminOrganization{Code: "TEST", Name: "测试公司", NameEn: tc.value, Kind: tc.kind, BaseCurrency: "CNY"}
			if tc.kind != OrganizationKindCompany {
				input.ParentID = &parent
				input.BaseCurrency = ""
			}
			// 非公司非空及字段边界必须在任何仓储写入前拒绝。
			normalized, err := normalizeOrganization(input)
			if tc.invalid {
				if err == nil {
					t.Fatal("创建未拒绝非法英文名")
				}
			} else if err != nil || normalized.NameEn != tc.want {
				t.Fatalf("创建规范化=%#v,%v", normalized, err)
			}
			currency := "CNY"
			if tc.kind != OrganizationKindCompany {
				currency = ""
			}
			result, err := uc.UpdateOrganization(context.Background(), uuid.New(), id, id, "中文名称", true, currency, tc.value)
			if tc.invalid {
				if err == nil || repo.auditEvent != nil {
					t.Fatalf("更新未零写入拒绝: %#v %v", result, err)
				}
			} else if err != nil || result.NameEn != tc.want {
				t.Fatalf("更新英文名=%#v,%v", result, err)
			}
		})
	}
}

func TestAdminUsecaseCreatesCompanyWithNameEn(t *testing.T) {
	repo := &adminRepoStub{}
	result, err := NewAdminUsecase(repo, nil).CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{Code: "NEW", Name: "新公司", Kind: OrganizationKindCompany, BaseCurrency: "CNY", NameEn: " New Co., LTD "})
	if err != nil || result.NameEn != "New Co., LTD" || repo.organizationInput.NameEn != "New Co., LTD" {
		t.Fatalf("创建没有传递英文名: %#v %v", result, err)
	}
}
