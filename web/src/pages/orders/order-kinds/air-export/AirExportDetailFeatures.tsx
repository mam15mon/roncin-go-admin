import { Button, Drawer, Form, Space } from 'antd';
import { useRef, useState } from 'react';
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
import AirCargoWorkspace from './AirCargoWorkspace';
import AirTransportMaintenance from './AirTransportMaintenance';

export default function AirExportDetailFeatures({
  context,
  children,
}: OrderDetailFeaturesProps) {
  const cargoRef = useRef<CargoItemDrawerRef>(null);
  const [houseHistoryOpen, setHouseHistoryOpen] = useState(false);
  const masterId = context.order.airExport?.master?.id;
  const canRead = context.canOrder('read');
  const canReadCargo = context.canOrder('cargo_item.read');
  const openMaster = () =>
    history.push(`/orders/${context.kind}/masters/${masterId}/awb`);
  const openHouse = () =>
    history.push(`/orders/${context.kind}/${context.orderId}/awb/HAWB`);
  const hasHouse =
    context.order.airExport?.documentMode === 'HOUSE' &&
    Boolean(context.order.airExport.house?.id);
  return (
    <>
      {children({
        documentMenuItems: [
          ...(masterId && context.canOrder('read')
            ? [
                {
                  key: 'awb-master',
                  label: 'MAWB 制作归档',
                  onClick: openMaster,
                },
              ]
            : []),
          ...(context.order.airExport?.documentMode === 'HOUSE' &&
          context.order.airExport.house?.id &&
          context.canOrder('read')
            ? [
                {
                  key: 'awb-house',
                  label:
                    context.kind === 'air-import'
                      ? 'HAWB 制作归档'
                      : 'HAWB 制作',
                  onClick: openHouse,
                },
              ]
            : []),
        ],
        airCargoWorkspace: (
          <AirCargoWorkspace
            order={context.order}
            canReadCargo={canReadCargo}
            canUpdate={
              context.canOrder('update') && !context.businessWritesDisabled
            }
            onOpenCargo={() => cargoRef.current?.open(context.order)}
          />
        ),
        airTransportMaintenance: <AirTransportMaintenance context={context} />,
        airDocumentActions: {
          MAWB:
            masterId && canRead ? (
              <Space wrap>
                <Button disabled={false} onClick={openMaster}>
                  制作 / 预览 MAWB
                </Button>
                <Button
                  disabled={false}
                  onClick={() =>
                    history.push(`/orders/${context.kind}/masters/${masterId}`)
                  }
                >
                  主单历史与成员
                </Button>
              </Space>
            ) : undefined,
          HAWB:
            hasHouse && canRead ? (
              <Space wrap>
                <Button disabled={false} onClick={openHouse}>
                  制作 / 预览 HAWB
                </Button>
                <Button
                  disabled={false}
                  onClick={() => setHouseHistoryOpen(true)}
                >
                  查看分单历史
                </Button>
              </Space>
            ) : undefined,
        },
        actionMenuItems: [
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
        overlays: (
          <>
            {hasHouse && canRead && (
              <Drawer
                title="本票分单历史"
                open={houseHistoryOpen}
                onClose={() => setHouseHistoryOpen(false)}
                size={1000}
                destroyOnHidden
              >
                {houseHistoryOpen && (
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
                )}
              </Drawer>
            )}
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
          </>
        ),
      })}
    </>
  );
}
