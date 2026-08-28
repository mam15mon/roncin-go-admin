import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { Alert, App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceCreatePartnerInvoiceProfile,
  partnerServiceListPartnerInvoiceProfiles,
  partnerServiceUpdatePartnerInvoiceProfile,
} from '@/services/roncin/partnerService';

const { Text } = Typography;

type InvoiceProfileFormValues = {
  invoiceTitle?: string;
  taxpayerIdentificationNo?: string;
  registeredAddress?: string;
  registeredPhone?: string;
  bankName?: string;
  bankAccount?: string;
  defaultInvoiceType?: string;
  isDefault?: boolean;
  enabled?: boolean;
};

type InvoiceProfilesPanelProps = {
  partner?: API.Partner;
  canManage: boolean;
};

export default function InvoiceProfilesPanel({
  partner,
  canManage,
}: InvoiceProfilesPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<
    ProFormInstance<InvoiceProfileFormValues> | undefined
  >(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingProfile, setEditingProfile] =
    useState<API.PartnerInvoiceProfile>();

  const openForm = (profile?: API.PartnerInvoiceProfile) => {
    setEditingProfile(profile);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        title="同一往来单位可以维护多套开票抬头；默认抬头仅用于预选，创建发票时仍会明确选择并固化资料快照。"
      />
      <ProTable<API.PartnerInvoiceProfile>
        headerTitle="开票抬头"
        rowKey="id"
        actionRef={actionRef}
        search={false}
        pagination={false}
        bordered
        size="small"
        columns={[
          {
            title: '发票抬头',
            dataIndex: 'invoiceTitle',
            width: 220,
            ellipsis: true,
          },
          {
            title: '纳税人识别号',
            dataIndex: 'taxpayerIdentificationNo',
            width: 190,
            render: (_, row) => (
              <Text copyable>{row.taxpayerIdentificationNo}</Text>
            ),
          },
          {
            title: '默认票种',
            dataIndex: 'defaultInvoiceType',
            width: 100,
            renderText: (value) =>
              value === 'SPECIAL' ? '专用发票' : '普通发票',
          },
          {
            title: '默认',
            dataIndex: 'isDefault',
            width: 70,
            render: (_, row) =>
              row.isDefault ? <Tag color="blue">默认</Tag> : '-',
          },
          {
            title: '状态',
            dataIndex: 'enabled',
            width: 70,
            render: (_, row) => (
              <Tag color={row.enabled ? 'success' : 'default'}>
                {row.enabled ? '启用' : '停用'}
              </Tag>
            ),
          },
          {
            title: '操作',
            valueType: 'option',
            width: 80,
            render: (_, row) =>
              canManage
                ? [
                    <a key="edit" onClick={() => openForm(row)}>
                      <EditOutlined /> 编辑
                    </a>,
                  ]
                : [],
          },
        ]}
        request={async () => {
          if (!partner?.id) return { data: [], success: true };
          const response = await partnerServiceListPartnerInvoiceProfiles({
            partnerId: partner.id,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
          };
        }}
        toolBarRender={() =>
          canManage
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => openForm()}
                >
                  新增开票抬头
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<InvoiceProfileFormValues>
        title={editingProfile ? '编辑开票抬头' : '新增开票抬头'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingProfile ?? {
            invoiceTitle: partner?.legalName,
            taxpayerIdentificationNo: partner?.unifiedSocialCreditCode,
            registeredAddress: partner?.registeredAddress,
            defaultInvoiceType: 'NORMAL',
            isDefault: false,
            enabled: true,
          }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (!partner?.id) return false;
          const common = {
            partnerId: partner.id,
            invoiceTitle: values.invoiceTitle?.trim() || '',
            taxpayerIdentificationNo:
              values.taxpayerIdentificationNo?.trim() || '',
            registeredAddress: values.registeredAddress?.trim() || '',
            registeredPhone: values.registeredPhone?.trim() || '',
            bankName: values.bankName?.trim() || '',
            bankAccount: values.bankAccount?.trim() || '',
            defaultInvoiceType: values.defaultInvoiceType || 'NORMAL',
            isDefault: Boolean(values.isDefault),
          };
          if (editingProfile?.id) {
            await partnerServiceUpdatePartnerInvoiceProfile(
              { partnerId: partner.id, id: editingProfile.id },
              {
                ...common,
                id: editingProfile.id,
                enabled: Boolean(values.enabled),
                expectedVersion: editingProfile.version || '0',
              },
            );
            message.success('开票抬头已更新，历史发票快照不受影响');
          } else {
            await partnerServiceCreatePartnerInvoiceProfile(
              { partnerId: partner.id },
              common,
            );
            message.success('开票抬头已新增');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="这里维护的是税务开票主体资料，不是按币种维护的收付款结算账户"
        />
        <Space size={24} style={{ marginBottom: 8 }}>
          <ProFormSwitch name="isDefault" label="设为默认抬头" />
          {editingProfile && <ProFormSwitch name="enabled" label="启用该抬头" />}
        </Space>
        <ProFormText
          name="invoiceTitle"
          label="发票抬头"
          rules={[
            { required: true, whitespace: true, message: '请输入发票抬头' },
          ]}
        />
        <ProFormText
          name="taxpayerIdentificationNo"
          label="纳税人识别号"
          rules={[
            { required: true, whitespace: true, message: '请输入纳税人识别号' },
          ]}
        />
        <ProFormSelect
          name="defaultInvoiceType"
          label="默认发票类型"
          options={[
            { label: '增值税普通发票', value: 'NORMAL' },
            { label: '增值税专用发票', value: 'SPECIAL' },
          ]}
          rules={[{ required: true, message: '请选择默认发票类型' }]}
        />
        <ProFormDependency name={['defaultInvoiceType']}>
          {({ defaultInvoiceType }) => {
            const required = defaultInvoiceType === 'SPECIAL';
            return (
              <>
                <ProFormText
                  name="registeredAddress"
                  label="税务登记地址"
                  rules={
                    required
                      ? [{ required: true, message: '专票必须填写登记地址' }]
                      : []
                  }
                />
                <ProFormText
                  name="registeredPhone"
                  label="税务登记电话"
                  rules={
                    required
                      ? [{ required: true, message: '专票必须填写登记电话' }]
                      : []
                  }
                />
                <ProFormText
                  name="bankName"
                  label="开户银行"
                  rules={
                    required
                      ? [{ required: true, message: '专票必须填写开户银行' }]
                      : []
                  }
                />
                <ProFormText
                  name="bankAccount"
                  label="银行账号"
                  rules={
                    required
                      ? [{ required: true, message: '专票必须填写银行账号' }]
                      : []
                  }
                />
              </>
            );
          }}
        </ProFormDependency>
      </ModalForm>
    </>
  );
}
