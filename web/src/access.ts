const permissions = {
  platformAccess: 'system.platform.access',
  organizationRead: 'system.organization.read',
  organizationCreate: 'system.organization.create',
  organizationUpdate: 'system.organization.update',
  userRead: 'system.user.read',
  userCreate: 'system.user.create',
  userUpdate: 'system.user.update',
  userDelete: 'system.user.delete',
  userAuthorizeWeCom: 'system.user.authorize_wecom',
  userAuthorizeDingTalk: 'system.user.authorize_dingtalk',
  userResetPassword: 'system.user.reset_password',
  roleRead: 'system.role.read',
  roleCreate: 'system.role.create',
  roleUpdate: 'system.role.update',
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
  masterDataCurrencyRead: 'system.master_data.currency.read',
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
  masterDataStatusTemplateRead: 'system.master_data.status_template.read',
  masterDataStatusTemplateCreate: 'system.master_data.status_template.create',
  masterDataStatusTemplatePublish: 'system.master_data.status_template.publish',
  masterDataStatusTemplateSetDefault:
    'system.master_data.status_template.set_default',
  masterDataMilestoneTemplateRead: 'system.master_data.milestone_template.read',
  masterDataMilestoneTemplateCreate:
    'system.master_data.milestone_template.create',
  masterDataMilestoneTemplatePublish:
    'system.master_data.milestone_template.publish',
  masterDataMilestoneTemplateSetDefault:
    'system.master_data.milestone_template.set_default',
  taskRead: 'system.task.read',
  taskRequeue: 'system.task.requeue',
} as const;

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

function orderPermission(businessType: number | string, operation: string) {
  const businessCode = orderBusinessCodes[businessType];
  return businessCode ? `business.order.${businessCode}.${operation}` : '';
}

