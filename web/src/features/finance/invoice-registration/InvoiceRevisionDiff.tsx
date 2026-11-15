import { Card, Typography } from 'antd';
import dayjs from 'dayjs';
import Decimal from 'decimal.js';
import { ResponsiveTable } from '@/components/ui';
import { formatAmount } from '@/utils/format';
import type {
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
} from './types';

export type RevisionDiffRow = {
  key: string;
  item: string;
  before: string;
  after: string;
};

const typeLabels: Record<string, string> = {
  NORMAL: '普通发票',
  SPECIAL: '专用发票',
};

function sameAmount(left?: string, right?: string) {
  try {
    return new Decimal(left || '0').eq(new Decimal(right || '0'));
  } catch {
    return left === right;
  }
}

function amountText(value?: string) {
  if (value === undefined || value === '') return '-';
  try {
    return formatAmount(new Decimal(value).toFixed());
  } catch {
    return value;
  }
}

/** 按原登记基线与当前表单计算变化项；只输出变化，不输出内部ID或版本。 */
export function revisionDiffRows(
  original: API.FinanceInvoice,
  draft: InvoiceRegistrationDraft | undefined,
  bills: InvoiceRegistrationBillSource[],
): RevisionDiffRow[] {
  if (!draft)
    return [
      {
        key: 'removed',
        item: '发票登记',
        before: original.taxInvoiceNo ?? '-',
        after: '移除',
      },
    ];
  const rows: RevisionDiffRow[] = [];
  const push = (key: string, item: string, before: string, after: string) => {
    if (before !== after) rows.push({ key, item, before, after });
  };
  push(
    'date',
    '发票日期',
    original.invoiceDate ?? '-',
    draft.invoiceDate && dayjs(draft.invoiceDate).isValid()
      ? dayjs(draft.invoiceDate).format('YYYY-MM-DD')
      : '-',
  );
  push(
    'number',
    '发票号码',
    original.taxInvoiceNo ?? '-',
    draft.taxInvoiceNo?.trim() || '-',
  );
  push(
    'type',
    '发票类型',
    typeLabels[original.invoiceType ?? ''] ?? original.invoiceType ?? '-',
    typeLabels[draft.invoiceType] ?? draft.invoiceType ?? '-',
  );
  if (!sameAmount(original.totalAmount, draft.totalAmount))
    rows.push({
      key: 'total',
      item: '发票含税总额',
      before: amountText(original.totalAmount),
      after: amountText(draft.totalAmount),
    });
  push('note', '备注', original.note || '-', draft.note?.trim() || '-');

  type Pair = { amount?: string; tax?: string };
  const before = new Map<string, Pair>();
  const after = new Map<string, Pair>();
  for (const link of original.billLinks ?? []) {
    if (link.active !== true) continue;
    for (const fee of link.fees ?? [])
      before.set(`${link.billId}:${fee.orderFeeId}`, {
        amount: fee.amount,
        tax: fee.taxAmount,
      });
  }
  for (const allocation of draft.allocations ?? [])
    for (const fee of allocation?.fees ?? [])
      if (allocation.billId && fee?.orderFeeId)
        after.set(`${allocation.billId}:${fee.orderFeeId}`, {
          amount: fee.amount,
          tax: fee.taxAmount,
        });
  const label = (key: string) => {
    const [billId, feeId] = key.split(':');
    const bill = bills.find((item) => item.id === billId);
    const fee = bill?.fees.find((item) => item.orderFeeId === feeId);
    return `${bill?.billNo ?? '账单'} · ${fee?.feeCode ?? ''} ${fee?.feeName ?? ''}`.trim();
  };
  for (const key of new Set([...before.keys(), ...after.keys()])) {
    const left = before.get(key);
    const right = after.get(key);
    if (!sameAmount(left?.amount, right?.amount))
      rows.push({
        key: `${key}:amount`,
        item: `${label(key)} 含税分配`,
        before: left ? amountText(left.amount) : '未分配',
        after: right ? amountText(right.amount) : '移除',
      });
    if (!sameAmount(left?.tax, right?.tax))
      rows.push({
        key: `${key}:tax`,
        item: `${label(key)} 登记税额`,
        before: left ? amountText(left.tax) : '未分配',
        after: right ? amountText(right.tax) : '移除',
      });
  }
  return rows;
}

/** 修订对照：原值与修改后值，变化项高亮。 */
export default function InvoiceRevisionDiff(props: {
  original: API.FinanceInvoice;
  draft?: InvoiceRegistrationDraft;
  bills: InvoiceRegistrationBillSource[];
}) {
  const rows = revisionDiffRows(props.original, props.draft, props.bills);
  return (
    <Card size="small" title="修订对照" style={{ marginBottom: 12 }}>
      {rows.length === 0 ? (
        <Typography.Text type="secondary">尚未修改任何登记内容</Typography.Text>
      ) : (
        <ResponsiveTable<RevisionDiffRow>
          rowKey="key"
          size="small"
          pagination={false}
          dataSource={rows}
          columns={[
            { title: '项目', dataIndex: 'item' },
            { title: '原值', dataIndex: 'before' },
            {
              title: '修改后',
              dataIndex: 'after',
              render: (value: string) => (
                <Typography.Text type="warning" strong>
                  {value}
                </Typography.Text>
              ),
            },
          ]}
        />
      )}
    </Card>
  );
}
