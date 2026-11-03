package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type orderAttachmentRepoStub struct {
	created *OrderAttachment
	audit   *AuditEvent
}

func (s *orderAttachmentRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderAttachment, error) {
	return nil, nil
}

func (s *orderAttachmentRepoStub) Create(_ context.Context, _, actorID, orderID uuid.UUID, input *OrderAttachment, audit *AuditEvent) (*OrderAttachment, error) {
	s.created = input
	s.audit = audit
	input.ID = uuid.New()
	input.OrderID = orderID
	input.UploadedBy = &actorID
	return input, nil
}

func (s *orderAttachmentRepoStub) RemoveReference(_ context.Context, _, _, _, _ uuid.UUID, audit *AuditEvent) error {
	s.audit = audit
	return nil
}

func TestOrderAttachmentRegisterNormalizesAndAudits(t *testing.T) {
	repo := &orderAttachmentRepoStub{}
	usecase := NewOrderAttachmentUsecase(repo, &attachmentStorageStub{})
	organizationID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()

	storage := usecase.storage.(*attachmentStorageStub)
	storage.attachment = &OrderAttachment{DocType: "  BL ", IdempotencyKey: "  upload-order-001 ", FileName: " 提单.pdf ", MIMEType: " application/pdf ", FileSize: 2048, ObjectKey: " orders/bl-001 ", Checksum: " sha256:fedcba "}
	created, err := usecase.Register(context.Background(), organizationID, actorID, orderID, "valid-token")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if created.DocType != "BL" || created.IdempotencyKey != "upload-order-001" || created.FileName != "提单.pdf" || created.MIMEType != "application/pdf" || created.ObjectKey != "orders/bl-001" || created.Checksum != "sha256:fedcba" || created.UploadedBy == nil || *created.UploadedBy != actorID {
		t.Fatalf("normalized attachment = %#v", created)
	}
	if repo.audit == nil || repo.audit.Action != "order.attachment.register" {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

func TestOrderAttachmentRejectsUnsafeMetadata(t *testing.T) {
	usecase := NewOrderAttachmentUsecase(&orderAttachmentRepoStub{}, &attachmentStorageStub{})
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
		usecase.storage.(*attachmentStorageStub).attachment = input
		if _, err := usecase.Register(context.Background(), uuid.New(), uuid.New(), uuid.New(), "valid-token"); err != ErrOrderAttachmentInvalidArgument {
			t.Fatalf("case %d error = %v, want ErrOrderAttachmentInvalidArgument", index, err)
		}
	}
}

var _ OrderAttachmentRepo = (*orderAttachmentRepoStub)(nil)

func (s *orderAttachmentRepoStub) CheckEditable(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (s *orderAttachmentRepoStub) Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*OrderAttachment, error) {
	return s.created, nil
}

type attachmentStorageStub struct {
	attachment *OrderAttachment
	err        error
	disabled   bool
}

func (s *attachmentStorageStub) Enabled() bool { return !s.disabled }
func (s *attachmentStorageStub) PrepareUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *OrderAttachment) (*OrderAttachmentUpload, error) {
	return &OrderAttachmentUpload{}, s.err
}
func (s *attachmentStorageStub) VerifyUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (*OrderAttachment, error) {
	return s.attachment, s.err
}
func (s *attachmentStorageStub) Access(context.Context, uuid.UUID, *OrderAttachment, bool) (*OrderAttachmentAccess, error) {
	return &OrderAttachmentAccess{}, s.err
}
func (s *attachmentStorageStub) Delete(context.Context, uuid.UUID, string) error { return nil }

func TestOrderAttachmentPrepareValidatesAndDoesNotRegister(t *testing.T) {
	repo := &orderAttachmentRepoStub{}
	uc := NewOrderAttachmentUsecase(repo, &attachmentStorageStub{})
	org, actor, order := uuid.New(), uuid.New(), uuid.New()
	input := &OrderAttachment{DocType: "BL", FileName: "a.pdf", MIMEType: "application/pdf", FileSize: 100 << 20, Checksum: strings.Repeat("a", 64)}
	if _, err := uc.PrepareUpload(context.Background(), org, actor, order, input); err != nil {
		t.Fatal(err)
	}
	if repo.created != nil {
		t.Fatal("准备上传不应登记引用")
	}
	cases := []OrderAttachment{*input, *input, *input, *input, *input}
	cases[0].FileSize = 0
	cases[1].FileSize = 100<<20 + 1
	cases[2].Checksum = "invalid"
	cases[3].MIMEType = "invalid"
	cases[4].FileName = "../a.pdf"
	for i := range cases {
		if _, err := uc.PrepareUpload(context.Background(), org, actor, order, &cases[i]); err != ErrOrderAttachmentInvalidArgument {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	uc.storage = &attachmentStorageStub{disabled: true}
	if _, err := uc.PrepareUpload(context.Background(), org, actor, order, input); err != ErrOrderAttachmentStorageUnavailable {
		t.Fatal(err)
	}
}

func TestOrderAttachmentInvalidUploadNeverPersists(t *testing.T) {
	repo := &orderAttachmentRepoStub{}
	uc := NewOrderAttachmentUsecase(repo, &attachmentStorageStub{err: ErrOrderAttachmentUploadInvalid})
	if _, err := uc.Register(context.Background(), uuid.New(), uuid.New(), uuid.New(), "bad"); err != ErrOrderAttachmentUploadInvalid {
		t.Fatal(err)
	}
	if repo.created != nil {
		t.Fatal("无效授权不应写入")
	}
}
