package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerInvoiceProfileRepoStub struct{ saved *PartnerInvoiceProfile }

func (stub *partnerInvoiceProfileRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*PartnerInvoiceProfile, error) {
	return stub.saved, nil
}

func (stub *partnerInvoiceProfileRepoStub) Save(_ context.Context, _ uuid.UUID, profile *PartnerInvoiceProfile, _ uint64, _ *AuditEvent) (*PartnerInvoiceProfile, error) {
	stub.saved = profile
	return profile, nil
}

func TestPartnerInvoiceProfileSpecialRequiresCompleteBankRegistration(t *testing.T) {
	repo := &partnerInvoiceProfileRepoStub{}
	usecase := NewPartnerInvoiceProfileUsecase(repo)
	input := SavePartnerInvoiceProfileInput{PartnerID: uuid.Must(uuid.NewV7()), InvoiceTitle: " 测试客户 ", TaxpayerIdentificationNo: " 91310000TEST ", DefaultInvoiceType: FinanceInvoiceSpecial}
	if _, err := usecase.Save(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), input); err != ErrPartnerInvoiceProfileInvalidArgument {
		t.Fatalf("专票资料缺少注册地址、电话和银行资料应被拒绝，实际错误为 %v", err)
	}

	input.RegisteredAddress, input.RegisteredPhone = " 上海市浦东新区 ", " 021-12345678 "
	input.BankName, input.BankAccount = " 测试银行 ", " 62220000 "
	profile, err := usecase.Save(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), input)
	if err != nil {
		t.Fatalf("完整专票资料保存失败: %v", err)
	}
	if profile.InvoiceTitle != "测试客户" || profile.BankName != "测试银行" {
		t.Fatalf("开票资料未正确归一化: %#v", profile)
	}
}

var _ PartnerInvoiceProfileRepo = (*partnerInvoiceProfileRepoStub)(nil)
