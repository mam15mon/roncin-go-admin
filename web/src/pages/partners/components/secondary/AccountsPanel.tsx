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
import { App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import { PartnerAccountUsage } from '@/enums.generated';
import {
  partnerServiceCreatePartnerAccount,
  partnerServiceListPartnerAccounts,
  partnerServiceUpdatePartnerAccount,
} from '@/services/roncin/partnerService';
import { toTableRequest } from '@/utils/api';

const { Text } = Typography;

const usageOptions = [
  {
    label: '应收',
    value: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_RECEIVABLE,
  },
  {
    label: '应付',
    value: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_PAYABLE,
  },
  {
    label: '应收及应付',
    value: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_BOTH,
  },
];

function usageText(usage?: number) {
  return usageOptions.find((item) => item.value === usage)?.label || '-';
}

type AccountsPanelProps = {
  partner?: API.Partner;
  canRead: boolean;
  canCreate: boolean;
  canUpdate: boolean;
};

/**
 * 往来单位结算账户是公司主体主数据，不因客户/供应商等角色重复维护。
 * 账户读取、创建和更新分别使用账户专属权限；账单用途候选由财务 RPC 另行处理。
 */
export default function AccountsPanel({
  partner,
  canRead,
  canCreate,
  canUpdate,
}: AccountsPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<API.PartnerAccount>();

  const openForm = (account?: API.PartnerAccount) => {
    setEditingAccount(account);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const columns: ProColumns<API.PartnerAccount>[] = [
    { title: '账户名称', dataIndex: 'name', width: 150, ellipsis: true },
    { title: '户名', dataIndex: 'accountHolder', width: 160, ellipsis: true },
    {
      title: '用途',
      dataIndex: 'usage',
      width: 115,
      render: (value) => (
        <Tag color="blue">
          {usageText(typeof value === 'number' ? value : undefined)}
        </Tag>
      ),
    },
    {
      title: '结算币种',
      dataIndex: 'currency',
      width: 100,
      render: (currency) => <Tag color="gold">{currency}</Tag>,
    },
    { title: '开户银行', dataIndex: 'bankName', width: 190, ellipsis: true },
    {
      title: '银行账号',
      dataIndex: 'accountNo',
      width: 180,
      copyable: true,
      ellipsis: true,
      render: (accountNo) => (
        <Text style={{ fontFamily: 'monospace' }}>{accountNo}</Text>
      ),
    },
    {
      title: '默认用途',
      width: 135,
      render: (_, record) => (
        <Space size={4} wrap>
          {record.isDefaultReceivable && <Tag color="green">应收默认</Tag>}
          {record.isDefaultPayable && <Tag color="volcano">应付默认</Tag>}
          {!record.isDefaultReceivable && !record.isDefaultPayable && '-'}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 90,
      render: (enabled) =>
        enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
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
        canUpdate ? (
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

  const columnSettings = useColumnSettings<ProColumns<API.PartnerAccount>>({
    tableKey: 'partners:accounts',
    columns,
  });

  if (!canRead) {
    return <Text type="secondary">暂无结算账户查看权限。</Text>;
  }

  return (
    <>
      <ProTable<API.PartnerAccount>
        headerTitle={
          <Space size={6}>
            <BankOutlined style={{ color: '#1677ff' }} />
            <span>结算账户列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columnSettings.columns}
        bordered
        search={false}
        pagination={false}
        options={{ reload: true, density: true, setting: false }}
        request={async () => {
          if (!partner?.id) return { data: [], success: true };
          const response = await partnerServiceListPartnerAccounts({
            partnerId: partner.id,
          });
          return toTableRequest(response);
        }}
        toolBarRender={() => [
          columnSettings.entry,
          ...(canCreate
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
            : []),
        ]}
      />

      <ModalForm<API.PartnerAccountInput>
        title={editingAccount ? '编辑结算账户' : '新增结算账户'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingAccount ?? {
            currency: 'CNY',
            usage: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_BOTH,
            enabled: true,
            isDefaultReceivable: false,
            isDefaultPayable: false,
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
          const account: API.PartnerAccountInput = {
            ...values,
            name: values.name.trim(),
            accountHolder: values.accountHolder.trim(),
            currency: values.currency.trim(),
            bankName: values.bankName.trim(),
            accountNo: values.accountNo.trim(),
            swiftCode: values.swiftCode?.trim() || undefined,
            remark: values.remark?.trim() || undefined,
          };
          try {
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
          } catch (error) {
            message.error(
              error instanceof Error ? error.message : '保存结算账户失败',
            );
            return false;
          }
        }}
      >
        <Space align="start" wrap size={16} style={{ width: '100%' }}>
          <ProFormText
            name="name"
            label="账户名称"
            width="sm"
            rules={[
              { required: true, whitespace: true, message: '请输入账户名称' },
            ]}
          />
          <ProFormText
            name="accountHolder"
            label="账户户名"
            width="sm"
            rules={[
              { required: true, whitespace: true, message: '请输入账户户名' },
            ]}
          />
          <ProFormText
            name="currency"
            label="账户币种"
            width="sm"
            placeholder="如 CNY、USD"
            rules={[{ required: true, len: 3, message: '请输入三位币种代码' }]}
          />
          <ProFormSelect
            name="usage"
            label="允许用途"
            width="sm"
            options={usageOptions}
            rules={[{ required: true, message: '请选择账户用途' }]}
          />
        </Space>
        <ProFormText
          name="bankName"
          label="开户银行名称及支行"
          rules={[
            { required: true, whitespace: true, message: '请输入开户银行' },
          ]}
        />
        <ProFormText
          name="accountNo"
          label="银行开户账号"
          rules={[
            { required: true, whitespace: true, message: '请输入银行账号' },
          ]}
        />
        <ProFormText
          name="swiftCode"
          label="SWIFT Code（外币国际结算）"
          placeholder="例如：ICBKCNBS"
        />
        <Space align="start" wrap size={24}>
          <ProFormSwitch name="isDefaultReceivable" label="设为应收默认账户" />
          <ProFormSwitch name="isDefaultPayable" label="设为应付默认账户" />
          <ProFormSwitch name="enabled" label="启用账户" />
        </Space>
        <ProFormTextArea
          name="remark"
          label="备注说明"
          fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
        />
      </ModalForm>
    </>
  );
}
