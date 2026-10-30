package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

// 同一共同航程下的 HOUSE 操作票各存最终目的港，修改本票或共同段都不能串改另一票去向。
func TestSeaSharedTransportPreservesIndependentDestinationsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture := newOrderPostgresFixture(t, data)
	ctx := context.Background()
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 共享更新生成不可变航程版本；先清当前版本引用，之后由原夹具按外键顺序清理业务行。
	t.Cleanup(func() {
		if _, err := client.SeaTransportExecution.Update().
			Where(seatransportexecutionent.OrganizationIDEQ(fixture.organizationID)).
			ClearCurrentVersionID().Save(ctx); err != nil {
			t.Errorf("清理航程当前版本引用失败: %v", err)
		}
		if _, err := client.SeaTransportExecutionVersion.Delete().
			Where(seatransportexecutionversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx); err != nil {
			t.Errorf("清理航程不可变版本失败: %v", err)
		}
	})
	createPort := func(code, country, name string) *ent.Port {
		port, err := client.Port.Create().SetUnLocode(code).SetCountryCode(country).
			SetNameZh(name).SetNameEn(name).SetTransportModes([]string{"SEA"}).
			SetSource("system").SetEnabled(true).Save(ctx)
		if err != nil {
			t.Fatalf("创建港口 %s 失败: %v", code, err)
		}
		fixture.portIDs = append(fixture.portIDs, port.ID)
		return port
	}
	origin := createPort("CNSHA", "CN", "上海")
	singapore := createPort("SGSIN", "SG", "新加坡")
	jakarta := createPort("IDJKT", "ID", "雅加达")
	bangkok := createPort("THBKK", "TH", "曼谷")
	surabaya := createPort("IDSUB", "ID", "泗水")
	portKlang := createPort("MYPKG", "MY", "巴生港")
	masterNo := "SHAREDDEST" + strings.ToUpper(fixture.suffix)
	orderUsecase := fixture.newUsecase()
	makeInput := func(destinationID uuid.UUID) *biz.Order {
		input := fixture.validInput()
		input.SeaMasterBillInput.MasterNo = masterNo
		input.OriginLocationID = &origin.ID
		input.DischargeLocationID = &singapore.ID
		input.DestinationLocationID = &destinationID
		input.VesselVoyage = "SHARED VESSEL / 001W"
		return input
	}
	first, err := orderUsecase.Create(ctx, fixture.organizationID, fixture.actorID, makeInput(jakarta.ID))
	if err != nil {
		t.Fatalf("创建最终港为雅加达的首票失败: %v", err)
	}
	if first.SeaMasterBill == nil {
		t.Fatal("首票缺少共同主单与航程摘要")
	}
	secondInput := makeInput(bangkok.ID)
	secondInput.SeaMasterBillInput.CandidateID = &first.SeaMasterBill.MasterBillID
	secondInput.SeaMasterBillInput.ExpectedCandidateVersion = &first.SeaMasterBill.Version
	secondInput.SeaMasterBillInput.CandidateTEID = &first.SeaMasterBill.TransportExecutionID
	secondInput.SeaMasterBillInput.ExpectedCandidateTEVersion = &first.SeaMasterBill.TransportExecutionVersion
	second, err := orderUsecase.Create(ctx, fixture.organizationID, fixture.actorID, secondInput)
	if err != nil {
		t.Fatalf("创建最终港为曼谷的共享分单失败: %v", err)
	}
	if second.SeaMasterBill == nil || second.SeaMasterBill.MasterBillID != first.SeaMasterBill.MasterBillID ||
		second.SeaMasterBill.TransportExecutionID != first.SeaMasterBill.TransportExecutionID {
		t.Fatal("两票未共享同一主单和共同航程")
	}
	for _, order := range []*biz.Order{first, second} {
		if order.SeaDocumentSummary == nil || order.SeaDocumentSummary.DocumentStructure != biz.SeaDocumentStructureHouse {
			t.Fatalf("订单 %s 应为 HOUSE 操作票", order.ID)
		}
	}
	assertDestinations := func(firstDestinationID, secondDestinationID uuid.UUID) {
		t.Helper()
		for orderID, wantDestinationID := range map[uuid.UUID]uuid.UUID{
			first.ID: firstDestinationID, second.ID: secondDestinationID,
		} {
			stored, err := client.Order.Get(ctx, orderID)
			if err != nil {
				t.Fatalf("读取订单原始目的港失败: %v", err)
			}
			projected, err := orderUsecase.Get(ctx, fixture.organizationID, orderID)
			if err != nil {
				t.Fatalf("读取订单目的港投影失败: %v", err)
			}
			if stored.DestinationLocationID == nil || *stored.DestinationLocationID != wantDestinationID ||
				projected.DestinationLocationID == nil || *projected.DestinationLocationID != wantDestinationID {
				t.Fatalf("订单 %s 的独立目的港被改写，原始值 %v、投影值 %v，期望 %s", orderID, stored.DestinationLocationID, projected.DestinationLocationID, wantDestinationID)
			}
		}
	}
	assertDestinations(jakarta.ID, bangkok.ID)
	teBefore, err := client.SeaTransportExecution.Get(ctx, first.SeaMasterBill.TransportExecutionID)
	if err != nil {
		t.Fatalf("读取原共同航程失败: %v", err)
	}

	// 普通保存只调整 A 票最终港；共同航程和 B 票保持不变。
	update := makeInput(surabaya.ID)
	update.SeaDocumentInput = nil
	update.SeaMasterBillInput.ExpectedCandidateVersion = &first.SeaMasterBill.Version
	if _, err := orderUsecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, first.ID, first.Version, update); err != nil {
		t.Fatalf("独立修改首票最终港失败: %v", err)
	}
	assertDestinations(surabaya.ID, bangkok.ID)
	teAfterDestination, err := client.SeaTransportExecution.Get(ctx, teBefore.ID)
	if err != nil {
		t.Fatalf("读取独立目的港修改后的共同航程失败: %v", err)
	}
	if teAfterDestination.Version != teBefore.Version || teAfterDestination.VesselName != teBefore.VesselName ||
		teAfterDestination.VoyageNo != teBefore.VoyageNo ||
		!optionalUUIDEquals(teAfterDestination.OriginLocationID, teBefore.OriginLocationID) ||
		!optionalUUIDEquals(teAfterDestination.DischargeLocationID, teBefore.DischargeLocationID) ||
		!optionalUUIDEquals(teAfterDestination.TransitLocationID, teBefore.TransitLocationID) ||
		!optionalTimeEquals(teAfterDestination.Etd, teBefore.Etd) || !optionalTimeEquals(teAfterDestination.Eta, teBefore.Eta) {
		t.Fatal("修改本票目的港不应改写共同航程或推进其版本")
	}

	// 走现有预览、外部确认及版本校验，调整共同卸货港和船期。
	newETD := time.Date(2026, time.September, 12, 0, 0, 0, 0, time.UTC)
	newETA := time.Date(2026, time.September, 27, 0, 0, 0, 0, time.UTC)
	command := &biz.SeaTransportExecutionUpdateCommand{
		OrderID: first.ID, ExpectedTransportExecutionVersion: teBefore.Version,
		Input: &biz.SeaTransportExecutionUpdateInput{
			OriginLocationID: &origin.ID, DischargeLocationID: &portKlang.ID,
			VesselName: "UPDATED SHARED VESSEL", VoyageNo: "002W", ETD: &newETD, ETA: &newETA,
		},
		Reason: "共同段改靠巴生港，各票最终目的港保持独立",
		Confirmation: &biz.SeaExternalConfirmation{
			ConfirmedByParty: "测试船代", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "已确认共同航程调整",
		},
		IdempotencyKey: "shared-destination-" + fixture.suffix,
	}
	changeUsecase := biz.NewSeaOrderChangeUsecase(NewSeaOrderChangeRepo(data), data)
	preview, err := changeUsecase.PreviewTransportExecutionUpdate(ctx, fixture.organizationID, command)
	if err != nil {
		t.Fatalf("预览共同航程调整失败: %v", err)
	}
	if !preview.Executable || preview.TransportExecutionID != teBefore.ID || preview.TransportExecutionVersion != teBefore.Version ||
		!equalUUIDSlices(sortAndDeduplicateUUIDs(preview.MemberOrderIDs), sortAndDeduplicateUUIDs([]uuid.UUID{first.ID, second.ID})) {
		t.Fatalf("共同航程预览必须准确包含两张操作票: %+v", preview)
	}
	result, err := changeUsecase.ExecuteTransportExecutionUpdate(ctx, fixture.organizationID, fixture.actorID, command)
	if err != nil {
		t.Fatalf("执行共同航程调整失败: %v", err)
	}
	if result.TransportExecution == nil || result.TransportExecution.ID != teBefore.ID ||
		result.TransportExecution.Version != teBefore.Version+1 || result.VersionID == uuid.Nil {
		t.Fatalf("共同航程更新未返回新版本及不可变快照: %+v", result)
	}
	assertDestinations(surabaya.ID, bangkok.ID)
	for _, orderID := range []uuid.UUID{first.ID, second.ID} {
		projected, err := orderUsecase.Get(ctx, fixture.organizationID, orderID)
		if err != nil {
			t.Fatalf("读取共同航程更新后的订单失败: %v", err)
		}
		if projected.SeaMasterBill == nil || projected.SeaMasterBill.TransportExecutionID != teBefore.ID ||
			projected.SeaMasterBill.TransportExecutionVersion != teBefore.Version+1 ||
			projected.OriginLocationID == nil || *projected.OriginLocationID != origin.ID ||
			projected.DischargeLocationID == nil || *projected.DischargeLocationID != portKlang.ID ||
			projected.VesselVoyage != "UPDATED SHARED VESSEL / 002W" ||
			projected.ETD != newETD.Format(time.RFC3339) || projected.ETA != newETA.Format(time.RFC3339) {
			t.Fatalf("订单 %s 未投影同一共同航程新事实: %+v", orderID, projected)
		}
	}
}
