import {
  ProTable,
  type ActionType,
  type ProColumns,
} from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { Card, Col, Row, Statistic, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';

const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};
const businessRoutes: Record<string, string> = {
  SE: 'sea-export',
  SI: 'sea-import',
  AE: 'air-export',
  AI: 'air-import',
};
const statusLabels: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  BILLED: { text: '已进账单', color: 'blue' },
  CANCELLED: { text: '已作废', color: 'default' },
};

function amount(value?: string) {
  return Number(value || 0);
}

export default function FinanceFeeLedgerPage() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const columns: ProColumns<API.FeeLedgerItem>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '订单号、费用或结算单位' },
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(businessLabels).map(([key, text]) => [key, { text }]),
      ),
    },
    {
      title: '收付方向',
      dataIndex: 'direction',
      width: 90,
      valueType: 'select',
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
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusLabels).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const value = statusLabels[row.status || 'DRAFT'];
        return <Tag color={value.color}>{value.text}</Tag>;
      },
    },
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      width: 150,
      copyable: true,
      render: (_, row) => {
        const route = businessRoutes[row.businessType || ''];
        return route ? (
          <a onClick={() => history.push(`/orders/${route}/${row.orderId}`)}>
            {row.orderNo}
          </a>
        ) : (
          row.orderNo
        );
      },
    },
    { title: '费用代码', dataIndex: 'feeCode', width: 120, search: false },
    { title: '费用名称', dataIndex: 'feeName', width: 140, search: false },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 190,
      ellipsis: true,
      search: false,
    },
    {
      title: '原币金额',
      dataIndex: 'totalAmount',
      width: 130,
      align: 'right',
      search: false,
      render: (_, row) => `${row.totalAmount} ${row.currency}`,
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 110,
      align: 'right',
      search: false,
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 100,
      align: 'right',
      search: false,
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong>
          {row.baseCurrencyAmount} {row.baseCurrency}
        </strong>
      ),
    },
    {
      title: '费用日期',
      dataIndex: 'expenseDate',
      width: 110,
      valueType: 'dateRange',
      search: {
        transform: (value) => ({
          expenseDateFrom: value[0],
          expenseDateTo: value[1],
        }),
      },
    },
    { title: '备注', dataIndex: 'note', ellipsis: true, search: false },
  ];

  return (
    <div style={{ paddingBottom: 24 }}>
      <Row gutter={12} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="有效费用"
              value={Number(summary?.activeCount || 0)}
              suffix="笔"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="应收折本币"
              value={amount(summary?.receivableBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="应付折本币"
              value={amount(summary?.payableBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="确认毛利"
              value={amount(summary?.profitBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{
                color:
                  amount(summary?.profitBaseAmount) >= 0
                    ? '#52c41a'
                    : '#ff4d4f',
              }}
            />
          </Card>
        </Col>
      </Row>
      <ProTable<API.FeeLedgerItem>
        headerTitle="集运费用明细"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1650 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        request={async (params) => {
          const response = await settlementServiceListFeeLedger({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            businessType: params.businessType,
            direction: params.direction,
            status: params.status,
            expenseDateFrom: params.expenseDateFrom,
            expenseDateTo: params.expenseDateTo,
          });
          setSummary(response.summary);
          return {
            data: response.data || [],
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />
    </div>
  );
}
