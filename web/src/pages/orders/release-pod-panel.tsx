import { FileDoneOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Drawer,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  orderReleasePodServiceAddReleasePod,
  orderReleasePodServiceListReleasePods,
  orderReleasePodServiceRemoveReleasePod,
  orderReleasePodServiceTransitionReleasePodStatus,
  orderReleasePodServiceUpdateReleasePod,
} from '@/services/roncin/orderReleasePodService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';

const { Text } = Typography;

type ReleasePodFormValues = {
  releaseNo?: string;
  podNo?: string;
  shippingDocumentId?: string;
  note?: string;
};

type ReleasePodPanelProps = {
  canManage: boolean;
};

export type ReleasePodPanelRef = {
  open: (order: API.Order) => void;
};

const releasePodStatusValueEnum: Record<
  number,
  { text: string; color: string }
> = {
  1: { text: '待签收', color: 'default' },
  2: { text: '已签收', color: 'processing' },
  3: { text: '已回单', color: 'success' },
};

export function getReleasePodTransition(record?: API.OrderReleasePod) {
  const status = record?.status;
  const toStatus = record?.allowedTargetStatuses?.[0];
  if (!status || !toStatus) return undefined;
  return {
    currentText: releasePodStatusValueEnum[status]?.text ?? '未知状态',
    nextText: releasePodStatusValueEnum[toStatus]?.text ?? '未知状态',
    toStatus,
  };
}

