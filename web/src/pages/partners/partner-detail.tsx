import { CheckCircleOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { ProForm, ProFormTextArea } from '@ant-design/pro-components';
import { history, useLocation, useParams } from '@umijs/max';
import { App, Button, Col, Space, Spin, Tag, Typography } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  adminServiceListOrganizations,
  adminServiceListUsers,
} from '@/services/roncin/adminService';
import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import {
  partnerServiceCreatePartner,
  partnerServiceGetPartner,
  partnerServiceListPartnerAssignmentOptions,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import { unwrapList } from '@/utils/api';
import { PageHeaderShell, SectionCard, StickyFooterBar } from '@/components/ui';
import AccountCardList from './components/AccountCardList';
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

const { Text } = Typography;

export default function PartnerDetailPage() {
  const { message } = App.useApp();
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

  // Detect roleType from pathname
  const { roleType, roleLabel, listUrl } = useMemo(() => {
    const path = location.pathname;
    if (path.includes('/suppliers')) {
      return {
        roleType: 2,
        roleLabel: '供应商',
        listUrl: '/partners/suppliers',
      };
    }
    if (path.includes('/foreign-agents')) {
      return {
        roleType: 3,
        roleLabel: '国外代理',
        listUrl: '/partners/foreign-agents',
      };
    }
    return { roleType: 1, roleLabel: '客户', listUrl: '/partners/customers' };
  }, [location.pathname]);

  const partnerId = params.id && params.id !== 'create' ? params.id : undefined;
  const isCreate = !partnerId;

  // Load auxiliary options
  useEffect(() => {
    const fetchOptions = async () => {
      const [usersRes, orgsRes, curRes, assignRes] = await Promise.allSettled([
        adminServiceListUsers(
          { page: 1, pageSize: 200 },
          { skipErrorHandler: true },
        ),
        adminServiceListOrganizations({ skipErrorHandler: true }),
        masterDataServiceListCurrencies({ skipErrorHandler: true }),
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
      if (curRes.status === 'fulfilled' && curRes.value.data) {
        setCurrencyOptions(
          curRes.value.data
            .filter((currency) => currency.code)
            .map((currency) => ({
              label: `${currency.code} (${currency.name || currency.code})`,
              value: currency.code ?? '',
            })),
        );
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
            const creditAmount = currentRule?.creditLimitMinor
              ? Number(currentRule.creditLimitMinor) / 100
              : undefined;

            formRef.current?.setFieldsValue({
              code: p.code,
              legalName: p.legalName,
              unifiedSocialCreditCode: p.unifiedSocialCreditCode,
              enabled: p.enabled ?? true,
              regionCodes: regionCodes.length > 0 ? regionCodes : undefined,
              addressDetail: profile.addressDetail || p.registeredAddress,
              nameEn: profile.nameEn,
              addressEn: profile.addressEn,
              nature: profile.nature || roleLabel,
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
        nature: roleLabel,
        customerTypes: [1],
        developmentMethod: '自主开发',
        businessTypes: [1],
        statementMode: 1,
        settlementMethod: 1,
        settlementBase: 1,
        settlementDay: 25,
        settlementCurrency: 'CNY',
        creditDays: 30,
      });
    }
  }, [partnerId, roleType, roleLabel, message]);

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

      const profile: API.PartnerProfile = {
        nameEn: values.nameEn?.trim(),
        addressEn: values.addressEn?.trim(),
        provinceCode,
        cityCode,
        districtCode,
        addressDetail: values.addressDetail?.trim(),
        nature: values.nature || roleLabel,
        developmentMethod: values.developmentMethod,
        customerTypes: values.customerTypes || [1],
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

      const settlementRuleInput: API.PartnerSettlementRuleInput = {
        statementMode: Number(values.statementMode || 1),
        settlementMethod: Number(values.settlementMethod || 1),
        settlementBase: Number(values.settlementBase || 1),
        settlementDay: Number(values.settlementDay || 25),
        settlementCurrency: values.settlementCurrency || 'CNY',
        settlementCycleDays: Number(values.creditDays || 30),
        creditLimitMinor,
        creditCurrency: values.settlementCurrency || 'CNY',
        isActive: true,
      };

      const roleInput: API.PartnerRoleInput = {
        type: roleType,
        enabled: values.enabled ?? true,
        settlementRule: settlementRuleInput,
      };

      if (partnerId) {
        await partnerServiceUpdatePartner(
          { id: partnerId },
          {
            id: partnerId,
            legalName: values.legalName.trim(),
            unifiedSocialCreditCode: values.unifiedSocialCreditCode?.trim(),
            registeredAddress: values.addressDetail?.trim(),
            enabled: values.enabled ?? true,
            roles: [roleInput],
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
          unifiedSocialCreditCode: values.unifiedSocialCreditCode?.trim(),
          registeredAddress: values.addressDetail?.trim(),
          roles: [roleInput],
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
        message.error('请检查标红的必填项');
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
    <div style={{ minHeight: '100%', paddingBottom: 24 }}>
      {/* 1. Page Header Shell */}
      <PageHeaderShell
        title={displayTitle}
        onBack={() => history.push(listUrl)}
        breadcrumbs={[
          { label: `${roleLabel}管理`, onClick: () => history.push(listUrl) },
        ]}
        tags={
          partner?.code ? (
            <Tag variant="filled" style={{ fontFamily: 'monospace' }}>
              {partner.code}
            </Tag>
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
        <div style={{ maxWidth: 1440, margin: '0 auto' }}>
          <ProForm
            formRef={formRef}
            submitter={false}
            layout="horizontal"
            grid
            rowProps={{ gutter: [16, 12] }}
          >
            <Col span={24}>
              {/* Section 1: 基础信息 */}
              <BasicInfoSection
                collapsed={!activeCollapseKeys.includes('basic')}
                onCollapseChange={(collapsed) =>
                  toggleSection('basic', collapsed)
                }
                partnerId={partnerId}
                roleLabel={roleLabel}
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
              />

              {/* Section 3: 账户信息 */}
              <SectionCard
                key="accounts"
                title="账户信息"
                collapsible
                collapsed={!activeCollapseKeys.includes('accounts')}
                onCollapseChange={(collapsed) =>
                  toggleSection('accounts', collapsed)
                }
              >
                <AccountCardList
                  partnerId={partnerId}
                  currencyOptions={currencyOptions}
                />
              </SectionCard>

              {/* Section 4: 联系方式 */}
              <SectionCard
                key="contacts"
                title="联系方式"
                collapsible
                collapsed={!activeCollapseKeys.includes('contacts')}
                onCollapseChange={(collapsed) =>
                  toggleSection('contacts', collapsed)
                }
              >
                <ContactCardList contacts={contacts} onChange={setContacts} />
              </SectionCard>

              {/* Section 5: 常用信息 (Shipping Presets) */}
              <SectionCard
                key="presets"
                title="常用信息"
                collapsible
                collapsed={!activeCollapseKeys.includes('presets')}
                onCollapseChange={(collapsed) =>
                  toggleSection('presets', collapsed)
                }
              >
                <ShippingPresetSection partnerId={partnerId} />
              </SectionCard>

              {/* Section 6: 合同管理 */}
              <SectionCard
                key="contracts"
                title="合同管理"
                collapsible
                collapsed={!activeCollapseKeys.includes('contracts')}
                onCollapseChange={(collapsed) =>
                  toggleSection('contracts', collapsed)
                }
              >
                <ContractCardList partnerId={partnerId} />
              </SectionCard>

              {/* Section 7: 客户备注 */}
              <SectionCard
                key="remark"
                title="客户备注"
                collapsible
                collapsed={!activeCollapseKeys.includes('remark')}
                onCollapseChange={(collapsed) =>
                  toggleSection('remark', collapsed)
                }
              >
                <ProFormTextArea
                  name="remark"
                  placeholder="可以添加客户信息录入时的备注信息"
                  fieldProps={{ rows: 3 }}
                />
              </SectionCard>

              {/* Section 8: 操作记录 */}
              {partnerId && (
                <SectionCard
                  key="logs"
                  title="操作记录"
                  collapsible
                  collapsed={!activeCollapseKeys.includes('logs')}
                  onCollapseChange={(collapsed) =>
                    toggleSection('logs', collapsed)
                  }
                >
                  <AuditLogSection partnerId={partnerId} />
                </SectionCard>
              )}
            </Col>
          </ProForm>
        </div>
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
    </div>
  );
}
