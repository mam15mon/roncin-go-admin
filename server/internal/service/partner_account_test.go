package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestPartnerAccountWriteRequiresExplicitEnabled(t *testing.T) {
	principal := &biz.Principal{
		UserID:       uuid.New(),
		Organization: biz.Organization{ID: uuid.New()},
	}
	ctx := biz.WithPrincipal(context.Background(), principal)
	partnerID := uuid.New().String()
	account := &partnerv1.PartnerAccountInput{
		Name: "测试账户", AccountHolder: "测试户名", Currency: "CNY",
		BankName: "测试银行", AccountNo: "62220000",
		Usage: partnerv1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_BOTH,
	}
	service := &PartnerService{}

	if _, err := service.CreatePartnerAccount(ctx, &partnerv1.CreatePartnerAccountRequest{PartnerId: partnerID, Account: account}); !errors.Is(err, biz.ErrPartnerAccountInvalidArgument) {
		t.Fatalf("创建账户省略 enabled 的错误 = %v，期望 %v", err, biz.ErrPartnerAccountInvalidArgument)
	}
	if _, err := service.UpdatePartnerAccount(ctx, &partnerv1.UpdatePartnerAccountRequest{PartnerId: partnerID, Id: uuid.New().String(), Account: account}); !errors.Is(err, biz.ErrPartnerAccountInvalidArgument) {
		t.Fatalf("更新账户省略 enabled 的错误 = %v，期望 %v", err, biz.ErrPartnerAccountInvalidArgument)
	}
}
