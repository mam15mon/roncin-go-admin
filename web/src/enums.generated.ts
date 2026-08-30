// 自动生成：依据 server/api/**/*.proto 生成，请勿手工修改。
// 再生成命令：pnpm run generate:proto-constants
export const AccessMode = {
  ACCESS_MODE_UNSPECIFIED: 0,
  ACCESS_MODE_PUBLIC: 1,
  ACCESS_MODE_AUTHENTICATED: 2,
  ACCESS_MODE_PERMISSION: 3,
  ACCESS_MODE_ORDER_PERMISSION: 4,
} as const;

export type AccessMode = (typeof AccessMode)[keyof typeof AccessMode];

export const AccessDataScope = {
  DATA_SCOPE_UNSPECIFIED: 0,
  DATA_SCOPE_ALL: 1,
  DATA_SCOPE_ORGANIZATION: 2,
  DATA_SCOPE_ORGANIZATION_TREE: 3,
  DATA_SCOPE_SELF: 4,
} as const;

export type AccessDataScope = (typeof AccessDataScope)[keyof typeof AccessDataScope];

export const AdminDataScope = {
  DATA_SCOPE_UNSPECIFIED: 0,
  DATA_SCOPE_ALL: 1,
  DATA_SCOPE_ORGANIZATION: 2,
  DATA_SCOPE_ORGANIZATION_TREE: 3,
  DATA_SCOPE_SELF: 4,
} as const;

export type AdminDataScope = (typeof AdminDataScope)[keyof typeof AdminDataScope];

export const OrganizationKind = {
  ORGANIZATION_KIND_UNSPECIFIED: 0,
  ORGANIZATION_KIND_HEADQUARTERS: 1,
  ORGANIZATION_KIND_COMPANY: 2,
  ORGANIZATION_KIND_DEPARTMENT: 3,
  ORGANIZATION_KIND_TEAM: 4,
} as const;

export type OrganizationKind = (typeof OrganizationKind)[keyof typeof OrganizationKind];

export const AdminUserStatus = {
  ADMIN_USER_STATUS_UNSPECIFIED: 0,
  ADMIN_USER_STATUS_ACTIVE: 1,
  ADMIN_USER_STATUS_PENDING_AUTHORIZATION: 2,
  ADMIN_USER_STATUS_TERMINATED: 3,
  ADMIN_USER_STATUS_REMOVED_FROM_ORGANIZATION: 4,
  ADMIN_USER_STATUS_DISABLED: 5,
} as const;

export type AdminUserStatus = (typeof AdminUserStatus)[keyof typeof AdminUserStatus];

export const DingTalkLoginStatus = {
  DING_TALK_LOGIN_STATUS_UNSPECIFIED: 0,
  DING_TALK_LOGIN_STATUS_AUTHENTICATED: 1,
  DING_TALK_LOGIN_STATUS_REGISTRATION_REQUIRED: 2,
} as const;

export type DingTalkLoginStatus = (typeof DingTalkLoginStatus)[keyof typeof DingTalkLoginStatus];

export const EnterpriseResourceType = {
  ENTERPRISE_RESOURCE_TYPE_UNSPECIFIED: 0,
  ENTERPRISE_RESOURCE_TYPE_ADDRESS: 1,
  ENTERPRISE_RESOURCE_TYPE_REMARK: 2,
  ENTERPRISE_RESOURCE_TYPE_IMAGE: 3,
  ENTERPRISE_RESOURCE_TYPE_TAG: 4,
  ENTERPRISE_RESOURCE_TYPE_SHIPPER: 5,
  ENTERPRISE_RESOURCE_TYPE_CONSIGNEE: 6,
  ENTERPRISE_RESOURCE_TYPE_NOTIFY_PARTY: 7,
  ENTERPRISE_RESOURCE_TYPE_ENGLISH_CARGO_NAME: 8,
  ENTERPRISE_RESOURCE_TYPE_HS_CODE: 9,
  ENTERPRISE_RESOURCE_TYPE_MARKS: 10,
} as const;

