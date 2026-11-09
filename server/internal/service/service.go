package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewPasskeyService, NewEnterpriseResourceService, NewAuthService, NewPartnerService, NewAdminService, NewMasterDataService, NewAirExportService, NewAirWeightService, NewOrderService, NewOrderLockService, NewOrderAttachmentService, NewOrderPersonnelService, NewBackgroundTaskService, NewOrderContainerService, NewOrderCargoItemService, NewOrderShippingDocumentService, NewSeaDocumentService, NewSeaSharedContainerService, NewSeaOrderChangeService, NewOrderAbnormalCaseService, NewOrderReleasePodService, NewExchangeRateService, NewFeeCatalogService, NewOrderFeeService, NewSettlementService, NewWorkbenchService)
