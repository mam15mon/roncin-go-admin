package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type correctionPaginationRepo struct {
	OrderFeeCorrectionRepo
	authorized, listed bool
	page, size         int
}

func (r *correctionPaginationRepo) Authorize(context.Context, *Principal, uuid.UUID, uuid.UUID, string) error {
	r.authorized = true
	return nil
}
func (r *correctionPaginationRepo) List(_ context.Context, _, _ uuid.UUID, page, size int) ([]*OrderFeeCorrectionRequest, int, error) {
	r.listed, r.page, r.size = true, page, size
	return []*OrderFeeCorrectionRequest{}, 0, nil
}

func TestOrderFeeCorrectionListPaginationBounds(t *testing.T) {
	for _, s := range []struct {
		name       string
		page, size int
		valid      bool
	}{
		{"页码零", 0, 20, false}, {"负页码", -1, 20, false},
		{"页容量零", 1, 0, false}, {"负页容量", 1, -1, false},
		{"页容量超过上限", 1, MaxListPageSize + 1, false},
		{"最小页容量", 1, 1, true}, {"最大页容量", 2, MaxListPageSize, true},
	} {
		t.Run(s.name, func(t *testing.T) {
			repo := &correctionPaginationRepo{}
			uc := NewOrderFeeCorrectionUsecase(repo, nil, nil, nil)
			_, _, err := uc.List(context.Background(), &Principal{UserID: uuid.New()}, uuid.New(), uuid.New(), s.page, s.size)
			if !s.valid {
				if !errors.Is(err, ErrFeeCorrectionInvalid) || repo.authorized || repo.listed {
					t.Fatalf("非法分页不得读授权或访问仓储：error=%v repo=%+v", err, repo)
				}
			} else if err != nil || !repo.authorized || !repo.listed || repo.page != s.page || repo.size != s.size {
				t.Fatalf("合法分页需原样传递：error=%v repo=%+v", err, repo)
			}
		})
	}
}
