import { Divider, Space, Typography } from 'antd';
import React, { useMemo } from 'react';
import type {
  FinanceLedgerGlobalSummary,
  FinanceLedgerSummaryItem,
} from './types';

const { Text } = Typography;

export interface FinanceSummaryBoardProps {
  selectedRows?: FinanceLedgerSummaryItem[];
  allRows?: FinanceLedgerSummaryItem[];
  totalCount?: number;
  globalSummary?: FinanceLedgerGlobalSummary;
}

interface AggregatedMetrics {
  receivableByCurrency: Record<string, number>;
  receivableBaseTotal: number;
  payableByCurrency: Record<string, number>;
  payableBaseTotal: number;
  profitByCurrency: Record<string, number>;
  profitBaseTotal: number;
  baseCurrency: string;
}

function formatNumber(num: number): string {
  return num.toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function aggregateFees(
  items: FinanceLedgerSummaryItem[],
  fallbackBaseCurrency = 'CNY',
): AggregatedMetrics {
  const receivableByCurrency: Record<string, number> = {};
  const payableByCurrency: Record<string, number> = {};
  const allCurrencies = new Set<string>();
  let receivableBaseTotal = 0;
  let payableBaseTotal = 0;
  let baseCurrency = fallbackBaseCurrency;

  for (const item of items) {
    if (item.status === 'CANCELLED' || item.status === 4 || item.status === '4')
      continue;
    const cur = item.currency || 'CNY';
    const total = Number(item.totalAmount || 0);
    const baseTotal = Number(item.baseCurrencyAmount || 0);
    if (item.baseCurrency) {
      baseCurrency = item.baseCurrency;
    }

    allCurrencies.add(cur);

    const isRec =
      item.direction === 'RECEIVABLE' ||
      item.direction === 1 ||
      item.direction === '1';
    const isPay =
      item.direction === 'PAYABLE' ||
      item.direction === 2 ||
      item.direction === '2';

    if (isRec) {
      receivableByCurrency[cur] = (receivableByCurrency[cur] || 0) + total;
      receivableBaseTotal += baseTotal;
    } else if (isPay) {
      payableByCurrency[cur] = (payableByCurrency[cur] || 0) + total;
      payableBaseTotal += baseTotal;
    }
  }

  const profitByCurrency: Record<string, number> = {};
  for (const cur of allCurrencies) {
    const rec = receivableByCurrency[cur] || 0;
    const pay = payableByCurrency[cur] || 0;
    const diff = rec - pay;
    if (diff !== 0 || rec !== 0 || pay !== 0) {
      profitByCurrency[cur] = diff;
    }
  }

  const profitBaseTotal = receivableBaseTotal - payableBaseTotal;

  return {
    receivableByCurrency,
    receivableBaseTotal,
    payableByCurrency,
    payableBaseTotal,
    profitByCurrency,
    profitBaseTotal,
    baseCurrency,
  };
}

export function FinanceSummaryBoard({
  selectedRows = [],
  allRows = [],
  totalCount,
  globalSummary,
}: FinanceSummaryBoardProps) {
  const selectedMetrics = useMemo(
    () => aggregateFees(selectedRows, globalSummary?.baseCurrency || 'CNY'),
    [selectedRows, globalSummary],
  );

  const allMetrics = useMemo(() => {
    const metrics = aggregateFees(
      allRows,
      globalSummary?.baseCurrency || 'CNY',
    );
    if (globalSummary?.receivableBaseAmount != null) {
      metrics.receivableBaseTotal = Number(
        globalSummary.receivableBaseAmount || 0,
      );
    }
    if (globalSummary?.payableBaseAmount != null) {
      metrics.payableBaseTotal = Number(globalSummary.payableBaseAmount || 0);
    }
    if (globalSummary?.profitBaseAmount != null) {
      metrics.profitBaseTotal = Number(globalSummary.profitBaseAmount || 0);
    }
    if (globalSummary?.baseCurrency) {
      metrics.baseCurrency = globalSummary.baseCurrency;
    }
    return metrics;
  }, [allRows, globalSummary]);

  const renderCurrencyList = (
    map: Record<string, number>,
    isProfit = false,
  ) => {
    const entries = Object.entries(map).filter(([_, val]) => val !== 0);
    if (entries.length === 0) {
      return (
        <span style={{ fontWeight: 600, color: '#8c8c8c' }}>0.00</span>
      );
    }

    return (
      <Space size={2} wrap>
        {entries.map(([curr, val], idx) => {
          let color = '#262626';
          if (isProfit) {
            color = val >= 0 ? '#52c41a' : '#ff4d4f';
          }
          return (
            <span key={curr} style={{ whiteSpace: 'nowrap' }}>
              {idx > 0 && (
                <span style={{ color: '#bfbfbf', margin: '0 3px' }}>+</span>
              )}
              <strong style={{ color }}>
                {isProfit && val > 0 ? '+' : ''}
                {formatNumber(val)}
              </strong>{' '}
              <span style={{ fontSize: 11, color: '#595959' }}>{curr}</span>
            </span>
          );
        })}
      </Space>
    );
  };

  const renderBaseAmount = (
    val: number,
    baseCurr: string,
    color: string,
    isProfit = false,
  ) => {
    return (
      <span style={{ marginLeft: 6, whiteSpace: 'nowrap' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          折合{' '}
        </Text>
        <strong style={{ color, fontSize: 13 }}>
          {isProfit && val > 0 ? '+' : ''}
          {formatNumber(val)}
        </strong>{' '}
        <span style={{ fontSize: 11, color: '#8c8c8c' }}>{baseCurr}</span>
      </span>
    );
  };

  return (
    <div
      style={{
        position: 'sticky',
        bottom: 0,
        zIndex: 9,
        background: '#ffffff',
        borderTop: '1px solid #e8e8e8',
        boxShadow: '0 -3px 12px rgba(0, 0, 0, 0.06)',
        padding: '8px 16px',
        fontSize: 12,
        lineHeight: 1.6,
      }}
    >
      {/* 1. 总计(选中) */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          marginBottom: 6,
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <div style={{ fontWeight: 700, color: '#1f1f1f', minWidth: 86 }}>
          总计(选中):{' '}
          <span style={{ color: selectedRows.length > 0 ? '#1677ff' : '#8c8c8c' }}>
            {selectedRows.length} 条
          </span>
        </div>
        {selectedRows.length === 0 ? (
          <span style={{ color: '#8c8c8c' }}>（未勾选费用）</span>
        ) : (
          <Space size={12} wrap style={{ flex: 1 }}>
            <div>
              <span style={{ color: '#666666' }}>应收(含税): </span>
              {renderCurrencyList(selectedMetrics.receivableByCurrency)}
              {renderBaseAmount(
                selectedMetrics.receivableBaseTotal,
                selectedMetrics.baseCurrency,
                '#1677ff',
              )}
            </div>
            <Divider vertical style={{ margin: '0 4px', height: 13 }} />
            <div>
              <span style={{ color: '#666666' }}>应付(含税): </span>
              {renderCurrencyList(selectedMetrics.payableByCurrency)}
              {renderBaseAmount(
                selectedMetrics.payableBaseTotal,
                selectedMetrics.baseCurrency,
                '#fa8c16',
              )}
            </div>
            <Divider vertical style={{ margin: '0 4px', height: 13 }} />
            <div>
              <span style={{ color: '#666666' }}>利润: </span>
              {renderCurrencyList(selectedMetrics.profitByCurrency, true)}
              {renderBaseAmount(
                selectedMetrics.profitBaseTotal,
                selectedMetrics.baseCurrency,
                selectedMetrics.profitBaseTotal >= 0 ? '#52c41a' : '#ff4d4f',
                true,
              )}
            </div>
          </Space>
        )}
      </div>

      {/* 2. 总计(全部) */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          gap: 12,
          flexWrap: 'wrap',
          paddingTop: 6,
          borderTop: '1px dashed #f0f0f0',
        }}
      >
        <div style={{ fontWeight: 700, color: '#1f1f1f', minWidth: 86 }}>
          总计(全部):{' '}
          <span style={{ color: '#595959' }}>
            {totalCount ?? (globalSummary?.activeCount || allRows.length)} 条
          </span>
        </div>
        <Space size={12} wrap style={{ flex: 1 }}>
          <div>
            <span style={{ color: '#666666' }}>应收(含税): </span>
            {renderCurrencyList(allMetrics.receivableByCurrency)}
            {renderBaseAmount(
              allMetrics.receivableBaseTotal,
              allMetrics.baseCurrency,
              '#1677ff',
            )}
          </div>
          <Divider vertical style={{ margin: '0 4px', height: 13 }} />
          <div>
            <span style={{ color: '#666666' }}>应付(含税): </span>
            {renderCurrencyList(allMetrics.payableByCurrency)}
            {renderBaseAmount(
              allMetrics.payableBaseTotal,
              allMetrics.baseCurrency,
              '#fa8c16',
            )}
          </div>
          <Divider vertical style={{ margin: '0 4px', height: 13 }} />
          <div>
            <span style={{ color: '#666666' }}>利润: </span>
            {renderCurrencyList(allMetrics.profitByCurrency, true)}
            {renderBaseAmount(
              allMetrics.profitBaseTotal,
              allMetrics.baseCurrency,
              allMetrics.profitBaseTotal >= 0 ? '#52c41a' : '#ff4d4f',
              true,
            )}
          </div>
        </Space>
      </div>
    </div>
  );
}
