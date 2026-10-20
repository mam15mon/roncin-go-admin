package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type airWeightServiceRepo struct {
	input      *biz.AirWeightInput
	org, actor uuid.UUID
}

func (r *airWeightServiceRepo) Preview(_ context.Context, org uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightPreview, error) {
	r.input = input
	r.org = org
	return &biz.AirWeightPreview{Scope: input.Scope, TargetID: input.TargetID, Source: input.Source, InputFingerprint: "input-hash", Available: true, SuggestedChargeableWeightKG: "166.667", VolumetricDivisor: input.VolumetricDivisor, BubbleShareNumerator: input.BubbleShareNumerator, BubbleShareDenominator: input.BubbleShareDenominator}, nil
}
func (r *airWeightServiceRepo) Confirm(_ context.Context, org, actor uuid.UUID, input *biz.AirWeightInput) (*biz.AirWeightConfirmation, error) {
	r.input = input
	r.org = org
	r.actor = actor
	return &biz.AirWeightConfirmation{ID: uuid.New(), Scope: input.Scope, TargetID: input.TargetID, Source: input.Source, Version: input.ExpectedVersion + 1, InputFingerprint: input.InputFingerprint, ConfirmedWeightKG: input.ConfirmedWeightKG, ConfirmedBy: actor, ConfirmedAt: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), Reason: input.Reason}, nil
}
func (r *airWeightServiceRepo) List(_ context.Context, org uuid.UUID, scope string, id uuid.UUID, page, size int) ([]*biz.AirWeightConfirmation, int, error) {
	r.org = org
	return nil, 0, nil
}
func TestAirWeightServiceBoundary(t *testing.T) {
	repo := &airWeightServiceRepo{}
	s := NewAirWeightService(biz.NewAirWeightUsecase(repo))
	id, org, actor := uuid.New(), uuid.New(), uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: actor, Organization: biz.Organization{ID: org}})
	req := &v1.PreviewAirWeightRequest{Scope: "ORDER", TargetId: id.String(), Source: "PLANNED"}
	if _, e := s.PreviewAirWeight(context.Background(), req); e == nil {
		t.Fatal("未登录必须拒绝")
	}
	if _, e := s.PreviewAirWeight(ctx, &v1.PreviewAirWeightRequest{Scope: "ORDER", TargetId: "bad"}); e != biz.ErrAirWeightInvalid {
		t.Fatal(e)
	}
	p, e := s.PreviewAirWeight(ctx, req)
	if e != nil || !p.Success || p.Data.Source != "PLANNED" || p.Data.SuggestedChargeableWeightKg != "166.667" || repo.input.VolumetricDivisor != "6000" || repo.input.BubbleShareNumerator != 1 || repo.input.BubbleShareDenominator != 1 || repo.org != org {
		t.Fatalf("请求范围与默认规则转换: %+v %v", p, e)
	}
	confirmation, e := s.ConfirmAirWeight(ctx, &v1.ConfirmAirWeightRequest{Scope: "ORDER", TargetId: id.String(), Source: "PLANNED", InputFingerprint: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ConfirmedWeightKg: "125", ExpectedVersion: 2, Reason: "客户确认125"})
	if e != nil || !confirmation.Success || confirmation.Data.Version != 3 || confirmation.Data.ConfirmedWeightKg != "125" || repo.actor != actor || repo.input.Reason != "客户确认125" {
		t.Fatalf("确认协议转换: %+v %v", confirmation, e)
	}
	if _, e := s.ConfirmAirWeight(ctx, &v1.ConfirmAirWeightRequest{Scope: "ORDER", TargetId: id.String(), InputFingerprint: "old", ConfirmedWeightKg: "125"}); e != biz.ErrAirWeightInvalid {
		t.Fatalf("无有效指纹不得执行: %v", e)
	}
	if _, e := s.ListAirWeightConfirmations(ctx, &v1.ListAirWeightConfirmationsRequest{Scope: "ORDER", TargetId: id.String(), Page: 1, PageSize: 201}); e != biz.ErrAirWeightInvalid {
		t.Fatalf("分页统一上限200: %v", e)
	}
}
