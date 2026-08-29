import {
  CheckOutlined,
  CloseCircleOutlined,
  EditOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { Popconfirm, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { statusOptions } from './billConstants';

interface GetFinanceBillColumnsParams {
  access: {
    canUpdateFinanceBills?: boolean;
    canConfirmFinanceBills?: boolean;
  };
  onOpenDetail: (bill: API.FinanceBill) => void;
  onOpenEdit: (bill: API.FinanceBill) => void;
  onConfirmBill: (bill: API.FinanceBill) => void;
  onCancelBill: (bill: API.FinanceBill) => void;
}

export function getFinanceBillColumns({
  access,
  onOpenDetail,
  onOpenEdit,
  onConfirmBill,
  onCancelBill,
}: GetFinanceBillColumnsParams): ProColumns<API.FinanceBill>[] {
  return [
    {
      title: '标签',
      dataIndex: 'tags',
      width: 140,
      render: (_, row) =>
        row.tags?.length ? (
          <React.Fragment>
            {row.tags.map((tag) => (
              <Tag
                key={tag.id}
                style={
                  tag.groupColor
                    ? { color: tag.groupColor, borderColor: tag.groupColor, marginInlineEnd: 4 }
                    : { marginInlineEnd: 4 }
                }
              >
                {tag.name}
              </Tag>
            ))}
          </React.Fragment>
        ) : (
          '-'
        ),
    },
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 75,
      fixed: 'left',
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
      render: (_, row) => (
        <Tag
          color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}
          style={{ margin: 0 }}
        >
          {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 85,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusOptions).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const value = statusOptions[row.status || 'DRAFT'];
        return (
          <Tag color={value?.color} style={{ margin: 0 }}>
            {value?.text}
          </Tag>
        );
      },
    },
    {
      title: '账单编号',
      dataIndex: 'billNo',
      width: 170,
      copyable: true,
      search: false,
      render: (val, row) => (
        <a style={{ fontWeight: 500 }} onClick={() => void onOpenDetail(row)}>
          {val}
        </a>
      ),
    },
    {
      title: '建单批次',
      dataIndex: 'batchNo',
      width: 175,
      copyable: true,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '对账抬头',
      dataIndex: 'statementTitle',
      width: 200,
      ellipsis: true,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '账单金额',
      dataIndex: 'totalAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong style={{ color: '#262626' }}>
          {row.totalAmount} {row.currency}
        </strong>
      ),
    },
    {
      title: '账单汇率',
      dataIndex: 'exchangeRate',
      width: 135,
      align: 'right',
      search: false,
      render: (_, row) => {
        if (!row.exchangeRate) return '-';
        const sourceLabel =
          row.exchangeRateSource === 'MANUAL'
            ? '手工'
            : row.exchangeRateSource === 'BASE_CURRENCY'
              ? '本币'
              : '系统';
        const sourceColor =
          row.exchangeRateSource === 'MANUAL' ? 'purple' : 'default';
        return (
          <Space size={4}>
            <span>{row.exchangeRate}</span>
            <Tag color={sourceColor} style={{ margin: 0, fontSize: 10 }}>
              {sourceLabel}
            </Tag>
          </Space>
        );
      },
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong
          style={{
            color: row.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
          }}
        >
          {row.baseCurrencyAmount} {row.baseCurrency}
        </strong>
      ),
    },
    {
      title: '已核销',
      dataIndex: 'verifiedAmount',
      width: 135,
      align: 'right',
      search: false,
      render: (_, row) =>
        `${row.verifiedAmount || '0.00000000'} ${row.currency}`,
    },
    {
      title: '未核销',
      dataIndex: 'unverifiedAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong
          style={{
            color:
              Number(row.unverifiedAmount || 0) > 0 ? '#cf1322' : '#389e0d',
          }}
        >
          {row.unverifiedAmount || '0.00000000'} {row.currency}
        </strong>
      ),
    },
    {
      title: '费用数',
      dataIndex: 'feeCount',
      width: 75,
      search: false,
      align: 'center',
    },
    {
      title: '账单日期',
      dataIndex: 'billDate',
      width: 120,
      search: false,
      render: (_, row) => row.billDate || '-',
    },
    {
      title: '到期日',
      dataIndex: 'dueDate',
      width: 110,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 160,
      search: false,
      render: (_, row) =>
        row.createdAt
          ? dayjs(row.createdAt).format('YYYY-MM-DD HH:mm')
          : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 170,
      render: (_, row) => [
        <a key="view" onClick={() => void onOpenDetail(row)}>
          <EyeOutlined /> 详情
        </a>,
        access.canUpdateFinanceBills && row.status === 'DRAFT' ? (
          <a key="edit" onClick={() => onOpenEdit(row)}>
            <EditOutlined /> 编辑
          </a>
        ) : null,
        access.canConfirmFinanceBills && row.status === 'DRAFT' ? (
          <Popconfirm
            key="confirm"
            title="确认该账单？确认后将锁定对账金额并进入开票与核销"
            onConfirm={() => void onConfirmBill(row)}
          >
            <a>
              <CheckOutlined /> 确认
            </a>
          </Popconfirm>
        ) : null,
        access.canUpdateFinanceBills && row.status !== 'CANCELLED' ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => onCancelBill(row)}
          >
            <CloseCircleOutlined /> 取消
          </a>
        ) : null,
      ],
    },
  ];
}
