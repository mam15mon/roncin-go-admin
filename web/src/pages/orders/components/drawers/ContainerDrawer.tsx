import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { App, Button, Drawer, Popconfirm, Space, Typography } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderContainerServiceAddContainer,
  orderContainerServiceListContainers,
  orderContainerServiceRemoveContainer,
  orderContainerServiceUpdateContainer,
} from '@/services/roncin/orderContainerService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';

const { Text } = Typography;

export type ContainerDrawerRef = {
  open: (order: API.Order) => void;
};

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
  shippingDocumentId?: string;
  sealNo?: string;
  grossWeightKg: number;
  volumeCbm: number;
  note?: string;
};

const ContainerDrawer = forwardRef<ContainerDrawerRef, ContainerDrawerProps>(
  function ContainerDrawer(
    { canCreate, canUpdate, canRemove, containerSpecOptions, containerSpecMap },
    ref,
  ) {
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const activeContainerOrderIdRef = useRef<string | undefined>(undefined);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [containerDocuments, setContainerDocuments] = useState<
      API.OrderShippingDocument[]
    >([]);
    const [editingContainer, setEditingContainer] =
      useState<API.OrderContainer>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setContainerDocuments([]);
        setDrawerOpen(true);
        const orderId = record.id as string;
        activeContainerOrderIdRef.current = orderId;
        orderShippingDocumentServiceListShippingDocuments({
          orderId,
        })
          .then((res) => {
            if (activeContainerOrderIdRef.current === orderId) {
              setContainerDocuments(res.data ?? []);
            }
          })
          .catch(() => {
            if (activeContainerOrderIdRef.current === orderId) {
              setContainerDocuments([]);
            }
          });
      },
    }));

    const containerDocumentOptions = containerDocuments.map((doc) => ({
      label: `${doc.masterNo} / ${doc.houseNo}`,
      value: doc.id ?? '',
    }));

    const containerDocumentMap = Object.fromEntries(
      containerDocuments
        .filter((doc) => doc.id)
        .map((doc) => [doc.id as string, `${doc.masterNo} / ${doc.houseNo}`]),
    );

    const openCreateContainer = () => {
      setEditingContainer(undefined);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEditContainer = (record: API.OrderContainer) => {
      setEditingContainer(record);
      formRef.current?.setFieldsValue({
        containerNo: record.containerNo,
        containerSpecId: record.containerSpecId,
        shippingDocumentId: record.shippingDocumentId,
        sealNo: record.sealNo,
        grossWeightKg: record.grossWeightKg,
        volumeCbm: record.volumeCbm,
        note: record.note,
      });
      setModalOpen(true);
    };

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
            ? containerSpecMap[record.containerSpecId] || record.containerSpecId
            : '-',
      },
      {
        title: '关联提单',
        dataIndex: 'shippingDocumentId',
        width: 180,
        ellipsis: true,
        render: (_, record) =>
          record.shippingDocumentId ? (
            containerDocumentMap[record.shippingDocumentId] ||
            record.shippingDocumentId
          ) : (
            <Text type="secondary">未关联</Text>
          ),
      },
      {
        title: '铅封号',
        dataIndex: 'sealNo',
        copyable: true,
        ellipsis: true,
        render: (_, record) => record.sealNo || '-',
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
      {
        title: '操作',
        valueType: 'option',
        width: 120,
        render: (_, record) => {
          if (!canUpdate && !canRemove) return null;
          return (
            <Space size="small">
              {canUpdate && (
                <Button
                  type="link"
                  size="small"
                  onClick={() => openEditContainer(record)}
                >
                  编辑
                </Button>
              )}
              {canRemove && (
                <Popconfirm
                  title="确定移除该集装箱？"
                  onConfirm={async () => {
                    if (!order?.id || !record.id) return;
                    await orderContainerServiceRemoveContainer({
                      orderId: order.id,
                      id: record.id,
                    });
                    message.success('移除集装箱成功');
                    actionRef.current?.reload();
                  }}
                >
                  <Button type="link" danger size="small">
                    删除
                  </Button>
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
          title={
            order
              ? `订单集装箱列表 - ${order.orderNo || order.id}`
              : '订单集装箱列表'
          }
          open={drawerOpen}
          onClose={() => {
            activeContainerOrderIdRef.current = undefined;
            setDrawerOpen(false);
            setOrder(undefined);
            setContainerDocuments([]);
          }}
          size={920}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderContainer>
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderContainerServiceListContainers({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canCreate && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreateContainer}
                  >
                    添加集装箱
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<ContainerFormValues>
          title={editingContainer ? '编辑集装箱' : '添加集装箱'}
          open={modalOpen}
          formRef={formRef}
          initialValues={
            editingContainer
              ? {
                  containerNo: editingContainer.containerNo,
                  containerSpecId: editingContainer.containerSpecId,
                  shippingDocumentId: editingContainer.shippingDocumentId,
                  sealNo: editingContainer.sealNo,
                  grossWeightKg: editingContainer.grossWeightKg,
                  volumeCbm: editingContainer.volumeCbm,
                  note: editingContainer.note,
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
            if (editingContainer?.id) {
              await orderContainerServiceUpdateContainer(
                {
                  orderId: order.id,
                  id: editingContainer.id,
                },
                {
                  id: editingContainer.id,
                  orderId: order.id,
                  containerNo: values.containerNo.trim(),
                  containerSpecId: values.containerSpecId,
                  shippingDocumentId: values.shippingDocumentId || undefined,
                  sealNo: values.sealNo?.trim() || undefined,
                  grossWeightKg: Number(values.grossWeightKg),
                  volumeCbm: Number(values.volumeCbm),
                  note: values.note?.trim() || undefined,
                },
              );
              message.success('更新集装箱成功');
            } else {
              await orderContainerServiceAddContainer(
                {
                  orderId: order.id,
                },
                {
                  orderId: order.id,
                  containerNo: values.containerNo.trim(),
                  containerSpecId: values.containerSpecId,
                  shippingDocumentId: values.shippingDocumentId || undefined,
                  sealNo: values.sealNo?.trim() || undefined,
                  grossWeightKg: Number(values.grossWeightKg),
                  volumeCbm: Number(values.volumeCbm),
                  note: values.note?.trim() || undefined,
                },
              );
              message.success('添加集装箱成功');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
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
          <ProFormSearchableSelect
            name="shippingDocumentId"
            label="关联提单"
            options={containerDocumentOptions}
            placeholder="请选择关联提单 (可选)"
          />
          <ProFormText
            name="sealNo"
            label="铅封号"
            placeholder="请输入铅封号 (可选)"
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
        </ModalForm>
      </>
    );
  },
);

export default ContainerDrawer;
