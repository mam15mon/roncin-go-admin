import type { TableColumnsType } from 'antd';
import React from 'react';
import { ResponsiveTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { OrderFeeDirection } from '@/enums.generated';
import { formatAmount, trimDecimal } from '@/utils/format';

const columns: TableColumnsType<API.OrderFee> = [
  {
    title: '费用项目',
    key: 'fee',
    width: 180,
    render: (_, fee) => (
      <>
        <div>{fee.feeCode || '-'}</div>
        <div>{fee.feeName || '-'}</div>
        {fee.feeNameEn && <div>{fee.feeNameEn}</div>}
      </>
    ),
  },
  { title: '结算单位', dataIndex: 'settlementPartyName', width: 200 },
  { title: '计费单位', dataIndex: 'billingUnit', width: 90 },
  {
    title: '数量',
    dataIndex: 'quantity',
    width: 90,
    align: 'right',
    render: (value) => trimDecimal(value),
  },
  {
    title: '单价',
    key: 'unitPrice',
    width: 110,
    align: 'right',
    render: (_, fee) => (
      <>
        <div>{trimDecimal(fee.unitPrice)}</div>
        <div>{fee.taxInclusive ? '含税' : '不含税'}</div>
      </>
    ),
  },
  {
    title: '原币金额',
    key: 'totalAmount',
    width: 140,
    align: 'right',
    render: (_, fee) =>
      `${formatAmount(fee.totalAmount)} ${fee.currency || '-'}`,
  },
  {
    title: '汇率',
    dataIndex: 'exchangeRate',
    width: 100,
    align: 'right',
    render: (value) => trimDecimal(value),
  },
  {
    title: '折本币金额',
    key: 'baseCurrencyAmount',
    width: 150,
    align: 'right',
    render: (_, fee) =>
      `${formatAmount(fee.baseCurrencyAmount)} ${fee.baseCurrency || '-'}`,
  },
  {
    title: '税率',
    dataIndex: 'taxRate',
    width: 90,
    align: 'right',
    render: (value) => (value == null ? '-' : `${trimDecimal(value)}%`),
  },
  {
    title: '不含税金额',
    dataIndex: 'netAmount',
    width: 120,
    align: 'right',
    render: (value) => formatAmount(value),
  },
  {
    title: '税额',
    dataIndex: 'taxAmount',
    width: 110,
    align: 'right',
    render: (value) => formatAmount(value),
  },
  {
    title: '建账状态',
    dataIndex: 'hasActiveBill',
    width: 100,
    render: (value) => (value ? '已建账' : '未建账'),
  },
  {
    title: '备注',
    dataIndex: 'note',
    width: 240,
    render: (value) => (
      <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
        {value || '-'}
      </span>
    ),
  },
];

/** 审核接口已按服务端当前有效费用口径筛选，完整展示全部行。 */
export default function OrderLockFeeDetails({
  fees,
}: {
  fees: API.OrderFee[];
}) {
  const settings = useColumnSettings<TableColumnsType<API.OrderFee>[number]>({
    tableKey: 'orders:lock-fee-review',
    columns,
    // 审核字段始终显示，保留公共列排序与冻结能力。
    lockVisibleKeys: columns.map((column) =>
      String(column.key ?? ('dataIndex' in column ? column.dataIndex : '')),
    ),
  });
  return (
    <>
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}
      >
        {settings.entry}
      </div>
      {[
        {
          label: '应收费用',
          direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
        },
        {
          label: '应付费用',
          direction: OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE,
        },
      ].map(({ label, direction }) => {
        const rows = fees.filter((fee) => fee.direction === direction);
        return (
          <section key={direction} aria-label={label} style={{ marginTop: 16 }}>
            <h4 style={{ marginBottom: 8 }}>
              {label}（{rows.length}笔）
            </h4>
            <ResponsiveTable<API.OrderFee>
              rowKey="id"
              size="small"
              pagination={false}
              columns={settings.columns}
              dataSource={rows}
              locale={{ emptyText: `暂无${label}` }}
            />
          </section>
        );
      })}
    </>
  );
}
