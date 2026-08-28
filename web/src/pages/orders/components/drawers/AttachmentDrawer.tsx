import type { ProColumns } from '@ant-design/pro-components';
import { ProFormDigit, ProFormText } from '@ant-design/pro-components';
import { Alert, Tag, Typography } from 'antd';
import React, { forwardRef } from 'react';
import {
  SubEntityDrawerTemplate,
  type SubEntityDrawerRef,
} from '@/components/ui/sub-entity-drawer';
import {
  orderAttachmentServiceListAttachments,
  orderAttachmentServiceRegisterAttachment,
} from '@/services/roncin/orderAttachmentService';

const { Text } = Typography;

export type AttachmentDrawerRef = SubEntityDrawerRef<API.Order>;

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

const AttachmentDrawer = forwardRef<
  AttachmentDrawerRef,
  AttachmentDrawerProps
>(function AttachmentDrawer({ canRegister }, ref) {
  return (
    <SubEntityDrawerTemplate<
      API.OrderAttachment,
      API.Order,
      AttachmentFormValues
    >
      ref={ref}
      entityName="附件"
      drawerTitle={(order) =>
        order
          ? `订单附件档案 - ${order.orderNo || order.id}`
          : '订单附件档案'
      }
      canCreate={canRegister}
      canUpdate={false}
      canRemove={false}
      columns={columns}
      fetchList={(order) =>
        orderAttachmentServiceListAttachments({
          orderId: order.id as string,
        })
      }
      createItem={(values, order) =>
        orderAttachmentServiceRegisterAttachment(
          { orderId: order.id as string },
          {
            orderId: order.id as string,
            docType: values.docType.trim(),
            idempotencyKey: values.idempotencyKey.trim(),
            fileName: values.fileName.trim(),
            mimeType: values.mimeType.trim(),
            fileSize: String(values.fileSize),
            objectKey: values.objectKey.trim(),
            checksum: values.checksum?.trim() || undefined,
          },
        )
      }
      renderFormItems={() => (
        <>
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
        </>
      )}
    />
  );
});

export default AttachmentDrawer;
