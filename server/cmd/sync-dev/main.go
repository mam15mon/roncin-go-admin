package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
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
	financebillbatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillbatch"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financecommissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	financecommissionapplicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	financecommissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	financecommissionruleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	masterdataitem "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seadocumentvoideventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentvoidevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	taxableserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/taxableservice"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

type seedContext struct {
	tx              *ent.Tx
	systemWorkspace *ent.Organization
	company         *ent.Organization
	adminUser       *ent.User
	users           map[string]*ent.User
	ports           map[string]*ent.Port
	shippingLines   map[string]*ent.ShippingLine
	billingUnits    map[string]*ent.BillingUnit
	taxableSvcs     map[string]*ent.TaxableService
	chargeCats      map[string]*ent.MasterDataItem
	containerSpecs  map[string]*ent.MasterDataItem
	feeSettings     map[string]*ent.FeeSetting
	partners        map[string]*ent.Partner
	orders          map[string]*ent.Order
	orderFees       map[string]*ent.OrderFee
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

	sc, err := runSeed(ctx, client)
	if err != nil {
		logger.Error("开发测试数据同步失败", "error", err)
		os.Exit(1)
	}
	printSeedSummary(sc)
}

func runSeed(ctx context.Context, client *ent.Client) (*seedContext, error) {
	var sc *seedContext
	err := data.WithClientTx(ctx, client, func(tx *ent.Tx) error {
		sc = &seedContext{
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
			return fmt.Errorf("组织与员工同步失败: %w", err)
		}
		if err := seedReferenceData(ctx, sc); err != nil {
			return fmt.Errorf("基础参考数据（港口/船司）同步失败: %w", err)
		}
		if err := seedFinanceMasterData(ctx, sc); err != nil {
			return fmt.Errorf("财务主数据（计费单位/税目/费用/汇率）同步失败: %w", err)
		}
		if err := seedPartners(ctx, sc); err != nil {
			return fmt.Errorf("往来单位同步失败: %w", err)
		}
		if err := seedOrdersAndFees(ctx, sc); err != nil {
			return fmt.Errorf("业务订单与费用明细同步失败: %w", err)
		}
		if err := seedFinanceBillsAndCashflows(ctx, sc); err != nil {
			return fmt.Errorf("财务账单、流水与核销同步失败: %w", err)
		}
		if err := seedOrderLockGovernance(ctx, sc); err != nil {
			return fmt.Errorf("订单锁治理测试数据同步失败: %w", err)
		}
		if err := seedCommissionRules(ctx, sc); err != nil {
			return fmt.Errorf("提成方案与规则同步失败: %w", err)
		}
		if err := seedCommissionLedger(ctx, sc); err != nil {
			return fmt.Errorf("提成台账与月度申请同步失败: %w", err)
		}
		if err := seedSeaDocumentOperations(ctx, sc); err != nil {
			return fmt.Errorf("海运单证操作（拆票/改配/共享箱/改单作废）同步失败: %w", err)
		}
		if err := seedOrderOperations(ctx, sc); err != nil {
			return fmt.Errorf("订单运营数据（里程碑/POD/附件/异常）同步失败: %w", err)
		}

		return nil
	})
	return sc, err
}

func printSeedSummary(sc *seedContext) {
	fmt.Println("==================================================")
	fmt.Println("🎉 开发测试数据同步完成 (sync:dev successful)！")
	fmt.Printf("✔ 组织体系: 系统管理 [%s] + 独立公司已完备；经营数据归属 [%s]，员工具备公司身份\n", sc.systemWorkspace.Name, sc.company.Name)
	fmt.Printf("✔ 测试人员: %d 名业务员工 (张强/王丽/李明/陈华/赵芳/刘敏)\n", len(sc.users))
	fmt.Printf("✔ 基础参考: %d 个核心海港 (CNSHA/CNNBO/USLAX...) + %d 家主流船公司\n", len(sc.ports), len(sc.shippingLines))
	fmt.Printf("✔ 财务字典: %d 个计费单位 + %d 个费用科目 + 开发基准汇率已生效\n", len(sc.billingUnits), len(sc.feeSettings))
	fmt.Printf("✔ 往来单位: %d 家客商档案 (包含进出口商、电商、船代、车队、报关行及对公账户/联系人)\n", len(sc.partners))
	fmt.Printf("✔ 海运订单: %d 票全生命周期订单 (草稿/订舱/配舱/拖车/截单/报关安排/放单/终止/结案，含锁定票、拼舱与主分单)\n", len(sc.orders))
	fmt.Printf("✔ 费用账单: 应收/应付费用明细已录入，含已核销、部分核销+汇兑损益、美元确认、草稿、应付与批量建账批次\n")
	fmt.Printf("✔ 订单锁治理: 核销自动锁+钉钉解锁审批(审批中)、人工锁定含单证版本快照、锁后补录(待审批+已审批)\n")
	fmt.Printf("✔ 提成链路: 经营归属+核销计提台账(销售10%%/操作3%%)+月度申请(待审批/已审批)\n")
	fmt.Printf("✔ 单证操作: 拆票+整体改配(含ENDED关系)、共享箱守恒、HBL改单/作废重出、直单转主分单\n")
	fmt.Printf("✔ 运营数据: 异常标记(未解决/已解决)、里程碑、放单POD(待签/已签)、附件登记\n")
	fmt.Println("==================================================")
}

