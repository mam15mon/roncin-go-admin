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
import { OrderShippingDocumentStatus } from '@/enums.generated';
import { shippingDocumentStatusValueEnum } from '../../common';
import {
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
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
      houseNo: record.houseNo,
      releaseType: record.releaseType,
      note: record.note,
    });
    setModalOpen(true);
  };

  const columns: ProColumns<API.OrderShippingDocument>[] = [
    {
      title: '分单号 (HBL)',
      dataIndex: 'houseNo',
      copyable: true,
      ellipsis: true,
    },
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
      render: (_, record) => {
        const status =
          record.status ??
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_UNSPECIFIED;
        const color =
          status ===
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED
            ? 'green'
            : status ===
                OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED
              ? 'blue'
              : 'orange';
        const label =
          (
            shippingDocumentStatusValueEnum[
              status as unknown as keyof typeof shippingDocumentStatusValueEnum
            ] as { text?: string } | undefined
          )?.text ?? String(status);
        return (
          <Tag color={color} variant="filled">
            {label}
          </Tag>
        );
      },
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
      width: 170,
      valueType: 'dateTime',
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 170,
      valueType: 'dateTime',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 180,
      render: (_, record) => {
        if (!canManage || !order?.id || !record.id) return null;
        const status =
          record.status ??
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_UNSPECIFIED;
        const isDraft =
          status ===
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT;
        const isConfirmed =
          status ===
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED;
        const isReleased =
          status ===
          OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED;

        return (
          <Space size="small">
            {!isReleased && (
              <a
                onClick={() => {
                  openEditShippingDocument(record);
                }}
              >
                编辑
              </a>
            )}
            {isDraft && (
              <Popconfirm
                title="确认提单"
                description="确认后提单将进入已确认状态，确定继续？"
                onConfirm={async () => {
                  await orderShippingDocumentServiceTransitionShippingDocumentStatus(
                    {
                      orderId: order.id as string,
                      id: record.id as string,
                    },
                    {
                      orderId: order.id as string,
                      id: record.id as string,
                      expectedStatus:
                        OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT,
                      toStatus:
                        OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED,
                    },
                  );
                  message.success('提单已确认');
                  actionRef.current?.reload();
                }}
              >
                <a style={{ color: '#1677ff' }}>确认</a>
              </Popconfirm>
            )}
            {isConfirmed && (
              <Popconfirm
                title="放行提单"
                description="放行后提单将无法再次编辑或删除，确定继续？"
                onConfirm={async () => {
                  await orderShippingDocumentServiceTransitionShippingDocumentStatus(
                    {
                      orderId: order.id as string,
                      id: record.id as string,
                    },
                    {
                      orderId: order.id as string,
                      id: record.id as string,
                      expectedStatus:
                        OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED,
                      toStatus:
                        OrderShippingDocumentStatus.ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED,
                    },
                  );
                  message.success('提单已放行');
                  actionRef.current?.reload();
                }}
              >
                <a style={{ color: '#52c41a' }}>放行</a>
              </Popconfirm>
            )}
            {!isReleased && (
              <Popconfirm
                title="删除提单"
                description="确定要删除该分单吗？"
                onConfirm={async () => {
                  await orderShippingDocumentServiceRemoveShippingDocument({
                    orderId: order.id as string,
                    id: record.id as string,
                  });
                  message.success('删除提单成功');
                  actionRef.current?.reload();
                }}
              >
                <a style={{ color: '#ff4d4f' }}>删除</a>
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
        title="分单管理 (HBL)"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={960}
      >
        {category === 'sea' && (
          <Alert
            type="info"
            showIcon
            message={
              <span>
                当前订单关联海运主单 (MBL)：
                <strong>{order?.seaMasterBill?.masterNo || '未录入'}</strong>
                {order?.seaMasterBill?.issuerPartnerName && (
                  <span>
                    {' '}
                    (实际签发主体: {order.seaMasterBill.issuerPartnerName})
                  </span>
                )}
              </span>
            }
            style={{ marginBottom: 16 }}
          />
        )}

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
                  添加分单 (HBL)
                </Button>
              ),
            ]}
          />
        )}
      </Drawer>

      <ModalForm<ShippingDocumentFormValues>
        title={editingShippingDocument ? '编辑分单 (HBL)' : '添加分单 (HBL)'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingShippingDocument
            ? {
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
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('更新分单成功');
          } else {
            await orderShippingDocumentServiceAddShippingDocument(
              {
                orderId: order.id,
              },
              {
                orderId: order.id,
                houseNo: values.houseNo.trim(),
                releaseType: values.releaseType?.trim() || undefined,
                note: values.note?.trim() || undefined,
              },
            );
            message.success('添加分单成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
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
            label="签放方式"
            placeholder="请输入签放方式"
          />
        )}
        <ProFormTextArea
          name="note"
          label="备注"
          placeholder="请输入备注"
          fieldProps={{ rows: 3, maxLength: 500 }}
        />
      </ModalForm>
    </>
  );
});

export default ShippingDocumentDrawer;
