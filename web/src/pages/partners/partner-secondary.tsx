import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Drawer, Space, Tabs, Tag, Typography } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { useRef, useState } from 'react';
import {
  partnerServiceCreatePartnerAccount,
  partnerServiceCreatePartnerContract,
  partnerServiceListPartnerAccounts,
  partnerServiceListPartnerContracts,
  partnerServiceUpdatePartnerAccount,
  partnerServiceUpdatePartnerContract,
} from '@/services/roncin/partnerService';

const accountStatusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
];

const contractStatusOptions = [
  { label: '待生效', value: 1 },
  { label: '生效中', value: 2 },
  { label: '已到期', value: 3 },
  { label: '已终止', value: 4 },
];

const contractStatusLabels: Record<number, string> = Object.fromEntries(
  contractStatusOptions.map((option) => [option.value, option.label]),
);

const contractStatusColors: Record<number, string | undefined> = {
  1: 'processing',
  2: 'success',
  3: undefined,
  4: 'error',
};

type AccountFormValues = API.PartnerAccountInput;

type ContractFormValues = {
  contractNo?: string;
  name?: string;
  status?: number;
  dateRange?: [Dayjs, Dayjs];
  paymentTerms?: string;
  disputeResolution?: string;
  otherNotes?: string;
};

type PartnerSecondaryProps = {
  partner?: API.Partner;
  open: boolean;
  canManage: boolean;
  onClose: () => void;
};

function availableContractStatuses(contract?: API.PartnerContract) {
  if (!contract) return contractStatusOptions.slice(0, 2);
  if (contract.status === 1) {
    return contractStatusOptions.filter((option) =>
      [1, 2, 4].includes(option.value),
    );
  }
  if (contract.status === 2) {
    return contractStatusOptions.filter((option) =>
      [2, 3, 4].includes(option.value),
    );
  }
  return contractStatusOptions.filter(
    (option) => option.value === contract.status,
  );
}

