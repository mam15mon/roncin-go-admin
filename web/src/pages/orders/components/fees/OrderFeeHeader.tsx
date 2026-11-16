import { Card, Col, Descriptions, Row, Space, Tag } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { formatAmount, formatDate } from '@/utils/format';
import { tradeTermOptions } from '../../common';
import type { OrderBusinessWritePolicy } from '../../use-order-lock-state';
import {
  computeFeeProfit,
  type FeeCurrencyTotal,
  type FeeSideSummary,
} from './feeSummary';

type OrderFeeHeaderProps = {
  order: API.Order;
  kind: string;
  orderId: string;
  configTitle: string;
  customerName?: string;
  lockWritePolicy: OrderBusinessWritePolicy;
  onRetryLockState: () => Promise<API.OrderLockStateData | null>;
  /** 已落库费用汇总；undefined 表示尚未成功取得数据。 */
  receivableSummary?: FeeSideSummary;
  payableSummary?: FeeSideSummary;
};

/** 有费用但全部未折算时没有可展示的本币合计，显示 '-' 而非 0.00。 */
function shown(summary?: FeeSideSummary) {
  if (!summary) return undefined;
  return summary.count > 0 && summary.unconvertedCount === summary.count
    ? undefined
    : summary;
}

function SummaryCard({
  title,
  color,
  gross,
  net,
  unconvertedCount,
  currencies,
  extra,
  testId,
}: {
  title: string;
  color: string;
  gross?: string;
  net?: string;
  unconvertedCount: number;
  currencies?: FeeCurrencyTotal[];
  extra?: React.ReactNode;
  testId: string;
}) {
  const row = (label: string, value?: string) => (
    <div
      style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginTop: 4 }}
    >
      <span style={{ color: '#64748b', width: 64 }}>{label}</span>
      <span style={{ color, fontWeight: 700, fontSize: 20 }}>
        {value === undefined ? '-' : `¥${formatAmount(value)}`}
      </span>
    </div>
  );
  return (
    <Card
      variant="borderless"
      style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
      data-testid={testId}
    >
      <Space style={{ color: '#64748b' }}>
        <span>{title}（本币）</span>
        {extra}
      </Space>
      {row('含税', gross)}
      {row('不含税', net)}
      {unconvertedCount > 0 ? (
        <div style={{ color: '#fa8c16', marginTop: 4 }}>
          {unconvertedCount} 笔未折算，未计入本币合计
        </div>
      ) : null}
      {currencies && currencies.length > 0 ? (
        <div style={{ color: '#64748b', marginTop: 4, fontSize: 12 }}>
          {currencies.map((item) => (
            <div key={item.currency}>
              {item.currency}：含税 {formatAmount(item.gross)} / 不含税{' '}
              {formatAmount(item.net)}
            </div>
          ))}
        </div>
      ) : null}
    </Card>
  );
}

