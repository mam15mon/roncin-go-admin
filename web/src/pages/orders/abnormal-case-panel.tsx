import { PlusOutlined, WarningOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ModalForm, ProTable } from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { App, Button, Drawer, Popconfirm, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderAbnormalCaseServiceListAbnormalCases,
  orderAbnormalCaseServiceMarkAbnormalCase,
  orderAbnormalCaseServiceRemoveAbnormalCase,
  orderAbnormalCaseServiceResolveAbnormalCase,
} from '@/services/roncin/orderAbnormalCaseService';

const { Text } = Typography;

type AbnormalCaseFormValues = {
  abnormalCaseId: string;
};

type AbnormalCasePanelProps = {
  canManage: boolean;
  masterOptions: API.MasterDataItem[];
};

export type AbnormalCasePanelRef = {
  open: (order: API.Order) => void;
};

const isAbnormalCase = (kind?: number | string) =>
  kind === 10 ||
  kind === '10' ||
  kind === 'MASTER_DATA_KIND_ABNORMAL_CASE' ||
  kind === 'abnormal_case';

const abnormalCaseStatusValueEnum: Record<
  number,
  { text: string; color: string }
> = {
  1: { text: '处理中', color: 'error' },
  2: { text: '已解决', color: 'success' },
};

const AbnormalCasePanel = forwardRef<
  AbnormalCasePanelRef,
  AbnormalCasePanelProps
>(function AbnormalCasePanel({ canManage, masterOptions }, ref) {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [order, setOrder] = useState<API.Order>();

  useImperativeHandle(ref, () => ({
    open: (record) => {
      setOrder(record);
      setDrawerOpen(true);
    },
  }));

  const options = masterOptions
    .filter(
      (item) => isAbnormalCase(item.kind) && item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));
  const nameMap = Object.fromEntries(
    masterOptions
      .filter((item) => isAbnormalCase(item.kind) && item.id)
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

  const columns: ProColumns<API.OrderAbnormalCase>[] = [
    {
      title: '异常类型',
      dataIndex: 'abnormalCaseId',
      ellipsis: true,
      render: (_, record) => {
        const label = (record.abnormalCaseId && nameMap[record.abnormalCaseId]) || record.abnormalCaseId;
        return (
          <Space size={6}>
            <WarningOutlined style={{ color: '#ff4d4f' }} />
            <Text strong>{label || '-'}</Text>
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        1: { text: '处理中' },
        2: { text: '已解决' },
      },
      render: (_, record) => {
        const config = abnormalCaseStatusValueEnum[record.status ?? 0];
        return config ? <Tag color={config.color}>{config.text}</Tag> : '-';
      },
    },
    {
      title: '标记时间',
      dataIndex: 'markedAt',
      valueType: 'dateTime',
      width: 170,
      render: (_, record) =>
        record.markedAt
          ? dayjs(record.markedAt).format('YYYY-MM-DD HH:mm:ss')
          : '-',
    },
    {
      title: '标记人',
      dataIndex: 'markedBy',
      copyable: true,
      ellipsis: true,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.markedBy || '-'}
        </Text>
      ),
    },
    {
      title: '解决时间',
      dataIndex: 'resolvedAt',
      valueType: 'dateTime',
      width: 170,
      render: (_, record) =>
        record.resolvedAt
          ? dayjs(record.resolvedAt).format('YYYY-MM-DD HH:mm:ss')
          : <Text type="secondary">待解决</Text>,
    },
    {
      title: '解决人',
      dataIndex: 'resolvedBy',
      copyable: true,
      ellipsis: true,
      render: (_, record) => (
        record.resolvedBy ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.resolvedBy}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        )
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      search: false,
      width: 130,
      fixed: 'right',
      render: (_, record) => {
        if (!canManage) return null;
        return (
          <Space size={6}>
            {record.status === 1 && (
              <Popconfirm
                title="确定解决该异常？"
                onConfirm={async () => {
                  if (!order?.id || !record.id) return;
                  await orderAbnormalCaseServiceResolveAbnormalCase(
                    { orderId: order.id, id: record.id },
                    { orderId: order.id, id: record.id },
                  );
                  message.success('解决异常成功');
                  actionRef.current?.reload();
                }}
              >
                <Button type="link" size="small" style={{ padding: 0 }}>
                  解决
                </Button>
              </Popconfirm>
            )}
            <Popconfirm
              title="确定移除该异常？"
              onConfirm={async () => {
                if (!order?.id || !record.id) return;
                await orderAbnormalCaseServiceRemoveAbnormalCase({
                  orderId: order.id,
                  id: record.id,
                });
                message.success('移除异常成功');
                actionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small" style={{ padding: 0 }}>
                移除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  return (
    <>
      <Drawer
        title={order ? `订单异常协同 - ${order.orderNo || order.id}` : '订单异常协同'}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setOrder(undefined);
        }}
        width={920}
        destroyOnHidden
      >
        {order?.id && (
          <ProTable<API.OrderAbnormalCase>
            headerTitle={
              <Space size={8}>
                <WarningOutlined style={{ color: '#faad14' }} />
                <span>异常登记与闭环处理列表</span>
              </Space>
            }
            actionRef={actionRef}
            rowKey="id"
            columns={columns}
            bordered
            search={false}
            pagination={false}
            request={async () => {
              const response = await orderAbnormalCaseServiceListAbnormalCases({
                orderId: order.id as string,
              });
              return {
                data: response.data ?? [],
                success: response.success ?? true,
              };
            }}
            toolBarRender={() => [
              canManage && (
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    formRef.current?.resetFields();
                    setModalOpen(true);
                  }}
                >
                  标记异常
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<AbnormalCaseFormValues>
        title="标记订单异常"
        open={modalOpen}
        formRef={formRef}
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (!order?.id) return false;
          await orderAbnormalCaseServiceMarkAbnormalCase(
            { orderId: order.id },
            { orderId: order.id, abnormalCaseId: values.abnormalCaseId },
          );
          message.success('标记异常成功');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSearchableSelect
          name="abnormalCaseId"
          label="异常类别"
          rules={[{ required: true, message: '请选择异常类型' }]}
          options={options}
          placeholder="请选择异常类型"
        />
      </ModalForm>
    </>
  );
});

export default AbnormalCasePanel;
