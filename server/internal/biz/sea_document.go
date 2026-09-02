package biz

import (
	"context"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

const (
	MaxSeaHouseNoLength       = 128
	MaxSeaPartyTextLength     = 5000  // Shipper, Consignee, NotifyParty, SecondNotifyParty, Marks
	MaxSeaDescriptionLength   = 10000 // GoodsDescription, Clauses
	MaxSeaGeneralFieldLength  = 64    // PackageUnit, FreightTerms, TransportTerms, BillForm, ReleaseType
	MaxSeaHouseBillNoteLength = 500
)

var (
	ErrSeaHouseBillNotFound                         = errors.NotFound("SEA_HOUSE_BILL_NOT_FOUND", "海运分单不存在")
	ErrSeaHouseBillInvalidArgument                  = errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "海运分单参数不合法")
	ErrSeaHouseBillExists                           = errors.Conflict("SEA_HOUSE_BILL_EXISTS", "海运分单号已存在")
	ErrSeaHouseBillConflict                         = errors.Conflict("SEA_HOUSE_BILL_CONFLICT", "海运分单已被更新，请刷新后重试")
	ErrSeaHouseBillStatusConflict                   = errors.Conflict("SEA_HOUSE_BILL_STATUS_CONFLICT", "海运分单状态或版本已被修改，请刷新后重试")
	ErrSeaMasterBillConflict                        = errors.Conflict("SEA_MASTER_BILL_CONFLICT", "海运主单已被更新，请刷新后重试")
	ErrSeaDocumentNoActiveLink                      = errors.BadRequest("SEA_DOCUMENT_NO_ACTIVE_LINK", "海运订单未关联活动主单")
	ErrSeaDocumentStructureConflict                 = errors.Conflict("SEA_DOCUMENT_STRUCTURE_CONFLICT", "单证结构状态或版本已被修改，请刷新后重试")
	ErrSeaDocumentStructureInvalid                  = errors.BadRequest("SEA_DOCUMENT_STRUCTURE_INVALID", "当前单证结构不允许该操作")
	ErrSeaDocumentDirectAddHBLBlocked               = errors.Conflict("SEA_DOCUMENT_DIRECT_ADD_HBL_BLOCKED", "DIRECT 直单下禁止直接添加 HBL，请先取消直单标记")
	ErrSeaDocumentDeleteLastHBLConfirmationRequired = errors.BadRequest("SEA_DOCUMENT_DELETE_LAST_HBL_CONFIRMATION_REQUIRED", "删除最后一张分单需明确确认回到未确定状态")
	ErrSeaDocumentIssuerOrgNotFound                 = errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "未找到所属公司或总部签发主体")
	ErrOrderCustomerChangeWithHouseBillBlocked      = errors.Conflict("ORDER_CUSTOMER_CHANGE_WITH_HOUSE_BILL_BLOCKED", "存在以委托单位签发的分单，请先调整分单签发主体后再修改客户")
	ErrSeaShippingDocumentsDeprecated               = errors.BadRequest("ORDER_SHIPPING_DOCUMENT_INVALID_ARGUMENT", "海运出口订单禁止使用旧提单接口，请使用海运单证接口")
	ErrSeaBillContentInvalidArgument                = errors.BadRequest("SEA_BILL_CONTENT_INVALID_ARGUMENT", "提单内容参数不合法")
	ErrSeaDocumentInvalidArgument                   = errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "海运单证参数不合法")
)

// NormalizeSeaHouseNo 校验并规范化海运出口分单号（HBL）。
// 规则：
// 1. 原始号码无损保存（由调用方单独保留）；
// 2. Unicode NFC 规范化；
// 3. 去除首尾 Unicode 空白；
// 4. 仅英文字母统一转大写（a-z -> A-Z），保留内部全部字符/标点/年份/前导零/非英文字符；
// 5. 长度限制最多 128 个字符（UTF-8 字符/码点数）。
func NormalizeSeaHouseNo(raw string) (string, error) {
	if raw == "" {
		return "", errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "海运分单号不能为空")
	}
	if utf8.RuneCountInString(raw) > MaxSeaHouseNoLength {
		return "", errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "海运分单号长度不能超过 128 个字符")
	}
	// 1. NFC 规范化
	nfcStr := norm.NFC.String(raw)
	// 2. 去除首尾 Unicode 空白
	trimmed := strings.TrimFunc(nfcStr, unicode.IsSpace)
	if trimmed == "" {
		return "", errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "海运分单号不能为空")
	}
	// 3. 仅对英文字母统一转大写（a-z -> A-Z），其他字符完全保留
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String(), nil
}

