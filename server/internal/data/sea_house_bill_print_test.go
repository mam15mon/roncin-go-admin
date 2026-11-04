package data

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
)

func TestSeaHouseBillPrintTemplateUsesRealIssuer(t *testing.T) {
	id := uuid.New()
	cases := []struct {
		name, code string
		source     seahousebillent.IssuerSource
		issuer     *uuid.UUID
		expected   string
	}{
		{"北京自签", "BJ", seahousebillent.IssuerSourceSELF_ORGANIZATION, &id, biz.SeaHouseBillPrintTemplateRoncinBeijing},
		{"成都自签", "CD", seahousebillent.IssuerSourceSELF_ORGANIZATION, &id, ""},
		{"客户签发", "BJ", seahousebillent.IssuerSourceCUSTOMER_PARTNER, &id, ""},
		{"代理签发", "BJ", seahousebillent.IssuerSourceOTHER_PARTNER, &id, ""},
		{"无真实主体", "BJ", seahousebillent.IssuerSourceSELF_ORGANIZATION, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := &ent.SeaHouseBill{IssuerSource: tc.source, IssuerOrganizationID: tc.issuer}
			if got := seaHouseBillPrintTemplate(item, tc.code); got != tc.expected {
				t.Fatalf("模板=%s，期望%s", got, tc.expected)
			}
		})
	}
}

func TestSeaHouseBillPrintDetailsChangesAffectHashDifferenceAndFingerprint(t *testing.T) {
	original := "2026-11-04"
	changed := "2026-11-05"
	item := &ent.SeaHouseBill{HouseNo: "HBL-001", PrintDetails: &schema.SeaHouseBillPrintDetails{IssueDate: &original}}
	previousHash := computeHBLContentHash(item)
	item.PrintDetails = &schema.SeaHouseBillPrintDetails{IssueDate: &changed}
	if computeHBLContentHash(item) == previousHash {
		t.Fatal("印字修改未进入内容哈希")
	}
	before := &biz.SeaDocumentVersion{PrintDetails: &biz.SeaHouseBillPrintDetails{IssueDate: &original}}
	input := &biz.SeaHouseBillInput{PrintDetails: &biz.SeaHouseBillPrintDetails{IssueDate: &changed}}
	diffs := diffHouseVersionToInput(before, input, nil, nil)
	if len(diffs) != 1 || diffs[0].Field != "print_details.issue_date" || diffs[0].BeforeValue != original || diffs[0].AfterValue != changed {
		t.Fatalf("改单差异遗漏印字: %#v", diffs)
	}
	first := changeFingerprint(input)
	input.PrintDetails.IssueDate = &original
	if first == changeFingerprint(input) {
		t.Fatal("印字修改未进入请求指纹")
	}
	details := seaHouseBillPrintDetailsToBiz(item.PrintDetails)
	if !reflect.DeepEqual(item.PrintDetails, seaHouseBillPrintDetailsToPO(details)) {
		t.Fatal("印字持久化转换丢字段")
	}
}
