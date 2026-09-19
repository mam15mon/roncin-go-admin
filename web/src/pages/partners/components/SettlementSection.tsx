import { QuestionCircleOutlined } from '@ant-design/icons';
import { ProFormDigit, ProFormSelect } from '@ant-design/pro-components';
import { Button, Col, Form, Input, Row, Select, Space, Tooltip } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import {
  PartnerSettlementBase,
  PartnerSettlementMethod,
  PartnerStatementMode,
} from '@/enums.generated';
import type { InterestRuleValues } from './InterestRuleModal';

export const STATEMENT_MODE_OPTIONS = [
  { label: '单票', value: PartnerStatementMode.PARTNER_STATEMENT_MODE_SINGLE },
  { label: '汇总', value: PartnerStatementMode.PARTNER_STATEMENT_MODE_MULTI },
];

export const SETTLEMENT_METHOD_OPTIONS = [
  {
    label: '票结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BY_TICKET,
  },
  {
    label: '月结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_MONTHLY,
  },
  {
    label: '周结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_WEEKLY,
  },
  {
    label: '半月结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY,
  },
  {
    label: '双月结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BI_MONTHLY,
  },
  {
    label: '季结',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_QUARTERLY,
  },
  {
    label: '45天',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_DAYS_45,
  },
  {
    label: '预付',
    value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_PREPAID,
  },
];

export const SETTLEMENT_BASE_OPTIONS = [
  {
    label: '开票后',
    value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_BILL_DATE,
  },
  {
    label: '出运后',
    value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_SAILING_DATE,
  },
  {
    label: '到港后',
    value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE,
  },
];

export const SETTLEMENT_DAY_OPTIONS = Array.from({ length: 31 }, (_, i) => ({
  label: `${i + 1}日`,
  value: i + 1,
}));

// 统一 3 列网格标签宽度（右对齐），保证各行输入框对齐线一致且无标签折行。
const UNIFORM_LABEL_WIDTH = 110;
const labelCol = { style: { width: UNIFORM_LABEL_WIDTH } };

type SettlementSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  currencyOptions: { label: string; value: string }[];
  interestRule: InterestRuleValues;
  onOpenInterestModal: () => void;
  roleLabel?: string;
};

export default function SettlementSection({
  collapsed,
  onCollapseChange,
  currencyOptions,
  interestRule,
  onOpenInterestModal,
  roleLabel,
}: SettlementSectionProps) {
  const isSupplier = roleLabel === '供应商';
  return (
    <SectionCard
      id="section-settlement"
      sectionKey="settlement"
      title="财务结算规则"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <div>
        {/* 标准 3 列网格 (span=8) 自上而下排列，垂直动线清晰 */}
        <Row gutter={[20, 12]} align="middle">
          {/* Row 1 - Col 1: 对账方式 */}
          <Col xs={24} sm={12} md={8}>
            <ProFormSelect
              name="statementMode"
              label="对账方式"
              labelCol={labelCol}
              options={STATEMENT_MODE_OPTIONS}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {/* Row 1 - Col 2: 结算方式 */}
          <Col xs={24} sm={12} md={8}>
            <ProFormSelect
              name="settlementMethod"
              label="结算方式"
              labelCol={labelCol}
              options={SETTLEMENT_METHOD_OPTIONS}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {/* Row 1 - Col 3: 结算币种 */}
          <Col xs={24} sm={12} md={8}>
            <ProFormSelect
              name="settlementCurrency"
              label="结算币种"
              labelCol={labelCol}
              options={currencyOptions}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {/* Row 2 - Col 1: 结算日期 */}
          <Col xs={24} sm={12} md={8}>
            <Form.Item
              label={
                <Space size={4}>
                  <span>结算日期</span>
                  <Tooltip title="每月固定结算与对账截止日">
                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              labelCol={labelCol}
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
                    whiteSpace: 'nowrap',
                    flexShrink: 0,
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

          {/* Row 2 - Col 2: 账期 */}
          <Col xs={24} sm={12} md={8}>
            <Form.Item
              label={
                <Space size={4}>
                  <span>账期</span>
                  <Tooltip title="账期基准与有效信用天数">
                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              labelCol={labelCol}
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

          {/* Row 2 - Col 3: 默认账期天数 */}
          <Col xs={24} sm={12} md={8}>
            <ProFormDigit
              name="paymentTermsDays"
              label={
                <Space size={4}>
                  <span>默认账期(天)</span>
                  <Tooltip title="账单创建时按该天数默认带出账期（账单日 + N 天），可调整；留空表示未配置">
                    <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              labelCol={labelCol}
              placeholder="例如: 30"
              min={0}
              max={3650}
              formItemProps={{ style: { marginBottom: 0 } }}
              fieldProps={{ precision: 0 }}
            />
          </Col>

          {/* Row 3 - Col 1: 信用额度(本币) - 仅客户适用（应收信用敞口管控） */}
          {!isSupplier && (
            <Col xs={24} sm={12} md={8}>
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
                labelCol={labelCol}
                placeholder="输入信用额度"
                min={0}
                formItemProps={{ style: { marginBottom: 0 } }}
                addonAfter="元"
                fieldProps={{
                  precision: 2,
                }}
              />
            </Col>
          )}

          {/* Row 3 - Col 2: 利息规则 - 仅客户应收适用 */}
          {!isSupplier && (
            <Col xs={24} sm={12} md={8}>
              <Form.Item
                label="利息规则"
                labelCol={labelCol}
                style={{ marginBottom: 0 }}
              >
                <div
                  style={{
                    height: 32,
                    display: 'flex',
                    alignItems: 'center',
                  }}
                >
                  <Button
                    type="link"
                    onClick={onOpenInterestModal}
                    style={{ padding: 0, fontWeight: 500 }}
                  >
                    {interestRule.enabled
                      ? `已启用 (万分之${interestRule.dailyRateBp || 5}/日)`
                      : '编辑规则'}
                  </Button>
                </div>
              </Form.Item>
            </Col>
          )}
        </Row>
      </div>
    </SectionCard>
  );
}
