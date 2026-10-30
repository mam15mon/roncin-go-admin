package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

// TestOrderMasterBillDownstreamFactsPostgres 确认主单更正仍受真实放货与实箱事实约束，
// 被阻断的身份/航程修改不能部分改写订单、共享主单、运输执行或审计。
func TestOrderMasterBillDownstreamFactsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	for _, fact := range []string{"放货POD", "实际集装箱"} {
		t.Run(fact, func(t *testing.T) {
			ctx := context.Background()
			fixture := newOrderPostgresFixture(t, data)
			usecase := fixture.newUsecase()
			input := fixture.validInput()
			shipmentType := biz.OrderShipmentFCL
			input.ShipmentType = &shipmentType
			input.SeaMasterBillInput.MasterNo = "FACT" + strings.ToUpper(fixture.suffix)
			input.VesselVoyage = "VESSEL A / 001"
			created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
			if err != nil {
				t.Fatalf("创建下游事实测试订单: %v", err)
			}
			if created.SeaMasterBill == nil {
				t.Fatal("新订单缺少主单候选摘要")
			}

			switch fact {
			case "放货POD":
				releaseNo := "POD-" + fixture.suffix
				pod, createErr := NewOrderReleasePodRepo(data).Add(ctx, fixture.organizationID, created.ID,
					&biz.OrderReleasePod{ID: uuid.Must(uuid.NewV7()), ReleaseNo: &releaseNo},
					testGateAuditEvent(fixture, "order.release_pod.add"))
				if createErr != nil {
					t.Fatalf("创建放货事实: %v", createErr)
				}
				t.Cleanup(func() {
					if err := data.db.OrderReleasePod.DeleteOneID(pod.ID).Exec(ctx); err != nil {
						t.Errorf("清理放货事实: %v", err)
					}
				})
			case "实际集装箱":
				spec := data.db.MasterDataItem.Create().SetKind(masterdataitement.KindContainerSpec).
					SetCode("FACT-40HQ-" + fixture.suffix).SetName("下游门禁测试40HQ").SaveX(ctx)
				t.Cleanup(func() {
					if err := data.db.MasterDataItem.DeleteOne(spec).Exec(ctx); err != nil {
						t.Errorf("清理箱型: %v", err)
					}
				})
				container, createErr := NewOrderContainerRepo(data).Add(ctx, fixture.organizationID, created.ID,
					&biz.OrderContainer{ID: uuid.Must(uuid.NewV7()), ContainerNo: "FACT-" + fixture.suffix,
						ContainerSpecID: spec.ID, PackageCount: 10, GrossWeightKg: 100, VolumeCbm: 1},
					testGateAuditEvent(fixture, "order.container.add"))
				if createErr != nil {
					t.Fatalf("创建实箱事实: %v", createErr)
				}
				t.Cleanup(func() {
					if err := data.db.OrderContainer.DeleteOneID(container.ID).Exec(ctx); err != nil {
						t.Errorf("清理实箱事实: %v", err)
					}
				})
			}

			snapshot := func() []byte {
				t.Helper()
				order := data.db.Order.GetX(ctx, created.ID)
				masterBill := data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(fixture.organizationID)).OnlyX(ctx)
				execution := data.db.SeaTransportExecution.Query().Where(seatransportexecutionent.OrganizationIDEQ(fixture.organizationID)).OnlyX(ctx)
				value, marshalErr := json.Marshal(struct {
					Order      *ent.Order
					MasterBill *ent.SeaMasterBill
					Execution  *ent.SeaTransportExecution
					AuditCount int
				}{order, masterBill, execution, data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).CountX(ctx)})
				if marshalErr != nil {
					t.Fatalf("序列化下游门禁快照: %v", marshalErr)
				}
				return value
			}
			before := snapshot()
			for _, change := range []string{"主单身份", "共享航程"} {
				update := fixture.validUpdateInput()
				update.ShipmentType = &shipmentType
				update.GoodsDescription = "被阻断时不应保存的业务字段"
				update.SeaMasterBillInput.MasterNo = input.SeaMasterBillInput.MasterNo
				update.SeaMasterBillInput.ExpectedCandidateVersion = &created.SeaMasterBill.Version
				update.VesselVoyage = input.VesselVoyage
				if change == "主单身份" {
					update.SeaMasterBillInput.MasterNo = "NEW" + input.SeaMasterBillInput.MasterNo
					update.SeaMasterBillInput.CorrectionReason = "录入错误"
				} else {
					update.VesselVoyage = "VESSEL B / 002"
				}
				_, updateErr := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, update)
				if !errors.Is(updateErr, biz.ErrSeaMasterBillCorrectionBlocked) {
					t.Fatalf("已有%s时%s更正应被阻断，实际: %v", fact, change, updateErr)
				}
				if after := snapshot(); !bytes.Equal(before, after) {
					t.Fatalf("已有%s时被阻断的%s更正存在部分写入\nbefore=%s\nafter=%s", fact, change, before, after)
				}
			}
		})
	}
}
