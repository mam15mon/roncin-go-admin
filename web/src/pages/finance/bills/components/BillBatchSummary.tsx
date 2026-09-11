import { Alert, Card, Col, Row, Statistic, Typography } from 'antd';
import Decimal from 'decimal.js';
import React from 'react';

const { Text } = Typography;

type BillBatchSummaryProps = {
  groups: API.BillBatchPreviewGroup[];
  currentGroup?: API.BillBatchPreviewGroup;
  incompleteCount: number;
};

export default function BillBatchSummary({
  groups,
  currentGroup,
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
  return (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Row gutter={[16, 12]}>
        <Col xs={12} md={6}>
          <Statistic title="叶子总数" value={groups.length} suffix="张" />
        </Col>
        <Col xs={12} md={6}>
          <Statistic
            title="已完成配置"
            value={groups.length - incompleteCount}
            suffix="张"
          />
        </Col>
        <Col xs={12} md={6}>
          <Statistic title="仍缺配置" value={incompleteCount} suffix="张" />
        </Col>
        <Col xs={12} md={6}>
          <Statistic
            title="当前叶子账单金额"
            value={currentGroup?.totalAmount || '-'}
            suffix={currentGroup?.currency || ''}
          />
        </Col>
      </Row>
      <Alert
        showIcon
        type="info"
        style={{ marginTop: 12 }}
        title="批次账单金额（按固定账单币种分列）"
        description={
          <Text>
            {Object.entries(totalsByCurrency).map(([currency, amount]) => (
              <span key={currency} style={{ marginRight: 12 }}>
                {amount.toString()} {currency}
              </span>
            ))}
          </Text>
        }
      />
    </Card>
  );
}
