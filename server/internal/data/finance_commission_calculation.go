package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type commissionCalculationStore struct {
	verifications *ent.FinanceVerificationClient
	nettings      *ent.FinanceNettingClient
	rules         *ent.FinanceCommissionRuleClient
	users         *ent.UserClient
	bills         *ent.FinanceBillClient
	billLines     *ent.FinanceBillLineClient
	attributions  *ent.OrderCommissionAttributionClient
	fees          *ent.OrderFeeClient
	orders        *ent.OrderClient
}

func commissionStoreFromClient(client *ent.Client) commissionCalculationStore {
	return commissionCalculationStore{verifications: client.FinanceVerification, nettings: client.FinanceNetting, rules: client.FinanceCommissionRule, users: client.User, bills: client.FinanceBill, billLines: client.FinanceBillLine, attributions: client.OrderCommissionAttribution, fees: client.OrderFee, orders: client.Order}
}

func commissionStoreFromTx(tx *ent.Tx) commissionCalculationStore {
	return commissionCalculationStore{verifications: tx.FinanceVerification, nettings: tx.FinanceNetting, rules: tx.FinanceCommissionRule, users: tx.User, bills: tx.FinanceBill, billLines: tx.FinanceBillLine, attributions: tx.OrderCommissionAttribution, fees: tx.OrderFee, orders: tx.Order}
}

func commissionCalculationBillsQuery(store commissionCalculationStore, org uuid.UUID, billIDs []uuid.UUID, lock bool) *ent.FinanceBillQuery {
	bq := store.bills.Query().
		Where(
			bill.IDIn(billIDs...),
			bill.OrganizationIDEQ(org),
			bill.StatusEQ(bill.StatusCONFIRMED),
			bill.DirectionEQ(bill.DirectionRECEIVABLE),
		).
		WithLines().
		Order(bill.ByID())
	if lock {
		bq.ForUpdate()
	}
	return bq
}

func (r *commissionRepo) Preview(ctx context.Context, org, verificationID, nettingID, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole) (*biz.CommissionCalculation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommission(ctx, commissionStoreFromClient(client), org, verificationID, nettingID, employeeID, personnelRole, false)
}

// resolveCommissionRuleForDate 按来源归属日期为「组织 + 员工 + 人员身份」解析
// 唯一有效方案与员工分配：方案必须已启用且归属日期落在方案区间内，且该员工
// 存在未取消分配、归属日期同样落在分配区间内。零命中或多命中（并发或数据
// 异常）均返回 ErrCommissionRuleNotResolved；迁移前停用的旧规则因 enabled=false
// 自然被排除。lock=true 时按方案主键升序 ForUpdate，与方案写入共享串行化点。
func resolveCommissionRuleForDate(ctx context.Context, store commissionCalculationStore, org, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole, commissionDate string, lock bool) (*ent.FinanceCommissionRule, *ent.FinanceCommissionRuleAssignment, error) {
	if commissionDate == "" {
		return nil, nil, biz.ErrCommissionRuleNotResolved
	}
	q := store.rules.Query().Where(
		rule.OrganizationIDEQ(org),
		rule.EnabledEQ(true),
		rule.PersonnelRoleEQ(rule.PersonnelRole(personnelRole)),
	).WithAssignments(func(aq *ent.FinanceCommissionRuleAssignmentQuery) {
		aq.Where(assignment.EmployeeIDEQ(employeeID), assignment.CancelledAtIsNil())
	}).Order(rule.ByID())
	if lock {
		q.ForUpdate()
	}
	rules, err := q.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	var (
		hitRule       *ent.FinanceCommissionRule
		hitAssignment *ent.FinanceCommissionRuleAssignment
	)
	for _, ruleItem := range rules {
		if (ruleItem.EffectiveFrom != nil && commissionDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && commissionDate > *ruleItem.EffectiveTo) {
			continue
		}
		for _, seg := range ruleItem.Edges.Assignments {
			if commissionDate < seg.EffectiveFrom || (seg.EffectiveTo != nil && commissionDate > *seg.EffectiveTo) {
				continue
			}
			if hitRule != nil {
				// 同员工同身份在同一归属日期命中多个方案：数据异常或并发窗口，
				// 拒绝生成而不是静默选择。
				return nil, nil, biz.ErrCommissionRuleNotResolved
			}
			hitRule, hitAssignment = ruleItem, seg
		}
	}
	if hitRule == nil {
		return nil, nil, biz.ErrCommissionRuleNotResolved
	}
	return hitRule, hitAssignment, nil
}

