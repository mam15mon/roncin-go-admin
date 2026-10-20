package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAirWaybillInvalid  = errors.BadRequest("AIR_WAYBILL_INVALID", "空运主分单资料不合法")
	ErrAirWaybillNotFound = errors.NotFound("AIR_WAYBILL_NOT_FOUND", "空运主单不存在")
	ErrAirWaybillConflict = errors.Conflict("AIR_WAYBILL_CONFLICT", "已被更新，请刷新后重试")
	ErrAirWaybillExists   = errors.Conflict("AIR_WAYBILL_EXISTS", "主单号已存在，请明确选择已有主单；分单号同主单内不可重复")
	ErrAirWaybillBlocked  = errors.Conflict("AIR_WAYBILL_BLOCKED", "主单成员存在无权、锁定、终止或结案订单，不能修改共享资料")
)

type AirMasterWaybillInput struct {
	MasterNo              string
	AirlineID             uuid.UUID
	OriginAirportID       uuid.UUID
	DestinationAirportID  uuid.UUID
	FlightNo              string
	ETD                   string
	ETA                   string
	ActualDepartureAt     string
	ActualArrivalAt       string
	DocumentCutoffAt      string
	CustomsCutoffAt       string
	Status                string
	DeclaredPackages      *int
	DeclaredGrossWeightKg *float64
	DeclaredVolumeCbm     *float64
	Notes                 string
}
type AirMasterWaybill struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Version                uint64
	Content                AirMasterWaybillInput
	AirlineName            string
	OriginAirportName      string
	DestinationAirportName string
	MemberCount            int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
type AirOrderInput struct {
	MasterWaybillID       *uuid.UUID
	ExpectedMasterVersion uint64
	ExpectedLinkVersion   uint64
	Master                *AirMasterWaybillInput
	HouseNo               string
	ChangeReason          string
	HouseContent          string
	HouseStatus           string
}
type AirOrderInfo struct {
	Master       *AirMasterWaybill
	HouseNo      string
	HouseStatus  string
	HouseVersion uint64
	LinkID       uuid.UUID
	LinkVersion  uint64
	HouseContent string
	HouseID      uuid.UUID
	Active       bool
}
type AirMasterMember struct {
	OrderID                     uuid.UUID
	OrderNo                     string
	CustomerName                string
	HouseNo                     string
	Packages                    *int
	PackageUnit                 string
	GrossWeightKg               *float64
	VolumeCbm                   *float64
	FlowStatus                  string
	TerminationStatus           string
	ClosureStatus               string
	Locked                      bool
	OrderVersion                uint64
	HouseStatus                 string
	ConfirmedChargeableWeightKg string
	ConfirmationStale           bool
}
type AirMasterTotals struct {
	PackagesByUnit map[string]int
	GrossWeightKg  float64
	VolumeCbm      float64
	Complete       bool
	MemberCount    int
}
type AirMasterHistory struct {
	ID         uuid.UUID
	Version    uint64
	Action     string
	Reason     string
	ActorID    uuid.UUID
	OccurredAt time.Time
	Content    AirMasterWaybillInput
}
type AirMasterDetail struct {
	HistoryTotal   int
	Master         *AirMasterWaybill
	Members        []*AirMasterMember
	Total          int
	Totals         AirMasterTotals
	History        []*AirMasterHistory
	BlockedReasons []string
}
type AirMasterPreview struct {
	InputFingerprint string
	Members          []*AirMasterMember
	BlockedReasons   []string
	CanExecute       bool
}
type AirExportRepo interface {
	HouseVersions(context.Context, uuid.UUID, uuid.UUID, SelectorListOptions) ([]*AirHouseVersion, int, error)
	LinkCandidates(context.Context, uuid.UUID, uuid.UUID, SelectorListOptions) ([]*AirMasterMember, int, error)
	List(context.Context, uuid.UUID, SelectorListOptions) ([]*AirMasterWaybill, int, error)
	Get(context.Context, uuid.UUID, uuid.UUID, SelectorListOptions) (*AirMasterDetail, error)
	Preview(context.Context, uuid.UUID, uuid.UUID, uint64, *AirMasterWaybillInput) (*AirMasterPreview, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, *AirMasterWaybillInput, string, string) error
	Link(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uint64, uint64, string, string) error
}
type AirExportUsecase struct{ repo AirExportRepo }

