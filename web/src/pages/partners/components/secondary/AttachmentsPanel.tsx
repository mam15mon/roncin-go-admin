import { PaperClipOutlined, PlusOutlined } from '@ant-design/icons';
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
import { Alert, App, Button, Space, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceListPartnerAttachments,
  partnerServiceRegisterPartnerAttachment,
} from '@/services/roncin/partnerService';
import { toTableRequest } from '@/utils/api';

const { Text } = Typography;

type AttachmentFormValues = {
  fileName?: string;
  mimeType?: string;
  fileSize?: string | number;
  objectKey?: string;
  checksum?: string;
  idempotencyKey?: string;
};

type AttachmentsPanelProps = {
  partner?: API.Partner;
  canManage: boolean;
};

export default function AttachmentsPanel({
  partner,
  canManage,
}: AttachmentsPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);

  const openForm = () => {
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const columns: ProColumns<API.PartnerAttachment>[] = [
    {
      title: '文件名',
      dataIndex: 'fileName',
      ellipsis: true,
      render: (name) => <Text strong>{name}</Text>,
    },
    { title: 'MIME 类型', dataIndex: 'mimeType', width: 140, ellipsis: true },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      width: 110,
      render: (s) => `${s} 字节`,
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
      title: '幂等键',
      dataIndex: 'idempotencyKey',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 170,
    },
  ];

  return (
    <>
      <ProTable<API.PartnerAttachment>
        headerTitle={
          <Space size={6}>
            <PaperClipOutlined style={{ color: '#1677ff' }} />
            <span>往来单位附件与证照</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        search={false}
        pagination={false}
        request={async () => {
          if (!partner?.id) return { data: [], success: true };
          const response = await partnerServiceListPartnerAttachments({
            partnerId: partner.id,
          });
          return toTableRequest(response);
        }}
        toolBarRender={() =>
          canManage
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openForm}
                >
                  登记新附件
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<AttachmentFormValues>
        title="登记往来单位证照与附件"
        open={modalOpen}
        formRef={formRef}
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (
            !partner?.id ||
            !values.fileName ||
            !values.mimeType ||
            !values.fileSize ||
            !values.objectKey ||
            !values.idempotencyKey
          ) {
            return false;
          }
          await partnerServiceRegisterPartnerAttachment(
            { partnerId: partner.id },
            {
              partnerId: partner.id,
              fileName: values.fileName.trim(),
              mimeType: values.mimeType.trim(),
              fileSize: String(values.fileSize),
              objectKey: values.objectKey.trim(),
              checksum: values.checksum?.trim() || undefined,
              idempotencyKey: values.idempotencyKey.trim(),
            },
          );
          message.success('证照附件登记成功');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          title="此处登记企业营业执照、开户许可证、水运许可证等对象存储引用。"
          style={{ marginBottom: 16 }}
        />
        <ProFormText
          name="fileName"
          label="附件名称"
          placeholder="例如: 营业执照扫描件.pdf"
          rules={[{ required: true, message: '请输入文件名' }]}
        />
        <ProFormText
          name="mimeType"
          label="MIME 类型"
          placeholder="例如: application/pdf 或 image/jpeg"
          rules={[{ required: true, message: '请输入 MIME 类型' }]}
        />
        <ProFormDigit
          name="fileSize"
          label="文件字节数 (Byte)"
          min={1}
          fieldProps={{ precision: 0 }}
          placeholder="请输入文件大小"
          rules={[{ required: true, message: '请输入文件大小' }]}
        />
        <ProFormText
          name="objectKey"
          label="对象存储标识键 (Object Key)"
          placeholder="例如: partners/licenses/cust001_license.pdf"
          rules={[{ required: true, message: '请输入对象键' }]}
        />
        <ProFormText
          name="checksum"
          label="SHA256 校验和"
          placeholder="请输入文件哈希校验和 (可选)"
        />
        <ProFormText
          name="idempotencyKey"
          label="幂等键"
          placeholder="请输入请求幂等键"
          rules={[{ required: true, message: '请输入幂等键' }]}
        />
      </ModalForm>
    </>
  );
}