// GetGenerationContext 读取生成提成所需的来源上下文：归属日期和本位币。
// 核销来源的归属日期为 verification_date；对冲来源为确认日（confirmed_at 的 UTC 日期）。
// CNY 折算汇率按归属日期解析，来源单不再携带汇率快照。
// 事务内首次读取即加 ForUpdate：同一事务内的写入阶段还会对同一来源行 ForUpdate，
// 若先 ForShare 再升级，两个并发创建事务可同持共享锁互等升级形成死锁，因此从入口
// 串行化；普通上下文保持无锁读取。
func (r *commissionRepo) GetGenerationContext(ctx context.Context, org, verificationID, nettingID uuid.UUID) (*biz.CommissionGenerationContext, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if nettingID != uuid.Nil {
		query := client.FinanceNetting.Query().Where(nettingent.IDEQ(nettingID), nettingent.OrganizationIDEQ(org))
		if _, transactional := transactionFromContext(ctx); transactional {
			query.ForUpdate()
		}
		item, queryErr := query.Only(ctx)
		if queryErr != nil {
			return nil, mapEntError(queryErr, biz.ErrCommissionSource, nil)
		}
		if item.Status != nettingent.StatusCONFIRMED || item.ConfirmedAt == nil {
			return nil, biz.ErrCommissionSource
		}
		return &biz.CommissionGenerationContext{CommissionDate: item.ConfirmedAt.UTC().Format("2006-01-02"), BaseCurrency: item.BaseCurrency}, nil
	}
	query := client.FinanceVerification.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	v, err := query.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionSource, nil)
	}
	return &biz.CommissionGenerationContext{CommissionDate: v.VerificationDate, BaseCurrency: v.BaseCurrency}, nil
}

// commissionCalculationSource 是提成计算的来源快照：核销或对冲二选一。
// 两种来源同构地按「分摊金额 / 账单总额 × 账单行本位币」摊入 orderRealized。
// 方案与员工分配不在此结构内：由调用方按「员工 + 人员身份 + 归属日期」解析。
type commissionCalculationSource struct {
	organizationID    uuid.UUID
	verification      *ent.FinanceVerification // 核销来源时非空
	netting           *ent.FinanceNetting      // 对冲来源时非空
	commissionDate    string                   // 归属日期：核销 verification_date / 对冲确认日
	baseCurrency      string
	orderIDs          []uuid.UUID
	orderRealized     map[uuid.UUID]decimal.Decimal
	orderByID         map[uuid.UUID]*ent.Order
	feesByOrder       map[uuid.UUID][]*ent.OrderFee
	billLineBaseByFee map[uuid.UUID]decimal.Decimal
	fingerprintBase   []string
}

// commissionSourceAllocation 是核销/对冲分摊的统一条目，供下游同一聚合管线消费。
type commissionSourceAllocation struct {
	billID      uuid.UUID
	amount      decimal.Decimal
	fingerprint string
}

func calculateCommission(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole, lock bool) (*biz.CommissionCalculation, error) {
	source, err := loadCommissionCalculationSource(ctx, store, org, verificationID, nettingID, lock)
	if err != nil {
		return nil, err
	}
	uq := store.users.Query().Where(user.IDEQ(employeeID))
	if lock {
		uq.ForUpdate()
	}
	employee, err := uq.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionInvalid, nil)
	}
	// 员工已离职（账号停用、成员关系解除）不影响解析：历史资格只由来源日期、
	// 方案区间与分配区间决定，不以当前账号状态抹除。
	ruleItem, assignmentItem, err := resolveCommissionRuleForDate(ctx, store, org, employeeID, personnelRole, source.commissionDate, lock)
	if err != nil {
		return nil, err
	}
	aq := store.attributions.Query().Where(
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.EmployeeIDEQ(employeeID),
		attribution.PersonnelRoleEQ(attribution.PersonnelRole(personnelRole)),
	).Order(attribution.ByAttributedAt(), attribution.ByID())
	if lock {
		aq.ForUpdate()
	}
	attributions, err := aq.All(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommissionFromSource(source, ruleItem, assignmentItem, employee, attributions)
}

