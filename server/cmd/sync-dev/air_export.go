package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybill"
	airportent "github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	personnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	taxableent "github.com/roncin/roncin-go-admin/server/internal/data/ent/taxableservice"
)

// seedAirExportOrders 补首期双票样例；只创建缺失样例，不覆盖已经人工维护的订单和主单。
// 放在海运种子最后执行，空运关系不进入海运专属演示写入路径。
func seedAirExportOrders(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	airline, err := tx.Airline.Query().Where(airlineent.IataCodeEQ("MU")).Only(ctx)
	if ent.IsNotFound(err) {
		airline, err = tx.Airline.Create().SetIataCode("MU").SetIcaoCode("CES").SetAwbPrefix("781").
			SetNameZh("中国东方航空").SetNameEn("China Eastern Airlines").SetCountryCode("CN").SetSource("dev-seed").Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("查询或创建空运样例航司: %w", err)
	}
	airports := make(map[string]*ent.Airport, 2)
	for _, spec := range []struct{ code, nameZh, nameEn, country string }{
		{"PVG", "上海浦东国际机场", "Shanghai Pudong International Airport", "CN"},
		{"LAX", "洛杉矶国际机场", "Los Angeles International Airport", "US"},
	} {
		item, queryErr := tx.Airport.Query().Where(airportent.IataCodeEQ(spec.code)).Only(ctx)
		if ent.IsNotFound(queryErr) {
			item, queryErr = tx.Airport.Create().SetIataCode(spec.code).SetNameZh(spec.nameZh).
				SetNameEn(spec.nameEn).SetCountryCode(spec.country).SetSource("dev-seed").Save(ctx)
		}
		if queryErr != nil {
			return fmt.Errorf("查询或创建空运样例机场 %s: %w", spec.code, queryErr)
		}
		airports[spec.code] = item
	}
	master, err := tx.AirMasterWaybill.Query().Where(masterent.OrganizationIDEQ(sc.company.ID), masterent.MasterNoEQ("78112345675")).Only(ctx)
	if ent.IsNotFound(err) {
		master, err = tx.AirMasterWaybill.Create().SetOrganizationID(sc.company.ID).SetMasterNo("78112345675").
			SetAirlineID(airline.ID).SetOriginAirportID(airports["PVG"].ID).SetDestinationAirportID(airports["LAX"].ID).
			SetFlightNo("MU583").SetEtd("2026-10-22T03:30:00Z").SetEta("2026-10-22T15:00:00Z").
			SetDocumentCutoffAt("2026-10-21T08:00:00Z").SetCustomsCutoffAt("2026-10-21T10:00:00Z").
			SetDeclaredPackages(30).SetDeclaredGrossWeightKg(450).SetDeclaredVolumeCbm(1.9).
			SetNotes("开发演示：主单申报量与成员合计独立，差额用于核对，不覆盖本票货量").Save(ctx)
		if err == nil {
			err = seedAirMasterHistory(ctx, sc, master, "CREATE")
		}
	}
	if err != nil {
		return fmt.Errorf("查询或创建空运样例主单: %w", err)
	}
	unit := sc.billingUnits["KG"]
	if unit == nil {
		return fmt.Errorf("空运样例缺少 KG 计费单位")
	}
	taxable, err := tx.TaxableService.Query().Where(taxableent.OrganizationIDEQ(sc.company.ID), taxableent.NameEQ("国际空运运费")).Only(ctx)
	if ent.IsNotFound(err) {
		taxable, err = tx.TaxableService.Create().SetOrganizationID(sc.company.ID).SetName("国际空运运费").SetDefaultTaxRate("0").Save(ctx)
	}
	if err != nil {
		return err
	}
	_, err = tx.FeeSetting.Query().Where(feeent.OrganizationIDEQ(sc.company.ID), feeent.FeeCodeEQ("AF")).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = tx.FeeSetting.Create().SetOrganizationID(sc.company.ID).SetFeeCode("AF").SetNameZh("空运费").
			SetNameEn("Air Freight").SetDefaultCurrency("CNY").SetBillingUnitID(unit.ID).
			SetChargeCategoryID(sc.chargeCats["BOOKING"].ID).SetTaxableServiceID(taxable.ID).
			SetTaxRate("0").Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("查询或创建空运样例费用科目: %w", err)
	}
	for _, spec := range []struct {
		orderNo, houseNo, customerCode string
		packages                       int
		weight, volume                 float64
	}{
		{"AE26100001", "AE-DEMO-A", "CUST-HY-001", 10, 100, 1.2},
		{"AE26100002", "AE-DEMO-B", "CUST-JS-002", 20, 300, 0.6},
	} {
		ord, queryErr := tx.Order.Query().Where(orderent.OrganizationIDEQ(sc.company.ID), orderent.OrderNoEQ(spec.orderNo)).Only(ctx)
		if queryErr == nil {
			sc.airOrders = append(sc.airOrders, ord)
			continue
		}
		if !ent.IsNotFound(queryErr) {
			return queryErr
		}
		customer := sc.partners[spec.customerCode]
		ord, err = tx.Order.Create().SetOrganizationID(sc.company.ID).SetOrderNo(spec.orderNo).
			SetIdempotencyKey("DEV-IDEM-" + spec.orderNo).SetCustomerID(customer.ID).
			SetBusinessType(orderent.BusinessTypeAE).SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetOrderDate("2026-10-21").SetGoodsDescription("空运共享主单演示货物").
			SetTotalPackages(spec.packages).SetTotalGrossWeightKg(spec.weight).SetTotalVolumeCbm(spec.volume).Save(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.OrderCargoItem.Create().SetOrganizationID(sc.company.ID).SetOrderID(ord.ID).
			SetCargoName("空运共享主单演示货物").SetPackageCount(spec.packages).SetGrossWeightKg(spec.weight).SetVolumeCbm(spec.volume).Save(ctx); err != nil {
			return err
		}
		if _, err = tx.OrderServiceType.Create().SetOrderID(ord.ID).SetMasterDataItemID(sc.chargeCats["BOOKING"].ID).Save(ctx); err != nil {
			return err
		}
		for _, assignment := range []struct {
			username string
			role     personnelent.Role
		}{
			{"zhangqiang", personnelent.RoleSALES}, {"liming", personnelent.RoleOPERATOR}, {"chenhua", personnelent.RoleCUSTOMER_SERVICE},
		} {
			employee := sc.users[assignment.username]
			personnel, err := tx.OrderPersonnel.Create().SetOrganizationID(sc.company.ID).SetOrderID(ord.ID).
				SetUserID(employee.ID).SetRole(assignment.role).Save(ctx)
			if err != nil {
				return err
			}
			if _, err = tx.OrderCommissionAttribution.Create().SetOrganizationID(sc.company.ID).SetOrderID(ord.ID).
				SetCustomerID(customer.ID).SetSourceAssignmentID(personnel.ID).SetEmployeeID(employee.ID).
				SetEmployeeName(employee.DisplayName).SetPersonnelRole(attributionent.PersonnelRole(assignment.role)).SetAttributedAt(time.Now()).Save(ctx); err != nil {
				return err
			}
		}
		if _, err = tx.AirMasterWaybillOrderLink.Create().SetOrganizationID(sc.company.ID).SetMasterWaybillID(master.ID).SetOrderID(ord.ID).Save(ctx); err != nil {
			return err
		}
		house, err := tx.AirHouseWaybill.Create().SetOrganizationID(sc.company.ID).SetMasterWaybillID(master.ID).SetOrderID(ord.ID).SetHouseNo(spec.houseNo).Save(ctx)
		if err != nil {
			return err
		}
		if err = tx.AirHouseWaybillVersion.Create().SetOrganizationID(sc.company.ID).SetOrderID(ord.ID).
			SetHouseWaybillID(house.ID).SetVersion(1).SetHouseNo(house.HouseNo).SetStatus(house.Status).
			SetContent(house.Content).SetActorID(sc.adminUser.ID).SetReason("开发空运建票样例").Exec(ctx); err != nil {
			return err
		}
		master, err = master.Update().AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if err = seedAirMasterHistory(ctx, sc, master, "LINK"); err != nil {
			return err
		}
		// 初始保持未审核、未建账，便于通过真实录费/审核/财务入口继续操作。
		sc.airOrders = append(sc.airOrders, ord)
	}
	return nil
}

func seedAirMasterHistory(ctx context.Context, sc *seedContext, master *ent.AirMasterWaybill, action string) error {
	content, err := json.Marshal(biz.AirMasterWaybillInput{
		MasterNo: master.MasterNo, AirlineID: master.AirlineID,
		OriginAirportID: master.OriginAirportID, DestinationAirportID: master.DestinationAirportID,
		FlightNo: master.FlightNo, ETD: master.Etd, ETA: master.Eta,
		DocumentCutoffAt: master.DocumentCutoffAt, CustomsCutoffAt: master.CustomsCutoffAt,
		Status: master.Status, DeclaredPackages: master.DeclaredPackages,
		DeclaredGrossWeightKg: master.DeclaredGrossWeightKg, DeclaredVolumeCbm: master.DeclaredVolumeCbm,
		Notes: master.Notes,
	})
	if err != nil {
		return err
	}
	return sc.tx.AirMasterWaybillHistory.Create().SetOrganizationID(sc.company.ID).SetMasterWaybillID(master.ID).
		SetVersion(master.Version).SetAction(action).SetReason("开发空运共享主单样例").SetActorID(sc.adminUser.ID).SetContent(content).Exec(ctx)
}
