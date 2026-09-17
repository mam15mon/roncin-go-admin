import {
  ApartmentOutlined,
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  ProFormRadio,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import {
  Button,
  Cascader,
  Col,
  Divider,
  Form,
  Input,
  Row,
  Select,
  Space,
  Tooltip,
  Typography,
} from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import {
  PartnerBusinessType,
  PartnerCustomerType,
  PartnerRoleType,
} from '@/enums.generated';
import { pcaCascaderOptions } from '@/utils/chinaDivision';

const { Text } = Typography;

// 同列标签统一等宽（右对齐），保证多行控件的左缘上下成列对齐。
// 各列宽度集中在此配置：调整某列宽度只需改对应键值。
const LABEL_COL_WIDTH = {
  /** 主列：公司抬头/中文地址/英文名/英文地址/公司别名/人员矩阵 */
  primary: 88,
  /** 境外代理标签列 */
  foreignPrimary: 120,
  /** 社会统一信用代码列 */
  uscc: 144,
  /** 统一属性栏标签宽度，确保4列栅格下对齐 */
  attribute: 80,
} as const;

const labelCol = (width: number) => ({ style: { width } });

export const BUSINESS_TYPE_OPTIONS = [
  {
    label: 'SE（海运出口）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_SE,
  },
  {
    label: 'SI（海运进口）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_SI,
  },
  {
    label: 'AE（空运出口）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_AE,
  },
  {
    label: 'AI（空运进口）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_AI,
  },
  {
    label: 'LAND（陆运业务）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_LAND,
  },
  {
    label: 'RAIL（铁路运输）',
    value: PartnerBusinessType.PARTNER_BUSINESS_TYPE_RAIL,
  },
];

export const CUSTOMER_TYPE_OPTIONS = [
  { label: '直客', value: PartnerCustomerType.PARTNER_CUSTOMER_TYPE_DIRECT },
  { label: '同行', value: PartnerCustomerType.PARTNER_CUSTOMER_TYPE_PEER },
];

export const DEVELOPMENT_METHOD_OPTIONS = [
  { label: '自主开发', value: '自主开发' },
  { label: '网络推广', value: '网络推广' },
  { label: '老客转介', value: '老客转介' },
  { label: '商务分配', value: '商务分配' },
  { label: '展会获取', value: '展会获取' },
  { label: '公开招标', value: '公开招标' },
  { label: '其它方式', value: '其它方式' },
];

type BasicInfoSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  roleLabel: string;
  roleType?: number;
  userSelectOptions: { label: string; value: string }[];
  orgSelectOptions?: { label: string; value: string }[];
  aliases: string[];
  onAliasesChange?: (aliases: string[]) => void;
  newAliasInput?: string;
  setNewAliasInput?: (val: string) => void;
  onAddAlias?: () => void;
  onRemoveAlias?: (alias: string) => void;
  onTianyanchaVerify: () => void;
  onUserChange: (userField: string, orgField: string, userId?: string) => void;
  creatorMeta?: React.ReactNode;
};