export default function access(
  initialState: { currentUser?: API.CurrentUser } | undefined,
) {
  const granted = new Set(initialState?.currentUser?.permissions ?? []);
  const roleScopes = initialState?.currentUser?.roleScopes ?? [];
  const has = (permission: string) => granted.has(permission);
  const hasAny = (...items: string[]) => items.some(has);
  const hasScope = (minimum: string) => {
    const rank: Record<string, number> = {
      self: 1,
      organization: 2,
      organization_tree: 3,
      all: 4,
    };
    return roleScopes.some(
      (scope) => (rank[scope.dataScope ?? ''] ?? 0) >= (rank[minimum] ?? 0),
    );
  };
  const inOrganization = hasScope('organization');
  const inAll = hasScope('all');
  const canOrder = (businessType: number | string, operation: string) => {
    const permission = orderPermission(businessType, operation);
    return permission !== '' && has(permission) && inOrganization;
  };

  const result = {
    isAuthenticated: Boolean(initialState?.currentUser),
    canAccessPlatform: has(permissions.platformAccess),
    canReadOrganizations: has(permissions.organizationRead) && inAll,
    canCreateOrganizations: has(permissions.organizationCreate) && inAll,
    canUpdateOrganizations: has(permissions.organizationUpdate) && inAll,
    canReadUsers: has(permissions.userRead) && inOrganization,
    canCreateUsers: has(permissions.userCreate) && inOrganization,
    canUpdateUsers: has(permissions.userUpdate) && inOrganization,
    canDeleteUsers: has(permissions.userDelete) && inOrganization,
    canAuthorizeWeComUsers: has(permissions.userAuthorizeWeCom) && inAll,
    canAuthorizeDingTalkUsers: has(permissions.userAuthorizeDingTalk) && inAll,
    canResetUserPasswords: has(permissions.userResetPassword) && inOrganization,
    canReadRoles: has(permissions.roleRead) && inOrganization,
    canCreateRoles: has(permissions.roleCreate) && inOrganization,
    canUpdateRoles: has(permissions.roleUpdate) && inOrganization,
    canReadPermissions: has(permissions.permissionRead) && inOrganization,
    canReadAudit: has(permissions.auditRead) && inOrganization,
    canReadExchangeRates:
      has(permissions.financeExchangeRateRead) && inOrganization,
    canCreateExchangeRates:
      has(permissions.financeExchangeRateCreate) && inOrganization,
    canUpdateExchangeRates:
      has(permissions.financeExchangeRateUpdate) && inOrganization,
    canDisableExchangeRates:
      has(permissions.financeExchangeRateDisable) && inOrganization,
    canOverrideFeeExchangeRate:
      has(permissions.financeExchangeRateOverride) && inOrganization,
    canReadFeeSettings:
      has(permissions.financeFeeSettingRead) && inOrganization,
    canCreateFeeSettings:
      has(permissions.financeFeeSettingCreate) && inOrganization,
    canUpdateFeeSettings:
      has(permissions.financeFeeSettingUpdate) && inOrganization,
    canReadPartners: has(permissions.partnerRead) && inOrganization,
    canCreatePartners: has(permissions.partnerCreate) && inOrganization,
    canUpdatePartners: has(permissions.partnerUpdate) && inOrganization,
    canBlacklistPartners: has(permissions.partnerBlacklist) && inOrganization,
    canImportPartners: has(permissions.partnerImport) && inOrganization,
    canExportPartners: has(permissions.partnerExport) && inOrganization,
    canReadPartnerAccounts:
      has(permissions.partnerAccountRead) && inOrganization,
    canCreatePartnerAccounts:
      has(permissions.partnerAccountCreate) && inOrganization,
    canUpdatePartnerAccounts:
      has(permissions.partnerAccountUpdate) && inOrganization,
    canReadPartnerContracts:
      has(permissions.partnerContractRead) && inOrganization,
    canCreatePartnerContracts:
      has(permissions.partnerContractCreate) && inOrganization,
    canUpdatePartnerContracts:
      has(permissions.partnerContractUpdate) && inOrganization,
    canReadPartnerSettlementRules:
      has(permissions.partnerSettlementRuleRead) && inOrganization,
    canCreatePartnerSettlementRules:
      has(permissions.partnerSettlementRuleCreate) && inOrganization,
    canUpdatePartnerSettlementRules:
      has(permissions.partnerSettlementRuleUpdate) && inOrganization,
    canReadPartnerAttachments:
      has(permissions.partnerAttachmentRead) && inOrganization,
    canRegisterPartnerAttachments:
      has(permissions.partnerAttachmentRegister) && inOrganization,
    canReadPartnerShippingPresets:
      has(permissions.partnerShippingPresetRead) && inOrganization,
    canCreatePartnerShippingPresets:
      has(permissions.partnerShippingPresetCreate) && inOrganization,
    canUpdatePartnerShippingPresets:
      has(permissions.partnerShippingPresetUpdate) && inOrganization,
    canReadPartnerAudit: has(permissions.partnerAuditRead) && inOrganization,
    canReadPartnerAssignmentOptions:
      has(permissions.partnerAssignmentOptionRead) && inOrganization,
    canReadMasterDataCurrencies:
      has(permissions.masterDataCurrencyRead) && inOrganization,
    canReadMasterDataAdministrativeRegions:
      has(permissions.masterDataAdministrativeRegionRead) && inOrganization,
    canReadMasterDataOptions:
      has(permissions.masterDataOptionRead) && inOrganization,
    canReadMasterDataItems:
      has(permissions.masterDataItemRead) && inOrganization,
    canCreateMasterDataItems:
      has(permissions.masterDataItemCreate) && inOrganization,
    canUpdateMasterDataItems:
      has(permissions.masterDataItemUpdate) && inOrganization,
    canImportMasterDataItems:
      has(permissions.masterDataItemImport) && inOrganization,
    canReadMasterDataPorts:
      has(permissions.masterDataPortRead) && inOrganization,
    canCreateMasterDataPorts:
      has(permissions.masterDataPortCreate) && inOrganization,
    canUpdateMasterDataPorts:
      has(permissions.masterDataPortUpdate) && inOrganization,
    canReadMasterDataAirports:
      has(permissions.masterDataAirportRead) && inOrganization,
    canCreateMasterDataAirports:
      has(permissions.masterDataAirportCreate) && inOrganization,
    canUpdateMasterDataAirports:
      has(permissions.masterDataAirportUpdate) && inOrganization,
    canReadMasterDataAirlines:
      has(permissions.masterDataAirlineRead) && inOrganization,
    canCreateMasterDataAirlines:
      has(permissions.masterDataAirlineCreate) && inOrganization,
    canUpdateMasterDataAirlines:
      has(permissions.masterDataAirlineUpdate) && inOrganization,
    canReadMasterDataShippingLines:
      has(permissions.masterDataShippingLineRead) && inOrganization,
    canCreateMasterDataShippingLines:
      has(permissions.masterDataShippingLineCreate) && inOrganization,
    canUpdateMasterDataShippingLines:
      has(permissions.masterDataShippingLineUpdate) && inOrganization,
    canReadMasterDataNumberRules:
      has(permissions.masterDataNumberRuleRead) && inOrganization,
    canCreateMasterDataNumberRules:
      has(permissions.masterDataNumberRuleCreate) && inOrganization,
    canUpdateMasterDataNumberRules:
      has(permissions.masterDataNumberRuleUpdate) && inOrganization,
    canReadMasterDataStatusTemplates:
      has(permissions.masterDataStatusTemplateRead) && inOrganization,
    canCreateMasterDataStatusTemplates:
      has(permissions.masterDataStatusTemplateCreate) && inOrganization,
    canPublishMasterDataStatusTemplates:
      has(permissions.masterDataStatusTemplatePublish) && inOrganization,
    canSetDefaultMasterDataStatusTemplates:
      has(permissions.masterDataStatusTemplateSetDefault) && inOrganization,
    canReadMasterDataMilestoneTemplates:
      has(permissions.masterDataMilestoneTemplateRead) && inOrganization,
    canCreateMasterDataMilestoneTemplates:
      has(permissions.masterDataMilestoneTemplateCreate) && inOrganization,
    canPublishMasterDataMilestoneTemplates:
      has(permissions.masterDataMilestoneTemplatePublish) && inOrganization,
    canSetDefaultMasterDataMilestoneTemplates:
      has(permissions.masterDataMilestoneTemplateSetDefault) && inOrganization,
    canReadTasks: has(permissions.taskRead) && inOrganization,
    canRequeueTasks: has(permissions.taskRequeue) && inOrganization,
  };

  return {
    ...result,
    canReadMasterData:
      hasAny(
        ...Object.entries(permissions)
          .filter(
            ([key]) => key.startsWith('masterData') && key.endsWith('Read'),
          )
          .map(([, value]) => value),
      ) && inOrganization,
    canManageMasterData:
      hasAny(
        ...Object.entries(permissions)
          .filter(
            ([key]) => key.startsWith('masterData') && !key.endsWith('Read'),
          )
          .map(([, value]) => value),
      ) && inOrganization,
    canManageOrganizations:
      result.canCreateOrganizations || result.canUpdateOrganizations,
    canManageUsers:
      result.canCreateUsers ||
      result.canUpdateUsers ||
      result.canDeleteUsers ||
      result.canAuthorizeWeComUsers ||
      result.canAuthorizeDingTalkUsers ||
      result.canResetUserPasswords,
    canManageRoles: result.canCreateRoles || result.canUpdateRoles,
    canManagePartners:
      result.canCreatePartners ||
      result.canUpdatePartners ||
      result.canBlacklistPartners ||
      result.canImportPartners,
    canOrder,
    canReadAnyOrders: [1, 2, 3, 4].some((businessType) =>
      canOrder(businessType, 'read'),
    ),
    canReadSEOrders: canOrder(1, 'read'),
    canReadSIOrders: canOrder(2, 'read'),
    canReadAEOrders: canOrder(3, 'read'),
    canReadAIOrders: canOrder(4, 'read'),
    canManageTasks: result.canRequeueTasks,
    canReadParameterSettings:
      result.canReadMasterDataNumberRules ||
      result.canReadFeeSettings ||
      result.canReadExchangeRates ||
      result.canReadMasterDataMilestoneTemplates,
  };
}