// 1. 组织架构与人员
func seedOrganizationAndStaff(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	systemWorkspace, err := tx.Organization.Query().Where(organizationent.KindEQ(organizationent.KindSystem)).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if systemWorkspace == nil {
		created, createErr := tx.Organization.Create().
			SetCode("SYSTEM").
			SetName("系统管理").
			SetKind(organizationent.KindSystem).
			SetBaseCurrency("CNY").
			SetEnabledCurrencies([]string{"CNY", "USD", "EUR", "HKD"}).
			SetEnabled(true).
			Save(ctx)
		if createErr != nil {
			return fmt.Errorf("创建系统管理工作台: %w", createErr)
		}
		systemWorkspace = created
	}
	sc.systemWorkspace = systemWorkspace

	// 补充独立公司；公共管理工作台不作为公司父节点。
	if _, err := data.CreateDefaultBranchCompanies(ctx, tx, systemWorkspace.ID); err != nil {
		return err
	}

	// 开发数据明确归属上海公司，不向其他公司隐式写入。
	company, companyErr := tx.Organization.Query().Where(
		organizationent.KindEQ(organizationent.KindCompany),
		organizationent.EnabledEQ(true),
		organizationent.CodeEQ("SH"),
	).First(ctx)
	if companyErr != nil {
		return fmt.Errorf("缺少启用的上海公司作为开发数据归属组织: %w", companyErr)
	}
	sc.company = company

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

		// 重复执行时也补齐本公司角色，不授予系统管理成员资格。
		companyAdminRole, err := tx.Role.Query().Where(roleent.OrganizationIDEQ(company.ID), roleent.CodeEQ("administrator")).Only(ctx)
		if err != nil {
			return fmt.Errorf("查询公司管理员角色: %w", err)
		}
		cm, err := tx.Membership.Query().Where(membershipent.UserID(u.ID), membershipent.OrganizationIDEQ(company.ID)).Only(ctx)
		if ent.IsNotFound(err) {
			cm, err = tx.Membership.Create().SetUserID(u.ID).SetOrganizationID(company.ID).SetPrimary(true).SetEnabled(true).Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("确保公司测试成员: %w", err)
		}
		assigned, err := tx.RoleAssignment.Query().Where(roleassignmentent.MembershipIDEQ(cm.ID), roleassignmentent.RoleIDEQ(companyAdminRole.ID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !assigned {
			if _, err := tx.RoleAssignment.Create().SetMembershipID(cm.ID).SetRoleID(companyAdminRole.ID).Save(ctx); err != nil {
				return err
			}
		}
	}
	// 经营操作的创建人与审批人使用真实公司成员，不借用系统管理员身份。
	sc.adminUser = sc.users["zhangqiang"]
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
		found, _ := tx.Port.Query().Where(portent.UnLocodeEQ(p.code)).First(ctx)
		if found == nil {
			created, err := tx.Port.Create().
				SetUnLocode(p.code).
				SetNameZh(p.nameZh).
				SetNameEn(p.nameEn).
				SetCountryCode(p.country).
				SetTransportModes([]string{"SEA"}).
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
	co := sc.company

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
		found, _ := tx.TaxableService.Query().Where(taxableserviceent.OrganizationIDEQ(co.ID), taxableserviceent.NameEQ(s.name)).First(ctx)
		if found == nil {
			created, err := tx.TaxableService.Create().
				SetOrganizationID(co.ID).
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
		found, _ := tx.FeeSetting.Query().Where(feesettingent.OrganizationIDEQ(co.ID), feesettingent.FeeCodeEQ(f.code)).First(ctx)
		if found == nil {
			created, err := tx.FeeSetting.Create().
				SetOrganizationID(co.ID).
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
			exchangeratesettingent.OrganizationIDEQ(co.ID),
			exchangeratesettingent.FromCurrencyEQ(r.from),
			exchangeratesettingent.ToCurrencyEQ(r.to),
		).First(ctx)
		if rFound == nil {
			_, err := tx.ExchangeRateSetting.Create().
				SetOrganizationID(co.ID).
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
	co := sc.company

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
		p, _ := tx.Partner.Query().Where(partnerent.OrganizationIDEQ(co.ID), partnerent.CodeEQ(s.code)).First(ctx)
		if p == nil {
			created, err := tx.Partner.Create().
				SetOrganizationID(co.ID).
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
					SetOrganizationID(co.ID).
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
	co := sc.company

	type orderSeedDef struct {
		orderNo       string
		custCode      string
		refNo         string
		lineScac      string
		vessel        string
		voyage        string
		mblNo         string
		pol, pod      string
		flowStatus    orderent.FlowStatus
		term          orderent.TradeTerm
		payTerm       orderent.PaymentTerm
		shipType      orderent.ShipmentType
		desc          string
		pkgs          int
		weight        float64
		volume        float64
		isLocked      bool
		salesRep      string
		opRep         string
		docRep        string
		container     string
		sealNo        string
		containerSpec string
		// 终止与结案事实；terminationType 非空表示注入一票已终止订单，
		// closureReason 非空表示注入一票已人工结案订单。
		terminationType   orderent.TerminationType
		terminationReason string
		closureReason     string
		docStructure      seamasterbillorderlinkent.DocumentStructure
		hblNo             string
		hblShipper        string
		hblConsignee      string
		hblNotify         string
		fees              []feeItemDef
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
		{
			orderNo: "SE26090010", custCode: "CUST-HT-004", refNo: "PO-HT-260910",
			lineScac: "EGLV", vessel: "EVER GENIUS", voyage: "0712W", mblNo: "EGLV33445566",
			pol: "CNQDG", pod: "USLAX", flowStatus: orderent.FlowStatusTRUCKING_ARRANGED,
			term: orderent.TradeTermCFR, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "轮胎硫化机及成型配件 (20GP整箱)", pkgs: 120, weight: 9800.0, volume: 28.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "EGLV2468013", sealNo: "EGL531122", containerSpec: "20GP",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "TRUCK", name: "集装箱拖车费", party: "CUST-HT-004", unit: "CONT", qty: "1.0000", price: "980.0000", total: "980.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "980.00000000", taxRate: "9.00"},
				{dir: "PAYABLE", code: "TRUCK", name: "集装箱拖车费(付远集车队)", party: "SUPP-YJTC-04", unit: "CONT", qty: "1.0000", price: "820.0000", total: "820.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "820.00000000", taxRate: "9.00"},
			},
		},
		{
			orderNo: "SE26090011", custCode: "CUST-TG-005", refNo: "PO-TG-260911",
			lineScac: "ONEY", vessel: "ONE HARBOUR", voyage: "012E", mblNo: "ONEY88776655",
			pol: "CNNBO", pod: "DEHAM", flowStatus: orderent.FlowStatusDOCUMENT_CUTOFF,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "户外露营帐篷及庭院配件", pkgs: 640, weight: 8800.0, volume: 68.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "ONEY4433221", sealNo: "ONE998877",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "DOC", name: "文件费", party: "CUST-TG-005", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "TLX", name: "电放费", party: "CUST-TG-005", unit: "BL", qty: "1.0000", price: "350.0000", total: "350.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "350.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090012", custCode: "CUST-ML-003", refNo: "PO-ML-260912",
			lineScac: "MSCU", vessel: "MSC AMBER", voyage: "2638E", mblNo: "MSCU99886644",
			pol: "CNSZX", pod: "SGSIN", flowStatus: orderent.FlowStatusCUSTOMS_DECLARATION_ARRANGED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "庭院遮阳伞及五金配件", pkgs: 700, weight: 11200.0, volume: 88.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "MSCU6677889", sealNo: "MSC334455",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "CUSTOMS", name: "代理报关费", party: "CUST-ML-003", unit: "BL", qty: "1.0000", price: "350.0000", total: "350.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "350.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "VGM", name: "重量验证费", party: "CUST-ML-003", unit: "CONT", qty: "1.0000", price: "60.0000", total: "60.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "60.00000000", taxRate: "6.00"},
				{dir: "PAYABLE", code: "CUSTOMS", name: "代理报关费(付海通)", party: "SUPP-HTBG-03", unit: "BL", qty: "1.0000", price: "200.0000", total: "200.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "200.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090013", custCode: "CUST-JS-002", refNo: "PO-JS-260913",
			lineScac: "COSU", vessel: "COSCO HARMONY", voyage: "051W", mblNo: "",
			pol: "CNSHA", pod: "USLGB", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "智能家居网关及传感器 (客户临时取消出运)", pkgs: 260, weight: 1900.0, volume: 15.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "COSU7788123", sealNo: "COS901177",
			terminationType:   orderent.TerminationTypeCUSTOMER_CANCEL,
			terminationReason: "客户因目的港收货人变更临时取消出运，已安排退关退载",
			docStructure:      seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees:              []feeItemDef{},
		},
		{
			orderNo: "SE26090014", custCode: "CUST-HY-001", refNo: "PO-HY-260914",
			lineScac: "MSCU", vessel: "MSC MIRJA", voyage: "2642E", mblNo: "MSCU10102020",
			pol: "CNNBO", pod: "NLRTM", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "医用防护服及无纺布制品 (已放单结案)", pkgs: 400, weight: 4200.0, volume: 38.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "MSCU2233445", sealNo: "MSC667788",
			closureReason: "已放单且费用收付结清，人工结案归档",
			docStructure:  seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费", party: "CUST-HY-001", unit: "CONT", qty: "1.0000", price: "2050.0000", total: "2050.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "14862.50000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "文件费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
				{dir: "PAYABLE", code: "OF", name: "海运费(付地中海代理)", party: "SUPP-MSC-02", unit: "CONT", qty: "1.0000", price: "1800.0000", total: "1800.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "13050.00000000", taxRate: "0.00"},
			},
		},
		{
			orderNo: "SE26090015", custCode: "CUST-ML-003", refNo: "PO-ML-260915",
			lineScac: "MSCU", vessel: "MSC VELA", voyage: "2650E", mblNo: "MSCU77889900",
			pol: "CNSHA", pod: "SGSIN", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeLCL,
			desc: "松木园艺工具把手 (拆票后剩余部分)", pkgs: 30, weight: 390.0, volume: 3.48, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090015",
			hblShipper:   "浙江美林工艺家具有限公司\nZHEJIANG MEILIN CRAFT FURNITURE CO., LTD.",
			hblConsignee: "GREEN GARDEN TOOLS PTE LTD\n12 JURONG EAST STREET, SINGAPORE",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运拼箱运费", party: "CUST-ML-003", unit: "CBM", qty: "3.4800", price: "45.0000", total: "156.60000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "1135.35000000", taxRate: "0.00"},
			},
		},
		{
			orderNo: "SE26090016", custCode: "CUST-JS-002", refNo: "PO-JS-260916",
			lineScac: "MAEU", vessel: "MAERSK KOTKA", voyage: "2610W", mblNo: "MAEU31112233",
			pol: "CNSHA", pod: "SGSIN", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeLCL,
			desc: "折叠野餐桌及铝合金支架 (拆票拆出部分)", pkgs: 20, weight: 260.0, volume: 2.32, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090016",
			hblShipper:   "深圳市极速跨境智能实业有限公司\nSHENZHEN SPEED CROSS-BORDER INDUSTRIAL CO.",
			hblConsignee: "SINGAPORE OUTDOOR LIVING PTE LTD\n8 MARINA VIEW, SINGAPORE",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "DOC", name: "分单文件费", party: "CUST-JS-002", unit: "BL", qty: "1.0000", price: "350.0000", total: "350.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "350.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090017", custCode: "CUST-TG-005", refNo: "PO-TG-260917",
			lineScac: "ONEY", vessel: "ONE AKASAKA", voyage: "021E", mblNo: "ONEY55667788",
			pol: "CNNBO", pod: "USLGB", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermFOB, payTerm: orderent.PaymentTermCOLLECT, shipType: orderent.ShipmentTypeFCL,
			desc: "玻璃相框及镜面装饰品 (HBL 作废重出)", pkgs: 380, weight: 4100.0, volume: 54.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "ONEY6655441", sealNo: "ONE334455",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090017R",
			hblShipper:   "TransGlobal Trading (HK) Co., Limited\nUNIT 8, 12/F, KWAI CHUNG, HONG KONG",
			hblConsignee: "PACIFIC FRAME & MIRROR INC.\n4500 E PICO BLVD, LOS ANGELES, CA",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "DOC", name: "分单文件费", party: "CUST-TG-005", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090018", custCode: "CUST-HY-001", refNo: "PO-HY-260918",
			lineScac: "EGLV", vessel: "EVER FORTHRIGHT", voyage: "0715E", mblNo: "EGLV90807060",
			pol: "CNQDG", pod: "NLRTM", flowStatus: orderent.FlowStatusBOOKED,
			term: orderent.TradeTermCIF, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "出口办公家具 (直单转主分单结构)", pkgs: 320, weight: 4600.0, volume: 52.0, isLocked: false,
			salesRep: "zhangqiang", opRep: "liming", docRep: "chenhua",
			container: "EGLV1357246", sealNo: "EGL990011",
			docStructure: seamasterbillorderlinkent.DocumentStructureHOUSE,
			hblNo:        "RC-HBL26090018",
			hblShipper:   "上海宏远国际贸易进出口有限公司\nSHANGHAI HONGYUAN TRADING CO., LTD.",
			hblConsignee: "EUROPE OFFICE FURNISHING B.V.\nHAVENWEG 12, ROTTERDAM, NETHERLANDS",
			hblNotify:    "SAME AS CONSIGNEE",
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费", party: "CUST-HY-001", unit: "CONT", qty: "1.0000", price: "2350.0000", total: "2350.00000000", cur: "USD", rate: "7.25000000", baseCur: "CNY", baseAmt: "17037.50000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "分单文件费", party: "CUST-HY-001", unit: "BL", qty: "1.0000", price: "500.0000", total: "500.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "500.00000000", taxRate: "6.00"},
			},
		},
		{
			orderNo: "SE26090019", custCode: "CUST-HT-004", refNo: "PO-HT-260919",
			lineScac: "MSCU", vessel: "MSC AMBITION", voyage: "2655E", mblNo: "MSCU40506070",
			pol: "CNNBO", pod: "DEHAM", flowStatus: orderent.FlowStatusDOCUMENT_RELEASED,
			term: orderent.TradeTermDAP, payTerm: orderent.PaymentTermPREPAID, shipType: orderent.ShipmentTypeFCL,
			desc: "精密医用影像设备备件 (多币种结算)", pkgs: 85, weight: 2900.0, volume: 24.0, isLocked: false,
			salesRep: "wangli", opRep: "liming", docRep: "chenhua",
			container: "MSCU8090506", sealNo: "MSC102030", containerSpec: "40GP",
			docStructure: seamasterbillorderlinkent.DocumentStructureDIRECT,
			fees: []feeItemDef{
				{dir: "RECEIVABLE", code: "OF", name: "海运费(欧元)", party: "CUST-HT-004", unit: "CONT", qty: "1.0000", price: "1650.0000", total: "1650.00000000", cur: "EUR", rate: "7.85000000", baseCur: "CNY", baseAmt: "12952.50000000", taxRate: "0.00"},
				{dir: "RECEIVABLE", code: "DOC", name: "文件费(港币)", party: "CUST-HT-004", unit: "BL", qty: "1.0000", price: "480.0000", total: "480.00000000", cur: "HKD", rate: "0.92500000", baseCur: "CNY", baseAmt: "444.00000000", taxRate: "6.00"},
				{dir: "RECEIVABLE", code: "THC", name: "码头操作费", party: "CUST-HT-004", unit: "CONT", qty: "1.0000", price: "1150.0000", total: "1150.00000000", cur: "CNY", rate: "1.00000000", baseCur: "CNY", baseAmt: "1150.00000000", taxRate: "6.00"},
			},
		},
	}

	for _, o := range orders {
		cust := sc.partners[o.custCode]
		line := sc.shippingLines[o.lineScac]
		pol := sc.ports[o.pol]
		pod := sc.ports[o.pod]

		ord, _ := tx.Order.Query().Where(orderent.OrganizationIDEQ(co.ID), orderent.OrderNoEQ(o.orderNo)).First(ctx)
		if ord == nil {
			idempotencyKey := "DEV-IDEM-" + o.orderNo
			vesselVoyage := o.vessel + " / " + o.voyage
			builder := tx.Order.Create().
				SetOrganizationID(co.ID).
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

			if o.terminationType != "" && sc.adminUser != nil {
				builder.SetTerminationStatus(orderent.TerminationStatusTERMINATED).
					SetTerminationType(o.terminationType).
					SetTerminationReason(o.terminationReason).
					SetTerminatedAt(time.Now().UTC()).
					SetTerminatedBy(sc.adminUser.ID)
			}
			if o.closureReason != "" && sc.adminUser != nil {
				builder.SetClosureStatus(orderent.ClosureStatusCLOSED).
					SetClosureReason(o.closureReason).
					SetClosedAt(time.Now().UTC()).
					SetClosedBy(sc.adminUser.ID)
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
					SetOrganizationID(co.ID).
					SetRole(a.role).
					Save(ctx)
			}
		}

		// 货物明细
		cargoExists, _ := tx.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(ord.ID)).Exist(ctx)
		if !cargoExists {
			_, _ = tx.OrderCargoItem.Create().
				SetOrganizationID(co.ID).
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
				if o.containerSpec != "" {
					spec = sc.containerSpecs[o.containerSpec]
				}
				if spec == nil {
					spec = sc.containerSpecs["20GP"]
				}
				_, _ = tx.OrderContainer.Create().
					SetOrganizationID(co.ID).
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
			mbl, _ := tx.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(co.ID), seamasterbillent.MasterNoEQ(o.mblNo)).First(ctx)
			if mbl == nil {
				createdMbl, err := tx.SeaMasterBill.Create().
					SetOrganizationID(co.ID).
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
					seatransportexecutionent.OrganizationIDEQ(co.ID),
					seatransportexecutionent.ShippingLineIDEQ(line.ID),
					seatransportexecutionent.VesselNameEQ(o.vessel),
					seatransportexecutionent.VoyageNoEQ(o.voyage),
				).First(ctx)
				if exec == nil {
					createdExec, err := tx.SeaTransportExecution.Create().
						SetOrganizationID(co.ID).
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
						seamasterbillorderlinkent.OrganizationIDEQ(co.ID),
						seamasterbillorderlinkent.OrderIDEQ(ord.ID),
					).First(ctx)
					docStruct := seamasterbillorderlinkent.DocumentStructureDIRECT
					if o.docStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
						docStruct = seamasterbillorderlinkent.DocumentStructureHOUSE
					}
					if link == nil {
						_, _ = tx.SeaMasterBillOrderLink.Create().
							SetOrganizationID(co.ID).
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
							seahousebillent.OrganizationIDEQ(co.ID),
							seahousebillent.OrderIDEQ(ord.ID),
						).First(ctx)
						if hb == nil {
							freightTerms := "FREIGHT PREPAID"
							if o.payTerm == orderent.PaymentTermCOLLECT {
								freightTerms = "FREIGHT COLLECT"
							}
							_, _ = tx.SeaHouseBill.Create().
								SetOrganizationID(co.ID).
								SetOrderID(ord.ID).
								SetMasterBillID(mbl.ID).
								SetHouseNo(o.hblNo).
								SetNormalizedHouseNo(o.hblNo).
								SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
								SetIssuerOrganizationID(co.ID).
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

		// 人工锁定票：按真实锁定链路创建单证版本快照、订单锁字段与锁定记录
		if o.isLocked && sc.adminUser != nil {
			if _, err := ensureOrderLockedWithRecord(ctx, sc, ord, lockSeedSpec{
				lockSource:     orderlockrecordent.LockSourceMANUAL,
				manualLockedBy: sc.adminUser.ID,
			}); err != nil {
				return fmt.Errorf("锁定订单 %s: %w", o.orderNo, err)
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
	co := sc.company

	// 账单 1：已核销的应收账单 (SE26090002 人民币费用)
	bill1No := "AR26090001"
	party1 := sc.partners["CUST-ML-003"]
	ord2 := sc.orders["SE26090002"]

	b1, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(bill1No)).First(ctx)
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
			SetOrganizationID(co.ID).
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
		cf, _ := tx.FinanceCashflow.Query().Where(financecashflowent.OrganizationIDEQ(co.ID), financecashflowent.FlowNoEQ(flowNo)).First(ctx)
		if cf == nil {
			created, err := tx.FinanceCashflow.Create().
				SetOrganizationID(co.ID).
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
		vf, _ := tx.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(co.ID), financeverificationent.VerificationNoEQ(vfNo)).First(ctx)
		if vf == nil {
			created, err := tx.FinanceVerification.Create().
				SetOrganizationID(co.ID).
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

	// 账单创建批次：AR26090002 通过批量建账产生
	batchNo := "BT26090001"
	batch, _ := tx.FinanceBillBatch.Query().Where(financebillbatchent.OrganizationIDEQ(co.ID), financebillbatchent.BatchNoEQ(batchNo)).First(ctx)
	if batch == nil {
		createdBatch, err := tx.FinanceBillBatch.Create().
			SetOrganizationID(co.ID).
			SetBatchNo(batchNo).
			SetIdempotencyKey("DEV-BATCH-" + batchNo).
			SetRequestHash("DEVSEEDBT26090001" + strings.Repeat("0", 47)).
			SetSplitByOrder(true).
			SetGroupingMode(financebillbatchent.GroupingModeNORMAL).
			SetFeeCount(1).
			SetBillCount(1).
			SetTotalBaseAmount("15225.00000000").
			SetBaseCurrency("CNY").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建账单批次 %s: %w", batchNo, err)
		}
		batch = createdBatch
	}

	// 账单 2：待付款的确认应收美金账单 (SE26090006)
	bill2No := "AR26090002"
	party2 := sc.partners["CUST-JS-002"]
	b2, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(bill2No)).First(ctx)
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
		created2, err := tx.FinanceBill.Create().
			SetOrganizationID(co.ID).
			SetBillNo(bill2No).
			SetIdempotencyKey("DEV-BILL-" + bill2No).
			SetBatchID(batch.ID).
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
		if err != nil {
			return fmt.Errorf("创建账单 %s: %w", bill2No, err)
		}
		b2 = created2
	}

	// 部分核销与汇兑损益：AR26090002 首期收款 1000 美元，收汇汇率与账单汇率
	// 不同产生 50 元汇兑损益，账单剩余 1100 美元未核销。
	if b2 != nil {
		flowNo2 := "CF-AR-26090002"
		cf2, _ := tx.FinanceCashflow.Query().Where(financecashflowent.OrganizationIDEQ(co.ID), financecashflowent.FlowNoEQ(flowNo2)).First(ctx)
		if cf2 == nil {
			createdCf2, err := tx.FinanceCashflow.Create().
				SetOrganizationID(co.ID).
				SetFlowNo(flowNo2).
				SetIdempotencyKey("DEV-CF-" + flowNo2).
				SetDirection(financecashflowent.DirectionRECEIVABLE).
				SetStatus(financecashflowent.StatusCONFIRMED).
				SetSettlementPartyID(party2.ID).
				SetSettlementPartyName(party2.LegalName).
				SetCurrency("USD").
				SetAmount("1000.00000000").
				SetExchangeRate("7.20000000").
				SetExchangeRateSource(financecashflowent.ExchangeRateSourceMANUAL).
				SetExchangeRateDate("2026-09-21").
				SetBaseCurrency("CNY").
				SetBaseAmount("7200.00000000").
				SetTransactionDate("2026-09-21").
				SetOurAccount("招商银行上海自贸区分行 121908888888888").
				SetPaymentMethod("银行电汇").
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建收款流水 %s: %w", flowNo2, err)
			}
			cf2 = createdCf2
		}

		vfNo2 := "VF-AR-26090002"
		vf2, _ := tx.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(co.ID), financeverificationent.VerificationNoEQ(vfNo2)).First(ctx)
		if vf2 == nil {
			createdVf2, err := tx.FinanceVerification.Create().
				SetOrganizationID(co.ID).
				SetVerificationNo(vfNo2).
				SetIdempotencyKey("DEV-VF-" + vfNo2).
				SetStatus(financeverificationent.StatusACTIVE).
				SetDirection(financeverificationent.DirectionRECEIVABLE).
				SetSettlementPartyID(party2.ID).
				SetSettlementPartyName(party2.LegalName).
				SetCurrency("USD").
				SetAmount("1000.00000000").
				SetBaseCurrency("CNY").
				SetBaseAmount("7200.00000000").
				SetBillBaseAmount("7250.00000000").
				SetCashflowBaseAmount("7200.00000000").
				SetExchangeGainLoss("50.00000000").
				SetVerificationDate("2026-09-21").
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建核销记录 %s: %w", vfNo2, err)
			}

			alloc2Exists, _ := tx.FinanceVerificationAllocation.Query().Where(
				financeverificationallocationent.VerificationIDEQ(createdVf2.ID),
				financeverificationallocationent.BillIDEQ(b2.ID),
			).Exist(ctx)
			if !alloc2Exists {
				_, _ = tx.FinanceVerificationAllocation.Create().
					SetVerificationID(createdVf2.ID).
					SetCashflowID(cf2.ID).
					SetBillID(b2.ID).
					SetCashflowNo(cf2.FlowNo).
					SetBillNo(b2.BillNo).
					SetAmount("1000.00000000").
					SetBillBaseAmount("7250.00000000").
					SetCashflowBaseAmount("7200.00000000").
					SetExchangeGainLoss("50.00000000").
					SetActive(true).
					Save(ctx)
			}
		}
	}

	// 账单 3：草稿账单
	bill3No := "AR26090003"
	b3, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(bill3No)).First(ctx)
	if b3 == nil {
		account3, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(party1.ID), partneraccountent.CurrencyEQ("CNY")).First(ctx)
		acc3Name, acc3Holder, bank3Name, acc3No := "对公结算户", party1.LegalName, "工商银行", "5719000122334455"
		var acc3ID uuid.UUID
		if account3 != nil {
			acc3ID = account3.ID
			acc3Name = account3.Name
			acc3Holder = account3.AccountHolder
			bank3Name = account3.BankName
			acc3No = account3.AccountNo
		}
		_, err := tx.FinanceBill.Create().
			SetOrganizationID(co.ID).
			SetBillNo(bill3No).
			SetIdempotencyKey("DEV-BILL-" + bill3No).
			SetDirection(financebillent.DirectionRECEIVABLE).
			SetStatus(financebillent.StatusDRAFT).
			SetSettlementPartyID(party1.ID).
			SetSettlementPartyName(party1.LegalName).
			SetSettlementAccountID(acc3ID).
			SetSettlementAccountName(acc3Name).
			SetSettlementAccountHolder(acc3Holder).
			SetSettlementBankName(bank3Name).
			SetSettlementBankAccount(acc3No).
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
		if err != nil {
			return fmt.Errorf("创建账单 %s: %w", bill3No, err)
		}
	}

	// 账单 4：应付船公司账单
	apBillNo := "AP26090001"
	suppCosco := sc.partners["SUPP-COSCO-01"]
	apB, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(apBillNo)).First(ctx)
	if apB == nil && suppCosco != nil {
		account4, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(suppCosco.ID), partneraccountent.CurrencyEQ("USD")).First(ctx)
		acc4Name, acc4Holder, bank4Name, acc4No := "中远海运对公户", suppCosco.LegalName, "中国银行上海分行", "1001234509008899"
		var acc4ID uuid.UUID
		if account4 != nil {
			acc4ID = account4.ID
			acc4Name = account4.Name
			acc4Holder = account4.AccountHolder
			bank4Name = account4.BankName
			acc4No = account4.AccountNo
		}
		_, err := tx.FinanceBill.Create().
			SetOrganizationID(co.ID).
			SetBillNo(apBillNo).
			SetIdempotencyKey("DEV-BILL-" + apBillNo).
			SetDirection(financebillent.DirectionPAYABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(suppCosco.ID).
			SetSettlementPartyName(suppCosco.LegalName).
			SetSettlementAccountID(acc4ID).
			SetSettlementAccountName(acc4Name).
			SetSettlementAccountHolder(acc4Holder).
			SetSettlementBankName(bank4Name).
			SetSettlementBankAccount(acc4No).
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
		if err != nil {
			return fmt.Errorf("创建账单 %s: %w", apBillNo, err)
		}
	}

	// 应付客户账单：客户兼作协载同行，与应收形成同客商双向余额，构成对冲场景
	apBill2No := "AP26090002"
	apB2, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(apBill2No)).First(ctx)
	if apB2 == nil && party1 != nil {
		account5, _ := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(party1.ID), partneraccountent.CurrencyEQ("CNY")).First(ctx)
		acc5Name, acc5Holder, bank5Name, acc5No := "对公结算户", party1.LegalName, "工商银行", "5719000122334455"
		var acc5ID uuid.UUID
		if account5 != nil {
			acc5ID = account5.ID
			acc5Name = account5.Name
			acc5Holder = account5.AccountHolder
			bank5Name = account5.BankName
			acc5No = account5.AccountNo
		}
		_, err := tx.FinanceBill.Create().
			SetOrganizationID(co.ID).
			SetBillNo(apBill2No).
			SetIdempotencyKey("DEV-BILL-" + apBill2No).
			SetDirection(financebillent.DirectionPAYABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(party1.ID).
			SetSettlementPartyName(party1.LegalName).
			SetSettlementAccountID(acc5ID).
			SetSettlementAccountName(acc5Name).
			SetSettlementAccountHolder(acc5Holder).
			SetSettlementBankName(bank5Name).
			SetSettlementBankAccount(acc5No).
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
		if err != nil {
			return fmt.Errorf("创建账单 %s: %w", apBill2No, err)
		}
	}

	// 对冲草稿：勾选 CUST-ML-003 名下 1500 应收（AR26090003）与 1500 应付，
	// 待财务复核后确认。
	nettingNo := "NT26090001"
	nt, _ := tx.FinanceNetting.Query().Where(financenettingent.OrganizationIDEQ(co.ID), financenettingent.NettingNoEQ(nettingNo)).First(ctx)
	arDraft, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ("AR26090003")).First(ctx)
	apCustomer, _ := tx.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(co.ID), financebillent.BillNoEQ(apBill2No)).First(ctx)
	if nt == nil && arDraft != nil && apCustomer != nil {
		createdNt, err := tx.FinanceNetting.Create().
			SetOrganizationID(co.ID).
			SetNettingNo(nettingNo).
			SetIdempotencyKey("DEV-NT-" + nettingNo).
			SetRequestHash("DEVSEEDNT26090001").
			SetStatus(financenettingent.StatusDRAFT).
			SetSettlementPartyID(party1.ID).
			SetSettlementPartyName(party1.LegalName).
			SetCurrency("CNY").
			SetAmount("1500.00000000").
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("1500.00000000").
			SetPayableBaseAmount("1500.00000000").
			SetExchangeGainLoss("0.00000000").
			SetNote("客户协载应付与应收运费对冲，待财务复核确认").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建对冲 %s: %w", nettingNo, err)
		}

		ntAllocExists, _ := tx.FinanceNettingAllocation.Query().Where(financenettingallocationent.NettingIDEQ(createdNt.ID)).Exist(ctx)
		if !ntAllocExists {
			_, _ = tx.FinanceNettingAllocation.Create().
				SetNettingID(createdNt.ID).
				SetBillID(arDraft.ID).
				SetBillNo(arDraft.BillNo).
				SetDirection(financenettingallocationent.DirectionRECEIVABLE).
				SetAmount("1500.00000000").
				SetBaseCurrencyAmount("1500.00000000").
				SetActive(true).
				Save(ctx)
			_, _ = tx.FinanceNettingAllocation.Create().
				SetNettingID(createdNt.ID).
				SetBillID(apCustomer.ID).
				SetBillNo(apCustomer.BillNo).
				SetDirection(financenettingallocationent.DirectionPAYABLE).
				SetAmount("1500.00000000").
				SetBaseCurrencyAmount("1500.00000000").
				SetActive(true).
				Save(ctx)
		}
	}

	return nil
}