export default function PartnerSecondary({
  partner,
  open,
  canManage,
  onClose,
}: PartnerSecondaryProps) {
  const { message } = App.useApp();
  const accountActionRef = useRef<ActionType | undefined>(undefined);
  const contractActionRef = useRef<ActionType | undefined>(undefined);
  const accountFormRef = useRef<ProFormInstance | undefined>(undefined);
  const contractFormRef = useRef<ProFormInstance | undefined>(undefined);
  const [accountModalOpen, setAccountModalOpen] = useState(false);
  const [contractModalOpen, setContractModalOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<API.PartnerAccount>();
  const [editingContract, setEditingContract] = useState<API.PartnerContract>();
  const hasCustomerRole =
    partner?.roles?.some((role) => role.type === 1) ?? false;

  const openAccountForm = (account?: API.PartnerAccount) => {
    setEditingAccount(account);
    accountFormRef.current?.resetFields();
    setAccountModalOpen(true);
  };

  const openContractForm = (contract?: API.PartnerContract) => {
    setEditingContract(contract);
    contractFormRef.current?.resetFields();
    setContractModalOpen(true);
  };

  const accountColumns: ProColumns<API.PartnerAccount>[] = [
    { title: '发票抬头', dataIndex: 'invoiceTitle', ellipsis: true },
    { title: '币种', dataIndex: 'currency', width: 80 },
    { title: '开户行', dataIndex: 'bankName', ellipsis: true },
    {
      title: '银行账号',
      dataIndex: 'bankAccount',
      copyable: true,
      ellipsis: true,
    },
    {
      title: '默认',
      dataIndex: 'isDefault',
      width: 70,
      render: (_, record) =>
        record.isDefault ? <Tag color="blue">默认</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (_, record) =>
        record.status === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
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
      render: (_, record) =>
        canManage ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openAccountForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const contractColumns: ProColumns<API.PartnerContract>[] = [
    { title: '合同编号', dataIndex: 'contractNo', width: 140, copyable: true },
    { title: '合同名称', dataIndex: 'name', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => (
        <Tag color={contractStatusColors[record.status ?? 0]}>
          {contractStatusLabels[record.status ?? 0] ?? '未知'}
        </Tag>
      ),
    },
    {
      title: '开始日期',
      dataIndex: 'startDate',
      valueType: 'date',
      width: 110,
    },
    { title: '结束日期', dataIndex: 'endDate', valueType: 'date', width: 110 },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, record) =>
        canManage ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openContractForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const accountPanel = hasCustomerRole ? (
    <ProTable<API.PartnerAccount>
      rowKey="id"
      actionRef={accountActionRef}
      columns={accountColumns}
      search={false}
      pagination={false}
      request={async () => {
        if (!partner?.id) return { data: [], success: true };
        const response = await partnerServiceListPartnerAccounts({
          partnerId: partner.id,
        });
        return { data: response.data ?? [], success: response.success ?? true };
      }}
      toolBarRender={() =>
        canManage
          ? [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openAccountForm()}
              >
                新增结算账户
              </Button>,
            ]
          : []
      }
    />
  ) : (
    <Typography.Text type="secondary">
      该往来单位没有客户角色，不能配置客户结算账户。
    </Typography.Text>
  );

  const contractPanel = (
    <ProTable<API.PartnerContract>
      rowKey="id"
      actionRef={contractActionRef}
      columns={contractColumns}
      search={false}
      pagination={false}
      request={async () => {
        if (!partner?.id) return { data: [], success: true };
        const response = await partnerServiceListPartnerContracts({
          partnerId: partner.id,
        });
        return { data: response.data ?? [], success: response.success ?? true };
      }}
      toolBarRender={() =>
        canManage
          ? [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openContractForm()}
              >
                新增合同
              </Button>,
            ]
          : []
      }
    />
  );

  return (
    <>
      <Drawer
        title={`账户与合同 - ${partner?.legalName ?? ''}`}
        open={open}
        onClose={onClose}
        size="large"
        destroyOnHidden
      >
        <Tabs
          items={[
            { key: 'accounts', label: '结算账户', children: accountPanel },
            { key: 'contracts', label: '合同', children: contractPanel },
          ]}
        />
      </Drawer>

      <ModalForm<AccountFormValues>
        title={editingAccount ? '编辑结算账户' : '新增结算账户'}
        open={accountModalOpen}
        formRef={accountFormRef}
        initialValues={
          editingAccount ?? { currency: 'CNY', status: 1, isDefault: false }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setAccountModalOpen(false),
        }}
        onOpenChange={setAccountModalOpen}
        onFinish={async (values) => {
          if (!partner?.id) return false;
          const account: API.PartnerAccountInput = {
            ...values,
            currency: values.currency.trim(),
            invoiceTitle: values.invoiceTitle.trim(),
          };
          if (editingAccount?.id) {
            await partnerServiceUpdatePartnerAccount(
              { partnerId: partner.id, id: editingAccount.id },
              { partnerId: partner.id, id: editingAccount.id, account },
            );
            message.success('结算账户已更新');
          } else {
            await partnerServiceCreatePartnerAccount(
              { partnerId: partner.id },
              { partnerId: partner.id, account },
            );
            message.success('结算账户已创建');
          }
          setAccountModalOpen(false);
          accountActionRef.current?.reload();
          return true;
        }}
      >
        <Space align="start" wrap>
          <ProFormText
            name="currency"
            label="币种"
            width="xs"
            rules={[{ required: true, len: 3, message: '请输入三位币种代码' }]}
          />
          <ProFormSelect
            name="status"
            label="状态"
            width="xs"
            options={accountStatusOptions}
            rules={[{ required: true }]}
          />
          <ProFormSwitch name="isDefault" label="默认账户" />
        </Space>
        <ProFormText
          name="invoiceTitle"
          label="发票抬头"
          rules={[{ required: true, message: '请输入发票抬头' }]}
        />
        <ProFormText name="unifiedSocialCreditCode" label="统一社会信用代码" />
        <ProFormText name="billingAddress" label="开票地址" />
        <ProFormText name="billingPhone" label="开票电话" />
        <ProFormText name="bankName" label="开户行" />
        <ProFormText name="bankAccount" label="银行账号" />
        <ProFormText name="swiftCode" label="SWIFT Code" />
        <ProFormTextArea
          name="remark"
          label="备注"
          fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <ModalForm<ContractFormValues>
        title={editingContract ? '编辑合同' : '新增合同'}
        open={contractModalOpen}
        formRef={contractFormRef}
        initialValues={
          editingContract
            ? {
                ...editingContract,
                dateRange:
                  editingContract.startDate && editingContract.endDate
                    ? [
                        dayjs(editingContract.startDate),
                        dayjs(editingContract.endDate),
                      ]
                    : undefined,
              }
            : { status: 1 }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setContractModalOpen(false),
        }}
        onOpenChange={setContractModalOpen}
        onFinish={async (values) => {
          if (
            !partner?.id ||
            !values.name ||
            !values.status ||
            !values.dateRange
          )
            return false;
          const common = {
            name: values.name.trim(),
            status: values.status,
            startDate: values.dateRange[0].startOf('day').toISOString(),
            endDate: values.dateRange[1].startOf('day').toISOString(),
            paymentTerms: values.paymentTerms,
            disputeResolution: values.disputeResolution,
            otherNotes: values.otherNotes,
          };
          if (editingContract?.id) {
            await partnerServiceUpdatePartnerContract(
              { partnerId: partner.id, id: editingContract.id },
              {
                partnerId: partner.id,
                id: editingContract.id,
                contract: common,
              },
            );
            message.success('合同已更新');
          } else {
            if (!values.contractNo) return false;
            await partnerServiceCreatePartnerContract(
              { partnerId: partner.id },
              {
                partnerId: partner.id,
                contract: { ...common, contractNo: values.contractNo.trim() },
              },
            );
            message.success('合同已创建');
          }
          setContractModalOpen(false);
          contractActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="contractNo"
          label="合同编号"
          disabled={Boolean(editingContract)}
          rules={[{ required: !editingContract, message: '请输入合同编号' }]}
        />
        <ProFormText
          name="name"
          label="合同名称"
          rules={[{ required: true, message: '请输入合同名称' }]}
        />
        <ProFormSelect
          name="status"
          label="合同状态"
          options={availableContractStatuses(editingContract)}
          rules={[{ required: true, message: '请选择合同状态' }]}
        />
        <ProFormDateRangePicker
          name="dateRange"
          label="合同期限"
          rules={[{ required: true, message: '请选择合同期限' }]}
          fieldProps={{ allowEmpty: [false, false] }}
        />
        <ProFormTextArea
          name="paymentTerms"
          label="付款条款"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="disputeResolution"
          label="争议解决"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="otherNotes"
          label="其他约定"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
      </ModalForm>
    </>
  );
}
