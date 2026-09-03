import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { Typography } from 'antd';
import React, { forwardRef } from 'react';
import {
  SubEntityDrawerTemplate,
  type SubEntityDrawerRef,
} from '@/components/ui/sub-entity-drawer';
import {
  orderContainerServiceAddContainer,
  orderContainerServiceListContainers,
  orderContainerServiceRemoveContainer,
  orderContainerServiceUpdateContainer,
} from '@/services/roncin/orderContainerService';

const { Text } = Typography;

export type ContainerDrawerRef = SubEntityDrawerRef<API.Order>;

type ContainerDrawerProps = {
  canCreate: boolean;
  canUpdate: boolean;
  canRemove: boolean;
  containerSpecOptions: { label: string; value: string }[];
  containerSpecMap: Record<string, string>;
};

type ContainerFormValues = {
  containerNo: string;
  containerSpecId: string;
  sealNo?: string;
  packageCount: number;
  grossWeightKg: number;
  volumeCbm: number;
  note?: string;
};

const ContainerDrawer = forwardRef<ContainerDrawerRef, ContainerDrawerProps>(
  function ContainerDrawer(
    {
      canCreate,
      canUpdate,
      canRemove,
      containerSpecOptions,
      containerSpecMap,
    },
    ref,
  ) {
    const columns: ProColumns<API.OrderContainer>[] = [
      {
        title: '箱号',
        dataIndex: 'containerNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>
            {record.containerNo || '-'}
          </Text>
        ),
      },
      {
        title: '集装箱规格',
        dataIndex: 'containerSpecId',
        width: 160,
        render: (_, record) =>
          record.containerSpecId
            ? containerSpecMap[record.containerSpecId] ||
              record.containerSpecId
            : '-',
      },
      {
        title: '铅封号',
        dataIndex: 'sealNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => record.sealNo || '-',
      },
      {
        title: '件数 (PCS)',
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
        title: '备注',
        dataIndex: 'note',
        ellipsis: true,
        render: (_, record) => record.note || '-',
      },
    ];

    return (
      <SubEntityDrawerTemplate<
        API.OrderContainer,
        API.Order,
        ContainerFormValues
      >
        ref={ref}
        entityName="集装箱"
        drawerTitle={(order) =>
          order
            ? `订单集装箱列表 - ${order.orderNo || order.id}`
            : '订单集装箱列表'
        }
        canCreate={canCreate}
        canUpdate={canUpdate}
        canRemove={canRemove}
        columns={columns}
        fetchList={(order) =>
          orderContainerServiceListContainers({
            orderId: order.id as string,
          })
        }
        createItem={(values, order) =>
          orderContainerServiceAddContainer(
            { orderId: order.id as string },
            {
              orderId: order.id as string,
              containerNo: values.containerNo.trim(),
              containerSpecId: values.containerSpecId,
              sealNo: values.sealNo?.trim() || undefined,
              packageCount: Number(values.packageCount),
              grossWeightKg: Number(values.grossWeightKg),
              volumeCbm: Number(values.volumeCbm),
              note: values.note?.trim() || undefined,
            },
          )
        }
        updateItem={(record, values, order) =>
          orderContainerServiceUpdateContainer(
            {
              orderId: order.id as string,
              id: record.id as string,
            },
            {
              id: record.id as string,
              orderId: order.id as string,
              containerNo: values.containerNo.trim(),
              containerSpecId: values.containerSpecId,
              sealNo: values.sealNo?.trim() || undefined,
              packageCount: Number(values.packageCount),
              grossWeightKg: Number(values.grossWeightKg),
              volumeCbm: Number(values.volumeCbm),
              note: values.note?.trim() || undefined,
              expectedVersion: String(record.version ?? '1'),
            },
          )
        }
        removeItem={(record, order) =>
          orderContainerServiceRemoveContainer({
            orderId: order.id as string,
            id: record.id as string,
          })
        }
        initialValues={(editing) =>
          editing
            ? {
                containerNo: editing.containerNo ?? '',
                containerSpecId: editing.containerSpecId ?? '',
                sealNo: editing.sealNo,
                packageCount: editing.packageCount ?? 1,
                grossWeightKg: editing.grossWeightKg ?? 0,
                volumeCbm: editing.volumeCbm ?? 0,
                note: editing.note,
              }
            : {
                containerNo: '',
                containerSpecId: '',
                packageCount: 1,
                grossWeightKg: 0,
                volumeCbm: 0,
              }
        }
        renderFormItems={() => (
          <>
            <ProFormText
              name="containerNo"
              label="箱号"
              placeholder="请输入箱号 (如 COSU1234567)"
              rules={[{ required: true, message: '请输入箱号' }]}
            />
            <ProFormSearchableSelect
              name="containerSpecId"
              label="集装箱规格"
              rules={[{ required: true, message: '请选择箱型' }]}
              options={containerSpecOptions}
              placeholder="请选择箱型"
            />
            <ProFormText
              name="sealNo"
              label="铅封号"
              placeholder="请输入铅封号 (可选)"
            />
            <ProFormDigit
              name="packageCount"
              label="件数 (PCS)"
              min={1}
              fieldProps={{ precision: 0 }}
              placeholder="请输入件数"
              rules={[{ required: true, message: '请输入件数' }]}
            />
            <ProFormDigit
              name="grossWeightKg"
              label="货物毛重 (KG)"
              min={0.001}
              placeholder="请输入毛重"
              rules={[{ required: true, message: '请输入毛重' }]}
            />
            <ProFormDigit
              name="volumeCbm"
              label="货物体积 (CBM)"
              min={0.001}
              placeholder="请输入体积"
              rules={[{ required: true, message: '请输入体积' }]}
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

export default ContainerDrawer;
