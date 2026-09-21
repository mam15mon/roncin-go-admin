package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financecommissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	financecommissionruleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	masterdataitem "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	taxableserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/taxableservice"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

type seedContext struct {
	tx             *ent.Tx
	headquarters   *ent.Organization
	adminUser      *ent.User
	adminRole      *ent.Role
	users          map[string]*ent.User
	ports          map[string]*ent.Port
	shippingLines  map[string]*ent.ShippingLine
	billingUnits   map[string]*ent.BillingUnit
	taxableSvcs    map[string]*ent.TaxableService
	chargeCats     map[string]*ent.MasterDataItem
	containerSpecs map[string]*ent.MasterDataItem
	feeSettings    map[string]*ent.FeeSetting
	partners       map[string]*ent.Partner
	orders         map[string]*ent.Order
	orderFees      map[string]*ent.OrderFee
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	apply := flag.Bool("apply", false, "将开发测试数据写入数据库")
	flag.Parse()

	dbSource := strings.TrimSpace(os.Getenv("DATABASE_SOURCE"))
	if dbSource == "" {
		logger.Error("缺少 DATABASE_SOURCE 环境变量")
		os.Exit(1)
	}

	sqlDB, err := sql.Open("pgx", dbSource)
	if err != nil {
		logger.Error("连接数据库驱动失败", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()

	ctx := context.Background()
	if !*apply {
		fmt.Println("【开发测试种子同步 - 预览模式】")
		fmt.Println("当前为只读检测模式。追加 -apply 参数即可执行完整的数据写入：")
		fmt.Println("  pnpm run sync:dev")
		return
	}

	fmt.Println("【开始同步开发测试数据...】")

	tx, err := client.Tx(ctx)
	if err != nil {
		logger.Error("开启事务失败", "error", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	sc := &seedContext{
		tx:             tx,
		users:          make(map[string]*ent.User),
		ports:          make(map[string]*ent.Port),
		shippingLines:  make(map[string]*ent.ShippingLine),
		billingUnits:   make(map[string]*ent.BillingUnit),
		taxableSvcs:    make(map[string]*ent.TaxableService),
		chargeCats:     make(map[string]*ent.MasterDataItem),
		containerSpecs: make(map[string]*ent.MasterDataItem),
		feeSettings:    make(map[string]*ent.FeeSetting),
		partners:       make(map[string]*ent.Partner),
		orders:         make(map[string]*ent.Order),
		orderFees:      make(map[string]*ent.OrderFee),
	}

	if err := seedOrganizationAndStaff(ctx, sc); err != nil {
		logger.Error("组织与员工同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedReferenceData(ctx, sc); err != nil {
		logger.Error("基础参考数据（港口/船司）同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedFinanceMasterData(ctx, sc); err != nil {
		logger.Error("财务主数据（计费单位/税目/费用/汇率）同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedPartners(ctx, sc); err != nil {
		logger.Error("往来单位同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedOrdersAndFees(ctx, sc); err != nil {
		logger.Error("业务订单与费用明细同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedFinanceBillsAndCashflows(ctx, sc); err != nil {
		logger.Error("财务账单、流水与核销同步失败", "error", err)
		os.Exit(1)
	}
	if err := seedCommissionRules(ctx, sc); err != nil {
		logger.Error("提成方案与规则同步失败", "error", err)
		os.Exit(1)
	}

	if err := tx.Commit(); err != nil {
		logger.Error("提交数据事务失败", "error", err)
		os.Exit(1)
	}

	fmt.Println("==================================================")
	fmt.Println("🎉 开发测试数据同步完成 (sync:dev successful)！")
	fmt.Printf("✔ 组织体系: 总部 [%s] + 默认分公司已完备\n", sc.headquarters.Name)
	fmt.Printf("✔ 测试人员: %d 名业务员工 (张强/王丽/李明/陈华/赵芳/刘敏，默认密码: Dev123456!)\n", len(sc.users))
	fmt.Printf("✔ 基础参考: %d 个核心海港 (CNSHA/CNNBO/USLAX...) + %d 家主流船公司\n", len(sc.ports), len(sc.shippingLines))
	fmt.Printf("✔ 财务字典: %d 个计费单位 + %d 个费用科目 + 开发基准汇率已生效\n", len(sc.billingUnits), len(sc.feeSettings))
	fmt.Printf("✔ 往来单位: %d 家客商档案 (包含进出口商、电商、船代、车队、报关行及对公账户/联系人)\n", len(sc.partners))
	fmt.Printf("✔ 海运订单: %d 票全生命周期真实订单 (含草稿、已订舱、在途、锁定及箱货明细)\n", len(sc.orders))
	fmt.Printf("✔ 费用账单: 应收/应付费用明细已录入，已生成核销流水、草稿/确认账单及提成规则\n")
	fmt.Println("==================================================")
}

// 1. 组织架构与人员
func seedOrganizationAndStaff(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq, err := tx.Organization.Query().Where(organizationent.KindEQ(organizationent.KindHeadquarters)).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if hq == nil {
		created, createErr := tx.Organization.Create().
			SetCode("HQ").
			SetName("融迅供应链管理总部").
			SetKind(organizationent.KindHeadquarters).
			SetBaseCurrency("CNY").
			SetEnabledCurrencies([]string{"CNY", "USD", "EUR", "HKD"}).
			SetEnabled(true).
			Save(ctx)
		if createErr != nil {
			return fmt.Errorf("创建总部组织: %w", createErr)
		}
		hq = created
		_ = data.CreateDefaultNumberRules(ctx, tx, hq.ID)
	}
	sc.headquarters = hq

	// 补充分公司
	_, _ = data.CreateDefaultBranchCompanies(ctx, tx, hq.ID)

	// 查找系统管理员角色
	adminRole, err := tx.Role.Query().Where(roleent.OrganizationIDEQ(hq.ID), roleent.CodeEQ("administrator")).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	sc.adminRole = adminRole

	// 查找已有 admin 用户
	adminUser, _ := tx.User.Query().Where(userent.UsernameEQ("admin")).First(ctx)
	if adminUser == nil {
		adminUser, _ = tx.User.Query().Where(userent.IsBootstrapAdmin(true)).First(ctx)
	}
	sc.adminUser = adminUser

	// 测试员工列表
	staffList := []struct {
		username    string
		displayName string
	}{
		{"zhangqiang", "张强 (销售经理)"},
		{"wangli", "王丽 (资深销售)"},
		{"liming", "李明 (海运操作主管)"},
		{"chenhua", "陈华 (单证客服专员)"},
		{"zhaofang", "赵芳 (财务经理)"},
		{"liumin", "刘敏 (财务结算出纳)"},
	}

	pwdHash, _ := password.Hash("Dev123456!")
	for _, s := range staffList {
		u, err := tx.User.Query().Where(userent.UsernameEQ(s.username)).First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if u == nil {
			created, err := tx.User.Create().
				SetUsername(s.username).
				SetDisplayName(s.displayName).
				SetPasswordHash(pwdHash).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建测试员工 %s: %w", s.username, err)
			}
			u = created
		}
		sc.users[s.username] = u

		// 绑定总部 Membership
		mExists, _ := tx.Membership.Query().Where(membershipent.UserID(u.ID), membershipent.OrganizationID(hq.ID)).Exist(ctx)
		if !mExists {
			m, mErr := tx.Membership.Create().
				SetUserID(u.ID).
				SetOrganizationID(hq.ID).
				SetPrimary(true).
				SetEnabled(true).
				Save(ctx)
			if mErr == nil && adminRole != nil {
				_, _ = tx.RoleAssignment.Create().SetMembershipID(m.ID).SetRoleID(adminRole.ID).Save(ctx)
			}
		}
	}
	if sc.adminUser == nil && sc.users["zhangqiang"] != nil {
		sc.adminUser = sc.users["zhangqiang"]
	}
	return nil
}

// 2. 基础参考数据（港口、船司）
func seedReferenceData(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	ports := []struct {
		code, nameZh, nameEn, country string
	}{
		{"CNSHA", "上海港", "Port of Shanghai", "CN"},
		{"CNNBO", "宁波舟山港", "Port of Ningbo-Zhoushan", "CN"},
		{"CNSZX", "深圳盐田港", "Port of Shenzhen", "CN"},
		{"CNQDG", "青岛港", "Port of Qingdao", "CN"},
		{"USLAX", "洛杉矶港", "Port of Los Angeles", "US"},
		{"USLGB", "长滩港", "Port of Long Beach", "US"},
		{"DEHAM", "汉堡港", "Port of Hamburg", "DE"},
		{"SGSIN", "新加坡港", "Port of Singapore", "SG"},
		{"NLRTM", "鹿特丹港", "Port of Rotterdam", "NL"},
	}

	for idx, p := range ports {
		found, _ := tx.Port.Query().Where(portent.UnLocodeEQ(p.code), portent.OrganizationIDIsNil()).First(ctx)
		if found == nil {
			created, err := tx.Port.Create().
				SetUnLocode(p.code).
				SetNameZh(p.nameZh).
				SetNameEn(p.nameEn).
				SetCountryCode(p.country).
				SetTransportModes([]string{"maritime"}).
				SetSortOrder((idx + 1) * 10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建港口 %s: %w", p.code, err)
			}
			found = created
		}
		sc.ports[p.code] = found
	}

	shippingLines := []struct {
		scac, nameZh, nameEn, country string
	}{
		{"COSU", "中远海运集运", "COSCO SHIPPING Lines", "CN"},
		{"MSCU", "地中海航运", "MSC Mediterranean Shipping", "CH"},
		{"MAEU", "马士基航运", "Maersk Line", "DK"},
		{"ONEY", "海洋网联船务", "Ocean Network Express", "SG"},
		{"EGLV", "长荣海运", "Evergreen Line", "TW"},
	}

	for idx, s := range shippingLines {
		found, _ := tx.ShippingLine.Query().Where(shippinglineent.ScacCodeEQ(s.scac)).First(ctx)
		if found == nil {
			created, err := tx.ShippingLine.Create().
				SetScacCode(s.scac).
				SetNameZh(s.nameZh).
				SetNameEn(s.nameEn).
				SetCountryCode(s.country).
				SetSortOrder((idx + 1) * 10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建船公司 %s: %w", s.scac, err)
			}
			found = created
		}
		sc.shippingLines[s.scac] = found
	}
	return nil
}

// 3. 财务主数据（计费单位、应税服务、费用目录、汇率）
func seedFinanceMasterData(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq := sc.headquarters

	// 计费单位
	units := []struct {
		code, name  string
		isContainer bool
	}{
		{"CONT", "箱", true},
		{"BL", "票", false},
		{"CBM", "立方米", false},
		{"TON", "吨", false},
		{"SET", "套", false},
		{"DOC", "份", false},
	}
	for idx, u := range units {
		found, _ := tx.BillingUnit.Query().Where(billingunitent.CodeEQ(u.code)).First(ctx)
		if found == nil {
			created, err := tx.BillingUnit.Create().
				SetCode(u.code).
				SetName(u.name).
				SetIsContainerUnit(u.isContainer).
				SetSortOrder((idx + 1) * 10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建计费单位 %s: %w", u.code, err)
			}
			found = created
		}
		sc.billingUnits[u.code] = found
	}

	// 应税服务
	services := []struct {
		name, taxRate string
	}{
		{"国际海运运费", "0.00"},
		{"海运代理订舱服务", "6.00"},
		{"港口操作及港杂服务", "6.00"},
		{"报关报检代理服务", "6.00"},
		{"集装箱陆路运输服务", "9.00"},
	}
	for _, s := range services {
		found, _ := tx.TaxableService.Query().Where(taxableserviceent.OrganizationIDEQ(hq.ID), taxableserviceent.NameEQ(s.name)).First(ctx)
		if found == nil {
			created, err := tx.TaxableService.Create().
				SetOrganizationID(hq.ID).
				SetName(s.name).
				SetDefaultTaxRate(s.taxRate).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建应税服务 %s: %w", s.name, err)
			}
			found = created
		}
		sc.taxableSvcs[s.name] = found
	}

	// 费用大类
	chargeCatCodes := []string{"BOOKING", "TRUCKING", "STUFFING", "CUSTOMS_EXPORT"}
	for _, c := range chargeCatCodes {
		found, _ := tx.MasterDataItem.Query().Where(masterdataitem.KindEQ(masterdataitem.KindChargeCategory), masterdataitem.CodeEQ(c)).First(ctx)
		if found == nil {
			name := c
			switch c {
			case "BOOKING":
				name = "订舱"
			case "TRUCKING":
				name = "拖车"
			case "STUFFING":
				name = "内装"
			case "CUSTOMS_EXPORT":
				name = "报关"
			}
			created, err := tx.MasterDataItem.Create().
				SetKind(masterdataitem.KindChargeCategory).
				SetCode(c).
				SetName(name).
				SetSource("system").
				SetSortOrder(10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建费用大类 %s: %w", c, err)
			}
			found = created
		}
		sc.chargeCats[c] = found
	}

	// 箱型
	containerSpecCodes := []string{"20GP", "40HQ", "40GP"}
	for _, c := range containerSpecCodes {
		found, _ := tx.MasterDataItem.Query().Where(masterdataitem.KindEQ(masterdataitem.KindContainerSpec), masterdataitem.CodeEQ(c)).First(ctx)
		if found == nil {
			created, err := tx.MasterDataItem.Create().
				SetKind(masterdataitem.KindContainerSpec).
				SetCode(c).
				SetName(c).
				SetSource("system").
				SetSortOrder(10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建箱型 %s: %w", c, err)
			}
			found = created
		}
		sc.containerSpecs[c] = found
	}

	// 费用科目
	feeList := []struct {
		code, nameZh, nameEn, currency, unit, cat, svc, taxRate string
	}{
		{"OF", "海运费", "Ocean Freight", "USD", "CONT", "BOOKING", "国际海运运费", "0.00"},
		{"THC", "码头操作费", "Terminal Handling Charge", "CNY", "CONT", "BOOKING", "港口操作及港杂服务", "6.00"},
		{"DOC", "文件费", "Documentation Fee", "CNY", "BL", "BOOKING", "海运代理订舱服务", "6.00"},
		{"SEAL", "封条费", "Seal Fee", "CNY", "CONT", "BOOKING", "海运代理订舱服务", "6.00"},
		{"EIR", "打单费", "Equipment Interchange Receipt", "CNY", "CONT", "BOOKING", "海运代理订舱服务", "6.00"},
		{"TLX", "电放费", "Telex Release Fee", "CNY", "BL", "BOOKING", "海运代理订舱服务", "6.00"},
		{"TRUCK", "集装箱拖车费", "Trucking Fee", "CNY", "CONT", "TRUCKING", "集装箱陆路运输服务", "9.00"},
		{"CUSTOMS", "代理报关费", "Customs Clearance Fee", "CNY", "BL", "CUSTOMS_EXPORT", "报关报检代理服务", "6.00"},
		{"STORAGE", "码头堆存费", "Storage Fee", "CNY", "CONT", "BOOKING", "港口操作及港杂服务", "6.00"},
		{"VGM", "重量验证费", "VGM Fee", "CNY", "CONT", "BOOKING", "港口操作及港杂服务", "6.00"},
	}

	for idx, f := range feeList {
		found, _ := tx.FeeSetting.Query().Where(feesettingent.OrganizationIDEQ(hq.ID), feesettingent.FeeCodeEQ(f.code)).First(ctx)
		if found == nil {
			created, err := tx.FeeSetting.Create().
				SetOrganizationID(hq.ID).
				SetFeeCode(f.code).
				SetNameZh(f.nameZh).
				SetNameEn(f.nameEn).
				SetDefaultCurrency(f.currency).
				SetBillingUnitID(sc.billingUnits[f.unit].ID).
				SetChargeCategoryID(sc.chargeCats[f.cat].ID).
				SetTaxableServiceID(sc.taxableSvcs[f.svc].ID).
				SetTaxRate(f.taxRate).
				SetSortOrder((idx + 1) * 10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建费用科目 %s: %w", f.code, err)
			}
			found = created
		}
		sc.feeSettings[f.code] = found
	}

	// 汇率
	effFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rates := []struct {
		from, to, rate, ar, ap string
	}{
		{"USD", "CNY", "7.25000000", "7.26500000", "7.23500000"},
		{"EUR", "CNY", "7.85000000", "7.87000000", "7.83000000"},
		{"HKD", "CNY", "0.92500000", "0.92800000", "0.92200000"},
	}
	for _, r := range rates {
		rFound, _ := tx.ExchangeRateSetting.Query().Where(
			exchangeratesettingent.OrganizationIDEQ(hq.ID),
			exchangeratesettingent.FromCurrencyEQ(r.from),
			exchangeratesettingent.ToCurrencyEQ(r.to),
		).First(ctx)
		if rFound == nil {
			_, err := tx.ExchangeRateSetting.Create().
				SetOrganizationID(hq.ID).
				SetFromCurrency(r.from).
				SetToCurrency(r.to).
				SetRate(r.rate).
				SetArRate(r.ar).
				SetApRate(r.ap).
				SetEffectiveFrom(effFrom).
				SetIsActive(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建基准汇率 %s/%s: %w", r.from, r.to, err)
			}
		}
	}
	return nil
}

// 4. 往来单位（客户、供应商）
func seedPartners(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq := sc.headquarters

	type partnerSeedDef struct {
		code, name, uscc, role, contactName, phone string
		isCust                                     bool
		accCNY, accUSD                             string
		salesRep                                   string
	}

	seeds := []partnerSeedDef{
		{
			code: "CUST-HY-001", name: "上海宏远国际贸易进出口有限公司", uscc: "91310115MA1H789012",
			role: "customer", contactName: "陈志高", phone: "13812345678", isCust: true,
			accCNY: "3100661234567890", accUSD: "987654321001", salesRep: "zhangqiang",
		},
		{
			code: "CUST-JS-002", name: "深圳市极速跨境智能实业有限公司", uscc: "91440300MA5F888999",
			role: "customer", contactName: "林思捷", phone: "13988776655", isCust: true,
			accCNY: "7559123488880001", accUSD: "888800029999", salesRep: "wangli",
		},
		{
			code: "CUST-ML-003", name: "浙江美林工艺家具有限公司", uscc: "91330100MA2B334455",
			role: "customer", contactName: "汪建国", phone: "13766554433", isCust: true,
			accCNY: "5719000122334455", accUSD: "665544332211", salesRep: "zhangqiang",
		},
		{
			code: "CUST-HT-004", name: "宁波恒泰汽车零部件制造有限公司", uscc: "91330200MA28667788",
			role: "customer", contactName: "周晓波", phone: "13611223344", isCust: true,
			accCNY: "5749888811223344", accUSD: "112233445566", salesRep: "wangli",
		},
		{
			code: "CUST-TG-005", name: "TransGlobal Trading (HK) Co., Limited", uscc: "CR-78901234",
			role: "customer", contactName: "Michael Wong", phone: "+852-98765432", isCust: true,
			accCNY: "", accUSD: "012-888-12345678", salesRep: "zhangqiang",
		},
		{
			code: "SUPP-COSCO-01", name: "中远海运集装箱运输有限公司", uscc: "913100001322000111",
			role: "supplier", contactName: "王经理 (订舱处)", phone: "021-65966666", isCust: false,
			accCNY: "1001234509006677", accUSD: "1001234509008899",
		},
		{
			code: "SUPP-MSC-02", name: "地中海航运代理（上海）有限公司", uscc: "913100007178000222",
			role: "supplier", contactName: "张专员 (客服部)", phone: "021-38558888", isCust: false,
			accCNY: "3100158888880001", accUSD: "3100158888880002",
		},
		{
			code: "SUPP-HTBG-03", name: "上海海通报关服务有限公司", uscc: "913101157456000333",
			role: "supplier", contactName: "朱师傅 (洋山驻点)", phone: "13500112233", isCust: false,
			accCNY: "1219088899990001", accUSD: "",
		},
		{
			code: "SUPP-YJTC-04", name: "上海远集集装箱汽车运输有限公司", uscc: "913101136622000444",
			role: "supplier", contactName: "赵调度 (车队)", phone: "13912348899", isCust: false,
			accCNY: "3100778899001122", accUSD: "",
		},
		{
			code: "SUPP-SHCC-05", name: "上海申海集装箱仓储服务有限公司", uscc: "913101151234000555",
			role: "supplier", contactName: "钱主管 (仓库)", phone: "13800223344", isCust: false,
			accCNY: "3100990011223344", accUSD: "",
		},
	}

	for _, s := range seeds {
		p, _ := tx.Partner.Query().Where(partnerent.OrganizationIDEQ(hq.ID), partnerent.CodeEQ(s.code)).First(ctx)
		if p == nil {
			created, err := tx.Partner.Create().
				SetOrganizationID(hq.ID).
				SetCode(s.code).
				SetLegalName(s.name).
				SetNormalizedName(s.name).
				SetUnifiedSocialCreditCode(s.uscc).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建往来单位 %s: %w", s.code, err)
			}
			p = created
		}
		sc.partners[s.code] = p

		// 角色
		roleType := partnerroleent.RoleType(s.role)
		pr, _ := tx.PartnerRole.Query().Where(partnerroleent.PartnerIDEQ(p.ID), partnerroleent.RoleTypeEQ(roleType)).First(ctx)
		if pr == nil {
			created, err := tx.PartnerRole.Create().
				SetPartnerID(p.ID).
				SetRoleType(roleType).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建往来单位角色 %s/%s: %w", s.code, s.role, err)
			}
			pr = created
		}

		// 结算规则
		srExists, _ := tx.PartnerSettlementRule.Query().Where(partnersettlementruleent.PartnerRoleIDEQ(pr.ID)).Exist(ctx)
		if !srExists {
			method := partnersettlementruleent.SettlementMethodMonthly
			limit := int64(50000000) // 50万
			days := 30
			if !s.isCust {
				limit = 0
				days = 15
			}
			_, _ = tx.PartnerSettlementRule.Create().
				SetPartnerRoleID(pr.ID).
				SetStatementMode(partnersettlementruleent.StatementModeSingle).
				SetSettlementMethod(method).
				SetSettlementCurrency("CNY").
				SetCreditLimitMinor(limit).
				SetPaymentTermsDays(days).
				SetIsActive(true).
				Save(ctx)
		}

		// 银行结算账户
		usage := partneraccountent.UsageRECEIVABLE
		if !s.isCust {
			usage = partneraccountent.UsagePAYABLE
		}
		if s.accCNY != "" {
			accExists, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(p.ID), partneraccountent.AccountNoEQ(s.accCNY)).Exist(ctx)
			if !accExists {
				_, _ = tx.PartnerAccount.Create().
					SetPartnerID(p.ID).
					SetName(s.name + " (CNY基本户)").
					SetAccountHolder(s.name).
					SetCurrency("CNY").
					SetBankName("中国工商银行上海自贸试验区分行").
					SetAccountNo(s.accCNY).
					SetUsage(usage).
					SetIsDefaultReceivable(s.isCust).
					SetIsDefaultPayable(!s.isCust).
					SetEnabled(true).
					Save(ctx)
			}
		}
		if s.accUSD != "" {
			accExists, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(p.ID), partneraccountent.AccountNoEQ(s.accUSD)).Exist(ctx)
			if !accExists {
				_, _ = tx.PartnerAccount.Create().
					SetPartnerID(p.ID).
					SetName(s.name + " (USD外汇户)").
					SetAccountHolder(s.name).
					SetCurrency("USD").
					SetBankName("交通银行上海分行外汇业务部").
					SetAccountNo(s.accUSD).
					SetUsage(usage).
					SetIsDefaultReceivable(s.isCust).
					SetIsDefaultPayable(!s.isCust).
					SetEnabled(true).
					Save(ctx)
			}
		}

		// 联系人
		cExists, _ := tx.PartnerContact.Query().Where(partnercontactent.PartnerIDEQ(p.ID), partnercontactent.NameEQ(s.contactName)).Exist(ctx)
		if !cExists {
			_, _ = tx.PartnerContact.Create().
				SetPartnerID(p.ID).
				SetName(s.contactName).
				SetPhone(s.phone).
				SetIsPrimary(true).
				Save(ctx)
		}

		// 销售员指派
		if s.salesRep != "" && sc.users[s.salesRep] != nil {
			u := sc.users[s.salesRep]
			aExists, _ := tx.PartnerAssignment.Query().Where(partnerassignmentent.PartnerIDEQ(p.ID), partnerassignmentent.RoleEQ(partnerassignmentent.RoleSALES)).Exist(ctx)
			if !aExists {
				_, _ = tx.PartnerAssignment.Create().
					SetPartnerID(p.ID).
					SetUserID(u.ID).
					SetOrganizationID(hq.ID).
					SetRole(partnerassignmentent.RoleSALES).
					Save(ctx)
			}
		}
	}
	return nil
}

// 5. 业务订单与费用明细
func seedOrdersAndFees(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq := sc.headquarters

	type orderSeedDef struct {
		orderNo      string
		custCode     string
		refNo        string
		lineScac     string
		vessel       string
		voyage       string
		mblNo        string
		pol, pod     string
		flowStatus   orderent.FlowStatus
		term         orderent.TradeTerm
		payTerm      orderent.PaymentTerm
		shipType     orderent.ShipmentType
		desc         string
		pkgs         int
		weight       float64
		volume       float64
		isLocked     bool
		salesRep     string
		opRep        string
		docRep       string
		container    string
		sealNo       string
		docStructure seamasterbillorderlinkent.DocumentStructure
		hblNo        string
		hblShipper   string
		hblConsignee string
		hblNotify    string
		fees         []feeItemDef
	}

	orders := []orderSeedDef{
		{
			orderNo: "SE26090001", custCode: "CUST-HY-001", refNo: "PO-HY-260901",
			lineScac: "COSU", vessel: "COSCO PRIDE", voyage: "042W", mblNo: "COSU89123450",
			pol: "CNSHA", pod: "USLAX", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "智能小家电与电炸锅 (40HQ整箱)", pkgs: 500, weight: 6800.0, volume: 58.5, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "COSU8912345", sealNo: "COS887612",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费", party: "CUST-HY-001", unit: "CONT", qty: "1.0000", price: "2200.0000", total: "2200.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "15950.00000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-HY-001", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "文件费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "CUSTOMS", name: "代理报关费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "350.0000", total: "350.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "350.00000000", taxRate: "6.00"},
				{dir: "PAYABLE", code: "OF", name: "海运费(付中远)", party: "SUPP-COSCO-01", unit: "CONT", qty: "1.0000", price: "1950.0000", total: "1950.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "14137.50000000", taxRate: "0.00"},
				{dir: "PAYABLE", code: "THC", name: "码头操作费(付中远)", party: "SUPP-COSCO-01", unit: "CONT", qty: "1.0000", price: "950.0000", total: "950.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "950.00000000", taxRate: "6.00"},
				{dir: "PAYABLE", code: "CUSTOMS", name: "代理报关费(付海通)", party: "SUPP-HTBG-03", unit: "BL", qty: "1.0000", price: "200.0000", total: "200.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "200.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090002", custCode: "CUST-ML-003", refNo: "PO-ML-260902",
			lineScac: "MSCU", vessel: "MSC ISABELLA", voyage: "2635W", mblNo: "MSCU67812300",
			pol: "CNNBO", pod: "DEHAM", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "户外竹木花园家具 (2*40HQ整箱)", pkgs: 920, weight: 15200.0, volume: 130.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "MSCU6781230", sealNo: "MSC998811",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-ML-003", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "文件费", party: "CUST-ML-003", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "TRUCK", name: "集装箱拖车费", party: "CUST-ML-003", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "9.00"},
			},
		},
		{
			orderNo: "SE26090003", custCode: "CUST-JS-002", refNo: "PO-JS-260903",
			lineScac: "MSCU", vessel: "MSC MARINA", voyage: "018E", mblNo: "MSCU11223344",
			pol: "CNSZX", pod: "USLGB", flowStatus: orderent.FlowStatusDRAFT,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "跨境电商 3C 数码电子及配件", pkgs: 300, weight: 3200.0, volume: 25.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "MSCU3344556", sealNo: "MSC778899",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees:         []feeItemDef{},
		},
		{
			orderNo: "SE26090004", custCode: "CUST-HT-004", refNo: "PO-HT-260904",
			lineScac: "MAEU", vessel: "MAERSK MC-KINNEY", voyage: "2609W", mblNo: "MAEU99887766",
			pol: "CNNBO", pod: "NLRTM", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "汽车高压精密铝铸件 (重箱)", pkgs: 600, weight: 18500.0, volume: 45.0, isLocked: true,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "MAEU5566778", sealNo: "MAE112233",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费", party: "CUST-HT-004", unit: "CONT", qty: "1.0000", price: "2400.0000", total: "2400.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "17400.00000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-HT-004", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090005", custCode: "CUST-HY-001", refNo: "PO-HY-260905",
			lineScac: "ONEY", vessel: "ONE INTEGRITY", voyage: "005S", mblNo: "ONEY12345678",
			pol: "CNSHA", pod: "SGSIN", flowStatus: orderent.FlowStatusSPACE_ALLOCATED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeLCL,
			desc: "展会样品展示架及印刷宣传物料 (拼箱分单)", pkgs: 50, weight: 650.0, volume: 5.8, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "", sealNo: "",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090005",
			hblShipper:   "上海宏远国际贸易进出口有限公司\nSHANGHAI HONGYUAN TRADING CO., LTD.",
			hblConsignee: "SINGAPORE EXHIBITION LOGISTICS PTE LTD\n10 ANSON ROAD #26-04, SINGAPORE",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运拼箱运费", party: "CUST-HY-001", unit: "CBM", qty: "5.8000", price: "45.0000", total: "261.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "1892.25000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "分单文件费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "300.0000", total: "300.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "300.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090006", custCode: "CUST-JS-002", refNo: "PO-JS-260906",
			lineScac: "COSU", vessel: "COSCO FAITH", voyage: "099W", mblNo: "COSU66554433",
			pol: "CNSZX", pod: "USLAX", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "高保真无线耳机及音响设备 (40HQ整箱)", pkgs: 800, weight: 5600.0, volume: 62.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "COSU1199887", sealNo: "COS665544",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费", party: "CUST-JS-002", unit: "CONT", qty: "1.0000", price: "2100.0000", total: "2100.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "15225.00000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-JS-002", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090007", custCode: "CUST-TG-005", refNo: "PO-TG-260907",
			lineScac: "COSU", vessel: "COSCO ASIA", voyage: "088E", mblNo: "COSU77889901",
			pol: "CNNBO", pod: "USLGB", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "精装智能办公桌椅配件 (FOB指定货主分单)", pkgs: 450, weight: 8500.0, volume: 65.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "COSU5566778", sealNo: "COS443322",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090007",
			hblShipper:   "浙江美林工艺家具有限公司\nZHEJIANG MEILIN CRAFT FURNITURE CO., LTD.",
			hblConsignee: "TO ORDER OF SHIPPER",
			hblNotify:    "TRANSGLOBAL SUPPLY CHAIN (USA) INC.\n2200 PACIFIC AVE, LONG BEACH, CA 90806",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "DOC", name: "分单文件费", party: "CUST-TG-005", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-TG-005", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
				{dir: "PAYABLE", code: "THC", name: "码头操作费(付中远)", party: "SUPP-COSCO-01", unit: "CONT", qty: "1.0000", price: "950.0000", total: "950.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "950.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090008", custCode: "CUST-HY-001", refNo: "PO-HY-260908",
			lineScac: "MSCU", vessel: "MSC OSCAR", voyage: "2640E", mblNo: "MSCU55667788",
			pol: "CNSHA", pod: "USLAX", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "智能扫地机器人及配件 (共主单合拼A票)", pkgs: 200, weight: 2400.0, volume: 22.5, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "MSCU9900112", sealNo: "MSC881122",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090008",
			hblShipper:   "上海宏远国际贸易进出口有限公司\nSHANGHAI HONGYUAN TRADING CO., LTD.",
			hblConsignee: "AMERICAN SMART ROBOTICS CORP.\n1230 HARBOR BLVD, LOS ANGELES, CA",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费(A票分摊)", party: "CUST-HY-001", unit: "CONT", qty: "1.0000", price: "1200.0000", total: "1200.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "8700.00000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "分提单制单费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090009", custCode: "CUST-JS-002", refNo: "PO-JS-260909",
			lineScac: "MSCU", vessel: "MSC OSCAR", voyage: "2640E", mblNo: "MSCU55667788",
			pol: "CNSHA", pod: "USLAX", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "智能穿戴手表及蓝牙音箱 (共主单合拼B票)", pkgs: 350, weight: 3100.0, volume: 28.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "MSCU9900112", sealNo: "MSC881122",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090009",
			hblShipper:   "深圳市极速跨境智能实业有限公司\nSHENZHEN SPEED CROSS-BORDER INDUSTRIAL CO.",
			hblConsignee: "CALIFORNIA DIGITAL ACCESSORIES LLC\n900 WILSHIRE BLVD, LOS ANGELES, CA",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费(B票分摊)", party: "CUST-JS-002", unit: "CONT", qty: "1.0000", price: "1400.0000", total: "1400.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "10150.00000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "分提单制单费", party: "CUST-JS-002", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
			},
		},
	}

	for _, o := range orders {
		cust := sc.partners[o.custCode]
		line := sc.shippingLines[o.lineScac]
		pol := sc.ports[o.pol]
		pod := sc.ports[o.pod]

		ord, _ := tx.Order.Query().Where(orderent.OrganizationIDEQ(hq.ID), orderent.OrderNoEQ(o.orderNo)).First(ctx)
		if ord == nil {
			idempotencyKey := "DEV-IDEM-" + o.orderNo
			vesselVoyage := o.vessel + " / " + o.voyage
			builder := tx.Order.Create().
				SetOrganizationID(hq.ID).
				SetOrderNo(o.orderNo).
				SetIdempotencyKey(idempotencyKey).
				SetCustomerID(cust.ID).
				SetCustomerReferenceNo(o.refNo).
				SetShippingLineID(line.ID).
				SetBusinessType(orderent.BusinessTypeSE).
				SetTradeDirection(orderent.TradeDirectionExport).
				SetTradeTerm(o.term).
				SetPaymentTerm(o.payTerm).
				SetShipmentType(o.shipType).
				SetFlowStatus(o.flowStatus).
				SetGoodsDescription(o.desc).
				SetTotalPackages(o.pkgs).
				SetTotalGrossWeightKg(o.weight).
				SetTotalVolumeCbm(o.volume).
				SetOriginLocationID(pol.ID).
				SetDestinationLocationID(pod.ID).
				SetDischargeLocationID(pod.ID).
				SetVesselVoyage(vesselVoyage).
				SetEtd("2026-09-25 18:00").
				SetEta("2026-10-10 08:00").
				SetOrderDate("2026-09-20").
				SetBookingNo("BKG-" + o.orderNo)

			if o.isLocked && sc.adminUser != nil {
				builder.SetLockedAt(time.Now()).
					SetLockedBy(sc.adminUser.ID).
					SetLockSource(orderent.LockSourceMANUAL)
			}

			created, err := builder.Save(ctx)
			if err != nil {
				return fmt.Errorf("创建订单 %s: %w", o.orderNo, err)
			}
			ord = created
		}
		sc.orders[o.orderNo] = ord

		// 订单协作人员
		assigns := []struct {
			username string
			role     orderpersonnelent.Role
		}{
			{o.salesRep, orderpersonnelent.RoleSALES},
			{o.opRep, orderpersonnelent.RoleOPERATOR},
			{o.docRep, orderpersonnelent.RoleDOCUMENT},
		}
		for _, a := range assigns {
			u := sc.users[a.username]
			if u == nil {
				continue
			}
			pExists, _ := tx.OrderPersonnel.Query().Where(orderpersonnelent.OrderIDEQ(ord.ID), orderpersonnelent.RoleEQ(a.role)).Exist(ctx)
			if !pExists {
				_, _ = tx.OrderPersonnel.Create().
					SetOrderID(ord.ID).
					SetUserID(u.ID).
					SetOrganizationID(hq.ID).
					SetRole(a.role).
					Save(ctx)
			}
		}

		// 货物明细
		cargoExists, _ := tx.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(ord.ID)).Exist(ctx)
		if !cargoExists {
			_, _ = tx.OrderCargoItem.Create().
				SetOrganizationID(hq.ID).
				SetOrderID(ord.ID).
				SetCargoName(o.desc).
				SetPackageCount(o.pkgs).
				SetGrossWeightKg(o.weight).
				SetVolumeCbm(o.volume).
				Save(ctx)
		}

		// 集装箱
		if o.container != "" {
			cExists, _ := tx.OrderContainer.Query().Where(ordercontainerent.OrderIDEQ(ord.ID), ordercontainerent.ContainerNoEQ(o.container)).Exist(ctx)
			if !cExists {
				spec := sc.containerSpecs["40HQ"]
				if spec == nil {
					spec = sc.containerSpecs["20GP"]
				}
				_, _ = tx.OrderContainer.Create().
					SetOrganizationID(hq.ID).
					SetOrderID(ord.ID).
					SetContainerNo(o.container).
					SetContainerSpecID(spec.ID).
					SetPackageCount(o.pkgs).
					SetSealNo(o.sealNo).
					SetGrossWeightKg(o.weight).
					SetVolumeCbm(o.volume).
					Save(ctx)
			}
		}

		// 海运主单与航程
		if o.mblNo != "" {
			mbl, _ := tx.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(hq.ID), seamasterbillent.MasterNoEQ(o.mblNo)).First(ctx)
			if mbl == nil {
				createdMbl, err := tx.SeaMasterBill.Create().
					SetOrganizationID(hq.ID).
					SetShippingLineID(line.ID).
					SetMasterNo(o.mblNo).
					SetNormalizedMasterNo(o.mblNo).
					SetStatus(seamasterbillent.StatusCONFIRMED).
					Save(ctx)
				if err == nil {
					mbl = createdMbl
				}
			}
			if mbl != nil {
				exec, _ := tx.SeaTransportExecution.Query().Where(
					seatransportexecutionent.OrganizationIDEQ(hq.ID),
					seatransportexecutionent.ShippingLineIDEQ(line.ID),
					seatransportexecutionent.VesselNameEQ(o.vessel),
					seatransportexecutionent.VoyageNoEQ(o.voyage),
				).First(ctx)
				if exec == nil {
					createdExec, err := tx.SeaTransportExecution.Create().
						SetOrganizationID(hq.ID).
						SetShippingLineID(line.ID).
						SetOriginLocationID(pol.ID).
						SetDischargeLocationID(pod.ID).
						SetVesselName(o.vessel).
						SetVoyageNo(o.voyage).
						Save(ctx)
					if err == nil {
						exec = createdExec
					}
				}
				if exec != nil {
					link, _ := tx.SeaMasterBillOrderLink.Query().Where(
						seamasterbillorderlinkent.OrganizationIDEQ(hq.ID),
						seamasterbillorderlinkent.OrderIDEQ(ord.ID),
					).First(ctx)
					docStruct := seamasterbillorderlinkent.DocumentStructureDIRECT
					if o.docStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
						docStruct = seamasterbillorderlinkent.DocumentStructureHOUSE
					}
					if link == nil {
						_, _ = tx.SeaMasterBillOrderLink.Create().
							SetOrganizationID(hq.ID).
							SetMasterBillID(mbl.ID).
							SetTransportExecutionID(exec.ID).
							SetOrderID(ord.ID).
							SetStatus(seamasterbillorderlinkent.StatusACTIVE).
							SetDocumentStructure(docStruct).
							Save(ctx)
					} else if link.DocumentStructure != docStruct {
						_ = link.Update().SetDocumentStructure(docStruct).Exec(ctx)
					}

					// 如果是 HOUSE 主分单结构且配置了分提单号，创建 SeaHouseBill
					if docStruct == seamasterbillorderlinkent.DocumentStructureHOUSE && o.hblNo != "" {
						hb, _ := tx.SeaHouseBill.Query().Where(
							seahousebillent.OrganizationIDEQ(hq.ID),
							seahousebillent.OrderIDEQ(ord.ID),
						).First(ctx)
						if hb == nil {
							freightTerms := "FREIGHT PREPAID"
							if o.payTerm == orderent.PaymentTermCOLLECT {
								freightTerms = "FREIGHT COLLECT"
							}
							_, _ = tx.SeaHouseBill.Create().
								SetOrganizationID(hq.ID).
								SetOrderID(ord.ID).
								SetMasterBillID(mbl.ID).
								SetHouseNo(o.hblNo).
								SetNormalizedHouseNo(o.hblNo).
								SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
								SetIssuerOrganizationID(hq.ID).
								SetStatus(seahousebillent.StatusCONFIRMED).
								SetShipperText(o.hblShipper).
								SetConsigneeText(o.hblConsignee).
								SetNotifyPartyText(o.hblNotify).
								SetPackageCount(o.pkgs).
								SetGrossWeightKg(o.weight).
								SetVolumeCbm(o.volume).
								SetGoodsDescriptionText(o.desc).
								SetFreightTerms(freightTerms).
								Save(ctx)
						}
					}
				}
			}
		}

		// 费用明细
		for _, f := range o.fees {
			feeKey := fmt.Sprintf("DEV-FEE-%s-%s-%s", o.orderNo, f.dir, f.code)
			orderFee, _ := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(ord.ID), orderfeeent.IdempotencyKeyEQ(feeKey)).First(ctx)
			if orderFee == nil {
				party := sc.partners[f.party]
				fSetting := sc.feeSettings[f.code]
				bUnit := sc.billingUnits[f.unit]
				dir := orderfeeent.Direction(f.dir)

				created, err := tx.OrderFee.Create().
					SetOrderID(ord.ID).
					SetIdempotencyKey(feeKey).
					SetDirection(dir).
					SetStatus(orderfeeent.StatusCONFIRMED).
					SetFeeSettingID(fSetting.ID).
					SetFeeCode(f.code).
					SetFeeName(f.name).
					SetSettlementPartyID(party.ID).
					SetBillingUnitID(bUnit.ID).
					SetBillingUnit(bUnit.Name).
					SetTaxRate(f.taxRate).
					SetQuantity(f.qty).
					SetUnitPrice(f.price).
					SetTotalAmount(f.total).
					SetNetAmount(f.total).
					SetTaxAmount("0.00000000").
					SetCurrency(f.cur).
					SetExchangeRate(f.rate).
					SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
					SetExchangeRateDate("2026-09-20").
					SetBaseCurrency(f.baseCur).
					SetBaseCurrencyAmount(f.baseAmt).
					SetExpenseDate("2026-09-20").
					Save(ctx)
				if err != nil {
					return fmt.Errorf("创建费用 %s: %w", feeKey, err)
				}
				orderFee = created
			}
			sc.orderFees[feeKey] = orderFee
		}
	}
	return nil
}

type feeItemDef struct {
	dir, code, name, party, unit, qty, price, total, cur, rate, baseCur, baseAmt, taxRate string
}

// 6. 财务账单、收付款流水与核销
func seedFinanceBillsAndCashflows(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq := sc.headquarters

	// 账单 1：已核销的应收账单 (SE26090002 人民币费用)
	bill1No := "AR26090001"
	party1 := sc.partners["CUST-ML-003"]
	ord2 := sc.orders["SE26090002"]

	b1, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(hq.ID), financebillent.BillNoEQ(bill1No)).First(ctx)
	if b1 == nil && ord2 != nil {
		account, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(party1.ID), partneraccountent.CurrencyEQ("CNY")).First(ctx)
		accName, accHolder, bankName, accNo := "对公结算户", party1.LegalName, "工商银行", "5719000122334455"
		var accID uuid.UUID
		if account != nil {
			accID = account.ID
			accName = account.Name
			accHolder = account.AccountHolder
			bankName = account.BankName
			accNo = account.AccountNo
		}

		created, err := tx.FinanceBill.Create().
			SetOrganizationID(hq.ID).
			SetBillNo(bill1No).
			SetIdempotencyKey("DEV-BILL-" + bill1No).
			SetDirection(financebillent.DirectionRECEIVABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(party1.ID).
			SetSettlementPartyName(party1.LegalName).
			SetSettlementAccountID(accID).
			SetSettlementAccountName(accName).
			SetSettlementAccountHolder(accHolder).
			SetSettlementBankName(bankName).
			SetSettlementBankAccount(accNo).
			SetSettlementAccountCurrency("CNY").
			SetCurrency("CNY").
			SetBaseCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-20").
			SetTotalAmount("2800.00000000").
			SetNetAmount("2800.00000000").
			SetTaxAmount("0.00000000").
			SetBaseCurrencyAmount("2800.00000000").
			SetFeeCount(3).
			SetBillDate("2026-09-20").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建账单 %s: %w", bill1No, err)
		}
		b1 = created

		// 挂载明细行
		keys := []string{
			"DEV-FEE-SE26090002-RECEIVABLE-THC",
			"DEV-FEE-SE26090002-RECEIVABLE-DOC",
			"DEV-FEE-SE26090002-RECEIVABLE-TRUCK",
		}
		for _, k := range keys {
			f := sc.orderFees[k]
			if f != nil {
				lineExists, _ := tx.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(b1.ID), financebilllineent.OrderFeeIDEQ(f.ID)).Exist(ctx)
				if !lineExists {
					_, _ = tx.FinanceBillLine.Create().
						SetBillID(b1.ID).
						SetOrderFeeID(f.ID).
						SetOrderID(ord2.ID).
						SetOrderNo(ord2.OrderNo).
						SetFeeCode(f.FeeCode).
						SetFeeName(f.FeeName).
						SetQuantity(f.Quantity).
						SetUnitPrice(f.UnitPrice).
						SetTotalAmount(f.TotalAmount).
						SetNetAmount(f.NetAmount).
						SetTaxAmount(f.TaxAmount).
						SetCurrency(f.Currency).
						SetExchangeRate(f.ExchangeRate).
						SetBaseCurrency(f.BaseCurrency).
						SetBaseCurrencyAmount(f.BaseCurrencyAmount).
						SetActive(true).
						Save(ctx)
				}
			}
		}
	}

	// 流水与核销 (对应 AR26090001)
	if b1 != nil {
		flowNo := "CF-AR-26090001"
		cf, _ := tx.FinanceCashflow.Query().Where(financecashflowent.OrganizationIDEQ(hq.ID), financecashflowent.FlowNoEQ(flowNo)).First(ctx)
		if cf == nil {
			created, err := tx.FinanceCashflow.Create().
				SetOrganizationID(hq.ID).
				SetFlowNo(flowNo).
				SetIdempotencyKey("DEV-CF-" + flowNo).
				SetDirection(financecashflowent.DirectionRECEIVABLE).
				SetStatus(financecashflowent.StatusCONFIRMED).
				SetSettlementPartyID(party1.ID).
				SetSettlementPartyName(party1.LegalName).
				SetCurrency("CNY").
				SetAmount("2800.00000000").
				SetExchangeRate("1.00000000").
				SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
				SetExchangeRateDate("2026-09-21").
				SetBaseCurrency("CNY").
				SetBaseAmount("2800.00000000").
				SetTransactionDate("2026-09-21").
				SetOurAccount("招商银行上海自贸区分行 121908888888888").
				SetPaymentMethod("银行电汇").
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建收款流水 %s: %w", flowNo, err)
			}
			cf = created
		}

		vfNo := "VF-AR-26090001"
		vf, _ := tx.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(hq.ID), financeverificationent.VerificationNoEQ(vfNo)).First(ctx)
		if vf == nil {
			created, err := tx.FinanceVerification.Create().
				SetOrganizationID(hq.ID).
				SetVerificationNo(vfNo).
				SetIdempotencyKey("DEV-VF-" + vfNo).
				SetStatus(financeverificationent.StatusACTIVE).
				SetDirection(financeverificationent.DirectionRECEIVABLE).
				SetSettlementPartyID(party1.ID).
				SetSettlementPartyName(party1.LegalName).
				SetCurrency("CNY").
				SetAmount("2800.00000000").
				SetBaseCurrency("CNY").
				SetBaseAmount("2800.00000000").
				SetBillBaseAmount("2800.00000000").
				SetCashflowBaseAmount("2800.00000000").
				SetExchangeGainLoss("0.00000000").
				SetVerificationDate("2026-09-21").
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建核销记录 %s: %w", vfNo, err)
			}
			vf = created

			// 核销分配行
			allocExists, _ := tx.FinanceVerificationAllocation.Query().Where(
				financeverificationallocationent.VerificationIDEQ(vf.ID),
				financeverificationallocationent.BillIDEQ(b1.ID),
			).Exist(ctx)
			if !allocExists {
				_, _ = tx.FinanceVerificationAllocation.Create().
					SetVerificationID(vf.ID).
					SetCashflowID(cf.ID).
					SetBillID(b1.ID).
					SetCashflowNo(cf.FlowNo).
					SetBillNo(b1.BillNo).
					SetAmount("2800.00000000").
					SetBillBaseAmount("2800.00000000").
					SetCashflowBaseAmount("2800.00000000").
					SetExchangeGainLoss("0.00000000").
					SetActive(true).
					Save(ctx)
			}
		}
	}

	// 账单 2：待付款的确认应收美金账单 (SE26090006)
	bill2No := "AR26090002"
	party2 := sc.partners["CUST-JS-002"]
	b2, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(hq.ID), financebillent.BillNoEQ(bill2No)).First(ctx)
	if b2 == nil {
		account, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(party2.ID), partneraccountent.CurrencyEQ("USD")).First(ctx)
		accName, accHolder, bankName, accNo := "外汇账户", party2.LegalName, "交通银行", "888800029999"
		var accID uuid.UUID
		if account != nil {
			accID = account.ID
			accName = account.Name
			accHolder = account.AccountHolder
			bankName = account.BankName
			accNo = account.AccountNo
		}
		_, _ = tx.FinanceBill.Create().
			SetOrganizationID(hq.ID).
			SetBillNo(bill2No).
			SetIdempotencyKey("DEV-BILL-" + bill2No).
			SetDirection(financebillent.DirectionRECEIVABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(party2.ID).
			SetSettlementPartyName(party2.LegalName).
			SetSettlementAccountID(accID).
			SetSettlementAccountName(accName).
			SetSettlementAccountHolder(accHolder).
			SetSettlementBankName(bankName).
			SetSettlementBankAccount(accNo).
			SetSettlementAccountCurrency("USD").
			SetCurrency("USD").
			SetBaseCurrency("CNY").
			SetExchangeRate("7.25000000").
			SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-20").
			SetTotalAmount("2100.00000000").
			SetNetAmount("2100.00000000").
			SetTaxAmount("0.00000000").
			SetBaseCurrencyAmount("15225.00000000").
			SetFeeCount(1).
			SetBillDate("2026-09-20").
			Save(ctx)
	}

	// 账单 3：草稿账单
	bill3No := "AR26090003"
	b3, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(hq.ID), financebillent.BillNoEQ(bill3No)).First(ctx)
	if b3 == nil {
		_, _ = tx.FinanceBill.Create().
			SetOrganizationID(hq.ID).
			SetBillNo(bill3No).
			SetIdempotencyKey("DEV-BILL-" + bill3No).
			SetDirection(financebillent.DirectionRECEIVABLE).
			SetStatus(financebillent.StatusDRAFT).
			SetSettlementPartyID(party1.ID).
			SetSettlementPartyName(party1.LegalName).
			SetSettlementAccountName("结算账户").
			SetSettlementAccountHolder(party1.LegalName).
			SetSettlementBankName("工商银行").
			SetSettlementBankAccount("3100661234567890").
			SetSettlementAccountCurrency("CNY").
			SetCurrency("CNY").
			SetBaseCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-21").
			SetTotalAmount("1500.00000000").
			SetNetAmount("1500.00000000").
			SetTaxAmount("0.00000000").
			SetBaseCurrencyAmount("1500.00000000").
			SetFeeCount(1).
			SetBillDate("2026-09-21").
			Save(ctx)
	}

	// 账单 4：应付船公司账单
	apBillNo := "AP26090001"
	suppCosco := sc.partners["SUPP-COSCO-01"]
	apB, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(hq.ID), financebillent.BillNoEQ(apBillNo)).First(ctx)
	if apB == nil && suppCosco != nil {
		_, _ = tx.FinanceBill.Create().
			SetOrganizationID(hq.ID).
			SetBillNo(apBillNo).
			SetIdempotencyKey("DEV-BILL-" + apBillNo).
			SetDirection(financebillent.DirectionPAYABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(suppCosco.ID).
			SetSettlementPartyName(suppCosco.LegalName).
			SetSettlementAccountName("中远海运对公户").
			SetSettlementAccountHolder(suppCosco.LegalName).
			SetSettlementBankName("中国银行上海分行").
			SetSettlementBankAccount("1001234509008899").
			SetSettlementAccountCurrency("USD").
			SetCurrency("USD").
			SetBaseCurrency("CNY").
			SetExchangeRate("7.25000000").
			SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-20").
			SetTotalAmount("1950.00000000").
			SetNetAmount("1950.00000000").
			SetTaxAmount("0.00000000").
			SetBaseCurrencyAmount("14137.50000000").
			SetFeeCount(1).
			SetBillDate("2026-09-20").
			Save(ctx)
	}

	return nil
}

