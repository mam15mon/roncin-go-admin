import { BankOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { Alert, App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceCreatePartnerAccount,
  partnerServiceListPartnerAccounts,
  partnerServiceUpdatePartnerAccount,
} from '@/services/roncin/partnerService';

const { Text } = Typography;

const accountStatusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
];

type AccountFormValues = API.PartnerAccountInput;

type AccountsPanelProps = {
  partner?: API.Partner;
  canManage: boolean;
};

export default function AccountsPanel({
  partner,
  canManage,
}: AccountsPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<API.PartnerAccount>();

  const hasCustomerRole =
    partner?.roles?.some((role) => role.type === 1) ?? false;

  const openForm = (account?: API.PartnerAccount) => {
    setEditingAccount(account);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const columns: ProColumns<API.PartnerAccount>[] = [
    {
      title: '结算币种',
      dataIndex: 'currency',
      width: 100,
      render: (cur) => (
        <Tag color="gold" variant="filled">
          {cur}
        </Tag>
      ),
    },
    { title: '开户银行', dataIndex: 'bankName', ellipsis: true },
    {
      title: '银行账号',
      dataIndex: 'bankAccount',
      copyable: true,
      ellipsis: true,
      render: (acc) => <Text style={{ fontFamily: 'monospace' }}>{acc}</Text>,
    },
    {
      title: '默认账户',
      dataIndex: 'isDefault',
      width: 90,
      render: (_, record) =>
        record.isDefault ? (
          <Tag color="blue">默认</Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) =>
        record.status === 1 ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 170,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      fixed: 'right',
      render: (_, record) =>
        canManage ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => openForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  if (!hasCustomerRole) {
    return (
      <Alert
        showIcon
        type="info"
        title="无需结算账户配置"
        description="该往来单位当前未分配客户角色，仅客户身份支持配置结算银行账户。"
        style={{ margin: '16px 0' }}
      />
    );
  }

  return (
    <>
      <ProTable<API.PartnerAccount>
        headerTitle={
          <Space size={6}>
            <BankOutlined style={{ color: '#1677ff' }} />
            <span>客户结算账户列表</span>
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
          const response = await partnerServiceListPartnerAccounts({
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
                  新增结算账户
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<AccountFormValues>
        title={editingAccount ? '编辑结算账户' : '新增结算账户'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingAccount ?? { currency: 'CNY', status: 1, isDefault: false }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (!partner?.id) return false;
          const account: API.PartnerAccountInput = {
            ...values,
            currency: values.currency?.trim() ?? 'CNY',
          };
          if (editingAccount?.id) {
            await partnerServiceUpdatePartnerAccount(
              { partnerId: partner.id, id: editingAccount.id },
              { partnerId: partner.id, id: editingAccount.id, account },
            );
            message.success('结算账户已成功更新');
          } else {
            await partnerServiceCreatePartnerAccount(
              { partnerId: partner.id },
              { partnerId: partner.id, account },
            );
            message.success('结算账户已成功创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <Space
          align="start"
          wrap
          size={16}
          style={{ width: '100%', marginBottom: 12 }}
        >
          <ProFormText
            name="currency"
            label="结算币种"
            width="sm"
            placeholder="如 CNY、USD"
            rules={[{ required: true, len: 3, message: '请输入三位币种代码' }]}
          />
          <ProFormSelect
            name="status"
            label="账户状态"
            width="sm"
            options={accountStatusOptions}
            rules={[{ required: true }]}
          />
          <ProFormSwitch name="isDefault" label="设为默认结算账户" />
        </Space>
        <ProFormText
          name="bankName"
          label="开户银行名称及支行"
          placeholder="例如：中国工商银行上海自贸试验区分行"
        />
        <ProFormText
          name="bankAccount"
          label="银行开户账号"
          placeholder="请输入银行结算账号"
        />
        <ProFormText
          name="swiftCode"
          label="SWIFT Code (外币国际结算)"
          placeholder="例如：ICBKCNBS"
        />
        <ProFormTextArea
          name="remark"
          label="备注说明"
          placeholder="请输入其他结算说明"
          fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
        />
      </ModalForm>
    </>
  );
}
