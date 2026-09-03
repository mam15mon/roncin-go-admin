package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrSeaMasterBillNotFound             = errors.NotFound("SEA_MASTER_BILL_NOT_FOUND", "海运主单不存在")
	ErrSeaMasterBillInvalidArgument      = errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "海运主单参数不合法")
	ErrSeaMasterBillExists               = errors.Conflict("SEA_MASTER_BILL_EXISTS", "海运主单已存在")
	ErrSeaMasterBillConfirmationRequired = errors.Conflict("SEA_MASTER_BILL_CONFIRMATION_REQUIRED", "发现已有海运主单，请确认关联")
	ErrSeaMasterBillVoyageConflict       = errors.Conflict("SEA_MASTER_BILL_VOYAGE_CONFLICT", "海运主单航程信息与本票不一致")
	ErrSeaMasterBillStatusConflict       = errors.Conflict("SEA_MASTER_BILL_STATUS_CONFLICT", "海运主单状态或版本已被修改，请刷新后重试")
	ErrSeaMasterBillCorrectionBlocked    = errors.Conflict("SEA_MASTER_BILL_CORRECTION_BLOCKED", "共享主单禁止直接修改主单号或签发主体")
)

var seaMasterNoInputRegex = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// ValidateAndNormalizeSeaMasterNo 校验并规范化海运出口主单号。
// 规则：除 ASCII 小写转大写外不得 TrimSpace 或静默删除任何字符，首尾空格及其他字符均明确非法。
func ValidateAndNormalizeSeaMasterNo(masterNo string) (string, error) {
	if masterNo == "" {
		return "", errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "海运出口订单必须填写主单号")
	}
	if !seaMasterNoInputRegex.MatchString(masterNo) {
		return "", errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "主单号仅允许包含英文字母和数字")
	}
	return strings.ToUpper(masterNo), nil
}

