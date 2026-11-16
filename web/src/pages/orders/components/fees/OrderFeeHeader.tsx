import { Card, Col, Row, Space, Tag } from 'antd';
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
import styles from './OrderFeeHeader.module.css';

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
      {/* 1. 基础信息卡片 */}
      <SectionCard title="订单基础信息" style={{ marginBottom: 16 }}>
        <div className={styles.identity}>
          <div className={styles.orderIdentity}>
            <span className={styles.label}>订单编号</span>
            <strong className={styles.orderNo}>
              {order.orderNo || order.id}
            </strong>
            <Tag color="blue">{configTitle}</Tag>
            <span className={styles.tradeTerm}>
              <span className={styles.label}>贸易条款</span>
              {tradeTermOptions.find((o) => o.value === order.tradeTerm)
                ?.label ?? '-'}
            </span>
          </div>
          <div className={styles.customer}>
            <span className={styles.label}>委托单位</span>
            <strong>{customerName || order.customerName || '-'}</strong>
          </div>
        </div>
        <div className={styles.transport}>
          <div className={styles.route}>
            <div className={styles.location}>
              <span className={styles.label}>
                {isAir ? '起运机场' : '起运港 (POL)'}
              </span>
              <strong>
                {(isAir
                  ? master?.originAirportName
                  : order.originLocationName) || '-'}
              </strong>
            </div>
            <div className={styles.routeConnector} aria-hidden="true">
              <span />
              <span className={styles.routeArrow}>→</span>
              <span />
            </div>
            <div className={styles.location}>
              <span className={styles.label}>
                {isAir ? '目的机场' : '目的港 (POD)'}
              </span>
              <strong>
                {(isAir
                  ? master?.destinationAirportName
                  : order.destinationLocationName) || '-'}
              </strong>
            </div>
          </div>
          <div className={styles.schedule}>
            <span className={styles.label}>
              {isImport
                ? isAir
                  ? '计划到达（经营日期）'
                  : 'ETA（经营日期）'
                : isAir
                  ? '计划起飞'
                  : 'ETD'}
            </span>
            <strong>
              {formatDate(
                isImport
                  ? isAir
                    ? master?.content?.eta
                    : order.eta
                  : isAir
                    ? master?.content?.etd
                    : order.etd,
                isAir ? 'minute' : 'date',
              )}
            </strong>
          </div>
        </div>
        <dl className={styles.details}>
          <div>
            <dt>{isAir ? '主运单号 (MAWB)' : '主单号 (MBL)'}</dt>
            <dd className={styles.documentNo}>
              {(isAir
                ? master?.content?.masterNo
                : order.seaMasterBill?.masterNo) || '-'}
            </dd>
          </div>
          <div>
            <dt>{isAir ? '航空公司' : '船公司'}</dt>
            <dd>
              {(isAir
                ? master?.airlineName
                : order.seaMasterBill?.shippingLineName) || '-'}
            </dd>
          </div>
          <div>
            <dt>{isAir ? '航班号' : '船名航次'}</dt>
            <dd>
              {(isAir ? master?.content?.flightNo : order.vesselVoyage) || '-'}
            </dd>
          </div>
          <div>
            <dt>订舱代理</dt>
            <dd>{order.bookingAgentName || '-'}</dd>
          </div>
          <div>
            <dt>件重尺</dt>
            <dd>
              {order.totalPackages || '-'} 件 /{' '}
              {order.totalGrossWeightKg || '-'} kg /{' '}
              {order.totalVolumeCbm || '-'} m³
            </dd>
          </div>
        </dl>
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
