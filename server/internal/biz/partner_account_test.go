package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerAccountRepoStub struct {
	created *PartnerAccount
	updated *PartnerAccount
	audit   *AuditEvent
}

func (s *partnerAccountRepoStub) List(context.Context, uuid.UUID, uuid.UUID, PartnerAccountFilter) ([]*PartnerAccount, error) {
	return nil, nil
}

func (s *partnerAccountRepoStub) Create(_ context.Context, _, _ uuid.UUID, input *PartnerAccount, audit *AuditEvent) (*PartnerAccount, error) {
	s.created = input
	input.ID = uuid.New()
	audit.Details["account.id"] = input.ID.String()
	s.audit = audit
	return input, nil
}

func (s *partnerAccountRepoStub) Update(_ context.Context, _, _, id uuid.UUID, input *PartnerAccount, audit *AuditEvent) (*PartnerAccount, error) {
	s.updated = input
	input.ID = id
	s.audit = audit
	return input, nil
}

func TestPartnerAccountCreateNormalizesAndAudits(t *testing.T) {
	repo := &partnerAccountRepoStub{}
	usecase := NewPartnerAccountUsecase(repo)
	organizationID := uuid.New()
	actorID := uuid.New()
	partnerID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, &PartnerAccount{
		Name: "  上海收款账户  ", AccountHolder: "  测试结算单位  ", Currency: " cny ",
		BankName: " 中国银行 ", AccountNo: " 62220000 ", SwiftCode: " bocccnbj ",
		Usage: PartnerAccountUsageReceivable, IsDefaultReceivable: true, Enabled: true, Remark: "  月结  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "上海收款账户" || created.AccountHolder != "测试结算单位" || created.Currency != "CNY" || created.BankName != "中国银行" || created.AccountNo != "62220000" || created.SwiftCode != "BOCCCNBJ" || created.Remark != "月结" {
		t.Fatalf("normalized account = %#v", created)
	}
	if repo.audit == nil || repo.audit.Action != "partner.account.create" || repo.audit.Details["partner.id"] != partnerID.String() {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

func TestPartnerAccountRejectsInactiveDefault(t *testing.T) {
	usecase := NewPartnerAccountUsecase(&partnerAccountRepoStub{})
	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &PartnerAccount{
		Name: "账户", AccountHolder: "户名", Currency: "CNY", BankName: "银行", AccountNo: "123",
		Usage: PartnerAccountUsageReceivable, Enabled: false, IsDefaultReceivable: true,
	})
	if err != ErrPartnerAccountInvalidArgument {
		t.Fatalf("Create() error = %v, want ErrPartnerAccountInvalidArgument", err)
	}
}

func TestPartnerAccountRejectsDefaultOutsideUsageAndMissingBankName(t *testing.T) {
	usecase := NewPartnerAccountUsecase(&partnerAccountRepoStub{})
	base := PartnerAccount{
		Name: "账户", AccountHolder: "户名", Currency: "CNY", BankName: "银行", AccountNo: "123",
		Usage: PartnerAccountUsagePayable, Enabled: true,
	}
	wrongDirection := base
	wrongDirection.IsDefaultReceivable = true
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &wrongDirection); err != ErrPartnerAccountInvalidArgument {
		t.Fatalf("应付用途标应收默认错误 = %v，期望 %v", err, ErrPartnerAccountInvalidArgument)
	}
	missingBank := base
	missingBank.BankName = " "
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &missingBank); err != ErrPartnerAccountInvalidArgument {
		t.Fatalf("缺少银行名称错误 = %v，期望 %v", err, ErrPartnerAccountInvalidArgument)
	}
}

var _ PartnerAccountRepo = (*partnerAccountRepoStub)(nil)
