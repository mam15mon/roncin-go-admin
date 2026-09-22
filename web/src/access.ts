import { AuthOrganizationKind, BackgroundTaskKind } from '@/enums.generated';
import type {
  ManifestPermissionKey,
  OrderPermissionOperation,
} from '@/permissions.generated';

// 键值必须命中后端权限清单生成的前缀类型：后端权限码改名或删除后，这里的旧
// 键名会在 `pnpm --dir web tsc` 时直接报错，避免按钮权限静默失效。
const permissions = {
  platformAccess: 'system.platform.access',
  organizationRead: 'system.organization.read',
  organizationCreate: 'system.organization.create',
  organizationUpdate: 'system.organization.update',
  userRead: 'system.user.read',
  userCreate: 'system.user.create',
  userUpdate: 'system.user.update',
  userTerminate: 'system.user.delete',
  userAuthorizeWeCom: 'system.user.authorize_wecom',
  userAuthorizeDingTalk: 'system.user.authorize_dingtalk',
  userDingTalkInvitationManage: 'system.user.dingtalk_invitation.manage',
  userResetPassword: 'system.user.reset_password',
  roleRead: 'system.role.read',
  roleCreate: 'system.role.create',
  roleUpdate: 'system.role.update',
  roleDelete: 'system.role.delete',
  permissionRead: 'system.permission.read',
  auditRead: 'system.audit.read',
  financeExchangeRateRead: 'system.finance.exchange_rate.read',
  financeExchangeRateCreate: 'system.finance.exchange_rate.create',
  financeExchangeRateUpdate: 'system.finance.exchange_rate.update',
  financeExchangeRateDisable: 'system.finance.exchange_rate.disable',
  financeExchangeRateOverride: 'system.finance.exchange_rate.override',
  financeFeeSettingRead: 'system.finance.fee_setting.read',
  financeFeeSettingCreate: 'system.finance.fee_setting.create',
  financeFeeSettingUpdate: 'system.finance.fee_setting.update',
  financeFeeRead: 'system.finance.fee.read',
  financeFeeTag: 'system.finance.fee.tag',
  financeBillRead: 'system.finance.bill.read',
  financeBillCreate: 'system.finance.bill.create',
  financeBillUpdate: 'system.finance.bill.update',
  financeBillConfirm: 'system.finance.bill.confirm',
  financeInvoiceRead: 'system.finance.invoice.read',
  financeInvoiceCreate: 'system.finance.invoice.create',
  financeInvoiceUpdate: 'system.finance.invoice.update',
  financeCashflowRead: 'system.finance.cashflow.read',
  financeCashflowCreate: 'system.finance.cashflow.create',
  financeCashflowUpdate: 'system.finance.cashflow.update',
  financeVerificationRead: 'system.finance.verification.read',
  financeVerificationCreate: 'system.finance.verification.create',
  financeVerificationReverse: 'system.finance.verification.reverse',
  financeNettingRead: 'system.finance.netting.read',
  financeNettingCreate: 'system.finance.netting.create',
  financeNettingConfirm: 'system.finance.netting.confirm',
  financeNettingReverse: 'system.finance.netting.reverse',
  financeCommissionRead: 'system.finance.commission.read',
  financeCommissionManage: 'system.finance.commission.manage',
  financeCommissionConfigure: 'system.finance.commission.configure',
  financeCommissionExport: 'system.finance.commission.export',
  partnerRead: 'business.partner.read',
  partnerCreate: 'business.partner.create',
  partnerUpdate: 'business.partner.update',
  partnerBlacklist: 'business.partner.blacklist',
  partnerImport: 'business.partner.import',
  partnerExport: 'business.partner.export',
  partnerAccountRead: 'business.partner.account.read',
  partnerAccountCreate: 'business.partner.account.create',
  partnerAccountUpdate: 'business.partner.account.update',
  partnerContractRead: 'business.partner.contract.read',
  partnerContractCreate: 'business.partner.contract.create',
  partnerContractUpdate: 'business.partner.contract.update',
  partnerSettlementRuleRead: 'business.partner.settlement_rule.read',
  partnerSettlementRuleCreate: 'business.partner.settlement_rule.create',
  partnerSettlementRuleUpdate: 'business.partner.settlement_rule.update',
  partnerAttachmentRead: 'business.partner.attachment.read',
  partnerAttachmentRegister: 'business.partner.attachment.register',
  partnerShippingPresetRead: 'business.partner.shipping_preset.read',
  partnerShippingPresetCreate: 'business.partner.shipping_preset.create',
  partnerShippingPresetUpdate: 'business.partner.shipping_preset.update',
  partnerAuditRead: 'business.partner.audit.read',
  partnerAssignmentOptionRead: 'business.partner.assignment_option.read',
  enterpriseResourceRead: 'business.enterprise_resource.read',
  enterpriseResourceCreate: 'business.enterprise_resource.create',
  enterpriseResourceUpdate: 'business.enterprise_resource.update',
  enterpriseResourceDelete: 'business.enterprise_resource.delete',
  masterDataCurrencyRead: 'system.master_data.currency.read',
  masterDataCurrencyUpdate: 'system.master_data.currency.update',
  masterDataAdministrativeRegionRead:
    'system.master_data.administrative_region.read',
  masterDataOptionRead: 'system.master_data.option.read',
  masterDataItemRead: 'system.master_data.item.read',
  masterDataItemCreate: 'system.master_data.item.create',
  masterDataItemUpdate: 'system.master_data.item.update',
  masterDataItemImport: 'system.master_data.item.import',
  masterDataPortRead: 'system.master_data.port.read',
  masterDataPortCreate: 'system.master_data.port.create',
  masterDataPortUpdate: 'system.master_data.port.update',
  masterDataAirportRead: 'system.master_data.airport.read',
  masterDataAirportCreate: 'system.master_data.airport.create',
  masterDataAirportUpdate: 'system.master_data.airport.update',
  masterDataAirlineRead: 'system.master_data.airline.read',
  masterDataAirlineCreate: 'system.master_data.airline.create',
  masterDataAirlineUpdate: 'system.master_data.airline.update',
  masterDataShippingLineRead: 'system.master_data.shipping_line.read',
  masterDataShippingLineCreate: 'system.master_data.shipping_line.create',
  masterDataShippingLineUpdate: 'system.master_data.shipping_line.update',
  masterDataNumberRuleRead: 'system.master_data.number_rule.read',
  masterDataNumberRuleCreate: 'system.master_data.number_rule.create',
  masterDataNumberRuleUpdate: 'system.master_data.number_rule.update',
  taskRead: 'system.task.read',
  taskRequeue: 'system.task.requeue',
} as const satisfies Record<string, ManifestPermissionKey>;