export type EnterpriseResourceType = (typeof EnterpriseResourceType)[keyof typeof EnterpriseResourceType];

export const EnterpriseRemarkType = {
  ENTERPRISE_REMARK_TYPE_UNSPECIFIED: 0,
  ENTERPRISE_REMARK_TYPE_BOOKING: 1,
  ENTERPRISE_REMARK_TYPE_ALLOCATION: 2,
  ENTERPRISE_REMARK_TYPE_TRANSPORT: 3,
  ENTERPRISE_REMARK_TYPE_ORDER: 4,
  ENTERPRISE_REMARK_TYPE_BILL_OF_LADING: 5,
  ENTERPRISE_REMARK_TYPE_CUSTOMER: 6,
  ENTERPRISE_REMARK_TYPE_SUPPLIER: 7,
  ENTERPRISE_REMARK_TYPE_FOREIGN_AGENT: 8,
  ENTERPRISE_REMARK_TYPE_QUOTATION: 9,
  ENTERPRISE_REMARK_TYPE_MANIFEST: 10,
  ENTERPRISE_REMARK_TYPE_PACKING_LIST: 11,
  ENTERPRISE_REMARK_TYPE_OPERATION: 12,
  ENTERPRISE_REMARK_TYPE_COMMISSION: 13,
  ENTERPRISE_REMARK_TYPE_WAREHOUSE: 14,
} as const;

export type EnterpriseRemarkType = (typeof EnterpriseRemarkType)[keyof typeof EnterpriseRemarkType];

export const EnterpriseAddressType = {
  ENTERPRISE_ADDRESS_TYPE_UNSPECIFIED: 0,
  ENTERPRISE_ADDRESS_TYPE_CONTAINER_OPERATION: 1,
  ENTERPRISE_ADDRESS_TYPE_PICKUP: 2,
  ENTERPRISE_ADDRESS_TYPE_DELIVERY: 3,
} as const;

export type EnterpriseAddressType = (typeof EnterpriseAddressType)[keyof typeof EnterpriseAddressType];

export const BilledFeeEditableField = {
  BILLED_FEE_EDITABLE_FIELD_UNSPECIFIED: 0,
  BILLED_FEE_EDITABLE_FIELD_FEE_NAME: 1,
  BILLED_FEE_EDITABLE_FIELD_CURRENCY: 2,
  BILLED_FEE_EDITABLE_FIELD_EXCHANGE_RATE: 3,
  BILLED_FEE_EDITABLE_FIELD_QUANTITY: 4,
  BILLED_FEE_EDITABLE_FIELD_UNIT_PRICE: 5,
  BILLED_FEE_EDITABLE_FIELD_TAX_RATE: 6,
} as const;

export type BilledFeeEditableField = (typeof BilledFeeEditableField)[keyof typeof BilledFeeEditableField];

export const MasterDataImportMode = {
  MASTER_DATA_IMPORT_MODE_UNSPECIFIED: 0,
  MASTER_DATA_IMPORT_MODE_CREATE_ONLY: 1,
  MASTER_DATA_IMPORT_MODE_UPSERT: 2,
} as const;

export type MasterDataImportMode = (typeof MasterDataImportMode)[keyof typeof MasterDataImportMode];

export const MasterDataKind = {
  MASTER_DATA_KIND_UNSPECIFIED: 0,
  MASTER_DATA_KIND_CURRENCY: 1,
  MASTER_DATA_KIND_COUNTRY: 2,
  MASTER_DATA_KIND_REGION: 3,
  MASTER_DATA_KIND_CONTAINER_SPEC: 7,
  MASTER_DATA_KIND_SERVICE_TYPE: 8,
  MASTER_DATA_KIND_CARGO_CATEGORY: 9,
  MASTER_DATA_KIND_ABNORMAL_CASE: 10,
} as const;

export type MasterDataKind = (typeof MasterDataKind)[keyof typeof MasterDataKind];

