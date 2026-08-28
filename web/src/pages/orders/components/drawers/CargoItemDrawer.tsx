import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Drawer, Popconfirm, Space } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderCargoItemServiceAddCargoItem,
  orderCargoItemServiceListCargoItems,
  orderCargoItemServiceRemoveCargoItem,
  orderCargoItemServiceUpdateCargoItem,
} from '@/services/roncin/orderCargoItemService';

export type CargoItemDrawerRef = {
  open: (order: API.Order) => void;
};

type CargoItemDrawerProps = {
  canCreate: boolean;
  canUpdate: boolean;
  canRemove: boolean;
};

type CargoItemFormValues = {
  cargoName: string;
  packageCount: number;
  grossWeightKg: number;
  volumeCbm: number;
  netWeightKg?: number;
  note?: string;
};

const CargoItemDrawer = forwardRef<CargoItemDrawerRef, CargoItemDrawerProps>(
  function CargoItemDrawer({ canCreate, canUpdate, canRemove }, ref) {
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [editingCargoItem, setEditingCargoItem] =
      useState<API.OrderCargoItem>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
      },
    }));

    const openCreateCargoItem = () => {
      setEditingCargoItem(undefined);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEditCargoItem = (record: API.OrderCargoItem) => {
      setEditingCargoItem(record);
      formRef.current?.setFieldsValue({
        cargoName: record.cargoName,
        packageCount: record.packageCount,
        grossWeightKg: record.grossWeightKg,
        volumeCbm: record.volumeCbm,
        netWeightKg: record.netWeightKg,
        note: record.note,
      });
      setModalOpen(true);
    };

    const columns: ProColumns<API.OrderCargoItem>[] = [
      {
        title: '货物名称',
        dataIndex: 'cargoName',
        ellipsis: true,
        render: (_, record) => record.cargoName || '-',
      },
      {
        title: '件数',
        dataIndex: 'packageCount',
        width: 100,
        render: (_, record) => record.packageCount ?? '-',
      },
      {
        title: '毛重(KG)',
        dataIndex: 'grossWeightKg',
        width: 120,
        render: (_, record) => record.grossWeightKg ?? '-',
      },
      {
        title: '体积(CBM)',
        dataIndex: 'volumeCbm',
        width: 120,
        render: (_, record) => record.volumeCbm ?? '-',
      },
      {
        title: '净重(KG)',
        dataIndex: 'netWeightKg',
        width: 120,
        render: (_, record) =>
          record.netWeightKg !== undefined && record.netWeightKg !== null
            ? record.netWeightKg
            : '-',
      },
      {
        title: '备注',
        dataIndex: 'note',
        ellipsis: true,
        render: (_, record) => record.note || '-',
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        valueType: 'dateTime',
        width: 180,
        render: (_, record) =>
          record.createdAt
            ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss')
            : '-',
      },
      {
        title: '操作',
        valueType: 'option',
        width: 120,
        render: (_, record) => {
          if (!canUpdate && !canRemove) return null;
          return (
            <Space size="small">
              {canUpdate && (
                <Button
                  type="link"
                  size="small"
                  onClick={() => openEditCargoItem(record)}
                >
                  编辑
                </Button>
              )}
              {canRemove && (
                <Popconfirm
                  title="确定移除该货物明细？"
                  onConfirm={async () => {
                    if (!order?.id || !record.id) return;
                    await orderCargoItemServiceRemoveCargoItem({
                      orderId: order.id,
                      id: record.id,
                    });
                    message.success('移除货物明细成功');
                    actionRef.current?.reload();
                  }}
                >
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              )}
            </Space>
          );
        },
      },
    ];

    return (
      <>
        <Drawer
          title={
            order
              ? `订单货物明细 - ${order.orderNo || order.id}`
              : '订单货物明细'
          }
          open={drawerOpen}
          onClose={() => {
            setDrawerOpen(false);
            setOrder(undefined);
          }}
          size={920}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderCargoItem>
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderCargoItemServiceListCargoItems({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canCreate && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreateCargoItem}
                  >
                    添加货物明细
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<CargoItemFormValues>
          title={editingCargoItem ? '编辑货物明细' : '添加货物明细'}
          open={modalOpen}
          formRef={formRef}
          initialValues={
            editingCargoItem
              ? {
                  cargoName: editingCargoItem.cargoName,
                  packageCount: editingCargoItem.packageCount,
                  grossWeightKg: editingCargoItem.grossWeightKg,
                  volumeCbm: editingCargoItem.volumeCbm,
                  netWeightKg: editingCargoItem.netWeightKg,
                  note: editingCargoItem.note,
                }
              : undefined
          }
          modalProps={{
            destroyOnHidden: true,
            width: 560,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!order?.id) return false;
            if (editingCargoItem?.id) {
              await orderCargoItemServiceUpdateCargoItem(
                {
                  orderId: order.id,
                  id: editingCargoItem.id,
                },
                {
                  id: editingCargoItem.id,
                  orderId: order.id,
                  cargoName: values.cargoName.trim(),
                  packageCount: Number(values.packageCount),
                  grossWeightKg: Number(values.grossWeightKg),
                  volumeCbm: Number(values.volumeCbm),
                  netWeightKg:
                    values.netWeightKg !== undefined &&
                    values.netWeightKg !== null
                      ? Number(values.netWeightKg)
                      : undefined,
                  note: values.note?.trim() || undefined,
                },
              );
              message.success('更新货物明细成功');
            } else {
              await orderCargoItemServiceAddCargoItem(
                {
                  orderId: order.id,
                },
                {
                  orderId: order.id,
                  cargoName: values.cargoName.trim(),
                  packageCount: Number(values.packageCount),
                  grossWeightKg: Number(values.grossWeightKg),
                  volumeCbm: Number(values.volumeCbm),
                  netWeightKg:
                    values.netWeightKg !== undefined &&
                    values.netWeightKg !== null
                      ? Number(values.netWeightKg)
                      : undefined,
                  note: values.note?.trim() || undefined,
                },
              );
              message.success('添加货物明细成功');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <ProFormText
            name="cargoName"
            label="货物名称"
            placeholder="请输入货名"
            rules={[{ required: true, message: '请输入货名' }]}
          />
          <ProFormDigit
            name="packageCount"
            label="件数"
            min={1}
            fieldProps={{ precision: 0 }}
            placeholder="请输入件数"
            rules={[{ required: true, message: '请输入件数' }]}
          />
          <ProFormDigit
            name="grossWeightKg"
            label="毛重 (KG)"
            min={0.001}
            placeholder="请输入毛重"
            rules={[{ required: true, message: '请输入毛重' }]}
          />
          <ProFormDigit
            name="volumeCbm"
            label="体积 (CBM)"
            min={0.001}
            placeholder="请输入体积"
            rules={[{ required: true, message: '请输入体积' }]}
          />
          <ProFormDigit
            name="netWeightKg"
            label="净重 (KG)"
            min={0.001}
            placeholder="请输入净重 (可选)"
          />
          <ProFormTextArea
            name="note"
            label="备注说明"
            placeholder="请输入备注 (可选)"
            fieldProps={{ maxLength: 500, showCount: true }}
          />
        </ModalForm>
      </>
    );
  },
);

export default CargoItemDrawer;
