import { Typography } from 'antd';
import React from 'react';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';

function feeReviewProfitRate(summary?: API.OrderFeeReviewSummary) {
  return summary?.profitRate == null || summary.profitRate === ''
    ? '不可计算'
    : `${trimDecimal(summary.profitRate)}%`;
}

/** 列表悬停只展示三行金额；完整审核使用横向统计，保持整票口径。 */
export function FeeReviewSummary({
  summary,
  compact = false,
}: {
  summary?: API.OrderFeeReviewSummary;
  compact?: boolean;
}) {
  if (!summary)
    return (
      <Typography.Text type="secondary">暂无可查看的费用摘要</Typography.Text>
    );
  const loss = Number(summary.profit) < 0;
  const rows = [
    {
      label: '应收',
      count: summary.receivableCount ?? 0,
      amount: summary.receivable,
      color: '#1677ff',
    },
    {
      label: '应付',
      count: summary.payableCount ?? 0,
      amount: summary.payable,
      color: '#d46b08',
    },
    {
      label: loss ? '利润 · 亏损' : '利润',
      amount: summary.profit,
      color: loss ? '#cf1322' : '#237804',
    },
  ];
  if (compact)
    return (
      <table
        aria-label="整票费用摘要"
        style={{ width: 290, borderCollapse: 'collapse', fontSize: 12 }}
      >
        <tbody>
          {rows.map((row) => (
            <tr key={row.label}>
              <th
                style={{
                  textAlign: 'left',
                  fontWeight: 500,
                  padding: '5px 10px 5px 0',
                }}
              >
                {row.label}
              </th>
              <td
                style={{
                  textAlign: 'right',
                  padding: '5px 0',
                  fontWeight: 600,
                  color: row.color,
                  fontVariantNumeric: 'tabular-nums',
                }}
              >
                {formatAmount(row.amount)}{' '}
                <span style={{ fontWeight: 400, color: '#8c8c8c' }}>
                  {summary.baseCurrency}
                </span>
              </td>
              <td
                style={{
                  textAlign: 'right',
                  paddingLeft: 12,
                  color: '#8c8c8c',
                  whiteSpace: 'nowrap',
                }}
              >
                {row.count == null
                  ? feeReviewProfitRate(summary)
                  : `${row.count}笔`}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  return (
    <div
      style={{
        border: '1px solid #f0f0f0',
        borderRadius: 6,
        background: '#fff',
      }}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
        }}
      >
        {rows.map((row, index) => (
          <div
            key={row.label}
            style={{
              padding: '10px 14px',
              borderRight: index < 2 ? '1px solid #f0f0f0' : undefined,
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                gap: 6,
                color: '#8c8c8c',
                fontSize: 12,
              }}
            >
              <span>{row.label}</span>
              <span>
                {row.count == null
                  ? feeReviewProfitRate(summary)
                  : `${row.count}笔`}
              </span>
            </div>
            <div
              style={{
                fontSize: 21,
                fontWeight: 600,
                color: row.color,
                fontVariantNumeric: 'tabular-nums',
                lineHeight: '30px',
              }}
            >
              {formatAmount(row.amount)}{' '}
              <span style={{ fontSize: 12, fontWeight: 400, color: '#8c8c8c' }}>
                {summary.baseCurrency}
              </span>
            </div>
          </div>
        ))}
      </div>
      <div
        style={{
          borderTop: '1px solid #f0f0f0',
          padding: '5px 14px',
          display: 'flex',
          flexWrap: 'wrap',
          gap: '4px 24px',
          color: '#8c8c8c',
          fontSize: 12,
        }}
      >
        <span>
          折算币种{' '}
          <Typography.Text>{summary.baseCurrency || '-'}</Typography.Text>
        </span>
        <span>
          提交人{' '}
          <Typography.Text>{summary.submittedByName || '-'}</Typography.Text>
        </span>
        <span>
          提交时间{' '}
          <Typography.Text>
            {formatDate(summary.submittedAt, 'minute')}
          </Typography.Text>
        </span>
        <span>整票应收、应付统一折算</span>
      </div>
    </div>
  );
}