export const DocumentType = {
  DOCUMENT_TYPE_UNSPECIFIED: 0,
  DOCUMENT_TYPE_ORDER: 1,
  DOCUMENT_TYPE_BILL: 2,
  DOCUMENT_TYPE_QUOTATION: 3,
  DOCUMENT_TYPE_WRITE_OFF: 4,
  DOCUMENT_TYPE_RECEIPT_PAYMENT: 5,
  DOCUMENT_TYPE_CONTRACT: 6,
  DOCUMENT_TYPE_INTERNAL_REFERENCE: 7,
  DOCUMENT_TYPE_CUSTOMER_REFERENCE: 8,
  DOCUMENT_TYPE_HOUSE_BILL: 9,
  DOCUMENT_TYPE_INVOICE: 11,
  DOCUMENT_TYPE_FREIGHT_RATE: 12,
  DOCUMENT_TYPE_COMMISSION: 13,
  DOCUMENT_TYPE_BILL_BATCH: 14,
} as const;

export type DocumentType = (typeof DocumentType)[keyof typeof DocumentType];

export const DateFormat = {
  DATE_FORMAT_UNSPECIFIED: 0,
  DATE_FORMAT_YYYYMMDD: 1,
  DATE_FORMAT_YYYYMM: 2,
  DATE_FORMAT_YYYY: 3,
  DATE_FORMAT_NONE: 4,
} as const;

export type DateFormat = (typeof DateFormat)[keyof typeof DateFormat];

export const ResetPolicy = {
  RESET_POLICY_UNSPECIFIED: 0,
  RESET_POLICY_DAILY: 1,
  RESET_POLICY_MONTHLY: 2,
  RESET_POLICY_YEARLY: 3,
  RESET_POLICY_NEVER: 4,
} as const;

export type ResetPolicy = (typeof ResetPolicy)[keyof typeof ResetPolicy];

export const MasterdataBusinessType = {
  BUSINESS_TYPE_UNSPECIFIED: 0,
  BUSINESS_TYPE_SE: 1,
  BUSINESS_TYPE_SI: 2,
  BUSINESS_TYPE_AE: 3,
  BUSINESS_TYPE_AI: 4,
  BUSINESS_TYPE_LAND: 5,
  BUSINESS_TYPE_RAIL: 6,
} as const;

export type MasterdataBusinessType = (typeof MasterdataBusinessType)[keyof typeof MasterdataBusinessType];

export const OrderBusinessType = {
  BUSINESS_TYPE_UNSPECIFIED: 0,
  BUSINESS_TYPE_SE: 1,
  BUSINESS_TYPE_SI: 2,
  BUSINESS_TYPE_AE: 3,
  BUSINESS_TYPE_AI: 4,
  BUSINESS_TYPE_LAND: 5,
  BUSINESS_TYPE_RAIL: 6,
} as const;

export type OrderBusinessType = (typeof OrderBusinessType)[keyof typeof OrderBusinessType];

export const TradeDirection = {
  TRADE_DIRECTION_UNSPECIFIED: 0,
  TRADE_DIRECTION_EXPORT: 1,
  TRADE_DIRECTION_IMPORT: 2,
} as const;

export type TradeDirection = (typeof TradeDirection)[keyof typeof TradeDirection];

export const TradeTerm = {
  TRADE_TERM_UNSPECIFIED: 0,
  TRADE_TERM_EXW: 1,
  TRADE_TERM_FCA: 2,
  TRADE_TERM_FOB: 3,
  TRADE_TERM_CFR: 4,
  TRADE_TERM_CIF: 5,
  TRADE_TERM_CPT: 6,
  TRADE_TERM_CIP: 7,
  TRADE_TERM_DAP: 8,
  TRADE_TERM_DPU: 9,
  TRADE_TERM_DDU: 10,
  TRADE_TERM_DDP: 11,
  TRADE_TERM_LDP: 12,
} as const;

export type TradeTerm = (typeof TradeTerm)[keyof typeof TradeTerm];

