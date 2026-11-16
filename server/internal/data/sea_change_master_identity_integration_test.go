package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSeaChangesRejectInvalidMasterMembersPostgres(t *testing.T) {
	for _, operation := range []string{"拆票", "改配"} {
		for _, location := range []string{"原主单", "目标主单"} {
			t.Run(operation+location, func(t *testing.T) {
				env := newSplitTestEnv(t)
				f := createTestSplitFixture(t, env, "identity", splitFixtureOptions{})
				ctx := context.Background()
				target := env.data.db.SeaMasterBill.Create().SetBusinessType("SE").SetOrganizationID(env.orgID).SetShippingLineID(env.carrier2ID).SetMasterNo(uuid.NewString()).SetNormalizedMasterNo(uuid.NewString()).SaveX(ctx)
				te := env.data.db.SeaTransportExecution.Create().SetOrganizationID(env.orgID).SetShippingLineID(env.carrier2ID).SaveX(ctx)
				badMaster, badTE := target.ID, te.ID
				if location == "原主单" {
					badMaster, badTE = f.link.MasterBillID, f.link.TransportExecutionID
				}
				member := env.data.db.Order.Create().SetOrganizationID(env.orgID).SetBusinessType("SI").SetTradeDirection("import").SetCustomerID(env.customerID).SetOrderNo(uuid.NewString()).SetTradeTerm("FOB").SetPaymentTerm("PREPAID").SetIdempotencyKey(uuid.NewString()).SaveX(ctx)
				env.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(env.orgID).SetOrderID(member.ID).SetMasterBillID(badMaster).SetTransportExecutionID(badTE).SetDocumentStructure("HOUSE").SaveX(ctx)
				confirmation := &biz.SeaExternalConfirmation{ConfirmedByParty: "船代", ConfirmedAt: time.Now(), ConfirmationNote: "确认改配"}
				beforeOrders := env.data.db.Order.Query().CountX(ctx)
				beforeLinks := env.data.db.SeaMasterBillOrderLink.Query().CountX(ctx)
				beforeAudit := env.data.db.AuditLog.Query().CountX(ctx)
				repo := NewSeaOrderChangeRepo(env.data)
				var err error
				if operation == "拆票" {
					input := f.standardSplitInput(uuid.NewString(), uuid.NewString(), splitFixtureOptions{})
					input.Confirmation = confirmation
					input.Targets = append(input.Targets, &biz.SeaOrderSplitTargetInput{ClientTargetKey: "candidate", TargetType: biz.SplitTargetTypeCandidate, CandidateID: &target.ID, CandidateVersion: &target.Version, CandidateTEID: &te.ID, CandidateTEVersion: &te.Version, ShippingLineID: &env.carrier2ID})
					input.Results[1].ClientTargetKey = "candidate"
					_, err = repo.ExecuteSplit(ctx, env.orgID, env.userID, input, &biz.AuditEvent{})
				} else {
					_, err = repo.ExecuteReassignment(ctx, env.orgID, env.userID, &biz.SeaOrderReassignmentInput{OrderID: f.order.ID, ExpectedOrderVersion: f.order.Version, ExpectedLinkVersion: f.link.Version, IdempotencyKey: uuid.NewString(), RequestFingerprint: uuid.NewString(), Confirmation: confirmation, Target: &biz.SeaOrderReassignmentTargetInput{TargetType: biz.SplitTargetTypeCandidate, CandidateID: &target.ID, CandidateVersion: &target.Version, CandidateTEID: &te.ID, CandidateTEVersion: &te.Version}}, &biz.AuditEvent{})
				}
				if !errors.Is(err, biz.ErrSeaDocumentStructureConflict) {
					t.Fatalf("直接执行应拒绝跨方向完整成员: %v", err)
				}
				if env.data.db.Order.Query().CountX(ctx) != beforeOrders || env.data.db.SeaMasterBillOrderLink.Query().CountX(ctx) != beforeLinks || env.data.db.AuditLog.Query().CountX(ctx) != beforeAudit {
					t.Fatal("拒绝后留下订单/关联/审计写入")
				}
				if env.data.db.Order.GetX(ctx, f.order.ID).Version != f.order.Version || env.data.db.SeaMasterBillOrderLink.GetX(ctx, f.link.ID).Status != f.link.Status {
					t.Fatal("拒绝后原票状态被改写")
				}
			})
		}
	}
}