const ReleasePodPanel = forwardRef<ReleasePodPanelRef, ReleasePodPanelProps>(
  function ReleasePodPanel({ canManage }, ref) {
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const { message } = App.useApp();
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [documents, setDocuments] = useState<API.OrderShippingDocument[]>([]);
    const [documentsError, setDocumentsError] = useState('');
    const [editingRecord, setEditingRecord] = useState<API.OrderReleasePod>();
    const activeOrderIdRef = useRef<string | undefined>(undefined);

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDocuments([]);
        setDocumentsError('');
        setDrawerOpen(true);
        const orderId = record.id as string;
        activeOrderIdRef.current = orderId;
        orderShippingDocumentServiceListShippingDocuments({
          orderId,
        })
          .then((response) => {
            if (activeOrderIdRef.current === orderId) {
              setDocuments(unwrapList(response));
            }
          })
          .catch((error: Error) => {
            if (activeOrderIdRef.current === orderId) {
              setDocuments([]);
              setDocumentsError(error.message || '关联提单加载失败');
            }
          });
      },
    }));

    const documentOptions = documents.map((document) => ({
      label: `${document.masterNo} / ${document.houseNo}`,
      value: document.id ?? '',
    }));
    const documentMap = Object.fromEntries(
      documents
        .filter((document) => document.id)
        .map((document) => [
          document.id as string,
          `${document.masterNo} / ${document.houseNo}`,
        ]),
    );

    const openCreate = () => {
      setEditingRecord(undefined);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEdit = (record: API.OrderReleasePod) => {
      setEditingRecord(record);
      formRef.current?.setFieldsValue({
        releaseNo: record.releaseNo,
        podNo: record.podNo,
        shippingDocumentId: record.shippingDocumentId,
        note: record.note,
      });
      setModalOpen(true);
    };

    const columns: ProColumns<API.OrderReleasePod>[] = [
      {
        title: '放货编号',
        dataIndex: 'releaseNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.releaseNo || '-'}
          </Text>
        ),
      },
      {
        title: '回单编号',
        dataIndex: 'podNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.podNo || '-'}
          </Text>
        ),
      },
      {
        title: '关联提单',
        dataIndex: 'shippingDocumentId',
        ellipsis: true,
        render: (_, record) =>
          (record.shippingDocumentId &&
            documentMap[record.shippingDocumentId]) ||
          '-',
      },
      {
        title: '状态',
        dataIndex: 'status',
        valueType: 'select',
        valueEnum: {
          1: { text: '待签收' },
          2: { text: '已签收' },
          3: { text: '已回单' },
        },
        render: (_, record) => {
          const config = releasePodStatusValueEnum[record.status ?? 0];
          return config ? <Tag color={config.color}>{config.text}</Tag> : '-';
        },
      },
      {
        title: '签收时间',
        dataIndex: 'signedAt',
        valueType: 'dateTime',
        width: 170,
        render: (_, record) =>
          record.signedAt ? (
            dayjs(record.signedAt).format('YYYY-MM-DD HH:mm:ss')
          ) : (
            <Text type="secondary">-</Text>
          ),
      },
      {
        title: '签收人',
        dataIndex: 'signedBy',
        copyable: true,
        ellipsis: true,
        render: (_, record) =>
          record.signedBy ? (
            <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
              {record.signedBy}
            </Text>
          ) : (
            <Text type="secondary">-</Text>
          ),
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
        width: 170,
        render: (_, record) =>
          record.createdAt
            ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss')
            : '-',
      },
      {
        title: '操作',
        valueType: 'option',
        search: false,
        width: 170,
        fixed: 'right',
        render: (_, record) => {
          const transition = getReleasePodTransition(record);
          if (!canManage) return null;
          return (
            <Space size="small">
              <Button type="link" size="small" onClick={() => openEdit(record)}>
                编辑
              </Button>
              {transition && (
                <Popconfirm
                  title={`确定将放货凭证状态从「${transition.currentText}」流转为「${transition.nextText}」？`}
                  onConfirm={async () => {
                    if (!order?.id || !record.id || !record.status) return;
                    await orderReleasePodServiceTransitionReleasePodStatus(
                      { orderId: order.id, id: record.id },
                      {
                        orderId: order.id,
                        id: record.id,
                        expectedStatus: record.status,
                        toStatus: transition.toStatus,
                      },
                    );
                    message.success('流转放货凭证状态成功');
                    actionRef.current?.reload();
                  }}
                >
                  <Button type="link" size="small">
                    流转
                  </Button>
                </Popconfirm>
              )}
              <Popconfirm
                title="确定移除该放货凭证？"
                onConfirm={async () => {
                  if (!order?.id || !record.id) return;
                  await orderReleasePodServiceRemoveReleasePod({
                    orderId: order.id,
                    id: record.id,
                  });
                  message.success('移除放货凭证成功');
                  actionRef.current?.reload();
                }}
              >
                <Button type="link" danger size="small">
                  删除
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
          title={
            order
              ? `订单放货凭证 (POD) - ${order.orderNo || order.id}`
              : '订单放货凭证 (POD)'
          }
          open={drawerOpen}
          onClose={() => {
            activeOrderIdRef.current = undefined;
            setDrawerOpen(false);
            setOrder(undefined);
            setDocuments([]);
            setDocumentsError('');
          }}
          size={920}
          destroyOnHidden
        >
          {documentsError && (
            <Alert
              type="error"
              showIcon
              title="关联提单加载失败"
              description={documentsError}
              style={{ marginBottom: 16 }}
            />
          )}
          {order?.id && (
            <ProTable<API.OrderReleasePod>
              headerTitle={
                <Space size={8}>
                  <FileDoneOutlined style={{ color: '#52c41a' }} />
                  <span>放货回单记录</span>
                </Space>
              }
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderReleasePodServiceListReleasePods({
                  orderId: order.id as string,
                });
                return toTableRequest(response);
              }}
              toolBarRender={() => [
                canManage && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreate}
                  >
                    添加放货凭证
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<ReleasePodFormValues>
          title={editingRecord ? '编辑放货凭证' : '添加放货凭证'}
          open={modalOpen}
          formRef={formRef}
          initialValues={editingRecord}
          modalProps={{
            destroyOnHidden: true,
            width: 560,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!order?.id) return false;
            const input = {
              orderId: order.id,
              releaseNo: values.releaseNo?.trim() || undefined,
              podNo: values.podNo?.trim() || undefined,
              shippingDocumentId: values.shippingDocumentId || undefined,
              note: values.note?.trim() || undefined,
            };
            if (editingRecord?.id) {
              await orderReleasePodServiceUpdateReleasePod(
                { orderId: order.id, id: editingRecord.id },
                { ...input, id: editingRecord.id },
              );
              message.success('更新放货凭证成功');
            } else {
              await orderReleasePodServiceAddReleasePod(
                { orderId: order.id },
                input,
              );
              message.success('添加放货凭证成功');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <ProFormText
            name="releaseNo"
            label="放货编号"
            placeholder="请输入放货编号 (可选)"
          />
          <ProFormText
            name="podNo"
            label="回单编号 (POD No)"
            placeholder="请输入回单编号 (可选)"
          />
          <ProFormSearchableSelect
            name="shippingDocumentId"
            label="关联提单"
            options={documentOptions}
            placeholder="请选择关联提单 (可选)"
          />
          <ProFormTextArea
            name="note"
            label="备注说明"
            placeholder="请输入备注说明 (可选)"
            fieldProps={{ maxLength: 500, showCount: true }}
          />
        </ModalForm>
      </>
    );
  },
);

export default ReleasePodPanel;