export const PaymentTerm = {
  PAYMENT_TERM_UNSPECIFIED: 0,
  PAYMENT_TERM_PREPAID: 1,
  PAYMENT_TERM_COLLECT: 2,
} as const;

export type PaymentTerm = (typeof PaymentTerm)[keyof typeof PaymentTerm];

export const ShipmentType = {
  SHIPMENT_TYPE_UNSPECIFIED: 0,
  SHIPMENT_TYPE_FCL: 1,
  SHIPMENT_TYPE_LCL: 2,
  SHIPMENT_TYPE_BREAK_BULK: 3,
} as const;

export type ShipmentType = (typeof ShipmentType)[keyof typeof ShipmentType];

export const ContainerOwnership = {
  CONTAINER_OWNERSHIP_UNSPECIFIED: 0,
  CONTAINER_OWNERSHIP_COC: 1,
  CONTAINER_OWNERSHIP_SOC: 2,
} as const;

export type ContainerOwnership = (typeof ContainerOwnership)[keyof typeof ContainerOwnership];

export const ShipmentMode = {
  SHIPMENT_MODE_UNSPECIFIED: 0,
  SHIPMENT_MODE_TRADITIONAL_FORWARDING: 1,
  SHIPMENT_MODE_CROSS_BORDER: 2,
} as const;

export type ShipmentMode = (typeof ShipmentMode)[keyof typeof ShipmentMode];

export const OrderReferenceType = {
  ORDER_REFERENCE_TYPE_UNSPECIFIED: 0,
  ORDER_REFERENCE_TYPE_CUSTOMER: 1,
  ORDER_REFERENCE_TYPE_INTERNAL: 2,
} as const;

export type OrderReferenceType = (typeof OrderReferenceType)[keyof typeof OrderReferenceType];

export const OrderNumberFilterType = {
  ORDER_NUMBER_FILTER_TYPE_UNSPECIFIED: 0,
  ORDER_NUMBER_FILTER_TYPE_ORDER: 1,
  ORDER_NUMBER_FILTER_TYPE_MASTER: 2,
  ORDER_NUMBER_FILTER_TYPE_CONSOLIDATED_MASTER: 3,
} as const;

export type OrderNumberFilterType = (typeof OrderNumberFilterType)[keyof typeof OrderNumberFilterType];

export const OrderFlowStatus = {
  ORDER_FLOW_STATUS_UNSPECIFIED: 0,
  ORDER_FLOW_STATUS_DRAFT: 1,
  ORDER_FLOW_STATUS_BOOKED: 2,
  ORDER_FLOW_STATUS_SPACE_ALLOCATED: 3,
  ORDER_FLOW_STATUS_TRUCKING_ARRANGED: 4,
  ORDER_FLOW_STATUS_DOCUMENT_CUTOFF: 5,
  ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED: 6,
  ORDER_FLOW_STATUS_DOCUMENT_RELEASED: 7,
} as const;

export type OrderFlowStatus = (typeof OrderFlowStatus)[keyof typeof OrderFlowStatus];

export const OrderTerminationStatus = {
  ORDER_TERMINATION_STATUS_UNSPECIFIED: 0,
  ORDER_TERMINATION_STATUS_ACTIVE: 1,
  ORDER_TERMINATION_STATUS_TERMINATING: 2,
  ORDER_TERMINATION_STATUS_TERMINATED: 3,
} as const;

export type OrderTerminationStatus = (typeof OrderTerminationStatus)[keyof typeof OrderTerminationStatus];

export const OrderTerminationType = {
  ORDER_TERMINATION_TYPE_UNSPECIFIED: 0,
  ORDER_TERMINATION_TYPE_CUSTOMER_CANCEL: 1,
  ORDER_TERMINATION_TYPE_CARRIER_CANCEL: 2,
  ORDER_TERMINATION_TYPE_CUSTOMS_RETURN: 3,
  ORDER_TERMINATION_TYPE_OPERATION_CANCEL: 4,
  ORDER_TERMINATION_TYPE_OTHER: 5,
} as const;

