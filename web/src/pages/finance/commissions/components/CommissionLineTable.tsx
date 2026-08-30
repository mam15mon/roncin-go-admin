import { Space, Table, Tag, Typography } from 'antd';
import React from 'react';
import { formatDate } from '@/utils/format';
import {
  decimalText,
  personnelRoleText,
} from '../types';

export const renderExpandedFees = (record: API.FinanceCommissionLine) => {
  if (!record.fees || record.fees.length === 0) {
    return (
      <Typography.Text
        type="secondary"
        style={{ padding: '8px 16px', display: 'block' }}
      >
        暂无关联费用明细
      </Typography.Text>
    );
  }
  const feeColumns = [
    {
      title: '费用项目',
      key: 'feeName',
      render: (_: unknown, fee: API.CommissionFeeDetail) => (
        <Space size={4}>
          <Typography.Text strong>{fee.feeName || '-'}</Typography.Text>
          {fee.feeCode && <Tag>{fee.feeCode}</Tag>}
        </Space>
      ),
    },
    {
      title: '收付方向',
      dataIndex: 'direction',
      key: 'direction',
      width: 90,
      render: (dir: string) => (
        <Tag color={dir === 'RECEIVABLE' ? 'blue' : 'orange'}>
          {dir === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      key: 'settlementPartyName',
      render: (val?: string) => val || '-',
    },
    {
      title: '原币金额',
      key: 'totalAmount',
      align: 'right' as const,
      render: (_: unknown, fee: API.CommissionFeeDetail) =>
        `${decimalText(fee.totalAmount)} ${fee.currency || ''}`,
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      key: 'exchangeRate',
      align: 'right' as const,
      width: 90,
      render: (val?: string) => decimalText(val),
    },
    {
      title: '折本币金额',
      dataIndex: 'standardAmount',
      key: 'standardAmount',
      align: 'right' as const,
      render: (val: string, fee: API.CommissionFeeDetail) =>
        `${decimalText(val)} ${fee.baseCurrency || ''}`,
    },
    {
      title: '费用发生日',
      dataIndex: 'expenseDate',
      key: 'expenseDate',
      width: 110,
      render: (val?: string) => val || '-',
    },
  ];

  return (
    <div
      style={{
        padding: '10px 16px',
        backgroundColor: '#fafbfc',
        borderRadius: 4,
        border: '1px solid #f0f0f0',
        margin: '4px 0',
      }}
    >
      <Typography.Text
        type="secondary"
        style={{ fontSize: 12, marginBottom: 6, display: 'block' }}
      >
        订单费用构成核算快照（共 {record.fees.length} 笔）
      </Typography.Text>
      <Table<API.CommissionFeeDetail>
        rowKey={(item) =>
          item.feeId || `${item.feeCode}-${item.expenseDate}-${item.totalAmount}`
        }
        columns={feeColumns}
        dataSource={record.fees}
        pagination={false}
        size="small"
        bordered
      />
    </div>
  );
};

export const previewColumns = [
  {
    title: '客户名称',
    dataIndex: 'customerName',
    key: 'customerName',
    width: 180,
    render: (val: string, line: API.FinanceCommissionLine) => (
      <Space vertical size={0}>
        <Typography.Text strong>{val || '-'}</Typography.Text>
        {line.customerCode && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {line.customerCode}
          </Typography.Text>
        )}
      </Space>
    ),
  },
  {
    title: '订单编号',
    dataIndex: 'orderNo',
    key: 'orderNo',
    width: 170,
    render: (val: string, line: API.FinanceCommissionLine) => (
      <Space vertical size={0}>
        <Typography.Text strong>{val}</Typography.Text>
        {line.orderDate && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {line.orderDate}
          </Typography.Text>
        )}
      </Space>
    ),
  },
  {
    title: '客户人员归属',
    key: 'personnelSnapshot',
    width: 170,
    render: (_: unknown, line: API.FinanceCommissionLine) => (
      <Space vertical size={0}>
        <span>{`${line.employeeName} · ${personnelRoleText(line.personnelRole)}`}</span>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {line.customerAssignedAt || line.personnelAssignedAt
            ? `归属于 ${formatDate(line.customerAssignedAt || line.personnelAssignedAt, 'date')}`
            : '-'}
        </Typography.Text>
      </Space>
    ),
  },
  {
    title: '已实现收入',
    dataIndex: 'realizedRevenue',
    key: 'realizedRevenue',
    align: 'right' as const,
    render: (value: string, line: API.FinanceCommissionLine) =>
      `${decimalText(value)} ${line.baseCurrency}`,
  },
  {
    title: '分摊成本',
    dataIndex: 'allocatedCost',
    key: 'allocatedCost',
    align: 'right' as const,
    render: (value: string, line: API.FinanceCommissionLine) =>
      `${decimalText(value)} ${line.baseCurrency}`,
  },
  {
    title: '已实现毛利',
    dataIndex: 'realizedProfit',
    key: 'realizedProfit',
    align: 'right' as const,
    render: (value: string, line: API.FinanceCommissionLine) => (
      <strong>{`${decimalText(value)} ${line.baseCurrency}`}</strong>
    ),
  },
  {
    title: '提成金额',
    dataIndex: 'commissionAmount',
    key: 'commissionAmount',
    align: 'right' as const,
    render: (value: string, line: API.FinanceCommissionLine) => (
      <Typography.Text strong type="success">
        {`${decimalText(value)} ${line.baseCurrency}`}
      </Typography.Text>
    ),
  },
];