type lockSeedSpec struct {
	lockSource      orderlockrecordent.LockSource
	manualLockedBy  uuid.UUID
	autoTriggerType orderlockrecordent.TriggerType
	autoResourceID  uuid.UUID
	autoTriggeredBy uuid.UUID
}

// ensureOrderLockedWithRecord 模拟真实锁定链路：为已具备 ACTIVE 主单关系的订单
// 创建 SE 单证不可变版本快照、订单锁字段与锁定记录。已存在种子锁定记录时直接
// 返回；订单已被真实业务锁定（锁代次大于零）时不改写既有锁事实。仅支持
// DIRECT 单证结构，HOUSE 锁定还需要 HBL 版本与快照，不在本种子范围。
func ensureOrderLockedWithRecord(ctx context.Context, sc *seedContext, ord *ent.Order, spec lockSeedSpec) (*ent.OrderLockRecord, error) {
	tx := sc.tx
	company := sc.company

	idemKey := "DEV-LOCK-" + ord.OrderNo
	existing, _ := tx.OrderLockRecord.Query().Where(
		orderlockrecordent.OrganizationIDEQ(company.ID),
		orderlockrecordent.IdempotencyKeyEQ(idemKey),
	).First(ctx)
	if existing != nil {
		return existing, nil
	}
	if ord.LockGeneration > 0 {
		byGen, err := tx.OrderLockRecord.Query().Where(
			orderlockrecordent.OrderIDEQ(ord.ID),
			orderlockrecordent.GenerationEQ(ord.LockGeneration),
		).First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, err
		}
		if byGen == nil {
			return nil, fmt.Errorf("订单 %s 已有锁代次 %d 但无锁定记录，跳过种子锁定", ord.OrderNo, ord.LockGeneration)
		}
		return byGen, nil
	}

	link, err := tx.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(company.ID),
		seamasterbillorderlinkent.OrderIDEQ(ord.ID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("订单 %s 缺少 ACTIVE 主单关系: %w", ord.OrderNo, err)
	}
	hblCount, err := tx.SeaHouseBill.Query().Where(
		seahousebillent.OrganizationIDEQ(company.ID),
		seahousebillent.OrderIDEQ(ord.ID),
	).Count(ctx)
	if err != nil {
		return nil, err
	}
	if hblCount > 0 {
		return nil, fmt.Errorf("订单 %s 为 HOUSE 结构，种子锁定仅支持 DIRECT", ord.OrderNo)
	}
	mbl, err := tx.SeaMasterBill.Get(ctx, link.MasterBillID)
	if err != nil {
		return nil, err
	}
	exec, err := tx.SeaTransportExecution.Get(ctx, link.TransportExecutionID)
	if err != nil {
		return nil, err
	}

	lockedAt := time.Now().UTC()
	if ord.LockedAt != nil {
		lockedAt = *ord.LockedAt
	}
	lockGeneration := uint64(1)
	orderVersionAtLock := ord.Version + 1

	var manualActor *uuid.UUID
	orderUpdate := tx.Order.UpdateOne(ord).
		SetLockedAt(lockedAt).
		SetLockGeneration(lockGeneration).
		SetLockSource(orderent.LockSource(spec.lockSource)).
		SetVersion(orderVersionAtLock)
	recordCreate := tx.OrderLockRecord.Create().
		SetOrganizationID(company.ID).
		SetOrderID(ord.ID).
		SetOrderNo(ord.OrderNo).
		SetBusinessType(orderlockrecordent.BusinessTypeSE).
		SetGeneration(lockGeneration).
		SetLockSource(spec.lockSource).
		SetLockedAt(lockedAt).
		SetOrderVersionAtLock(orderVersionAtLock).
		SetIdempotencyKey(idemKey).
		SetRequestFingerprint("DEV-LOCK-FP-" + ord.OrderNo).
		SetMasterBillID(mbl.ID).
		SetTransportExecutionID(exec.ID)
	if spec.lockSource == orderlockrecordent.LockSourceMANUAL {
		orderUpdate = orderUpdate.SetLockedBy(spec.manualLockedBy)
		recordCreate = recordCreate.SetLockedBy(spec.manualLockedBy)
		manualActor = &spec.manualLockedBy
	} else {
		orderUpdate = orderUpdate.
			ClearLockedBy().
			SetAutoLockTriggerType(orderent.AutoLockTriggerType(spec.autoTriggerType)).
			SetAutoLockTriggerResourceID(spec.autoResourceID).
			SetAutoLockTriggeredBy(spec.autoTriggeredBy)
		recordCreate = recordCreate.
			SetTriggerType(spec.autoTriggerType).
			SetTriggerResourceID(spec.autoResourceID).
			SetTriggeredBy(spec.autoTriggeredBy)
	}
	if _, err := orderUpdate.Save(ctx); err != nil {
		return nil, err
	}

	mblVersionID, err := ensureSeedMasterBillVersion(ctx, tx, company.ID, mbl, manualActor)
	if err != nil {
		return nil, err
	}
	teVersionID, err := ensureSeedTransportExecutionVersion(ctx, tx, company.ID, exec, manualActor)
	if err != nil {
		return nil, err
	}
	rec, err := recordCreate.
		SetMasterBillVersionID(mblVersionID).
		SetTransportExecutionVersionID(teVersionID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// ensureSeedMasterBillVersion 为锁定快照创建 MBL 不可变版本；开发种子使用稳定
// 占位内容哈希，后续真实锁定会按内容哈希比对并自行生成新版本。
func ensureSeedMasterBillVersion(ctx context.Context, tx *ent.Tx, orgID uuid.UUID, mbl *ent.SeaMasterBill, actorID *uuid.UUID) (uuid.UUID, error) {
	if mbl.CurrentVersionID != nil {
		return *mbl.CurrentVersionID, nil
	}
	nextNo := uint64(1)
	latest, _ := tx.SeaMasterBillVersion.Query().
		Where(seamasterbillversionent.MasterBillIDEQ(mbl.ID)).
		Order(ent.Desc(seamasterbillversionent.FieldVersionNo)).
		First(ctx)
	if latest != nil {
		nextNo = latest.VersionNo + 1
	}
	created, err := tx.SeaMasterBillVersion.Create().
		SetOrganizationID(orgID).
		SetMasterBillID(mbl.ID).
		SetVersionNo(nextNo).
		SetSourceEntityVersion(mbl.Version).
		SetShippingLineID(mbl.ShippingLineID).
		SetMasterNo(mbl.MasterNo).
		SetNormalizedMasterNo(mbl.NormalizedMasterNo).
		SetStatus(seamasterbillversionent.Status(mbl.Status)).
		SetContentHash("devseed-mbl-" + mbl.MasterNo).
		SetSource(seamasterbillversionent.SourceORDER_LOCK).
		SetNillableCreatedBy(actorID).
		Save(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.SeaMasterBill.UpdateOneID(mbl.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

// ensureSeedTransportExecutionVersion 为锁定快照创建航次执行不可变版本，语义与
// MBL 版本一致。
func ensureSeedTransportExecutionVersion(ctx context.Context, tx *ent.Tx, orgID uuid.UUID, exec *ent.SeaTransportExecution, actorID *uuid.UUID) (uuid.UUID, error) {
	if exec.CurrentVersionID != nil {
		return *exec.CurrentVersionID, nil
	}
	nextNo := uint64(1)
	latest, _ := tx.SeaTransportExecutionVersion.Query().
		Where(seatransportexecutionversionent.TransportExecutionIDEQ(exec.ID)).
		Order(ent.Desc(seatransportexecutionversionent.FieldVersionNo)).
		First(ctx)
	if latest != nil {
		nextNo = latest.VersionNo + 1
	}
	created, err := tx.SeaTransportExecutionVersion.Create().
		SetOrganizationID(orgID).
		SetTransportExecutionID(exec.ID).
		SetVersionNo(nextNo).
		SetSourceEntityVersion(exec.Version).
		SetShippingLineID(exec.ShippingLineID).
		SetNillableOriginLocationID(exec.OriginLocationID).
		SetNillableDischargeLocationID(exec.DischargeLocationID).
		SetVesselName(exec.VesselName).
		SetVoyageNo(exec.VoyageNo).
		SetContentHash("devseed-te-" + exec.VesselName + "-" + exec.VoyageNo).
		SetSource(seatransportexecutionversionent.SourceORDER_LOCK).
		SetNillableCreatedBy(actorID).
		Save(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.SeaTransportExecution.UpdateOneID(exec.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

// 8. 订单锁治理：核销自动锁、钉钉解锁审批与锁后费用补录
func seedOrderLockGovernance(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	co := sc.company

	zhaofang := sc.users["zhaofang"]
	zhangqiang := sc.users["zhangqiang"]
	if zhaofang == nil || zhangqiang == nil {
		return fmt.Errorf("缺少订单锁治理所需测试员工 (zhaofang/zhangqiang)")
	}

	// SE26090002 应收已全额核销，模拟核销动作触发的系统自动锁定
	ord2 := sc.orders["SE26090002"]
	if ord2 == nil {
		return fmt.Errorf("缺少订单 SE26090002，无法注入自动锁")
	}
	vf, err := tx.FinanceVerification.Query().Where(
		financeverificationent.OrganizationIDEQ(co.ID),
		financeverificationent.VerificationNoEQ("VF-AR-26090001"),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询核销记录 VF-AR-26090001: %w", err)
	}
	lockRec2, err := ensureOrderLockedWithRecord(ctx, sc, ord2, lockSeedSpec{
		lockSource:      orderlockrecordent.LockSourceAUTO_SETTLEMENT,
		autoTriggerType: orderlockrecordent.TriggerTypeVERIFICATION,
		autoResourceID:  vf.ID,
		autoTriggeredBy: zhaofang.ID,
	})
	if err != nil {
		return fmt.Errorf("自动锁定订单 SE26090002: %w", err)
	}

	// 销售张强对自动锁订单发起钉钉解锁审批（审批中）
	unlockExists, _ := tx.OrderUnlockRequest.Query().Where(
		orderunlockrequestent.OrganizationIDEQ(co.ID),
		orderunlockrequestent.IdempotencyKeyEQ("DEV-UNLOCK-SE26090002"),
	).Exist(ctx)
	if !unlockExists {
		_, err := tx.OrderUnlockRequest.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord2.ID).
			SetOrderNo(ord2.OrderNo).
			SetBusinessType(orderunlockrequestent.BusinessTypeSE).
			SetLockRecordID(lockRec2.ID).
			SetLockGeneration(lockRec2.Generation).
			SetRequestedBy(zhangqiang.ID).
			SetRequestedAt(time.Now().UTC()).
			SetReason("客户要求更正提单收货人抬头，需解锁改单后重新提交锁定").
			SetExpectedOrderVersion(lockRec2.OrderVersionAtLock).
			SetIdempotencyKey("DEV-UNLOCK-SE26090002").
			SetRequestFingerprint("DEV-UNLOCK-FP-SE26090002").
			SetRoute(orderunlockrequestent.RouteDINGTALK_APPROVAL).
			SetStatus(orderunlockrequestent.StatusPENDING_APPROVAL).
			SetDingtalkProcessInstanceID("DEV-DING-SE26090002-0001").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建解锁审批申请: %w", err)
		}
	}

	// 人工锁定票 SE26090004 的锁后应付补录：一笔待审批 + 一笔已审批并生成费用
	ord4, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(co.ID), orderent.OrderNoEQ("SE26090004")).First(ctx)
	if err != nil {
		return fmt.Errorf("查询订单 SE26090004: %w", err)
	}
	if ord4.LockGeneration == 0 {
		return fmt.Errorf("订单 SE26090004 未锁定，无法注入费用补录")
	}
	wangli := sc.users["wangli"]

	supp1Key := "DEV-SUPP-SE26090004-TRUCK"
	supp1Exists, _ := tx.OrderFeeSupplementRequest.Query().Where(
		orderfeesupplementrequestent.OrganizationIDEQ(co.ID),
		orderfeesupplementrequestent.IdempotencyKeyEQ(supp1Key),
	).Exist(ctx)
	if !supp1Exists {
		_, err := tx.OrderFeeSupplementRequest.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord4.ID).
			SetLockBasis(orderfeesupplementrequestent.LockBasisBUSINESS).
			SetBusinessLockGeneration(ord4.LockGeneration).
			SetIdempotencyKey(supp1Key).
			SetRequestFingerprint("DEV-SUPP-FP-SE26090004-TRUCK").
			SetDirection(orderfeesupplementrequestent.DirectionPAYABLE).
			SetFeeSettingID(sc.feeSettings["TRUCK"].ID).
			SetFeeCode("TRUCK").
			SetFeeName("集装箱拖车费").
			SetSettlementPartyID(sc.partners["SUPP-YJTC-04"].ID).
			SetBillingUnitID(sc.billingUnits["CONT"].ID).
			SetBillingUnit(sc.billingUnits["CONT"].Name).
			SetTaxRate("9.00").
			SetQuantity("1.0000").
			SetUnitPrice("780.0000").
			SetTotalAmount("780.00000000").
			SetNetAmount("780.00000000").
			SetTaxAmount("0.00000000").
			SetCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(orderfeesupplementrequestent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-21").
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("780.00000000").
			SetExpenseDate("2026-09-21").
			SetReason("旺季拖车附加费漏录，供应商账单已到，需补录应付成本").
			SetRequestedBy(wangli.ID).
			SetRequestedAt(time.Now().UTC()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建费用补录申请 (拖车费): %w", err)
		}
	}

	supp2Key := "DEV-SUPP-SE26090004-STORAGE"
	supp2, _ := tx.OrderFeeSupplementRequest.Query().Where(
		orderfeesupplementrequestent.OrganizationIDEQ(co.ID),
		orderfeesupplementrequestent.IdempotencyKeyEQ(supp2Key),
	).First(ctx)
	if supp2 == nil {
		createdSupp, err := tx.OrderFeeSupplementRequest.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord4.ID).
			SetLockBasis(orderfeesupplementrequestent.LockBasisBUSINESS).
			SetBusinessLockGeneration(ord4.LockGeneration).
			SetIdempotencyKey(supp2Key).
			SetRequestFingerprint("DEV-SUPP-FP-SE26090004-STORAGE").
			SetDirection(orderfeesupplementrequestent.DirectionPAYABLE).
			SetFeeSettingID(sc.feeSettings["STORAGE"].ID).
			SetFeeCode("STORAGE").
			SetFeeName("码头堆存费").
			SetSettlementPartyID(sc.partners["SUPP-SHCC-05"].ID).
			SetBillingUnitID(sc.billingUnits["CONT"].ID).
			SetBillingUnit(sc.billingUnits["CONT"].Name).
			SetTaxRate("6.00").
			SetQuantity("1.0000").
			SetUnitPrice("350.0000").
			SetTotalAmount("350.00000000").
			SetNetAmount("350.00000000").
			SetTaxAmount("0.00000000").
			SetCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(orderfeesupplementrequestent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate("2026-09-21").
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("350.00000000").
			SetExpenseDate("2026-09-21").
			SetReason("目的港免堆期外的堆存费漏录，仓库账单已核对").
			SetRequestedBy(wangli.ID).
			SetRequestedAt(time.Now().UTC()).
			SetStatus(orderfeesupplementrequestent.StatusAPPROVED).
			SetDecidedBy(zhaofang.ID).
			SetDecidedAt(time.Now().UTC()).
			SetDecisionReason("堆存费发票已核验，金额无误，同意补录").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建费用补录申请 (堆存费): %w", err)
		}
		supp2 = createdSupp

		// 审批通过后按快照生成的 CONFIRMED 费用（supplement_request_id 反向关联）
		feeKey := "DEV-FEE-SE26090004-PAYABLE-STORAGE-SUPP"
		feeExists, _ := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(ord4.ID), orderfeeent.IdempotencyKeyEQ(feeKey)).Exist(ctx)
		if !feeExists {
			_, err := tx.OrderFee.Create().
				SetOrderID(ord4.ID).
				SetIdempotencyKey(feeKey).
				SetDirection(orderfeeent.DirectionPAYABLE).
				SetStatus(orderfeeent.StatusCONFIRMED).
				SetFeeSettingID(sc.feeSettings["STORAGE"].ID).
				SetFeeCode("STORAGE").
				SetFeeName("码头堆存费(补录)").
				SetSettlementPartyID(sc.partners["SUPP-SHCC-05"].ID).
				SetBillingUnitID(sc.billingUnits["CONT"].ID).
				SetBillingUnit(sc.billingUnits["CONT"].Name).
				SetTaxRate("6.00").
				SetQuantity("1.0000").
				SetUnitPrice("350.0000").
				SetTotalAmount("350.00000000").
				SetNetAmount("350.00000000").
				SetTaxAmount("0.00000000").
				SetCurrency("CNY").
				SetExchangeRate("1.00000000").
				SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
				SetExchangeRateDate("2026-09-21").
				SetBaseCurrency("CNY").
				SetBaseCurrencyAmount("350.00000000").
				SetExpenseDate("2026-09-21").
				SetSupplementRequestID(supp2.ID).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建补录生成费用 (堆存费): %w", err)
			}
		}
	}
	return nil
}

// 7. 提成方案与分配
func seedCommissionRules(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	co := sc.company

	rule1Name := "海运出口业务员毛利提成方案 (标准10%)"
	r1, _ := tx.FinanceCommissionRule.Query().Where(
		financecommissionruleent.OrganizationIDEQ(co.ID),
		financecommissionruleent.NameEQ(rule1Name),
	).First(ctx)
	if r1 == nil {
		created, err := tx.FinanceCommissionRule.Create().
			SetOrganizationID(co.ID).
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
		financecommissionruleent.OrganizationIDEQ(co.ID),
		financecommissionruleent.NameEQ(rule2Name),
	).First(ctx)
	if r2 == nil {
		created, err := tx.FinanceCommissionRule.Create().
			SetOrganizationID(co.ID).
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
			financecommissionruleassignmentent.OrganizationIDEQ(co.ID),
			financecommissionruleassignmentent.RuleIDEQ(r1.ID),
			financecommissionruleassignmentent.EmployeeIDEQ(u.ID),
		).Exist(ctx)
		if !aExists {
			_, _ = tx.FinanceCommissionRuleAssignment.Create().
				SetOrganizationID(co.ID).
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
			financecommissionruleassignmentent.OrganizationIDEQ(co.ID),
			financecommissionruleassignmentent.RuleIDEQ(r2.ID),
			financecommissionruleassignmentent.EmployeeIDEQ(opUser.ID),
		).Exist(ctx)
		if !aExists {
			_, _ = tx.FinanceCommissionRuleAssignment.Create().
				SetOrganizationID(co.ID).
				SetRuleID(r2.ID).
				SetEmployeeID(opUser.ID).
				SetEffectiveFrom("2026-01-01").
				SetCreatedBy(sc.adminUser.ID).
				Save(ctx)
		}
	}

	return nil
}