// 7. 提成方案与分配
func seedCommissionRules(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	hq := sc.headquarters

	rule1Name := "海运出口业务员毛利提成方案 (标准10%)"
	r1, _ := tx.FinanceCommissionRule.Query().Where(
		financecommissionruleent.OrganizationIDEQ(hq.ID),
		financecommissionruleent.NameEQ(rule1Name),
	).First(ctx)
	if r1 == nil {
		created, err := tx.FinanceCommissionRule.Create().
			SetOrganizationID(hq.ID).
			SetName(rule1Name).
			SetPersonnelRole(financecommissionruleent.PersonnelRoleSALES).
			SetCalculationBasis(financecommissionruleent.CalculationBasisREALIZED_PROFIT).
			SetRatePercent("10.0000").
			SetEffectiveFrom("2026-01-01").
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建提成方案 %s: %w", rule1Name, err)
		}
		r1 = created
	}

	rule2Name := "海运出口操作员单票提成方案 (3%)"
	r2, _ := tx.FinanceCommissionRule.Query().Where(
		financecommissionruleent.OrganizationIDEQ(hq.ID),
		financecommissionruleent.NameEQ(rule2Name),
	).First(ctx)
	if r2 == nil {
		created, err := tx.FinanceCommissionRule.Create().
			SetOrganizationID(hq.ID).
			SetName(rule2Name).
			SetPersonnelRole(financecommissionruleent.PersonnelRoleOPERATOR).
			SetCalculationBasis(financecommissionruleent.CalculationBasisREALIZED_PROFIT).
			SetRatePercent("3.0000").
			SetEffectiveFrom("2026-01-01").
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建提成方案 %s: %w", rule2Name, err)
		}
		r2 = created
	}

	// 分配给销售员工张强、王丽
	salesList := []string{"zhangqiang", "wangli"}
	for _, username := range salesList {
		u := sc.users[username]
		if u == nil {
			continue
		}
		aExists, _ := tx.FinanceCommissionRuleAssignment.Query().Where(
			financecommissionruleassignmentent.OrganizationIDEQ(hq.ID),
			financecommissionruleassignmentent.RuleIDEQ(r1.ID),
			financecommissionruleassignmentent.EmployeeIDEQ(u.ID),
		).Exist(ctx)
		if !aExists {
			_, _ = tx.FinanceCommissionRuleAssignment.Create().
				SetOrganizationID(hq.ID).
				SetRuleID(r1.ID).
				SetEmployeeID(u.ID).
				SetEffectiveFrom("2026-01-01").
				SetCreatedBy(sc.adminUser.ID).
				Save(ctx)
		}
	}

	// 分配给操作李明
	opUser := sc.users["liming"]
	if opUser != nil && r2 != nil {
		aExists, _ := tx.FinanceCommissionRuleAssignment.Query().Where(
			financecommissionruleassignmentent.OrganizationIDEQ(hq.ID),
			financecommissionruleassignmentent.RuleIDEQ(r2.ID),
			financecommissionruleassignmentent.EmployeeIDEQ(opUser.ID),
		).Exist(ctx)
		if !aExists {
			_, _ = tx.FinanceCommissionRuleAssignment.Create().
				SetOrganizationID(hq.ID).
				SetRuleID(r2.ID).
				SetEmployeeID(opUser.ID).
				SetEffectiveFrom("2026-01-01").
				SetCreatedBy(sc.adminUser.ID).
				Save(ctx)
		}
	}

	return nil
}