// SplitVesselVoyage 解析船名航次组合字符串。
func SplitVesselVoyage(vesselVoyage string) (string, string) {
	v := strings.TrimSpace(vesselVoyage)
	if v == "" {
		return "", ""
	}
	if parts := strings.SplitN(v, "/", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if idx := strings.LastIndex(v, " "); idx != -1 {
		return strings.TrimSpace(v[:idx]), strings.TrimSpace(v[idx+1:])
	}
	return v, ""
}

// CombineVesselVoyage 拼接船名航次组合字符串。
func CombineVesselVoyage(vessel, voyage string) string {
	v := strings.TrimSpace(vessel)
	vy := strings.TrimSpace(voyage)
	if v == "" {
		return vy
	}
	if vy == "" {
		return v
	}
	return v + " / " + vy
}

type SeaMasterBillStatus string

const (
	SeaMasterBillStatusDraft     SeaMasterBillStatus = "DRAFT"
	SeaMasterBillStatusConfirmed SeaMasterBillStatus = "CONFIRMED"
	SeaMasterBillStatusReleased  SeaMasterBillStatus = "RELEASED"
)

type SeaMasterBillOrderLinkStatus string

const (
	SeaMasterBillOrderLinkActive SeaMasterBillOrderLinkStatus = "ACTIVE"
	SeaMasterBillOrderLinkEnded  SeaMasterBillOrderLinkStatus = "ENDED"
)

// SeaTransportExecution 独立运输执行（实际航程事实）。
type SeaTransportExecution struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	CarrierID             uuid.UUID
	CarrierName           string
	OriginLocationID      uuid.UUID
	OriginLocationName    string
	DischargeLocationID   uuid.UUID
	DischargeLocationName string
	TransitLocationID     *uuid.UUID
	TransitLocationName   string
	VesselName            string
	VoyageNo              string
	ETD                   *time.Time
	ETA                   *time.Time
	Version               uint64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// SeaMasterBill 海运共享 MBL。
type SeaMasterBill struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	IssuerPartnerID      uuid.UUID
	IssuerPartnerName    string
	TransportExecutionID uuid.UUID
	TransportExecution   *SeaTransportExecution
	MasterNo             string
	NormalizedMasterNo   string
	Status               SeaMasterBillStatus
	Version              uint64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SeaMasterBillOrderLink 海运操作票与共享 MBL 当前/历史成员关系。
type SeaMasterBillOrderLink struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	MasterBillID   uuid.UUID
	OrderID        uuid.UUID
	Status         SeaMasterBillOrderLinkStatus
	StartedAt      time.Time
	EndedAt        *time.Time
	EndedReason    string
	Version        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SeaMasterBillInput 表单提交的海运主单输入。
type SeaMasterBillInput struct {
	MasterNo                 string
	IssuerPartnerID          uuid.UUID
	CandidateID              *uuid.UUID
	ExpectedCandidateVersion *uint64
	CorrectionReason         string
}

// SeaVoyageConflict 航程冲突项。
type SeaVoyageConflict struct {
	Field       string
	MasterValue string
	OrderValue  string
	Message     string
}

// SeaMasterBillMemberSummary 共享主单成员操作票摘要。
type SeaMasterBillMemberSummary struct {
	OrderID             uuid.UUID
	OrderNo             string
	CustomerReferenceNo string
}

// SeaMasterBillCandidate 已有共享主单候选。
type SeaMasterBillCandidate struct {
	ID                 uuid.UUID
	Version            uint64
	MasterNo           string
	IssuerPartnerID    uuid.UUID
	IssuerPartnerName  string
	TransportExecution *SeaTransportExecution
	MemberCount        int
	Members            []*SeaMasterBillMemberSummary
}

// SeaMasterBillMatchResult 候选匹配结果。
type SeaMasterBillMatchResult struct {
	Matched   bool
	Candidate *SeaMasterBillCandidate
	Conflicts []*SeaVoyageConflict
}

// SeaMasterBillSummary 订单当前关联主单摘要。
type SeaMasterBillSummary struct {
	MasterBillID              uuid.UUID
	MasterNo                  string
	IssuerPartnerID           uuid.UUID
	IssuerPartnerName         string
	TransportExecutionID      uuid.UUID
	TransportExecutionVersion uint64
	CarrierID                 *uuid.UUID
	CarrierName               string
	OriginLocationID          *uuid.UUID
	OriginLocationName        string
	DischargeLocationID       *uuid.UUID
	DischargeLocationName     string
	TransitLocationID         *uuid.UUID
	TransitLocationName       string
	VesselName                string
	VoyageNo                  string
	ETD                       string
	ETA                       string
	Status                    string
	Version                   uint64
	MemberCount               int
}

// CheckSeaVoyageConflicts 比对已有运输执行与本票航程是否一致。
// 仅当已有运输执行和本票均提供相应字段值时进行比对；若任一方未填写则不视为冲突。
func CheckSeaVoyageConflicts(masterVoyage *SeaTransportExecution, orderVoyage *SeaTransportExecution) []*SeaVoyageConflict {
	var conflicts []*SeaVoyageConflict
	if masterVoyage == nil || orderVoyage == nil {
		return conflicts
	}

	// 0. 承运人（双方均有值时比对）
	if masterVoyage.CarrierID != uuid.Nil && orderVoyage.CarrierID != uuid.Nil {
		if masterVoyage.CarrierID != orderVoyage.CarrierID {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "carrier_id",
				MasterValue: masterVoyage.CarrierID.String(),
				OrderValue:  orderVoyage.CarrierID.String(),
				Message:     "承运人/船公司不一致",
			})
		}
	}

	// 1. 起运港（双方均有值时比对）
	if masterVoyage.OriginLocationID != uuid.Nil && orderVoyage.OriginLocationID != uuid.Nil {
		if masterVoyage.OriginLocationID != orderVoyage.OriginLocationID {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "origin_location_id",
				MasterValue: masterVoyage.OriginLocationID.String(),
				OrderValue:  orderVoyage.OriginLocationID.String(),
				Message:     "起运港不一致",
			})
		}
	}

	// 2. 卸货港（双方均有值时比对）
	if masterVoyage.DischargeLocationID != uuid.Nil && orderVoyage.DischargeLocationID != uuid.Nil {
		if masterVoyage.DischargeLocationID != orderVoyage.DischargeLocationID {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "discharge_location_id",
				MasterValue: masterVoyage.DischargeLocationID.String(),
				OrderValue:  orderVoyage.DischargeLocationID.String(),
				Message:     "卸货港不一致",
			})
		}
	}

	// 3. 船名（双方均非空时比对）
	if strings.TrimSpace(masterVoyage.VesselName) != "" && strings.TrimSpace(orderVoyage.VesselName) != "" {
		if !strings.EqualFold(strings.TrimSpace(masterVoyage.VesselName), strings.TrimSpace(orderVoyage.VesselName)) {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "vessel_name",
				MasterValue: masterVoyage.VesselName,
				OrderValue:  orderVoyage.VesselName,
				Message:     "船名不一致",
			})
		}
	}

	// 4. 航次（双方均非空时比对）
	if strings.TrimSpace(masterVoyage.VoyageNo) != "" && strings.TrimSpace(orderVoyage.VoyageNo) != "" {
		if !strings.EqualFold(strings.TrimSpace(masterVoyage.VoyageNo), strings.TrimSpace(orderVoyage.VoyageNo)) {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "voyage_no",
				MasterValue: masterVoyage.VoyageNo,
				OrderValue:  orderVoyage.VoyageNo,
				Message:     "航次不一致",
			})
		}
	}

	// 5. 中转港（双方均有值时比对）
	if masterVoyage.TransitLocationID != nil && *masterVoyage.TransitLocationID != uuid.Nil &&
		orderVoyage.TransitLocationID != nil && *orderVoyage.TransitLocationID != uuid.Nil {
		if *masterVoyage.TransitLocationID != *orderVoyage.TransitLocationID {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "transit_location_id",
				MasterValue: masterVoyage.TransitLocationID.String(),
				OrderValue:  orderVoyage.TransitLocationID.String(),
				Message:     "中转港不一致",
			})
		}
	}

	// 6. ETD（双方有值时匹配）
	if masterVoyage.ETD != nil && !masterVoyage.ETD.IsZero() &&
		orderVoyage.ETD != nil && !orderVoyage.ETD.IsZero() {
		masterDate := masterVoyage.ETD.Format("2006-01-02")
		orderDate := orderVoyage.ETD.Format("2006-01-02")
		if masterDate != orderDate {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "etd",
				MasterValue: masterDate,
				OrderValue:  orderDate,
				Message:     "ETD 开航日期不一致",
			})
		}
	}

	// 7. ETA（双方有值时匹配）
	if masterVoyage.ETA != nil && !masterVoyage.ETA.IsZero() &&
		orderVoyage.ETA != nil && !orderVoyage.ETA.IsZero() {
		masterDate := masterVoyage.ETA.Format("2006-01-02")
		orderDate := orderVoyage.ETA.Format("2006-01-02")
		if masterDate != orderDate {
			conflicts = append(conflicts, &SeaVoyageConflict{
				Field:       "eta",
				MasterValue: masterDate,
				OrderValue:  orderDate,
				Message:     "ETA 预计到达日期不一致",
			})
		}
	}

	return conflicts
}

type SeaMasterBillRepo interface {
	MatchCandidate(ctx context.Context, organizationID, issuerPartnerID uuid.UUID, normalizedMasterNo string, voyage *SeaTransportExecution) (*SeaMasterBillMatchResult, error)
	GetSummaryByOrderID(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaMasterBillSummary, error)
	GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*SeaMasterBillSummary, error)
}
