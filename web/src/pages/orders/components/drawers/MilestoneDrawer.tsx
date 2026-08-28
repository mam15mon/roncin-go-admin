import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateTimePicker,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Drawer, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderMilestoneServiceListMilestones,
  orderMilestoneServiceSetMilestone,
} from '@/services/roncin/orderMilestoneService';

const { Text } = Typography;

export type MilestoneDrawerRef = {
  open: (order: API.Order) => void;
};

type MilestoneDrawerProps = {
  canSet: boolean;
};

type MilestoneFormValues = {
  type: string;
  occurredAt?: string;
  clearOccurredAt?: boolean;
  note?: string;
};

const MilestoneDrawer = forwardRef<MilestoneDrawerRef, MilestoneDrawerProps>(
  function MilestoneDrawer({ canSet }, ref) {
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [editingMilestone, setEditingMilestone] = useState<API.OrderMilestone>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
      },
    }));

    const openCreateMilestone = () => {
      setEditingMilestone(undefined);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEditMilestone = (record: API.OrderMilestone) => {
      setEditingMilestone(record);
      formRef.current?.setFieldsValue({
        type: record.type,
        occurredAt: record.occurredAt ? dayjs(record.occurredAt) : undefined,
        clearOccurredAt: false,
        note: record.note,
      });
      setModalOpen(true);
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
            dayjs(record.occurredAt).format('YYYY-MM-DD HH:mm:ss')
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
      {
        title: '操作',
        valueType: 'option',
        width: 80,
        render: (_, record) => {
          if (!canSet) return null;
          return (
            <Button
              type="link"
              size="small"
              onClick={() => openEditMilestone(record)}
            >
              更新
            </Button>
          );
        },
      },
    ];

    return (
      <>
        <Drawer
          title={
            order
              ? `订单履约里程碑 - ${order.orderNo || order.id}`
              : '订单履约里程碑'
          }
          open={drawerOpen}
          onClose={() => {
            setDrawerOpen(false);
            setOrder(undefined);
          }}
          size={860}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderMilestone>
              actionRef={actionRef}
              rowKey={(record) => record.id || record.type || ''}
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderMilestoneServiceListMilestones({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canSet && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreateMilestone}
                  >
                    设置里程碑
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<MilestoneFormValues>
          title={editingMilestone ? '编辑里程碑' : '设置里程碑'}
          open={modalOpen}
          formRef={formRef}
          modalProps={{
            destroyOnHidden: true,
            width: 520,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!order?.id || !order?.version) return false;
            const milestoneType = (
              editingMilestone?.type ||
              values.type ||
              ''
            ).trim();
            if (!milestoneType) return false;

            await orderMilestoneServiceSetMilestone(
              {
                orderId: order.id,
                type: milestoneType,
              },
              {
                orderId: order.id,
                type: milestoneType,
                expectedOrderVersion: order.version,
                occurredAt: values.clearOccurredAt
                  ? undefined
                  : values.occurredAt
                    ? dayjs(values.occurredAt).toISOString()
                    : undefined,
                clearOccurredAt: Boolean(values.clearOccurredAt),
                note: values.note,
              },
            );
            message.success('里程碑设置成功');
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <ProFormText
            name="type"
            label="里程碑类型"
            placeholder="请输入里程碑类型 (如 BOOKING_CONFIRMED)"
            disabled={Boolean(editingMilestone?.type)}
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
        </ModalForm>
      </>
    );
  },
);

export default MilestoneDrawer;
