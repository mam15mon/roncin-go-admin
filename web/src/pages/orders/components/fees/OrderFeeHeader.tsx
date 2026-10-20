import { Card, Col, Descriptions, Row, Space, Statistic, Tag } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { formatDate } from '@/utils/format';
import { tradeTermOptions } from '../../common';
import type { OrderBusinessWritePolicy } from '../../use-order-lock-state';

type OrderFeeHeaderProps = {
  order: API.Order;
  kind: string;
  orderId: string;
  configTitle: string;
  customerName?: string;
  financeLocked: boolean;
  financeLockReason?: string;
  financeLockCommissionNos: string[];
  lockWritePolicy: OrderBusinessWritePolicy;
  onRetryLockState: () => Promise<API.OrderLockStateData | null>;
  receivableSummary: { totalAmount: number; count: number };
  payableSummary: { totalAmount: number; count: number };
  profitCny: number;
  profitRate: string;
};

export default function OrderFeeHeader({
  order,
  kind: _kind,
  orderId: _orderId,
  configTitle,
  customerName,
  financeLocked: _financeLocked,
  financeLockReason: _financeLockReason,
  financeLockCommissionNos: _financeLockCommissionNos,
  lockWritePolicy: _lockWritePolicy,
  onRetryLockState: _onRetryLockState,
  receivableSummary,
  payableSummary,
  profitCny,
  profitRate,
}: OrderFeeHeaderProps) {
  const isAir = order.businessType === OrderBusinessType.BUSINESS_TYPE_AE;
  const master = order.airExport?.master;
  return (
    <>
      {/* 1. 基础信息卡片 */}
      <SectionCard title="订单基础信息" style={{ marginBottom: 16 }}>
        <Descriptions
          size="small"
          column={{ xs: 1, sm: 2, md: 3, lg: 4, xl: 4 }}
        >
          <Descriptions.Item label="订单编号">
            <span
              style={{
                fontWeight: 600,
                fontFamily: 'monospace',
              }}
            >
              {order.orderNo || order.id}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="委托单位">
            {customerName || order.customerName || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="业务类型">{configTitle}</Descriptions.Item>
          <Descriptions.Item label="贸易条款">
            {tradeTermOptions.find((o) => o.value === order.tradeTerm)?.label ??
              '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '主运单号 (MAWB)' : '主单号 (MBL)'}>
            {(isAir
              ? master?.content?.masterNo
              : order.seaMasterBill?.masterNo) || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '航班号' : '船名航次'}>
            {(isAir ? master?.content?.flightNo : order.vesselVoyage) || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '起运机场' : '起运港 (POL)'}>
            {(isAir ? master?.originAirportName : order.originLocationName) ||
              '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '目的机场' : '目的港 (POD)'}>
            {(isAir
              ? master?.destinationAirportName
              : order.destinationLocationName) || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '航空公司' : '船公司'}>
            {(isAir
              ? master?.airlineName
              : order.seaMasterBill?.shippingLineName) || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="订舱代理">
            {order.bookingAgentName || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={isAir ? '计划起飞' : 'ETD'}>
            {formatDate(
              isAir ? master?.content?.etd : order.etd,
              isAir ? 'minute' : 'date',
            )}
          </Descriptions.Item>
          <Descriptions.Item label="件重尺">
            {order.totalPackages || '-'} 件 / {order.totalGrossWeightKg || '-'}{' '}
            kg / {order.totalVolumeCbm || '-'} m³
          </Descriptions.Item>
        </Descriptions>
      </SectionCard>

      {/* 2. 费用汇总统计指标卡 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={<span style={{ color: '#64748b' }}>应收总计</span>}
              value={receivableSummary.totalAmount}
              precision={2}
              prefix="¥"
              styles={{ content: { color: '#1677ff', fontWeight: 700 } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={<span style={{ color: '#64748b' }}>应付总计</span>}
              value={payableSummary.totalAmount}
              precision={2}
              prefix="¥"
              styles={{ content: { color: '#fa8c16', fontWeight: 700 } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={
                <Space>
                  <span style={{ color: '#64748b' }}>预计毛利</span>
                  <Tag color={profitCny >= 0 ? 'success' : 'error'}>
                    {profitRate}%
                  </Tag>
                </Space>
              }
              value={profitCny}
              precision={2}
              prefix="¥"
              styles={{
                content: {
                  color: profitCny >= 0 ? '#52c41a' : '#ff4d4f',
                  fontWeight: 700,
                },
              }}
            />
          </Card>
        </Col>
      </Row>
    </>
  );
}
