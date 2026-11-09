import { Divider, Space } from 'antd';
import type Decimal from 'decimal.js';
import { formatAmount } from '@/utils/format';
import { sumAmountsByCurrency } from './summary';
import type {
  FinanceLedgerMatchedSummary,
  FinanceLedgerSummaryConfig,
} from './types';

export interface FinanceSummaryBoardProps {
  summary?: FinanceLedgerMatchedSummary;
  selected?: boolean;
  config: Pick<FinanceLedgerSummaryConfig<unknown>, 'groups' | 'profit'>;
  /** 非分页的整单费用入口可明确其全量范围。 */
  scopeLabel?: string;
}

function AmountList({
  values,
  signed = false,
}: {
  values: Map<string, Decimal>;
  signed?: boolean;
}) {
  if (!values.size) return <strong>0.00</strong>;
  return (
    <>
      {[...values].map(([currency, amount], index) => (
        <span key={currency} style={{ whiteSpace: 'nowrap' }}>
          {index > 0 ? ' / ' : ''}
          <strong>
            {signed && amount.gt(0) ? '+' : ''}
            {formatAmount(amount.toFixed())}
          </strong>{' '}
          {currency}
        </span>
      ))}
    </>
  );
}

export function FinanceSummaryBoard({
  summary,
  selected = false,
  config,
  scopeLabel = '筛选结果共',
}: FinanceSummaryBoardProps) {
  const renderAmounts = (
    items: FinanceLedgerMatchedSummary['amounts'],
    signed = false,
    sign?: (item: FinanceLedgerMatchedSummary['amounts'][number]) => number,
  ) => {
    const original = sumAmountsByCurrency(items, false, sign);
    const base = sumAmountsByCurrency(items, true, sign);
    const missing = items.reduce(
      (count, item) => count + (item.unconvertedCount ?? 0),
      0,
    );
    return (
      <>
        <AmountList values={original} signed={signed} />
        {base.size > 0 && (
          <span style={{ marginLeft: 6 }}>
            折合 <AmountList values={base} signed={signed} />
          </span>
        )}
        {missing > 0 && (
          <span style={{ marginLeft: 6, color: '#ad6800' }}>
            （{missing} 条未折算）
          </span>
        )}
      </>
    );
  };
  return (
    <div
      role="status"
      aria-label="金额合计"
      style={{
        position: 'sticky',
        bottom: 0,
        zIndex: 9,
        background: '#fff',
        borderTop: '1px solid #e8e8e8',
        boxShadow: '0 -3px 12px rgba(0,0,0,0.06)',
        padding: '8px 16px',
        fontSize: 12,
      }}
    >
      {!summary ? (
        <span>汇总尚未加载</span>
      ) : (
        <Space size={12} wrap>
          <strong>
            {selected ? '已选' : scopeLabel} {summary.count} 条
          </strong>
          {config.groups.map(({ direction, label, color }) => (
            <span key={direction} style={{ color }}>
              <span>{label}: </span>
              {renderAmounts(
                summary.amounts.filter((item) => item.direction === direction),
              )}
            </span>
          ))}
          {config.profit && (
            <>
              <Divider vertical />
              <span>
                <span>{config.profit.label}: </span>
                {renderAmounts(
                  summary.amounts.filter(
                    (item) =>
                      item.direction === config.profit?.receivableDirection ||
                      item.direction === config.profit?.payableDirection,
                  ),
                  true,
                  (item) =>
                    item.direction === config.profit?.receivableDirection
                      ? 1
                      : -1,
                )}
              </span>
            </>
          )}
        </Space>
      )}
    </div>
  );
}
