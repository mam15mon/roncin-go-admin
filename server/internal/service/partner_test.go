package service

import (
	"testing"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestParseContractDatesRequiresRFC3339(t *testing.T) {
	if _, _, err := parseContractDates("2026-08-20", "2027-08-20"); err != biz.ErrPartnerContractInvalidArgument {
		t.Fatalf("parse date error = %v, want ErrPartnerContractInvalidArgument", err)
	}
	start, end, err := parseContractDates("2026-08-20T00:00:00Z", "2027-08-20T00:00:00Z")
	if err != nil {
		t.Fatalf("parse RFC3339 dates error = %v", err)
	}
	if start.Format(time.RFC3339) != "2026-08-20T00:00:00Z" || end.Format(time.RFC3339) != "2027-08-20T00:00:00Z" {
		t.Fatalf("parsed dates = %s - %s", start, end)
	}
}

func TestPartnerAssignmentFinanceRoleRoundTrip(t *testing.T) {
	apiRole := partnerAssignmentRoleToAPI(biz.PartnerAssignmentFinance)
	if apiRole != v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_FINANCE {
		t.Fatalf("财务人员转换到 API 角色为 %v", apiRole)
	}
	if role := partnerAssignmentRoleFromAPI(apiRole); role != biz.PartnerAssignmentFinance {
		t.Fatalf("财务人员从 API 转回领域角色为 %q", role)
	}
}

func TestPartnerAssignmentDocumentRoleRoundTrip(t *testing.T) {
	apiRole := partnerAssignmentRoleToAPI(biz.PartnerAssignmentDocument)
	if apiRole != v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_DOCUMENT {
		t.Fatalf("单证人员转换到 API 角色为 %v", apiRole)
	}
	if role := partnerAssignmentRoleFromAPI(apiRole); role != biz.PartnerAssignmentDocument {
		t.Fatalf("单证人员从 API 转回领域角色为 %q", role)
	}
}

func TestPartnerRoleToAPIOutputsBlacklistedOperatorName(t *testing.T) {
	role := &biz.PartnerRole{Type: biz.PartnerRoleCustomer, Enabled: true, Blacklisted: true, BlacklistedByName: "张风控"}
	apiRole := partnerRoleToAPI(role)
	if apiRole.BlacklistedByName != "张风控" {
		t.Fatalf("拉黑操作人显示姓名转换结果为 %q", apiRole.BlacklistedByName)
	}
	if apiRole.Type != v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER || !apiRole.Blacklisted {
		t.Fatalf("角色类型与拉黑状态转换结果为 %+v", apiRole)
	}
}

func TestPartnerExportItemToAPICarriesFullRolesContactsUpdatedAt(t *testing.T) {
	value := &biz.Partner{
		Code:      "CUST-EXP",
		LegalName: "导出测试公司",
		Enabled:   true,
		Roles: []*biz.PartnerRole{
			{Type: biz.PartnerRoleCustomer, Enabled: false, Blacklisted: true, BlacklistedByName: "张风控"},
		},
		Contacts:  []*biz.PartnerContact{{Name: "王联系", Phone: "13800000000"}},
		UpdatedAt: time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC),
	}
	item := partnerExportItemToAPI(value)
	if len(item.Roles) != 1 || item.Roles[0].Enabled || !item.Roles[0].Blacklisted || item.Roles[0].BlacklistedByName != "张风控" {
		t.Fatalf("导出条目角色应保留停用与拉黑状态及操作人姓名，实际为 %+v", item.Roles)
	}
	if len(item.Contacts) != 1 || item.Contacts[0].Name != "王联系" || item.Contacts[0].Phone != "13800000000" {
		t.Fatalf("导出条目联系人转换结果为 %+v", item.Contacts)
	}
	if item.UpdatedAt != "2026-09-20T08:00:00Z" {
		t.Fatalf("导出条目更新时间转换结果为 %q", item.UpdatedAt)
	}
}
