package biz

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

func (uc *OrderUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*Order, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	order, err := uc.repo.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if err := attachOrderTags(ctx, uc.tagRepo, order); err != nil {
		return nil, err
	}
	if err := attachSeaMasterBillSummaries(ctx, uc.seaMasterBillRepo, organizationID, order); err != nil {
		return nil, err
	}
	if err := attachSeaDocumentSummaries(ctx, uc.seaDocumentRepo, organizationID, order); err != nil {
		return nil, err
	}
	return order, nil
}

func attachOrderTags(ctx context.Context, tagRepo BusinessTagRepo, orders ...*Order) error {
	if tagRepo == nil || len(orders) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	tags, err := tagRepo.LoadOrderTags(ctx, ids)
	if err != nil {
		return err
	}
	for _, order := range orders {
		order.Tags = tags[order.ID]
	}
	return nil
}

// FindAuthorized 在仓储查询中同时限制订单 ID、业务类型和组织范围，供传输鉴权
// 定位订单锚点使用。它不加载详情扩展数据，避免鉴权路径扩大查询范围。
func (uc *OrderUsecase) FindAuthorized(ctx context.Context, id uuid.UUID, scopes []OrderOrganizationScope) (*Order, error) {
	if id == uuid.Nil || !validOrderOrganizationScopes(scopes) {
		return nil, ErrOrderNotFound
	}
	return uc.repo.FindAuthorized(ctx, id, scopes)
}

