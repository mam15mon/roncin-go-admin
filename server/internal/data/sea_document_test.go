package data

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func getIntegrationData(t *testing.T) (*Data, func()) {
	t.Helper()
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminDB, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 集成测试数据库失败: %v", err)
	}
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		t.Fatalf("连接 PostgreSQL 集成测试数据库失败: %v", err)
	}

	schemaName := "roncin_sea_document_test_" + fmt.Sprintf("%x", uuid.New())
	quotedSchema := `"` + schemaName + `"`
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = adminDB.Close()
		t.Fatalf("创建隔离测试 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := adminDB.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("删除隔离测试 Schema 失败: %v", err)
		}
		_ = adminDB.Close()
	})

	pgxConfig, err := pgx.ParseConfig(source)
	if err != nil {
		t.Fatalf("解析 PostgreSQL 集成测试连接串失败: %v", err)
	}
	pgxConfig.RuntimeParams["search_path"] = schemaName + ",public"
	isolatedSource := stdlib.RegisterConnConfig(pgxConfig)
	t.Cleanup(func() {
		stdlib.UnregisterConnConfig(isolatedSource)
	})
	migrationDB, err := sql.Open("pgx", isolatedSource)
	if err != nil {
		t.Fatalf("打开隔离迁移连接失败: %v", err)
	}
	migrationDB.SetMaxOpenConns(1)
	if err := migration.Apply(ctx, migrationDB, filepath.Join("..", "..", "migrations")); err != nil {
		_ = migrationDB.Close()
		t.Fatalf("在隔离 Schema 执行迁移失败: %v", err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatalf("关闭隔离迁移连接失败: %v", err)
	}

	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             isolatedSource,
		AutoMigrate:        false,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, nil)
	if err != nil {
		t.Fatalf("无法连接集成测试数据库: %v", err)
	}
	return data, cleanup
}