// ValidateSeaBillContent 校验并规范化通用提单内容（首尾空白去除并持久化规范化副本）。
func ValidateSeaBillContent(content *SeaBillContent) (*SeaBillContent, error) {
	if content == nil {
		return nil, nil
	}

	normalizeAndValidateText := func(name string, p *string, maxLen int) (*string, error) {
		if p == nil {
			return nil, nil
		}
		trimmed := strings.TrimSpace(*p)
		if utf8.RuneCountInString(trimmed) > maxLen {
			return nil, errors.BadRequest("SEA_BILL_CONTENT_INVALID_ARGUMENT", name+"长度不能超过限制")
		}
		if trimmed == "" {
			return nil, nil
		}
		return &trimmed, nil
	}

	var err error
	res := &SeaBillContent{
		PackageCount:  content.PackageCount,
		GrossWeightKg: content.GrossWeightKg,
		VolumeCbm:     content.VolumeCbm,
	}

	if res.ShipperText, err = normalizeAndValidateText("发货人", content.ShipperText, MaxSeaPartyTextLength); err != nil {
		return nil, err
	}
	if res.ConsigneeText, err = normalizeAndValidateText("收货人", content.ConsigneeText, MaxSeaPartyTextLength); err != nil {
		return nil, err
	}
	if res.NotifyPartyText, err = normalizeAndValidateText("通知人", content.NotifyPartyText, MaxSeaPartyTextLength); err != nil {
		return nil, err
	}
	if res.SecondNotifyPartyText, err = normalizeAndValidateText("第二通知人", content.SecondNotifyPartyText, MaxSeaPartyTextLength); err != nil {
		return nil, err
	}
	if res.MarksText, err = normalizeAndValidateText("唛头", content.MarksText, MaxSeaPartyTextLength); err != nil {
		return nil, err
	}
	if res.GoodsDescriptionText, err = normalizeAndValidateText("货描", content.GoodsDescriptionText, MaxSeaDescriptionLength); err != nil {
		return nil, err
	}
	if res.PackageUnit, err = normalizeAndValidateText("包装单位", content.PackageUnit, MaxSeaGeneralFieldLength); err != nil {
		return nil, err
	}
	if res.FreightTerms, err = normalizeAndValidateText("运费条款", content.FreightTerms, MaxSeaGeneralFieldLength); err != nil {
		return nil, err
	}
	if res.TransportTerms, err = normalizeAndValidateText("运输条款", content.TransportTerms, MaxSeaGeneralFieldLength); err != nil {
		return nil, err
	}
	if res.BillForm, err = normalizeAndValidateText("提单形式", content.BillForm, MaxSeaGeneralFieldLength); err != nil {
		return nil, err
	}
	if res.ReleaseType, err = normalizeAndValidateText("放单方式", content.ReleaseType, MaxSeaGeneralFieldLength); err != nil {
		return nil, err
	}
	if res.Clauses, err = normalizeAndValidateText("特别条款", content.Clauses, MaxSeaDescriptionLength); err != nil {
		return nil, err
	}

	if content.PackageCount != nil && *content.PackageCount < 0 {
		return nil, errors.BadRequest("SEA_BILL_CONTENT_INVALID_ARGUMENT", "件数不能为负数")
	}

	if content.GrossWeightKg != nil {
		w := *content.GrossWeightKg
		if math.IsNaN(w) || math.IsInf(w, 0) || w < 0 {
			return nil, errors.BadRequest("SEA_BILL_CONTENT_INVALID_ARGUMENT", "毛重必须为有效的非负数值")
		}
	}

	if content.VolumeCbm != nil {
		v := *content.VolumeCbm
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return nil, errors.BadRequest("SEA_BILL_CONTENT_INVALID_ARGUMENT", "体积必须为有效的非负数值")
		}
	}

	return res, nil
}

