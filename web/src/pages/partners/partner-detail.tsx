import { CheckCircleOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProForm } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { history } from '@/router/history';
import { useAccess } from '@/app/access';
import { useLocation, useParams, useSearchParams } from 'react-router';
import { App, Button, Space, Spin, Tag, Typography } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  PageHeaderShell,
  StickyFooterBar,
  scrollToFirstFormError,
} from '@/components/ui';
import {
  PartnerBusinessType,
  PartnerCustomerType,
  PartnerRoleType,
  PartnerSettlementBase,
  PartnerSettlementMethod,
  PartnerStatementMode,
} from '@/enums.generated';
import { adminServiceListUsers } from '@/services/roncin/adminService';
import {
  partnerServiceCreatePartner,
  partnerServiceGetPartner,
  partnerServiceListPartnerAssignmentOptions,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import { unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate } from '@/utils/format';
import { getCurrencyOptions } from '@/utils/options';
import AccountsSection from './components/AccountsSection';
import BasicInfoSection from './components/BasicInfoSection';
import type { ContactItem } from './components/ContactCardList';
import ContactsSection from './components/ContactsSection';

/** 识别 antd 表单校验失败异常（validateFields 抛出的 ValidateErrorEntity）。 */
function isFormValidationError(
  error: unknown,
): error is { errorFields: { name: (string | number)[]; errors: string[] }[] } {
  if (typeof error !== 'object' || error === null) return false;
  return Boolean((error as { errorFields?: unknown }).errorFields);
}

import ContractsSection from './components/ContractsSection';
import InterestRuleModal, {
  type InterestRuleValues,
} from './components/InterestRuleModal';
import LogsSection from './components/LogsSection';
import PartnerAnchorNav from './components/PartnerAnchorNav';
import PresetsSection from './components/PresetsSection';
import RemarkSection from './components/RemarkSection';
import SettlementSection from './components/SettlementSection';

const { Text } = Typography;

export const UUID_REGEX =
  /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

export function isValidPartnerId(id?: string): boolean {
  return typeof id === 'string' && UUID_REGEX.test(id);
}

export default function PartnerDetailPage() {
  const { message } = App.useApp();
  const access = useAccess();
  const params = useParams<{ id?: string }>();
  const location = useLocation();
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const [saving, setSaving] = useState(false);

  // Contacts state for ContactCardList
  const [contacts, setContacts] = useState<ContactItem[]>([]);

  // Aliases state
  const [aliases, setAliases] = useState<string[]>([]);
  const [newAliasInput, setNewAliasInput] = useState('');

  // Interest Rule state
  const [interestModalOpen, setInterestModalOpen] = useState(false);
  const [interestRule, setInterestRule] = useState<InterestRuleValues>({
    enabled: false,
    dailyRateBp: 5,
    graceDays: 3,
    calcMode: 'daily_simple',
    remark: '',
  });

  // Collapsible active keys (all expanded by default)
  const [activeCollapseKeys, setActiveCollapseKeys] = useState<string[]>([
    'basic',
    'settlement',
    'accounts',
    'contacts',
    'presets',
    'contracts',
    'remark',
    'logs',
  ]);
  // 表单导航分节错误统计
  const [sectionErrors, setSectionErrors] = useState<Record<string, number>>(
    {},
  );

  // 表单导航浮层折叠状态：展开时内容区预留 164px 右侧空间，避免遮挡控件
  const [navCollapsed, setNavCollapsed] = useState(true);

  // Detect roleType from pathname（按路由段结构解析：/partners/{roleSegment}/...）
  const { roleType, roleLabel, listUrl } = useMemo(() => {
    const roleSegment = location.pathname.split('/')[2];
    if (roleSegment === 'suppliers') {
      return {
        roleType: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
        roleLabel: '供应商',
        listUrl: '/partners/suppliers',
      };
    }
    if (roleSegment === 'foreign-agents') {
      return {
        roleType: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
        roleLabel: '国外代理',
        listUrl: '/partners/foreign-agents',
      };
    }
    return {
      roleType: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      roleLabel: '客户',
      listUrl: '/partners/customers',
    };
  }, [location.pathname]);

  const rawId = params.id;
  const isCreate = !rawId || rawId === 'create';
  const hasValidUuid = isValidPartnerId(rawId);
  const partnerId = !isCreate && hasValidUuid ? rawId : undefined;
  const isInvalidId = !isCreate && Boolean(rawId) && !hasValidUuid;

  const canReadSettlementRules = Boolean(
    access.canReadPartnerSettlementRules || access.canManagePartners,
  );
  const canReadContracts = Boolean(
    access.canReadPartnerContracts || access.canManagePartners,
  );
  const canReadShippingPresets = Boolean(
    access.canReadPartnerShippingPresets || access.canManagePartners,
  );
  const canReadAudit = Boolean(
    access.canReadPartnerAudit || access.canManagePartners,
  );
  const canReadAccounts = Boolean(
    access.canReadPartnerAccounts || access.canManagePartners,
  );

  // ------------------------------------------------------------------
  // 服务端状态统一由 React Query 管理（state-management.md 唯一模式）
  // ------------------------------------------------------------------

  // 档案详情：仅编辑模式按档案标识拉取；失败提示与旧 catch 文案一致
  const partnerDetailQuery = useQuery({
    queryKey: ['partner-detail', 'profile', { id: partnerId }],
    enabled: Boolean(partnerId),
    queryFn: async () => {
      if (!partnerId) {
        // enabled 已保证；此处仅为类型收窄兜底。
        throw new Error('缺少档案标识');
      }
      const res = await partnerServiceGetPartner({ id: partnerId });
      return res.data ?? null;
    },
    meta: { errorMessage: '加载档案详情失败' },
  });
  const partner = partnerDetailQuery.data ?? undefined;

  // 结算规则：仅编辑模式且有读权限时拉取；失败静默，不阻断档案主体渲染
  const settlementRuleQuery = useQuery({
    queryKey: ['partner-detail', 'settlement-rule', { partnerId, roleType }],
    enabled: Boolean(partnerId) && canReadSettlementRules,
    queryFn: async () => {
      if (!partnerId) {
        throw new Error('缺少档案标识');
      }
      const res = await partnerServiceListPartnerSettlementRules({
        partnerId,
        roleType,
      });
      return unwrapList(res)[0] ?? null;
    },
    meta: { silent: true },
  });

  // 编辑模式加载态：档案请求进行中时全页 Spin（与旧 loading state 行为一致）
  const loading = Boolean(partnerId) && partnerDetailQuery.isPending;

  // 辅助选项（用户、币种、人员归属）：失败只提示不阻断；
  // 全部结算后合并提示一次，与旧 Promise.allSettled 行为一致。
  const usersQuery = useQuery({
    queryKey: ['partner-detail', 'options', 'users'],
    queryFn: async () => {
      const res = await adminServiceListUsers(
        { page: 1, pageSize: 200 },
        { skipErrorHandler: true },
      );
      return res.data ?? [];
    },
    meta: { silent: true },
  });
  const currencyOptionsQuery = useQuery({
    queryKey: ['partner-detail', 'options', 'currencies'],
    queryFn: () => getCurrencyOptions(),
    meta: { silent: true },
  });
  const assignmentOptionsQuery = useQuery({
    queryKey: ['partner-detail', 'options', 'assignments'],
    queryFn: async () => {
      const res = await partnerServiceListPartnerAssignmentOptions({
        skipErrorHandler: true,
      });
      return res.data ?? [];
    },
    meta: { silent: true },
  });
  const users = usersQuery.data ?? [];
  const assignmentOptions = assignmentOptionsQuery.data ?? [];
  const currencyOptions = currencyOptionsQuery.data ?? [];

  const failedOptionLabels = [
    usersQuery.isError ? '用户' : '',
    currencyOptionsQuery.isError ? '币种' : '',
    assignmentOptionsQuery.isError ? '人员归属' : '',
  ].filter(Boolean);
  const failedOptionSummary = failedOptionLabels.join('、');
  const optionsSettled =
    !usersQuery.isPending &&
    !currencyOptionsQuery.isPending &&
    !assignmentOptionsQuery.isPending;
  useEffect(() => {
    if (optionsSettled && failedOptionSummary) {
      message.warning(`${failedOptionSummary}选项加载失败`);
    }
  }, [optionsSettled, failedOptionSummary, message]);

  const canOperate = isCreate
    ? access.canOperateBusiness
    : access.canOperateOrganization(partner?.organizationId);

  const canSave =
    canOperate &&
    (isCreate
      ? Boolean(access.canCreatePartners || access.canManagePartners)
      : Boolean(access.canUpdatePartners || access.canManagePartners));

  const canSaveSettlement =
    canOperate &&
    (isCreate
      ? Boolean(
          access.canCreatePartnerSettlementRules || access.canManagePartners,
        )
      : Boolean(
          access.canUpdatePartnerSettlementRules || access.canManagePartners,
        ));

  useEffect(() => {
    if (isInvalidId) {
      message.error('无效的档案标识，已返回列表');
      history.replace(listUrl);
    }
  }, [isInvalidId, listUrl, message]);

  // 创建模式读取 legalName 查询参数预填（订单表单快捷新增「添加公司详情」携带）；
  // 编辑模式忽略该参数，不覆盖已加载的公司抬头。
  const [searchParams] = useSearchParams();
  const prefillLegalName = isCreate
    ? (searchParams.get('legalName') ?? '').trim()
    : '';

  // 创建模式（含非法路由）表单默认值：跟随当前地址重建；携带 legalName
  // 查询参数时预填公司抬头，参数移除或变化时按新地址重建，不残留上一条地址的预填。
  useEffect(() => {
    if (partnerId) return;
    setContacts([]);
    setAliases([]);
    formRef.current?.resetFields();
    formRef.current?.setFieldsValue({
      enabled: true,
      isCasual: false,
      nature: roleLabel,
      roleTypes: [roleType],
      customerType: PartnerCustomerType.PARTNER_CUSTOMER_TYPE_DIRECT,
      customerTypes: [PartnerCustomerType.PARTNER_CUSTOMER_TYPE_DIRECT],
      developmentMethod: '自主开发',
      businessTypes: [PartnerBusinessType.PARTNER_BUSINESS_TYPE_SE],
      statementMode: PartnerStatementMode.PARTNER_STATEMENT_MODE_SINGLE,
      settlementMethod:
        PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BY_TICKET,
      settlementBase: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_BILL_DATE,
      settlementDay: 25,
      settlementCurrency: 'CNY',
      creditDays: 30,
      legalName: prefillLegalName,
    });
  }, [partnerId, prefillLegalName, roleLabel, roleType]);

  // 编辑模式回填：档案与结算规则都出结果后一次性填表；结算规则请求失败
  // （静默）时按「无规则」回填，仅填档案字段，不阻断主体渲染。
  useEffect(() => {
    if (!partnerId) return;
    const p = partnerDetailQuery.data;
    if (!p) return;
    // 有读权限时等结算规则请求出结果再统一回填，避免结算字段先空后有的闪烁
    if (canReadSettlementRules && settlementRuleQuery.isPending) return;
    const currentRule = settlementRuleQuery.data ?? undefined;

    const profile = p.profile || {};
    const assignments = p.assignments || [];

    const findAssignment = (role: number, index = 0) => {
      const item = assignments
        .filter((assignment) => assignment.role === role)
        .sort((left, right) => (left.sortOrder ?? 0) - (right.sortOrder ?? 0))[
        index
      ];
      return { userId: item?.userId };
    };

    const regionCodes: string[] = [];
    if (profile.provinceCode) regionCodes.push(profile.provinceCode);
    if (profile.cityCode) regionCodes.push(profile.cityCode);
    if (profile.districtCode) regionCodes.push(profile.districtCode);

    // Aliases
    const loadedAliases = (p.aliases || [])
      .map((a) => a.aliasName || '')
      .filter(Boolean);
    setAliases(loadedAliases);

    // Contacts
    const loadedContacts: ContactItem[] = (p.contacts || []).map((c) => ({
      id: c.id,
      name: c.name || '',
      phone: c.phone,
      email: c.email,
      note: c.note,
      isPrimary: c.isPrimary,
    }));
    setContacts(loadedContacts);

    // Credit Limit conversion
    const creditAmount = currentRule?.creditLimitMinor;
    const isForeign =
      roleType === PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT;

    formRef.current?.setFieldsValue({
      code: p.code,
      legalName: p.legalName,
      unifiedSocialCreditCode: isForeign
        ? undefined
        : p.unifiedSocialCreditCode,
      enabled: p.enabled ?? true,
      isCasual: isForeign ? false : (p.isCasual ?? false),
      regionCodes: regionCodes.length > 0 ? regionCodes : undefined,
      addressDetail: profile.addressDetail || p.registeredAddress,
      nameEn: profile.nameEn || (isForeign ? p.legalName : undefined),
      addressEn:
        profile.addressEn || (isForeign ? p.registeredAddress : undefined),
      nature: profile.nature || roleLabel,
      roleTypes:
        (p.roles ?? []).filter((r) => r.enabled).map((r) => r.type as number)
          .length > 0
          ? (p.roles ?? [])
              .filter((r) => r.enabled)
              .map((r) => r.type as number)
          : [roleType],
      customerType:
        profile.customerTypes?.[0] ||
        PartnerCustomerType.PARTNER_CUSTOMER_TYPE_DIRECT,
      customerTypes: profile.customerTypes || [1],
      developmentMethod: profile.developmentMethod || '自主开发',
      businessTypes: profile.businessTypes || [1],
      remark: profile.remark,

      // 责任人员只选择人员，归属公司由服务端固定为档案所属公司。
      assignCreatorUser: findAssignment(1).userId,
      assignOperatorUser: findAssignment(2).userId,
      assignSalesUser: findAssignment(3).userId,
      assignServiceUser: findAssignment(4).userId,
      assignFinanceUser: findAssignment(5).userId,
      assignCommercialUser: findAssignment(6).userId,
      assignContactUser: findAssignment(7).userId,
      assignContact2User: findAssignment(7, 1).userId,
      assignDocUser: findAssignment(8).userId,

      // Settlement Info
      ...(currentRule
        ? {
            statementMode: currentRule.statementMode ?? 1,
            settlementMethod: currentRule.settlementMethod ?? 1,
            settlementBase: currentRule.settlementBase ?? 1,
            settlementDay: currentRule.settlementDay ?? 25,
            settlementCurrency: currentRule.settlementCurrency ?? 'CNY',
            creditDays: currentRule.settlementCycleDays ?? 30,
            creditLimit: creditAmount,
            paymentTermsDays: currentRule.paymentTermsDays,
          }
        : {}),
    });
  }, [
    partnerId,
    partnerDetailQuery.data,
    settlementRuleQuery.data,
    settlementRuleQuery.isPending,
    canReadSettlementRules,
    roleType,
    roleLabel,
  ]);

  // User and Organization Select Options
  const userSelectOptions = useMemo(() => {
    if (assignmentOptions.length > 0) {
      const map = new Map<string, string>();
      for (const item of assignmentOptions) {
        if (item.userId && item.displayName && !map.has(item.userId)) {
          map.set(item.userId, item.displayName);
        }
      }
      if (map.size > 0) {
        return Array.from(map.entries()).map(([value, label]) => ({
          label,
          value,
        }));
      }
    }
    return users.map((u) => ({
      label: `${u.displayName || u.username} (${u.username})`,
      value: u.id ?? '',
    }));
  }, [assignmentOptions, users]);

  // Tianyancha Verify
  const handleTianyanchaVerify = () => {
    const legalName = formRef.current?.getFieldValue('legalName');
    if (!legalName?.trim()) {
      message.warning('请先填写公司抬头再进行校验');
      return;
    }
    const targetUrl = `https://www.tianyancha.com/nsearch?key=${encodeURIComponent(
      legalName.trim(),
    )}`;
    window.open(targetUrl, '_blank', 'noopener,noreferrer');
  };

  // Alias add/remove
  const handleAddAlias = () => {
    if (!newAliasInput.trim()) return;
    if (aliases.includes(newAliasInput.trim())) {
      message.warning('该别名已存在');
      return;
    }
    setAliases([...aliases, newAliasInput.trim()]);
    setNewAliasInput('');
  };

  const handleRemoveAlias = (tagToRemove: string) => {
    setAliases(aliases.filter((t) => t !== tagToRemove));
  };

  // Submit Handler (Atomic Save)
  const handleSubmit = async () => {
    if (!canSave) {
      message.error(isCreate ? '暂无创建权限' : '暂无编辑权限');
      return;
    }

    try {
      const values = await formRef.current?.validateFields();
      if (!values) return;

      setSaving(true);

      const regionCodes: string[] = values.regionCodes || [];
      const provinceCode = regionCodes[0] || '';
      const cityCode = regionCodes[1] || '';
      const districtCode = regionCodes[2] || '';

      const isForeign =
        roleType === PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT;
      const isSupplier =
        roleType === PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER;
      const effectiveAddress = isForeign
        ? values.addressEn?.trim()
        : values.addressDetail?.trim();

      const profile: API.PartnerProfile = {
        nameEn: isForeign
          ? values.nameEn?.trim() || values.legalName.trim()
          : values.nameEn?.trim(),
        addressEn: values.addressEn?.trim(),
        provinceCode: isForeign ? '' : provinceCode,
        cityCode: isForeign ? '' : cityCode,
        districtCode: isForeign ? '' : districtCode,
        addressDetail: effectiveAddress,
        nature: values.nature || roleLabel,
        developmentMethod: values.developmentMethod,
        customerTypes:
          isForeign || isSupplier
            ? []
            : values.customerType !== undefined && values.customerType !== null
              ? [Number(values.customerType)]
              : values.customerTypes || [1],
        businessTypes: values.businessTypes || [1],
        remark: values.remark?.trim(),
      };

      const assignments: API.PartnerAssignmentInput[] = [];
      const seenMembers = new Set<string>();

      const addAssignment = (role: number, userField: string) => {
        const userId = values[userField];
        if (userId && !seenMembers.has(userId)) {
          seenMembers.add(userId);
          assignments.push({ role, userId });
        }
      };

      // 注意：Creator (role: 1) 由服务端从会话自动记录，API 显式传入会触发 ErrPartnerInvalidArgument
      addAssignment(2, 'assignOperatorUser');
      addAssignment(3, 'assignSalesUser');
      addAssignment(4, 'assignServiceUser');
      addAssignment(5, 'assignFinanceUser');
      addAssignment(6, 'assignCommercialUser');
      addAssignment(7, 'assignContactUser');
      addAssignment(7, 'assignContact2User'); // 内部联系人2同为 role: 7
      addAssignment(8, 'assignDocUser');

      const contactInputs: API.PartnerContactInput[] = contacts.map((c) => ({
        name: c.name,
        phone: c.phone,
        email: c.email,
        note: c.note,
        isPrimary: c.isPrimary,
      }));

      const aliasInputs: API.PartnerAliasInput[] = aliases.map(
        (aliasName, idx) => ({
          aliasName,
          sortOrder: idx,
        }),
      );

      // Settlement Rule Payload
      const creditLimitMinor =
        values.creditLimit !== undefined && values.creditLimit !== null
          ? String(Math.round(Number(values.creditLimit) * 100))
          : '0';
      const settlementMethod = Number(values.settlementMethod || 1);
      // 票结/预付不适用结算日、账期与结算基准，后端要求这三项不下发
      const isTermFree =
        settlementMethod ===
          PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BY_TICKET ||
        settlementMethod ===
          PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_PREPAID;

      const settlementRuleInput: API.PartnerSettlementRuleInput = {
        statementMode: Number(values.statementMode || 1),
        settlementMethod,
        ...(isTermFree
          ? {}
          : {
              settlementDay: Number(values.settlementDay || 25),
              settlementBase: Number(values.settlementBase || 1),
              settlementCycleDays: Number(values.creditDays || 30),
            }),
        settlementCurrency: values.settlementCurrency || 'CNY',
        creditLimitMinor,
        creditCurrency: values.settlementCurrency || 'CNY',
        paymentTermsDays: values.paymentTermsDays,
        isActive: true,
      };

      const selectedRoleTypes: number[] = values.roleTypes || [roleType];
      const roleInputs: API.PartnerRoleInput[] = selectedRoleTypes.map(
        (type) => ({
          type,
          enabled: true,
          settlementRule:
            canSaveSettlement && type === roleType
              ? settlementRuleInput
              : undefined,
        }),
      );

      if (partnerId) {
        await partnerServiceUpdatePartner(
          { id: partnerId },
          {
            id: partnerId,
            code: values.code?.trim() ?? '',
            legalName: values.legalName.trim(),
            unifiedSocialCreditCode: isForeign
              ? undefined
              : values.unifiedSocialCreditCode?.trim(),
            registeredAddress: effectiveAddress,
            enabled: values.enabled ?? true,
            isCasual: isForeign ? false : Boolean(values.isCasual),
            roles: roleInputs,
            profile,
            assignments,
            contacts: contactInputs,
            aliases: aliasInputs,
          },
        );
        message.success(`${roleLabel}档案已成功更新`);
      } else {
        const createRes = await partnerServiceCreatePartner({
          code: values.code?.trim() || '',
          legalName: values.legalName.trim(),
          unifiedSocialCreditCode: isForeign
            ? undefined
            : values.unifiedSocialCreditCode?.trim(),
          registeredAddress: effectiveAddress,
          isCasual: isForeign ? false : Boolean(values.isCasual),
          roles: roleInputs,
          profile,
          assignments,
          contacts: contactInputs,
          aliases: aliasInputs,
        });
        message.success(`${roleLabel}档案已成功创建`);
        if (createRes.data?.id) {
          history.push(`${listUrl}/${createRes.data.id}`);
          return;
        }
      }

      history.push(listUrl);
    } catch (err) {
      if (isFormValidationError(err)) {
        const res = scrollToFirstFormError({
          errorFields: err.errorFields,
          onExpandSection: (sectionKey) => {
            setActiveCollapseKeys((prev) =>
              Array.from(new Set([...prev, sectionKey])),
            );
          },
          notify: (msg) => message.warning(msg),
        });
        setSectionErrors(res.errorsBySection);
      } else {
        message.error(getErrorMessage(err, '保存失败，请重试'));
      }
    } finally {
      setSaving(false);
    }
  };

  const displayTitle = partner?.legalName
    ? partner.legalName
    : isCreate
      ? `新建${roleLabel}`
      : `${roleLabel}详情`;

  const toggleSection = (key: string, collapsed: boolean) => {
    setActiveCollapseKeys((prev) =>
      collapsed
        ? prev.filter((k) => k !== key)
        : prev.includes(key)
          ? prev
          : [...prev, key],
    );
  };

  const creatorAssignment = partner?.assignments?.find((a) => a.role === 1);
  const creatorUser = users.find((u) => u.id === creatorAssignment?.userId);
  const creatorName =
    creatorUser?.displayName || creatorUser?.username || '系统';
  const creatorMeta = partner?.createdAt
    ? `创建人: ${creatorName} | 创建时间: ${formatDate(partner.createdAt)}`
    : isCreate
      ? '创建人: 当前用户 (自动关联)'
      : undefined;

  if (isInvalidId) {
    return (
      <PageContainer
        title={false}
        breadcrumbRender={false}
        header={{
          title: false,
          breadcrumb: undefined,
          style: { padding: 0 },
        }}
        style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
      >
        <div style={{ padding: 48, textAlign: 'center' }}>
          <Spin description="无效的档案标识，正在返回列表..." />
        </div>
      </PageContainer>
    );
  }

  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{
        title: false,
        breadcrumb: undefined,
        style: { padding: 0 },
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      {/* 1. Page Header Shell */}
      <PageHeaderShell
        title={displayTitle}
        onBack={() => history.push(listUrl)}
        breadcrumbs={[{ label: `${roleLabel}管理`, href: listUrl }]}
        tags={
          partner ? (
            <Space size={6}>
              {partner.code ? (
                <Tag variant="filled" style={{ fontFamily: 'monospace' }}>
                  {partner.code}
                </Tag>
              ) : null}
              {partner.isCasual && <Tag color="warning">散客</Tag>}
              <Tag color={partner.enabled ? 'success' : 'default'}>
                {partner.enabled ? '正常启用' : '已停用'}
              </Tag>
            </Space>
          ) : undefined
        }
        extra={
          <Space size={8}>
            <Button onClick={() => history.push(listUrl)} disabled={saving}>
              取消
            </Button>
            {canSave && (
              <Button
                type="primary"
                onClick={handleSubmit}
                loading={saving}
                icon={<CheckCircleOutlined />}
              >
                {saving ? '保存中...' : `保存${roleLabel}档案`}
              </Button>
            )}
          </Space>
        }
      />

      {/* 2. Main Container */}
      <Spin spinning={loading}>
        <ProForm
          formRef={formRef}
          submitter={false}
          disabled={!canOperate}
          layout="horizontal"
          style={{
            paddingRight: navCollapsed ? 0 : 164,
            transition: 'padding-right 0.25s ease',
          }}
        >
          {/* Section 1: 基础信息 */}
          <BasicInfoSection
            collapsed={!activeCollapseKeys.includes('basic')}
            onCollapseChange={(collapsed) => toggleSection('basic', collapsed)}
            roleLabel={roleLabel}
            roleType={roleType}
            userSelectOptions={userSelectOptions}
            aliases={aliases}
            onAliasesChange={setAliases}
            newAliasInput={newAliasInput}
            setNewAliasInput={setNewAliasInput}
            onAddAlias={handleAddAlias}
            onRemoveAlias={handleRemoveAlias}
            onTianyanchaVerify={handleTianyanchaVerify}
            creatorMeta={creatorMeta}
          />

          {/* Section 2: 财务结算规则 */}
          {(isCreate || canReadSettlementRules) && (
            <SettlementSection
              collapsed={!activeCollapseKeys.includes('settlement')}
              onCollapseChange={(collapsed) =>
                toggleSection('settlement', collapsed)
              }
              currencyOptions={currencyOptions}
              interestRule={interestRule}
              onOpenInterestModal={() => setInterestModalOpen(true)}
              roleLabel={roleLabel}
            />
          )}

          {/* Section 3: 账户信息（依赖已保存档案与读权限，新建模式不展示） */}
          {partnerId && canReadAccounts && (
            <AccountsSection
              key="accounts"
              collapsed={!activeCollapseKeys.includes('accounts')}
              onCollapseChange={(collapsed) =>
                toggleSection('accounts', collapsed)
              }
              partner={partner}
              canRead={canReadAccounts}
              canCreate={
                canOperate &&
                (access.canCreatePartnerAccounts || access.canManagePartners)
              }
              canUpdate={
                canOperate &&
                (access.canUpdatePartnerAccounts || access.canManagePartners)
              }
            />
          )}

          {/* Section 4: 联系方式 */}
          <ContactsSection
            key="contacts"
            collapsed={!activeCollapseKeys.includes('contacts')}
            onCollapseChange={(collapsed) =>
              toggleSection('contacts', collapsed)
            }
            contacts={contacts}
            onChange={setContacts}
          />

          {/* Section 5: 常用信息 (Shipping Presets，依赖已保存档案与读权限，新建模式不展示) */}
          {partnerId && canReadShippingPresets && (
            <PresetsSection
              key="presets"
              collapsed={!activeCollapseKeys.includes('presets')}
              onCollapseChange={(collapsed) =>
                toggleSection('presets', collapsed)
              }
              partnerId={partnerId}
              roleLabel={roleLabel}
              canCreate={
                canOperate &&
                (access.canCreatePartnerShippingPresets ||
                  access.canManagePartners)
              }
              canUpdate={
                canOperate &&
                (access.canUpdatePartnerShippingPresets ||
                  access.canManagePartners)
              }
            />
          )}

          {/* Section 6: 合同管理（依赖已保存档案与读权限，新建模式不展示） */}
          {partnerId && canReadContracts && (
            <ContractsSection
              key="contracts"
              collapsed={!activeCollapseKeys.includes('contracts')}
              onCollapseChange={(collapsed) =>
                toggleSection('contracts', collapsed)
              }
              partnerId={partnerId}
              roleLabel={roleLabel}
              canCreate={
                canOperate &&
                (access.canCreatePartnerContracts || access.canManagePartners)
              }
              canUpdate={
                canOperate &&
                (access.canUpdatePartnerContracts || access.canManagePartners)
              }
            />
          )}

          {/* Section 7: 备注 */}
          <RemarkSection
            key="remark"
            collapsed={!activeCollapseKeys.includes('remark')}
            onCollapseChange={(collapsed) => toggleSection('remark', collapsed)}
            roleLabel={roleLabel}
          />

          {/* Section 8: 操作记录（依赖已保存档案与读权限） */}
          {partnerId && canReadAudit && (
            <LogsSection
              key="logs"
              collapsed={!activeCollapseKeys.includes('logs')}
              onCollapseChange={(collapsed) => toggleSection('logs', collapsed)}
              partnerId={partnerId}
              roleLabel={roleLabel}
            />
          )}
        </ProForm>
      </Spin>

      {/* 3. Sticky Footer Action Bar */}
      <StickyFooterBar
        info={
          partner?.legalName ? (
            <Text type="secondary" style={{ fontSize: 13 }}>
              当前档案：
              <Text strong style={{ color: 'rgba(0, 0, 0, 0.88)' }}>
                {partner.legalName}
              </Text>
            </Text>
          ) : undefined
        }
      >
        <Button onClick={() => history.push(listUrl)} disabled={saving}>
          取消
        </Button>
        {canSave && (
          <Button
            type="primary"
            onClick={handleSubmit}
            loading={saving}
            icon={<CheckCircleOutlined />}
            style={{ minWidth: 120 }}
          >
            {saving ? '保存中...' : `保存${roleLabel}档案`}
          </Button>
        )}
      </StickyFooterBar>

      {/* Interest Rule Modal */}
      <InterestRuleModal
        open={interestModalOpen}
        value={interestRule}
        onOpenChange={setInterestModalOpen}
        onFinish={async (values) => {
          setInterestRule(values);
          message.success('利息规则已更新');
          setInterestModalOpen(false);
        }}
      />

      {/* 4. 楼层大纲与错误定位导航 */}
      {!loading && (
        <PartnerAnchorNav
          sectionErrors={sectionErrors}
          isCreate={isCreate}
          partnerId={partnerId}
          canReadSettlementRules={canReadSettlementRules}
          canReadAccounts={canReadAccounts}
          canReadShippingPresets={canReadShippingPresets}
          canReadContracts={canReadContracts}
          canReadAudit={canReadAudit}
          roleLabel={roleLabel}
          onActiveCollapseKeysChange={setActiveCollapseKeys}
          onCollapsedChange={setNavCollapsed}
        />
      )}
    </PageContainer>
  );
}
