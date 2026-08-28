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
  ProTable,
} from '@ant-design/pro-components';
import { Alert, App, Button, Drawer, Tag, Typography } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import {
  orderAttachmentServiceListAttachments,
  orderAttachmentServiceRegisterAttachment,
} from '@/services/roncin/orderAttachmentService';

const { Text } = Typography;

export type AttachmentDrawerRef = {
  open: (order: API.Order) => void;
};

type AttachmentDrawerProps = {
  canRegister: boolean;
};

type AttachmentFormValues = {
  docType: string;
  idempotencyKey: string;
  fileName: string;
  mimeType: string;
  fileSize: number;
  objectKey: string;
  checksum?: string;
};

const AttachmentDrawer = forwardRef<AttachmentDrawerRef, AttachmentDrawerProps>(
  function AttachmentDrawer({ canRegister }, ref) {
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const formRef = useRef<ProFormInstance | undefined>(undefined);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
      },
    }));

    const openRegisterAttachment = () => {
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const columns: ProColumns<API.OrderAttachment>[] = [
      {
        title: '文档类型',
        dataIndex: 'docType',
        width: 140,
        render: (type) => (
          <Tag color="geekblue" variant="filled">
            {type}
          </Tag>
        ),
      },
      {
        title: '文件名',
        dataIndex: 'fileName',
        ellipsis: true,
        render: (name) => <Text strong>{name}</Text>,
      },
      {
        title: 'MIME 类型',
        dataIndex: 'mimeType',
        width: 140,
        ellipsis: true,
      },
      {
        title: '文件大小',
        dataIndex: 'fileSize',
        width: 120,
        render: (size) => `${size} 字节`,
      },
      {
        title: '对象键',
        dataIndex: 'objectKey',
        copyable: true,
        ellipsis: true,
        render: (key) => (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{key}</Text>
        ),
      },
      {
        title: '校验和',
        dataIndex: 'checksum',
        copyable: true,
        ellipsis: true,
      },
      {
        title: '登记时间',
        dataIndex: 'createdAt',
        valueType: 'dateTime',
        width: 180,
      },
    ];

    return (
      <>
        <Drawer
          title={
            order
              ? `订单附件档案 - ${order.orderNo || order.id}`
              : '订单附件档案'
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
            <ProTable<API.OrderAttachment>
              actionRef={actionRef}
              rowKey={(record) => record.id || record.objectKey || ''}
              columns={columns}
              bordered
              search={false}
              pagination={false}
              request={async () => {
                const response = await orderAttachmentServiceListAttachments({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canRegister && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openRegisterAttachment}
                  >
                    登记附件
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<AttachmentFormValues>
          title="登记订单附件"
          open={modalOpen}
          formRef={formRef}
          modalProps={{
            destroyOnHidden: true,
            width: 560,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onFinish={async (values) => {
            if (!order?.id) return false;
            await orderAttachmentServiceRegisterAttachment(
              { orderId: order.id },
              {
                orderId: order.id,
                docType: values.docType.trim(),
                idempotencyKey: values.idempotencyKey.trim(),
                fileName: values.fileName.trim(),
                mimeType: values.mimeType.trim(),
                fileSize: String(values.fileSize),
                objectKey: values.objectKey.trim(),
                checksum: values.checksum?.trim() || undefined,
              },
            );
            message.success('登记附件成功');
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <Alert
            type="info"
            showIcon
            title="此处登记外部对象存储引用与元数据，不直接进行二进制文件上传。"
            style={{ marginBottom: 16 }}
          />
          <ProFormText
            name="docType"
            label="文档类型"
            placeholder="请输入文档类型 (如 BL, INVOICE, PACKING_LIST)"
            rules={[{ required: true, message: '请输入文档类型' }]}
          />
          <ProFormText
            name="idempotencyKey"
            label="幂等键"
            placeholder="请输入幂等键"
            rules={[{ required: true, message: '请输入幂等键' }]}
          />
          <ProFormText
            name="fileName"
            label="文件名"
            placeholder="请输入文件名"
            rules={[{ required: true, message: '请输入文件名' }]}
          />
          <ProFormText
            name="mimeType"
            label="MIME 类型"
            placeholder="请输入 MIME 类型 (如 application/pdf)"
            rules={[{ required: true, message: '请输入 MIME 类型' }]}
          />
          <ProFormDigit
            name="fileSize"
            label="文件大小 (字节)"
            min={1}
            fieldProps={{ precision: 0 }}
            placeholder="请输入文件大小"
            rules={[{ required: true, message: '请输入文件大小' }]}
          />
          <ProFormText
            name="objectKey"
            label="对象存储键 (Object Key)"
            placeholder="请输入对象存储键"
            rules={[{ required: true, message: '请输入对象键' }]}
          />
          <ProFormText
            name="checksum"
            label="校验和 (Checksum)"
            placeholder="请输入校验和 (可选)"
          />
        </ModalForm>
      </>
    );
  },
);

export default AttachmentDrawer;
