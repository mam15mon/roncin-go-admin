import {
  ArrowLeftOutlined,
  BankOutlined,
  CheckCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ProForm,
  ProFormCheckbox,
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { history, useLocation, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Cascader,
  Col,
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
  partnerServiceCreatePartnerSettlementRule,
  partnerServiceGetPartner,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartner,
  partnerServiceUpdatePartnerSettlementRule,
} from '@/services/roncin/partnerService';
import { pcaCascaderOptions } from '@/utils/chinaDivision';
import PartnerSecondary from './partner-secondary';

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
  { label: '单票对账', value: 1 },
  { label: '汇总对账', value: 2 },
];

const SETTLEMENT_METHOD_OPTIONS = [
  { label: '单票结算', value: 1 },
  { label: '月结', value: 2 },
  { label: '周结', value: 3 },
  { label: '半月结', value: 4 },
  { label: '双月结', value: 5 },
  { label: '季结', value: 6 },
  { label: '45天', value: 7 },
  { label: '预付', value: 8 },
];

const SETTLEMENT_BASE_OPTIONS = [
  { label: '账单日', value: 1 },
  { label: '开航日 (ETD)', value: 2 },
  { label: '到港日 (ETA)', value: 3 },
];

const BILLING_TERM_OPTIONS = [
  { label: '票结', value: '票结' },
  { label: '月结', value: '月结' },
  { label: '约定账期', value: '约定账期' },
];

