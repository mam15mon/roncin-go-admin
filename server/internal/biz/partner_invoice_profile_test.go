package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerInvoiceProfileRepoStub struct{ saved *PartnerInvoiceProfile }

func (stub *partnerInvoiceProfileRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*PartnerInvoiceProfile, error) {
	if stub.saved == nil {
		return nil, nil
	}
	return []*PartnerInvoiceProfile{stub.saved}, nil
}

func (stub *partnerInvoiceProfileRepoStub) Create(_ context.Context, _ uuid.UUID, profile *PartnerInvoiceProfile, _ *AuditEvent) (*PartnerInvoiceProfile, error) {
	stub.saved = profile
	return profile, nil
}

func (stub *partnerInvoiceProfileRepoStub) Update(_ context.Context, _ uuid.UUID, profile *PartnerInvoiceProfile, _ uint64, _ *AuditEvent) (*PartnerInvoiceProfile, error) {
	stub.saved = profile
	return profile, nil
}

func TestPartnerInvoiceProfileSpecialRequiresCompleteBankRegistration(t *testing.T) {
	repo := &partnerInvoiceProfileRepoStub{}
	usecase := NewPartnerInvoiceProfileUsecase(repo)
	input := CreatePartnerInvoiceProfileInput{PartnerID: uuid.Must(uuid.NewV7()), InvoiceTitle: " 测试客户 ", TaxpayerIdentificationNo: " 91310000TEST ", DefaultInvoiceType: FinanceInvoiceSpecial}
	if _, err := usecase.Create(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), input); err != ErrPartnerInvoiceProfileInvalidArgument {
		t.Fatalf("专票资料缺少注册地址、电话和银行资料应被拒绝，实际错误为 %v", err)
	}

	input.RegisteredAddress, input.RegisteredPhone = " 上海市浦东新区 ", " 021-12345678 "
	input.BankName, input.BankAccount = " 测试银行 ", " 62220000 "
	profile, err := usecase.Create(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), input)
	if err != nil {
		t.Fatalf("完整专票资料保存失败: %v", err)
	}
	if profile.InvoiceTitle != "测试客户" || profile.BankName != "测试银行" {
		t.Fatalf("开票资料未正确归一化: %#v", profile)
	}
}

func TestPartnerInvoiceProfileRejectsDisabledDefault(t *testing.T) {
	usecase := NewPartnerInvoiceProfileUsecase(&partnerInvoiceProfileRepoStub{})
	_, err := usecase.Update(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), UpdatePartnerInvoiceProfileInput{
		PartnerID: uuid.Must(uuid.NewV7()), ID: uuid.Must(uuid.NewV7()), InvoiceTitle: "测试客户", TaxpayerIdentificationNo: "91310000TEST", DefaultInvoiceType: FinanceInvoiceNormal, IsDefault: true, Enabled: false, ExpectedVersion: 1,
	})
	if err != ErrPartnerInvoiceProfileInvalidArgument {
		t.Fatalf("停用抬头不能同时设为默认，实际错误为 %v", err)
	}
}

var _ PartnerInvoiceProfileRepo = (*partnerInvoiceProfileRepoStub)(nil)