export type OrderTerminationType = (typeof OrderTerminationType)[keyof typeof OrderTerminationType];

export const OrderClosureStatus = {
  ORDER_CLOSURE_STATUS_UNSPECIFIED: 0,
  ORDER_CLOSURE_STATUS_OPEN: 1,
  ORDER_CLOSURE_STATUS_CLOSED: 2,
} as const;

export type OrderClosureStatus = (typeof OrderClosureStatus)[keyof typeof OrderClosureStatus];

export const OrderAllowedAction = {
  ORDER_ALLOWED_ACTION_UNSPECIFIED: 0,
  ORDER_ALLOWED_ACTION_EDIT: 1,
  ORDER_ALLOWED_ACTION_TRANSITION_FLOW: 2,
  ORDER_ALLOWED_ACTION_START_TERMINATION: 3,
  ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION: 4,
  ORDER_ALLOWED_ACTION_CANCEL_TERMINATION: 5,
  ORDER_ALLOWED_ACTION_CLOSE: 6,
  ORDER_ALLOWED_ACTION_REOPEN: 7,
} as const;

export type OrderAllowedAction = (typeof OrderAllowedAction)[keyof typeof OrderAllowedAction];

export const OrderAbnormalCaseStatus = {
  ORDER_ABNORMAL_CASE_STATUS_UNSPECIFIED: 0,
  ORDER_ABNORMAL_CASE_STATUS_ACTIVE: 1,
  ORDER_ABNORMAL_CASE_STATUS_RESOLVED: 2,
} as const;

export type OrderAbnormalCaseStatus = (typeof OrderAbnormalCaseStatus)[keyof typeof OrderAbnormalCaseStatus];

export const OrderFeeDirection = {
  ORDER_FEE_DIRECTION_UNSPECIFIED: 0,
  ORDER_FEE_DIRECTION_RECEIVABLE: 1,
  ORDER_FEE_DIRECTION_PAYABLE: 2,
} as const;

export type OrderFeeDirection = (typeof OrderFeeDirection)[keyof typeof OrderFeeDirection];

export const OrderFeeStatus = {
  ORDER_FEE_STATUS_UNSPECIFIED: 0,
  ORDER_FEE_STATUS_DRAFT: 1,
  ORDER_FEE_STATUS_CONFIRMED: 2,
  ORDER_FEE_STATUS_BILLED: 3,
  ORDER_FEE_STATUS_CANCELLED: 4,
} as const;

export type OrderFeeStatus = (typeof OrderFeeStatus)[keyof typeof OrderFeeStatus];

export const OrderPersonnelRole = {
  ORDER_PERSONNEL_ROLE_UNSPECIFIED: 0,
  ORDER_PERSONNEL_ROLE_CREATOR: 1,
  ORDER_PERSONNEL_ROLE_OPERATOR: 2,
  ORDER_PERSONNEL_ROLE_SALES: 3,
  ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE: 4,
  ORDER_PERSONNEL_ROLE_DOCUMENT: 5,
  ORDER_PERSONNEL_ROLE_COMMERCIAL: 6,
  ORDER_PERSONNEL_ROLE_ASSOCIATE: 7,
  ORDER_PERSONNEL_ROLE_ASSOCIATE2: 8,
} as const;

export type OrderPersonnelRole = (typeof OrderPersonnelRole)[keyof typeof OrderPersonnelRole];

export const OrderReleasePodStatus = {
  ORDER_RELEASE_POD_STATUS_UNSPECIFIED: 0,
  ORDER_RELEASE_POD_STATUS_PENDING: 1,
  ORDER_RELEASE_POD_STATUS_SIGNED: 2,
  ORDER_RELEASE_POD_STATUS_RETURNED: 3,
} as const;

export type OrderReleasePodStatus = (typeof OrderReleasePodStatus)[keyof typeof OrderReleasePodStatus];

export const OrderShippingDocumentStatus = {
  ORDER_SHIPPING_DOCUMENT_STATUS_UNSPECIFIED: 0,
  ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT: 1,
  ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED: 2,
  ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED: 3,
} as const;

