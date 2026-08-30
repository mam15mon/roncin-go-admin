import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDateTimePicker,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { forwardRef } from 'react';
import {
  SubEntityDrawerTemplate,
  type SubEntityDrawerRef,
} from '@/components/ui/sub-entity-drawer';
import {
  orderMilestoneServiceListMilestones,
  orderMilestoneServiceSetMilestone,
} from '@/services/roncin/orderMilestoneService';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

export type MilestoneDrawerRef = SubEntityDrawerRef<API.Order>;

type MilestoneDrawerProps = {
  canSet: boolean;
};

type MilestoneFormValues = {
  type: string;
  occurredAt?: string;
  clearOccurredAt?: boolean;
  note?: string;
};

const columns: ProColumns<API.OrderMilestone>[] = [
  {
    title: '里程碑类型',
    dataIndex: 'type',
    width: 160,
    render: (_, record) => (
      <Tag color="blue" variant="filled">
        {record.type || '-'}
      </Tag>
    ),
  },
  {
    title: '节点编码',
    dataIndex: 'templateNodeCode',
    width: 150,
    render: (_, record) =>
      record.templateNodeCode ? (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.templateNodeCode}
        </Text>
      ) : (
        '-'
      ),
  },
  {
    title: '节点名称',
    dataIndex: 'templateNodeLabel',
    width: 150,
    render: (_, record) => record.templateNodeLabel || '-',
  },
  {
    title: '发生时间',
    dataIndex: 'occurredAt',
    valueType: 'dateTime',
    width: 180,
    render: (_, record) =>
      record.occurredAt ? (
        formatDate(record.occurredAt)
      ) : (
        <Text type="secondary">未完成</Text>
      ),
  },
  {
    title: '备注说明',
    dataIndex: 'note',
    ellipsis: true,
    render: (_, record) => record.note || '-',
  },
];

const MilestoneDrawer = forwardRef<MilestoneDrawerRef, MilestoneDrawerProps>(
  function MilestoneDrawer({ canSet }, ref) {
    return (
      <SubEntityDrawerTemplate<
        API.OrderMilestone,
        API.Order,
        MilestoneFormValues
      >
        ref={ref}
        entityName="里程碑"
        drawerTitle={(order) =>
          order
            ? `订单履约里程碑 - ${order.orderNo || order.id}`
            : '订单履约里程碑'
        }
        drawerWidth={860}
        canCreate={canSet}
        canUpdate={canSet}
        canRemove={false}
        columns={columns}
        fetchList={(order) =>
          orderMilestoneServiceListMilestones({
            orderId: order.id as string,
          })
        }
        createItem={(values, order) => {
          const milestoneType = values.type.trim();
          return orderMilestoneServiceSetMilestone(
            { orderId: order.id as string, type: milestoneType },
            {
              orderId: order.id as string,
              type: milestoneType,
              expectedOrderVersion: order.version || '',
              occurredAt: values.clearOccurredAt
                ? undefined
                : values.occurredAt
                  ? dayjs(values.occurredAt).toISOString()
                  : undefined,
              clearOccurredAt: Boolean(values.clearOccurredAt),
              note: values.note,
            },
          );
        }}
        updateItem={(record, values, order) => {
          const milestoneType = (record.type || values.type || '').trim();
          return orderMilestoneServiceSetMilestone(
            { orderId: order.id as string, type: milestoneType },
            {
              orderId: order.id as string,
              type: milestoneType,
              expectedOrderVersion: order.version || '',
              occurredAt: values.clearOccurredAt
                ? undefined
                : values.occurredAt
                  ? dayjs(values.occurredAt).toISOString()
                  : undefined,
              clearOccurredAt: Boolean(values.clearOccurredAt),
              note: values.note,
            },
          );
        }}
        initialValues={(editing) =>
          editing
            ? {
                type: editing.type ?? '',
                occurredAt: editing.occurredAt
                  ? (dayjs(editing.occurredAt) as any)
                  : undefined,
                clearOccurredAt: false,
                note: editing.note,
              }
            : {
                type: '',
                clearOccurredAt: false,
              }
        }
        modalWidth={520}
        renderFormItems={(editing) => (
          <>
            <ProFormText
              name="type"
              label="里程碑类型"
              placeholder="请输入里程碑类型 (如 BOOKING_CONFIRMED)"
              disabled={Boolean(editing?.type)}
              rules={[{ required: true, message: '请输入里程碑类型' }]}
            />
            <ProFormDateTimePicker
              name="occurredAt"
              label="发生时间"
              fieldProps={{ style: { width: '100%' } }}
            />
            <ProFormSwitch name="clearOccurredAt" label="清除发生时间" />
            <ProFormTextArea
              name="note"
              label="备注说明"
              placeholder="请输入备注"
              fieldProps={{ maxLength: 500, showCount: true }}
            />
          </>
        )}
      />
    );
  },
);

export default MilestoneDrawer;
