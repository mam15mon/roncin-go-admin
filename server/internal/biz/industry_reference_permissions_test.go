package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

// 嵌入未消费接口，确保只有授权通过的港口机场写入才会到达仓储。
type publicLocationWriteRepo struct {
	IndustryReferenceRepo
	calls int
}

func (r *publicLocationWriteRepo) CreatePort(_ context.Context, _ uuid.UUID, p *Port, _ *AuditEvent) (*Port, error) {
	r.calls++
	return p, nil
}
func (r *publicLocationWriteRepo) UpdatePort(_ context.Context, _, _ uuid.UUID, p *Port, _ *AuditEvent) (*Port, error) {
	r.calls++
	return p, nil
}
func (r *publicLocationWriteRepo) CreateAirport(_ context.Context, _ uuid.UUID, p *Airport, _ *AuditEvent) (*Airport, error) {
	r.calls++
	return p, nil
}
func (r *publicLocationWriteRepo) UpdateAirport(_ context.Context, _, _ uuid.UUID, p *Airport, _ *AuditEvent) (*Airport, error) {
	r.calls++
	return p, nil
}

func TestPublicLocationWriteRequiresSystemWorkspaceAndPermission(t *testing.T) {
	actor, id := uuid.New(), uuid.New()
	port := &Port{UNLocode: "CNSHA", NameEN: "Shanghai", NameZH: "上海港", CountryCode: "CN", TransportModes: []string{"SEA"}}
	airport := &Airport{IATACode: "PVG", NameEN: "Pudong", NameZH: "浦东机场", CityNameZH: "上海", CountryCode: "CN"}
	cases := []struct {
		name, permission string
		run              func(*IndustryReferenceUsecase, context.Context, uuid.UUID) error
	}{
		{"创建港口", access.MasterDataPortCreate, func(uc *IndustryReferenceUsecase, ctx context.Context, org uuid.UUID) error {
			_, e := uc.CreatePort(ctx, org, actor, port)
			return e
		}},
		{"更新港口", access.MasterDataPortUpdate, func(uc *IndustryReferenceUsecase, ctx context.Context, org uuid.UUID) error {
			_, e := uc.UpdatePort(ctx, org, actor, id, port)
			return e
		}},
		{"创建机场", access.MasterDataAirportCreate, func(uc *IndustryReferenceUsecase, ctx context.Context, org uuid.UUID) error {
			_, e := uc.CreateAirport(ctx, org, actor, airport)
			return e
		}},
		{"更新机场", access.MasterDataAirportUpdate, func(uc *IndustryReferenceUsecase, ctx context.Context, org uuid.UUID) error {
			_, e := uc.UpdateAirport(ctx, org, actor, id, airport)
			return e
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, p := range []*Principal{companyPrincipal(c.permission), headquartersPrincipal()} {
				repo := &publicLocationWriteRepo{}
				err := c.run(NewIndustryReferenceUsecase(repo), principalContext(p), p.Organization.ID)
				if err == nil || repo.calls != 0 {
					t.Fatalf("公司身份或缺少权限必须在仓储之前拒绝: err=%v calls=%d", err, repo.calls)
				}
			}
			p := headquartersPrincipal(c.permission)
			repo := &publicLocationWriteRepo{}
			if err := c.run(NewIndustryReferenceUsecase(repo), principalContext(p), p.Organization.ID); err != nil || repo.calls != 1 {
				t.Fatalf("系统身份且持权应允许写入: err=%v calls=%d", err, repo.calls)
			}
		})
	}
}
