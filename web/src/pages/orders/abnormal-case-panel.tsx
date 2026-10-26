import { PlusOutlined, WarningOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { Alert, App, Button, Drawer, Space, Typography } from 'antd';
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import {
  AsyncPopconfirm,
  GuardedModalForm,
  ProFormSearchableSelect,
  ResponsiveProTable as ProTable,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  makeValueEnum,
  orderAbnormalCaseStatusMeta,
  statusTag,
} from '@/constants/statusMeta';
import { MasterDataKind, OrderAbnormalCaseStatus } from '@/enums.generated';
import { masterDataServiceSearchOptions } from '@/services/roncin/masterDataService';
import {
  orderAbnormalCaseServiceListAbnormalCases,
  orderAbnormalCaseServiceMarkAbnormalCase,
  orderAbnormalCaseServiceRemoveAbnormalCase,
  orderAbnormalCaseServiceResolveAbnormalCase,
} from '@/services/roncin/orderAbnormalCaseService';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
  unwrapList,
} from '@/utils/api';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

type AbnormalCaseFormValues = {
  abnormalCaseId: string;
};

type AbnormalCasePanelProps = {
  canRead: boolean;
  canCreate: boolean;
  canResolve: boolean;
  canDelete: boolean;
  businessWriteBlockedReason?: string;
};

export type AbnormalCasePanelRef = {
  open: (order: API.Order) => void;
};

const AbnormalCasePanel = forwardRef<
  AbnormalCasePanelRef,
  AbnormalCasePanelProps
