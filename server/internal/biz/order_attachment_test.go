package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type orderAttachmentRepoStub struct {
	created *OrderAttachment
}

func (s *orderAttachmentRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderAttachment, error) {
	return nil, nil
}

func (s *orderAttachmentRepoStub) Create(_ context.Context, _, actorID, orderID uuid.UUID, input *OrderAttachment) (*OrderAttachment, error) {
	s.created = input
	input.ID = uuid.New()
	input.OrderID = orderID
	input.UploadedBy = &actorID
	return input, nil
}

func TestOrderAttachmentRegisterNormalizesAndAudits(t *testing.T) {
	repo := &orderAttachmentRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderAttachmentUsecase(repo, audit)
	organizationID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()

	created, err := usecase.Register(context.Background(), organizationID, actorID, orderID, &OrderAttachment{
		DocType:        "  BL ",
		IdempotencyKey: "  upload-order-001 ",
		FileName:       " 提单.pdf ",
		MIMEType:       " application/pdf ",
		FileSize:       2048,
		ObjectKey:      " orders/bl-001 ",
		Checksum:       " sha256:fedcba ",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if created.DocType != "BL" || created.IdempotencyKey != "upload-order-001" || created.FileName != "提单.pdf" || created.MIMEType != "application/pdf" || created.ObjectKey != "orders/bl-001" || created.Checksum != "sha256:fedcba" || created.UploadedBy == nil || *created.UploadedBy != actorID {
		t.Fatalf("normalized attachment = %#v", created)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "order.attachment.register" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestOrderAttachmentRejectsUnsafeMetadata(t *testing.T) {
	usecase := NewOrderAttachmentUsecase(&orderAttachmentRepoStub{}, &auditRepoStub{})
	cases := []*OrderAttachment{
		{DocType: "BL", IdempotencyKey: "k", FileName: "../secret.pdf", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "foo/bar.pdf", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "foo\\bar.pdf", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 101 << 20, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 0, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: -1, ObjectKey: "key"},
		{DocType: "BL", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key\n"},
		{DocType: "BL\x00", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
		{DocType: "   ", IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
	}
	for index, input := range cases {
		if _, err := usecase.Register(context.Background(), uuid.New(), uuid.New(), uuid.New(), input); err != ErrOrderAttachmentInvalidArgument {
			t.Fatalf("case %d error = %v, want ErrOrderAttachmentInvalidArgument", index, err)
		}
	}
}

var _ OrderAttachmentRepo = (*orderAttachmentRepoStub)(nil)
