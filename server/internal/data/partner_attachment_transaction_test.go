package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func TestPartnerAttachmentRepoCreateAuditErrorRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	defer client.Close()

	repo := NewPartnerAttachmentRepo(&Data{db: client, sqlDB: db})
	organizationID, actorID, partnerID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	mock.ExpectQuery(`SELECT "partners"\."id"`).WithArgs(partnerID, organizationID).WillReturnRows(
		sqlmock.NewRows(partnerent.Columns).AddRow(partnerID, now, now, organizationID, "P001", "测试伙伴", "测试伙伴", nil, "", true, ""),
	)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "partner_attachments"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, err := repo.Create(context.Background(), organizationID, actorID, partnerID, &biz.PartnerAttachment{
		IdempotencyKey: "upload-001",
		FileName:       "合同.pdf",
		MIMEType:       "application/pdf",
		FileSize:       1024,
		ObjectKey:      "partners/contract.pdf",
	}, &biz.AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "partner.attachment.register",
		ResourceType:   "partner",
		ResourceID:     partnerID.String(),
		Result:         "success",
		Details:        map[string]string{"partner.id": partnerID.String()},
	})
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回附件: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
