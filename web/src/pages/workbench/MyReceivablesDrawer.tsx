import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { TableColumnsType } from 'antd';
import { Alert, Drawer, Space, Table, Tag } from 'antd';
import React, { useState } from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import { workbenchServiceListMyReceivables } from '@/services/roncin/workbenchService';
import { formatDate } from '@/utils/format';
import { amountWithCurrency } from './display';

type MyReceivable = API.WorkbenchMyReceivable;

type ReceivablesQuery = {
  page: number;
  pageSize: number;
};

const DEFAULT_PAGE_SIZE = 20;

type Props = {
  open: boolean;
  onClose: () => void;
};

/** 服务端状态域前缀：在途回款下钻查询的统一 key 前缀。 */
const RECEIVABLES_QUERY_BASE = ['workbench', 'my-receivables'] as const;

/**
 * 在途回款抽屉：本人提成归属相关的已确认应收未结项，服务端分页。
 * 余额保留原币，不同币种不合并；潜在提成为预计值，非应发承诺。
 */
export default function MyReceivablesDrawer({ open, onClose }: Props) {
  const [query, setQuery] = useState<ReceivablesQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });

  const { data, isFetching } = useQuery({
    queryKey: [
      ...RECEIVABLES_QUERY_BASE,
      { page: query.page, pageSize: query.pageSize },
    ],
    queryFn: () =>
      workbenchServiceListMyReceivables({
        page: query.page,
        pageSize: query.pageSize,
      }),
    enabled: open,
    // 旧行为为空 catch 静默，仅靠请求层 notification；显式声明避免全局 message 补充弹错
    meta: { silent: true },
    // 翻页加载期间保留当前页内容，与既有手写层行为一致。
    placeholderData: keepPreviousData,
  });

  const items = data?.data ?? [];
  const total = Number(data?.total ?? 0);

  const columns: TableColumnsType<MyReceivable> = [
    {
      title: '账单号',
      dataIndex: 'billNo',
      width: 170,
      render: (_, record) => record.billNo || '-',
    },
    {
      title: '结算方',
      dataIndex: 'settlementPartyName',
      width: 180,
      render: (_, record) => record.settlementPartyName || '-',
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 80,
      render: (_, record) => record.currency || '-',
    },
    {
      title: '账单金额',
      dataIndex: 'totalAmount',
      width: 130,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(record.totalAmount, record.currency),
    },
    {
      title: '未核销余额',
      dataIndex: 'unverifiedAmount',
      width: 130,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(record.unverifiedAmount, record.currency),
    },
    {
      title: '账单日期',
      dataIndex: 'billDate',
      width: 110,
      render: (_, record) => formatDate(record.billDate, 'date'),
    },
    {
      title: '到期日',
      dataIndex: 'dueDate',
      width: 110,
      render: (_, record) => formatDate(record.dueDate, 'date'),
    },
    {
      title: '逾期状态',
      dataIndex: 'overdueDays',
      width: 110,
      render: (_, record) => {
        const days = record.overdueDays ?? 0;
        if (days > 0) {
          return <Tag color="red">逾期 {days} 天</Tag>;
        }
        return <Tag color="green">未逾期</Tag>;
      },
    },
  ];

  const columnSettings = useColumnSettings<TableColumnsType<MyReceivable>[number]>({
    tableKey: 'workbench:my-receivables',
    columns,
  });

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="在途回款"
      size={1080}
      destroyOnHidden
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <Alert
          type="info"
          showIcon
          title="未核销余额按账单原币展示，不同币种不合并计算；对应潜在提成仅为预计，非应发承诺，尚未回款核销、费用或规则变化都会影响结果。"
        />
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
          {columnSettings.entry}
        </div>
        <Table<MyReceivable>
          rowKey={(record) => record.billId || record.billNo || ''}
          size="small"
          loading={isFetching}
          columns={columnSettings.columns}
          dataSource={items}
          pagination={{
            current: query.page,
            pageSize: query.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (count) => `共 ${count} 条`,
            onChange: (page, pageSize) =>
              setQuery((prev) => ({ ...prev, page, pageSize })),
          }}
        />
        {columnSettings.modal}
      </Space>
    </Drawer>
  );
}
