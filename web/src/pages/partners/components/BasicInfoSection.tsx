import {
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import {
  ProFormCheckbox,
  ProFormSelect,
  ProFormText,
} from '@ant-design/pro-components';
import { SectionCard } from '@/components/ui';
import { pcaCascaderOptions } from '@/utils/chinaDivision';
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
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React from 'react';

const { Text } = Typography;

export const BUSINESS_TYPE_OPTIONS = [
  { label: 'SE（海运出口）', value: 1 },
  { label: 'SI（海运进口）', value: 2 },
  { label: 'AE（空运出口）', value: 3 },
  { label: 'AI（空运进口）', value: 4 },
  { label: 'LAND（陆运业务）', value: 5 },
  { label: 'RAIL（铁路运输）', value: 6 },
];

export const CUSTOMER_TYPE_OPTIONS = [
  { label: '直客', value: 1 },
  { label: '同行', value: 2 },
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
  partnerId?: string;
  roleLabel: string;
  userSelectOptions: { label: string; value: string }[];
  orgSelectOptions: { label: string; value: string }[];
  aliases: string[];
  newAliasInput: string;
  setNewAliasInput: (val: string) => void;
  onAddAlias: () => void;
  onRemoveAlias: (alias: string) => void;
  onTianyanchaVerify: () => void;
  onUserChange: (userField: string, orgField: string, userId?: string) => void;
};

export default function BasicInfoSection({
  collapsed,
  onCollapseChange,
  partnerId,
  roleLabel,
  userSelectOptions,
  orgSelectOptions,
  aliases,
  newAliasInput,
  setNewAliasInput,
  onAddAlias,
  onRemoveAlias,
  onTianyanchaVerify,
  onUserChange,
}: BasicInfoSectionProps) {
  return (
    <SectionCard
      title="基础信息"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
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
                  onClick={onTianyanchaVerify}
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
                  <Input placeholder="请输入详细地址" allowClear />
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

        {/* Row 6: 9 Personnel Assignment Slots */}
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
                    onChange={(val) =>
                      onUserChange('assignCreatorUser', 'assignCreatorOrg', val)
                    }
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
                    onChange={(val) =>
                      onUserChange('assignSalesUser', 'assignSalesOrg', val)
                    }
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
                    onChange={(val) =>
                      onUserChange('assignServiceUser', 'assignServiceOrg', val)
                    }
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
                    onChange={(val) =>
                      onUserChange('assignContactUser', 'assignContactOrg', val)
                    }
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

          {/* Slot 6: 财务人员 */}
          <Col xs={24} md={12}>
            <Form.Item label="财务人员" style={{ marginBottom: 0 }}>
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item name="assignFinanceUser" noStyle>
                  <Select
                    showSearch
                    placeholder="请选择"
                    options={userSelectOptions}
                    style={{ width: '50%' }}
                    allowClear
                    onChange={(val) =>
                      onUserChange('assignFinanceUser', 'assignFinanceOrg', val)
                    }
                  />
                </Form.Item>
                <Form.Item name="assignFinanceOrg" noStyle>
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

          {/* Slot 7: 单证人员 */}
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
                    onChange={(val) =>
                      onUserChange('assignDocUser', 'assignDocOrg', val)
                    }
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

          {/* Slot 8: 商务人员 */}
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
                    placeholder="请选择"
                    options={orgSelectOptions}
                    style={{ width: '50%' }}
                    allowClear
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>

          {/* Slot 9: 关联人员2 */}
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
                  onPressEnter={onAddAlias}
                  style={{ width: 220 }}
                />
                <Button type="dashed" onClick={onAddAlias}>
                  添加
                </Button>
                {aliases.map((alias) => (
                  <Tag
                    key={alias}
                    closable
                    onClose={() => onRemoveAlias(alias)}
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
    </SectionCard>
  );
}