func (uc *OrderUsecase) List(ctx context.Context, scopes []OrderOrganizationScope, options OrderListOptions) (*OrderList, error) {
	if !validOrderOrganizationScopes(scopes) || !ValidListPagination(options.Page, options.PageSize) || options.BusinessType != "" && !options.BusinessType.Valid() || options.BusinessType == "" && len(options.BusinessTypes) == 0 {
		return nil, ErrOrderInvalidArgument
	}
	for _, businessType := range options.BusinessTypes {
		if !businessType.Valid() {
			return nil, ErrOrderInvalidArgument
		}
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	options.NumberKeyword = strings.TrimSpace(options.NumberKeyword)
	options.ConsigneeShortName = strings.TrimSpace(options.ConsigneeShortName)
	options.ShipperShortName = strings.TrimSpace(options.ShipperShortName)
	if options.NumberKeyword != "" && !options.NumberType.Valid() || options.NumberKeyword == "" && options.NumberType != "" {
		return nil, ErrOrderInvalidArgument
	}
	if options.FlowStatus != "" && !options.FlowStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if options.TerminationStatus != "" && !options.TerminationStatus.Valid() || options.ClosureStatus != "" && !options.ClosureStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	result, err := uc.repo.List(ctx, scopes, options)
	if err != nil {
		return nil, err
	}
	if err := attachOrderTags(ctx, uc.tagRepo, result.Items...); err != nil {
		return nil, err
	}
	for _, organizationID := range orderOrganizationIDs(scopes) {
		organizationOrders := make([]*Order, 0, len(result.Items))
		for _, order := range result.Items {
			if order.OrganizationID == organizationID {
				organizationOrders = append(organizationOrders, order)
			}
		}
		if err := attachSeaMasterBillSummaries(ctx, uc.seaMasterBillRepo, organizationID, organizationOrders...); err != nil {
			return nil, err
		}
		if err := attachSeaDocumentSummaries(ctx, uc.seaDocumentRepo, organizationID, organizationOrders...); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func validOrderOrganizationScopes(scopes []OrderOrganizationScope) bool {
	if len(scopes) == 0 {
		return false
	}
	for _, scope := range scopes {
		if !scope.BusinessType.Valid() || len(scope.OrganizationIDs) == 0 {
			return false
		}
		for _, organizationID := range scope.OrganizationIDs {
			if organizationID == uuid.Nil {
				return false
			}
		}
	}
	return true
}

func orderOrganizationIDs(scopes []OrderOrganizationScope) []uuid.UUID {
	unique := make(map[uuid.UUID]struct{})
	for _, scope := range scopes {
		for _, organizationID := range scope.OrganizationIDs {
			unique[organizationID] = struct{}{}
		}
	}
	return sortedOrganizationIDs(unique)
}

func attachSeaMasterBillSummaries(ctx context.Context, repo SeaMasterBillRepo, organizationID uuid.UUID, orders ...*Order) error {
	if repo == nil || organizationID == uuid.Nil || len(orders) == 0 {
		return nil
	}
	orderIDs := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
	}
	summaries, err := repo.GetSummariesByOrderIDs(ctx, organizationID, orderIDs)
	if err != nil {
		return err
	}
	for _, order := range orders {
		order.SeaMasterBill = summaries[order.ID]
	}
	return nil
}

func attachSeaDocumentSummaries(ctx context.Context, repo SeaDocumentRepo, organizationID uuid.UUID, orders ...*Order) error {
	if repo == nil || organizationID == uuid.Nil || len(orders) == 0 {
		return nil
	}
	orderIDs := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
	}
	summaries, err := repo.GetSummariesByOrderIDs(ctx, organizationID, orderIDs)
	if err != nil {
		return err
	}
	for _, order := range orders {
		if summary, ok := summaries[order.ID]; ok {
			order.SeaDocumentSummary = summary
			order.SeaDocumentStructure = &summary.DocumentStructure
			order.SeaDocumentLinkVersion = &summary.LinkVersion
		}
	}
	return nil
}

func (uc *OrderUsecase) CheckReference(ctx context.Context, organizationID uuid.UUID, check OrderReferenceCheck) (*OrderReferenceMatch, error) {
	check.ReferenceNo = strings.TrimSpace(check.ReferenceNo)
	if organizationID == uuid.Nil || !check.ReferenceType.Valid() || check.ReferenceNo == "" || utf8.RuneCountInString(check.ReferenceNo) > 100 {
		return nil, ErrOrderInvalidArgument
	}
	if check.ReferenceType == OrderReferenceCustomer && (check.CustomerID == nil || *check.CustomerID == uuid.Nil) {
		return nil, ErrOrderInvalidArgument
	}
	if check.ExcludeOrderID != nil && *check.ExcludeOrderID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.FindReferenceDuplicate(ctx, organizationID, check)
}

func (uc *OrderUsecase) ListPersonnelOptions(ctx context.Context, organizationID uuid.UUID, options SelectorListOptions) (*PagedList[*OrderPersonnelOption], error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrOrderInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListPersonnelOptions(ctx, organizationID, options)
}

func (uc *OrderUsecase) ListConsolidationSummaries(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderConsolidationSummary, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.ListConsolidationSummaries(ctx, organizationID, orderID)
}

func (uc *OrderUsecase) ListSameBatchOrders(ctx context.Context, organizationID, orderID uuid.UUID) ([]*SameBatchOrderSummary, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.ListSameBatchOrders(ctx, organizationID, orderID)
}

func (uc *OrderUsecase) MatchSeaMasterBillCandidate(ctx context.Context, organizationID, shippingLineID uuid.UUID, masterNo string, voyage *SeaTransportExecution) (*SeaMasterBillMatchResult, error) {
	if organizationID == uuid.Nil || shippingLineID == uuid.Nil {
		return nil, ErrSeaMasterBillInvalidArgument
	}
	normalizedNo, err := ValidateAndNormalizeSeaMasterNo(masterNo)
	if err != nil {
		return nil, err
	}
	return uc.seaMasterBillRepo.MatchCandidate(ctx, organizationID, shippingLineID, normalizedNo, voyage)
}

func (uc *OrderUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *Order) (*Order, error) {
	normalized, err := normalizeOrder(input, true)
	if err != nil {
		return nil, err
	}
	// 直接干预模式（组织显式关闭「超额后允许选择」）下，委托客户超额则拒绝入库；
	// 校验为尽力而为的时点查询，默认的仅提醒模式不拦截。
	if err := uc.creditControl.EnsurePartnerSelectionAllowed(ctx, organizationID, normalized.CustomerID); err != nil {
		return nil, err
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.create",
		Result:         "success",
		Details: map[string]string{
			"customer.id":   normalized.CustomerID.String(),
			"business_type": string(normalized.BusinessType),
		},
	}
	var created *Order
	err = func() error {
		// 幂等键可选：传入才启用，与建账口径一致；未提供时保持既有直建行为。
		if normalized.IdempotencyKey == "" {
			var createErr error
			created, createErr = uc.repo.Create(ctx, organizationID, actorID, normalized, audit)
			return createErr
		}
		if uc.transactor == nil {
			return ErrOrderInvalidArgument
		}
		// 镜像建账幂等段：事务内按幂等键查既有订单，同意图重放返回原单，
		// 意图不同返回冲突；并发同键由 (organization_id, idempotency_key)
		// 唯一索引兜底，见下方事务后重查。
		return uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
			existing, lookupErr := uc.repo.GetByIdempotencyKey(txCtx, organizationID, normalized.IdempotencyKey)
			if lookupErr != nil {
				return lookupErr
			}
			if existing != nil {
				if !sameOrderCreateIntent(existing, normalized) {
					return ErrOrderIdempotencyConflict
				}
				created = existing
				return nil
			}
			var createErr error
			created, createErr = uc.repo.Create(txCtx, organizationID, actorID, normalized, audit)
			return createErr
		})
	}()
	if err != nil && normalized.IdempotencyKey != "" {
		// 并发同键兜底（对齐建账 Create 的事务后重查）：另一请求已写入时，
		// 意图一致视为重放成功，否则保留原始错误。
		if existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey); lookupErr == nil && existing != nil && sameOrderCreateIntent(existing, normalized) {
			created = existing
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}
	if err := attachSeaMasterBillSummaries(ctx, uc.seaMasterBillRepo, organizationID, created); err != nil {
		return nil, err
	}
	if err := attachSeaDocumentSummaries(ctx, uc.seaDocumentRepo, organizationID, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *OrderUsecase) UpdateDraft(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, input *Order) (*Order, error) {
	if id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrOrderInvalidArgument
	}
	normalized, err := normalizeOrder(input, false)
	if err != nil {
		return nil, err
	}
	// 更新草稿更换/设定委托客户时同样执行直接干预拦截；口径与创建路径一致。
	if err := uc.creditControl.EnsurePartnerSelectionAllowed(ctx, organizationID, normalized.CustomerID); err != nil {
		return nil, err
	}
	if normalized.ShipmentType == nil || *normalized.ShipmentType != OrderShipmentFCL {
		hasContainers, err := uc.repo.HasContainers(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
		if hasContainers {
			return nil, ErrOrderContainerShipmentType
		}
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.update",
		Result:         "success",
		Details:        map[string]string{"order.id": id.String()},
	}
	if normalized.SeaMasterBillInput != nil && strings.TrimSpace(normalized.SeaMasterBillInput.CorrectionReason) != "" {
		audit.Action = "order.sea_master_bill.correct"
		audit.ResourceType = "sea_master_bill"
	}
	updated, err := uc.repo.UpdateDraft(ctx, organizationID, id, expectedVersion, normalized, audit)
	if err != nil {
		return nil, err
	}
	if err := attachSeaMasterBillSummaries(ctx, uc.seaMasterBillRepo, organizationID, updated); err != nil {
		return nil, err
	}
	if err := attachSeaDocumentSummaries(ctx, uc.seaDocumentRepo, organizationID, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func normalizeOrder(input *Order, creating bool) (*Order, error) {
	if input == nil || input.CustomerID == uuid.Nil || !input.BusinessType.Valid() || !input.TradeDirection.Valid() || (input.TradeTerm != "" && !input.TradeTerm.Valid()) || !input.PaymentTerm.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if input.BusinessType != OrderBusinessSE || input.TradeDirection != OrderTradeExport {
		return nil, ErrOrderBusinessUnsupported
	}
	output := *input
	output.IdempotencyKey = strings.TrimSpace(output.IdempotencyKey)
	output.CustomerReferenceNo = strings.TrimSpace(output.CustomerReferenceNo)
	output.InternalReferenceNo = strings.TrimSpace(output.InternalReferenceNo)
	output.ShipperShortName = strings.TrimSpace(output.ShipperShortName)
	output.ConsigneeShortName = strings.TrimSpace(output.ConsigneeShortName)
	output.ContractNo = strings.TrimSpace(output.ContractNo)
	output.CargoValue = strings.TrimSpace(output.CargoValue)
	output.CargoCurrency = strings.ToUpper(strings.TrimSpace(output.CargoCurrency))
	output.InsurancePremium = strings.TrimSpace(output.InsurancePremium)
	output.InsuranceCurrency = strings.ToUpper(strings.TrimSpace(output.InsuranceCurrency))
	output.UNNumber = strings.TrimSpace(output.UNNumber)
	output.HazardClass = strings.TrimSpace(output.HazardClass)
	output.FactoryName = strings.TrimSpace(output.FactoryName)
	output.CargoReadyAt = strings.TrimSpace(output.CargoReadyAt)
	output.DeclarationCutoffAt = strings.TrimSpace(output.DeclarationCutoffAt)
	output.ReceivedAt = strings.TrimSpace(output.ReceivedAt)
	output.VesselVoyage = strings.TrimSpace(output.VesselVoyage)
	output.ETD = strings.TrimSpace(output.ETD)
	output.ETA = strings.TrimSpace(output.ETA)
	output.SICutoff = strings.TrimSpace(output.SICutoff)
	output.DocCutoff = strings.TrimSpace(output.DocCutoff)
	output.CustomsCutoff = strings.TrimSpace(output.CustomsCutoff)
	output.VGMCutoff = strings.TrimSpace(output.VGMCutoff)
	output.GoodsDescription = strings.TrimSpace(output.GoodsDescription)
	output.TotalPackageUnit = strings.TrimSpace(output.TotalPackageUnit)
	output.SpecialRequirements = strings.TrimSpace(output.SpecialRequirements)
	output.OrderDate = strings.TrimSpace(output.OrderDate)
	output.BookingNo = strings.TrimSpace(output.BookingNo)
	output.Notes = strings.TrimSpace(output.Notes)
	output.BookingNotes = strings.TrimSpace(output.BookingNotes)
	output.AllocationNotes = strings.TrimSpace(output.AllocationNotes)
	output.OperationNotes = strings.TrimSpace(output.OperationNotes)
	if output.OrderDate == "" && creating {
		output.OrderDate = time.Now().UTC().Format(time.RFC3339)
	}
	if utf8.RuneCountInString(output.CustomerReferenceNo) > 100 || utf8.RuneCountInString(output.InternalReferenceNo) > 100 || utf8.RuneCountInString(output.BookingNo) > 100 || utf8.RuneCountInString(output.IdempotencyKey) > 128 || utf8.RuneCountInString(output.ShipperShortName) > 200 || utf8.RuneCountInString(output.ConsigneeShortName) > 200 || utf8.RuneCountInString(output.ContractNo) > 100 || utf8.RuneCountInString(output.HazardClass) > 16 || utf8.RuneCountInString(output.FactoryName) > 200 || utf8.RuneCountInString(output.VesselVoyage) > 100 || utf8.RuneCountInString(output.GoodsDescription) > 1000 || utf8.RuneCountInString(output.SpecialRequirements) > 1000 || utf8.RuneCountInString(output.Notes) > 1000 || utf8.RuneCountInString(output.BookingNotes) > 1000 || utf8.RuneCountInString(output.AllocationNotes) > 1000 || utf8.RuneCountInString(output.OperationNotes) > 1000 || output.TotalPackages != nil && *output.TotalPackages < 0 || output.TotalGrossWeightKg != nil && *output.TotalGrossWeightKg < 0 || output.TotalVolumeCbm != nil && *output.TotalVolumeCbm < 0 {
		return nil, ErrOrderInvalidArgument
	}
	roleCounts := make(map[OrderPersonnelRole]int, len(output.PersonnelAssignments))
	normalizedPersonnel := make([]*OrderPersonnel, 0, len(output.PersonnelAssignments))
	for _, assignment := range output.PersonnelAssignments {
		if assignment == nil || !assignment.Role.Valid() || assignment.Role == OrderPersonnelRoleCreator || assignment.UserID == uuid.Nil {
			return nil, ErrOrderInvalidArgument
		}
		roleCounts[assignment.Role]++
		if roleCounts[assignment.Role] > 1 {
			return nil, ErrOrderInvalidArgument
		}
		normalizedAssignment := *assignment
		if creating {
			normalizedAssignment.Notification = NewOrderPersonnelNotification(assignment.UserID)
		} else {
			normalizedAssignment.Notification = nil
		}
		normalizedPersonnel = append(normalizedPersonnel, &normalizedAssignment)
	}
	output.PersonnelAssignments = normalizedPersonnel
	if creating {
		// 提成相关岗位（销售/操作/客服）创建时必须配齐：订单人员是提成归属快照
		// 的唯一取数源，缺岗直接拒绝开单（创建者 CREATOR 岗位不计入）。
		if err := validateOrderCommissionPersonnel(output.PersonnelAssignments); err != nil {
			return nil, err
		}
	}
	houseNumbers := make(map[string]struct{}, len(output.ShippingDocuments))
	for index, document := range output.ShippingDocuments {
		normalized, err := normalizeOrderShippingDocument(document)
		if err != nil {
			return nil, err
		}
		if _, exists := houseNumbers[strings.ToLower(normalized.HouseNo)]; exists {
			return nil, ErrOrderShippingDocumentExists
		}
		houseNumbers[strings.ToLower(normalized.HouseNo)] = struct{}{}
		output.ShippingDocuments[index] = normalized
	}
	containerSpecs := make(map[uuid.UUID]struct{}, len(output.ContainerRequests))
	for _, request := range output.ContainerRequests {
		if request == nil || request.ContainerSpecID == uuid.Nil || request.Quantity < 1 || request.Quantity > 999 {
			return nil, ErrOrderInvalidArgument
		}
		if _, exists := containerSpecs[request.ContainerSpecID]; exists {
			return nil, ErrOrderInvalidArgument
		}
		containerSpecs[request.ContainerSpecID] = struct{}{}
	}
	if (output.CargoValue == "") != (output.CargoCurrency == "") || output.CargoValue != "" && (!cargoValuePattern.MatchString(output.CargoValue) || len(output.CargoCurrency) != 3) {
		return nil, ErrOrderInvalidArgument
	}
	if (output.InsurancePremium == "") != (output.InsuranceCurrency == "") || output.InsurancePremium != "" && (!cargoValuePattern.MatchString(output.InsurancePremium) || len(output.InsuranceCurrency) != 3) || output.UNNumber != "" && !unNumberPattern.MatchString(output.UNNumber) {
		return nil, ErrOrderInvalidArgument
	}
	for _, value := range []string{output.ETD, output.ETA, output.SICutoff, output.DocCutoff, output.CustomsCutoff, output.VGMCutoff, output.CargoReadyAt, output.DeclarationCutoffAt, output.ReceivedAt, output.OrderDate} {
		if value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return nil, ErrOrderInvalidArgument
			}
		}
	}
	if output.ShipmentType != nil && !output.ShipmentType.Valid() || output.ContainerOwnership != nil && !output.ContainerOwnership.Valid() || output.ShipmentMode != nil && !output.ShipmentMode.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if output.ShipmentType != nil && *output.ShipmentType == OrderShipmentBreakBulk && (len(output.ContainerRequests) > 0 || output.VGMCutoff != "") {
		return nil, ErrOrderInvalidArgument
	}
	if err := validateUUIDSet(output.ServiceTypeIDs); err != nil {
		return nil, err
	}
	if err := validateUUIDSet(output.CargoCategoryIDs); err != nil {
		return nil, err
	}
	if output.BusinessType == OrderBusinessSE {
		if output.ShippingLineID == nil || *output.ShippingLineID == uuid.Nil {
			return nil, errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "海运出口订单必须选择船公司")
		}
		contentOnlyUpdate := !creating && output.SeaMasterBillInput == nil &&
			output.SeaDocumentInput != nil && output.SeaDocumentInput.MasterBillContent != nil
		if output.SeaMasterBillInput == nil && !contentOnlyUpdate {
			return nil, errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "海运出口订单必须提供主单信息")
		}
		if output.SeaMasterBillInput != nil {
			masterBillInput := *output.SeaMasterBillInput
			output.SeaMasterBillInput = &masterBillInput
			normalizedMasterNo, err := ValidateAndNormalizeSeaMasterNo(output.SeaMasterBillInput.MasterNo)
			if err != nil {
				return nil, err
			}
			output.SeaMasterBillInput.MasterNo = normalizedMasterNo
			if creating {
				candidateIDSet := output.SeaMasterBillInput.CandidateID != nil && *output.SeaMasterBillInput.CandidateID != uuid.Nil
				candidateTEProvided := output.SeaMasterBillInput.CandidateTEID != nil
				candidateTEIDSet := candidateTEProvided && *output.SeaMasterBillInput.CandidateTEID != uuid.Nil
				if candidateIDSet {
					if output.SeaMasterBillInput.ExpectedCandidateVersion == nil || *output.SeaMasterBillInput.ExpectedCandidateVersion == 0 ||
						!candidateTEIDSet || output.SeaMasterBillInput.ExpectedCandidateTEVersion == nil || *output.SeaMasterBillInput.ExpectedCandidateTEVersion == 0 {
						return nil, ErrSeaMasterBillInvalidArgument
					}
				} else if output.SeaMasterBillInput.ExpectedCandidateVersion != nil || candidateTEProvided || output.SeaMasterBillInput.ExpectedCandidateTEVersion != nil {
					return nil, ErrSeaMasterBillInvalidArgument
				}
			}
		}

		if creating && output.SeaDocumentInput == nil {
			return nil, ErrSeaDocumentStructureInvalid
		}
		if output.SeaDocumentInput != nil {
			validatedDoc, err := ValidateSeaOrderDocumentInput(output.SeaDocumentInput, creating)
			if err != nil {
				return nil, err
			}
			output.SeaDocumentInput = validatedDoc
		}
	}
	return &output, nil
}

