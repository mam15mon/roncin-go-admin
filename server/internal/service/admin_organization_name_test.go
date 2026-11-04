package service

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"testing"
)

func TestOrganizationAndHblNameEnReadOnlyProjection(t *testing.T) {
	name := "Company EXAMPLE CO., LTD"
	org := organizationToAPI(&biz.AdminOrganization{NameEn: name})
	if org.GetNameEn() != name {
		t.Fatal("组织响应遗漏英文名")
	}
	if organizationToAPI(&biz.AdminOrganization{}).NameEn != nil {
		t.Fatal("空英文名不应伪造默认值")
	}
	house := seaHouseBillToAPI(&biz.SeaHouseBill{IssuerOrganizationNameEn: name, PrintTemplate: biz.SeaHouseBillPrintTemplateRoncinSelf})
	if house.GetIssuerOrganizationNameEn() != name || house.GetPrintTemplate() != "RONCIN_SELF" {
		t.Fatalf("HBL响应遗漏英文抬头或模板: %#v", house)
	}
	if seaHouseBillToAPI(&biz.SeaHouseBill{}).IssuerOrganizationNameEn != nil {
		t.Fatal("缺少英文名不应自动补北京")
	}
}
