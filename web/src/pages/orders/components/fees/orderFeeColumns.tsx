import { EditOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Popconfirm, Space, Tag } from 'antd';
import React from 'react';
import {
  FEE_BILLED,
  FEE_CANCELLED,
  FEE_CONFIRMED,
  FEE_DRAFT,
  RECEIVABLE,
  feeStatusCode,
} from './feeConstants';
import { trimExactDecimal } from '@/utils/decimal';

type OrderFeeColumnProps = {
  direction: number;
  financeLocked: boolean;
  onOpenModal: (direction: number, record?: API.OrderFee) => void;
  onConfirmFee: (record: API.OrderFee) => void;
  onReopenFee: (record: API.OrderFee) => void;
  onCancelFee: (record: API.OrderFee) => void;
};

export function getOrderFeeTableColumns({
  direction,
  financeLocked,
  onOpenModal,
  onConfirmFee,
  onReopenFee,
  onCancelFee,
}: OrderFeeColumnProps): ProColumns<API.OrderFee>[] {
  return [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => {
        if (feeStatusCode(record.status) === FEE_CONFIRMED)
          return <Tag color="green">已确认</Tag>;
        if (feeStatusCode(record.status) === FEE_BILLED)
          return <Tag color="blue">已进账单</Tag>;
        if (feeStatusCode(record.status) === FEE_CANCELLED)
          return <Tag>已作废</Tag>;
        return <Tag color="gold">草稿</Tag>;
      },
    },
    {
      title: '费用代码',
      dataIndex: 'feeCode',
      width: 120,
      copyable: true,
      render: (_, record) => record.feeCode || '-',
    },
    {
      title: '费用名称',
      dataIndex: 'feeName',
      width: 140,
      render: (_, record) => record.feeName || '-',
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 180,
      ellipsis: true,
      render: (_, record) => record.settlementPartyName || '-',
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 80,
      render: (_, record) => <Tag color="blue">{record.currency}</Tag>,
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 100,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.unitPrice),
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 80,
      align: 'right',
      render: (_, record) => trimExactDecimal(record.quantity),
    },
    {
      title: '计费单位',
      dataIndex: 'billingUnit',
      width: 90,
      render: (_, record) => record.billingUnit || '-',
    },
    {
      title: '总金额',
      dataIndex: 'totalAmount',
      width: 130,
      align: 'right',
      render: (_, record) => (
        <span
          style={{
            fontWeight: 600,
            color: direction === RECEIVABLE ? '#1677ff' : '#fa8c16',
          }}
        >
          {trimExactDecimal(record.totalAmount)} {record.currency}
        </span>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 100,
      align: 'right',
      render: (_, record) => (
        <Space size={4}>
          <span>{trimExactDecimal(record.exchangeRate)}</span>
          {record.exchangeRateSource === 'MANUAL' && (
            <Tag color="gold">手工</Tag>
          )}
          {record.exchangeRateSource === 'SYSTEM' && (
            <Tag color="blue">系统</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '发生日期',
      dataIndex: 'expenseDate',
      width: 220,
      render: (_, record) => record.expenseDate || '-',
    },
    {
      title: '备注',
      dataIndex: 'note',
      ellipsis: true,
      render: (_, record) => record.note || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 110,
      fixed: 'right',
      render: (_, record) =>
        financeLocked
          ? []
          : [
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_BILLED) && (
                <Button
                  key="edit"
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => onOpenModal(direction, record)}
                >
                  编辑
                </Button>
              ),
              feeStatusCode(record.status) === FEE_DRAFT && (
                <Popconfirm
                  key="confirm"
                  title="确认后该费用才能进入账单，确定继续？"
                  onConfirm={() => onConfirmFee(record)}
                >
                  <Button type="link" size="small">
                    确认
                  </Button>
                </Popconfirm>
              ),
              feeStatusCode(record.status) === FEE_CONFIRMED && (
                <Button
                  key="reopen"
                  type="link"
                  size="small"
                  onClick={() => onReopenFee(record)}
                >
                  撤回
                </Button>
              ),
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_CONFIRMED) && (
                <Button
                  key="cancel"
                  type="link"
                  size="small"
                  danger
                  onClick={() => onCancelFee(record)}
                >
                  作废
                </Button>
              ),
            ],
    },
  ];
}