// 9. 提成经营归属、台账与月度申请
func seedCommissionLedger(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	co := sc.company
	if sc.adminUser == nil {
		return fmt.Errorf("缺少管理员用户，无法注入提成数据")
	}

	// 经营归属：全部种子订单补销售归属；SE26090002 另补操作与单证客服归属
	for _, orderNo := range sortedOrderNos(sc.orders) {
		ord := sc.orders[orderNo]
		roles := []orderpersonnelent.Role{orderpersonnelent.RoleSALES}
		if orderNo == "SE26090002" {
			roles = append(roles, orderpersonnelent.RoleOPERATOR, orderpersonnelent.RoleDOCUMENT)
		}
		for _, role := range roles {
			personnel, pErr := tx.OrderPersonnel.Query().Where(
				orderpersonnelent.OrderIDEQ(ord.ID),
				orderpersonnelent.RoleEQ(role),
			).First(ctx)
			if pErr != nil {
				continue
			}
			user, uErr := tx.User.Get(ctx, personnel.UserID)
			if uErr != nil {
				continue
			}
			attributionRole := ordercommissionattributionent.PersonnelRoleSALES
			switch role {
			case orderpersonnelent.RoleOPERATOR:
				attributionRole = ordercommissionattributionent.PersonnelRoleOPERATOR
			case orderpersonnelent.RoleDOCUMENT:
				attributionRole = ordercommissionattributionent.PersonnelRoleCUSTOMER_SERVICE
			}
			aExists, _ := tx.OrderCommissionAttribution.Query().Where(
				ordercommissionattributionent.OrderIDEQ(ord.ID),
				ordercommissionattributionent.PersonnelRoleEQ(attributionRole),
				ordercommissionattributionent.EmployeeIDEQ(user.ID),
			).Exist(ctx)
			if !aExists {
				if _, err := tx.OrderCommissionAttribution.Create().
					SetOrganizationID(co.ID).
					SetOrderID(ord.ID).
					SetCustomerID(ord.CustomerID).
					SetSourceAssignmentID(personnel.ID).
					SetEmployeeID(user.ID).
					SetEmployeeName(user.DisplayName).
					SetPersonnelRole(attributionRole).
					SetAttributedAt(personnel.CreatedAt).
					Save(ctx); err != nil {
					return fmt.Errorf("创建订单经营归属 %s/%s: %w", orderNo, attributionRole, err)
				}
			}
		}
	}

	vf, err := tx.FinanceVerification.Query().Where(
		financeverificationent.OrganizationIDEQ(co.ID),
		financeverificationent.VerificationNoEQ("VF-AR-26090001"),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询核销记录 VF-AR-26090001: %w", err)
	}
	ord2 := sc.orders["SE26090002"]
	if ord2 == nil {
		return fmt.Errorf("缺少订单 SE26090002，无法注入提成台账")
	}
	cust, _ := tx.Partner.Get(ctx, ord2.CustomerID)

	type commissionDef struct {
		no         string
		username   string
		ruleName   string
		rate       string
		basis      string
		personRole string
	}
	commissions := []commissionDef{
		{no: "CM26090001", username: "zhangqiang", ruleName: "海运出口业务员毛利提成方案 (标准10%)", rate: "10.0000", basis: "REALIZED_PROFIT", personRole: "SALES"},
		{no: "CM26090002", username: "liming", ruleName: "海运出口操作员单票提成方案 (3%)", rate: "3.0000", basis: "REALIZED_PROFIT", personRole: "OPERATOR"},
	}

	var salesPersonnel *ent.OrderPersonnel
	if p, pErr := tx.OrderPersonnel.Query().Where(
		orderpersonnelent.OrderIDEQ(ord2.ID), orderpersonnelent.RoleEQ(orderpersonnelent.RoleSALES),
	).First(ctx); pErr == nil {
		salesPersonnel = p
	}
	var operatorPersonnel *ent.OrderPersonnel
	if p, pErr := tx.OrderPersonnel.Query().Where(
		orderpersonnelent.OrderIDEQ(ord2.ID), orderpersonnelent.RoleEQ(orderpersonnelent.RoleOPERATOR),
	).First(ctx); pErr == nil {
		operatorPersonnel = p
	}

	for _, def := range commissions {
		employee := sc.users[def.username]
		if employee == nil {
			return fmt.Errorf("缺少提成员工 %s", def.username)
		}
		rule, _ := tx.FinanceCommissionRule.Query().Where(
			financecommissionruleent.OrganizationIDEQ(co.ID),
			financecommissionruleent.NameEQ(def.ruleName),
		).First(ctx)
		personnel := salesPersonnel
		if def.personRole == "OPERATOR" {
			personnel = operatorPersonnel
		}
		if personnel == nil {
			return fmt.Errorf("缺少订单 SE26090002 的 %s 协作人员", def.personRole)
		}
		amount := "280.00000000"
		if def.rate == "3.0000" {
			amount = "84.00000000"
		}

		cmExists, _ := tx.FinanceCommission.Query().Where(
			financecommissionent.OrganizationIDEQ(co.ID),
			financecommissionent.CommissionNoEQ(def.no),
		).Exist(ctx)
		if !cmExists {
			created, cErr := tx.FinanceCommission.Create().
				SetOrganizationID(co.ID).
				SetCommissionNo(def.no).
				SetIdempotencyKey("DEV-CM-" + def.no).
				SetVerificationID(vf.ID).
				SetVerificationNo(vf.VerificationNo).
				SetEmployeeID(employee.ID).
				SetEmployeeName(employee.DisplayName).
				SetCustomerCount(1).
				SetOrderCount(1).
				SetFeeCount(3).
				SetRuleID(rule.ID).
				SetRuleName(rule.Name).
				SetPersonnelRole(def.personRole).
				SetCalculationBasis(def.basis).
				SetSourceFingerprint("devseed").
				SetStatus(financecommissionent.StatusCONFIRMED).
				SetBaseCurrency("CNY").
				SetRealizedRevenue("2800.00000000").
				SetAllocatedCost("0.00000000").
				SetRealizedProfit("2800.00000000").
				SetCommissionBaseAmount("2800.00000000").
				SetRatePercent(def.rate).
				SetCommissionAmount(amount).
				SetCommissionDate("2026-09-21").
				SetCnyExchangeRate("1.00000000").
				SetCnyExchangeRateSource(financecommissionent.CnyExchangeRateSourceBASE_CURRENCY).
				SetCnyExchangeRateDate("2026-09-21").
				SetCnyCommissionAmount(amount).
				Save(ctx)
			if cErr != nil {
				return fmt.Errorf("创建提成台账 %s: %w", def.no, cErr)
			}

			if _, lErr := tx.FinanceCommissionLine.Create().
				SetOrganizationID(co.ID).
				SetCommissionID(created.ID).
				SetOrderID(ord2.ID).
				SetOrderNo(ord2.OrderNo).
				SetOrderDate("2026-09-20").
				SetCustomerID(ord2.CustomerID).
				SetCustomerCode(strOr(cust.Code, "")).
				SetCustomerName(cust.LegalName).
				SetPersonnelAssignmentID(personnel.ID).
				SetPersonnelOrganizationID(co.ID).
				SetPersonnelAssignedAt(personnel.CreatedAt).
				SetFeeCount(3).
				SetEmployeeID(employee.ID).
				SetEmployeeName(employee.DisplayName).
				SetPersonnelRole(def.personRole).
				SetCalculationBasis(def.basis).
				SetBaseCurrency("CNY").
				SetRealizedRevenue("2800.00000000").
				SetAllocatedCost("0.00000000").
				SetRealizedProfit("2800.00000000").
				SetCommissionBaseAmount("2800.00000000").
				SetRatePercent(def.rate).
				SetCommissionAmount(amount).
				Save(ctx); lErr != nil {
				return fmt.Errorf("创建提成明细 %s: %w", def.no, lErr)
			}
		}
	}

	// 月度申请：张强 9 月申请待审批；李明 9 月申请已审批
	type applicationDef struct {
		username string
		status   financecommissionapplicationent.Status
		lineNo   string
	}
	applications := []applicationDef{
		{username: "zhangqiang", status: financecommissionapplicationent.StatusPENDING_REVIEW, lineNo: "CM26090001"},
		{username: "liming", status: financecommissionapplicationent.StatusAPPROVED, lineNo: "CM26090002"},
	}
	zhaofang := sc.users["zhaofang"]
	for _, app := range applications {
		employee := sc.users[app.username]
		if employee == nil {
			return fmt.Errorf("缺少申请员工 %s", app.username)
		}
		appExists, _ := tx.FinanceCommissionApplication.Query().Where(
			financecommissionapplicationent.OrganizationIDEQ(co.ID),
			financecommissionapplicationent.EmployeeIDEQ(employee.ID),
			financecommissionapplicationent.ApplicationMonthEQ("2026-09"),
		).Exist(ctx)
		if appExists {
			continue
		}
		cm, cmErr := tx.FinanceCommission.Query().Where(
			financecommissionent.OrganizationIDEQ(co.ID),
			financecommissionent.CommissionNoEQ(app.lineNo),
		).First(ctx)
		if cmErr != nil {
			return fmt.Errorf("查询提成 %s: %w", app.lineNo, cmErr)
		}

		createdApp, aErr := tx.FinanceCommissionApplication.Create().
			SetOrganizationID(co.ID).
			SetEmployeeID(employee.ID).
			SetApplicationMonth("2026-09").
			SetCoverageTo("2026-09-30").
			SetStatus(app.status).
			SetCommissionCount(1).
			SetBaseCurrency("CNY").
			SetTotalCommissionAmount(cm.CommissionAmount).
			SetTotalCnyCommissionAmount(cm.CnyCommissionAmount).
			SetSubmittedAt(time.Now().UTC()).
			SetSubmittedBy(employee.ID).
			Save(ctx)
		if aErr != nil {
			return fmt.Errorf("创建月度提成申请 %s: %w", app.username, aErr)
		}
		if app.status == financecommissionapplicationent.StatusAPPROVED {
			if _, err := createdApp.Update().
				SetDecidedAt(time.Now().UTC()).
				SetDecidedBy(zhaofang.ID).
				SetDecisionReason("9月提成核对无误，同意发放").
				Save(ctx); err != nil {
				return fmt.Errorf("审批月度提成申请 %s: %w", app.username, err)
			}
		}

		if _, err := tx.FinanceCommissionApplicationLine.Create().
			SetOrganizationID(co.ID).
			SetEmployeeID(employee.ID).
			SetApplicationID(createdApp.ID).
			SetCommissionID(cm.ID).
			SetCommissionDate(cm.CommissionDate).
			SetVerificationID(vf.ID).
			SetVerificationNo(vf.VerificationNo).
			SetPersonnelRole(strOr(cm.PersonnelRole, "")).
			SetRuleID(uuidOr(cm.RuleID, uuid.Nil)).
			SetRuleVersion(cm.RuleVersion).
			SetRuleName(strOr(cm.RuleName, "")).
			SetCalculationBasis(strOr(cm.CalculationBasis, "")).
			SetBaseCurrency("CNY").
			SetCommissionAmount(cm.CommissionAmount).
			SetCnyCommissionAmount(cm.CnyCommissionAmount).
			SetSourceFingerprint("devseed").
			Save(ctx); err != nil {
			return fmt.Errorf("创建月度提成申请明细 %s: %w", app.username, err)
		}
	}
	return nil
}

