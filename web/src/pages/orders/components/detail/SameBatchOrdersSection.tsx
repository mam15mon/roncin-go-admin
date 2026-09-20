import { LinkOutlined } from '@ant-design/icons';
import { history } from '@umijs/max';
import { Alert, Button, Empty, Skeleton, Space, Table, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import { orderServiceListSameBatchOrders } from '@/services/roncin/orderService';

const matchSourceLabels: Record<string, string> = {
  CUSTOMER_REFERENCE: '同客户业务号',
  BOOKING: '同订舱号',
  MASTER: '同 MBL',
};

type SameBatchOrdersSectionProps = {
  orderId: string;
  orderKind: string;
};

/** 展示由客户业务号、订舱号或真实 MBL 关系命中的同批订单。 */
export default function SameBatchOrdersSection({
  orderId,
  orderKind,
}: SameBatchOrdersSectionProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [orders, setOrders] = useState<API.SameBatchOrderSummary[]>([]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');

    orderServiceListSameBatchOrders({ id: orderId })
      .then((response) => {
        if (!active) return;
        const seen = new Set<string>();
        setOrders(
          (response.data ?? []).filter((item) => {
            if (
              !item.orderId ||
              item.orderId === orderId ||
              seen.has(item.orderId)
            ) {
              return false;
            }
            seen.add(item.orderId);
            return true;
          }),
        );
      })
      .catch((reason: unknown) => {
        if (!active) return;
        setOrders([]);
        setError(reason instanceof Error ? reason.message : '同批订单加载失败');
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [orderId]);

  if (loading) return <Skeleton active paragraph={{ rows: 2 }} />;
  if (error) return <Alert type="warning" showIcon title={error} />;
  if (orders.length === 0) {
    return (
      <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无同批订单" />
    );
  }

  return (
    <Table<API.SameBatchOrderSummary>
      rowKey={(record) => record.orderId || record.orderNo || ''}
      size="small"
      pagination={false}
      dataSource={orders}
      columns={[
        {
          title: '订单号',
          dataIndex: 'orderNo',
          render: (value: string, record) => (
            <Button
              type="link"
              size="small"
              icon={<LinkOutlined />}
              onClick={() =>
                history.push(`/orders/${orderKind}/${record.orderId}`)
              }
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
      ]}
    />
  );
}