func TestSeaDocumentPostgresIntegration(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)

	// 1. 创建测试总部组织与下属部门组织
	hqOrg, err := data.db.Organization.Create().
		SetCode("TEST-HQ-" + uuid.New().String()[:8]).
		SetName("测试总部").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试总部组织失败: %v", err)
	}
	defer func() {
		_ = data.db.Organization.DeleteOne(hqOrg).Exec(ctx)
	}()

	deptOrg, err := data.db.Organization.Create().
		SetCode("TEST-DEPT-" + uuid.New().String()[:8]).
		SetName("测试业务部").
		SetKind("department").
		SetParentID(hqOrg.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试部门组织失败: %v", err)
	}
	defer func() {
		_ = data.db.Organization.DeleteOne(deptOrg).Exec(ctx)
	}()

	// 2. 创建客户 Partner
	customerPartner, err := data.db.Partner.Create().
		SetOrganizationID(deptOrg.ID).
		SetCode("CUST-" + uuid.New().String()[:8]).
		SetLegalName("测试客户").
		SetNormalizedName("测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户失败: %v", err)
	}
	defer func() {
		_ = data.db.Partner.DeleteOne(customerPartner).Exec(ctx)
	}()
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(customerPartner.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试客户角色失败: %v", err)
	}

	// 3. 创建主单签发方 Partner
	issuerPartner, err := data.db.Partner.Create().
		SetOrganizationID(deptOrg.ID).
		SetCode("CARR-" + uuid.New().String()[:8]).
		SetLegalName("测试船公司").
		SetNormalizedName("测试船公司").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试船公司失败: %v", err)
	}
	defer func() {
		_ = data.db.Partner.DeleteOne(issuerPartner).Exec(ctx)
	}()

	// 4. 创建测试订单（所属 deptOrg）
	testOrder, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-TEST-" + uuid.New().String()[:8]).
		SetCustomerID(customerPartner.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单失败: %v", err)
	}
	defer func() {
		_ = data.db.Order.DeleteOne(testOrder).Exec(ctx)
	}()

	// 4.5. 创建运输执行实体
	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(deptOrg.ID).
		SetVesselName("EVER GIVEN").
		SetVoyageNo("001W").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试运输执行失败: %v", err)
	}
	defer func() {
		_ = data.db.SeaTransportExecution.DeleteOne(te).Exec(ctx)
	}()

	// 5. 创建主单与 link
	masterNo := "TESTMBL" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(masterNo).
		SetNormalizedMasterNo(masterNo).
		SetIssuerPartnerID(issuerPartner.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试主单失败: %v", err)
	}
	defer func() {
		_ = data.db.SeaMasterBill.DeleteOne(mbl).Exec(ctx)
	}()

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(testOrder.ID).
		SetMasterBillID(mbl.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureUNDETERMINED).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试主单关联合约失败: %v", err)
	}
	defer func() {
		_ = data.db.SeaMasterBillOrderLink.DeleteOne(link).Exec(ctx)
	}()

	actorID := uuid.New()
	makeAudit := func() *biz.AuditEvent {
		return &biz.AuditEvent{
			OrganizationID: &deptOrg.ID,
			UserID:         &actorID,
			Result:         "success",
		}
	}

	// 校验 1：初始查询应为未确定
	docAgg, err := repo.GetSeaOrderDocuments(ctx, deptOrg.ID, testOrder.ID)
	if err != nil {
		t.Fatalf("GetSeaOrderDocuments failed: %v", err)
	}
	if docAgg.DocumentStructure != biz.SeaDocumentStructureUndetermined {
		t.Fatalf("expected UNDETERMINED, got %s", docAgg.DocumentStructure)
	}
	if docAgg.LinkVersion != link.Version {
		t.Fatalf("expected link version %d, got %d", link.Version, docAgg.LinkVersion)
	}

	// 校验 2：业务行更新后 writeAudit 失败时，整个事务必须回滚。
	failedAudit := makeAudit()
	failedAudit.Result = "invalid-result"
	_, err = repo.MarkSeaOrderDirect(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, failedAudit)
	if err == nil {
		t.Fatal("writeAudit 使用非法 result 时应失败")
	}
	rolledBackLink, err := data.db.SeaMasterBillOrderLink.Get(ctx, link.ID)
	if err != nil {
		t.Fatalf("审计失败后重读活动关联失败: %v", err)
	}
	if rolledBackLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureUNDETERMINED || rolledBackLink.Version != link.Version {
		t.Fatalf("审计失败后业务写入未回滚: structure=%s version=%d", rolledBackLink.DocumentStructure, rolledBackLink.Version)
	}
	var failedAuditCount int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE action = 'order.sea_document.mark_direct'`).Scan(&failedAuditCount); err != nil {
		t.Fatalf("审计失败后统计审计行失败: %v", err)
	}
	if failedAuditCount != 0 {
		t.Fatalf("审计失败不得留下审计行，实际 %d", failedAuditCount)
	}

	// 校验 3：MarkSeaOrderDirect 标记直单
	docAgg, err = repo.MarkSeaOrderDirect(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, makeAudit())
	if err != nil {
		t.Fatalf("MarkSeaOrderDirect failed: %v", err)
	}
	if docAgg.DocumentStructure != biz.SeaDocumentStructureDirect {
		t.Fatalf("expected DIRECT, got %s", docAgg.DocumentStructure)
	}

	// 校验 4：DIRECT 下禁止添加 HBL
	_, err = repo.AddSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      "HBL-DIRECT-FAIL",
		IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
	}, makeAudit())
	if err != biz.ErrSeaDocumentDirectAddHBLBlocked {
		t.Fatalf("expected ErrSeaDocumentDirectAddHBLBlocked under DIRECT, got: %v", err)
	}

	// 校验 5：CancelSeaOrderDirect 取消直单回到未确定
	docAgg, err = repo.CancelSeaOrderDirect(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, makeAudit())
	if err != nil {
		t.Fatalf("CancelSeaOrderDirect failed: %v", err)
	}
	if docAgg.DocumentStructure != biz.SeaDocumentStructureUndetermined {
		t.Fatalf("expected UNDETERMINED, got %s", docAgg.DocumentStructure)
	}

	// 校验 5：添加 SELF_ORGANIZATION 分单（应向上解析到 hqOrg），原号无损保存
	rawHouseNo := "  COSU 000123 / 2026.B  "
	createdHB, err := repo.AddSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      rawHouseNo,
		IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
		Content: &biz.SeaBillContent{
			ShipperText: ptr("  SHIPPER 1  "),
		},
	}, makeAudit())
	if err != nil {
		t.Fatalf("AddSeaHouseBill self org failed: %v", err)
	}
	if createdHB.HouseNo != rawHouseNo {
		t.Fatalf("AddSeaHouseBill houseNo should be preserved verbatim %q, got %q", rawHouseNo, createdHB.HouseNo)
	}
	if createdHB.IssuerOrganizationID == nil || *createdHB.IssuerOrganizationID != hqOrg.ID {
		t.Fatalf("expected IssuerOrganizationID = %s (hqOrg), got %v", hqOrg.ID, createdHB.IssuerOrganizationID)
	}
	defer func() {
		_ = data.db.SeaHouseBill.DeleteOneID(createdHB.ID).Exec(ctx)
	}()

	// 重新获取聚合对象验证结构已自动转为 HOUSE
	docAgg, err = repo.GetSeaOrderDocuments(ctx, deptOrg.ID, testOrder.ID)
	if err != nil {
		t.Fatalf("GetSeaOrderDocuments after AddSeaHouseBill failed: %v", err)
	}
	if docAgg.DocumentStructure != biz.SeaDocumentStructureHouse {
		t.Fatalf("expected HOUSE after adding HBL, got %s", docAgg.DocumentStructure)
	}

	// 校验 6：UpdateSeaMasterBillContent 校验与版本冲突
	pkgCount := int32(50)
	gw := 1200.0
	cbm := 15.5
	updatedMbl, err := repo.UpdateSeaMasterBillContent(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.MasterBill.Version, &biz.SeaBillContent{
		ShipperText:   ptr("  MBL SHIPPER  "),
		PackageCount:  &pkgCount,
		GrossWeightKg: &gw,
		VolumeCbm:     &cbm,
	}, makeAudit())
	if err != nil {
		t.Fatalf("UpdateSeaMasterBillContent failed: %v", err)
	}
	if updatedMbl.Version != docAgg.MasterBill.Version+1 {
		t.Fatalf("expected MBL version %d, got %d", docAgg.MasterBill.Version+1, updatedMbl.Version)
	}
	if updatedMbl.Content.ShipperText == nil || *updatedMbl.Content.ShipperText != "MBL SHIPPER" {
		t.Fatalf("expected MBL ShipperText to be trimmed to 'MBL SHIPPER', got %v", *updatedMbl.Content.ShipperText)
	}

	// 旧版本更新应触发 409 Conflict
	_, err = repo.UpdateSeaMasterBillContent(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.MasterBill.Version, &biz.SeaBillContent{
		ShipperText: ptr("STALE UPDATE"),
	}, makeAudit())
	if err != biz.ErrSeaMasterBillConflict {
		t.Fatalf("expected ErrSeaMasterBillConflict on stale MBL version, got: %v", err)
	}

	// 校验 7：删除最后一张 HBL 如果没有传 returnToUndetermined=true 必须被拒绝
	err = repo.RemoveSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, createdHB.ID, createdHB.Version, docAgg.LinkVersion, false, makeAudit())
	if err != biz.ErrSeaDocumentDeleteLastHBLConfirmationRequired {
		t.Fatalf("expected ErrSeaDocumentDeleteLastHBLConfirmationRequired, got %v", err)
	}

	// 校验 8：HBL 版本冲突校验
	err = repo.RemoveSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, createdHB.ID, createdHB.Version+99, docAgg.LinkVersion, true, makeAudit())
	if err != biz.ErrSeaHouseBillConflict {
		t.Fatalf("expected ErrSeaHouseBillConflict on mismatched HBL version, got %v", err)
	}

	// 校验 9：删除最后一张 HBL 传 returnToUndetermined=true 成功回到 UNDETERMINED
	err = repo.RemoveSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, createdHB.ID, createdHB.Version, docAgg.LinkVersion, true, makeAudit())
	if err != nil {
		t.Fatalf("RemoveSeaHouseBill with confirmation failed: %v", err)
	}

	docAgg, err = repo.GetSeaOrderDocuments(ctx, deptOrg.ID, testOrder.ID)
	if err != nil {
		t.Fatalf("GetSeaOrderDocuments after delete failed: %v", err)
	}
	if docAgg.DocumentStructure != biz.SeaDocumentStructureUndetermined {
		t.Fatalf("expected UNDETERMINED after deleting last HBL, got %s", docAgg.DocumentStructure)
	}

	// 校验 10：当存在 CUSTOMER_PARTNER 签发的 HBL 时，修改订单客户必须被阻断
	customerHB, err := repo.AddSeaHouseBill(ctx, deptOrg.ID, actorID, testOrder.ID, docAgg.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      "HBL-CUST-002",
		IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner,
	}, makeAudit())
	if err != nil {
		t.Fatalf("AddSeaHouseBill customer partner failed: %v", err)
	}
	defer func() {
		_ = data.db.SeaHouseBill.DeleteOneID(customerHB.ID).Exec(ctx)
	}()

	otherCustomer, err := data.db.Partner.Create().
		SetOrganizationID(deptOrg.ID).
		SetCode("CUST-OTHER-" + uuid.New().String()[:8]).
		SetLegalName("另一个测试客户").
		SetNormalizedName("另一个测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建另一个测试客户失败: %v", err)
	}
	defer func() {
		_ = data.db.Partner.DeleteOne(otherCustomer).Exec(ctx)
	}()
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(otherCustomer.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建另一个测试客户角色失败: %v", err)
	}

	orderRepo := NewOrderRepo(data)
	_, err = orderRepo.UpdateDraft(ctx, deptOrg.ID, testOrder.ID, testOrder.Version, &biz.Order{
		CustomerID:     otherCustomer.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
	}, &biz.AuditEvent{OrganizationID: &deptOrg.ID, UserID: &actorID, Result: "success"})
	if err != biz.ErrOrderCustomerChangeWithHouseBillBlocked {
		t.Fatalf("expected ErrOrderCustomerChangeWithHouseBillBlocked when Customer HBL exists, got %v", err)
	}
}

func TestSeaDocument_AuditEnforcementAndRollback(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)

	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()

	t.Run("nil audit is rejected", func(t *testing.T) {
		_, err := repo.MarkSeaOrderDirect(ctx, orgID, actorID, orderID, 1, nil)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for nil audit, got %v", err)
		}
	})

	t.Run("mismatched audit is rejected", func(t *testing.T) {
		diffOrg := uuid.New()
		mismatchedAudit := &biz.AuditEvent{
			OrganizationID: &diffOrg,
			UserID:         &actorID,
			Result:         "success",
		}
		_, err := repo.MarkSeaOrderDirect(ctx, orgID, actorID, orderID, 1, mismatchedAudit)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for mismatched audit, got %v", err)
		}
	})
}

func TestSeaDocument_ConcurrentOperationsNoDeadlock(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)

	org, err := data.db.Organization.Create().
		SetCode("TEST-CONC-" + uuid.New().String()[:8]).
		SetName("并发测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org failed: %v", err)
	}
	defer func() { _ = data.db.Organization.DeleteOne(org).Exec(ctx) }()

	cust, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-CONC-" + uuid.New().String()[:8]).
		SetLegalName("并发客户").
		SetNormalizedName("并发客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("create customer failed: %v", err)
	}
	defer func() { _ = data.db.Partner.DeleteOne(cust).Exec(ctx) }()
	if _, err := data.db.PartnerRole.Create().SetPartnerID(cust.ID).SetRoleType(partnerroleent.RoleTypeCustomer).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("create customer role failed: %v", err)
	}

	carr, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CARR-CONC-" + uuid.New().String()[:8]).
		SetLegalName("并发船公司").
		SetNormalizedName("并发船公司").
		Save(ctx)
	if err != nil {
		t.Fatalf("create carrier failed: %v", err)
	}
	defer func() { _ = data.db.Partner.DeleteOne(carr).Exec(ctx) }()

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-CONC-" + uuid.New().String()[:8]).
		SetCustomerID(cust.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	defer func() { _ = data.db.Order.DeleteOne(order).Exec(ctx) }()

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetVesselName("CONC SHIP").
		SetVoyageNo("888").
		Save(ctx)
	if err != nil {
		t.Fatalf("create te failed: %v", err)
	}
	defer func() { _ = data.db.SeaTransportExecution.DeleteOne(te).Exec(ctx) }()

	masterNo := "CONCMBL" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(masterNo).
		SetNormalizedMasterNo(masterNo).
		SetIssuerPartnerID(carr.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create mbl failed: %v", err)
	}
	defer func() { _ = data.db.SeaMasterBill.DeleteOne(mbl).Exec(ctx) }()

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureUNDETERMINED).
		Save(ctx)
	if err != nil {
		t.Fatalf("create link failed: %v", err)
	}
	defer func() { _ = data.db.SeaMasterBillOrderLink.DeleteOne(link).Exec(ctx) }()

	actorID := uuid.New()

	// 并发执行多次添加分单（严格锁序保证无死锁，版本冲突被正确处理）
	var wg sync.WaitGroup
	workers := 5
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			audit := &biz.AuditEvent{
				OrganizationID: &org.ID,
				UserID:         &actorID,
				Result:         "success",
			}
			hbNo := "CONC-HB-" + uuid.New().String()[:6]
			// 读取当前 link version
			currentDoc, qErr := repo.GetSeaOrderDocuments(ctx, org.ID, order.ID)
			if qErr != nil {
				return
			}
			created, addErr := repo.AddSeaHouseBill(ctx, org.ID, actorID, order.ID, currentDoc.LinkVersion, &biz.SeaHouseBillInput{
				HouseNo:      hbNo,
				IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
			}, audit)
			if addErr == nil && created != nil {
				defer func() {
					_ = data.db.SeaHouseBill.DeleteOneID(created.ID).Exec(context.Background())
				}()
			}
		}(i)
	}
	wg.Wait()
}

func TestSeaDocument_UpdateOrderValidation(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	org, err := data.db.Organization.Create().
		SetCode("TEST-UO-" + uuid.New().String()[:8]).
		SetName("UpdateOrder测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org failed: %v", err)
	}
	defer func() { _ = data.db.Organization.DeleteOne(org).Exec(ctx) }()

	cust, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-UO-" + uuid.New().String()[:8]).
		SetLegalName("UO客户").
		SetNormalizedName("UO客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("create customer failed: %v", err)
	}
	defer func() { _ = data.db.Partner.DeleteOne(cust).Exec(ctx) }()
	if _, err := data.db.PartnerRole.Create().SetPartnerID(cust.ID).SetRoleType(partnerroleent.RoleTypeCustomer).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("create customer role failed: %v", err)
	}

	carr, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CARR-UO-" + uuid.New().String()[:8]).
		SetLegalName("UO船公司").
		SetNormalizedName("UO船公司").
		Save(ctx)
	if err != nil {
		t.Fatalf("create carrier failed: %v", err)
	}
	defer func() { _ = data.db.Partner.DeleteOne(carr).Exec(ctx) }()

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-UO-" + uuid.New().String()[:8]).
		SetCustomerID(cust.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	defer func() { _ = data.db.Order.DeleteOne(order).Exec(ctx) }()

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetVesselName("UO SHIP").
		SetVoyageNo("101").
		Save(ctx)
	if err != nil {
		t.Fatalf("create te failed: %v", err)
	}
	defer func() { _ = data.db.SeaTransportExecution.DeleteOne(te).Exec(ctx) }()

	masterNo := "UOMBL" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(masterNo).
		SetNormalizedMasterNo(masterNo).
		SetIssuerPartnerID(carr.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create mbl failed: %v", err)
	}
	defer func() { _ = data.db.SeaMasterBill.DeleteOne(mbl).Exec(ctx) }()

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureUNDETERMINED).
		Save(ctx)
	if err != nil {
		t.Fatalf("create link failed: %v", err)
	}
	defer func() { _ = data.db.SeaMasterBillOrderLink.DeleteOne(link).Exec(ctx) }()

	orderRepo := NewOrderRepo(data)
	actorID := uuid.New()
	makeAudit := func() *biz.AuditEvent {
		return &biz.AuditEvent{
			OrganizationID: &org.ID,
			UserID:         &actorID,
			Result:         "success",
		}
	}

	strDirect := biz.SeaDocumentStructureDirect

	// 1. UpdateDraft 缺少 expected_link_version 应被拒绝
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			DocumentStructure: &strDirect,
			// Missing ExpectedLinkVersion
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for missing expected_link_version in UpdateDraft, got nil")
	}

	// 2. UpdateDraft 缺少 expected_mbl_version 应被拒绝
	s := "NEW SHIPPER"
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			MasterBillContent: &biz.SeaBillContent{ShipperText: &s},
			// Missing ExpectedMblVersion
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for missing expected_mbl_version in UpdateDraft, got nil")
	}

	// 3. UpdateDraft 携带 HouseBills 应被拒绝
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			HouseBills: []*biz.SeaHouseBillInput{
				{HouseNo: "HBL1", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization},
			},
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for HouseBills in UpdateDraft, got nil")
	}

	// 4. UpdateDraft 正确携带版本号成功更新
	ver := uint64(1)
	updatedOrder, err := orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			DocumentStructure:   &strDirect,
			ExpectedLinkVersion: &ver,
			MasterBillContent:   &biz.SeaBillContent{ShipperText: &s},
			ExpectedMblVersion:  &ver,
		},
	}, makeAudit())
	if err != nil {
		t.Fatalf("UpdateDraft with valid versions failed: %v", err)
	}
	if updatedOrder == nil {
		t.Fatalf("expected non-nil updated order")
	}
}

func ptr[T any](v T) *T {
	return &v
}
