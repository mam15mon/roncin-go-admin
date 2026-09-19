import type { TableColumnsType } from 'antd';
import { Alert, Drawer, Space, Table, Tag } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
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

/**
 * 在途回款抽屉：本人提成归属相关的已确认应收未结项，服务端分页。
 * 余额保留原币，不同币种不合并；潜在提成为预计值，非应发承诺。
 */
export default function MyReceivablesDrawer({ open, onClose }: Props) {
  const [items, setItems] = useState<MyReceivable[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<ReceivablesQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  const sequenceRef = useRef(0);

  useEffect(() => {
    if (!open) return;
    const sequence = ++sequenceRef.current;
    setLoading(true);
    workbenchServiceListMyReceivables({
      page: query.page,
      pageSize: query.pageSize,
    })
      .then((response) => {
        if (sequence !== sequenceRef.current) return;
        setItems(response.data ?? []);
        setTotal(Number(response.total ?? 0));
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；保留当前内容并停止加载。
      })
      .finally(() => {
        if (sequence === sequenceRef.current) setLoading(false);
      });
    return () => {
      sequenceRef.current += 1;
    };
  }, [open, query]);

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
        <Table<MyReceivable>
          rowKey={(record) => record.billId || record.billNo || ''}
          size="small"
          loading={loading}
          columns={columns}
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
      </Space>
    </Drawer>
  );
}
