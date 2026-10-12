import { Collapse, Space, type TableColumnsType, Typography } from 'antd';
import React from 'react';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';

const actions: Record<string, string> = {
  SUBMIT: '提交审核',
  WITHDRAW: '撤回',
  APPROVE: '批准',
  REJECT: '退回',
  REVOKE: '撤销批准',
  SUPPLEMENT_APPROVE: '批准补录',
};

function FeeReviewDirectionTable({
  data,
  direction,
}: {
  data: API.OrderFeeReviewData;
  direction: 1 | 2;
}) {
  const title = direction === 1 ? '应收' : '应付';
  const fees = (data.fees ?? []).filter((fee) => fee.direction === direction);
  const columns: TableColumnsType<API.OrderFee> = [
    {
      title: '费用项目',
      dataIndex: 'feeName',
      width: 140,
      render: (_, fee) => fee.feeName || fee.feeCode || '-',
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 160,
      ellipsis: true,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 70,
      align: 'right',
      render: (value) => trimDecimal(value),
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 95,
      align: 'right',
      render: (value) => trimDecimal(value),
    },
    { title: '币种', dataIndex: 'currency', width: 65, align: 'center' },
    {
      title: '含税金额',
      dataIndex: 'totalAmount',
      width: 120,
      align: 'right',
      render: (value) => (
        <Typography.Text strong>{formatAmount(value)}</Typography.Text>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 100,
      align: 'right',
      render: (value) => trimDecimal(value),
    },
    {
      title: `折本币${data.summary?.baseCurrency ? ` (${data.summary.baseCurrency})` : ''}`,
      dataIndex: 'baseCurrencyAmount',
      width: 130,
      align: 'right',
      render: (value) => formatAmount(value),
    },
    {
      title: '备注',
      dataIndex: 'note',
      width: 200,
      ellipsis: true,
      render: (value) => value || '-',
    },
  ];
  // 应收、应付采用同一标识和字段，保持审核明细列偏好共享。
  const columnSettings = useColumnSettings<
    TableColumnsType<API.OrderFee>[number]
  >({ tableKey: 'orders:fee-review-details', columns });
  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '5px 0',
          fontSize: 13,
        }}
      >
        <Typography.Text strong>
          <span
            style={{
              display: 'inline-block',
              width: 3,
              height: 13,
              background: direction === 1 ? '#1677ff' : '#fa8c16',
              marginRight: 7,
              verticalAlign: '-1px',
            }}
          />
          {title}费用{' '}
          <Typography.Text type="secondary">
            （{fees.length}笔）
          </Typography.Text>
        </Typography.Text>
        <Space size={8}>
          <Typography.Text type="secondary">
            合计{' '}
            {formatAmount(
              direction === 1
                ? data.summary?.receivable
                : data.summary?.payable,
            )}{' '}
            {data.summary?.baseCurrency}
          </Typography.Text>
          {columnSettings.entry}
        </Space>
      </div>
      <Table<API.OrderFee>
        size="small"
        bordered
        rowKey="id"
        pagination={false}
        scroll={{ x: 1170 }}
        dataSource={fees}
        locale={{ emptyText: `暂无${title}费用` }}
        columns={columnSettings.columns}
      />
    </div>
  );
}

function FeeReviewHistory({ data }: { data: API.OrderFeeReviewData }) {
  const columns: TableColumnsType<API.OrderFeeReviewEventData> = [
    {
      title: '操作',
      dataIndex: 'action',
      width: 100,
      render: (value: string) => actions[value] || value,
    },
    { title: '操作人', dataIndex: 'actorName', width: 100 },
    {
      title: '时间',
      dataIndex: 'createdAt',
      width: 170,
      render: (value) => formatDate(value, 'datetime'),
    },
    { title: '费用版本', dataIndex: 'contentVersion', width: 90 },
    {
      title: '快照利润',
      key: 'profit',
      width: 140,
      align: 'right',
      render: (_, event) =>
        event.summary
          ? `${formatAmount(event.summary.profit)} ${event.summary.baseCurrency}`
          : '-',
    },
    { title: '意见 / 原因', dataIndex: 'comment' },
  ];
  const columnSettings = useColumnSettings<
    TableColumnsType<API.OrderFeeReviewEventData>[number]
  >({ tableKey: 'orders:fee-review-history', columns });
  return (
    <div>
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 6 }}
      >
        {columnSettings.entry}
      </div>
      <Table<API.OrderFeeReviewEventData>
        size="small"
        rowKey="id"
        pagination={false}
        scroll={{ x: 750 }}
        dataSource={data.history ?? []}
        columns={columnSettings.columns}
      />
    </div>
  );
}

/** 明细仅用于核对整个订单费用包，始终只读且没有逐笔审批操作。 */
export function FeeReviewDetails({
  data,
  showHistory = true,
}: {
  data: API.OrderFeeReviewData;
  showHistory?: boolean;
}) {
  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <FeeReviewDirectionTable data={data} direction={1} />
      <FeeReviewDirectionTable data={data} direction={2} />
      {showHistory && (
        <Collapse
          size="small"
          items={[
            {
              key: 'history',
              label: `审核记录（${data.history?.length ?? 0}）`,
              children: <FeeReviewHistory data={data} />,
            },
          ]}
        />
      )}
    </div>
  );
}
