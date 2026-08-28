import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import dayjs from 'dayjs';
import React, { forwardRef } from 'react';
import {
  SubEntityDrawerTemplate,
  type SubEntityDrawerRef,
} from '@/components/ui/sub-entity-drawer';
import {
  orderCargoItemServiceAddCargoItem,
  orderCargoItemServiceListCargoItems,
  orderCargoItemServiceRemoveCargoItem,
  orderCargoItemServiceUpdateCargoItem,
} from '@/services/roncin/orderCargoItemService';

export type CargoItemDrawerRef = SubEntityDrawerRef<API.Order>;

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
];

const CargoItemDrawer = forwardRef<CargoItemDrawerRef, CargoItemDrawerProps>(
  function CargoItemDrawer({ canCreate, canUpdate, canRemove }, ref) {
    return (
      <SubEntityDrawerTemplate<
        API.OrderCargoItem,
        API.Order,
        CargoItemFormValues
      >
        ref={ref}
        entityName="货物明细"
        drawerTitle={(order) =>
          order
            ? `订单货物明细 - ${order.orderNo || order.id}`
            : '订单货物明细'
        }
        canCreate={canCreate}
        canUpdate={canUpdate}
        canRemove={canRemove}
        columns={columns}
        fetchList={(order) =>
          orderCargoItemServiceListCargoItems({
            orderId: order.id as string,
          })
        }
        createItem={(values, order) =>
          orderCargoItemServiceAddCargoItem(
            { orderId: order.id as string },
            {
              orderId: order.id as string,
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
          )
        }
        updateItem={(record, values, order) =>
          orderCargoItemServiceUpdateCargoItem(
            {
              orderId: order.id as string,
              id: record.id as string,
            },
            {
              id: record.id as string,
              orderId: order.id as string,
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
          )
        }
        removeItem={(record, order) =>
          orderCargoItemServiceRemoveCargoItem({
            orderId: order.id as string,
            id: record.id as string,
          })
        }
        initialValues={(editing) =>
          editing
            ? {
                cargoName: editing.cargoName ?? '',
                packageCount: editing.packageCount ?? 1,
                grossWeightKg: editing.grossWeightKg ?? 0,
                volumeCbm: editing.volumeCbm ?? 0,
                netWeightKg: editing.netWeightKg,
                note: editing.note,
              }
            : {
                cargoName: '',
                packageCount: 1,
                grossWeightKg: 0,
                volumeCbm: 0,
              }
        }
        renderFormItems={() => (
          <>
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
          </>
        )}
      />
    );
  },
);

export default CargoItemDrawer;
