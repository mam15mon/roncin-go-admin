package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
)

// TestOrderUnifiedSavePostgres 使用正式迁移验证一次保存、完整回滚和连续版本。
func TestOrderUnifiedSavePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newOrderPostgresFixture(t, data)
	uc := fixture.newUsecase()
	order, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
	if err != nil {
		t.Fatal(err)
	}
	docsRepo := NewSeaDocumentRepo(data)
	read := func() *biz.SeaOrderDocuments {
		t.Helper()
		d, e := docsRepo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
		if e != nil {
			t.Fatal(e)
		}
		return d
	}
	docs := read()
	build := func() *biz.Order {
		input := fixture.validUpdateInput()
		input.Notes = "统一保存"
		input.IdempotencyKey = uuid.NewString()
		shipper := "主单修改"
		houseShipper := "分单修改"
		input.SeaDocumentInput = &biz.SeaOrderDocumentInput{ExpectedMblVersion: &docs.MasterBill.Version, ExpectedLinkVersion: &docs.LinkVersion, MasterBillContent: &biz.SeaBillContent{ShipperText: &shipper}, HouseBill: &biz.SeaHouseBillInput{ID: &docs.HouseBill.ID, ExpectedVersion: &docs.HouseBill.Version, HouseNo: docs.HouseBill.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization, Content: &biz.SeaBillContent{ShipperText: &houseShipper}}}
		return input
	}
	otherInput := fixture.validInput()
	otherInput.SeaMasterBillInput.MasterNo = "COSCOOTHER"
	otherOrder, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, otherInput)
	if err != nil {
		t.Fatal(err)
	}
	otherDocs, err := docsRepo.GetSeaOrderDocuments(ctx, fixture.organizationID, otherOrder.ID)
	if err != nil {
		t.Fatal(err)
	}

	for _, conflict := range []string{"订单版本", "主单版本", "关联版本", "分单版本", "其他票分单", "签发主体无效"} {
		t.Run(conflict, func(t *testing.T) {
			input := build()
			version := order.Version
			bad := uint64(99999)
			switch conflict {
			case "订单版本":
				version = bad
			case "主单版本":
				input.SeaDocumentInput.ExpectedMblVersion = &bad
			case "关联版本":
				input.SeaDocumentInput.ExpectedLinkVersion = &bad
			case "分单版本":
				input.SeaDocumentInput.HouseBill.ExpectedVersion = &bad
			case "签发主体无效":
				id := uuid.New()
				input.SeaDocumentInput.HouseBill.IssuerSource = biz.SeaHouseBillIssuerSourceOtherPartner
				input.SeaDocumentInput.HouseBill.IssuerPartnerID = &id
			case "其他票分单":
				input.SeaDocumentInput.HouseBill.ID = &otherDocs.HouseBill.ID
			}
			countBefore, e := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, version, input); e == nil {
				t.Fatal("冲突应失败")
			}
			after := read()
			fresh, e := NewOrderRepo(data).Get(ctx, fixture.organizationID, order.ID)
			if e != nil {
				t.Fatal(e)
			}
			if after.MasterBill.Version != docs.MasterBill.Version || after.HouseBill.Version != docs.HouseBill.Version || after.LinkVersion != docs.LinkVersion || fresh.Version != order.Version || fresh.Notes != order.Notes {
				t.Fatal("冲突发生部分落库")
			}
			countAfter, e := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
			if e != nil || countAfter != countBefore {
				t.Fatal("失败写入了审计")
			}
		})
	}
	for range 2 {
		input := build()
		oldOrder := order.Version
		oldDocs := docs
		updated, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, oldOrder, input)
		if e != nil {
			t.Fatal(e)
		}
		docs = read()
		if updated.Notes != "统一保存" || updated.Version != oldOrder+1 || docs.HouseBill.Version != oldDocs.HouseBill.Version+1 || docs.LinkVersion != oldDocs.LinkVersion+1 || docs.MasterBill.Version != oldDocs.MasterBill.Version+1 || docs.MasterBill.Content.ShipperText == nil || *docs.MasterBill.Content.ShipperText != "主单修改" || docs.HouseBill.Content.ShipperText == nil || *docs.HouseBill.Content.ShipperText != "分单修改" {
			t.Fatal("整单未完整保存/版本未推进")
		}
		replay, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, oldOrder, input)
		if e != nil || replay.Version != updated.Version {
			t.Fatalf("幂等重试失败: %v", e)
		}
		order = updated
	}
	// 同包更正主单身份与正文，正文仍比对事务开始时的版本。
	input := build()
	input.SeaMasterBillInput.MasterNo = "COSCOUNIFIED"
	input.SeaMasterBillInput.ExpectedCandidateVersion = &docs.MasterBill.Version
	input.SeaMasterBillInput.CorrectionReason = "核对提单号后更正"
	if _, err := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, input); err != nil {
		t.Fatalf("主单身份与正文同包更新失败: %v", err)
	}
	if fresh := read(); fresh.MasterBill.MasterNo != "COSCOUNIFIED" {
		t.Fatal("主单身份未更新")
	}

	order, err = NewOrderRepo(data).Get(ctx, fixture.organizationID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	docs = read()
	t.Run("仅分单更新不推进主单版本", func(t *testing.T) {
		input := build()
		input.SeaMasterBillInput.MasterNo = docs.MasterBill.MasterNo
		input.SeaDocumentInput.MasterBillContent = nil
		input.SeaDocumentInput.ExpectedMblVersion = nil
		before := docs.MasterBill.Version
		order, err = uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, input)
		if err != nil {
			t.Fatal(err)
		}
		docs = read()
		if docs.MasterBill.Version != before {
			t.Fatal("未修改主单却推进版本")
		}
	})
	for _, status := range []seahousebillent.Status{seahousebillent.StatusRELEASED, seahousebillent.StatusVOIDED} {
		t.Run(string(status), func(t *testing.T) {
			data.db.SeaHouseBill.UpdateOneID(docs.HouseBill.ID).SetStatus(status).SaveX(ctx)
			input := build()
			input.SeaMasterBillInput.MasterNo = docs.MasterBill.MasterNo
			if _, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, input); e == nil {
				t.Fatal("终态分单应阻断更新")
			}
			if after := read(); after.MasterBill.Version != docs.MasterBill.Version || after.LinkVersion != docs.LinkVersion {
				t.Fatal("终态失败部分写入")
			}
			data.db.SeaHouseBill.UpdateOneID(docs.HouseBill.ID).SetStatus(seahousebillent.StatusDRAFT).SaveX(ctx)
		})
	}
	t.Run("同批重号和共享成员锁定", func(t *testing.T) {
		match, e := uc.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, docs.MasterBill.MasterNo, nil)
		if e != nil {
			t.Fatal(e)
		}
		member, e := createOrderConfirmCandidate(ctx, fixture, uc, docs.MasterBill.MasterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if e != nil {
			t.Fatal(e)
		}
		memberDocs, e := docsRepo.GetSeaOrderDocuments(ctx, fixture.organizationID, member.ID)
		if e != nil {
			t.Fatal(e)
		}
		docs = read()
		input := build()
		input.SeaMasterBillInput.MasterNo = docs.MasterBill.MasterNo
		input.SeaDocumentInput.HouseBill.HouseNo = memberDocs.HouseBill.HouseNo
		if _, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, input); e == nil {
			t.Fatal("同批重号应失败")
		}
		if after := read(); after.HouseBill.Version != docs.HouseBill.Version || after.MasterBill.Version != docs.MasterBill.Version || after.LinkVersion != docs.LinkVersion {
			t.Fatal("重号发生部分写入")
		}
		mustLockOrderDirectly(t, data, fixture, member.ID)
		input = build()
		input.SeaMasterBillInput.MasterNo = docs.MasterBill.MasterNo
		if _, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, input); e == nil {
			t.Fatal("共享成员锁定应阻止主单变更")
		}
		if after := read(); after.HouseBill.Version != docs.HouseBill.Version || after.MasterBill.Version != docs.MasterBill.Version || after.LinkVersion != docs.LinkVersion {
			t.Fatal("成员锁定发生部分写入")
		}
	})
	t.Run("DIRECT普通整单保存不要求分单", func(t *testing.T) {
		input := fixture.validInput()
		direct := biz.SeaDocumentStructureDirect
		input.SeaDocumentInput = &biz.SeaOrderDocumentInput{DocumentStructure: &direct}
		input.SeaMasterBillInput.MasterNo = "COSCODIRECT"
		created, e := uc.Create(ctx, fixture.organizationID, fixture.actorID, input)
		if e != nil {
			t.Fatal(e)
		}
		update := fixture.validUpdateInput()
		update.SeaMasterBillInput.MasterNo = "COSCODIRECT"
		update.Notes = "直单普通修改"
		saved, e := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, update)
		if e != nil {
			t.Fatal(e)
		}
		d, e := docsRepo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID)
		if e != nil || d.HouseBill != nil || saved.Notes != "直单普通修改" {
			t.Fatal("DIRECT保存异常")
		}
	})

}