export type OrderShippingDocumentStatus = (typeof OrderShippingDocumentStatus)[keyof typeof OrderShippingDocumentStatus];

export const PartnerRoleType = {
  PARTNER_ROLE_TYPE_UNSPECIFIED: 0,
  PARTNER_ROLE_TYPE_CUSTOMER: 1,
  PARTNER_ROLE_TYPE_SUPPLIER: 2,
  PARTNER_ROLE_TYPE_FOREIGN_AGENT: 3,
  PARTNER_ROLE_TYPE_CARRIER: 4,
} as const;

export type PartnerRoleType = (typeof PartnerRoleType)[keyof typeof PartnerRoleType];

export const PartnerCustomerType = {
  PARTNER_CUSTOMER_TYPE_UNSPECIFIED: 0,
  PARTNER_CUSTOMER_TYPE_DIRECT: 1,
  PARTNER_CUSTOMER_TYPE_PEER: 2,
} as const;

export type PartnerCustomerType = (typeof PartnerCustomerType)[keyof typeof PartnerCustomerType];

export const PartnerBusinessType = {
  PARTNER_BUSINESS_TYPE_UNSPECIFIED: 0,
  PARTNER_BUSINESS_TYPE_SE: 1,
  PARTNER_BUSINESS_TYPE_SI: 2,
  PARTNER_BUSINESS_TYPE_AE: 3,
  PARTNER_BUSINESS_TYPE_AI: 4,
  PARTNER_BUSINESS_TYPE_LAND: 5,
  PARTNER_BUSINESS_TYPE_RAIL: 6,
} as const;

export type PartnerBusinessType = (typeof PartnerBusinessType)[keyof typeof PartnerBusinessType];

export const PartnerAssignmentRole = {
  PARTNER_ASSIGNMENT_ROLE_UNSPECIFIED: 0,
  PARTNER_ASSIGNMENT_ROLE_CREATOR: 1,
  PARTNER_ASSIGNMENT_ROLE_OPERATOR: 2,
  PARTNER_ASSIGNMENT_ROLE_SALES: 3,
  PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE: 4,
  PARTNER_ASSIGNMENT_ROLE_FINANCE: 5,
  PARTNER_ASSIGNMENT_ROLE_COMMERCIAL: 6,
  PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT: 7,
  PARTNER_ASSIGNMENT_ROLE_DOCUMENT: 8,
} as const;

export type PartnerAssignmentRole = (typeof PartnerAssignmentRole)[keyof typeof PartnerAssignmentRole];

export const PartnerImportMode = {
  PARTNER_IMPORT_MODE_UNSPECIFIED: 0,
  PARTNER_IMPORT_MODE_CREATE_ONLY: 1,
  PARTNER_IMPORT_MODE_UPSERT: 2,
} as const;

export type PartnerImportMode = (typeof PartnerImportMode)[keyof typeof PartnerImportMode];

export const PartnerShippingPresetType = {
  PARTNER_SHIPPING_PRESET_TYPE_UNSPECIFIED: 0,
  PARTNER_SHIPPING_PRESET_TYPE_SHIPPER: 1,
  PARTNER_SHIPPING_PRESET_TYPE_CONSIGNEE: 2,
  PARTNER_SHIPPING_PRESET_TYPE_NOTIFY_PARTY: 3,
  PARTNER_SHIPPING_PRESET_TYPE_ENGLISH_CARGO_NAME: 4,
  PARTNER_SHIPPING_PRESET_TYPE_HS_CODE: 5,
  PARTNER_SHIPPING_PRESET_TYPE_MARKS: 6,
} as const;

export type PartnerShippingPresetType = (typeof PartnerShippingPresetType)[keyof typeof PartnerShippingPresetType];

export const PartnerAccountStatus = {
  PARTNER_ACCOUNT_STATUS_UNSPECIFIED: 0,
  PARTNER_ACCOUNT_STATUS_ACTIVE: 1,
  PARTNER_ACCOUNT_STATUS_INACTIVE: 2,
} as const;

