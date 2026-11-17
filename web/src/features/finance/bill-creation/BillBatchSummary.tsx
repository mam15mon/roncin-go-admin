import { Flex, Typography } from 'antd';
import Decimal from 'decimal.js';
import React from 'react';
import { formatAmount } from '@/utils/format';

const { Text } = Typography;

type BillBatchSummaryProps = {
  groups: API.BillBatchPreviewGroup[];
  incompleteCount: number;
};

export default function BillBatchSummary({
  groups,
  incompleteCount,
}: BillBatchSummaryProps) {
  const totalsByCurrency = groups.reduce<Record<string, Decimal>>(
    (totals, group) => {
      const currency = group.currency || '未配置币种';
      totals[currency] = (totals[currency] || new Decimal(0)).plus(
        group.totalAmount || 0,
      );
      return totals;
    },
    {},
  );
  const incomplete = incompleteCount > 0;
  return (
    <Flex
      wrap
      align="center"
      gap={16}
      style={{
        marginBottom: 12,
        padding: '8px 12px',
        border: '1px solid #f0f0f0',
        borderRadius: 6,
        background: '#fff',
      }}
    >
      <Text>
        共 <Text strong>{groups.length}</Text> 张账单
      </Text>
      <Text type={incomplete ? undefined : 'secondary'}>
        {incomplete ? (
          <Text style={{ color: '#d46b08' }}>
            {incompleteCount} 张待补全资料
          </Text>
        ) : (
          '资料已全部补全'
        )}
      </Text>
      <Text type="secondary">金额合计：</Text>
      {Object.entries(totalsByCurrency).map(([currency, amount]) => (
        <Text
          key={currency}
          strong
          style={{ fontVariantNumeric: 'tabular-nums' }}
        >
          {formatAmount(amount.toFixed(2))} {currency}
        </Text>
      ))}
    </Flex>
  );
}
