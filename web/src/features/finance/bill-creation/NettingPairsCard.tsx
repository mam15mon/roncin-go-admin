import { Card, Table, Tag, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import { useColumnSettings } from '@/components/ui/column-settings';

const { Text } = Typography;

export type NettingPairsCardProps = {
  pairs: API.BillBatchNettingPair[];
};

function amount(value: string | undefined, currency: string | undefined) {
  return `${value || '0'} ${currency || ''}`.trim();
}

// NettingPairsCard 展示对冲建账预览中每个“结算单位 + 账单币种”组合的毛额、
// 抵销额与净应收/净应付；金额只使用双方共同账单币种，不存在混合币种合计。
export default function NettingPairsCard({ pairs }: NettingPairsCardProps) {
  const columns: TableColumnsType<API.BillBatchNettingPair> = [
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 200,
      ellipsis: true,
    },
    {
      title: '账单币种',
      dataIndex: 'currency',
      width: 90,
    },
    {
      title: '应收毛额',
      dataIndex: 'receivableGrossAmount',
      align: 'right',
      render: (_, row) => (
        <span style={{ color: '#1677ff' }}>
          {amount(row.receivableGrossAmount, row.currency)}
        </span>
      ),
    },
    {
      title: '应付毛额',
      dataIndex: 'payableGrossAmount',
      align: 'right',
      render: (_, row) => (
        <span style={{ color: '#fa8c16' }}>
          {amount(row.payableGrossAmount, row.currency)}
        </span>
      ),
    },
    {
      title: '抵销额',
      dataIndex: 'offsetAmount',
      align: 'right',
      render: (_, row) => (
        <strong>{amount(row.offsetAmount, row.currency)}</strong>
      ),
    },
    {
      title: '净应收',
      dataIndex: 'netReceivableAmount',
      align: 'right',
      render: (_, row) =>
        Number(row.netReceivableAmount || 0) > 0 ? (
          <Tag color="blue" style={{ margin: 0 }}>
            {amount(row.netReceivableAmount, row.currency)}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '净应付',
      dataIndex: 'netPayableAmount',
      align: 'right',
      render: (_, row) =>
        Number(row.netPayableAmount || 0) > 0 ? (
          <Tag color="orange" style={{ margin: 0 }}>
            {amount(row.netPayableAmount, row.currency)}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
  ];

  const columnSettings =
    useColumnSettings<TableColumnsType<API.BillBatchNettingPair>[number]>({
      tableKey: 'finance:bill-netting-pairs',
      columns,
    });

  if (!pairs.length) return null;
  return (
    <Card
      size="small"
      style={{
        marginBottom: 16,
        background: '#f6ffed',
        border: '1px solid #b7eb8f',
      }}
      title={
        <Text strong>对冲抵销预览（{pairs.length} 组结算单位 × 币种）</Text>
      }
      extra={
        <Tag color="green" style={{ margin: 0 }}>
          抵销额 = 双方毛额较小值
        </Tag>
      }
    >
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
        {columnSettings.entry}
      </div>
      <Table<API.BillBatchNettingPair>
        rowKey={(row) => `${row.settlementPartyId || ''}|${row.currency || ''}`}
        size="small"
        bordered
        pagination={false}
        dataSource={pairs}
        columns={columnSettings.columns}
      />
      {columnSettings.modal}
      <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
        原始应收、应付账单仍分别生成并承担发票与毛额审计；对冲结算单只表达抵销事实。
      </Text>
    </Card>
  );
}
