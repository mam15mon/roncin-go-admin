package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

type commissionExportRepoProbe struct {
	commissionRepoStub
	countFn    func(context.Context) (int64, error)
	batchFn    func(context.Context, CommissionFilter) ([]*FinanceCommission, error)
	auditFn    func(context.Context, *AuditEvent) error
	audits     []*AuditEvent
	countCalls int
	batchCalls int
}

func (s *commissionExportRepoProbe) CountScoped(ctx context.Context, _ []uuid.UUID, _ CommissionFilter) (int64, error) {
	s.countCalls++
	if s.countFn != nil {
		return s.countFn(ctx)
	}
	return s.count, s.countErr
}
func (s *commissionExportRepoProbe) ExportBatchScoped(ctx context.Context, _ []uuid.UUID, f CommissionFilter) ([]*FinanceCommission, error) {
	s.batchCalls++
	if s.batchFn != nil {
		return s.batchFn(ctx, f)
	}
	return commissionExportBatch(1, 0), nil
}
func (s *commissionExportRepoProbe) SaveExportAudit(ctx context.Context, event *AuditEvent) error {
	s.audits = append(s.audits, event)
	if s.auditFn != nil {
		return s.auditFn(ctx, event)
	}
	return s.exportAuditErr
}

func TestCommissionExportFailureAudits(t *testing.T) {
	secret := "SELECT password='SECRET-SQL-DO-NOT-LOG'"
	original := errors.New(secret)
	for _, tt := range []struct {
		name      string
		filter    CommissionFilter
		count     int64
		countErr  error
		failBatch int
		stage     string
	}{
		{name: "非法状态", filter: CommissionFilter{Status: CommissionStatus(secret)}, stage: "validation"},
		{name: "非法日期", filter: CommissionFilter{CommissionDateFrom: secret}, stage: "validation"},
		{name: "逆序日期", filter: CommissionFilter{CommissionDateFrom: "2026-08-01", CommissionDateTo: "2026-07-01"}, stage: "validation"},
		{name: "超限", count: CommissionExportLimit + 1, stage: "limit"},
		{name: "计数故障", countErr: original, stage: "count"},
		{name: "首批故障", count: 201, failBatch: 1, stage: "batch"},
		{name: "后续批故障不交付前批", count: 201, failBatch: 2, stage: "batch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &commissionExportRepoProbe{commissionRepoStub: commissionRepoStub{count: tt.count, countErr: tt.countErr}}
			repo.batchFn = func(_ context.Context, f CommissionFilter) ([]*FinanceCommission, error) {
				if f.Page == tt.failBatch {
					return nil, original
				}
				return commissionExportBatch(200, 0), nil
			}
			tt.filter.Keyword = secret
			items, err := NewCommissionUsecase(repo, nil, nil).Export(context.Background(), uuid.New(), uuid.New(), tt.filter)
			if err == nil || items != nil || len(repo.audits) != 1 {
				t.Fatalf("items=%v err=%v audits=%d", items, err, len(repo.audits))
			}
			event := repo.audits[0]
			if event.Result != "failure" || event.Details["row_count"] != "0" || event.Details["failure_stage"] != tt.stage {
				t.Fatalf("审计=%#v", event)
			}
			details, _ := json.Marshal(event)
			if strings.Contains(string(details), secret) || strings.Contains(err.Error(), secret) || strings.Contains(kerrors.FromError(err).Message, secret) {
				t.Fatal("失败内容泄漏原始输入/驱动错误")
			}
			if (tt.countErr != nil || tt.failBatch != 0) && !errors.Is(err, original) {
				t.Fatal("原失败不可识别")
			}
		})
	}
}