export type PartnerAccountStatus = (typeof PartnerAccountStatus)[keyof typeof PartnerAccountStatus];

export const PartnerContractStatus = {
  PARTNER_CONTRACT_STATUS_UNSPECIFIED: 0,
  PARTNER_CONTRACT_STATUS_PENDING: 1,
  PARTNER_CONTRACT_STATUS_ACTIVE: 2,
  PARTNER_CONTRACT_STATUS_EXPIRED: 3,
  PARTNER_CONTRACT_STATUS_TERMINATED: 4,
} as const;

export type PartnerContractStatus = (typeof PartnerContractStatus)[keyof typeof PartnerContractStatus];

export const PartnerStatementMode = {
  PARTNER_STATEMENT_MODE_UNSPECIFIED: 0,
  PARTNER_STATEMENT_MODE_SINGLE: 1,
  PARTNER_STATEMENT_MODE_MULTI: 2,
} as const;

export type PartnerStatementMode = (typeof PartnerStatementMode)[keyof typeof PartnerStatementMode];

export const PartnerSettlementMethod = {
  PARTNER_SETTLEMENT_METHOD_UNSPECIFIED: 0,
  PARTNER_SETTLEMENT_METHOD_BY_TICKET: 1,
  PARTNER_SETTLEMENT_METHOD_MONTHLY: 2,
  PARTNER_SETTLEMENT_METHOD_WEEKLY: 3,
  PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY: 4,
  PARTNER_SETTLEMENT_METHOD_BI_MONTHLY: 5,
  PARTNER_SETTLEMENT_METHOD_QUARTERLY: 6,
  PARTNER_SETTLEMENT_METHOD_DAYS_45: 7,
  PARTNER_SETTLEMENT_METHOD_PREPAID: 8,
} as const;

export type PartnerSettlementMethod = (typeof PartnerSettlementMethod)[keyof typeof PartnerSettlementMethod];

export const PartnerSettlementBase = {
  PARTNER_SETTLEMENT_BASE_UNSPECIFIED: 0,
  PARTNER_SETTLEMENT_BASE_BILL_DATE: 1,
  PARTNER_SETTLEMENT_BASE_SAILING_DATE: 2,
  PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE: 3,
} as const;

export type PartnerSettlementBase = (typeof PartnerSettlementBase)[keyof typeof PartnerSettlementBase];

export const BackgroundTaskKind = {
  BACKGROUND_TASK_KIND_UNSPECIFIED: 0,
  BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT: 1,
  BACKGROUND_TASK_KIND_UNLOCODE_IMPORT: 2,
  BACKGROUND_TASK_KIND_ORDER_REMINDER: 3,
  BACKGROUND_TASK_KIND_INTEGRATION: 4,
  BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION: 5,
} as const;

export type BackgroundTaskKind = (typeof BackgroundTaskKind)[keyof typeof BackgroundTaskKind];

export const BackgroundTaskStatus = {
  BACKGROUND_TASK_STATUS_UNSPECIFIED: 0,
  BACKGROUND_TASK_STATUS_PENDING: 1,
  BACKGROUND_TASK_STATUS_RUNNING: 2,
  BACKGROUND_TASK_STATUS_SUCCEEDED: 3,
  BACKGROUND_TASK_STATUS_FAILED: 4,
  BACKGROUND_TASK_STATUS_DEAD_LETTER: 5,
} as const;

export type BackgroundTaskStatus = (typeof BackgroundTaskStatus)[keyof typeof BackgroundTaskStatus];

export const BackgroundTaskPhase = {
  BACKGROUND_TASK_PHASE_UNSPECIFIED: 0,
  BACKGROUND_TASK_PHASE_ACTIVE: 1,
  BACKGROUND_TASK_PHASE_HISTORY: 2,
} as const;

export type BackgroundTaskPhase = (typeof BackgroundTaskPhase)[keyof typeof BackgroundTaskPhase];
