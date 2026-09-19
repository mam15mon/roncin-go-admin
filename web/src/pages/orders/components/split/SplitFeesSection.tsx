import { Card, Col, Row, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import Decimal from 'decimal.js';
import { SectionCard } from '@/components/ui';
import type { FeeCurrencySummary, ResultConfig } from '../../splitUtils';

const { Text } = Typography;

interface SplitFeesSectionProps {
  splitContext: API.SeaOrderSplitContextData | null;
  results: ResultConfig[];
  feeAssignments: Record<string, string>;
  setFeeAssignments: (
    value:
      | Record<string, string>
      | ((prev: Record<string, string>) => Record<string, string>),
  ) => void;
  feeCurrencySummaries: FeeCurrencySummary[];
}

/** 拆票页区块 4：草稿费用整行归属与各币种费用实时守恒。 */
export default function SplitFeesSection({
  splitContext,
  results,
  feeAssignments,
  setFeeAssignments,
  feeCurrencySummaries,
}: SplitFeesSectionProps) {
  // 费用分配列
  const feeColumns: ColumnsType<API.SeaOrderSplitDraftFeeItem> = [
    {
      title: '费用名称',
      dataIndex: 'feeName',
      render: (val, r) => (
        <span>
          <Tag color={r.direction === 'RECEIVABLE' ? 'green' : 'red'}>
            {r.direction === 'RECEIVABLE' ? '应收' : '应付'}
          </Tag>
          {val}
        </span>
      ),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      ellipsis: true,
      render: (val) => val || '-',
    },
    {
      title: '费用金额',
      render: (_, f) => (
        <Text strong>
          {f.currency} {f.totalAmount}
        </Text>
      ),
    },
    {
      title: '整行归属结果票',
      width: 260,
      render: (_, fee) => (
        <Select
          value={fee.id ? feeAssignments[fee.id] : undefined}
          style={{ width: '100%' }}
          onChange={(val) => {
            if (fee.id) {
              setFeeAssignments({ ...feeAssignments, [fee.id]: val });
            }
          }}
          options={results.map((r) => ({
            label: (
              <span>
                <Tag color={r.role === 'ORIGINAL' ? 'default' : 'blue'}>
                  {r.role === 'ORIGINAL' ? '原' : '新'}
                </Tag>
                {r.title}
              </span>
            ),
            value: r.key,
          }))}
        />
      ),
    },
  ];

  return (
    <SectionCard
      title={
        <Space>
          <Text strong>草稿费用整行归属</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            （仅未确认/未账单化的草稿费用可整行转移，税费与汇率快照全量保留）
          </Text>
        </Space>
      }
    >
      <Table<API.SeaOrderSplitDraftFeeItem>
        columns={feeColumns}
        dataSource={splitContext?.draftFees || []}
        rowKey="id"
        pagination={false}
        size="middle"
      />
      {feeCurrencySummaries.length > 0 && (
        <div style={{ marginTop: 16 }}>
          <Text strong>各币种费用实时守恒：</Text>
          <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
            {feeCurrencySummaries.map((summary) => {
              const assigned = Object.values(summary.assignedByResult).reduce(
                (total, amount) => total.add(amount),
                new Decimal(0),
              );
              const remainingColor = summary.remaining.isZero()
                ? 'success'
                : summary.remaining.isPositive()
                  ? 'processing'
                  : 'error';
              return (
                <Col span={12} key={summary.key}>
                  <Card
                    size="small"
                    type="inner"
                    title={`${summary.direction === 'RECEIVABLE' ? '应收' : '应付'} ${summary.currency}`}
                    extra={
                      <Tag color={remainingColor}>
                        {summary.remaining.isZero()
                          ? '已完整归属'
                          : '存在归属差额'}
                      </Tag>
                    }
                  >
                    <div>
                      基准：{summary.currency} {summary.baseline.toString()}
                    </div>
                    <div>
                      已分配：{summary.currency} {assigned.toString()}
                    </div>
                    <div>
                      剩余：{summary.currency} {summary.remaining.toString()}
                    </div>
                    <div style={{ marginTop: 6 }}>
                      {results.map((result) => (
                        <Tag key={result.key}>
                          {result.title}：{summary.currency}{' '}
                          {summary.assignedByResult[result.key]?.toString() ??
                            '0'}
                        </Tag>
                      ))}
                    </div>
                  </Card>
                </Col>
              );
            })}
          </Row>
        </div>
      )}
    </SectionCard>
  );
}
