import { Collapse, Typography } from 'antd';
import { ResponsiveTable } from '@/components/ui';
import { formatAmount, trimDecimal } from '@/utils/format';

/** 在票据和账单两端呈现同一份原始分配事实，包括已释放的历史分配。 */
export default function InvoiceAllocationDetails({
  links,
}: {
  links?: API.FinanceInvoiceBill[];
}) {
  return (
    <Collapse
      items={(links ?? []).map((link) => ({
        key: link.id,
        label: `账单 ${link.billNo ?? ''} · ${formatAmount(link.amount)} · ${link.active === true ? '有效' : '已释放，历史保留'}`,
        children: (
          <>
            <Typography.Paragraph>
              本账单登记税额：{formatAmount(link.taxAmount)}
            </Typography.Paragraph>
            <ResponsiveTable<API.FinanceInvoiceFeeAllocation>
              rowKey="id"
              pagination={false}
              size="small"
              dataSource={link.fees ?? []}
              columns={[
                {
                  title: '来源费用',
                  key: 'fee',
                  render: (_, fee) =>
                    `${fee.feeCode ?? ''} ${fee.feeName ?? ''}`,
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
          </>
        ),
      }))}
    />
  );
}