func sortedOrderNos(orders map[string]*ent.Order) []string {
	keys := make([]string, 0, len(orders))
	for k := range orders {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// 10. 海运单证操作：拆票、整体改配、共享箱、HBL 改单与作废、单证模式切换
func seedSeaDocumentOperations(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	co := sc.company
	if sc.adminUser == nil {
		return fmt.Errorf("缺少管理员用户，无法注入单证操作数据")
	}

	// —— 拆票：SE26090015(原票剩余) → SE26090016(拆出结果票) ——
	ord15 := sc.orders["SE26090015"]
	ord16 := sc.orders["SE26090016"]
	if ord15 == nil || ord16 == nil {
		return fmt.Errorf("缺少拆票订单 SE26090015/SE26090016")
	}
	link15, err := tx.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrderIDEQ(ord15.ID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询 SE26090015 ACTIVE 主单关系: %w", err)
	}
	link16, err := tx.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrderIDEQ(ord16.ID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询 SE26090016 ACTIVE 主单关系: %w", err)
	}

	splitIdem := "DEV-SPLIT-SE26090015"
	splitEvent, _ := tx.SeaOrderSplitEvent.Query().Where(
		seaorderspliteventent.OrganizationIDEQ(co.ID),
		seaorderspliteventent.IdempotencyKeyEQ(splitIdem),
	).First(ctx)
	if splitEvent == nil {
		splitEvent, err = tx.SeaOrderSplitEvent.Create().
			SetOrganizationID(co.ID).
			SetSourceOrderID(ord15.ID).
			SetSourceOrderNo(ord15.OrderNo).
			SetIdempotencyKey(splitIdem).
			SetRequestFingerprint("DEV-SPLIT-FP-SE26090015").
			SetNote("花园家具拼箱货分批出运，拆出 20 件至新操作票").
			SetSourceOrderVersion(ord15.Version).
			SetSourceLinkID(link15.ID).
			SetSourceLinkVersion(link15.Version).
			SetSourceAllocationVersion(1).
			SetBeforeSnapshot(json.RawMessage(`{"packages":50,"gross_weight_kg":650.0,"volume_cbm":5.8}`)).
			SetConservationSnapshot(json.RawMessage(`{"SE26090015":{"packages":30,"gross_weight_kg":390.0,"volume_cbm":3.48},"SE26090016":{"packages":20,"gross_weight_kg":260.0,"volume_cbm":2.32},"balanced":true}`)).
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建拆票事件: %w", err)
		}

		results := []struct {
			ord       *ent.Order
			role      seaordersplitresultent.ResultRole
			seq       int
			clientKey string
			initial   uuid.UUID
			final     uuid.UUID
			snapshot  string
		}{
			{ord15, seaordersplitresultent.ResultRoleORIGINAL, 0, "original", link15.MasterBillID, link15.MasterBillID, `{"order_no":"SE26090015","result_role":"ORIGINAL","packages":30,"gross_weight_kg":390.0,"volume_cbm":3.48}`},
			{ord16, seaordersplitresultent.ResultRoleCREATED, 1, "created-1", link15.MasterBillID, link16.MasterBillID, `{"order_no":"SE26090016","result_role":"CREATED","packages":20,"gross_weight_kg":260.0,"volume_cbm":2.32,"master_bill_changed":true}`},
		}
		for _, r := range results {
			if _, err := tx.SeaOrderSplitResult.Create().
				SetSplitEventID(splitEvent.ID).
				SetOrganizationID(co.ID).
				SetOrderID(r.ord.ID).
				SetOrderNo(r.ord.OrderNo).
				SetResultRole(r.role).
				SetSequence(r.seq).
				SetClientResultKey(r.clientKey).
				SetInitialMasterBillID(r.initial).
				SetFinalMasterBillID(r.final).
				SetResultSnapshot(json.RawMessage(r.snapshot)).
				Save(ctx); err != nil {
				return fmt.Errorf("创建拆票结果 %s: %w", r.ord.OrderNo, err)
			}
		}
	}

	// —— 整体改配：SE26090016 由 MSC VELA 航次改配至 MAERSK KOTKA 航次 ——
	reassignIdem := "DEV-REASSIGN-SE26090016"
	reassignExists, _ := tx.SeaOrderReassignmentEvent.Query().Where(
		seaorderreassignmenteventent.OrganizationIDEQ(co.ID),
		seaorderreassignmenteventent.IdempotencyKeyEQ(reassignIdem),
	).Exist(ctx)
	if !reassignExists {
		endedLink, err := tx.SeaMasterBillOrderLink.Create().
			SetOrganizationID(co.ID).
			SetMasterBillID(link15.MasterBillID).
			SetTransportExecutionID(link15.TransportExecutionID).
			SetOrderID(ord16.ID).
			SetStatus(seamasterbillorderlinkent.StatusENDED).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
			SetStartedAt(time.Now().Add(-48 * time.Hour)).
			SetEndedAt(time.Now()).
			SetEndedReason("整票改配：原航次舱位甩柜").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建改配前 ENDED 主单关系: %w", err)
		}

		responsible := sc.partners["SUPP-MSC-02"]
		if _, err := tx.SeaOrderReassignmentEvent.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord16.ID).
			SetOrderNo(ord16.OrderNo).
			SetSplitEventID(splitEvent.ID).
			SetIdempotencyKey(reassignIdem).
			SetRequestFingerprint("DEV-REASSIGN-FP-SE26090016").
			SetPreviousMasterBillID(link15.MasterBillID).
			SetTargetMasterBillID(link16.MasterBillID).
			SetPreviousTransportExecutionID(link15.TransportExecutionID).
			SetTargetTransportExecutionID(link16.TransportExecutionID).
			SetPreviousLinkID(endedLink.ID).
			SetTargetLinkID(link16.ID).
			SetPreviousLinkVersion(1).
			SetTargetLinkVersion(link16.Version).
			SetReason("原航次舱位不足被承运人甩柜，整票改配至马士基航次").
			SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityTypeCARRIER).
			SetResponsiblePartnerID(responsible.ID).
			SetResponsiblePartnerName(responsible.LegalName).
			SetBeforeSnapshot(json.RawMessage(`{"master_bill_no":"MSCU77889900","vessel":"MSC VELA","voyage":"2650E"}`)).
			SetAfterSnapshot(json.RawMessage(`{"master_bill_no":"MAEU31112233","vessel":"MAERSK KOTKA","voyage":"2610W"}`)).
			SetCreatedBy(sc.adminUser.ID).
			SetConfirmedByParty("地中海航运代理（上海）有限公司").
			SetConfirmedAt(time.Now()).
			SetConfirmationNote("船司订舱处邮件确认甩柜并同意改配至 MAERSK KOTKA 2610W").
			Save(ctx); err != nil {
			return fmt.Errorf("创建改配事件: %w", err)
		}
	}

	// —— 共享箱：MSC OSCAR 航次上 MSCU9900112 由 SE26090008/09 两票拼用 ——
	execOscar, err := tx.SeaTransportExecution.Query().Where(
		seatransportexecutionent.OrganizationIDEQ(co.ID),
		seatransportexecutionent.VesselNameEQ("MSC OSCAR"),
		seatransportexecutionent.VoyageNoEQ("2640E"),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询 MSC OSCAR 航次: %w", err)
	}
	sharedExists, _ := tx.SeaSharedContainer.Query().Where(
		seasharedcontainerent.OrganizationIDEQ(co.ID),
		seasharedcontainerent.ContainerNoEQ("MSCU9900112"),
	).Exist(ctx)
	if !sharedExists {
		createdShared, err := tx.SeaSharedContainer.Create().
			SetOrganizationID(co.ID).
			SetTransportExecutionID(execOscar.ID).
			SetContainerNo("MSCU9900112").
			SetContainerSpecID(sc.containerSpecs["40HQ"].ID).
			SetSealNo("MSC881122").
			SetPackageCount(550).
			SetGrossWeightKg("5500.000").
			SetVolumeCbm("50.500000").
			SetStatus(seasharedcontainerent.StatusCONFIRMED).
			SetConfirmedAt(time.Now()).
			SetConfirmedBy(sc.adminUser.ID).
			SetNote("一主多分单拼舱共享箱，由两票分单按件重体分配").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建共享箱: %w", err)
		}
		sharedOrders := []struct {
			orderNo string
			pkgs    int
			weight  string
			volume  string
		}{
			{"SE26090008", 200, "2400.000", "22.500000"},
			{"SE26090009", 350, "3100.000", "28.000000"},
		}
		for _, so := range sharedOrders {
			ord := sc.orders[so.orderNo]
			hbl, _ := tx.SeaHouseBill.Query().Where(
				seahousebillent.OrderIDEQ(ord.ID),
				seahousebillent.StatusEQ(seahousebillent.StatusCONFIRMED),
			).First(ctx)
			cargo, _ := tx.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(ord.ID)).First(ctx)
			if _, err := tx.SeaSharedContainerAllocation.Create().
				SetOrganizationID(co.ID).
				SetSharedContainerID(createdShared.ID).
				SetOrderID(ord.ID).
				SetHouseBillID(hbl.ID).
				SetCargoItemID(cargo.ID).
				SetPackageCount(so.pkgs).
				SetGrossWeightKg(so.weight).
				SetVolumeCbm(so.volume).
				Save(ctx); err != nil {
				return fmt.Errorf("创建共享箱分配 %s: %w", so.orderNo, err)
			}
		}
	}

	// —— HBL 改单与作废：SE26090017 旧分单改收货人后作废，重出 RC-HBL26090017R ——
	ord17 := sc.orders["SE26090017"]
	if ord17 == nil {
		return fmt.Errorf("缺少订单 SE26090017")
	}
	mbl17, err := tx.SeaMasterBill.Query().Where(
		seamasterbillent.OrganizationIDEQ(co.ID),
		seamasterbillent.MasterNoEQ("ONEY55667788"),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询主单 ONEY55667788: %w", err)
	}
	voidIdem := "DEV-VOID-SE26090017-HBL"
	voidExists, _ := tx.SeaDocumentVoidEvent.Query().Where(
		seadocumentvoideventent.OrganizationIDEQ(co.ID),
		seadocumentvoideventent.IdempotencyKeyEQ(voidIdem),
	).Exist(ctx)
	if !voidExists {
		oldHbl, err := tx.SeaHouseBill.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord17.ID).
			SetMasterBillID(mbl17.ID).
			SetHouseNo("RC-HBL26090017").
			SetNormalizedHouseNo("RC-HBL26090017").
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillent.StatusVOIDED).
			SetShipperText("TransGlobal Trading (HK) Co., Limited\nUNIT 8, 12/F, KWAI CHUNG, HONG KONG").
			SetConsigneeText("TO ORDER OF SHIPPER").
			SetNotifyPartyText("PACIFIC FRAME & MIRROR INC.\n4500 E PICO BLVD, LOS ANGELES, CA").
			SetPackageCount(intOr(ord17.TotalPackages, 0)).
			SetGrossWeightKg(floatOr(ord17.TotalGrossWeightKg, 0)).
			SetVolumeCbm(floatOr(ord17.TotalVolumeCbm, 0)).
			SetGoodsDescriptionText(ord17.GoodsDescription).
			SetFreightTerms("FREIGHT COLLECT").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建作废演示分单: %w", err)
		}

		// v1 制单确认 → v2 修改收货人 → v3 作废（当前版本）
		v1, err := tx.SeaHouseBillVersion.Create().
			SetOrganizationID(co.ID).
			SetHouseBillID(oldHbl.ID).
			SetOrderID(ord17.ID).
			SetMasterBillID(mbl17.ID).
			SetVersionNo(1).
			SetSourceEntityVersion(oldHbl.Version).
			SetHouseNo(oldHbl.HouseNo).
			SetNormalizedHouseNo(oldHbl.NormalizedHouseNo).
			SetIssuerSource(seahousebillversionent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillversionent.StatusCONFIRMED).
			SetContentHash("devseed-hbl-RC-HBL26090017-v1").
			SetSource(seahousebillversionent.SourceAMENDMENT).
			SetReason("首次制单确认").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建分单版本 v1: %w", err)
		}
		v2, err := tx.SeaHouseBillVersion.Create().
			SetOrganizationID(co.ID).
			SetHouseBillID(oldHbl.ID).
			SetOrderID(ord17.ID).
			SetMasterBillID(mbl17.ID).
			SetVersionNo(2).
			SetSourceEntityVersion(oldHbl.Version).
			SetHouseNo(oldHbl.HouseNo).
			SetNormalizedHouseNo(oldHbl.NormalizedHouseNo).
			SetIssuerSource(seahousebillversionent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillversionent.StatusCONFIRMED).
			SetContentHash("devseed-hbl-RC-HBL26090017-v2").
			SetSource(seahousebillversionent.SourceAMENDMENT).
			SetReason("按客户指示修改收货人抬头").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建分单版本 v2: %w", err)
		}
		v3, err := tx.SeaHouseBillVersion.Create().
			SetOrganizationID(co.ID).
			SetHouseBillID(oldHbl.ID).
			SetOrderID(ord17.ID).
			SetMasterBillID(mbl17.ID).
			SetVersionNo(3).
			SetSourceEntityVersion(oldHbl.Version).
			SetHouseNo(oldHbl.HouseNo).
			SetNormalizedHouseNo(oldHbl.NormalizedHouseNo).
			SetIssuerSource(seahousebillversionent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillversionent.StatusVOIDED).
			SetContentHash("devseed-hbl-RC-HBL26090017-v3").
			SetSource(seahousebillversionent.SourceVOID).
			SetReason("客户改由记名直放，作废旧分单重出").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建分单版本 v3: %w", err)
		}
		if _, err := oldHbl.Update().SetCurrentVersionID(v3.ID).Save(ctx); err != nil {
			return fmt.Errorf("推进作废演示分单当前版本: %w", err)
		}

		if _, err := tx.SeaDocumentVoidEvent.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord17.ID).
			SetDocumentType(seadocumentvoideventent.DocumentTypeHOUSE).
			SetHouseBillID(oldHbl.ID).
			SetHouseBillVersionID(v3.ID).
			SetPreviousHouseBillVersionID(v2.ID).
			SetPreviousStatus("CONFIRMED").
			SetVoidedStatus("VOIDED").
			SetReason("客户改由记名直放，旧分单作废并重出 RC-HBL26090017R").
			SetImpactSummary("作废分单不影响主单与已录费用，重出分单沿用原箱货数据").
			SetCreatedBy(sc.adminUser.ID).
			SetIdempotencyKey(voidIdem).
			SetRequestFingerprint("DEV-VOID-FP-SE26090017-HBL").
			SetConfirmedByParty("TransGlobal Trading (HK) Co., Limited").
			SetConfirmedAt(time.Now()).
			SetConfirmationNote("客户邮件确认作废并授权按新抬头重出").
			Save(ctx); err != nil {
			return fmt.Errorf("创建分单作废事件: %w", err)
		}
		_ = v1
	}

	// —— 单证模式切换：SE26090018 直单转主分单（DIRECT → HOUSE） ——
	ord18 := sc.orders["SE26090018"]
	if ord18 == nil {
		return fmt.Errorf("缺少订单 SE26090018")
	}
	modeIdem := "DEV-MODE-SE26090018"
	modeExists, _ := tx.SeaDocumentModeChangeEvent.Query().Where(
		seadocumentmodechangeeventent.OrganizationIDEQ(co.ID),
		seadocumentmodechangeeventent.IdempotencyKeyEQ(modeIdem),
	).Exist(ctx)
	if !modeExists {
		hbl18, err := tx.SeaHouseBill.Query().Where(
			seahousebillent.OrderIDEQ(ord18.ID),
			seahousebillent.HouseNoEQ("RC-HBL26090018"),
		).First(ctx)
		if err != nil {
			return fmt.Errorf("查询分单 RC-HBL26090018: %w", err)
		}
		v18, err := tx.SeaHouseBillVersion.Create().
			SetOrganizationID(co.ID).
			SetHouseBillID(hbl18.ID).
			SetOrderID(ord18.ID).
			SetMasterBillID(hbl18.MasterBillID).
			SetVersionNo(1).
			SetSourceEntityVersion(hbl18.Version).
			SetHouseNo(hbl18.HouseNo).
			SetNormalizedHouseNo(hbl18.NormalizedHouseNo).
			SetIssuerSource(seahousebillversionent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillversionent.StatusCONFIRMED).
			SetContentHash("devseed-hbl-RC-HBL26090018-v1").
			SetSource(seahousebillversionent.SourceMODE_CHANGE).
			SetReason("直单转主分单，客户要求签发自有分单").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建模式切换分单版本: %w", err)
		}
		if _, err := hbl18.Update().SetCurrentVersionID(v18.ID).Save(ctx); err != nil {
			return fmt.Errorf("推进模式切换分单当前版本: %w", err)
		}

		if _, err := tx.SeaDocumentModeChangeEvent.Create().
			SetOrganizationID(co.ID).
			SetOrderID(ord18.ID).
			SetPreviousMode(seadocumentmodechangeeventent.PreviousModeDIRECT).
			SetTargetMode(seadocumentmodechangeeventent.TargetModeHOUSE).
			SetTargetHouseBillID(hbl18.ID).
			SetTargetHouseBillVersionID(v18.ID).
			SetReason("客户要求以我司分单抬头清关，直单转主分单结构").
			SetImpactSummary("主单收货人改为我司目的港代理，箱货数据不变").
			SetConfirmedByParty("上海宏远国际贸易进出口有限公司").
			SetConfirmedAt(time.Now()).
			SetConfirmationNote("客户书面委托确认切换单证结构").
			SetCreatedBy(sc.adminUser.ID).
			SetIdempotencyKey(modeIdem).
			SetRequestFingerprint("DEV-MODE-FP-SE26090018").
			Save(ctx); err != nil {
			return fmt.Errorf("创建单证模式切换事件: %w", err)
		}
	}

	// —— HBL 改单版本历史：SE26090008 分单修改通知人 ——
	ord08 := sc.orders["SE26090008"]
	if ord08 == nil {
		return fmt.Errorf("缺少订单 SE26090008")
	}
	hbl08, err := tx.SeaHouseBill.Query().Where(
		seahousebillent.OrderIDEQ(ord08.ID),
		seahousebillent.HouseNoEQ("RC-HBL26090008"),
	).First(ctx)
	if err != nil {
		return fmt.Errorf("查询分单 RC-HBL26090008: %w", err)
	}
	if hbl08.CurrentVersionID == nil {
		v08, err := tx.SeaHouseBillVersion.Create().
			SetOrganizationID(co.ID).
			SetHouseBillID(hbl08.ID).
			SetOrderID(ord08.ID).
			SetMasterBillID(hbl08.MasterBillID).
			SetVersionNo(1).
			SetSourceEntityVersion(hbl08.Version).
			SetHouseNo(hbl08.HouseNo).
			SetNormalizedHouseNo(hbl08.NormalizedHouseNo).
			SetIssuerSource(seahousebillversionent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(co.ID).
			SetStatus(seahousebillversionent.StatusCONFIRMED).
			SetContentHash("devseed-hbl-RC-HBL26090008-v1").
			SetSource(seahousebillversionent.SourceAMENDMENT).
			SetReason("按目的港代理要求补充通知人税号").
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建分单改单版本: %w", err)
		}
		if _, err := hbl08.Update().SetCurrentVersionID(v08.ID).Save(ctx); err != nil {
			return fmt.Errorf("推进分单当前版本: %w", err)
		}
	}
	return nil
}

