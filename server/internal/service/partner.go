package service

import (
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

// PartnerService 按子域拆分实现：基础资料（partner_profile.go）、DTO 转换
// （partner_convert.go）、开票资料（partner_invoice_profile.go）、账户
// （partner_account.go）、合同（partner_contract.go）、结算规则
// （partner_settlement_rule.go）、附件（partner_attachment.go）、运输预设
// （partner_shipping_preset.go）；本文件保留服务锚点与分页、可选值辅助。
type PartnerService struct {
	v1.UnimplementedPartnerServiceServer
	usecase               *biz.PartnerUsecase
	accountUsecase        *biz.PartnerAccountUsecase
	contractUsecase       *biz.PartnerContractUsecase
	settlementRuleUsecase *biz.PartnerSettlementRuleUsecase
	attachmentUsecase     *biz.PartnerAttachmentUsecase
	shippingPresetUsecase *biz.PartnerShippingPresetUsecase
	invoiceProfileUsecase *biz.PartnerInvoiceProfileUsecase
}

func NewPartnerService(usecase *biz.PartnerUsecase, accountUsecase *biz.PartnerAccountUsecase, contractUsecase *biz.PartnerContractUsecase, settlementRuleUsecase *biz.PartnerSettlementRuleUsecase, attachmentUsecase *biz.PartnerAttachmentUsecase, shippingPresetUsecase *biz.PartnerShippingPresetUsecase, invoiceProfileUsecase *biz.PartnerInvoiceProfileUsecase) *PartnerService {
	return &PartnerService{usecase: usecase, accountUsecase: accountUsecase, contractUsecase: contractUsecase, settlementRuleUsecase: settlementRuleUsecase, attachmentUsecase: attachmentUsecase, shippingPresetUsecase: shippingPresetUsecase, invoiceProfileUsecase: invoiceProfileUsecase}
}

func pageValues(page, pageSize int32) (int, int, error) {
	pageValue := int(page)
	if pageValue == 0 {
		pageValue = 1
	}
	pageSizeValue := int(pageSize)
	if pageSizeValue == 0 {
		pageSizeValue = 20
	}
	if !biz.ValidListPagination(pageValue, pageSizeValue) {
		return 0, 0, biz.ErrPartnerInvalidArgument
	}
	return pageValue, pageSizeValue, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func formatOptionalUUID(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

var _ v1.PartnerServiceServer = (*PartnerService)(nil)