// loadCommissionCalculationSource 按来源（核销 ACTIVE+RECEIVABLE / 对冲 CONFIRMED 的
// RECEIVABLE 分摊）加载统一分摊条目，并锁定账单、订单、费用与账单行事实。
// 多行加锁一律按主键稳定排序，固定加锁顺序防止并发提成创建死锁。
func loadCommissionCalculationSource(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID uuid.UUID, lock bool) (*commissionCalculationSource, error) {
	var (
		fingerprintParts   []string
		commissionDate     string
		allocationEntries  []commissionSourceAllocation
		verificationEntity *ent.FinanceVerification
		nettingEntity      *ent.FinanceNetting
	)
	if nettingID != uuid.Nil {
		nq := store.nettings.Query().Where(nettingent.IDEQ(nettingID), nettingent.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceNettingAllocationQuery) {
			q.Where(nettingalloc.ActiveEQ(true))
		})
		if lock {
			nq.ForUpdate()
		}
		item, err := nq.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrCommissionSource, nil)
		}
		if item.Status != nettingent.StatusCONFIRMED || item.ConfirmedAt == nil {
			return nil, biz.ErrCommissionSource
		}
		nettingEntity = item
		commissionDate = item.ConfirmedAt.UTC().Format("2006-01-02")
		fingerprintParts = append(fingerprintParts,
			fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
			fmt.Sprintf("src=netting|%s", item.ID),
			fmt.Sprintf("netting|%s|%s|%s|%s|%d", item.ID, item.NettingNo, item.Status, commissionDate, item.Version),
		)
		for _, allocationItem := range item.Edges.Allocations {
			if allocationItem.Direction != nettingalloc.DirectionRECEIVABLE {
				continue
			}
			amount, parseErr := decimalOf(allocationItem.Amount)
			if parseErr != nil {
				return nil, parseErr
			}
			allocationEntries = append(allocationEntries, commissionSourceAllocation{
				billID:      allocationItem.BillID,
				amount:      amount,
				fingerprint: fmt.Sprintf("netting_allocation|%s|%s|%s|%s|%t", allocationItem.ID, allocationItem.BillID, allocationItem.Direction, allocationItem.Amount, allocationItem.Active),
			})
		}
	} else {
		vq := store.verifications.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceVerificationAllocationQuery) {
			q.Where(allocation.ActiveEQ(true))
		})
		if lock {
			vq.ForUpdate()
		}
		v, err := vq.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrCommissionSource, nil)
		}
		if v.Status != verification.StatusACTIVE || v.Direction != verification.DirectionRECEIVABLE {
			return nil, biz.ErrCommissionSource
		}
		verificationEntity = v
		commissionDate = v.VerificationDate
		fingerprintParts = append(fingerprintParts,
			fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
			fmt.Sprintf("src=verification|%s", v.ID),
			fmt.Sprintf("verification|%s|%s|%s|%s|%s|%d", v.ID, v.VerificationNo, v.Status, v.Direction, v.VerificationDate, v.Version),
		)
		for _, item := range v.Edges.Allocations {
			amount, parseErr := decimalOf(item.Amount)
			if parseErr != nil {
				return nil, parseErr
			}
			allocationEntries = append(allocationEntries, commissionSourceAllocation{
				billID:      item.BillID,
				amount:      amount,
				fingerprint: fmt.Sprintf("allocation|%s|%s|%s|%s|%t", item.ID, item.BillID, item.CashflowID, item.Amount, item.Active),
			})
		}
	}
	if len(allocationEntries) == 0 {
		return nil, biz.ErrCommissionSource
	}
	billIDs := make([]uuid.UUID, 0, len(allocationEntries))
	allocationByBill := make(map[uuid.UUID]decimal.Decimal)
	seenBills := make(map[uuid.UUID]struct{})
	for _, item := range allocationEntries {
		if _, exists := seenBills[item.billID]; !exists {
			seenBills[item.billID] = struct{}{}
			billIDs = append(billIDs, item.billID)
		}
		allocationByBill[item.billID] = allocationByBill[item.billID].Add(item.amount)
		fingerprintParts = append(fingerprintParts, item.fingerprint)
	}
	bq := commissionCalculationBillsQuery(store, org, billIDs, lock)
	bills, err := bq.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(bills) != len(billIDs) {
		return nil, biz.ErrCommissionSource
	}
	orderRealized := make(map[uuid.UUID]decimal.Decimal)
	baseCurrency := ""
	for _, billItem := range bills {
		total, parseErr := decimalOf(billItem.TotalAmount)
		if parseErr != nil || !total.IsPositive() {
			return nil, biz.ErrCommissionSource
		}
		ratio := allocationByBill[billItem.ID].Div(total)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill|%s|%s|%s|%s|%s|%d", billItem.ID, billItem.BillNo, billItem.Status, billItem.TotalAmount, billItem.BaseCurrencyAmount, billItem.Version))
		for _, billLine := range billItem.Edges.Lines {
			if !billLine.Active {
				continue
			}
			base, parseErr := decimalOf(billLine.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if baseCurrency == "" {
				baseCurrency = billLine.BaseCurrency
			} else if baseCurrency != billLine.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			orderRealized[billLine.OrderID] = orderRealized[billLine.OrderID].Add(base.Mul(ratio))
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill_line|%s|%s|%s|%s|%s|%t", billLine.ID, billLine.OrderFeeID, billLine.OrderID, billLine.BaseCurrencyAmount, billLine.BaseCurrency, billLine.Active))
		}
	}
	orderIDs := make([]uuid.UUID, 0, len(orderRealized))
	for id := range orderRealized {
		orderIDs = append(orderIDs, id)
	}
	if len(orderIDs) == 0 || baseCurrency == "" {
		return nil, biz.ErrCommissionSource
	}
	sort.Slice(orderIDs, func(i, j int) bool { return orderIDs[i].String() < orderIDs[j].String() })
	oq := store.orders.Query().Where(orderent.IDIn(orderIDs...), orderent.OrganizationIDEQ(org)).WithCustomer().Order(orderent.ByID())
	if lock {
		oq.ForUpdate()
	}
	orders, err := oq.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) != len(orderIDs) {
		return nil, biz.ErrCommissionSource
	}
	orderByID := make(map[uuid.UUID]*ent.Order, len(orders))
	for _, item := range orders {
		orderByID[item.ID] = item
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("order|%s|%s|%s|%s|%d", item.ID, item.OrderNo, item.CustomerID, item.OrderDate, item.Version))
	}
	fq := store.fees.Query().Where(fee.OrderIDIn(orderIDs...), fee.StatusIn(fee.StatusUNBILLED, fee.StatusBILLED)).WithSettlementParty().Order(fee.ByOrderID(), fee.ByCreatedAt(), fee.ByID())
	if lock {
		fq.ForUpdate()
	}
	fees, err := fq.All(ctx)
	if err != nil {
		return nil, err
	}
	feesByOrder := make(map[uuid.UUID][]*ent.OrderFee)
	for _, item := range fees {
		feesByOrder[item.OrderID] = append(feesByOrder[item.OrderID], item)
	}
	// B3：分母聚合拉齐分子口径。分子（已实现收入）按账单行本位币快照计算，
	// 分母（订单总应收/总应付）改用各费用关联账单行的本位币合计；尚未建账的费用
	// 没有账单日快照，回落到费用自身本位币快照，保证分母仍覆盖订单全部费用。
	feeIDs := make([]uuid.UUID, 0, len(fees))
	for _, item := range fees {
		feeIDs = append(feeIDs, item.ID)
	}
	billLineQuery := store.billLines.Query().Where(billline.OrderFeeIDIn(feeIDs...), billline.ActiveEQ(true)).
		// 多行加锁前先按主键排序，固定加锁顺序防止并发提成创建在重叠账单行上死锁。
		Order(billline.ByID())
	if lock {
		billLineQuery.ForUpdate()
	}
	feeBillLines, err := billLineQuery.All(ctx)
	if err != nil {
		return nil, err
	}
	billLineBaseByFee := make(map[uuid.UUID]decimal.Decimal, len(feeBillLines))
	for _, line := range feeBillLines {
		base, parseErr := decimalOf(line.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		billLineBaseByFee[line.OrderFeeID] = billLineBaseByFee[line.OrderFeeID].Add(base)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("fee_bill_line|%s|%s|%s|%t", line.ID, line.OrderFeeID, line.BaseCurrencyAmount, line.Active))
	}
	return &commissionCalculationSource{
		organizationID: org, verification: verificationEntity, netting: nettingEntity, commissionDate: commissionDate,
		baseCurrency: baseCurrency,
		orderIDs:     orderIDs, orderRealized: orderRealized, orderByID: orderByID, feesByOrder: feesByOrder,
		billLineBaseByFee: billLineBaseByFee,
		fingerprintBase:   fingerprintParts,
	}, nil
}

