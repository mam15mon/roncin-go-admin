package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewOrderTagService, NewEnterpriseResourceService, NewAuthService, NewPartnerService, NewAdminService, NewMasterDataService, NewAirExportService, NewAirWeightService, NewOrderService, NewOrderLockService, NewOrderMilestoneService, NewOrderAttachmentService, NewOrderPersonnelService, NewBackgroundTaskService, NewOrderContainerService, NewOrderCargoItemService, NewOrderShippingDocumentService, NewSeaDocumentService, NewSeaSharedContainerService, NewSeaOrderChangeService, NewOrderAbnormalCaseService, NewOrderReleasePodService, NewExchangeRateService, NewFeeCatalogService, NewOrderFeeService, NewSettlementService, NewWorkbenchService)