>(function AbnormalCasePanel(
  { canRead, canCreate, canResolve, canDelete, businessWriteBlockedReason },
  ref,
) {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [order, setOrder] = useState<API.Order>();

  useEffect(() => {
    if (!canCreate || businessWriteBlockedReason) setModalOpen(false);
  }, [canCreate, businessWriteBlockedReason]);

  useEffect(() => {
    if (!canRead) {
      setDrawerOpen(false);
      setModalOpen(false);
      setOrder(undefined);
    }
  }, [canRead]);

  const ensureCanWrite = (permitted: boolean) => {
    if (canRead && permitted && !businessWriteBlockedReason) return true;
    message.warning(businessWriteBlockedReason || '没有该异常操作权限');
    return false;
  };

  useImperativeHandle(ref, () => ({
    open: (record) => {
      if (!canRead) return;
      setOrder(record);
      setDrawerOpen(true);
    },
  }));

  const columns: ProColumns<API.OrderAbnormalCase>[] = [
    {
      title: '异常类型',
      dataIndex: 'abnormalCaseId',
      ellipsis: true,
      render: (_, record) => {
        const label = record.abnormalCaseName;
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
      valueEnum: makeValueEnum(orderAbnormalCaseStatusMeta),
      render: (_, record) =>
        statusTag(orderAbnormalCaseStatusMeta, record.status ?? 0),
    },
    {
      title: '标记时间',
      dataIndex: 'markedAt',
      valueType: 'dateTime',
      width: 170,
      render: (_, record) => formatDate(record.markedAt, 'datetime'),
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
        record.resolvedAt ? (
          formatDate(record.resolvedAt, 'datetime')
        ) : (
          <Text type="secondary">待解决</Text>
        ),
    },
    {
      title: '解决人',
      dataIndex: 'resolvedBy',
      copyable: true,
      ellipsis: true,
      render: (_, record) =>
        record.resolvedBy ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.resolvedBy}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      search: false,
      width: 130,
      fixed: 'right',
      render: (_, record) => {
        if (!canResolve && !canDelete) return null;
        return (
          <Space size={6}>
            {canResolve &&
              record.status ===
                OrderAbnormalCaseStatus.ORDER_ABNORMAL_CASE_STATUS_ACTIVE && (
                <AsyncPopconfirm
                  title="确定解决该异常？"
                  onConfirm={async (): Promise<false | undefined> => {
                    if (!ensureCanWrite(canResolve) || !order?.id || !record.id)
                      return false;
                    const writeResponse1 =
                      await orderAbnormalCaseServiceResolveAbnormalCase(
                        { orderId: order.id, id: record.id },
                        { orderId: order.id, id: record.id },
                      );
                    if (!isWriteResponse(writeResponse1, true)) return false;
                    message.success('解决异常成功');
                    actionRef.current?.reload();

                    return undefined;
                  }}
                >
                  <Button
                    type="link"
                    size="small"
                    disabled={Boolean(businessWriteBlockedReason)}
                    style={{ padding: 0 }}
                  >
                    解决
                  </Button>
                </AsyncPopconfirm>
              )}
            {canDelete && (
              <AsyncPopconfirm
                title="确定移除该异常？"
                onConfirm={async (): Promise<false | undefined> => {
                  if (!ensureCanWrite(canDelete) || !order?.id || !record.id)
                    return false;
                  const writeResponse2 =
                    await orderAbnormalCaseServiceRemoveAbnormalCase({
                      orderId: order.id,
                      id: record.id,
                    });
                  if (!isWriteResponse(writeResponse2)) return false;
                  message.success('移除异常成功');
                  actionRef.current?.reload();

                  return undefined;
                }}
              >
                <Button
                  type="link"
                  danger
                  size="small"
                  disabled={Boolean(businessWriteBlockedReason)}
                  style={{ padding: 0 }}
                >
                  移除
                </Button>
              </AsyncPopconfirm>
            )}
          </Space>
        );
      },
    },
  ];

  const columnSettings = useColumnSettings<ProColumns<API.OrderAbnormalCase>>({
    tableKey: 'orders:abnormal-cases',
    columns,
  });

  return (
    <>
      <Drawer
        title={
          order ? `订单异常协同 - ${order.orderNo || order.id}` : '订单异常协同'
        }
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setModalOpen(false);
          setOrder(undefined);
        }}
        size={920}
        destroyOnHidden
      >
        {businessWriteBlockedReason && (
          <Alert
            type="info"
            showIcon
            title={businessWriteBlockedReason}
            style={{ marginBottom: 12 }}
          />
        )}
        {canRead && order?.id && (
          <ProTable<API.OrderAbnormalCase>
            headerTitle={
              <Space size={8}>
                <WarningOutlined style={{ color: '#faad14' }} />
                <span>异常登记与闭环处理列表</span>
              </Space>
            }
            actionRef={actionRef}
            rowKey="id"
            columns={columnSettings.columns}
            bordered
            search={false}
            pagination={false}
            options={{ reload: true, density: true, setting: false }}
            request={async () => {
              const response = await orderAbnormalCaseServiceListAbnormalCases({
                orderId: order.id as string,
              });
              return toTableRequest(response);
            }}
            toolBarRender={() => [
              columnSettings.entry,
              canCreate && (
                <Button
                  key="create"
                  type="primary"
                  disabled={Boolean(businessWriteBlockedReason)}
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

      <GuardedModalForm<AbnormalCaseFormValues>
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
          if (!ensureCanWrite(canCreate) || !order?.id) return false;
          const writeResponse3 = await orderAbnormalCaseServiceMarkAbnormalCase(
            { orderId: order.id },
            { orderId: order.id, abnormalCaseId: values.abnormalCaseId },
          );
          if (!isWriteResponse(writeResponse3, true)) return false;
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
          fieldProps={{ filterOption: false }}
          request={async ({ keyWords }: { keyWords?: string }) => {
            if (!canCreate || !modalOpen || businessWriteBlockedReason)
              return [];
            const response = await masterDataServiceSearchOptions({
              kind: MasterDataKind.MASTER_DATA_KIND_ABNORMAL_CASE,
              keyword: keyWords,
              page: 1,
              pageSize: 50,
            });
            return unwrapList(
              ensureListResponse(response, '异常类型加载失败'),
            ).map((item) => ({
              label: item.code
                ? `${item.name} (${item.code})`
                : item.name || '-',
              value: item.id || '',
            }));
          }}
          placeholder="请选择异常类型"
        />
      </GuardedModalForm>
    </>
  );
});

export default AbnormalCasePanel;
