import { CheckCircleOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  PageContainer,
  ProForm,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  history,
  useAccess,
  useLocation,
  useParams,
  useSearchParams,
} from '@umijs/max';
import { App, Button, Space, Spin, Tag, Typography } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  FormAnchorNav,
  focusFieldInput,
  PageHeaderShell,
  pulseHighlightElement,
  SectionCard,
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
import {
  adminServiceListOrganizations,
  adminServiceListUsers,
} from '@/services/roncin/adminService';
import {
  partnerServiceCreatePartner,
  partnerServiceGetPartner,
  partnerServiceListPartnerAssignmentOptions,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import { unwrapList } from '@/utils/api';
import { getCurrencyOptions } from '@/utils/options';
import AuditLogSection from './components/AuditLogSection';
import BasicInfoSection from './components/BasicInfoSection';
import ContactCardList, {
  type ContactItem,
} from './components/ContactCardList';
import ContractCardList from './components/ContractCardList';
import InterestRuleModal, {
  type InterestRuleValues,
} from './components/InterestRuleModal';
import SettlementSection from './components/SettlementSection';
import ShippingPresetSection from './components/ShippingPresetSection';
import AccountsPanel from './components/secondary/AccountsPanel';

const { Text } = Typography;

export default function PartnerDetailPage() {
  const { message } = App.useApp();
  const access = useAccess();
  const params = useParams<{ id?: string }>();
  const location = useLocation();
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [partner, setPartner] = useState<API.Partner | undefined>(undefined);

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

  // Options state
  const [users, setUsers] = useState<API.AdminUser[]>([]);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>(
    [],
  );
  const [assignmentOptions, setAssignmentOptions] = useState<
    API.PartnerAssignmentOption[]
  >([]);
  const [currencyOptions, setCurrencyOptions] = useState<
    { label: string; value: string }[]
  >([]);

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

  // Detect roleType from pathname
  const { roleType, roleLabel, listUrl } = useMemo(() => {
    const path = location.pathname;
    if (path.includes('/suppliers')) {
      return {
        roleType: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
        roleLabel: '供应商',
        listUrl: '/partners/suppliers',
      };
    }
    if (path.includes('/foreign-agents')) {
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

  const partnerId = params.id && params.id !== 'create' ? params.id : undefined;
  const isCreate = !partnerId;

  // 创建模式读取 legalName 查询参数预填（订单表单快捷新增「添加公司详情」携带）；
  // 编辑模式忽略该参数，不覆盖已加载的公司抬头。
  const [searchParams] = useSearchParams();
  const prefillLegalName = isCreate
    ? (searchParams.get('legalName') ?? '').trim()
    : '';

  // Load auxiliary options
  useEffect(() => {
    const fetchOptions = async () => {
      const [usersRes, orgsRes, curRes, assignRes] = await Promise.allSettled([
        adminServiceListUsers(
          { page: 1, pageSize: 200 },
          { skipErrorHandler: true },
        ),
        adminServiceListOrganizations({ skipErrorHandler: true }),
        getCurrencyOptions(),
        partnerServiceListPartnerAssignmentOptions({ skipErrorHandler: true }),
      ]);

      if (usersRes.status === 'fulfilled' && usersRes.value.data) {
        setUsers(usersRes.value.data);
      }
      if (orgsRes.status === 'fulfilled' && orgsRes.value.data) {
        setOrganizations(orgsRes.value.data);
      }
      if (assignRes.status === 'fulfilled' && assignRes.value.data) {
        setAssignmentOptions(assignRes.value.data);
      }
      if (curRes.status === 'fulfilled') {
        setCurrencyOptions(curRes.value);
      }

      const failedLabels = [
        usersRes.status === 'rejected' ? '用户' : '',
        orgsRes.status === 'rejected' ? '组织' : '',
        curRes.status === 'rejected' ? '币种' : '',
        assignRes.status === 'rejected' ? '人员归属' : '',
      ].filter(Boolean);
      if (failedLabels.length > 0) {
        message.warning(`${failedLabels.join('、')}选项加载失败`);
      }
    };

    fetchOptions();
  }, [message]);

  // Map user ID to organization ID for auto-fill
  const userOrgMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const opt of assignmentOptions) {
      if (opt.userId && opt.organizationId && !map.has(opt.userId)) {
        map.set(opt.userId, opt.organizationId);
      }
    }
    return map;
  }, [assignmentOptions]);

  // Load partner detail when editing
  useEffect(() => {
    if (partnerId) {
      setLoading(true);
      Promise.all([
        partnerServiceGetPartner({ id: partnerId }),
        partnerServiceListPartnerSettlementRules({ partnerId, roleType }),
      ])
        .then(([partnerRes, ruleRes]) => {
          const p = partnerRes.data;
          setPartner(p);
          const rules = unwrapList(ruleRes);
          const currentRule = rules[0];

          if (p) {
            const profile = p.profile || {};
            const assignments = p.assignments || [];

            const findAssignment = (role: number, index = 0) => {
              const item = assignments
                .filter((assignment) => assignment.role === role)
                .sort(
                  (left, right) =>
                    (left.sortOrder ?? 0) - (right.sortOrder ?? 0),
                )[index];
              return {
                userId: item?.userId,
                organizationId: item?.organizationId,
              };
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
            const loadedContacts: ContactItem[] = (p.contacts || []).map(
              (c) => ({
                id: c.id,
                name: c.name || '',
                phone: c.phone,
                email: c.email,
                note: c.note,
                isPrimary: c.isPrimary,
              }),
            );
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
                profile.addressEn ||
                (isForeign ? p.registeredAddress : undefined),
              nature: profile.nature || roleLabel,
              roleTypes:
                (p.roles ?? [])
                  .filter((r) => r.enabled)
                  .map((r) => r.type as number).length > 0
                  ? (p.roles ?? [])
                      .filter((r) => r.enabled)
                      .map((r) => r.type as number)
                  : [roleType],
              customerTypes: profile.customerTypes || [1],
              developmentMethod: profile.developmentMethod || '自主开发',
              businessTypes: profile.businessTypes || [1],
              remark: profile.remark,

              // 9 Assignment slots (User + Organization pairs)
              assignCreatorUser: findAssignment(1).userId,
              assignCreatorOrg: findAssignment(1).organizationId,
              assignOperatorUser: findAssignment(2).userId,
              assignOperatorOrg: findAssignment(2).organizationId,
              assignSalesUser: findAssignment(3).userId,
              assignSalesOrg: findAssignment(3).organizationId,
              assignServiceUser: findAssignment(4).userId,
              assignServiceOrg: findAssignment(4).organizationId,
              assignFinanceUser: findAssignment(5).userId,
              assignFinanceOrg: findAssignment(5).organizationId,
              assignCommercialUser: findAssignment(6).userId,
              assignCommercialOrg: findAssignment(6).organizationId,
              assignContactUser: findAssignment(7).userId,
              assignContactOrg: findAssignment(7).organizationId,
              assignContact2User: findAssignment(7, 1).userId,
              assignContact2Org: findAssignment(7, 1).organizationId,
              assignDocUser: findAssignment(8).userId,
              assignDocOrg: findAssignment(8).organizationId,

              // Settlement Info
              statementMode: currentRule?.statementMode ?? 1,
              settlementMethod: currentRule?.settlementMethod ?? 1,
              settlementBase: currentRule?.settlementBase ?? 1,
              settlementDay: currentRule?.settlementDay ?? 25,
              settlementCurrency: currentRule?.settlementCurrency ?? 'CNY',
              creditDays: currentRule?.settlementCycleDays ?? 30,
              creditLimit: creditAmount,
              paymentTermsDays: currentRule?.paymentTermsDays,
            });
          }
        })
        .catch(() => {
          message.error('加载档案详情失败');
        })
        .finally(() => {
          setLoading(false);
        });
    } else {
      setPartner(undefined);
      setContacts([]);
      setAliases([]);
      formRef.current?.resetFields();
      formRef.current?.setFieldsValue({
        enabled: true,
        isCasual: false,
        nature: roleLabel,
        roleTypes: [roleType],
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
        // 创建默认值跟随当前地址重建：携带 legalName 查询参数时预填公司抬头，
        // 参数移除或变化时按新地址重建，不残留上一条地址的预填。
        legalName: prefillLegalName,
      });
    }
  }, [partnerId, roleType, roleLabel, prefillLegalName, message]);

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

  const orgSelectOptions = useMemo(
    () =>
      organizations.map((o) => ({
        label: `${o.name} (${o.code})`,
        value: o.id ?? '',
      })),
    [organizations],
  );

  // Auto fill org when user is selected
  const handleUserChange = (
    userFieldName: string,
    orgFieldName: string,
    selectedUserId?: string,
  ) => {
    formRef.current?.setFieldValue(userFieldName, selectedUserId);
    if (selectedUserId) {
      const defaultOrg = userOrgMap.get(selectedUserId);
      if (defaultOrg && !formRef.current?.getFieldValue(orgFieldName)) {
        formRef.current?.setFieldValue(orgFieldName, defaultOrg);
      }
    }
  };

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
          isForeign || isSupplier ? [] : values.customerTypes || [1],
        businessTypes: values.businessTypes || [1],
        remark: values.remark?.trim(),
      };

      const assignments: API.PartnerAssignmentInput[] = [];
      const seenMembers = new Set<string>();

      const addAssignment = (
        role: number,
        userField: string,
        orgField: string,
      ) => {
        const userId = values[userField];
        const orgId = values[orgField];
        if (userId && orgId) {
          const key = `${userId}:${orgId}`;
          if (!seenMembers.has(key)) {
            seenMembers.add(key);
            assignments.push({
              role,
              userId,
              organizationId: orgId,
            });
          }
        }
      };

      // 注意：Creator (role: 1) 由服务端从会话自动记录，API 显式传入会触发 ErrPartnerInvalidArgument
      addAssignment(2, 'assignOperatorUser', 'assignOperatorOrg');
      addAssignment(3, 'assignSalesUser', 'assignSalesOrg');
      addAssignment(4, 'assignServiceUser', 'assignServiceOrg');
      addAssignment(5, 'assignFinanceUser', 'assignFinanceOrg');
      addAssignment(6, 'assignCommercialUser', 'assignCommercialOrg');
      addAssignment(7, 'assignContactUser', 'assignContactOrg');
      addAssignment(7, 'assignContact2User', 'assignContact2Org'); // 内部联系人2同为 role: 7
      addAssignment(8, 'assignDocUser', 'assignDocOrg');

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
          settlementRule: type === roleType ? settlementRuleInput : undefined,
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
    } catch (err: any) {
      if (err?.errorFields) {
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
        message.error(err?.message || '保存失败，请重试');
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
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={saving}
              icon={<CheckCircleOutlined />}
            >
              {saving ? '保存中...' : `保存${roleLabel}档案`}
            </Button>
          </Space>
        }
      />

      {/* 2. Main Container */}
      <Spin spinning={loading}>
        <ProForm
          formRef={formRef}
          submitter={false}
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
            orgSelectOptions={orgSelectOptions}
            aliases={aliases}
            newAliasInput={newAliasInput}
            setNewAliasInput={setNewAliasInput}
            onAddAlias={handleAddAlias}
            onRemoveAlias={handleRemoveAlias}
            onTianyanchaVerify={handleTianyanchaVerify}
            onUserChange={handleUserChange}
          />

          {/* Section 2: 财务结算规则 */}
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

          {/* Section 3: 账户信息（依赖已保存档案，新建模式不展示） */}
          {partnerId && (
            <SectionCard
              key="accounts"
              id="section-accounts"
              sectionKey="accounts"
              title="账户信息"
              collapsible
              collapsed={!activeCollapseKeys.includes('accounts')}
              onCollapseChange={(collapsed) =>
                toggleSection('accounts', collapsed)
              }
            >
              <AccountsPanel
                partner={partner}
                canRead={access.canReadPartnerAccounts}
                canCreate={access.canCreatePartnerAccounts}
                canUpdate={access.canUpdatePartnerAccounts}
              />
            </SectionCard>
          )}

          {/* Section 4: 联系方式 */}
          <SectionCard
            key="contacts"
            id="section-contacts"
            sectionKey="contacts"
            title="联系方式"
            collapsible
            collapsed={!activeCollapseKeys.includes('contacts')}
            onCollapseChange={(collapsed) =>
              toggleSection('contacts', collapsed)
            }
          >
            <ContactCardList contacts={contacts} onChange={setContacts} />
          </SectionCard>

          {/* Section 5: 常用信息 (Shipping Presets，依赖已保存档案，新建模式不展示) */}
          {partnerId && (
            <SectionCard
              key="presets"
              id="section-presets"
              sectionKey="presets"
              title="常用信息"
              collapsible
              collapsed={!activeCollapseKeys.includes('presets')}
              onCollapseChange={(collapsed) =>
                toggleSection('presets', collapsed)
              }
            >
              <ShippingPresetSection
                partnerId={partnerId}
                roleLabel={roleLabel}
              />
            </SectionCard>
          )}

          {/* Section 6: 合同管理（依赖已保存档案，新建模式不展示） */}
          {partnerId && (
            <SectionCard
              key="contracts"
              id="section-contracts"
              sectionKey="contracts"
              title="合同管理"
              collapsible
              collapsed={!activeCollapseKeys.includes('contracts')}
              onCollapseChange={(collapsed) =>
                toggleSection('contracts', collapsed)
              }
            >
              <ContractCardList partnerId={partnerId} roleLabel={roleLabel} />
            </SectionCard>
          )}

          {/* Section 7: 备注 */}
          <SectionCard
            key="remark"
            id="section-remark"
            sectionKey="remark"
            title={`${roleLabel}备注`}
            collapsible
            collapsed={!activeCollapseKeys.includes('remark')}
            onCollapseChange={(collapsed) => toggleSection('remark', collapsed)}
          >
            <ProFormTextArea
              name="remark"
              placeholder={`可以添加${roleLabel}信息录入时的备注信息`}
              fieldProps={{ rows: 3 }}
            />
          </SectionCard>

          {/* Section 8: 操作记录 */}
          {partnerId && (
            <SectionCard
              key="logs"
              id="section-logs"
              sectionKey="logs"
              title="操作记录"
              collapsible
              collapsed={!activeCollapseKeys.includes('logs')}
              onCollapseChange={(collapsed) => toggleSection('logs', collapsed)}
            >
              <AuditLogSection partnerId={partnerId} roleLabel={roleLabel} />
            </SectionCard>
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
        <Button
          type="primary"
          onClick={handleSubmit}
          loading={saving}
          icon={<CheckCircleOutlined />}
          style={{ minWidth: 120 }}
        >
          {saving ? '保存中...' : `保存${roleLabel}档案`}
        </Button>
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
        <FormAnchorNav
          sectionErrors={sectionErrors}
          defaultCollapsed={true}
          onCollapsedChange={setNavCollapsed}
          items={[
            { key: 'basic', title: '基础信息' },
            { key: 'settlement', title: '财务结算' },
            ...(partnerId ? [{ key: 'accounts', title: '账户信息' }] : []),
            { key: 'contacts', title: '联系方式' },
            ...(partnerId ? [{ key: 'presets', title: '常用信息' }] : []),
            ...(partnerId ? [{ key: 'contracts', title: '合同管理' }] : []),
            { key: 'remark', title: `${roleLabel}备注` },
            ...(partnerId ? [{ key: 'logs', title: '操作记录' }] : []),
          ]}
          onSelect={(key) => {
            setActiveCollapseKeys((prev) =>
              Array.from(new Set([...prev, key])),
            );
          }}
          onErrorClick={(sectionKey) => {
            setActiveCollapseKeys((prev) =>
              Array.from(new Set([...prev, sectionKey])),
            );
            window.setTimeout(() => {
              const sectionEl = document.getElementById(
                `section-${sectionKey}`,
              );
              if (sectionEl) {
                const errorEl = sectionEl.querySelector<HTMLElement>(
                  '.ant-form-item-has-error',
                );
                if (errorEl) {
                  errorEl.scrollIntoView({
                    behavior: 'smooth',
                    block: 'center',
                  });
                  pulseHighlightElement(errorEl);
                  focusFieldInput(errorEl);
                } else {
                  sectionEl.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start',
                  });
                }
              }
            }, 100);
          }}
        />
      )}
    </PageContainer>
  );
}
