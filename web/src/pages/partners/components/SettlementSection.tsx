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

// 同列标签统一等宽（右对齐），保证多行控件的左缘上下成列对齐。
// 各列宽度集中在此配置：调整某列宽度只需改对应键值。
const LABEL_COL_WIDTH = {
  /** 第一列：对账方式/默认账期（取两行最长标签） */
  statement: 132,
  /** 第二列：结算方式/结算币种 */
  method: 88,
  /** 第三列：结算日期/利息规则 */
  date: 88,
  /** 账期列 */
  terms: 56,
  /** 信用额度列 */
  credit: 118,
} as const;

const labelCol = (width: number) => ({ style: { width } });

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
        <Row gutter={[16, 12]} align="middle">
          {/* 对账方式 */}
          <Col xs={24} sm={12} md={4}>
            <ProFormSelect
              name="statementMode"
              label="对账方式"
              labelCol={labelCol(LABEL_COL_WIDTH.statement)}
              options={STATEMENT_MODE_OPTIONS}
              rules={[{ required: true, message: '请选择对账方式' }]}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {/* 结算方式 */}
          <Col xs={24} sm={12} md={4}>
            <ProFormSelect
              name="settlementMethod"
              label="结算方式"
              labelCol={labelCol(LABEL_COL_WIDTH.method)}
              options={SETTLEMENT_METHOD_OPTIONS}
              rules={[{ required: true, message: '请选择结算方式' }]}
              formItemProps={{ style: { marginBottom: 0 } }}
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
              labelCol={labelCol(LABEL_COL_WIDTH.date)}
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
              labelCol={labelCol(LABEL_COL_WIDTH.terms)}
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

          {/* 信用额度(本币) - 仅客户适用（应收信用敞口管控） */}
          {!isSupplier && (
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
                labelCol={labelCol(LABEL_COL_WIDTH.credit)}
                placeholder="输入信用额度"
                min={0}
                formItemProps={{ style: { marginBottom: 0 } }}
                fieldProps={{
                  precision: 2,
                  addonAfter: '元',
                }}
              />
            </Col>
          )}
        </Row>

        <Row gutter={[16, 12]} align="middle" style={{ marginTop: 12 }}>
          {/* 默认账期天数 */}
          <Col xs={24} sm={12} md={isSupplier ? 6 : 4}>
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
              labelCol={labelCol(LABEL_COL_WIDTH.statement)}
              placeholder="例如: 30"
              min={0}
              max={3650}
              formItemProps={{ style: { marginBottom: 0 } }}
              fieldProps={{ precision: 0 }}
            />
          </Col>

          {/* 结算币种 */}
          <Col xs={24} sm={12} md={isSupplier ? 6 : 4}>
            <ProFormSelect
              name="settlementCurrency"
              label="结算币种"
              labelCol={labelCol(LABEL_COL_WIDTH.method)}
              options={currencyOptions}
              rules={[{ required: true, message: '请选择结算币种' }]}
              formItemProps={{ style: { marginBottom: 0 } }}
            />
          </Col>

          {/* 利息规则 - 仅客户应收适用 */}
          {!isSupplier && (
            <Col xs={24} sm={12} md={6}>
              <Form.Item
                label="利息规则"
                labelCol={labelCol(LABEL_COL_WIDTH.date)}
                style={{ marginBottom: 0 }}
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
              </Form.Item>
            </Col>
          )}
        </Row>
      </div>
    </SectionCard>
  );
}
