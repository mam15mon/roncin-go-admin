package data

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func TestSeaDocumentTerminationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)
	for _, structure := range []biz.SeaDocumentStructure{biz.SeaDocumentStructureDirect, biz.SeaDocumentStructureHouse} {
		t.Run(string(structure), func(t *testing.T) {
			fixture := newOrderPostgresFixture(t, data)
			usecase := fixture.newUsecase()
			input := fixture.validInput()
			input.SeaDocumentInput.DocumentStructure = &structure
			if structure == biz.SeaDocumentStructureDirect {
				input.SeaDocumentInput.HouseBill = nil
			}
			created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
			if err != nil {
				t.Fatal(err)
			}
			before, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			link := data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(created.ID)).OnlyX(ctx)
			mustCompleteTermination(t, usecase, fixture, created)
			stored := data.db.Order.GetX(ctx, created.ID)
			ended := data.db.SeaMasterBillOrderLink.GetX(ctx, link.ID)
			// 改配历史、上次退关历史及同刻其他结束原因都不能冒充本次退关关联。
			for _, history := range []struct {
				at     time.Time
				reason string
			}{
				{at: ended.EndedAt.Add(time.Hour), reason: "订单改配"},
				{at: ended.EndedAt.Add(-time.Hour), reason: orderTerminationLinkEndedReason},
				{at: *ended.EndedAt, reason: "订单改配"},
			} {
				data.db.SeaMasterBillOrderLink.Create().
					SetOrganizationID(fixture.organizationID).SetOrderID(created.ID).
					SetMasterBillID(link.MasterBillID).SetTransportExecutionID(link.TransportExecutionID).
					SetDocumentStructure(linkent.DocumentStructureDIRECT).SetStatus(linkent.StatusENDED).
					SetEndedAt(history.at).SetEndedReason(history.reason).SetVersion(99).SaveX(ctx)
			}

			for range 2 {
				docs, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID)
				if err != nil {
					t.Fatalf("已退关订单重新读取失败: %v", err)
				}
				if docs.OrderID != created.ID || docs.DocumentStructure != structure || docs.LinkVersion != ended.Version {
					t.Fatalf("未读取本次退关关联: %#v", docs)
				}
				if docs.MasterBill == nil || docs.MasterBill.ID != before.MasterBill.ID || docs.MasterBill.Version != before.MasterBill.Version || docs.MasterBill.MemberCount != 0 {
					t.Fatalf("保留主单或活动成员统计异常: %#v", docs.MasterBill)
				}
				if len(docs.AllowedActions) != 0 {
					t.Fatalf("终态仍提供写动作: %v", docs.AllowedActions)
				}
				if structure == biz.SeaDocumentStructureHouse {
					if docs.HouseBill == nil || docs.HouseBill.ID != before.HouseBill.ID || docs.HouseBill.HouseNo != before.HouseBill.HouseNo {
						t.Fatalf("保留分单异常: %#v", docs.HouseBill)
					}
				} else if docs.HouseBill != nil {
					t.Fatal("直单不应构造分单")
				}
			}
			if _, err := repo.GetSeaOrderDocuments(ctx, fixture.headquartersID, created.ID); !errors.Is(err, biz.ErrOrderNotFound) {
				t.Fatalf("跨组织读取应拒绝: %v", err)
			}
			audit := &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.actorID, Result: "success"}
			if _, err := repo.UpdateSeaMasterBillContent(ctx, fixture.organizationID, fixture.actorID, created.ID, before.MasterBill.Version, &biz.SeaBillContent{}, audit); !errors.Is(err, biz.ErrSeaDocumentNoActiveLink) {
				t.Fatalf("终态主单写入门禁失效: %v", err)
			}
			if before.HouseBill != nil {
				if _, err := repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, created.ID, before.HouseBill.ID, before.HouseBill.Version, ended.Version, input.SeaDocumentInput.HouseBill, audit); errors.Reason(err) != "ORDER_TERMINATED" {
					t.Fatalf("终态分单写入门禁失效: %v", err)
				}
			}
			if summary, err := repo.GetSummariesByOrderIDs(ctx, fixture.organizationID, []uuid.UUID{created.ID}); err != nil || len(summary) != 0 {
				t.Fatalf("终态不应进入活动单证摘要: %v %v", summary, err)
			}

			ended.Update().SetEndedReason("订单改配").SaveX(ctx)
			if _, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID); !errors.Is(err, biz.ErrSeaDocumentNoActiveLink) {
				t.Fatalf("本次退关关联缺失不得选择其他历史: %v", err)
			}
			ended.Update().SetEndedReason(orderTerminationLinkEndedReason).SaveX(ctx)
			restored, err := usecase.TransitionTermination(ctx, fixture.organizationID, fixture.actorID, created.ID, stored.Version, biz.OrderTerminationActive, nil, "测试恢复订单")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID); err != nil {
				t.Fatalf("恢复后活动关系应可读: %v", err)
			}
			version := mustStartTermination(t, usecase, fixture, restored)
			if _, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID); err != nil {
				t.Fatalf("退关中应保留新活动关系: %v", err)
			}
			// 新关系使用独立并发版本，确保第二轮读取不能借用旧结束关系。
			newLink := data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(created.ID), linkent.StatusEQ(linkent.StatusACTIVE)).OnlyX(ctx)
			newLink.Update().SetVersion(7).SaveX(ctx)
			mustCompleteTerminationFrom(t, usecase, fixture, created.ID, version)
			secondEnded := data.db.SeaMasterBillOrderLink.GetX(ctx, newLink.ID)
			second, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, created.ID)
			if err != nil || second.MasterBill == nil || second.LinkVersion != secondEnded.Version || second.LinkVersion == ended.Version {
				t.Fatalf("新一轮退关应读取其新结束关系: %v", err)
			}

		})
	}
}