// calculateCommissionFromSource 基于来源快照、已解析的方案与员工分配、员工及其
// 提成归属逐订单计算提成。指纹只覆盖影响来源日期计算资格的稳定标识：方案参数
// （身份、口径、比例、起始日）与分配段身份。方案版本、终止日与名单后续变更
// （未来生效）不改变来源日期的资格，不参与指纹，保证历史快照确认不被重算。
func calculateCommissionFromSource(source *commissionCalculationSource, ruleItem *ent.FinanceCommissionRule, assignmentItem *ent.FinanceCommissionRuleAssignment, employee *ent.User, attributions []*ent.OrderCommissionAttribution) (*biz.CommissionCalculation, error) {
	if ruleItem == nil || assignmentItem == nil || !ruleItem.Enabled {
		return nil, biz.ErrCommissionRuleNotResolved
	}
	rate, err := decimalOf(ruleItem.RatePercent)
	if err != nil {
		return nil, err
	}
	fingerprintParts := append([]string(nil), source.fingerprintBase...)
	fingerprintParts = append(fingerprintParts,
		fmt.Sprintf("rule|%s|%s|%s|%s|%s", ruleItem.ID, ruleItem.PersonnelRole, ruleItem.CalculationBasis, ruleItem.RatePercent, optionalStringValue(ruleItem.EffectiveFrom)),
		fmt.Sprintf("rule_assignment|%s|%s|%s", assignmentItem.ID, assignmentItem.EmployeeID, assignmentItem.EffectiveFrom),
		fmt.Sprintf("employee|%s|%s|%t", employee.ID, employee.DisplayName, employee.Enabled),
	)
	attributionByOrder := make(map[uuid.UUID]*ent.OrderCommissionAttribution, len(attributions))
	for _, item := range attributions {
		attributionByOrder[item.OrderID] = item
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("order_attribution|%s|%s|%s|%s|%s|%s|%s|%s", item.ID, item.OrderID, item.CustomerID, item.EmployeeID, item.PersonnelRole, item.SourceAssignmentID, item.EmployeeName, item.AttributedAt.UTC().Format(time.RFC3339Nano)))
	}
	eligibleOrderIDs := make([]uuid.UUID, 0, len(source.orderIDs))
	for _, id := range source.orderIDs {
		if _, eligible := attributionByOrder[id]; eligible {
			eligibleOrderIDs = append(eligibleOrderIDs, id)
		}
	}
	if len(eligibleOrderIDs) == 0 {
		return nil, biz.ErrCommissionEmployeeRole
	}
	result := &biz.CommissionCalculation{
		EmployeeID: employee.ID, EmployeeName: attributions[0].EmployeeName,
		RuleID: ruleItem.ID, RuleName: ruleItem.Name, PersonnelRole: biz.CommissionPersonnelRole(ruleItem.PersonnelRole),
		CalculationBasis: biz.CommissionCalculationBasis(ruleItem.CalculationBasis), RuleVersion: ruleItem.Version,
		CalculationVersion: biz.CommissionCalculationVersion, BaseCurrency: source.baseCurrency, RatePercent: rate,
		Lines: make([]*biz.FinanceCommissionLine, 0, len(eligibleOrderIDs)),
	}
	if source.verification != nil {
		result.VerificationID = source.verification.ID
		result.VerificationNo = source.verification.VerificationNo
	}
	if source.netting != nil {
		result.NettingID = source.netting.ID
		result.NettingNo = source.netting.NettingNo
	}
	customersWithFees := make(map[uuid.UUID]struct{})
	for _, orderID := range eligibleOrderIDs {
		orderItem := source.orderByID[orderID]
		orderFees := source.feesByOrder[orderItem.ID]
		customer, edgeErr := orderItem.Edges.CustomerOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		attributionItem := attributionByOrder[orderItem.ID]
		line := &biz.FinanceCommissionLine{
			OrderID: orderItem.ID, OrderNo: orderItem.OrderNo, OrderDate: orderItem.OrderDate,
			CustomerID: customer.ID, CustomerCode: partnerCodeValue(customer.Code), CustomerName: customer.LegalName,
			CustomerAssignmentID: attributionItem.SourceAssignmentID, CustomerAssignmentOrganizationID: attributionItem.OrganizationID, CustomerAssignedAt: attributionItem.AttributedAt,
			EmployeeID: employee.ID, EmployeeName: attributionItem.EmployeeName, PersonnelRole: result.PersonnelRole,
			CalculationBasis: result.CalculationBasis, RatePercent: rate, Fees: make([]*biz.CommissionFeeDetail, 0, len(orderFees)),
		}
		for _, feeItem := range orderFees {
			party, partyErr := feeItem.Edges.SettlementPartyOrErr()
			if partyErr != nil {
				return nil, partyErr
			}
			totalAmount, parseErr := decimalOf(feeItem.TotalAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			exchangeRate, parseErr := decimalOf(feeItem.ExchangeRate)
			if parseErr != nil {
				return nil, parseErr
			}
			baseAmount, parseErr := decimalOf(feeItem.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if result.BaseCurrency != feeItem.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			line.BaseCurrency = result.BaseCurrency
			// 总应收/总应付分母优先使用关联账单行的本位币快照（账单日口径），
			// 未建账费用回落到费用自身快照，避免分子分母汇率基准混用。
			aggregateBase := baseAmount
			if billedBase, billed := source.billLineBaseByFee[feeItem.ID]; billed {
				aggregateBase = billedBase
			}
			if feeItem.Direction == fee.DirectionRECEIVABLE {
				line.RealizedRevenue = line.RealizedRevenue.Add(aggregateBase)
			} else {
				line.AllocatedCost = line.AllocatedCost.Add(aggregateBase)
			}
			line.Fees = append(line.Fees, &biz.CommissionFeeDetail{
				FeeID: feeItem.ID, SettlementPartyID: feeItem.SettlementPartyID, Direction: string(feeItem.Direction),
				FeeCode: feeItem.FeeCode, FeeName: feeItem.FeeName, SettlementPartyName: party.LegalName,
				Currency: feeItem.Currency, TotalAmount: totalAmount, ExchangeRate: exchangeRate,
				BaseCurrency: feeItem.BaseCurrency, BaseCurrencyAmount: baseAmount, ExpenseDate: feeItem.ExpenseDate, Status: string(feeItem.Status),
			})
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("fee|%s|%s|%s|%s|%s|%s|%s|%s|%d", feeItem.ID, feeItem.OrderID, feeItem.Direction, feeItem.Status, feeItem.TotalAmount, feeItem.ExchangeRate, feeItem.BaseCurrencyAmount, feeItem.BaseCurrency, feeItem.Version))
		}
		line.FeeCount = len(line.Fees)
		totalReceivable := line.RealizedRevenue.Round(8)
		totalPayable := line.AllocatedCost.Round(8)
		realized := source.orderRealized[orderID].Round(8)
		cost, profit, commissionBase, amount, calculateErr := biz.CalculateCommissionLine(realized, totalReceivable, totalPayable, rate, result.CalculationBasis)
		if calculateErr != nil {
			return nil, calculateErr
		}
		line.RealizedRevenue, line.AllocatedCost, line.RealizedProfit = realized, cost, profit
		// 历史分母快照：新生成提成行固定写 READY + NATIVE，作为锁后费用补录
		// 审批的边际复算输入；分母为该行确认时冻结的历史总应收/总应付。
		line.TotalReceivableSnapshot, line.TotalPayableSnapshot = totalReceivable, totalPayable
		line.SnapshotStatus, line.SnapshotSource = biz.CommissionLineSnapshotReady, biz.CommissionLineSnapshotNative
		line.CommissionBaseAmount, line.CommissionAmount = commissionBase, amount
		result.Lines = append(result.Lines, line)
		customersWithFees[orderItem.CustomerID] = struct{}{}
		result.RealizedRevenue = result.RealizedRevenue.Add(realized)
		result.AllocatedCost = result.AllocatedCost.Add(cost)
		result.RealizedProfit = result.RealizedProfit.Add(profit)
		result.CommissionBaseAmount = result.CommissionBaseAmount.Add(commissionBase)
		result.CommissionAmount = result.CommissionAmount.Add(amount)
		result.FeeCount += line.FeeCount
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("result_line|%s|%s|%s|%s|%s", orderItem.ID, realized.StringFixed(8), cost.StringFixed(8), profit.StringFixed(8), amount.StringFixed(8)))
	}
	if len(result.Lines) == 0 || !result.RealizedRevenue.IsPositive() {
		return nil, biz.ErrCommissionSource
	}
	result.CustomerCount, result.OrderCount = len(customersWithFees), len(result.Lines)
	result.RealizedRevenue = result.RealizedRevenue.Round(8)
	result.AllocatedCost = result.AllocatedCost.Round(8)
	result.RealizedProfit = result.RealizedProfit.Round(8)
	result.CommissionBaseAmount = result.CommissionBaseAmount.Round(8)
	result.CommissionAmount = result.CommissionAmount.Round(8)
	sort.Strings(fingerprintParts)
	digest := sha256.Sum256([]byte(strings.Join(fingerprintParts, "\n")))
	result.SourceFingerprint = hex.EncodeToString(digest[:])
	return result, nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