// ValidateSeaHouseBillInput 校验单个分单输入。
func ValidateSeaHouseBillInput(input *SeaHouseBillInput) (*SeaHouseBillInput, error) {
	if input == nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	if input.HouseNo == "" {
		return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "分单号不能为空")
	}
	if _, err := NormalizeSeaHouseNo(input.HouseNo); err != nil {
		return nil, err
	}
	if !input.IssuerSource.Valid() {
		return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "签发主体来源无效")
	}
	switch input.IssuerSource {
	case SeaHouseBillIssuerSourceSelfOrganization, SeaHouseBillIssuerSourceCustomerPartner:
		if input.IssuerPartnerID != nil {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "本公司或委托单位签发时不得指定其他合作伙伴")
		}
	case SeaHouseBillIssuerSourceOtherPartner:
		if input.IssuerPartnerID == nil || *input.IssuerPartnerID == uuid.Nil {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "其他主体签发必须选择合作伙伴")
		}
	}

	var note *string
	if input.Note != nil {
		trimmedNote := strings.TrimSpace(*input.Note)
		if utf8.RuneCountInString(trimmedNote) > MaxSeaHouseBillNoteLength {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "分单备注不能超过 500 字符")
		}
		if trimmedNote != "" {
			note = &trimmedNote
		}
	}

	normalizedContent, err := ValidateSeaBillContent(input.Content)
	if err != nil {
		return nil, err
	}

	return &SeaHouseBillInput{
		ID:              input.ID,
		HouseNo:         input.HouseNo, // raw houseNo 保持原样无损
		IssuerSource:    input.IssuerSource,
		IssuerPartnerID: input.IssuerPartnerID,
		Note:            note,
		Content:         normalizedContent,
		ExpectedVersion: input.ExpectedVersion,
	}, nil
}

// ValidateSeaOrderDocumentInput 校验订单整包单证输入。
func ValidateSeaOrderDocumentInput(input *SeaOrderDocumentInput, isCreate bool) (*SeaOrderDocumentInput, error) {
	if input == nil {
		return nil, nil
	}

	if !isCreate {
		if input.HouseBills != nil && len(input.HouseBills) > 0 {
			return nil, errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "订单整单更新禁止直接提交分单集合变更，请使用专用单证命令")
		}
		if input.DocumentStructure != nil && input.ExpectedLinkVersion == nil {
			return nil, errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改单证结构必须提供 expected_link_version")
		}
		if input.MasterBillContent != nil && input.ExpectedMblVersion == nil {
			return nil, errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改主单内容必须提供 expected_mbl_version")
		}
	}

	var validatedMblContent *SeaBillContent
	if input.MasterBillContent != nil {
		var err error
		validatedMblContent, err = ValidateSeaBillContent(input.MasterBillContent)
		if err != nil {
			return nil, err
		}
	}

	var validatedHBs []*SeaHouseBillInput
	if input.HouseBills != nil {
		validatedHBs = make([]*SeaHouseBillInput, 0, len(input.HouseBills))
		for _, hb := range input.HouseBills {
			if hb == nil {
				return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "分单项不能为空")
			}
			vhb, err := ValidateSeaHouseBillInput(hb)
			if err != nil {
				return nil, err
			}
			validatedHBs = append(validatedHBs, vhb)
		}
	}

	if isCreate {
		hasHBLs := len(validatedHBs) > 0
		if input.DocumentStructure != nil {
			switch *input.DocumentStructure {
			case SeaDocumentStructureDirect:
				if hasHBLs {
					return nil, ErrSeaDocumentDirectAddHBLBlocked
				}
			case SeaDocumentStructureHouse:
				if !hasHBLs {
					return nil, errors.BadRequest("SEA_DOCUMENT_STRUCTURE_INVALID", "HOUSE 单证结构必须至少包含一张分单")
				}
			case SeaDocumentStructureUndetermined:
				// ok
			default:
				return nil, ErrSeaDocumentStructureInvalid
			}
		}
	}

	return &SeaOrderDocumentInput{
		DocumentStructure:   input.DocumentStructure,
		ExpectedLinkVersion: input.ExpectedLinkVersion,
		ExpectedMblVersion:  input.ExpectedMblVersion,
		MasterBillContent:   validatedMblContent,
		HouseBills:          validatedHBs,
	}, nil
}