func TestCommissionExportCancellationStillAudits(t *testing.T) {
	for _, when := range []string{"调用前", "计数中", "计数返回取消", "批读返回取消", "批次之间", "末批完成后", "超时"} {
		t.Run(when, func(t *testing.T) {
			type contextKey struct{}
			base := context.WithValue(context.Background(), contextKey{}, "保留元数据")
			ctx, cancel := context.WithCancel(base)
			defer cancel()
			expected := error(context.Canceled)
			if when == "超时" {
				var deadlineCancel context.CancelFunc
				ctx, deadlineCancel = context.WithDeadline(base, time.Now().Add(-time.Second))
				defer deadlineCancel()
				expected = context.DeadlineExceeded
			}
			if when == "调用前" {
				cancel()
			}
			repo := &commissionExportRepoProbe{}
			repo.countFn = func(context.Context) (int64, error) {
				if when == "计数返回取消" {
					return 0, context.Canceled
				}
				if when == "计数中" {
					cancel()
				}
				if when == "批次之间" {
					return 201, nil
				}
				return 1, nil
			}
			repo.batchFn = func(context.Context, CommissionFilter) ([]*FinanceCommission, error) {
				if when == "批读返回取消" {
					return nil, context.Canceled
				}
				cancel()
				return commissionExportBatch(1, 0), nil
			}
			repo.auditFn = func(auditCtx context.Context, _ *AuditEvent) error {
				if auditCtx.Err() != nil || auditCtx.Value(contextKey{}) != "保留元数据" {
					t.Fatal("审计继承取消或丢失元数据")
				}
				deadline, ok := auditCtx.Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 5*time.Second {
					t.Fatalf("审计期限=%v", deadline)
				}
				return nil
			}
			items, err := NewCommissionUsecase(repo, nil, nil).Export(ctx, uuid.New(), uuid.New(), CommissionFilter{})
			if items != nil || !errors.Is(err, expected) || len(repo.audits) != 1 {
				t.Fatalf("items=%v err=%v audits=%d", items, err, len(repo.audits))
			}
			stage := "cancelled"
			if when == "超时" {
				stage = "deadline"
			}
			if repo.audits[0].Details["failure_stage"] != stage {
				t.Fatalf("审计=%#v", repo.audits[0])
			}
			if when == "批次之间" && repo.batchCalls != 1 {
				t.Fatalf("取消后仍发起后续批读取: %d", repo.batchCalls)
			}
		})
	}
}

func TestCommissionExportAuditErrorChainAndTransportSafety(t *testing.T) {
	raw := errors.New("SQL SELECT SECRET-DB-PASSWORD")
	auditRaw := errors.New("INSERT audit SECRET-AUDIT-TOKEN")
	for _, tt := range []struct {
		name       string
		business   error
		wantCode   int
		wantReason string
	}{
		{name: "成功时审计失败", wantCode: 500},
		{name: "领域失败且审计失败", business: ErrCommissionInvalid, wantCode: 400, wantReason: ErrCommissionInvalid.Reason},
		{name: "未知失败且审计失败", business: raw, wantCode: 500},
		{name: "领域带原始cause且审计失败", business: ErrCommissionInvalid.WithCause(raw), wantCode: 400, wantReason: ErrCommissionInvalid.Reason},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &commissionExportRepoProbe{commissionRepoStub: commissionRepoStub{countErr: tt.business, exportAuditErr: auditRaw}}
			var logs bytes.Buffer
			handler := requestmeta.Logging(slog.New(slog.NewJSONHandler(&logs, nil)))(func(ctx context.Context, _ any) (any, error) {
				return NewCommissionUsecase(repo, nil, nil).Export(ctx, uuid.New(), uuid.New(), CommissionFilter{})
			})
			result, err := handler(context.Background(), nil)
			items, _ := result.([]*FinanceCommission)
			if items != nil || err == nil || !errors.Is(err, auditRaw) || !errors.Is(err, errCommissionExportAudit) {
				t.Fatalf("items=%v err=%v", items, err)
			}
			if tt.business != nil && !errors.Is(err, tt.business) {
				t.Fatal("业务失败被吞")
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v1/finance/commissions/export", nil)
			request.Header.Set("Accept", "application/json")
			khttp.DefaultErrorEncoder(response, request, err)
			status := kerrors.FromError(err)
			if response.Code != tt.wantCode || status.Reason != tt.wantReason {
				t.Fatalf("响应=%s status=%v", response.Body.String(), status)
			}
			for _, text := range []string{err.Error(), status.Error(), logs.String(), response.Body.String()} {
				if strings.Contains(text, "SECRET") || strings.Contains(text, "SELECT") || strings.Contains(text, "INSERT") {
					t.Fatalf("错误泄漏: %s", text)
				}
			}
		})
	}
}

func TestCommissionExportInvalidIdentityDoesNotAudit(t *testing.T) {
	org := uuid.New()
	for _, ids := range [][]uuid.UUID{nil, {uuid.Nil}, {org, org}} {
		repo := &commissionExportRepoProbe{}
		_, err := NewCommissionUsecase(repo, nil, nil).ExportScoped(context.Background(), ids, uuid.New(), CommissionFilter{})
		if !errors.Is(err, ErrCommissionInvalid) || repo.countCalls != 0 || len(repo.audits) != 0 {
			t.Fatalf("err=%v audits=%d", err, len(repo.audits))
		}
	}
	repo := &commissionExportRepoProbe{}
	_, err := NewCommissionUsecase(repo, nil, nil).Export(context.Background(), org, uuid.Nil, CommissionFilter{})
	if !errors.Is(err, ErrCommissionInvalid) || repo.countCalls != 0 || len(repo.audits) != 0 {
		t.Fatal("空身份仍执行导出")
	}
}