// validateOrderCommissionPersonnel 校验订单人员是否覆盖销售/操作/客服三岗，
// 缺失岗位按固定顺序收集并构造缺岗错误；三岗齐全返回 nil。仅创建路径调用，
// 草稿更新不携带人员字段，不施加该校验。
func validateOrderCommissionPersonnel(assignments []*OrderPersonnel) error {
	covered := make(map[OrderPersonnelRole]struct{}, len(assignments))
	for _, assignment := range assignments {
		covered[assignment.Role] = struct{}{}
	}
	missing := make([]OrderPersonnelRole, 0, len(orderCommissionPersonnelRoleOrder))
	for _, role := range orderCommissionPersonnelRoleOrder {
		if _, ok := covered[role]; !ok {
			missing = append(missing, role)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return NewOrderCommissionPersonnelMissing(missing)
}

// sameOrderCreateIntent 判定幂等键命中的既有订单与本次创建请求是否同一意图：
// 采用全量请求哈希比对（对齐建账 RequestHash 口径），任何载体字段差异都判为
// 不同意图返回冲突。不可纳入哈希的字段见 orderCreateIntentHash 注释中的排除
// 清单。
func sameOrderCreateIntent(existing *Order, requested *Order) bool {
	if existing == nil || requested == nil {
		return false
	}
	return orderCreateIntentHash(existing) == orderCreateIntentHash(requested)
}

// orderCreateIntentHash 计算订单创建意图指纹：哈希输入覆盖全部请求载体字段
// （标量、可空枚举、可空引用、货物/保险/危品、备注家族、服务类型与货物类别
// 集合、岗位人员、箱型箱量、海运主单号），集合类输入先排序再序列化，长度
// 前缀拼接后取 SHA-256。排除清单（不可比或非请求意图字段）：
//   - order_date：缺省时由服务端注入当前时间；
//   - etd/eta/vessel_voyage/origin/discharge/transit_location：
//     SE 订单读取时由主单航程（TransportExecution）回填，存储表示与请求原始
//     输入可能不一致（日期格式、候选航程差异）。注意：destination_location_id
//     与四类 cutoff（si/doc/customs/vgm）为请求原值入库并原样读回，非回填字段，
//     已纳入哈希；shipping_line_id 虽同为回填，但创建校验强制其与航程一致，
//     纳入哈希不会误伤真实重放，且能拦截同键换船公司（方向安全：宁可误 409 不可误放行）；
//   - sea_document 单证结构：缺省时由服务端按 HBL 存在性推导默认值，nil 与
//     默认值表示同一意图；
//   - 海运主单/分单内容与签发主体（MasterBillContent、HouseBill）：存储在
//     单证行而非订单行，需额外加载；主单号已单独覆盖；
//   - 人员通知意图、各类 ID/版本/时间戳/状态：服务端生成或派生。
func orderCreateIntentHash(order *Order) string {
	builder := strings.Builder{}
	part := func(values ...string) { writeFinanceHashParts(&builder, values...) }
	uuidText := func(value *uuid.UUID) string {
		if value == nil {
			return ""
		}
		return value.String()
	}
	intText := func(value *int) string {
		if value == nil {
			return ""
		}
		return strconv.Itoa(*value)
	}
	floatText := func(value *float64) string {
		if value == nil {
			return ""
		}
		return strconv.FormatFloat(*value, 'g', -1, 64)
	}

	part("customer", order.CustomerID.String())
	part("trade", string(order.BusinessType), string(order.TradeDirection), string(order.TradeTerm), string(order.PaymentTerm))
	part("agents", uuidText(order.ShippingLineID), uuidText(order.BookingAgentID), uuidText(order.ForeignAgentID), uuidText(order.ShippingAgentID))
	part("shipment", orderPointerText(order.ShipmentType), orderPointerText(order.ContainerOwnership), orderPointerText(order.ShipmentMode))
	part("references", order.CustomerReferenceNo, order.InternalReferenceNo, order.BookingNo, order.ContractNo)
	part("short-names", order.ShipperShortName, order.ConsigneeShortName)
	part("cargo", order.CargoValue, order.CargoCurrency, order.InsurancePremium, order.InsuranceCurrency)
	part("dangerous", order.UNNumber, order.HazardClass, order.FactoryName)
	part("times", order.CargoReadyAt, order.DeclarationCutoffAt, order.ReceivedAt, order.SICutoff, order.DocCutoff, order.CustomsCutoff, order.VGMCutoff)
	part("destination", uuidText(order.DestinationLocationID))
	part("goods", order.GoodsDescription, intText(order.TotalPackages), order.TotalPackageUnit, floatText(order.TotalGrossWeightKg), floatText(order.TotalVolumeCbm))
	part("notes", order.SpecialRequirements, order.Notes, order.BookingNotes, order.AllocationNotes, order.OperationNotes)
	part(append([]string{"service-types"}, orderSortedUUIDTexts(order.ServiceTypeIDs)...)...)
	part(append([]string{"cargo-categories"}, orderSortedUUIDTexts(order.CargoCategoryIDs)...)...)
	part(append([]string{"personnel"}, orderSortedPersonnelTexts(order.PersonnelAssignments)...)...)
	part(append([]string{"shipping-documents"}, orderSortedDocumentTexts(order.ShippingDocuments)...)...)
	part(append([]string{"container-requests"}, orderSortedContainerTexts(order.ContainerRequests)...)...)
	masterNo := ""
	if order.SeaMasterBillInput != nil {
		masterNo = order.SeaMasterBillInput.MasterNo
	} else if order.SeaMasterBill != nil {
		masterNo = order.SeaMasterBill.MasterNo
	}
	part("master-no", masterNo)
	return financeSHA256(builder.String())
}

func orderPointerText[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func orderSortedUUIDTexts(values []uuid.UUID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	sort.Strings(result)
	return result
}

func orderSortedPersonnelTexts(values []*OrderPersonnel) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		if item == nil || item.Role == OrderPersonnelRoleCreator {
			// 创建人由服务端注入，不属于请求意图。
			continue
		}
		result = append(result, fmt.Sprintf("%s:%s:%s", item.Role, item.UserID, item.OrganizationID))
	}
	sort.Strings(result)
	return result
}

func orderSortedDocumentTexts(values []*OrderShippingDocument) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		releaseType, note := "", ""
		if item.ReleaseType != nil {
			releaseType = *item.ReleaseType
		}
		if item.Note != nil {
			note = *item.Note
		}
		result = append(result, fmt.Sprintf("%s:%s:%s", strings.ToLower(item.HouseNo), releaseType, note))
	}
	sort.Strings(result)
	return result
}

func orderSortedContainerTexts(values []*OrderContainerRequest) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		result = append(result, fmt.Sprintf("%s:%d", item.ContainerSpecID, item.Quantity))
	}
	sort.Strings(result)
	return result
}

var cargoValuePattern = regexp.MustCompile(`^(0|[1-9]\d{0,17})(\.\d{1,4})?$`)
var unNumberPattern = regexp.MustCompile(`^\d{4}$`)

func validateUUIDSet(values []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return ErrOrderInvalidArgument
		}
		if _, exists := seen[value]; exists {
			return ErrOrderInvalidArgument
		}
		seen[value] = struct{}{}
	}
	return nil
}
