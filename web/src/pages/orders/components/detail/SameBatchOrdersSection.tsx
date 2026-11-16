import { LinkOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import type { TableColumnsType } from 'antd';
import { Alert, Button, Empty, Skeleton, Space, Tag } from 'antd';
import { useEffect } from 'react';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import { history } from '@/router/history';
import { orderServiceListSameBatchOrders } from '@/services/roncin/orderService';
import { ensureListResponse, unwrapList } from '@/utils/api';

const matchSourceLabels: Record<string, string> = {
  CUSTOMER_REFERENCE: '同客户业务号',
  BOOKING: '同订舱号',
  MASTER: '同 MBL',
};

type SameBatchOrdersSectionProps = {
  orderId: string;
  orderKind: string;
  /**
   * 向「关联与记录」页签角标暴露准确数量：加载中与失败时上报 undefined
   * （不显示数字），成功时上报去重并排除当前订单后的结果条数。
   */
  onCountChange?: (count: number | undefined) => void;
};

/** 展示由客户业务号、订舱号或真实 MBL 关系命中的同批订单。 */
export default function SameBatchOrdersSection({
  orderId,
  orderKind,
  onCountChange,
}: SameBatchOrdersSectionProps) {
  const query = useQuery({
    queryKey: ['orders', 'same-batch', { orderId, orderKind }],
    enabled: Boolean(orderId),
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await orderServiceListSameBatchOrders(
        { id: orderId },
        { signal, skipErrorHandler: true },
      );
      const seen = new Set<string>();
      return unwrapList(
        ensureListResponse(response, '同批订单加载失败'),
      ).filter((item) => {
        if (!item.orderId || item.orderId === orderId || seen.has(item.orderId))
          return false;
        seen.add(item.orderId);
        return true;
      });
    },
  });
  const orders = query.data ?? [];
  const loading = query.isPending && query.isFetching;
  const error = query.error?.message;

  useEffect(() => {
    onCountChange?.(query.isSuccess ? query.data.length : undefined);
  }, [onCountChange, query.data, query.isSuccess]);

  const columns: TableColumnsType<API.SameBatchOrderSummary> = [
    {
      title: '订单号',
      dataIndex: 'orderNo',
      render: (value: string, record) => (
        <Button
          type="link"
          size="small"
          icon={<LinkOutlined />}
          onClick={() => history.push(`/orders/${orderKind}/${record.orderId}`)}
        >
          {value || '-'}
        </Button>
      ),
    },
    {
      title: '匹配依据',
      dataIndex: 'matchSources',
      render: (sources: string[] | undefined) => (
        <Space size={[4, 4]} wrap>
          {(sources ?? []).map((source) => (
            <Tag key={source} color="blue">
              {matchSourceLabels[source] || source}
            </Tag>
          ))}
        </Space>
      ),
    },
    { title: '客户业务号', dataIndex: 'customerReferenceNo' },
    { title: '订舱号', dataIndex: 'bookingNo' },
    { title: 'MBL', dataIndex: 'masterNo' },
    { title: 'HBL', dataIndex: 'houseNo' },
    {
      title: '状态',
      dataIndex: 'flowStatus',
      render: (value: number | undefined) =>
        value === undefined
          ? '-'
          : statusText(orderFlowStatusMeta, value, '未知状态'),
    },
  ];

  const columnSettings = useColumnSettings<
    TableColumnsType<API.SameBatchOrderSummary>[number]
  >({
    tableKey: 'orders:same-batch',
    columns,
  });

  if (loading) return <Skeleton active paragraph={{ rows: 2 }} />;
  if (error) return <Alert type="warning" showIcon title={error} />;
  if (orders.length === 0) {
    return (
      <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无同批订单" />
    );
  }

  return (
    <>
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}
      >
        {columnSettings.entry}
      </div>
      <Table<API.SameBatchOrderSummary>
        rowKey={(record) => record.orderId || record.orderNo || ''}
        size="small"
        pagination={false}
        dataSource={orders}
        columns={columnSettings.columns}
      />
    </>
  );
}