// 11. 订单运营数据：异常标记、里程碑、放单 POD 与附件登记
func seedOrderOperations(ctx context.Context, sc *seedContext) error {
	tx := sc.tx
	co := sc.company
	if sc.adminUser == nil {
		return fmt.Errorf("缺少管理员用户，无法注入订单运营数据")
	}
	liming := sc.users["liming"]
	zhaofang := sc.users["zhaofang"]
	chenhua := sc.users["chenhua"]

	// 异常目录（全局主数据）
	abnormalDefs := []struct{ code, name string }{
		{"ABN-CUSTOMER-CANCEL", "客户取消出运"},
		{"ABN-ROLL-REASSIGN", "甩柜改配"},
	}
	for _, d := range abnormalDefs {
		found, _ := tx.MasterDataItem.Query().Where(
			masterdataitem.KindEQ(masterdataitem.KindAbnormalCase),
			masterdataitem.CodeEQ(d.code),
		).First(ctx)
		if found == nil {
			created, err := tx.MasterDataItem.Create().
				SetKind(masterdataitem.KindAbnormalCase).
				SetCode(d.code).
				SetName(d.name).
				SetSource("system").
				SetSortOrder(10).
				SetEnabled(true).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("创建异常目录 %s: %w", d.code, err)
			}
			found = created
		}
		if d.code == "ABN-CUSTOMER-CANCEL" {
			// SE26090013 终止票上保留一条未解决异常
			ord13 := sc.orders["SE26090013"]
			mExists, _ := tx.OrderAbnormalCase.Query().Where(
				orderabnormalcaseent.OrderIDEQ(ord13.ID),
				orderabnormalcaseent.AbnormalCaseIDEQ(found.ID),
			).Exist(ctx)
			if !mExists {
				if _, err := tx.OrderAbnormalCase.Create().
					SetOrderID(ord13.ID).
					SetAbnormalCaseID(found.ID).
					SetStatus(orderabnormalcaseent.StatusACTIVE).
					SetMarkedBy(liming.ID).
					Save(ctx); err != nil {
					return fmt.Errorf("标记订单异常 SE26090013: %w", err)
				}
			}
		}
		if d.code == "ABN-ROLL-REASSIGN" {
			// SE26090004 上一条已解决异常（甩柜后改配恢复）
			ord04 := sc.orders["SE26090004"]
			mExists, _ := tx.OrderAbnormalCase.Query().Where(
				orderabnormalcaseent.OrderIDEQ(ord04.ID),
				orderabnormalcaseent.AbnormalCaseIDEQ(found.ID),
			).Exist(ctx)
			if !mExists {
				if _, err := tx.OrderAbnormalCase.Create().
					SetOrderID(ord04.ID).
					SetAbnormalCaseID(found.ID).
					SetStatus(orderabnormalcaseent.StatusRESOLVED).
					SetMarkedBy(liming.ID).
					SetResolvedAt(time.Now()).
					SetResolvedBy(zhaofang.ID).
					Save(ctx); err != nil {
					return fmt.Errorf("标记订单异常 SE26090004: %w", err)
				}
			}
		}
	}

	// 里程碑：SE26090001 订舱确认 + 装船离港
	ord01 := sc.orders["SE26090001"]
	milestones := []struct {
		typ, label string
		occurred   *time.Time
		note       string
	}{
		{"BOOKING_CONFIRMED", "订舱确认", ptrTime(time.Now().Add(-72 * time.Hour)), "中远海运舱位确认，S/O 已回传"},
		{"VESSEL_DEPARTED", "装船离港", nil, "待船舶实际开航后补录"},
	}
	for _, m := range milestones {
		mExists, _ := tx.OrderMilestone.Query().Where(
			ordermilestoneent.OrderIDEQ(ord01.ID),
			ordermilestoneent.TypeEQ(m.typ),
		).Exist(ctx)
		if !mExists {
			create := tx.OrderMilestone.Create().
				SetOrderID(ord01.ID).
				SetType(m.typ).
				SetTemplateNodeLabel(m.label).
				SetNote(m.note).
				SetUpdatedBy(sc.adminUser.ID)
			if m.occurred != nil {
				create = create.SetOccurredAt(*m.occurred)
			}
			if _, err := create.Save(ctx); err != nil {
				return fmt.Errorf("创建订单里程碑 %s: %w", m.typ, err)
			}
		}
	}

	// 放单 POD：SE26090002 待签收；SE26090014 已签收
	podDefs := []struct {
		orderNo, releaseNo, podNo string
		signed                    bool
	}{
		{"SE26090002", "REL-MSCU67812300", "", false},
		{"SE26090014", "REL-MSCU10102020", "POD-26090014", true},
	}
	for _, p := range podDefs {
		ord := sc.orders[p.orderNo]
		pExists, _ := tx.OrderReleasePod.Query().Where(
			orderreleasepodent.OrderIDEQ(ord.ID),
			orderreleasepodent.ReleaseNoEQ(p.releaseNo),
		).Exist(ctx)
		if pExists {
			continue
		}
		link, _ := tx.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(ord.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).First(ctx)
		create := tx.OrderReleasePod.Create().
			SetOrderID(ord.ID).
			SetSeaMasterBillID(link.MasterBillID).
			SetReleaseNo(p.releaseNo).
			SetStatus(orderreleasepodent.StatusPENDING)
		if p.signed {
			create = create.
				SetPodNo(p.podNo).
				SetStatus(orderreleasepodent.StatusSIGNED).
				SetSignedAt(time.Now()).
				SetSignedBy(chenhua.ID)
		}
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("创建放单 POD %s: %w", p.releaseNo, err)
		}
	}

	// 附件登记：SE26090001 订舱确认书（对象存储占位 key，用于列表与登记形态测试）
	ord01Again := sc.orders["SE26090001"]
	attIdem := "DEV-ATT-SE26090001-BOOKING"
	attExists, _ := tx.OrderAttachment.Query().Where(
		orderattachmentent.OrderIDEQ(ord01Again.ID),
		orderattachmentent.IdempotencyKeyEQ(attIdem),
	).Exist(ctx)
	if !attExists {
		asset, err := tx.OrderAttachmentAsset.Create().
			SetOrganizationID(co.ID).
			SetObjectKey("dev-seed/se26090001/booking-confirmation.pdf").
			SetFileName("订舱确认书-COSCO PRIDE 042W.pdf").
			SetMimeType("application/pdf").
			SetFileSize(132688).
			SetUploadedBy(sc.adminUser.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建附件资产: %w", err)
		}
		if _, err := tx.OrderAttachment.Create().
			SetOrderID(ord01Again.ID).
			SetAssetID(asset.ID).
			SetDocType("BOOKING_CONFIRMATION").
			SetIdempotencyKey(attIdem).
			SetCreatedBy(sc.adminUser.ID).
			Save(ctx); err != nil {
			return fmt.Errorf("创建订单附件登记: %w", err)
		}
	}
	return nil
}

func ptrTime(t time.Time) *time.Time { return &t }

func strOr(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}

func intOr(i *int, fallback int) int {
	if i != nil {
		return *i
	}
	return fallback
}

func floatOr(f *float64, fallback float64) float64 {
	if f != nil {
		return *f
	}
	return fallback
}

func uuidOr(u *uuid.UUID, fallback uuid.UUID) uuid.UUID {
	if u != nil {
		return *u
	}
	return fallback
}
