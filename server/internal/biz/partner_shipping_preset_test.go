package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerShippingPresetRepoStub struct {
	created *PartnerShippingPreset
	updated *PartnerShippingPreset
	audit   *AuditEvent
}

func (stub *partnerShippingPresetRepoStub) List(context.Context, uuid.UUID, uuid.UUID, PartnerShippingPresetListOptions) ([]*PartnerShippingPreset, error) {
	return nil, nil
}

func (stub *partnerShippingPresetRepoStub) Create(_ context.Context, _ uuid.UUID, partnerID uuid.UUID, input *PartnerShippingPreset, audit *AuditEvent) (*PartnerShippingPreset, error) {
	stub.created = input
	input.ID = uuid.New()
	input.PartnerID = partnerID
	stub.audit = audit
	return input, nil
}

func (stub *partnerShippingPresetRepoStub) Update(_ context.Context, _ uuid.UUID, partnerID, id uuid.UUID, input *PartnerShippingPreset, audit *AuditEvent) (*PartnerShippingPreset, error) {
	stub.updated = input
	input.ID = id
	input.PartnerID = partnerID
	stub.audit = audit
	return input, nil
}

func TestPartnerShippingPresetNormalizesPartyPayload(t *testing.T) {
	repo := &partnerShippingPresetRepoStub{}
	usecase := NewPartnerShippingPresetUsecase(repo)
	organizationID := uuid.New()
	actorID := uuid.New()
	partnerID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, &PartnerShippingPreset{
		PresetType: PartnerShippingPresetShipper,
		Title:      " 默认发货人 ",
		Party: &PartnerShippingPartyPayload{
			CompanyName: " ACME LOGISTICS ", CountryCode: " cn ", Email: "ops@example.com",
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Title != "默认发货人" || created.Party.CompanyName != "ACME LOGISTICS" || created.Party.CountryCode != "CN" {
		t.Fatalf("normalized preset = %#v", created)
	}
	if repo.audit == nil || repo.audit.Action != "partner.shipping_preset.create" {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

func TestPartnerShippingPresetValidatesPayloadTypeAndHSCode(t *testing.T) {
	usecase := NewPartnerShippingPresetUsecase(&partnerShippingPresetRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()
	partnerID := uuid.New()

	invalid := []*PartnerShippingPreset{
		{PresetType: PartnerShippingPresetShipper, Title: "发货人", Text: &PartnerShippingTextPayload{Content: "ACME"}},
		{PresetType: PartnerShippingPresetEnglishCargoName, Title: "品名", Text: &PartnerShippingTextPayload{}},
		{PresetType: PartnerShippingPresetHSCode, Title: "HS", Text: &PartnerShippingTextPayload{Code: "ABC123"}},
		{PresetType: PartnerShippingPresetHSCode, Title: "HS", Party: &PartnerShippingPartyPayload{CompanyName: "ACME", CountryCode: "US"}},
	}
	for index, input := range invalid {
		if _, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, input); err != ErrPartnerShippingPresetInvalidArgument {
			t.Fatalf("invalid preset %d error = %v, want ErrPartnerShippingPresetInvalidArgument", index, err)
		}
	}

	if _, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, &PartnerShippingPreset{
		PresetType: PartnerShippingPresetHSCode, Title: "家具", Text: &PartnerShippingTextPayload{Code: "9403609990"}, Enabled: true,
	}); err != nil {
		t.Fatalf("valid HS code error = %v", err)
	}
}

var _ PartnerShippingPresetRepo = (*partnerShippingPresetRepoStub)(nil)