const orderBusinessCodes: Record<number | string, string | undefined> = {
  1: 'se',
  2: 'si',
  3: 'ae',
  4: 'ai',
  SE: 'se',
  SI: 'si',
  AE: 'ae',
  AI: 'ai',
};

function orderPermission(
  businessType: number | string,
  operation: OrderPermissionOperation,
): ManifestPermissionKey | '' {
  const businessCode = orderBusinessCodes[businessType];
  return businessCode
    ? (`business.order.${businessCode}.${operation}` as ManifestPermissionKey)
    : '';
}

export default function access(
  initialState: { currentUser?: API.CurrentUser } | undefined,
) {
  // 服务端按同一权限的有效角色授权聚合范围；禁止借用其他角色的范围。
  const capabilities = new Map(
    (initialState?.currentUser?.permissionCapabilities ?? []).map((item) => [
      item.key,
      item.dataScope,
    ]),
  );
  const scopeRank: Record<string, number> = {
    organization: 1,
    organization_tree: 2,
    all: 3,
  };
  const has = (
    permission: string,
    minimum: 'organization' | 'all' = 'organization',
  ) =>
    (scopeRank[capabilities.get(permission) ?? ''] ?? 0) >= scopeRank[minimum];
  const hasAny = (...items: string[]) => items.some((item) => has(item));
  // 组织身份来自 auth/me 的 kind（阶段一契约），不复制第二套权限真相：
  // 公共资料维护入口仅系统管理工作台可见。
  const isSystemWorkspace =
    initialState?.currentUser?.currentOrganization?.kind ===
    AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM;
  // 工作台身份仅约束经营入口；具体业务权限继续消费服务端有效权限集。
  const canOperateBusiness =
    initialState?.currentUser?.currentOrganization?.kind ===
    AuthOrganizationKind.ORGANIZATION_KIND_COMPANY;
  const canOperateOrganization = (organizationId?: string) =>
    canOperateBusiness &&
    Boolean(organizationId) &&
    organizationId === initialState?.currentUser?.currentOrganization?.id;
  const canOrder = (
    businessType: number | string,
    operation: OrderPermissionOperation,
  ) => {
    const permission = orderPermission(businessType, operation);
    return canOperateBusiness && permission !== '' && has(permission);
  };

  const result = {
    isAuthenticated: Boolean(initialState?.currentUser),
    isSystemWorkspace,
    canOperateBusiness,
    canOperateOrganization,
    canAccessPlatform: has(permissions.platformAccess),
    canReadOrganizations: has(permissions.organizationRead),
    canCreateOrganizations: has(permissions.organizationCreate),
    canUpdateOrganizations: has(permissions.organizationUpdate),
    canReadUsers: has(permissions.userRead),
    canCreateUsers: has(permissions.userCreate),
    canUpdateUsers: isSystemWorkspace && has(permissions.userUpdate),
    canTerminateUsers: isSystemWorkspace && has(permissions.userTerminate),
    canReadUserMemberships: has(permissions.userRead),
    canManageUserMemberships: has(permissions.userUpdate),
    canAuthorizeWeComUsers:
      isSystemWorkspace && has(permissions.userAuthorizeWeCom, 'all'),
    canAuthorizeDingTalkUsers:
      isSystemWorkspace && has(permissions.userAuthorizeDingTalk, 'all'),
    canManageDingTalkInvitations: has(permissions.userDingTalkInvitationManage),
    canResetUserPasswords:
      isSystemWorkspace && has(permissions.userResetPassword),
    canReadRoles: has(permissions.roleRead),
    canCreateRoles: has(permissions.roleCreate),
    canUpdateRoles: has(permissions.roleUpdate),
    canDeleteRoles: has(permissions.roleDelete),
    canReadPermissions: has(permissions.permissionRead),
    canReadAudit: has(permissions.auditRead),
    canReadExchangeRates: has(permissions.financeExchangeRateRead),
    canCreateExchangeRates: has(permissions.financeExchangeRateCreate),
    canUpdateExchangeRates: has(permissions.financeExchangeRateUpdate),
    canDisableExchangeRates: has(permissions.financeExchangeRateDisable),
    canOverrideFeeExchangeRate:
      canOperateBusiness && has(permissions.financeExchangeRateOverride),
    canReadFeeSettings: has(permissions.financeFeeSettingRead),
    canCreateFeeSettings: has(permissions.financeFeeSettingCreate),
    canUpdateFeeSettings: has(permissions.financeFeeSettingUpdate),
    canAccessFinanceManagement: [
      permissions.financeFeeRead,
      permissions.financeBillRead,
      permissions.financeInvoiceRead,
      permissions.financeCashflowRead,
      permissions.financeVerificationRead,
      permissions.financeNettingRead,
      permissions.financeCommissionRead,
      permissions.financeExchangeRateRead,
      permissions.financeFeeSettingRead,
    ].some((permission) => has(permission)),
    canReadFinanceFees: canOperateBusiness && has(permissions.financeFeeRead),
    canManageFinanceFeeTags:
      canOperateBusiness && has(permissions.financeFeeTag),
    canReadFinanceBills: canOperateBusiness && has(permissions.financeBillRead),
    canCreateFinanceBills:
      canOperateBusiness && has(permissions.financeBillCreate),
    canUpdateFinanceBills:
      canOperateBusiness && has(permissions.financeBillUpdate),
    canConfirmFinanceBills:
      canOperateBusiness && has(permissions.financeBillConfirm),
    canReadFinanceInvoices:
      canOperateBusiness && has(permissions.financeInvoiceRead),
    canCreateFinanceInvoices:
      canOperateBusiness && has(permissions.financeInvoiceCreate),
    canUpdateFinanceInvoices:
      canOperateBusiness && has(permissions.financeInvoiceUpdate),
    canReadFinanceCashflows:
      canOperateBusiness && has(permissions.financeCashflowRead),
    canCreateFinanceCashflows:
      canOperateBusiness && has(permissions.financeCashflowCreate),
    canUpdateFinanceCashflows:
      canOperateBusiness && has(permissions.financeCashflowUpdate),
    canReadFinanceVerifications:
      canOperateBusiness && has(permissions.financeVerificationRead),
    canCreateFinanceVerifications:
      canOperateBusiness && has(permissions.financeVerificationCreate),
    canReverseFinanceVerifications:
      canOperateBusiness && has(permissions.financeVerificationReverse),
    canReadFinanceNettings:
      canOperateBusiness && has(permissions.financeNettingRead),
    canCreateFinanceNettings:
      canOperateBusiness && has(permissions.financeNettingCreate),
    canConfirmFinanceNettings:
      canOperateBusiness && has(permissions.financeNettingConfirm),
    canReverseFinanceNettings:
      canOperateBusiness && has(permissions.financeNettingReverse),
    canReadFinanceCommissions:
      canOperateBusiness && has(permissions.financeCommissionRead),
    canManageFinanceCommissions:
      canOperateBusiness && has(permissions.financeCommissionManage),
    canConfigureFinanceCommissions:
      canOperateBusiness && has(permissions.financeCommissionConfigure),
    canExportFinanceCommissions:
      canOperateBusiness && has(permissions.financeCommissionExport),
    canReadPartners: canOperateBusiness && has(permissions.partnerRead),
    canReadEnterpriseResources:
      canOperateBusiness && has(permissions.enterpriseResourceRead),
    canCreateEnterpriseResources:
      canOperateBusiness && has(permissions.enterpriseResourceCreate),
    canUpdateEnterpriseResources:
      canOperateBusiness && has(permissions.enterpriseResourceUpdate),
    canDeleteEnterpriseResources:
      canOperateBusiness && has(permissions.enterpriseResourceDelete),
    canCreatePartners: canOperateBusiness && has(permissions.partnerCreate),
    canUpdatePartners: canOperateBusiness && has(permissions.partnerUpdate),
    canBlacklistPartners:
      canOperateBusiness && has(permissions.partnerBlacklist),
    canImportPartners: canOperateBusiness && has(permissions.partnerImport),
    canExportPartners: canOperateBusiness && has(permissions.partnerExport),
    canReadPartnerAccounts:
      canOperateBusiness && has(permissions.partnerAccountRead),
    canCreatePartnerAccounts:
      canOperateBusiness && has(permissions.partnerAccountCreate),
    canUpdatePartnerAccounts:
      canOperateBusiness && has(permissions.partnerAccountUpdate),
    canReadPartnerContracts:
      canOperateBusiness && has(permissions.partnerContractRead),
    canCreatePartnerContracts:
      canOperateBusiness && has(permissions.partnerContractCreate),
    canUpdatePartnerContracts:
      canOperateBusiness && has(permissions.partnerContractUpdate),
    canReadPartnerSettlementRules:
      canOperateBusiness && has(permissions.partnerSettlementRuleRead),
    canCreatePartnerSettlementRules:
      canOperateBusiness && has(permissions.partnerSettlementRuleCreate),
    canUpdatePartnerSettlementRules:
      canOperateBusiness && has(permissions.partnerSettlementRuleUpdate),
    canReadPartnerAttachments:
      canOperateBusiness && has(permissions.partnerAttachmentRead),
    canRegisterPartnerAttachments:
      canOperateBusiness && has(permissions.partnerAttachmentRegister),
    canReadPartnerShippingPresets:
      canOperateBusiness && has(permissions.partnerShippingPresetRead),
    canCreatePartnerShippingPresets:
      canOperateBusiness && has(permissions.partnerShippingPresetCreate),
    canUpdatePartnerShippingPresets:
      canOperateBusiness && has(permissions.partnerShippingPresetUpdate),
    canReadPartnerAudit:
      canOperateBusiness && has(permissions.partnerAuditRead),
    canReadPartnerAssignmentOptions:
      canOperateBusiness && has(permissions.partnerAssignmentOptionRead),
    canReadMasterDataCurrencies: has(permissions.masterDataCurrencyRead),
    canUpdateMasterDataCurrencies:
      has(permissions.masterDataCurrencyUpdate) ||
      has(permissions.financeFeeSettingUpdate) ||
      has(permissions.financeFeeSettingCreate),
    canReadMasterDataAdministrativeRegions: has(
      permissions.masterDataAdministrativeRegionRead,
    ),
    canReadMasterDataOptions: has(permissions.masterDataOptionRead),
    canReadMasterDataItems: has(permissions.masterDataItemRead),
    canCreateMasterDataItems: has(permissions.masterDataItemCreate),
    canUpdateMasterDataItems: has(permissions.masterDataItemUpdate),
    canImportMasterDataItems: has(permissions.masterDataItemImport),
    canReadMasterDataPorts: has(permissions.masterDataPortRead),
    canCreateMasterDataPorts: has(permissions.masterDataPortCreate),
    canUpdateMasterDataPorts: has(permissions.masterDataPortUpdate),
    canReadMasterDataAirports: has(permissions.masterDataAirportRead),
    canCreateMasterDataAirports: has(permissions.masterDataAirportCreate),
    canUpdateMasterDataAirports: has(permissions.masterDataAirportUpdate),
    canReadMasterDataAirlines: has(permissions.masterDataAirlineRead),
    canCreateMasterDataAirlines: has(permissions.masterDataAirlineCreate),
    canUpdateMasterDataAirlines: has(permissions.masterDataAirlineUpdate),
    canReadMasterDataShippingLines: has(permissions.masterDataShippingLineRead),
    canCreateMasterDataShippingLines: has(
      permissions.masterDataShippingLineCreate,
    ),
    canUpdateMasterDataShippingLines: has(
      permissions.masterDataShippingLineUpdate,
    ),
    canReadMasterDataNumberRules:
      canOperateBusiness && has(permissions.masterDataNumberRuleRead),
    canCreateMasterDataNumberRules:
      canOperateBusiness && has(permissions.masterDataNumberRuleCreate),
    canUpdateMasterDataNumberRules:
      canOperateBusiness && has(permissions.masterDataNumberRuleUpdate),
    canReadTasks: has(permissions.taskRead),
    canRequeueTasks: has(permissions.taskRequeue),
  };

  return {
    ...result,
    canReadMasterData: hasAny(
      ...Object.entries(permissions)
        .filter(([key]) => key.startsWith('masterData') && key.endsWith('Read'))
        .map(([, value]) => value),
    ),
    canManageMasterData: hasAny(
      ...Object.entries(permissions)
        .filter(
          ([key]) => key.startsWith('masterData') && !key.endsWith('Read'),
        )
        .map(([, value]) => value),
    ),
    canManageOrganizations:
      result.canCreateOrganizations || result.canUpdateOrganizations,
    canManageUsers:
      result.canCreateUsers ||
      result.canUpdateUsers ||
      result.canManageUserMemberships ||
      result.canTerminateUsers ||
      result.canAuthorizeWeComUsers ||
      result.canAuthorizeDingTalkUsers ||
      result.canManageDingTalkInvitations ||
      result.canResetUserPasswords,
    canManageRoles:
      result.canCreateRoles || result.canUpdateRoles || result.canDeleteRoles,
    canManagePartners:
      result.canCreatePartners ||
      result.canUpdatePartners ||
      result.canBlacklistPartners ||
      result.canImportPartners,
    canOrder,
    canConfirmAnyOrderFees: [1, 2, 3, 4].some((businessType) =>
      canOrder(businessType, 'fee.update'),
    ),
    canReadAnyOrders: [1, 2, 3, 4].some((businessType) =>
      canOrder(businessType, 'read'),
    ),
    canCreateAnyOrders: [1, 2, 3, 4].some((businessType) =>
      canOrder(businessType, 'create'),
    ),
    // 费用录入工作台路由守卫：费用录入人群（fee.read）之外，锁后补录
    // 审批人仅持 lock 权限码（后端按实时 lock grant 判定，不要求 fee.read），
    // 工作台「前往处理」入口同样需要放行。
    canAccessAnyOrderFees: [1, 2, 3, 4].some(
      (businessType) =>
        canOrder(businessType, 'fee.read') || canOrder(businessType, 'lock'),
    ),
    canReadSEOrders: canOrder(1, 'read'),
    canSplitSEOrders: canOrder(1, 'split'),
    canReadSIOrders: canOrder(2, 'read'),
    canReadAEOrders: canOrder(3, 'read'),
    canReadAIOrders: canOrder(4, 'read'),
    canManageTasks: result.canRequeueTasks,
    canRequeueTask: (kind?: number) => {
      if (!result.canRequeueTasks) return false;
      switch (kind) {
        case BackgroundTaskKind.BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT:
        case BackgroundTaskKind.BACKGROUND_TASK_KIND_UNLOCODE_IMPORT:
        case BackgroundTaskKind.BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION:
          return true;
        case BackgroundTaskKind.BACKGROUND_TASK_KIND_ORDER_REMINDER:
        case BackgroundTaskKind.BACKGROUND_TASK_KIND_INTEGRATION:
          return canOperateBusiness;
        default:
          return false;
      }
    },
    canReadParameterSettings:
      result.canReadMasterDataNumberRules ||
      result.canReadFeeSettings ||
      result.canReadExchangeRates,
  };
}
