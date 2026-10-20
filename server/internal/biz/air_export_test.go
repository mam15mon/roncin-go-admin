package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAirMasterAndHouseNumberContract(t *testing.T) {
	for _, v := range []string{"781-12345675", " 78112345675 "} {
		got, e := NormalizeAirMasterNo(v)
		if e != nil || got != "78112345675" {
			t.Fatalf("规范主单号 %q: %q %v", v, got, e)
		}
	}
	for _, v := range []string{"", "78112345678", "1234567", "abc"} {
		if _, e := NormalizeAirMasterNo(v); e == nil {
			t.Fatalf("非法主单放行 %q", v)
		}
	}
	got, e := NormalizeAirHouseNo(" h001 ")
	if e != nil || got != "H001" {
		t.Fatalf("分单规范化 %q %v", got, e)
	}
	for _, v := range []string{"", "\n", "<script>"} {
		if _, e := NormalizeAirHouseNo(v); e == nil {
			t.Fatalf("非法分单放行 %q", v)
		}
	}
}
func TestAirMasterTimeAndAirportValidation(t *testing.T) {
	in := &AirMasterWaybillInput{MasterNo: "78112345675", AirlineID: uuid.New(), OriginAirportID: uuid.New(), DestinationAirportID: uuid.New(), ETD: "2026-10-22T08:00:00+08:00", ETA: "2026-10-22T15:00:00+08:00"}
	out, e := NormalizeAirMasterInput(in)
	if e != nil || out.ETD != "2026-10-22T00:00:00Z" {
		t.Fatalf("时区归一 %v %v", out, e)
	}
	bad := *in
	bad.ETA = "2026-10-21T12:00:00Z"
	if _, e := NormalizeAirMasterInput(&bad); e == nil {
		t.Fatal("ETA 早于 ETD 放行")
	}
	bad = *in
	bad.DestinationAirportID = bad.OriginAirportID
	if _, e := NormalizeAirMasterInput(&bad); e == nil {
		t.Fatal("相同机场放行")
	}
}

type airExportPaginationRepo struct {
	AirExportRepo
	called  bool
	options SelectorListOptions
}

func (r *airExportPaginationRepo) List(_ context.Context, _ uuid.UUID, options SelectorListOptions) ([]*AirMasterWaybill, int, error) {
	r.called, r.options = true, options
	return nil, 0, nil
}
func (r *airExportPaginationRepo) Get(_ context.Context, _, _ uuid.UUID, options SelectorListOptions) (*AirMasterDetail, error) {
	r.called, r.options = true, options
	return &AirMasterDetail{}, nil
}
func (r *airExportPaginationRepo) LinkCandidates(_ context.Context, _, _ uuid.UUID, options SelectorListOptions) ([]*AirMasterMember, int, error) {
	r.called, r.options = true, options
	return nil, 0, nil
}
func (r *airExportPaginationRepo) HouseVersions(_ context.Context, _, _ uuid.UUID, options SelectorListOptions) ([]*AirHouseVersion, int, error) {
	r.called, r.options = true, options
	return nil, 0, nil
}

func TestAirExportPaginationBounds(t *testing.T) {
	org, id := uuid.New(), uuid.New()
	for _, endpoint := range []struct {
		name string
		run  func(*AirExportUsecase, SelectorListOptions) error
	}{
		{"主单列表", func(uc *AirExportUsecase, options SelectorListOptions) error {
			_, _, err := uc.List(context.Background(), org, options)
			return err
		}},
		{"主单成员与历史", func(uc *AirExportUsecase, options SelectorListOptions) error {
			_, err := uc.Get(context.Background(), org, id, options)
			return err
		}},
		{"恢复关联候选", func(uc *AirExportUsecase, options SelectorListOptions) error {
			_, _, err := uc.LinkCandidates(context.Background(), org, id, options)
			return err
		}},
		{"分单历史", func(uc *AirExportUsecase, options SelectorListOptions) error {
			_, _, err := uc.HouseVersions(context.Background(), org, id, options)
			return err
		}},
	} {
		for _, scenario := range []struct {
			name       string
			page, size int
			valid      bool
		}{
			{"页码零", 0, 20, false}, {"负页码", -1, 20, false},
			{"页容量零", 1, 0, false}, {"负页容量", 1, -1, false},
			{"页容量超过上限", 1, MaxListPageSize + 1, false},
			{"最小页容量", 1, 1, true}, {"最大页容量", 2, MaxListPageSize, true},
		} {
			t.Run(endpoint.name+"/"+scenario.name, func(t *testing.T) {
				repo := &airExportPaginationRepo{}
				options := SelectorListOptions{Page: scenario.page, PageSize: scenario.size}
				err := endpoint.run(NewAirExportUsecase(repo), options)
				if !scenario.valid {
					if !errors.Is(err, ErrAirWaybillInvalid) || repo.called {
						t.Fatalf("非法分页不得访问仓储：error=%v repo=%+v", err, repo)
					}
				} else if err != nil || !repo.called || repo.options != options {
					t.Fatalf("合法分页需原样传递：error=%v repo=%+v", err, repo)
				}
			})
		}
	}
}
