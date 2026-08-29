package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewEnterpriseResourceService, NewAuthService, NewPartnerService, NewAdminService, NewMasterDataService, NewOrderService, NewOrderMilestoneService, NewOrderAttachmentService, NewOrderPersonnelService, NewBackgroundTaskService, NewOrderContainerService, NewOrderCargoItemService, NewOrderShippingDocumentService, NewOrderAbnormalCaseService, NewOrderReleasePodService, NewExchangeRateService, NewFeeCatalogService, NewOrderFeeService, NewSettlementService)