type SeaDocumentStructure string

const (
	SeaDocumentStructureUndetermined SeaDocumentStructure = "UNDETERMINED"
	SeaDocumentStructureDirect       SeaDocumentStructure = "DIRECT"
	SeaDocumentStructureHouse        SeaDocumentStructure = "HOUSE"
)

func (s SeaDocumentStructure) Valid() bool {
	return s == SeaDocumentStructureUndetermined || s == SeaDocumentStructureDirect || s == SeaDocumentStructureHouse
}

type SeaHouseBillIssuerSource string

const (
	SeaHouseBillIssuerSourceSelfOrganization SeaHouseBillIssuerSource = "SELF_ORGANIZATION"
	SeaHouseBillIssuerSourceCustomerPartner  SeaHouseBillIssuerSource = "CUSTOMER_PARTNER"
	SeaHouseBillIssuerSourceOtherPartner     SeaHouseBillIssuerSource = "OTHER_PARTNER"
)

func (s SeaHouseBillIssuerSource) Valid() bool {
	return s == SeaHouseBillIssuerSourceSelfOrganization || s == SeaHouseBillIssuerSourceCustomerPartner || s == SeaHouseBillIssuerSourceOtherPartner
}

type SeaHouseBillStatus string

const (
	SeaHouseBillStatusDraft     SeaHouseBillStatus = "DRAFT"
	SeaHouseBillStatusConfirmed SeaHouseBillStatus = "CONFIRMED"
	SeaHouseBillStatusReleased  SeaHouseBillStatus = "RELEASED"
)

func (s SeaHouseBillStatus) Valid() bool {
	return s == SeaHouseBillStatusDraft || s == SeaHouseBillStatusConfirmed || s == SeaHouseBillStatusReleased
}

type SeaDocumentAction string

const (
	SeaDocumentActionMarkDirect              SeaDocumentAction = "MARK_DIRECT"
	SeaDocumentActionCancelDirect            SeaDocumentAction = "CANCEL_DIRECT"
	SeaDocumentActionAddHouseBill            SeaDocumentAction = "ADD_HOUSE_BILL"
	SeaDocumentActionUpdateHouseBill         SeaDocumentAction = "UPDATE_HOUSE_BILL"
	SeaDocumentActionRemoveHouseBill         SeaDocumentAction = "REMOVE_HOUSE_BILL"
	SeaDocumentActionUpdateMasterBillContent SeaDocumentAction = "UPDATE_MASTER_BILL_CONTENT"
)

// SeaBillContent 通用提单内容（MBL 与 HBL 独立维护）。
type SeaBillContent struct {
	ShipperText           *string
	ConsigneeText         *string
	NotifyPartyText       *string
	SecondNotifyPartyText *string
	MarksText             *string
	GoodsDescriptionText  *string
	PackageCount          *int32
	PackageUnit           *string
	GrossWeightKg         *float64
	VolumeCbm             *float64
	FreightTerms          *string
	TransportTerms        *string
	BillForm              *string
	ReleaseType           *string
	Clauses               *string
}

// SeaHouseBill 海运分单（HBL）信息。
type SeaHouseBill struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	OrderID                uuid.UUID
	MasterBillID           uuid.UUID
	HouseNo                string
	NormalizedHouseNo      string
	IssuerSource           SeaHouseBillIssuerSource
	IssuerOrganizationID   *uuid.UUID
	IssuerOrganizationName string
	IssuerPartnerID        *uuid.UUID
	IssuerPartnerName      string
	Status                 SeaHouseBillStatus
	Version                uint64
	Note                   *string
	Content                *SeaBillContent
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// SeaHouseBillInput 表单中的海运分单输入。
type SeaHouseBillInput struct {
	ID              *uuid.UUID
	HouseNo         string
	IssuerSource    SeaHouseBillIssuerSource
	IssuerPartnerID *uuid.UUID
	Note            *string
	Content         *SeaBillContent
	ExpectedVersion *uint64
}

// SeaMasterBillDetail 海运主单内容详情。
type SeaMasterBillDetail struct {
	ID                uuid.UUID
	MasterNo          string
	IssuerPartnerID   uuid.UUID
	IssuerPartnerName string
	Status            string
	Version           uint64
	Content           *SeaBillContent
	MemberCount       int
}

