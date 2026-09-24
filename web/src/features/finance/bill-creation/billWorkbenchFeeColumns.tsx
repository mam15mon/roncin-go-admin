import { DeleteOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, Typography } from 'antd';
import React from 'react';
import { orderFeeStatusMeta, statusTag } from '@/constants/statusMeta';
import { OrderFeeStatus } from '@/enums.generated';
import { formatAmount } from '@/utils/format';

const { Text } = Typography;

export const baseFeeColumns: ProColumns<API.FeeLedgerItem>[] = [
  {
    title: '订单编号',
    dataIndex: 'orderNo',
    width: 140,
    copyable: true,
    search: false,
  },
  {
    title: '方向',
    dataIndex: 'direction',
    width: 70,
    valueType: 'select',
    search: false,
    valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
    render: (_, row) => (
      <Tag color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
        {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
      </Tag>
    ),
  },
  {
    title: '状态',
    dataIndex: 'status',
    width: 85,
    search: false,
    // 与费用台账共用 orderFeeStatusMeta 统一展示映射（未建账 gold、
    // 已进账单 blue、已作废 default）。
    render: (_, row) =>
      statusTag(
        orderFeeStatusMeta,
        row.status ?? OrderFeeStatus.ORDER_FEE_STATUS_UNBILLED,
      ),
  },
  {
    title: '结算单位',
    dataIndex: 'settlementPartyName',
    width: 180,
    ellipsis: true,
    search: false,
  },
  { title: '费用名称', dataIndex: 'feeName', width: 130, search: false },
  {
    title: '税率',
    dataIndex: 'taxRate',
    width: 75,
    align: 'right',
    search: false,
    renderText: (value) => (value == null ? '-' : `${Number(value)}%`),
  },
  {
    title: '金额',
    dataIndex: 'totalAmount',
    width: 140,
    align: 'right',
    search: false,
    render: (_, row) => (
      <Text strong>
        {formatAmount(row.totalAmount)} {row.currency}
      </Text>
    ),
  },
  {
    title: '费用日期',
    dataIndex: 'expenseDate',
    width: 110,
    search: false,
  },
];

export const selectionFeeColumns: ProColumns<API.FeeLedgerItem>[] = [
  {
    title: '关键词',
    dataIndex: 'keyword',
    hideInTable: true,
    fieldProps: { placeholder: '订单号、费用或结算单位' },
  },
  {
    title: '方向',
    dataIndex: 'direction',
    hideInTable: true,
    valueType: 'select',
    valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
  },
  ...baseFeeColumns,
];

export function getPreviewFeeColumns(
  onRemoveFee?: (id?: string) => void,
): ProColumns<API.FeeLedgerItem>[] {
  return [
    ...baseFeeColumns.filter((col) => col.dataIndex !== 'direction'),
    ...(onRemoveFee
      ? [
          {
            title: '操作',
            width: 75,
            align: 'center' as const,
            search: false,
            render: (_: unknown, row: API.FeeLedgerItem) => (
              <Popconfirm
                title="确认将该笔费用移出本次建单？"
                onConfirm={() => void onRemoveFee(row.id)}
                okText="移出"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  style={{ padding: 0 }}
                >
                  移出
                </Button>
              </Popconfirm>
            ),
          },
        ]
      : []),
  ];
}
