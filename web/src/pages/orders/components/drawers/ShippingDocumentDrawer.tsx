import { PlusOutlined } from '@ant-design/icons';
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
import { ProFormSearchableSelect } from '@/components/ui';
import { Alert, App, Button, Drawer, Popconfirm, Space, Tag } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  shippingDocumentStatusValueEnum,
} from '../../common';
import {
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
  SEA_MASTER_DOCUMENT_TYPE_OPTIONS,
  SEA_MASTER_RELEASE_METHOD_OPTIONS,
  formatHouseReleaseType,
} from '../../order-plan-fields';
import {
  orderShippingDocumentServiceAddShippingDocument,
  orderShippingDocumentServiceListShippingDocuments,
  orderShippingDocumentServiceRemoveShippingDocument,
  orderShippingDocumentServiceTransitionShippingDocumentStatus,
  orderShippingDocumentServiceUpdateShippingDocument,
} from '@/services/roncin/orderShippingDocumentService';
import { toTableRequest } from '@/utils/api';

export type ShippingDocumentDrawerRef = {
  open: (order: API.Order) => void;
};

type ShippingDocumentDrawerProps = {
  canManage: boolean;
  category: string;
};

type ShippingDocumentFormValues = {
  masterNo: string;
  masterDocumentType?: string;
  masterReleaseMethod?: string;
  houseNo: string;
  releaseType?: string;
  note?: string;
};

const ShippingDocumentDrawer = forwardRef<
  ShippingDocumentDrawerRef,
  ShippingDocumentDrawerProps
