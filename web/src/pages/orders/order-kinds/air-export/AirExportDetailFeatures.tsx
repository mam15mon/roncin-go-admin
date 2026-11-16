import { Form } from 'antd';
import { useRef } from 'react';
import { ResponsiveProTable } from '@/components/ui';
import { history } from '@/router/history';
import { airExportServiceListAirHouseWaybillVersions } from '@/services/roncin/airExportService';
import { ensureListResponse } from '@/utils/api';
import { formatDate } from '@/utils/format';
import CargoItemDrawer, {
  type CargoItemDrawerRef,
} from '../../components/drawers/CargoItemDrawer';
import { AirWaybillFields } from '../../templates/components/air/AirWaybillFields';
import type { OrderDetailFeaturesProps } from '../types';
import AirWeightPanel from './AirWeightPanel';

export default function AirExportDetailFeatures({
  context,
  children,
}: OrderDetailFeaturesProps) {
  const cargoRef = useRef<CargoItemDrawerRef>(null);
  const masterId = context.order.airExport?.master?.id;
  return (
    <>
      {children({
        actionMenuItems: [
          ...(context.canOrder('cargo_item.read')
            ? [
                {
                  key: 'cargo-items',
                  label: '实际货物明细',
                  onClick: () => cargoRef.current?.open(context.order),
                },
              ]
            : []),
          ...(masterId
            ? [
                {
                  key: 'shared-master',
                  label: '共享主单与成员',
                  onClick: () =>
                    history.push(`/orders/${context.kind}/masters/${masterId}`),
                },
              ]
            : []),
        ],
        appendTabs: [
          {
            key: 'airWeight',
            label: '本票重量与确认',
            content: (
              <AirWeightPanel
                scope="ORDER"
                targetId={context.orderId}
                businessType={context.order.businessType}
                canUpdate={
                  context.canOrder('update') && !context.businessWritesDisabled
                }
              />
            ),
          },
          ...(context.order.airExport?.documentMode === 'HOUSE'
            ? [
                {
                  key: 'airHouseHistory',
                  label: '本票分单历史',
                  content: (
                    <ResponsiveProTable<API.AirHouseWaybillVersion>
                      rowKey="id"
                      search={false}
                      options={false}
                      size="small"
                      pagination={{ defaultPageSize: 10 }}
                      expandable={{
                        expandedRowRender: (item) => (
                          <Form
                            key={item.id}
                            initialValues={{ content: item.content }}
                            layout="vertical"
                            component={false}
                            disabled
                          >
                            <AirWaybillFields prefix={['content']} disabled />
                          </Form>
                        ),
                      }}
                      request={async ({ current, pageSize }) => {
                        const response =
                          await airExportServiceListAirHouseWaybillVersions({
                            orderId: context.orderId,
                            page: current || 1,
                            pageSize: pageSize || 10,
                          });
                        ensureListResponse(response, '分单历史加载失败');
                        return {
                          data: response?.data || [],
                          total: response?.total || 0,
                          success: true,
                        };
                      }}
                      columns={[
                        { title: '版本', dataIndex: 'version', width: 70 },
                        { title: '分单号', dataIndex: 'houseNo' },
                        {
                          title: '单证状态',
                          render: (_, item) =>
                            item.status === 'RELEASED'
                              ? '已签发'
                              : item.status === 'CONFIRMED'
                                ? '已确认'
                                : '资料待完善',
                        },
                        {
                          title: '单证备注',
                          render: (_, item) => item.content?.remarks || '-',
                          ellipsis: true,
                        },
                        {
                          title: '修改原因',
                          dataIndex: 'reason',
                          ellipsis: true,
                        },
                        {
                          title: '时间',
                          render: (_, item) =>
                            formatDate(item.occurredAt, 'datetime'),
                          width: 180,
                        },
                      ]}
                    />
                  ),
                },
              ]
            : []),
        ],
        overlays: (
          <CargoItemDrawer
            ref={cargoRef}
            canCreate={
              context.canOrder('cargo_item.create') &&
              !context.businessWritesDisabled
            }
            canUpdate={
              context.canOrder('cargo_item.update') &&
              !context.businessWritesDisabled
            }
            canRemove={
              context.canOrder('cargo_item.delete') &&
              !context.businessWritesDisabled
            }
          />
        ),
      })}
    </>
  );
}