// SeaOrderDocumentSummary 订单中的海运单证摘要。
type SeaOrderDocumentSummary struct {
	DocumentStructure SeaDocumentStructure
	LinkVersion       uint64
	HouseBillCount    int
	HouseNos          []string
}

// SeaOrderDocumentInput 订单创建/更新中的海运单证整包输入。
type SeaOrderDocumentInput struct {
	DocumentStructure   *SeaDocumentStructure
	ExpectedLinkVersion *uint64
	ExpectedMblVersion  *uint64
	MasterBillContent   *SeaBillContent
	HouseBills          []*SeaHouseBillInput
}

// SeaOrderDocuments 聚合单证响应。
type SeaOrderDocuments struct {
	OrderID           uuid.UUID
	DocumentStructure SeaDocumentStructure
	LinkVersion       uint64
	MasterBill        *SeaMasterBillDetail
	HouseBills        []*SeaHouseBill
	AllowedActions    []SeaDocumentAction
}

type SeaDocumentRepo interface {
	GetSeaOrderDocuments(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderDocuments, error)
	GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*SeaOrderDocumentSummary, error)
	MarkSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error)
	CancelSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error)
	AddSeaHouseBill(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error)
	UpdateSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error)
	RemoveSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, returnToUndetermined bool, audit *AuditEvent) error
	UpdateSeaMasterBillContent(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedMblVersion uint64, content *SeaBillContent, audit *AuditEvent) (*SeaMasterBillDetail, error)
}

type SeaDocumentUsecase struct {
	repo SeaDocumentRepo
}

func NewSeaDocumentUsecase(repo SeaDocumentRepo) *SeaDocumentUsecase {
	return &SeaDocumentUsecase{
		repo: repo,
	}
}

func validateAuditEvent(audit *AuditEvent, organizationID, actorID uuid.UUID) error {
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return ErrSeaDocumentInvalidArgument
	}
	return nil
}

func (uc *SeaDocumentUsecase) GetSeaOrderDocuments(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderDocuments, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	return uc.repo.GetSeaOrderDocuments(ctx, organizationID, orderID)
}

func (uc *SeaDocumentUsecase) MarkSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.MarkSeaOrderDirect(ctx, organizationID, actorID, orderID, expectedLinkVersion, audit)
}

func (uc *SeaDocumentUsecase) CancelSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.CancelSeaOrderDirect(ctx, organizationID, actorID, orderID, expectedLinkVersion, audit)
}

func (uc *SeaDocumentUsecase) AddSeaHouseBill(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return nil, err
	}
	validatedInput, err := ValidateSeaHouseBillInput(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.AddSeaHouseBill(ctx, organizationID, actorID, orderID, expectedLinkVersion, validatedInput, audit)
}

func (uc *SeaDocumentUsecase) UpdateSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || houseBillID == uuid.Nil {
		return nil, ErrSeaHouseBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return nil, err
	}
	validatedInput, err := ValidateSeaHouseBillInput(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.UpdateSeaHouseBill(ctx, organizationID, actorID, orderID, houseBillID, expectedVersion, expectedLinkVersion, validatedInput, audit)
}

func (uc *SeaDocumentUsecase) RemoveSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, returnToUndetermined bool, audit *AuditEvent) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || houseBillID == uuid.Nil {
		return ErrSeaHouseBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return err
	}
	return uc.repo.RemoveSeaHouseBill(ctx, organizationID, actorID, orderID, houseBillID, expectedVersion, expectedLinkVersion, returnToUndetermined, audit)
}

func (uc *SeaDocumentUsecase) UpdateSeaMasterBillContent(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedMblVersion uint64, content *SeaBillContent, audit *AuditEvent) (*SeaMasterBillDetail, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaMasterBillInvalidArgument
	}
	if err := validateAuditEvent(audit, organizationID, actorID); err != nil {
		return nil, err
	}
	if content == nil {
		return nil, ErrSeaMasterBillInvalidArgument
	}
	validatedContent, err := ValidateSeaBillContent(content)
	if err != nil {
		return nil, err
	}
	return uc.repo.UpdateSeaMasterBillContent(ctx, organizationID, actorID, orderID, expectedMblVersion, validatedContent, audit)
}
