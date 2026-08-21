import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Drawer, Popconfirm, Space, Tag } from 'antd';
import dayjs from 'dayjs';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderReleasePodServiceAddReleasePod,
  orderReleasePodServiceListReleasePods,
  orderReleasePodServiceRemoveReleasePod,
  orderReleasePodServiceTransitionReleasePodStatus,
  orderReleasePodServiceUpdateReleasePod,
} from '@/services/roncin/orderReleasePodService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';

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
  { text: string; status: 'Default' | 'Processing' | 'Success' }
> = {
  1: { text: '待签收', status: 'Default' },
  2: { text: '已签收', status: 'Processing' },
  3: { text: '已回单', status: 'Success' },
};

export function getReleasePodTransition(status?: number) {
  if (status === 1) {
    return { currentText: '待签收', nextText: '已签收', toStatus: 2 };
  }
  if (status === 2) {
    return { currentText: '已签收', nextText: '已回单', toStatus: 3 };
  }
  return undefined;
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
    const [editingRecord, setEditingRecord] = useState<API.OrderReleasePod>();
    const activeOrderIdRef = useRef<string | undefined>(undefined);

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDocuments([]);
        setDrawerOpen(true);
        const orderId = record.id as string;
        activeOrderIdRef.current = orderId;
        orderShippingDocumentServiceListShippingDocuments({
          orderId,
        })
          .then((response) => {
            if (activeOrderIdRef.current === orderId) {
              setDocuments(response.data ?? []);
            }
          })
          .catch(() => {
            if (activeOrderIdRef.current === orderId) {
              setDocuments([]);
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
        render: (_, record) => record.releaseNo || '-',
      },
      {
        title: '回单编号',
        dataIndex: 'podNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => record.podNo || '-',
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
        valueEnum: releasePodStatusValueEnum,
        render: (_, record) => {
          const status = releasePodStatusValueEnum[record.status ?? 0];
          return status ? <Tag color={status.status}>{status.text}</Tag> : '-';
        },
      },
      {
        title: '签收时间',
        dataIndex: 'signedAt',
        valueType: 'dateTime',
        width: 180,
        render: (_, record) =>
          record.signedAt
            ? dayjs(record.signedAt).format('YYYY-MM-DD HH:mm:ss')
            : '-',
      },
      {
        title: '签收人',
        dataIndex: 'signedBy',
        copyable: true,
        ellipsis: true,
        render: (_, record) => record.signedBy || '-',
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
        search: false,
        width: 180,
        render: (_, record) => {
          const transition = getReleasePodTransition(record.status);
          if (!canManage || !transition) return null;
          return (
            <Space size="small">
              <Button type="link" size="small" onClick={() => openEdit(record)}>
                编辑
              </Button>
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
              ? `订单放货凭证 - ${order.orderNo || order.id}`
              : '订单放货凭证'
          }
          open={drawerOpen}
          onClose={() => {
            activeOrderIdRef.current = undefined;
            setDrawerOpen(false);
            setOrder(undefined);
            setDocuments([]);
          }}
          width={900}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderReleasePod>
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderReleasePodServiceListReleasePods({
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
            label="回单编号"
            placeholder="请输入回单编号 (可选)"
          />
          <ProFormSelect
            name="shippingDocumentId"
            label="关联提单"
            options={documentOptions}
            placeholder="请选择关联提单 (可选)"
          />
          <ProFormTextArea
            name="note"
            label="备注"
            placeholder="请输入备注 (可选)"
            fieldProps={{ maxLength: 500, showCount: true }}
          />
        </ModalForm>
      </>
    );
  },
);

export default ReleasePodPanel;