>(function ShippingDocumentDrawer({ canManage, category }, ref) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [order, setOrder] = useState<API.Order>();
  const [editingShippingDocument, setEditingShippingDocument] =
    useState<API.OrderShippingDocument>();

  useImperativeHandle(ref, () => ({
    open: (record) => {
      setOrder(record);
      setDrawerOpen(true);
    },
  }));

  const openCreateShippingDocument = () => {
    setEditingShippingDocument(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEditShippingDocument = (record: API.OrderShippingDocument) => {
    setEditingShippingDocument(record);
    formRef.current?.setFieldsValue({
      masterNo: record.masterNo,
      masterDocumentType: record.masterDocumentType,
      masterReleaseMethod: record.masterReleaseMethod,
      houseNo: record.houseNo,
      releaseType: record.releaseType,
      note: record.note,
    });
    setModalOpen(true);
  };

  const columns: ProColumns<API.OrderShippingDocument>[] = [
    {
      title: '主单号 (MBL)',
      dataIndex: 'masterNo',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '分单号 (HBL)',
      dataIndex: 'houseNo',
      copyable: true,
      ellipsis: true,
    },
    ...(category === 'sea'
      ? [
          {
            title: '主单单证类型',
            dataIndex: 'masterDocumentType',
            width: 140,
            render: (_: unknown, record: API.OrderShippingDocument) =>
              record.masterDocumentType || '-',
          },
          {
            title: '主单签放方式',
            dataIndex: 'masterReleaseMethod',
            width: 140,
            render: (_: unknown, record: API.OrderShippingDocument) =>
              record.masterReleaseMethod || '-',
          },
        ]
      : []),
    {
      title: '分单签放方式',
      dataIndex: 'releaseType',
      width: 140,
      render: (_, record) =>
        record.releaseType ? (
          <Tag color="geekblue" variant="filled">
            {formatHouseReleaseType(record.releaseType)}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: '单证状态',
      dataIndex: 'status',
      width: 120,
      valueType: 'select',
      valueEnum: shippingDocumentStatusValueEnum,
      render: (_, record) =>
        record.status !== undefined &&
        shippingDocumentStatusValueEnum[record.status] ? (
          <Tag
            color={
              record.status === 3
                ? 'success'
                : record.status === 2
                  ? 'processing'
                  : 'default'
            }
            variant="filled"
          >
            {shippingDocumentStatusValueEnum[record.status]?.text}
          </Tag>
        ) : (
          '-'
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
      width: 160,
      render: (_, record) => {
        if (!canManage) return null;
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => openEditShippingDocument(record)}
            >
              编辑
            </Button>
            {record.status === 1 && (
              <Popconfirm
                title="确认将该提单状态流转为【已确认】？"
                onConfirm={async () => {
                  if (!order?.id || !record.id) return;
                  await orderShippingDocumentServiceTransitionShippingDocumentStatus(
                    {
                      orderId: order.id,
                      id: record.id,
                    },
                    {
                      orderId: order.id,
                      id: record.id,
                      expectedStatus: 1,
                      toStatus: 2,
                    },
                  );
                  message.success('提单已确认');
                  actionRef.current?.reload();
                }}
              >
                <Button type="link" size="small">
                  确认
                </Button>
              </Popconfirm>
            )}
            <Popconfirm
              title="确定移除该提单？"
              onConfirm={async () => {
                if (!order?.id || !record.id) return;
                await orderShippingDocumentServiceRemoveShippingDocument({
                  orderId: order.id,
                  id: record.id,
                });
                message.success('移除提单成功');
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
            ? `订单提单与放货 - ${order.orderNo || order.id}`
            : '订单提单与放货'
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
          <ProTable<API.OrderShippingDocument>
            actionRef={actionRef}
            rowKey="id"
            columns={columns}
            bordered
            search={false}
            pagination={false}
            request={async () => {
              const response =
                await orderShippingDocumentServiceListShippingDocuments({
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
                  onClick={openCreateShippingDocument}
                >
                  添加提单
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<ShippingDocumentFormValues>
        title={editingShippingDocument ? '编辑提单' : '添加提单'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingShippingDocument
            ? {
                masterNo: editingShippingDocument.masterNo,
                masterDocumentType: editingShippingDocument.masterDocumentType,
                masterReleaseMethod:
                  editingShippingDocument.masterReleaseMethod,
                houseNo: editingShippingDocument.houseNo,
                releaseType: editingShippingDocument.releaseType,
                note: editingShippingDocument.note,
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
          if (editingShippingDocument?.id) {
            await orderShippingDocumentServiceUpdateShippingDocument(
              {
                orderId: order.id,
                id: editingShippingDocument.id,
              },
              {
                orderId: order.id,
                id: editingShippingDocument.id,
                masterNo: values.masterNo.trim(),
                masterDocumentType: values.masterDocumentType,
                masterReleaseMethod: values.masterReleaseMethod,
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('更新提单成功');
          } else {
            await orderShippingDocumentServiceAddShippingDocument(
              {
                orderId: order.id,
              },
              {
                orderId: order.id,
                masterNo: values.masterNo.trim(),
                masterDocumentType: values.masterDocumentType,
                masterReleaseMethod: values.masterReleaseMethod,
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('添加提单成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="masterNo"
          label="主单号 (MBL)"
          placeholder="请输入主单号"
          rules={[{ required: true, message: '请输入主单号' }]}
        />
        {category === 'sea' && (
          <>
            <ProFormSearchableSelect
              name="masterDocumentType"
              label="主单单证类型"
              options={SEA_MASTER_DOCUMENT_TYPE_OPTIONS}
              placeholder="请选择主单单证类型"
              allowClear={false}
            />
            <ProFormSearchableSelect
              name="masterReleaseMethod"
              label="主单签放方式"
              options={SEA_MASTER_RELEASE_METHOD_OPTIONS}
              placeholder="请选择主单签放方式"
              allowClear={false}
            />
            <Alert
              type="warning"
              showIcon
              title="主单属性属于共享主单批次，修改后会影响其他引用同一主单的操作票。"
              style={{ marginBottom: 16 }}
            />
          </>
        )}
        <ProFormText
          name="houseNo"
          label="分单号 (HBL)"
          placeholder="请输入分单号"
          rules={[{ required: true, message: '请输入分单号' }]}
        />
        {category === 'sea' ? (
          <ProFormSearchableSelect
            name="releaseType"
            label="分单签放方式"
            options={SEA_HOUSE_RELEASE_TYPE_OPTIONS}
            placeholder="请选择分单签放方式"
          />
        ) : (
          <ProFormText
            name="releaseType"
            label="分单签放方式"
            placeholder="请输入分单签放方式"
          />
        )}
        <ProFormTextArea
          name="note"
          label="备注说明"
          placeholder="请输入备注 (可选)"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>
    </>
  );
});

export default ShippingDocumentDrawer;
