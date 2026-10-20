package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrAirWeightInvalid        = errors.BadRequest("AIR_WEIGHT_INVALID", "空运重量参数不合法")
	ErrAirWeightConflict       = errors.Conflict("AIR_WEIGHT_CONFLICT", "重量依据已被更新，请刷新后重试")
	ErrAirWeightNotFound       = errors.NotFound("AIR_WEIGHT_NOT_FOUND", "空运重量对象不存在")
	ErrAirWeightReasonRequired = errors.BadRequest("AIR_WEIGHT_REASON_REQUIRED", "人工调整或缺少参考时必须填写确认依据")
)

type AirWeightInput struct {
	Scope                  string
	TargetID               uuid.UUID
	Source                 string
	VolumetricDivisor      string
	BubbleShareNumerator   int32
	BubbleShareDenominator int32
	InputFingerprint       string
	ExpectedVersion        uint64
	ConfirmedWeightKG      string
	Reason                 string
}
type AirWeightFacts struct {
	GrossWeightKG string
	VolumeCBM     string
	HasCargo      bool
	// Identity 包含完整成员与货物行版本，独立于本票无关私有字段修改。
	Identity string
}
type AirWeightConfirmation struct {
	ID                uuid.UUID
	Scope             string
	TargetID          uuid.UUID
	Source            string
	Version           uint64
	InputFingerprint  string
	ConfirmedWeightKG string
	Reason            string
	ConfirmedBy       uuid.UUID
	ConfirmedAt       time.Time
	Snapshot          map[string]string
}
type AirWeightPreview struct {
	Scope                       string
	TargetID                    uuid.UUID
	Source                      string
	InputFingerprint            string
	Available                   bool
	Reason                      string
	GrossWeightKG               string
	VolumeCBM                   string
	VolumetricWeightKG          string
	BubbleDifferenceKG          string
	SharedBubbleWeightKG        string
	SuggestedChargeableWeightKG string
	VolumetricDivisor           string
	BubbleShareNumerator        int32
	BubbleShareDenominator      int32
	CurrentConfirmation         *AirWeightConfirmation
	ConfirmationStale           bool
	InputIdentity               string
}
type AirWeightRepo interface {
	Preview(context.Context, uuid.UUID, *AirWeightInput) (*AirWeightPreview, error)
	Confirm(context.Context, uuid.UUID, uuid.UUID, *AirWeightInput) (*AirWeightConfirmation, error)
	List(context.Context, uuid.UUID, string, uuid.UUID, int, int) ([]*AirWeightConfirmation, int, error)
}
type AirWeightUsecase struct{ repo AirWeightRepo }

func NewAirWeightUsecase(repo AirWeightRepo) *AirWeightUsecase { return &AirWeightUsecase{repo: repo} }
func NormalizeAirWeightInput(input *AirWeightInput) (*AirWeightInput, error) {
	if input == nil || (input.Scope != "ORDER" && input.Scope != "MASTER") || input.TargetID == uuid.Nil {
		return nil, ErrAirWeightInvalid
	}
	out := *input
	if out.Source == "" {
		out.Source = "ACTUAL"
	}
	if out.Source != "ACTUAL" && out.Source != "PLANNED" {
		return nil, ErrAirWeightInvalid
	}
	if out.VolumetricDivisor == "" {
		out.VolumetricDivisor = "6000"
	}
	d, err := decimal.NewFromString(out.VolumetricDivisor)
	if err != nil || !quantityOrPricePattern.MatchString(out.VolumetricDivisor) || !d.IsPositive() {
		return nil, ErrAirWeightInvalid
	}
	out.VolumetricDivisor = d.String()
	if out.BubbleShareDenominator == 0 && out.BubbleShareNumerator == 0 {
		out.BubbleShareNumerator = 1
		out.BubbleShareDenominator = 1
	}
	if out.BubbleShareDenominator <= 0 || out.BubbleShareNumerator < 0 || out.BubbleShareNumerator > out.BubbleShareDenominator {
		return nil, ErrAirWeightInvalid
	}
	out.Reason = strings.TrimSpace(out.Reason)
	if utf8.RuneCountInString(out.Reason) > 500 {
		return nil, ErrAirWeightInvalid
	}
	return &out, nil
}
func (uc *AirWeightUsecase) Preview(ctx context.Context, org uuid.UUID, input *AirWeightInput) (*AirWeightPreview, error) {
	n, e := NormalizeAirWeightInput(input)
	if e != nil {
		return nil, e
	}
	return uc.repo.Preview(ctx, org, n)
}
func (uc *AirWeightUsecase) Confirm(ctx context.Context, org, actor uuid.UUID, input *AirWeightInput) (*AirWeightConfirmation, error) {
	n, e := NormalizeAirWeightInput(input)
	if e != nil {
		return nil, e
	}
	weight, err := decimal.NewFromString(n.ConfirmedWeightKG)
	if org == uuid.Nil || actor == uuid.Nil || len(n.InputFingerprint) != 64 || err != nil || !weight.IsPositive() || !quantityOrPricePattern.MatchString(n.ConfirmedWeightKG) {
		return nil, ErrAirWeightInvalid
	}
	n.ConfirmedWeightKG = weight.String()
	return uc.repo.Confirm(ctx, org, actor, n)
}
func (uc *AirWeightUsecase) List(ctx context.Context, org uuid.UUID, scope string, id uuid.UUID, page, size int) ([]*AirWeightConfirmation, int, error) {
	if org == uuid.Nil || id == uuid.Nil || (scope != "ORDER" && scope != "MASTER") || !ValidListPagination(page, size) {
		return nil, 0, ErrAirWeightInvalid
	}
	return uc.repo.List(ctx, org, scope, id, page, size)
}

