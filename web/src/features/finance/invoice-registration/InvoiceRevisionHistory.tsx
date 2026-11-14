import { Alert, Collapse, Descriptions, Typography } from 'antd';
import { ResponsiveTable } from '@/components/ui';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';

type FeeSnapshot = {
  code?: string;
  name?: string;
  amount?: string;
  taxAmount?: string;
  netAmount?: string;
  sourceTaxRate?: string;
};
type BillSnapshot = {
  billNo?: string;
  amount?: string;
  taxAmount?: string;
  fees?: FeeSnapshot[];
};
type Snapshot = {
  number?: string;
  date?: string;
  invoiceType?: string;
  status?: string;
  invoiceTitle?: string;
  currency?: string;
  baseCurrency?: string;
  exchangeRate?: string;
  exchangeRateSource?: string;
  exchangeRateDate?: string;
  baseCurrencyAmount?: string;
  totalAmount?: string;
  netAmount?: string;
  taxAmount?: string;
  note?: string;
  bills?: BillSnapshot[];
};
const rateSourceLabels: Record<string, string> = {
  SYSTEM: '系统',
  MANUAL: '手工',
  WEEKLY: '公司周汇率',
  BOC_SYNC: '中行同步',
  DERIVED: '推导',
  INHERITED_LAST_WEEK: '上周继承（历史）',
};
function readSnapshot(value?: string): Snapshot | null | undefined {
  try {
    const parsed: unknown = JSON.parse(value ?? '');
    if (parsed === null) return null;
    if (typeof parsed !== 'object' || Array.isArray(parsed)) return undefined;
    const snapshot = parsed as Snapshot;
    if (
      typeof snapshot.totalAmount !== 'string' ||
      !Array.isArray(snapshot.bills)
    )
      return undefined;
    return snapshot;
  } catch {
    return undefined;
  }
}
function SnapshotView({ value }: { value?: string }) {
  const snapshot = readSnapshot(value);
  if (snapshot === undefined)
    return <Alert type="error" title="修订快照不完整，请联系管理员核对" />;
  if (snapshot === null)
    return <Typography.Paragraph>新增登记，此前无记录</Typography.Paragraph>;
  return (
    <>
      <Descriptions
        size="small"
        bordered
        column={3}
        items={[
          {
            key: 'number',
            label: '发票号码',
            children: snapshot.number || '-',
          },
          { key: 'date', label: '发票日期', children: snapshot.date || '-' },
          {
            key: 'type',
            label: '发票类型',
            children:
              snapshot.invoiceType === 'SPECIAL' ? '专用发票' : '普通发票',
          },
          {
            key: 'amount',
            label: '含税总额',
            children: `${formatAmount(snapshot.totalAmount)} ${snapshot.currency ?? ''}`,
          },
          {
            key: 'net',
            label: '未税总额',
            children: formatAmount(snapshot.netAmount),
          },
          {
            key: 'tax',
            label: '登记税额',
            children: formatAmount(snapshot.taxAmount),
          },
          {
            key: 'title',
            label: '抬头',
            children: snapshot.invoiceTitle || '-',
          },
          {
            key: 'exchangeRate',
            label: '登记汇率',
            children: trimDecimal(snapshot.exchangeRate),
          },
          {
            key: 'exchangeRateSource',
            label: '汇率来源',
            children:
              snapshot.exchangeRateSource === undefined
                ? '-'
                : (rateSourceLabels[snapshot.exchangeRateSource] ?? '未识别'),
          },
          {
            key: 'exchangeRateDate',
            label: '汇率生效日期',
            children: formatDate(snapshot.exchangeRateDate, 'date'),
          },
          {
            key: 'baseCurrencyAmount',
            label: '折本币金额',
            children: `${formatAmount(snapshot.baseCurrencyAmount)} ${snapshot.baseCurrency ?? ''}`,
          },
          { key: 'note', label: '备注', children: snapshot.note || '-' },
          {
            key: 'status',
            label: '登记状态',
            children:
              snapshot.status === 'CANCELLED' ? '已移除，原记录保留' : '有效',
          },
        ]}
      />
      {(snapshot.bills ?? []).map((bill) => (
        <div key={bill.billNo} style={{ marginTop: 8 }}>
          <Typography.Paragraph>
            账单 {bill.billNo} · 含税分配 {formatAmount(bill.amount)} · 登记税额{' '}
            {formatAmount(bill.taxAmount)}
          </Typography.Paragraph>
          <ResponsiveTable<FeeSnapshot>
            rowKey={(row, rowIndex) => `${row.code}:${rowIndex}`}
            size="small"
            pagination={false}
            dataSource={bill.fees ?? []}
            columns={[
              {
                title: '来源费用',
                key: 'fee',
                render: (_, fee) => `${fee.code ?? ''} ${fee.name ?? ''}`,
              },
              {
                title: '含税分配',
                dataIndex: 'amount',
                render: (value) => formatAmount(value),
              },
              {
                title: '未税分配',
                dataIndex: 'netAmount',
                render: (value) => formatAmount(value),
              },
              {
                title: '登记税额',
                dataIndex: 'taxAmount',
                render: (value) => formatAmount(value),
              },
              {
                title: '来源费用税率',
                dataIndex: 'sourceTaxRate',
                render: (value) =>
                  value === undefined ? '-' : `${trimDecimal(value)}%`,
              },
            ]}
          />
        </div>
      ))}
    </>
  );
}
export default function InvoiceRevisionHistory({
  revisions,
}: {
  revisions?: API.FinanceInvoiceRevision[];
}) {
  if (!revisions?.length)
    return <Typography.Paragraph>暂无修订记录</Typography.Paragraph>;
  return (
    <Collapse
      items={revisions.map((revision) => ({
        key: revision.id,
        label: `${formatDate(revision.createdAt, 'datetime')} · ${revision.actorName || '操作人未标识'} · ${revision.reason ?? ''}`,
        children: (
          <>
            <Typography.Title level={5}>修改前</Typography.Title>
            <SnapshotView value={revision.beforeSnapshot} />
            <Typography.Title level={5}>修改后</Typography.Title>
            <SnapshotView value={revision.afterSnapshot} />
          </>
        ),
      }))}
    />
  );
}
