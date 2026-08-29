package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerAttachmentRepoStub struct {
	created *PartnerAttachment
	audit   *AuditEvent
}

func (s *partnerAttachmentRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*PartnerAttachment, error) {
	return nil, nil
}

func (s *partnerAttachmentRepoStub) Create(_ context.Context, _, actorID, partnerID uuid.UUID, input *PartnerAttachment, audit *AuditEvent) (*PartnerAttachment, error) {
	s.created = input
	input.ID = uuid.New()
	input.PartnerID = partnerID
	input.UploadedBy = &actorID
	s.audit = audit
	return input, nil
}

func TestPartnerAttachmentRegisterNormalizesAndAudits(t *testing.T) {
	repo := &partnerAttachmentRepoStub{}
	usecase := NewPartnerAttachmentUsecase(repo)
	organizationID, actorID, partnerID := uuid.New(), uuid.New(), uuid.New()

	created, err := usecase.Register(context.Background(), organizationID, actorID, partnerID, &PartnerAttachment{
		IdempotencyKey: "  upload-001 ", FileName: " 合同.pdf ", MIMEType: " application/pdf ", FileSize: 1024, ObjectKey: " partners/contract-001 ", Checksum: " sha256:abc ",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if created.IdempotencyKey != "upload-001" || created.FileName != "合同.pdf" || created.MIMEType != "application/pdf" || created.ObjectKey != "partners/contract-001" || created.Checksum != "sha256:abc" || created.UploadedBy == nil || *created.UploadedBy != actorID {
		t.Fatalf("normalized attachment = %#v", created)
	}
	if repo.audit == nil || repo.audit.Action != "partner.attachment.register" {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

func TestPartnerAttachmentRejectsUnsafeMetadata(t *testing.T) {
	usecase := NewPartnerAttachmentUsecase(&partnerAttachmentRepoStub{})
	cases := []*PartnerAttachment{
		{IdempotencyKey: "k", FileName: "../secret", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key"},
		{IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 101 << 20, ObjectKey: "key"},
		{IdempotencyKey: "k", FileName: "a.txt", MIMEType: "text/plain", FileSize: 1, ObjectKey: "key\n"},
	}
	for index, input := range cases {
		if _, err := usecase.Register(context.Background(), uuid.New(), uuid.New(), uuid.New(), input); err != ErrPartnerAttachmentInvalidArgument {
			t.Fatalf("case %d error = %v, want ErrPartnerAttachmentInvalidArgument", index, err)
		}
	}
}

var _ PartnerAttachmentRepo = (*partnerAttachmentRepoStub)(nil)