export default function OrderFeeHeader({
  order,
  kind: _kind,
  orderId: _orderId,
  configTitle,
  customerName,
  lockWritePolicy: _lockWritePolicy,
  onRetryLockState: _onRetryLockState,
  receivableSummary,
  payableSummary,
}: OrderFeeHeaderProps) {
  const profit = computeFeeProfit(
    shown(receivableSummary) ? receivableSummary : undefined,
    shown(payableSummary) ? payableSummary : undefined,
  );
  const isAir =
    order.businessType === OrderBusinessType.BUSINESS_TYPE_AE ||
    order.businessType === OrderBusinessType.BUSINESS_TYPE_AI;
  const master = order.airExport?.master;
  const isImport =
    order.businessType === OrderBusinessType.BUSINESS_TYPE_SI ||
    order.businessType === OrderBusinessType.BUSINESS_TYPE_AI;
  return (
    <>
      {/* 1. 基础信息卡片：单号与业务类型并入标题栏，明细用标准 Descriptions 紧凑网格 */}
      <SectionCard
        title="订单基础信息"
        extra={
          <>
            <span
              style={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}
            >
              {order.orderNo || order.id}
            </span>
            <Tag color="blue" style={{ marginInlineEnd: 0 }}>
              {configTitle}
            </Tag>
          </>
        }
        style={{ marginBottom: 16 }}
      >
        <Descriptions
          size="small"
          colon={false}
          column={{ xs: 1, sm: 2, lg: 3, xl: 4 }}
          styles={{
            label: { minWidth: 96, color: '#64748b' },
            content: { color: '#1f2937', minWidth: 0 },
          }}
          items={[
            {
              key: 'customer',
              label: '委托单位',
              children: customerName || order.customerName || '-',
            },
            {
              key: 'masterNo',
              label: isAir ? '主运单号 (MAWB)' : '主单号 (MBL)',
              children: (
                <span style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {(isAir
                    ? master?.content?.masterNo
                    : order.seaMasterBill?.masterNo) || '-'}
                </span>
              ),
            },
            {
              key: 'houseNo',
              label: isAir ? '分运单号 (HAWB)' : '分单号 (HBL)',
              children: (
                <span style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {(isAir
                    ? order.airExport?.house?.houseNo
                    : order.seaDocumentSummary?.houseNo) || '-'}
                </span>
              ),
            },
            {
              key: 'carrier',
              label: isAir ? '航空公司' : '船公司',
              children:
                (isAir
                  ? master?.airlineName
                  : order.seaMasterBill?.shippingLineName) || '-',
            },
            {
              key: 'origin',
              label: isAir ? '起运机场' : '起运港 (POL)',
              children:
                (isAir
                  ? master?.originAirportName
                  : order.originLocationName) || '-',
            },
            {
              key: 'destination',
              label: isAir ? '目的机场' : '目的港 (POD)',
              children:
                (isAir
                  ? master?.destinationAirportName
                  : order.destinationLocationName) || '-',
            },
            {
              key: 'schedule',
              label: isImport
                ? isAir
                  ? '计划到达（经营日期）'
                  : 'ETA（经营日期）'
                : isAir
                  ? '计划起飞'
                  : 'ETD',
              children: formatDate(
                isImport
                  ? isAir
                    ? master?.content?.eta
                    : order.eta
                  : isAir
                    ? master?.content?.etd
                    : order.etd,
                isAir ? 'minute' : 'date',
              ),
            },
            {
              key: 'voyage',
              label: isAir ? '航班号' : '船名航次',
              children:
                (isAir ? master?.content?.flightNo : order.vesselVoyage) || '-',
            },
            {
              key: 'tradeTerm',
              label: '贸易条款',
              children:
                tradeTermOptions.find((o) => o.value === order.tradeTerm)
                  ?.label ?? '-',
            },
            {
              key: 'bookingAgent',
              label: '订舱代理',
              children: order.bookingAgentName || '-',
            },
            {
              key: 'cargo',
              label: '件重尺',
              children: `${order.totalPackages || '-'} 件 / ${
                order.totalGrossWeightKg || '-'
              } kg / ${order.totalVolumeCbm || '-'} m³`,
            },
            {
              key: 'sales',
              label: '业务员',
              children: order.salesName || '-',
            },
          ]}
        />
      </SectionCard>

      {/* 2. 费用汇总统计指标卡：含税与不含税同时展示 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <SummaryCard
            title="应收总计"
            color="#1677ff"
            gross={shown(receivableSummary)?.gross}
            net={shown(receivableSummary)?.net}
            unconvertedCount={receivableSummary?.unconvertedCount ?? 0}
            currencies={receivableSummary?.currencies}
            testId="fee-summary-receivable"
          />
        </Col>
        <Col xs={24} sm={8}>
          <SummaryCard
            title="应付总计"
            color="#fa8c16"
            gross={shown(payableSummary)?.gross}
            net={shown(payableSummary)?.net}
            unconvertedCount={payableSummary?.unconvertedCount ?? 0}
            currencies={payableSummary?.currencies}
            testId="fee-summary-payable"
          />
        </Col>
        <Col xs={24} sm={8}>
          <SummaryCard
            title="预计毛利"
            color={profit?.negative ? '#ff4d4f' : '#52c41a'}
            gross={profit?.gross}
            net={profit?.net}
            unconvertedCount={profit?.unconvertedCount ?? 0}
            extra={
              profit ? (
                <Tag color={profit.negative ? 'error' : 'success'}>
                  {profit.rate}%
                </Tag>
              ) : undefined
            }
            testId="fee-summary-profit"
          />
        </Col>
      </Row>
    </>
  );
}