// BuildAirWeightPreview 中间计算使用有理数；只在最终参考值执行十进制 HALF_UP。
func BuildAirWeightPreview(input *AirWeightInput, facts *AirWeightFacts, current *AirWeightConfirmation) (*AirWeightPreview, error) {
	normalized, err := NormalizeAirWeightInput(input)
	if err != nil {
		return nil, err
	}
	input = normalized
	p := &AirWeightPreview{Scope: input.Scope, TargetID: input.TargetID, Source: input.Source, VolumetricDivisor: input.VolumetricDivisor, BubbleShareNumerator: input.BubbleShareNumerator, BubbleShareDenominator: input.BubbleShareDenominator, CurrentConfirmation: current}
	p.GrossWeightKG = facts.GrossWeightKG
	p.VolumeCBM = facts.VolumeCBM
	p.InputIdentity = facts.Identity
	data, _ := json.Marshal([]any{input.Scope, input.TargetID.String(), input.Source, facts.Identity, facts.GrossWeightKG, facts.VolumeCBM, input.VolumetricDivisor, input.BubbleShareNumerator, input.BubbleShareDenominator})
	hash := sha256.Sum256(data)
	p.InputFingerprint = hex.EncodeToString(hash[:])
	p.ConfirmationStale = current != nil && current.InputFingerprint != p.InputFingerprint
	if !facts.HasCargo {
		p.Reason = "当前来源的货物重量或体积不完整，请补充资料或填写人工确认依据"
		return p, nil
	}
	g, ok := new(big.Rat).SetString(facts.GrossWeightKG)
	if !ok || g.Sign() <= 0 {
		return nil, ErrAirWeightInvalid
	}
	v, ok := new(big.Rat).SetString(facts.VolumeCBM)
	if !ok || v.Sign() <= 0 {
		p.Reason = "缺少有效体积，不能计算参考计费重"
		return p, nil
	}
	d, _ := new(big.Rat).SetString(input.VolumetricDivisor)
	vw := new(big.Rat).Quo(new(big.Rat).Mul(v, big.NewRat(1000000, 1)), d)
	difference := new(big.Rat).Sub(vw, g)
	if difference.Sign() < 0 {
		difference.SetInt64(0)
	}
	share := new(big.Rat).Mul(difference, big.NewRat(int64(input.BubbleShareNumerator), int64(input.BubbleShareDenominator)))
	suggested := new(big.Rat).Add(g, share)
	p.VolumetricWeightKG = roundPositiveRat(vw, 6)
	p.BubbleDifferenceKG = roundPositiveRat(difference, 6)
	p.SharedBubbleWeightKG = roundPositiveRat(share, 6)
	p.SuggestedChargeableWeightKG = roundPositiveRat(suggested, 3)
	value, _ := decimal.NewFromString(p.SuggestedChargeableWeightKG)
	if !value.IsPositive() || !quantityOrPricePattern.MatchString(value.String()) {
		p.Reason = "参考计费重超过费用数量允许范围，请核对输入"
		return p, nil
	}
	p.Available = true
	return p, nil
}
func roundPositiveRat(value *big.Rat, places int32) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	numerator := new(big.Int).Mul(value.Num(), scale)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(numerator, value.Denom(), r)
	if new(big.Int).Lsh(r, 1).Cmp(value.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return decimal.NewFromBigInt(q, -places).String()
}
func (p *AirWeightPreview) Snapshot() map[string]string {
	return map[string]string{
		"input_identity": p.InputIdentity, "weight_unit": "KG", "volume_unit": "CBM", "source": p.Source, "gross_weight_kg": p.GrossWeightKG, "volume_cbm": p.VolumeCBM, "volumetric_divisor": p.VolumetricDivisor,
		"bubble_share_numerator": big.NewInt(int64(p.BubbleShareNumerator)).String(), "bubble_share_denominator": big.NewInt(int64(p.BubbleShareDenominator)).String(),
		"volumetric_weight_kg": p.VolumetricWeightKG, "bubble_difference_kg": p.BubbleDifferenceKG, "shared_bubble_weight_kg": p.SharedBubbleWeightKG, "suggested_chargeable_weight_kg": p.SuggestedChargeableWeightKG, "rounding": "HALF_UP_0.001_KG", "rule_version": "1",
	}
}
