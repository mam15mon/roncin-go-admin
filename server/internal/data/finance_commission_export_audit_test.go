package data

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"go.opentelemetry.io/otel/trace"
)

func TestCommissionRepoSaveExportAuditRejectsSharedTransaction(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	event := &biz.AuditEvent{Action: "finance.commission.export", Result: "failure"}
	mock.ExpectBegin()
	mock.ExpectRollback()
	var closed context.Context
	err := repo.data.WithinTransaction(context.Background(), func(ctx context.Context) error {
		closed = ctx
		if err := repo.SaveExportAudit(context.WithoutCancel(ctx), event); !errors.Is(err, errCommissionExportAuditTransaction) {
			t.Fatalf("共享事务错误=%v", err)
		}
		return errCommissionExportAuditTransaction
	})
	if !errors.Is(err, errCommissionExportAuditTransaction) {
		t.Fatalf("事务结果=%v", err)
	}
	if err := repo.SaveExportAudit(context.WithoutCancel(closed), event); !errors.Is(err, errTransactionContextClosed) {
		t.Fatalf("结束事务错误=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("共享事务拒绝仍发起审计SQL: %v", err)
	}
}

type commissionAuditHeader struct{ http.Header }

func (h commissionAuditHeader) Keys() []string {
	keys := make([]string, 0, len(h.Header))
	for key := range h.Header {
		keys = append(keys, key)
	}
	return keys
}

type commissionAuditHTTPTransport struct {
	transport.Transporter
	request *http.Request
	reply   http.Header
}

func (t *commissionAuditHTTPTransport) RequestHeader() transport.Header {
	return commissionAuditHeader{t.request.Header}
}
func (t *commissionAuditHTTPTransport) ReplyHeader() transport.Header {
	return commissionAuditHeader{t.reply}
}
func (t *commissionAuditHTTPTransport) Request() *http.Request { return t.request }

func TestCommissionRepoSaveExportAuditAfterCancellationKeepsMetadata(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	request := httptest.NewRequest("GET", "/api/v1/finance/commissions/export", nil)
	request.Header.Set("X-Request-ID", "commission-export-cancelled")
	request.RemoteAddr = "192.0.2.10:4321"
	traceID := trace.TraceID{1, 2, 3}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: trace.SpanID{1}, TraceFlags: trace.FlagsSampled}))
	ctx = transport.NewServerContext(ctx, &commissionAuditHTTPTransport{request: request, reply: make(http.Header)})
	handler := requestmeta.Middleware()(func(ctx context.Context, _ any) (any, error) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		auditCtx, auditCancel := context.WithTimeout(context.WithoutCancel(cancelled), 5*time.Second)
		defer auditCancel()
		if auditCtx.Err() != nil {
			t.Fatal("审计上下文仍被请求取消")
		}
		id, trace, ip, _, err := resolveAuditValues(auditCtx, &biz.AuditEvent{})
		if err != nil || id != "commission-export-cancelled" || trace != traceID.String() || ip != "192.0.2.10" {
			t.Fatalf("元数据=%s/%s/%s err=%v", id, trace, ip, err)
		}
		return nil, repo.SaveExportAudit(auditCtx, &biz.AuditEvent{Action: "finance.commission.export", Result: "failure"})
	})
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnResult(sqlmock.NewResult(1, 1))
	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("取消后审计失败=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommissionRepoSaveExportAuditReturnsOwnDeadlineAndWriteFailure(t *testing.T) {
	t.Run("自身已超时", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		err := repo.SaveExportAudit(ctx, &biz.AuditEvent{Action: "finance.commission.export", Result: "failure"})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("审计期限错误=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("写入失败", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		cause := errors.New("审计存储故障")
		mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(cause)
		err := repo.SaveExportAudit(context.Background(), &biz.AuditEvent{Action: "finance.commission.export", Result: "failure"})
		if !errors.Is(err, cause) {
			t.Fatalf("审计写入失败被吞=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}
