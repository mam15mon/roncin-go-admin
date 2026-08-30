import { ProTable } from '@ant-design/pro-components';
import { Drawer } from 'antd';
import { forwardRef, useImperativeHandle, useState } from 'react';
import { orderServiceListOrderConsolidations } from '@/services/roncin/orderService';
import { toTableRequest } from '@/utils/api';

function formatCargoMeasurement(value?: API.OrderCargoMeasurement) {
  return `${value?.packages ?? 0} 件 / ${(value?.grossWeightKg ?? 0).toFixed(3)} KGS / ${(value?.volumeCbm ?? 0).toFixed(3)} CBM`;
}

export type ConsolidationDrawerRef = {
  open: (order: API.Order) => void;
};

const ConsolidationDrawer = forwardRef<ConsolidationDrawerRef>(
  function ConsolidationDrawer(_, ref) {
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
      },
    }));

    return (
      <Drawer
        title={
          order
            ? `自拼订单汇总 - ${order.orderNo || order.id}`
            : '自拼订单汇总'
        }
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setOrder(undefined);
        }}
        size={1080}
        destroyOnHidden
      >
        {order?.id && (
          <ProTable<API.OrderConsolidationSummary>
            rowKey="consolidationId"
            bordered
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderServiceListOrderConsolidations({
                id: order.id as string,
              });
              return toTableRequest(response);
            }}
            columns={[
              { title: '主单号', dataIndex: 'masterNo', copyable: true },
              { title: '成员票数', dataIndex: 'memberCount', width: 100 },
              {
                title: '委托合计',
                dataIndex: 'entrusted',
                render: (_, record) => formatCargoMeasurement(record.entrusted),
              },
              {
                title: '实际合计',
                dataIndex: 'actual',
                render: (_, record) => formatCargoMeasurement(record.actual),
              },
            ]}
            expandable={{
              expandedRowRender: (summary) => (
                <ProTable<API.OrderConsolidationMember>
                  rowKey="orderId"
                  bordered
                  search={false}
                  pagination={false}
                  options={false}
                  dataSource={summary.members ?? []}
                  columns={[
                    { title: '订单编号', dataIndex: 'orderNo', copyable: true },
                    {
                      title: '客户业务编号',
                      dataIndex: 'customerReferenceNo',
                      renderText: (value) => value || '-',
                    },
                    {
                      title: '分单号',
                      dataIndex: 'houseNos',
                      renderText: (value: string[]) => value?.join('、') || '-',
                    },
                    {
                      title: '委托件重尺',
                      dataIndex: 'entrusted',
                      render: (_, record) =>
                        formatCargoMeasurement(record.entrusted),
                    },
                    {
                      title: '实际件重尺',
                      dataIndex: 'actual',
                      render: (_, record) =>
                        formatCargoMeasurement(record.actual),
                    },
                  ]}
                />
              ),
            }}
          />
        )}
      </Drawer>
    );
  },
);

export default ConsolidationDrawer;