export default function BasicInfoSection({
  collapsed,
  onCollapseChange,
  roleLabel,
  roleType,
  userSelectOptions,
  orgSelectOptions = [],
  aliases,
  onAliasesChange,
  onTianyanchaVerify,
  onUserChange,
  creatorMeta,
}: BasicInfoSectionProps) {
  const isForeignAgent =
    roleType === PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT ||
    roleLabel === '国外代理';
  const isSupplier =
    roleType === PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER ||
    roleLabel === '供应商';
  const isCustomer = !isForeignAgent && !isSupplier;

  return (
    <SectionCard
      id="section-basic"
      sectionKey="basic"
      title="基础信息"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <div>
        {/* Row 1: Legal Name, USCC (仅国内企业), Code */}
        <Row gutter={[16, 12]} align="middle">
          <Col xs={24} lg={isForeignAgent ? 18 : 10}>
            <ProFormText
              name="legalName"
              label={isForeignAgent ? '公司抬头 (英文)' : '公司抬头'}
              labelCol={labelCol(
                isForeignAgent
                  ? LABEL_COL_WIDTH.foreignPrimary
                  : LABEL_COL_WIDTH.primary,
              )}
              placeholder={
                isForeignAgent
                  ? '请输入境外公司法定英文全称（公司抬头），如 PACIFIC LOGISTICS INC.'
                  : '请输入企业法人营业执照全称'
              }
              rules={[
                {
                  required: true,
                  message: isForeignAgent
                    ? '公司抬头 (英文) 为必填项'
                    : '公司抬头为必填项',
                },
              ]}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {!isForeignAgent && (
            <Col xs={24} lg={9}>
              <Form.Item
                name="unifiedSocialCreditCode"
                label={
                  <Space size={4}>
                    <span>社会统一信用代码</span>
                    <Tooltip title="18位纳税人统一社会信用代码">
                      <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                    </Tooltip>
                  </Space>
                }
                labelCol={labelCol(LABEL_COL_WIDTH.uscc)}
                rules={[
                  {
                    pattern: /^[0-9ABCDEFGHJKLMNPQRTUWXY]{18}$/,
                    message: '请输入正确的18位统一社会信用代码',
                  },
                ]}
                style={{ marginBottom: 0 }}
              >
                <Input
                  placeholder="91510108MAKB..."
                  allowClear
                  style={{ fontFamily: 'monospace' }}
                  addonAfter={
                    <Button
                      type="link"
                      size="small"
                      icon={<SafetyCertificateOutlined />}
                      onClick={onTianyanchaVerify}
                      style={{
                        padding: '0 4px',
                        height: 'auto',
                        fontWeight: 500,
                        color: '#1677ff',
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 4,
                      }}
                    >
                      校验公司信息
                    </Button>
                  }
                />
              </Form.Item>
            </Col>
          )}

          <Col xs={24} lg={isForeignAgent ? 6 : 5}>
            <ProFormText
              name="code"
              label="单位编码"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              placeholder={
                isForeignAgent
                  ? '选填，如 PAC-LAX'
                  : '选填，仅用于搜索，如 CDRT'
              }
              rules={[
                {
                  pattern: /^[A-Za-z0-9_-]+$/,
                  message: '仅支持字母数字',
                },
              ]}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>
        </Row>

        {/* Row 2: 英文地址 (国外代理主地址) / 中文地址 (国内客户/供应商) */}
        {!isForeignAgent ? (
          <>
            <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
              <Col span={24}>
                <Form.Item
                  label="中文地址"
                  labelCol={labelCol(LABEL_COL_WIDTH.primary)}
                  style={{ marginBottom: 0 }}
                >
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
                      <Input placeholder="请输入详细地址" allowClear />
                    </Form.Item>
                  </Space.Compact>
                </Form.Item>
              </Col>
            </Row>

            {/* Row 3: 英文名 */}
            <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
              <Col span={24}>
                <ProFormText
                  name="nameEn"
                  label="英文名"
                  labelCol={labelCol(LABEL_COL_WIDTH.primary)}
                  placeholder="请输入英文名称"
                  formItemProps={{ style: { marginBottom: 0 } }}
                />
              </Col>
            </Row>

            {/* Row 4: 英文地址 */}
            <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
              <Col span={24}>
                <ProFormText
                  name="addressEn"
                  label="英文地址"
                  labelCol={labelCol(LABEL_COL_WIDTH.primary)}
                  placeholder="请输入英文地址"
                  formItemProps={{ style: { marginBottom: 0 } }}
                />
              </Col>
            </Row>
          </>
        ) : (
          <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
            <Col span={24}>
              <ProFormText
                name="addressEn"
                label="英文地址"
                labelCol={labelCol(LABEL_COL_WIDTH.foreignPrimary)}
                placeholder="请输入境外公司注册/办公详细英文地址，如 Suite 200, 100 Main St, Los Angeles, CA 90001, USA"
                rules={[{ required: true, message: '请输入境外英文地址' }]}
                formItemProps={{ style: { marginBottom: 0 } }}
              />
            </Col>
          </Row>
        )}

        {/* Row 5: 公司别名 (移入基础信息，改用 Select[mode="tags"]) */}
        <Row gutter={[16, 12]} style={{ marginTop: 12 }}>
          <Col span={24}>
            <Form.Item
              label="公司别名"
              labelCol={labelCol(
                isForeignAgent
                  ? LABEL_COL_WIDTH.foreignPrimary
                  : LABEL_COL_WIDTH.primary,
              )}
              style={{ marginBottom: 0 }}
            >
              <Select
                mode="tags"
                value={aliases}
                onChange={onAliasesChange}
                placeholder="输入企业别名后按回车添加，支持添加多个别名"
                tokenSeparators={[',', '，']}
                style={{ width: '100%' }}
                open={false}
              />
            </Form.Item>
          </Col>
        </Row>

        {/* Row 6: 属性栏 (散客标识, 类型, 开发方式, 业务类型)
            采用标准 4 列 Col (span=6) 栅格，统一 Label 宽度；
            “类型”采用 Radio.Group（直客 / 同行），单选互斥语义更清晰 */}
        <Row gutter={[16, 12]} align="middle" style={{ marginTop: 12 }}>
          {isCustomer && (
            <Col xs={24} sm={12} md={6}>
              <ProFormSwitch
                name="isCasual"
                label="散客标识"
                labelCol={labelCol(LABEL_COL_WIDTH.attribute)}
                checkedChildren="散客"
                unCheckedChildren="正式"
                fieldProps={{
                  'aria-label': '单次合作 (散客)',
                }}
                formItemProps={{ style: { marginBottom: 0 } }}
              />
            </Col>
          )}

          {isCustomer && (
            <Col xs={24} sm={12} md={6}>
              <ProFormRadio.Group
                name="customerType"
                label="类型"
                labelCol={labelCol(LABEL_COL_WIDTH.attribute)}
                options={CUSTOMER_TYPE_OPTIONS}
                initialValue={PartnerCustomerType.PARTNER_CUSTOMER_TYPE_DIRECT}
                formItemProps={{ style: { marginBottom: 0 } }}
              />
            </Col>
          )}

          <Col xs={24} sm={12} md={isCustomer ? 6 : 12}>
            <ProFormSelect
              name="developmentMethod"
              label="开发方式"
              labelCol={labelCol(LABEL_COL_WIDTH.attribute)}
              options={DEVELOPMENT_METHOD_OPTIONS}
              initialValue="自主开发"
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          <Col xs={24} sm={12} md={isCustomer ? 6 : 12}>
            <ProFormSelect
              name="businessTypes"
              label="业务类型"
              labelCol={labelCol(LABEL_COL_WIDTH.attribute)}
              mode="multiple"
              options={BUSINESS_TYPE_OPTIONS}
              placeholder="请选择适用的业务类型"
              initialValue={[1]}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>
        </Row>

        <Divider style={{ margin: '14px 0' }} />

        {/* 责任人员分配矩阵 (重点降噪：移除创建人员输入项至标题侧元数据展示，去除双列占位下拉框) */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 10,
          }}
        >
          <Text strong style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
            责任人员分配矩阵
          </Text>
          {creatorMeta ? (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {creatorMeta}
            </Text>
          ) : null}
        </div>

        <Row gutter={[20, 10]}>
          {/* Slot 1: 操作人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="操作人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignOperatorUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange(
                        'assignOperatorUser',
                        'assignOperatorOrg',
                        val,
                      )
                    }
                  />
                </Form.Item>
                <Form.Item name="assignOperatorOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 2: 业务人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="业务人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignSalesUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignSalesUser', 'assignSalesOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignSalesOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 3: 客服人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="客服人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignServiceUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignServiceUser', 'assignServiceOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignServiceOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 4: 关联人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="关联人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignContactUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignContactUser', 'assignContactOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignContactOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 5: 财务人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="财务人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignFinanceUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignFinanceUser', 'assignFinanceOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignFinanceOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 6: 单证人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="单证人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignDocUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignDocUser', 'assignDocOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignDocOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 7: 商务人员 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="商务人员"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignCommercialUser" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange(
                        'assignCommercialUser',
                        'assignCommercialOrg',
                        val,
                      )
                    }
                  />
                </Form.Item>
                <Form.Item name="assignCommercialOrg" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 8: 关联人员2 */}
          <Col xs={24} md={12}>
            <Form.Item
              label="关联人员2"
              labelCol={labelCol(LABEL_COL_WIDTH.primary)}
              style={{ marginBottom: 0 }}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignContact2User" noStyle>
                  <Select
                    showSearch
                    placeholder="选择人员"
                    prefix={<UserOutlined style={{ color: '#8c8c8c' }} />}
                    options={userSelectOptions}
                    style={{ width: '52%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange(
                        'assignContact2User',
                        'assignContact2Org',
                        val,
                      )
                    }
                  />
                </Form.Item>
                <Form.Item name="assignContact2Org" noStyle>
                  <Select
                    showSearch
                    placeholder="归属组织/公司"
                    prefix={<ApartmentOutlined style={{ color: '#8c8c8c' }} />}
                    options={orgSelectOptions}
                    style={{ width: '48%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>
        </Row>
      </div>
    </SectionCard>
  );
}