func NewAirExportUsecase(repo AirExportRepo) *AirExportUsecase { return &AirExportUsecase{repo: repo} }
func (uc *AirExportUsecase) List(ctx context.Context, org uuid.UUID, o SelectorListOptions) ([]*AirMasterWaybill, int, error) {
	if org == uuid.Nil || !ValidListPagination(o.Page, o.PageSize) {
		return nil, 0, ErrAirWaybillInvalid
	}
	o.Keyword = strings.TrimSpace(o.Keyword)
	return uc.repo.List(ctx, org, o)
}
func (uc *AirExportUsecase) Get(ctx context.Context, org, id uuid.UUID, o SelectorListOptions) (*AirMasterDetail, error) {
	if org == uuid.Nil || id == uuid.Nil || !ValidListPagination(o.Page, o.PageSize) {
		return nil, ErrAirWaybillInvalid
	}
	return uc.repo.Get(ctx, org, id, o)
}
func (uc *AirExportUsecase) Preview(ctx context.Context, org, id uuid.UUID, v uint64, input *AirMasterWaybillInput) (*AirMasterPreview, error) {
	normalized, err := NormalizeAirMasterInput(input)
	if err != nil {
		return nil, err
	}
	if v == 0 {
		return nil, ErrAirWaybillInvalid
	}
	return uc.repo.Preview(ctx, org, id, v, normalized)
}
func (uc *AirExportUsecase) Update(ctx context.Context, org, actor, id uuid.UUID, v uint64, input *AirMasterWaybillInput, fingerprint, reason string) error {
	normalized, err := NormalizeAirMasterInput(input)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if v == 0 || fingerprint == "" || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return ErrAirWaybillInvalid
	}
	return uc.repo.Update(ctx, org, actor, id, v, normalized, fingerprint, reason)
}
func (uc *AirExportUsecase) Link(ctx context.Context, org, actor, master, order uuid.UUID, mv, ov uint64, house, reason string) error {
	house, err := NormalizeAirHouseNo(house)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if mv == 0 || ov == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return ErrAirWaybillInvalid
	}
	return uc.repo.Link(ctx, org, actor, master, order, mv, ov, house, reason)
}