const SETTLEMENT_DAY_OPTIONS = Array.from({ length: 31 }, (_, i) => ({
  label: `每月 ${i + 1} 日`,
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
  const [secondaryOpen, setSecondaryOpen] = useState(false);

  // Options state
  const [users, setUsers] = useState<API.AdminUser[]>([]);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<{ label: string; value: string }[]>([
    { label: 'CNY (人民币)', value: 'CNY' },
    { label: 'USD (美元)', value: 'USD' },
    { label: 'EUR (欧元)', value: 'EUR' },
    { label: 'HKD (港币)', value: 'HKD' },
  ]);

  const [existingRuleId, setExistingRuleId] = useState<string | undefined>(undefined);

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
        const [usersRes, orgsRes, curRes] = await Promise.allSettled([
          adminServiceListUsers({ page: 1, pageSize: 100 }),
          adminServiceListOrganizations(),
          masterDataServiceListCurrencies(),
        ]);

        if (usersRes.status === 'fulfilled' && usersRes.value.data) {
          setUsers(usersRes.value.data);
        }
        if (orgsRes.status === 'fulfilled' && orgsRes.value.data) {
          setOrganizations(orgsRes.value.data);
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
        // Silently keep fallbacks
      }
    };

    fetchOptions();
  }, []);

  // Load partner detail when editing
  useEffect(() => {
    if (partnerId) {
      setLoading(true);
      Promise.all([
        partnerServiceGetPartner({ id: partnerId }),
        partnerServiceListPartnerSettlementRules({
          partnerId,
          roleType,
        }),
      ])
        .then(([partnerRes, ruleRes]) => {
          const p = partnerRes.data;
          setPartner(p);
          const rules = ruleRes.data ?? [];
          const currentRule = rules[0];

          if (currentRule?.id) {
            setExistingRuleId(currentRule.id);
          } else {
            setExistingRuleId(undefined);
          }

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

              // User assignments
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

              aliases: p.aliases && p.aliases.length > 0 ? p.aliases : [],
              contacts: p.contacts && p.contacts.length > 0 ? p.contacts : [],

              statementMode: currentRule?.statementMode ?? 1,
              settlementMethod: currentRule?.settlementMethod ?? 2,
              settlementBase: currentRule?.settlementBase ?? 1,
              settlementDay: currentRule?.settlementDay ?? 25,
              settlementCurrency: currentRule?.settlementCurrency ?? 'CNY',
              creditDays: currentRule?.settlementCycleDays ?? 30,
              billingTerm: '月结',
              isRuleActive: currentRule?.isActive ?? true,
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
      setExistingRuleId(undefined);
      formRef.current?.resetFields();
      formRef.current?.setFieldsValue({
        enabled: true,
        nature: roleLabel,
        customerTypes: [1],
        developmentMethod: '自主开发',
        businessTypes: [1],
        statementMode: 1,
        settlementMethod: 2,
        settlementBase: 1,
        settlementDay: 25,
        settlementCurrency: 'CNY',
        creditDays: 30,
        billingTerm: '月结',
        isRuleActive: true,
        aliases: [],
        contacts: [],
      });
    }
  }, [partnerId, roleType, roleLabel, message]);

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

  const handleTianyanchaVerify = () => {
    const legalName = formRef.current?.getFieldValue('legalName');
    if (!legalName?.trim()) {
      message.warning('请先填写公司抬头再进行天眼查校验');
      return;
    }
    const targetUrl = `https://www.tianyancha.com/nsearch?key=${encodeURIComponent(
      legalName.trim(),
    )}`;
    window.open(targetUrl, '_blank', 'noopener,noreferrer');
  };

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

      addAssignment(2, 'assignOperatorUser', 'assignOperatorOrg');
      addAssignment(3, 'assignSalesUser', 'assignSalesOrg');
      addAssignment(4, 'assignServiceUser', 'assignServiceOrg');
      addAssignment(5, 'assignDocUser', 'assignDocOrg');
      addAssignment(6, 'assignCommercialUser', 'assignCommercialOrg');
      addAssignment(7, 'assignContactUser', 'assignContactOrg');

      const contacts: API.PartnerContactInput[] = (values.contacts || [])
        .filter((c: any) => c?.name?.trim())
        .map((c: any) => ({
          name: c.name.trim(),
          phone: c.phone?.trim(),
          email: c.email?.trim(),
          note: c.note?.trim(),
          isPrimary: Boolean(c.isPrimary),
        }));

      const aliases: API.PartnerAliasInput[] = (values.aliases || [])
        .filter((a: any) => a?.aliasName?.trim())
        .map((a: any) => ({
          aliasName: a.aliasName.trim(),
          sortOrder: Number(a.sortOrder || 0),
        }));

      let savedPartnerId = partnerId;

      if (partnerId) {
        await partnerServiceUpdatePartner(
          { id: partnerId },
          {
            id: partnerId,
            legalName: values.legalName.trim(),
            unifiedSocialCreditCode: values.unifiedSocialCreditCode?.trim(),
            registeredAddress: values.addressDetail?.trim(),
            enabled: values.enabled ?? true,
            roles: [{ type: roleType, enabled: values.enabled ?? true }],
            profile,
            assignments,
            contacts,
            aliases,
          },
        );
        message.success(`${roleLabel}档案已成功更新`);
      } else {
        const createRes = await partnerServiceCreatePartner({
          code: values.code?.trim() || '',
          legalName: values.legalName.trim(),
          unifiedSocialCreditCode: values.unifiedSocialCreditCode?.trim(),
          registeredAddress: values.addressDetail?.trim(),
          roles: [{ type: roleType, enabled: values.enabled ?? true }],
          profile,
          assignments,
          contacts,
          aliases,
        });
        savedPartnerId = createRes.data?.id;
        message.success(`${roleLabel}档案已成功创建`);
      }

      if (savedPartnerId) {
        const settlementRuleInput = {
          statementMode: Number(values.statementMode || 1),
          settlementMethod: Number(values.settlementMethod || 2),
          settlementBase: Number(values.settlementBase || 1),
          settlementDay: Number(values.settlementDay || 25),
          settlementCurrency: values.settlementCurrency || 'CNY',
          settlementCycleDays: Number(values.creditDays || 30),
          isActive: values.isRuleActive ?? true,
        };

        if (existingRuleId) {
          await partnerServiceUpdatePartnerSettlementRule(
            {
              partnerId: savedPartnerId,
              roleType,
              id: existingRuleId,
            },
            {
              id: existingRuleId,
              partnerId: savedPartnerId,
              roleType,
              rule: settlementRuleInput,
            },
          );
        } else {
          await partnerServiceCreatePartnerSettlementRule(
            {
              partnerId: savedPartnerId,
              roleType,
            },
            {
              partnerId: savedPartnerId,
              roleType,
              rule: settlementRuleInput,
            },
          );
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
    ? `${partner.legalName} 档案详情`
    : isCreate
    ? `新建${roleLabel}档案`
    : `${roleLabel}档案详情`;

  return (
    <div style={{ minHeight: '100%', paddingBottom: 40 }}>
      {/* Sticky Top Header Navigation & Action Bar */}
      <div
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 99,
          backgroundColor: '#ffffff',
          borderBottom: '1px solid #f0f0f0',
          padding: '12px 24px',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.04)',
          marginBottom: 16,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: 12,
        }}
      >
        {/* Left: Breadcrumb Path */}
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
        <Space size={10}>
          {partner && (
            <Button
              icon={<BankOutlined />}
              onClick={() => setSecondaryOpen(true)}
            >
              账户与合同资料
            </Button>
          )}
          <Button onClick={() => history.push(listUrl)} disabled={saving}>
            取消
          </Button>
          <Button
            type="primary"
            onClick={handleSubmit}
            loading={saving}
            icon={<CheckCircleOutlined />}
            style={{ padding: '0 24px' }}
          >
            {saving ? '保存中...' : `保存${roleLabel}档案`}
          </Button>
        </Space>
      </div>

      {/* Main Form Content */}
      <Spin spinning={loading}>
        <div style={{ maxWidth: 1360, margin: '0 auto', padding: '0 16px' }}>
          <ProForm
            formRef={formRef}
            submitter={false}
            layout="horizontal"
            grid
            rowProps={{ gutter: [16, 12] }}
          >
            {/* Section 1: 基础信息 */}
            <Col span={24}>
              <Card
                bordered={false}
                style={{
                  borderRadius: 8,
                  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.04)',
                  marginBottom: 16,
                }}
                styles={{ body: { padding: '24px 28px' } }}
              >
                {/* Section Header */}
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    marginBottom: 20,
                    paddingBottom: 10,
                    borderBottom: '1px solid #f0f0f0',
                  }}
                >
                  <div
                    style={{
                      width: 4,
                      height: 16,
                      backgroundColor: '#1677ff',
                      borderRadius: 2,
                    }}
                  />
                  <span
                    style={{
                      fontWeight: 600,
                      fontSize: 15,
                      color: 'rgba(0, 0, 0, 0.88)',
                    }}
                  >
                    基础信息
                  </span>
                  <Text type="secondary" style={{ fontSize: 12, marginLeft: 6 }}>
                    企业抬头、统一信用代码、中英文地址与内部责任人员
                  </Text>
                </div>

                {/* Row 1: Legal Name, USCC with Tianyancha, Code */}
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
                    <Form.Item label="社会统一信用代码" required style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item
                          name="unifiedSocialCreditCode"
                          noStyle
                          rules={[
                            { required: true, message: '请输入统一社会信用代码' },
                            {
                              pattern: /^[0-9ABCDEFGHJKLMNPQRTUWXY]{18}$/,
                              message: '请输入正确的18位统一社会信用代码',
                            },
                          ]}
                        >
                          <Input
                            placeholder="18位社会信用代码"
                            allowClear
                            style={{ fontFamily: 'monospace' }}
                          />
                        </Form.Item>
                        <Tooltip title="在天眼查新窗口快速核验工商信用档案">
                          <Button
                            icon={<SafetyCertificateOutlined style={{ color: '#1677ff' }} />}
                            onClick={handleTianyanchaVerify}
                          >
                            校验
                          </Button>
                        </Tooltip>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  <Col xs={24} lg={5}>
                    <ProFormText
                      name="code"
                      label="单位编码"
                      placeholder="如 CUST001"
                      disabled={Boolean(partnerId)}
                      rules={[
                        { required: true, message: '请输入单位唯一编码' },
                        {
                          pattern: /^[A-Za-z0-9_-]+$/,
                          message: '仅支持字母数字下划线',
                        },
                      ]}
                    />
                  </Col>
                </Row>

                {/* Row 2: Chinese Address with Cascader + Detail */}
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
                            placeholder="请输入详细办公/注册街道地址（如：浦东新区东方路888号1201室）"
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>
                </Row>

                {/* Row 3: English Name */}
                <Row gutter={[16, 12]} style={{ marginTop: 8 }}>
                  <Col span={24}>
                    <ProFormText
                      name="nameEn"
                      label="英文名称"
                      placeholder="请输入企业外文英文名称（如 RONCIN INTERNATIONAL LOGISTICS CO., LTD）"
                    />
                  </Col>
                </Row>

                {/* Row 4: English Address */}
                <Row gutter={[16, 12]}>
                  <Col span={24}>
                    <ProFormText
                      name="addressEn"
                      label="英文地址"
                      placeholder="请输入企业外文英文地址（如 ROOM 1201, NO. 888 PUDONG SOUTH ROAD, SHANGHAI, CHINA）"
                    />
                  </Col>
                </Row>

                {/* Row 5: Nature, Customer Types, Development Method, Business Types */}
                <Row gutter={[16, 12]} align="middle" style={{ marginTop: 4 }}>
                  <Col xs={24} sm={12} md={4}>
                    <ProFormSelect
                      name="nature"
                      label="单位性质"
                      options={[
                        { label: '客户', value: '客户' },
                        { label: '供应商', value: '供应商' },
                      ]}
                      initialValue={roleLabel}
                      disabled
                    />
                  </Col>

                  <Col xs={24} sm={12} md={5}>
                    <ProFormCheckbox.Group
                      name="customerTypes"
                      label="客户类型"
                      options={CUSTOMER_TYPE_OPTIONS}
                      initialValue={[1]}
                    />
                  </Col>

                  <Col xs={24} sm={12} md={5}>
                    <ProFormSelect
                      name="developmentMethod"
                      label="开发方式"
                      options={DEVELOPMENT_METHOD_OPTIONS}
                      initialValue="自主开发"
                    />
                  </Col>

                  <Col xs={24} md={10}>
                    <ProFormCheckbox.Group
                      name="businessTypes"
                      label="业务类型"
                      options={BUSINESS_TYPE_OPTIONS}
                      initialValue={[1]}
                    />
                  </Col>
                </Row>

                <Divider style={{ margin: '16px 0' }} />

                {/* Row 6: User Assignment Pairs */}
                <div style={{ marginBottom: 14 }}>
                  <Text strong style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
                    内部对接与责任人员指派
                  </Text>
                </div>

                <Row gutter={[16, 10]}>
                  {/* Operator */}
                  <Col xs={24} md={12}>
                    <Form.Item label="操作人员" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignOperatorUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择操作人"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignOperatorOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  {/* Sales */}
                  <Col xs={24} md={12}>
                    <Form.Item label="业务人员" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignSalesUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择业务员"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignSalesOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  {/* Customer Service */}
                  <Col xs={24} md={12}>
                    <Form.Item label="客服人员" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignServiceUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择客服"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignServiceOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  {/* Document */}
                  <Col xs={24} md={12}>
                    <Form.Item label="单证人员" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignDocUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择单证员"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignDocOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  {/* Commercial */}
                  <Col xs={24} md={12}>
                    <Form.Item label="商务人员" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignCommercialUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择商务"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignCommercialOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  {/* Internal Contact */}
                  <Col xs={24} md={12}>
                    <Form.Item label="内部对接人" style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="assignContactUser" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="选择对接人"
                            options={userSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                        <Form.Item name="assignContactOrg" noStyle>
                          <Select
                            showSearch
                            filterOption={(input, option) =>
                              String(option?.label ?? '')
                                .toLowerCase()
                                .includes(input.toLowerCase())
                            }
                            placeholder="所属公司/部门"
                            options={orgSelectOptions}
                            style={{ width: '50%' }}
                            allowClear
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>
                </Row>

                <Divider style={{ margin: '16px 0' }} />

                {/* Row 7: Enterprise Aliases & Contacts List */}
                <Row gutter={[16, 12]}>
                  <Col xs={24} md={10}>
                    <ProFormList
                      name="aliases"
                      label="企业常用简称 / 别名"
                      creatorButtonProps={{
                        creatorButtonText: '添加企业别名',
                        type: 'dashed',
                        size: 'small',
                      }}
                      copyIconProps={false}
                    >
                      <Space align="start" size={6}>
                        <ProFormText
                          name="aliasName"
                          placeholder="例如：上海罗新"
                          rules={[{ required: true, message: '请输入别名' }]}
                        />
                        <ProFormDigit
                          name="sortOrder"
                          placeholder="排序"
                          min={0}
                          width={70}
                          fieldProps={{ precision: 0 }}
                        />
                      </Space>
                    </ProFormList>
                  </Col>

                  <Col xs={24} md={14}>
                    <ProFormList
                      name="contacts"
                      label="客商联系人通讯录"
                      creatorButtonProps={{
                        creatorButtonText: '添加联系人',
                        type: 'dashed',
                        size: 'small',
                      }}
                      copyIconProps={false}
                    >
                      <Space align="start" wrap size={6}>
                        <ProFormText
                          name="name"
                          placeholder="姓名"
                          width={90}
                          rules={[{ required: true, message: '姓名' }]}
                        />
                        <ProFormText name="phone" placeholder="电话" width={110} />
                        <ProFormText name="email" placeholder="邮箱" width={130} />
                        <ProFormSwitch name="isPrimary" label="主" />
                        <ProFormText name="note" placeholder="职务/备注" width={90} />
                      </Space>
                    </ProFormList>
                  </Col>
                </Row>
              </Card>
            </Col>

            {/* Section 2: 结算与财务信息 */}
            <Col span={24}>
              <Card
                bordered={false}
                style={{
                  borderRadius: 8,
                  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.04)',
                  marginBottom: 16,
                }}
                styles={{ body: { padding: '24px 28px' } }}
              >
                {/* Section Header */}
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    marginBottom: 20,
                    paddingBottom: 10,
                    borderBottom: '1px solid #f0f0f0',
                  }}
                >
                  <div
                    style={{
                      width: 4,
                      height: 16,
                      backgroundColor: '#52c41a',
                      borderRadius: 2,
                    }}
                  />
                  <span
                    style={{
                      fontWeight: 600,
                      fontSize: 15,
                      color: 'rgba(0, 0, 0, 0.88)',
                    }}
                  >
                    结算信息
                  </span>
                  <Text type="secondary" style={{ fontSize: 12, marginLeft: 6 }}>
                    对账模式、结算周期、账期天数与币种风控
                  </Text>
                </div>

                <Row gutter={[16, 12]} align="middle">
                  <Col xs={24} sm={12} md={4}>
                    <ProFormSelect
                      name="statementMode"
                      label="对账方式"
                      options={STATEMENT_MODE_OPTIONS}
                      rules={[{ required: true, message: '请选择对账方式' }]}
                    />
                  </Col>

                  <Col xs={24} sm={12} md={4}>
                    <ProFormSelect
                      name="settlementMethod"
                      label="结算方式"
                      options={SETTLEMENT_METHOD_OPTIONS}
                      rules={[{ required: true, message: '请选择结算方式' }]}
                    />
                  </Col>

                  <Col xs={24} sm={12} md={6}>
                    <Form.Item label="结算日期" required style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="settlementBase" noStyle>
                          <Select
                            options={SETTLEMENT_BASE_OPTIONS}
                            placeholder="结算基准"
                            style={{ width: '50%' }}
                          />
                        </Form.Item>
                        <Form.Item name="settlementDay" noStyle>
                          <Select
                            options={SETTLEMENT_DAY_OPTIONS}
                            placeholder="结算日"
                            style={{ width: '50%' }}
                          />
                        </Form.Item>
                      </Space.Compact>
                    </Form.Item>
                  </Col>

                  <Col xs={24} sm={12} md={5}>
                    <Form.Item label="账期" required style={{ marginBottom: 0 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item name="billingTerm" noStyle>
                          <Select
                            options={BILLING_TERM_OPTIONS}
                            placeholder="账期模式"
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

                  <Col xs={24} sm={12} md={5}>
                    <ProFormSelect
                      name="settlementCurrency"
                      label="结算币种"
                      options={currencyOptions}
                      rules={[{ required: true, message: '请选择结算币种' }]}
                    />
                  </Col>
                </Row>
              </Card>
            </Col>

            {/* Section 3: 客户备注 */}
            <Col span={24}>
              <Card
                bordered={false}
                style={{
                  borderRadius: 8,
                  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.04)',
                }}
                styles={{ body: { padding: '24px 28px' } }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    marginBottom: 14,
                    paddingBottom: 10,
                    borderBottom: '1px solid #f0f0f0',
                  }}
                >
                  <div
                    style={{
                      width: 4,
                      height: 16,
                      backgroundColor: '#fa8c16',
                      borderRadius: 2,
                    }}
                  />
                  <span
                    style={{
                      fontWeight: 600,
                      fontSize: 15,
                      color: 'rgba(0, 0, 0, 0.88)',
                    }}
                  >
                    {roleLabel}备注
                  </span>
                  <Text type="secondary" style={{ fontSize: 12, marginLeft: 6 }}>
                    可以添加该客商信息录入时的特殊背景、协议说明或操作注意点
                  </Text>
                </div>

                <ProFormTextArea
                  name="remark"
                  placeholder="请输入客商录入备注信息..."
                  fieldProps={{ rows: 3 }}
                />
              </Card>
            </Col>
          </ProForm>
        </div>
      </Spin>

      {/* Secondary Drawer for Bank Accounts and Contracts when editing */}
      {partner && (
        <PartnerSecondary
          partner={partner}
          open={secondaryOpen}
          canManage={true}
          onClose={() => setSecondaryOpen(false)}
        />
      )}
    </div>
  );
}
