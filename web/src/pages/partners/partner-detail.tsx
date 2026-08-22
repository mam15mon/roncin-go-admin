import {
  ArrowLeftOutlined,
  CaretDownOutlined,
  CaretRightOutlined,
  CheckCircleOutlined,
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ProForm,
  ProFormCheckbox,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { history, useLocation, useParams } from '@umijs/max';
import {
  App,
  Button,
  Cascader,
  Col,
  Collapse,
  Divider,
  Form,
  Input,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
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
import { pcaCascaderOptions } from '@/utils/chinaDivision';
import AccountCardList from './components/AccountCardList';
import AuditLogSection from './components/AuditLogSection';
import ContactCardList, { type ContactItem } from './components/ContactCardList';
import ContractCardList from './components/ContractCardList';
import InterestRuleModal, { type InterestRuleValues } from './components/InterestRuleModal';
import ShippingPresetSection from './components/ShippingPresetSection';

const { Text } = Typography;

const BUSINESS_TYPE_OPTIONS = [
  { label: 'SE（海运出口）', value: 1 },
  { label: 'SI（海运进口）', value: 2 },
  { label: 'AE（空运出口）', value: 3 },
  { label: 'AI（空运进口）', value: 4 },
  { label: 'LAND（陆运业务）', value: 5 },
  { label: 'RAIL（铁路运输）', value: 6 },
];

const CUSTOMER_TYPE_OPTIONS = [
  { label: '直客', value: 1 },
  { label: '同行', value: 2 },
];

const DEVELOPMENT_METHOD_OPTIONS = [
  { label: '自主开发', value: '自主开发' },
  { label: '网络推广', value: '网络推广' },
  { label: '老客转介', value: '老客转介' },
  { label: '商务分配', value: '商务分配' },
  { label: '展会获取', value: '展会获取' },
  { label: '公开招标', value: '公开招标' },
  { label: '其它方式', value: '其它方式' },
];

const STATEMENT_MODE_OPTIONS = [
  { label: '单票', value: 1 },
  { label: '汇总', value: 2 },
];

const SETTLEMENT_METHOD_OPTIONS = [
  { label: '票结', value: 1 },
  { label: '月结', value: 2 },
  { label: '周结', value: 3 },
  { label: '半月结', value: 4 },
  { label: '双月结', value: 5 },
  { label: '季结', value: 6 },
  { label: '45天', value: 7 },
  { label: '预付', value: 8 },
];

const SETTLEMENT_BASE_OPTIONS = [
  { label: '开票后', value: 1 },
  { label: '出运后', value: 2 },
  { label: '到港后', value: 3 },
];

const SETTLEMENT_DAY_OPTIONS = Array.from({ length: 31 }, (_, i) => ({
  label: `${i + 1}日`,
  value: i + 1,
}));

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
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>([]);
  const [assignmentOptions, setAssignmentOptions] = useState<API.PartnerAssignmentOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<{ label: string; value: string }[]>([
    { label: 'CNY (人民币)', value: 'CNY' },
    { label: 'USD (美元)', value: 'USD' },
    { label: 'EUR (欧元)', value: 'EUR' },
    { label: 'HKD (港币)', value: 'HKD' },
  ]);

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
      return { roleType: 2, roleLabel: '供应商', listUrl: '/partners/suppliers' };
    }
    if (path.includes('/foreign-agents')) {
      return { roleType: 3, roleLabel: '国外代理', listUrl: '/partners/foreign-agents' };
    }
    return { roleType: 1, roleLabel: '客户', listUrl: '/partners/customers' };
  }, [location.pathname]);

  const partnerId = params.id && params.id !== 'create' ? params.id : undefined;
  const isCreate = !partnerId;

  // Load auxiliary options
  useEffect(() => {
    const fetchOptions = async () => {
      try {
        const [usersRes, orgsRes, curRes, assignRes] = await Promise.allSettled([
          adminServiceListUsers({ page: 1, pageSize: 200 }),
          adminServiceListOrganizations(),
          masterDataServiceListCurrencies(),
          partnerServiceListPartnerAssignmentOptions(),
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
          const list = curRes.value.data
            .filter((c) => c.code)
            .map((c) => ({
              label: `${c.code} (${c.name || c.code})`,
              value: c.code ?? '',
            }));
          if (list.length > 0) {
            setCurrencyOptions(list);
          }
        }
      } catch {
        // Keep fallbacks
      }
    };

    fetchOptions();
  }, []);

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
          const rules = ruleRes.data ?? [];
          const currentRule = rules[0];

          if (p) {
            const profile = p.profile || {};
            const assignments = p.assignments || [];

            const findAssignment = (role: number) => {
              const item = assignments.find((a) => a.role === role);
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
            const loadedAliases = (p.aliases || []).map((a) => a.aliasName || '').filter(Boolean);
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

              // 8 Assignment slots (User + Organization pairs)
              assignCreatorUser: findAssignment(1).userId,
              assignCreatorOrg: findAssignment(1).organizationId,
              assignOperatorUser: findAssignment(2).userId,
              assignOperatorOrg: findAssignment(2).organizationId,
              assignSalesUser: findAssignment(3).userId,
              assignSalesOrg: findAssignment(3).organizationId,
              assignServiceUser: findAssignment(4).userId,
              assignServiceOrg: findAssignment(4).organizationId,
              assignDocUser: findAssignment(5).userId,
              assignDocOrg: findAssignment(5).organizationId,
              assignCommercialUser: findAssignment(6).userId,
              assignCommercialOrg: findAssignment(6).organizationId,
              assignContactUser: findAssignment(7).userId,
              assignContactOrg: findAssignment(7).organizationId,
              assignContact2User: findAssignment(8).userId,
              assignContact2Org: findAssignment(8).organizationId,

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
  const userSelectOptions = useMemo(
    () =>
      users.map((u) => ({
        label: `${u.displayName || u.username} (${u.username})`,
        value: u.id ?? '',
      })),
    [users],
  );

  const orgSelectOptions = useMemo(
    () =>
      organizations.map((o) => ({
        label: `${o.name} (${o.code})`,
        value: o.id ?? '',
      })),
    [organizations],
  );

  // Auto fill org when user is selected
  const handleUserChange = (userFieldName: string, orgFieldName: string, selectedUserId: string) => {
    formRef.current?.setFieldValue(userFieldName, selectedUserId);
    const defaultOrg = userOrgMap.get(selectedUserId);
    if (defaultOrg && !formRef.current?.getFieldValue(orgFieldName)) {
      formRef.current?.setFieldValue(orgFieldName, defaultOrg);
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
      const addAssignment = (role: number, userField: string, orgField: string) => {
        const userId = values[userField];
        const orgId = values[orgField];
        if (userId && orgId) {
          assignments.push({
            role,
            userId,
            organizationId: orgId,
          });
        }
      };

      addAssignment(1, 'assignCreatorUser', 'assignCreatorOrg');
      addAssignment(2, 'assignOperatorUser', 'assignOperatorOrg');
      addAssignment(3, 'assignSalesUser', 'assignSalesOrg');
      addAssignment(4, 'assignServiceUser', 'assignServiceOrg');
      addAssignment(5, 'assignDocUser', 'assignDocOrg');
      addAssignment(6, 'assignCommercialUser', 'assignCommercialOrg');
      addAssignment(7, 'assignContactUser', 'assignContactOrg');
      addAssignment(8, 'assignContact2User', 'assignContact2Org');

      const contactInputs: API.PartnerContactInput[] = contacts.map((c) => ({
        name: c.name,
        phone: c.phone,
        email: c.email,
        note: c.note,
        isPrimary: c.isPrimary,
      }));

      const aliasInputs: API.PartnerAliasInput[] = aliases.map((aliasName, idx) => ({
        aliasName,
        sortOrder: idx,
      }));

      // Settlement Rule Payload
      const creditLimitMinor = values.creditLimit !== undefined && values.creditLimit !== null
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

  // Render collapsible header with blue vertical bar
  const renderSectionHeader = (title: string, subtitle?: string) => (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div
        style={{
          width: 3,
          height: 14,
          backgroundColor: '#1677ff',
          borderRadius: 2,
        }}
      />
      <span style={{ fontWeight: 600, fontSize: 14, color: 'rgba(0, 0, 0, 0.88)' }}>
        {title}
      </span>
      {subtitle && (
        <Text type="secondary" style={{ fontSize: 12, marginLeft: 6 }}>
          {subtitle}
        </Text>
      )}
    </div>
  );

  return (
    <div style={{ minHeight: '100%', paddingBottom: 60, backgroundColor: '#f5f7fa' }}>
      {/* Sticky Top Header Navigation & Action Bar */}
      <div
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 99,
          backgroundColor: '#ffffff',
          borderBottom: '1px solid #f0f0f0',
          padding: '10px 24px',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.04)',
          marginBottom: 16,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: 12,
        }}
      >
        {/* Left: Breadcrumb */}
        <Space size={8} align="center">
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => history.push(listUrl)}
            style={{ padding: '4px 8px' }}
          >
            返回列表
          </Button>
          <Divider type="vertical" style={{ margin: '0 4px' }} />
          <Button
            type="link"
            style={{ padding: 0, color: 'rgba(0, 0, 0, 0.45)', height: 'auto' }}
            onClick={() => history.push(listUrl)}
          >
            {roleLabel}管理
          </Button>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>/</span>
          <Text strong style={{ fontSize: 14, color: 'rgba(0, 0, 0, 0.88)' }}>
            {displayTitle}
          </Text>
          {partner?.code && (
            <Tag bordered={false} style={{ fontFamily: 'monospace', marginLeft: 4 }}>
              {partner.code}
            </Tag>
          )}
        </Space>

        {/* Right: Actions */}
        <Space size={12}>
          <Button onClick={() => history.push(listUrl)} disabled={saving}>
            取消
          </Button>
          <Button
            type="primary"
            onClick={handleSubmit}
            loading={saving}
            icon={<CheckCircleOutlined />}
            style={{ padding: '0 24px', height: 36, fontWeight: 500 }}
          >
            {saving ? '保存中...' : `保存${roleLabel}档案`}
          </Button>
        </Space>
      </div>

      {/* Main Container */}
      <Spin spinning={loading}>
        <div style={{ maxWidth: 1440, margin: '0 auto', padding: '0 16px' }}>
          <ProForm
            formRef={formRef}
            submitter={false}
            layout="horizontal"
            grid
            rowProps={{ gutter: [16, 12] }}
          >
            <Col span={24}>
              <Collapse
                activeKey={activeCollapseKeys}
                onChange={(keys) => setActiveCollapseKeys(keys as string[])}
                expandIcon={({ isActive }) => (
                  isActive ? <CaretDownOutlined /> : <CaretRightOutlined />
                )}
                bordered={false}
                style={{ backgroundColor: 'transparent' }}
                items={[
                  // Section 1: 基础信息
                  {
                    key: 'basic',
                    label: renderSectionHeader('基础信息'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <div>
                        {/* Row 1: Legal Name, USCC, Code */}
                        <Row gutter={[16, 12]} align="middle">
                          <Col xs={24} lg={10}>
                            <ProFormText
                              name="legalName"
                              label="公司抬头"
                              placeholder="请输入企业法人营业执照全称"
                              rules={[{ required: true, message: '请输入公司抬头全称' }]}
                            />
                          </Col>

                          <Col xs={24} lg={9}>
                            <Form.Item
                              label={
                                <Space size={4}>
                                  <span>社会统一信用代码</span>
                                  <Tooltip title="18位纳税人统一社会信用代码">
                                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                                  </Tooltip>
                                </Space>
                              }
                              style={{ marginBottom: 0 }}
                            >
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item
                                  name="unifiedSocialCreditCode"
                                  noStyle
                                  rules={[
                                    {
                                      pattern: /^[0-9ABCDEFGHJKLMNPQRTUWXY]{18}$/,
                                      message: '请输入正确的18位统一社会信用代码',
                                    },
                                  ]}
                                >
                                  <Input
                                    placeholder="91510108MAKB..."
                                    allowClear
                                    style={{ fontFamily: 'monospace' }}
                                  />
                                </Form.Item>
                                <Button
                                  type="primary"
                                  icon={<SafetyCertificateOutlined />}
                                  onClick={handleTianyanchaVerify}
                                >
                                  校验公司信息
                                </Button>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          <Col xs={24} lg={5}>
                            <ProFormText
                              name="code"
                              label="代码"
                              placeholder="如 CDRT"
                              disabled={Boolean(partnerId)}
                              rules={[
                                { required: true, message: '请输入唯一代码' },
                                {
                                  pattern: /^[A-Za-z0-9_-]+$/,
                                  message: '仅支持字母数字',
                                },
                              ]}
                            />
                          </Col>
                        </Row>

                        {/* Row 2: 中文地址 */}
                        <Row gutter={[16, 12]} style={{ marginTop: 8 }}>
                          <Col span={24}>
                            <Form.Item label="中文地址" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="regionCodes" noStyle>
                                  <Cascader
                                    options={pcaCascaderOptions}
                                    placeholder="省 / 市 / 区"
                                    style={{ width: 280 }}
                                    allowClear
                                    showSearch
                                  />
                                </Form.Item>
                                <Form.Item name="addressDetail" noStyle>
                                  <Input
                                    placeholder="请输入详细地址"
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>
                        </Row>

                        {/* Row 3: 英文名 */}
                        <Row gutter={[16, 12]} style={{ marginTop: 8 }}>
                          <Col span={24}>
                            <ProFormText
                              name="nameEn"
                              label="英文名"
                              placeholder="请输入英文名称"
                            />
                          </Col>
                        </Row>

                        {/* Row 4: 英文地址 */}
                        <Row gutter={[16, 12]}>
                          <Col span={24}>
                            <ProFormText
                              name="addressEn"
                              label="英文地址"
                              placeholder="请输入英文地址"
                            />
                          </Col>
                        </Row>

                        {/* Row 5: 性质, 类型, 开发方式, 业务类型 */}
                        <Row gutter={[16, 12]} align="middle" style={{ marginTop: 4 }}>
                          <Col xs={24} sm={12} md={4}>
                            <ProFormSelect
                              name="nature"
                              label="性质"
                              options={[
                                { label: '客户', value: '客户' },
                                { label: '供应商', value: '供应商' },
                              ]}
                              initialValue={roleLabel}
                              disabled
                            />
                          </Col>

                          <Col xs={24} sm={12} md={6}>
                            <ProFormCheckbox.Group
                              name="customerTypes"
                              label="类型"
                              options={CUSTOMER_TYPE_OPTIONS}
                              initialValue={[1]}
                            />
                          </Col>

                          <Col xs={24} sm={12} md={6}>
                            <ProFormSelect
                              name="developmentMethod"
                              label="开发方式"
                              options={DEVELOPMENT_METHOD_OPTIONS}
                              initialValue="自主开发"
                            />
                          </Col>

                          <Col xs={24} md={8}>
                            <ProFormSelect
                              name="businessTypes"
                              label="业务类型"
                              mode="multiple"
                              options={BUSINESS_TYPE_OPTIONS}
                              placeholder="请选择适用的业务类型"
                              initialValue={[1]}
                            />
                          </Col>
                        </Row>

                        <Divider style={{ margin: '14px 0' }} />

                        {/* Row 6: 8 Personnel Assignment Slots (4 Rows x 2 Columns) */}
                        <div style={{ marginBottom: 12 }}>
                          <Text strong style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
                            责任人员分配矩阵
                          </Text>
                        </div>

                        <Row gutter={[20, 10]}>
                          {/* Slot 1: 创建人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="创建人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignCreatorUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignCreatorUser', 'assignCreatorOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignCreatorOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 2: 操作人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="操作人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignOperatorUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignOperatorUser', 'assignOperatorOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignOperatorOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 3: 业务人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="业务人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignSalesUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignSalesUser', 'assignSalesOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignSalesOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 4: 客服人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="客服人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignServiceUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignServiceUser', 'assignServiceOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignServiceOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 5: 关联人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="关联人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignContactUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignContactUser', 'assignContactOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignContactOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 6: 单证人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="单证人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignDocUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignDocUser', 'assignDocOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignDocOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 7: 商务人员 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="商务人员" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignCommercialUser" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignCommercialUser', 'assignCommercialOrg', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignCommercialOrg" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* Slot 8: 关联人员2 */}
                          <Col xs={24} md={12}>
                            <Form.Item label="关联人员2" style={{ marginBottom: 0 }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="assignContact2User" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={userSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                    onChange={(val) => handleUserChange('assignContact2User', 'assignContact2Org', val)}
                                  />
                                </Form.Item>
                                <Form.Item name="assignContact2Org" noStyle>
                                  <Select
                                    showSearch
                                    placeholder="请选择"
                                    options={orgSelectOptions}
                                    style={{ width: '50%' }}
                                    allowClear
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>
                        </Row>

                        <Divider style={{ margin: '14px 0' }} />

                        {/* Row 7: 公司别名 */}
                        <Row gutter={[16, 12]} align="middle">
                          <Col span={24}>
                            <Form.Item label="公司别名" style={{ marginBottom: 0 }}>
                              <Space wrap align="center">
                                <Input
                                  placeholder="输入企业别名"
                                  value={newAliasInput}
                                  onChange={(e) => setNewAliasInput(e.target.value)}
                                  onPressEnter={handleAddAlias}
                                  style={{ width: 220 }}
                                />
                                <Button type="dashed" onClick={handleAddAlias}>
                                  添加
                                </Button>
                                {aliases.map((alias) => (
                                  <Tag
                                    key={alias}
                                    closable
                                    onClose={() => handleRemoveAlias(alias)}
                                    color="blue"
                                    style={{ fontSize: 12, padding: '2px 8px' }}
                                  >
                                    {alias}
                                  </Tag>
                                ))}
                              </Space>
                            </Form.Item>
                          </Col>
                        </Row>
                      </div>
                    ),
                  },

                  // Section 2: 结算信息
                  {
                    key: 'settlement',
                    label: renderSectionHeader('结算信息'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <div>
                        <Row gutter={[16, 12]} align="middle">
                          {/* 对账方式 */}
                          <Col xs={24} sm={12} md={4}>
                            <ProFormSelect
                              name="statementMode"
                              label="对账方式"
                              options={STATEMENT_MODE_OPTIONS}
                              rules={[{ required: true, message: '请选择对账方式' }]}
                            />
                          </Col>

                          {/* 结算方式 */}
                          <Col xs={24} sm={12} md={4}>
                            <ProFormSelect
                              name="settlementMethod"
                              label="结算方式"
                              options={SETTLEMENT_METHOD_OPTIONS}
                              rules={[{ required: true, message: '请选择结算方式' }]}
                            />
                          </Col>

                          {/* 结算日期 */}
                          <Col xs={24} sm={12} md={5}>
                            <Form.Item
                              label={
                                <Space size={4}>
                                  <span>结算日期</span>
                                  <Tooltip title="每月固定结算与对账截止日">
                                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                                  </Tooltip>
                                </Space>
                              }
                              style={{ marginBottom: 0 }}
                            >
                              <Space.Compact style={{ width: '100%' }}>
                                <div
                                  style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    padding: '0 8px',
                                    backgroundColor: '#fafafa',
                                    border: '1px solid #d9d9d9',
                                    borderRight: 0,
                                    borderRadius: '6px 0 0 6px',
                                    color: '#595959',
                                  }}
                                >
                                  每月
                                </div>
                                <Form.Item name="settlementDay" noStyle>
                                  <Select
                                    options={SETTLEMENT_DAY_OPTIONS}
                                    placeholder="请选择"
                                    style={{ width: '100%' }}
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* 账期 */}
                          <Col xs={24} sm={12} md={5}>
                            <Form.Item
                              label={
                                <Space size={4}>
                                  <span>账期</span>
                                  <Tooltip title="账期基准与有效信用天数">
                                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                                  </Tooltip>
                                </Space>
                              }
                              style={{ marginBottom: 0 }}
                            >
                              <Space.Compact style={{ width: '100%' }}>
                                <Form.Item name="settlementBase" noStyle>
                                  <Select
                                    options={SETTLEMENT_BASE_OPTIONS}
                                    placeholder="请选择"
                                    style={{ width: '55%' }}
                                  />
                                </Form.Item>
                                <Form.Item name="creditDays" noStyle>
                                  <Input
                                    placeholder="天数"
                                    style={{ width: '45%', textAlign: 'center' }}
                                    suffix="天"
                                  />
                                </Form.Item>
                              </Space.Compact>
                            </Form.Item>
                          </Col>

                          {/* 信用额度(本币) */}
                          <Col xs={24} sm={12} md={6}>
                            <ProFormDigit
                              name="creditLimit"
                              label={
                                <Space size={4}>
                                  <span>信用额度(本币)</span>
                                  <Tooltip title="本币最大允许未核销应收账款额度">
                                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                                  </Tooltip>
                                </Space>
                              }
                              placeholder="输入信用额度"
                              min={0}
                              fieldProps={{
                                precision: 2,
                                addonAfter: '元',
                              }}
                            />
                          </Col>
                        </Row>

                        <Row gutter={[16, 12]} align="middle" style={{ marginTop: 8 }}>
                          {/* 结算币种 */}
                          <Col xs={24} sm={12} md={4}>
                            <ProFormSelect
                              name="settlementCurrency"
                              label="结算币种"
                              options={currencyOptions}
                              rules={[{ required: true, message: '请选择结算币种' }]}
                            />
                          </Col>

                          {/* 利息规则 */}
                          <Col xs={24} sm={12} md={6}>
                            <Form.Item label="利息规则" style={{ marginBottom: 0 }}>
                              <Button
                                type="link"
                                onClick={() => setInterestModalOpen(true)}
                                style={{ padding: 0, fontWeight: 500 }}
                              >
                                {interestRule.enabled
                                  ? `已启用 (万分之${interestRule.dailyRateBp || 5}/日)`
                                  : '编辑规则'}
                              </Button>
                            </Form.Item>
                          </Col>
                        </Row>
                      </div>
                    ),
                  },

                  // Section 3: 账户信息
                  {
                    key: 'accounts',
                    label: renderSectionHeader('账户信息'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <AccountCardList
                        partnerId={partnerId}
                        currencyOptions={currencyOptions}
                      />
                    ),
                  },

                  // Section 4: 联系方式
                  {
                    key: 'contacts',
                    label: renderSectionHeader('联系方式'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <ContactCardList
                        contacts={contacts}
                        onChange={setContacts}
                      />
                    ),
                  },

                  // Section 5: 常用信息 (Shipping Presets)
                  {
                    key: 'presets',
                    label: renderSectionHeader('常用信息'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <ShippingPresetSection partnerId={partnerId} />
                    ),
                  },

                  // Section 6: 合同管理
                  {
                    key: 'contracts',
                    label: renderSectionHeader('合同管理'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <ContractCardList partnerId={partnerId} />
                    ),
                  },

                  // Section 7: 客户备注
                  {
                    key: 'remark',
                    label: renderSectionHeader('客户备注'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <ProFormTextArea
                        name="remark"
                        placeholder="可以添加客户信息录入时的备注信息"
                        fieldProps={{ rows: 3 }}
                      />
                    ),
                  },

                  // Section 8: 操作记录
                  {
                    key: 'logs',
                    label: renderSectionHeader('操作记录'),
                    style: {
                      marginBottom: 16,
                      backgroundColor: '#ffffff',
                      borderRadius: 8,
                      border: '1px solid #f0f0f0',
                      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.02)',
                    },
                    children: (
                      <AuditLogSection partnerId={partnerId} />
                    ),
                  },
                ]}
              />
            </Col>
          </ProForm>
        </div>
      </Spin>

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