var airMasterNoPattern = regexp.MustCompile(`^[0-9]{11}$`)
var airHouseNoPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9 ./_-]{0,63}$`)

// NormalizeAirMasterNo 采用 3 位前缀、7 位序号、1 位模 7 校验位，保留前导零。
func NormalizeAirMasterNo(value string) (string, error) {
	value = strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	if !airMasterNoPattern.MatchString(value) {
		return "", errors.BadRequest("AIR_WAYBILL_INVALID", "主运单号必须为 3 位航司前缀加 8 位号码")
	}
	serial, _ := strconv.Atoi(value[3:10])
	if serial%7 != int(value[10]-'0') {
		return "", errors.BadRequest("AIR_WAYBILL_INVALID", "主运单号校验位不正确")
	}
	return value, nil
}
func NormalizeAirHouseNo(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !airHouseNoPattern.MatchString(value) {
		return "", errors.BadRequest("AIR_WAYBILL_INVALID", "请填写有效分运单号（最多 64 位）")
	}
	return value, nil
}
func NormalizeAirMasterInput(input *AirMasterWaybillInput) (*AirMasterWaybillInput, error) {
	if input == nil {
		return nil, ErrAirWaybillInvalid
	}
	v := *input
	var err error
	v.MasterNo, err = NormalizeAirMasterNo(v.MasterNo)
	if err != nil {
		return nil, err
	}
	if v.AirlineID == uuid.Nil || v.OriginAirportID == uuid.Nil || v.DestinationAirportID == uuid.Nil || v.OriginAirportID == v.DestinationAirportID {
		return nil, errors.BadRequest("AIR_WAYBILL_INVALID", "请选择航司、不同的起运机场和目的机场")
	}
	v.FlightNo = strings.ToUpper(strings.TrimSpace(v.FlightNo))
	v.Notes = strings.TrimSpace(v.Notes)
	if utf8.RuneCountInString(v.FlightNo) > 40 || utf8.RuneCountInString(v.Notes) > 1000 {
		return nil, ErrAirWaybillInvalid
	}
	if v.Status == "" {
		v.Status = "DRAFT"
	}
	if v.Status != "DRAFT" && v.Status != "CONFIRMED" && v.Status != "RELEASED" {
		return nil, ErrAirWaybillInvalid
	}
	for _, p := range []*string{&v.ETD, &v.ETA, &v.ActualDepartureAt, &v.ActualArrivalAt, &v.DocumentCutoffAt, &v.CustomsCutoffAt} {
		*p = strings.TrimSpace(*p)
		if *p != "" {
			t, e := time.Parse(time.RFC3339, *p)
			if e != nil {
				return nil, errors.BadRequest("AIR_WAYBILL_INVALID", "空运时间须包含时区")
			}
			*p = t.UTC().Format(time.RFC3339)
		}
	}
	if v.ETD == "" || v.ETA == "" {
		return nil, errors.BadRequest("AIR_WAYBILL_INVALID", "请填写预计起飞和到达时间")
	}
	a, _ := time.Parse(time.RFC3339, v.ETD)
	b, _ := time.Parse(time.RFC3339, v.ETA)
	if b.Before(a) {
		return nil, ErrAirWaybillInvalid
	}
	if v.ActualDepartureAt != "" && v.ActualArrivalAt != "" {
		a, _ = time.Parse(time.RFC3339, v.ActualDepartureAt)
		b, _ = time.Parse(time.RFC3339, v.ActualArrivalAt)
		if b.Before(a) {
			return nil, ErrAirWaybillInvalid
		}
	}
	if v.DeclaredPackages != nil && *v.DeclaredPackages < 0 {
		return nil, ErrAirWaybillInvalid
	}
	for _, p := range []*float64{v.DeclaredGrossWeightKg, v.DeclaredVolumeCbm} {
		if p != nil && (*p < 0 || math.IsNaN(*p) || math.IsInf(*p, 0)) {
			return nil, ErrAirWaybillInvalid
		}
	}
	return &v, nil
}

// AirMasterFingerprint 将预览绑定到版本、完整成员和提交内容。
func AirMasterFingerprint(master *AirMasterWaybill, members []*AirMasterMember, input *AirMasterWaybillInput) string {
	b, _ := json.Marshal(struct {
		Master  *AirMasterWaybill
		Members []*AirMasterMember
		Input   *AirMasterWaybillInput
	}{master, members, input})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func normalizeAirOrder(order *Order, creating bool) error {
	for _, p := range []*float64{order.TotalGrossWeightKg, order.TotalVolumeCbm} {
		if p != nil && (math.IsNaN(*p) || math.IsInf(*p, 0)) {
			return ErrAirWaybillInvalid
		}
	}
	if order.ShippingLineID != nil || order.SeaMasterBillInput != nil || order.SeaDocumentInput != nil || len(order.ContainerRequests) > 0 || len(order.ShippingDocuments) > 0 || order.ShipmentType != nil || order.ContainerOwnership != nil || order.VesselVoyage != "" {
		return ErrAirWaybillInvalid
	}
	in := order.AirExportInput
	if in == nil {
		if creating || order.AirExport == nil || order.AirExport.Master == nil {
			return errors.BadRequest("AIR_WAYBILL_INVALID", "空运建票必须填写主单号及分单号")
		}
		in = &AirOrderInput{HouseNo: order.AirExport.HouseNo, HouseContent: order.AirExport.HouseContent, HouseStatus: order.AirExport.HouseStatus, ExpectedLinkVersion: order.AirExport.LinkVersion}
	}
	v := *in
	var err error
	v.HouseNo, err = NormalizeAirHouseNo(v.HouseNo)
	if err != nil {
		return err
	}
	if utf8.RuneCountInString(v.HouseContent) > 4000 {
		return ErrAirWaybillInvalid
	}
	if v.HouseStatus == "" {
		v.HouseStatus = "DRAFT"
	}
	if v.HouseStatus != "DRAFT" && v.HouseStatus != "CONFIRMED" && v.HouseStatus != "RELEASED" {
		return ErrAirWaybillInvalid
	}
	if creating {
		if v.MasterWaybillID != nil {
			if *v.MasterWaybillID == uuid.Nil || v.ExpectedMasterVersion == 0 || v.Master == nil {
				return ErrAirWaybillInvalid
			}
			if v.Master != nil {
				v.Master, err = NormalizeAirMasterInput(v.Master)
				if err != nil {
					return err
				}
			}
		} else {
			v.Master, err = NormalizeAirMasterInput(v.Master)
			if err != nil {
				return err
			}
		}
	}
	if !creating && order.AirExport != nil && order.AirExport.Master != nil {
		master := order.AirExport.Master
		if v.MasterWaybillID != nil && *v.MasterWaybillID != master.ID {
			return errors.BadRequest("AIR_WAYBILL_INVALID", "普通保存不能更换共享主单")
		}
		if v.Master != nil {
			normalized, e := NormalizeAirMasterInput(v.Master)
			if e != nil {
				return e
			}
			a, _ := json.Marshal(normalized)
			b, _ := json.Marshal(master.Content)
			if string(a) != string(b) {
				return errors.BadRequest("AIR_WAYBILL_INVALID", "共享运输资料请在主单页面预览后修改")
			}
		}
		order.OriginLocationID = &master.Content.OriginAirportID
		order.DestinationLocationID = &master.Content.DestinationAirportID
		order.ETD = master.Content.ETD
		order.ETA = master.Content.ETA
		order.DocCutoff = master.Content.DocumentCutoffAt
		order.CustomsCutoff = master.Content.CustomsCutoffAt
	}
	if creating && v.Master != nil {
		order.OriginLocationID = &v.Master.OriginAirportID
		order.DestinationLocationID = &v.Master.DestinationAirportID
		order.ETD = v.Master.ETD
		order.ETA = v.Master.ETA
		order.DocCutoff = v.Master.DocumentCutoffAt
		order.CustomsCutoff = v.Master.CustomsCutoffAt
	}
	order.AirExportInput = &v
	return nil
}

// ValidAirFlowTransition 与海运共用状态枚举，但由空运路径显式限定准入。
func ValidAirFlowTransition(from, to OrderFlowStatus) bool { return validSEFlowTransition(from, to) }
func NewAirFlowDataError(message string) error {
	return errors.BadRequest("AIR_FLOW_DATA_MISSING", message)
}

func (uc *AirExportUsecase) LinkCandidates(ctx context.Context, org, id uuid.UUID, o SelectorListOptions) ([]*AirMasterMember, int, error) {
	if org == uuid.Nil || id == uuid.Nil || !ValidListPagination(o.Page, o.PageSize) {
		return nil, 0, ErrAirWaybillInvalid
	}
	o.Keyword = strings.TrimSpace(o.Keyword)
	return uc.repo.LinkCandidates(ctx, org, id, o)
}

type AirHouseVersion struct {
	ID         uuid.UUID
	Version    uint64
	HouseNo    string
	Status     string
	Content    string
	ActorID    uuid.UUID
	Reason     string
	OccurredAt time.Time
}

func (uc *AirExportUsecase) HouseVersions(ctx context.Context, org, id uuid.UUID, o SelectorListOptions) ([]*AirHouseVersion, int, error) {
	if org == uuid.Nil || id == uuid.Nil || !ValidListPagination(o.Page, o.PageSize) {
		return nil, 0, ErrAirWaybillInvalid
	}
	return uc.repo.HouseVersions(ctx, org, id, o)
}
