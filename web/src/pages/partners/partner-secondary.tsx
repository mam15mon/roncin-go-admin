import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { Alert, App, Button, Drawer, Space, Tabs, Tag, Typography } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { useRef, useState } from 'react';
import {
  partnerServiceCreatePartnerAccount,
  partnerServiceCreatePartnerContract,
  partnerServiceCreatePartnerSettlementRule,
  partnerServiceListPartnerAccounts,
  partnerServiceListPartnerAttachments,
  partnerServiceListPartnerContracts,
  partnerServiceListPartnerSettlementRules,
  partnerServiceRegisterPartnerAttachment,
  partnerServiceUpdatePartnerAccount,
  partnerServiceUpdatePartnerContract,
  partnerServiceUpdatePartnerSettlementRule,
} from '@/services/roncin/partnerService';

const roleOptions = [
  { label: '客户', value: 1 },
  { label: '供应商', value: 2 },
  { label: '代理', value: 3 },
  { label: '承运人', value: 4 },
];

const roleLabels: Record<number, string> = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
);

const statementModeOptions = [
  { label: '单票对账', value: 1 },
  { label: '汇总对账', value: 2 },
];

const statementModeLabels: Record<number, string> = Object.fromEntries(
  statementModeOptions.map((option) => [option.value, option.label]),
);

const settlementMethodOptions = [
  { label: '单票结算', value: 1 },
  { label: '月结', value: 2 },
  { label: '周结', value: 3 },
  { label: '半月结', value: 4 },
  { label: '双月结', value: 5 },
  { label: '季结', value: 6 },
  { label: '45天', value: 7 },
  { label: '预付', value: 8 },
];

const settlementMethodLabels: Record<number, string> = Object.fromEntries(
  settlementMethodOptions.map((option) => [option.value, option.label]),
);

const settlementBaseOptions = [
  { label: '账单日', value: 1 },
  { label: '开航日', value: 2 },
  { label: '到港日', value: 3 },
];

const settlementBaseLabels: Record<number, string> = Object.fromEntries(
  settlementBaseOptions.map((option) => [option.value, option.label]),
);

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

type SettlementRuleItem = API.PartnerSettlementRule & {
  roleType?: number;
};

type SettlementRuleFormValues = {
  roleType?: number;
  statementMode?: number;
  settlementMethod?: number;
  settlementBase?: number;
  settlementDay?: number;
  settlementCycleDays?: number;
  settlementCurrency?: string;
  isActive?: boolean;
};

type AttachmentFormValues = {
  fileName?: string;
  mimeType?: string;
  fileSize?: string | number;
  objectKey?: string;
  checksum?: string;
  idempotencyKey?: string;
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
  const settlementRuleActionRef = useRef<ActionType | undefined>(undefined);
  const attachmentActionRef = useRef<ActionType | undefined>(undefined);
  const accountFormRef = useRef<ProFormInstance | undefined>(undefined);
  const contractFormRef = useRef<ProFormInstance | undefined>(undefined);
  const settlementRuleFormRef = useRef<ProFormInstance | undefined>(undefined);
  const attachmentFormRef = useRef<ProFormInstance | undefined>(undefined);
  const [accountModalOpen, setAccountModalOpen] = useState(false);
  const [contractModalOpen, setContractModalOpen] = useState(false);
  const [settlementRuleModalOpen, setSettlementRuleModalOpen] = useState(false);
  const [attachmentModalOpen, setAttachmentModalOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<API.PartnerAccount>();
  const [editingContract, setEditingContract] = useState<API.PartnerContract>();
  const [editingSettlementRule, setEditingSettlementRule] =
    useState<SettlementRuleItem>();
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

  const openSettlementRuleForm = (rule?: SettlementRuleItem) => {
    setEditingSettlementRule(rule);
    settlementRuleFormRef.current?.resetFields();
    setSettlementRuleModalOpen(true);
  };

  const openAttachmentForm = () => {
    attachmentFormRef.current?.resetFields();
    setAttachmentModalOpen(true);
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

  const settlementRuleColumns: ProColumns<SettlementRuleItem>[] = [
    {
      title: '角色',
      dataIndex: 'roleType',
      width: 90,
      render: (_, record) => (
        <Tag>{roleLabels[record.roleType ?? 0] ?? '未知'}</Tag>
      ),
    },
    {
      title: '对账模式',
      dataIndex: 'statementMode',
      width: 110,
      render: (_, record) =>
        statementModeLabels[record.statementMode ?? 0] ?? '-',
    },
    {
      title: '结算方式',
      dataIndex: 'settlementMethod',
      width: 110,
      render: (_, record) =>
        settlementMethodLabels[record.settlementMethod ?? 0] ?? '-',
    },
    {
      title: '结算基准',
      dataIndex: 'settlementBase',
      width: 110,
      render: (_, record) =>
        settlementBaseLabels[record.settlementBase ?? 0] ?? '-',
    },
    {
      title: '结算日',
      dataIndex: 'settlementDay',
      width: 90,
      render: (_, record) =>
        record.settlementDay != null ? `${record.settlementDay}日` : '-',
    },
    {
      title: '周期天数',
      dataIndex: 'settlementCycleDays',
      width: 100,
      render: (_, record) =>
        record.settlementCycleDays != null
          ? `${record.settlementCycleDays}天`
          : '-',
    },
    {
      title: '结算币种',
      dataIndex: 'settlementCurrency',
      width: 90,
    },
    {
      title: '启用',
      dataIndex: 'isActive',
      width: 80,
      render: (_, record) =>
        record.isActive ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
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
            onClick={() => openSettlementRuleForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const settlementRulePanel = (
    <ProTable<SettlementRuleItem>
      rowKey="id"
      actionRef={settlementRuleActionRef}
      columns={settlementRuleColumns}
      search={false}
      pagination={false}
      request={async () => {
        if (!partner?.id) return { data: [], success: true };
        const partnerId = partner.id;
        const rolesToQuery = (partner.roles ?? [])
          .map((r) => r.type)
          .filter((t): t is number => typeof t === 'number');
        const results = await Promise.all(
          rolesToQuery.map(async (roleType) => {
            const response = await partnerServiceListPartnerSettlementRules({
              partnerId,
              roleType,
            });
            return (response.data ?? []).map((item) => ({
              ...item,
              roleType,
            }));
          }),
        );
        return { data: results.flat(), success: true };
      }}
      toolBarRender={() =>
        canManage
          ? [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openSettlementRuleForm()}
              >
                新增结算规则
              </Button>,
            ]
          : []
      }
    />
  );

  const attachmentColumns: ProColumns<API.PartnerAttachment>[] = [
    { title: '文件名', dataIndex: 'fileName', ellipsis: true },
    { title: 'MIME 类型', dataIndex: 'mimeType', width: 140, ellipsis: true },
    { title: '文件大小', dataIndex: 'fileSize', width: 100 },
    {
      title: '对象键',
      dataIndex: 'objectKey',
      copyable: true,
      ellipsis: true,
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

  const attachmentPanel = (
    <ProTable<API.PartnerAttachment>
      rowKey="id"
      actionRef={attachmentActionRef}
      columns={attachmentColumns}
      search={false}
      pagination={false}
      request={async () => {
        if (!partner?.id) return { data: [], success: true };
        const response = await partnerServiceListPartnerAttachments({
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
                onClick={() => openAttachmentForm()}
              >
                登记附件
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
            {
              key: 'settlement-rules',
              label: '结算规则',
              children: settlementRulePanel,
            },
            {
              key: 'attachments',
              label: '附件',
              children: attachmentPanel,
            },
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

      <ModalForm<SettlementRuleFormValues>
        title={editingSettlementRule ? '编辑结算规则' : '新增结算规则'}
        open={settlementRuleModalOpen}
        formRef={settlementRuleFormRef}
        initialValues={
          editingSettlementRule ?? {
            roleType: partner?.roles?.[0]?.type ?? 1,
            statementMode: 1,
            settlementMethod: 1,
            settlementCurrency: 'CNY',
            isActive: true,
          }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setSettlementRuleModalOpen(false),
        }}
        onOpenChange={setSettlementRuleModalOpen}
        onFinish={async (values) => {
          if (
            !partner?.id ||
            !values.roleType ||
            !values.statementMode ||
            !values.settlementMethod ||
            !values.settlementCurrency
          ) {
            return false;
          }
          const rule: API.PartnerSettlementRuleInput = {
            statementMode: values.statementMode,
            settlementMethod: values.settlementMethod,
            settlementBase: values.settlementBase,
            settlementDay: values.settlementDay,
            settlementCycleDays: values.settlementCycleDays,
            settlementCurrency: values.settlementCurrency.trim().toUpperCase(),
            isActive: values.isActive ?? false,
          };
          if (editingSettlementRule?.id) {
            await partnerServiceUpdatePartnerSettlementRule(
              {
                partnerId: partner.id,
                roleType: values.roleType,
                id: editingSettlementRule.id,
              },
              {
                partnerId: partner.id,
                roleType: values.roleType,
                id: editingSettlementRule.id,
                rule,
              },
            );
            message.success('结算规则已更新');
          } else {
            await partnerServiceCreatePartnerSettlementRule(
              {
                partnerId: partner.id,
                roleType: values.roleType,
              },
              {
                partnerId: partner.id,
                roleType: values.roleType,
                rule,
              },
            );
            message.success('结算规则已创建');
          }
          setSettlementRuleModalOpen(false);
          settlementRuleActionRef.current?.reload();
          return true;
        }}
      >
        <Space align="start" wrap>
          <ProFormSelect
            name="roleType"
            label="角色"
            width="xs"
            options={roleOptions.filter((option) =>
              partner?.roles?.some((role) => role.type === option.value),
            )}
            disabled={Boolean(editingSettlementRule)}
            rules={[{ required: true, message: '请选择角色' }]}
          />
          <ProFormSelect
            name="statementMode"
            label="对账模式"
            width="xs"
            options={statementModeOptions}
            rules={[{ required: true, message: '请选择对账模式' }]}
          />
          <ProFormSelect
            name="settlementMethod"
            label="结算方式"
            width="xs"
            options={settlementMethodOptions}
            rules={[{ required: true, message: '请选择结算方式' }]}
          />
          <ProFormSelect
            name="settlementBase"
            label="结算基准"
            width="xs"
            options={settlementBaseOptions}
            allowClear
          />
        </Space>
        <Space align="start" wrap>
          <ProFormDigit
            name="settlementDay"
            label="结算日"
            width="xs"
            min={1}
            max={31}
            fieldProps={{ precision: 0 }}
          />
          <ProFormDigit
            name="settlementCycleDays"
            label="周期天数"
            width="xs"
            min={1}
            max={365}
            fieldProps={{ precision: 0 }}
          />
          <ProFormText
            name="settlementCurrency"
            label="结算币种"
            width="xs"
            rules={[{ required: true, len: 3, message: '请输入三位币种代码' }]}
          />
          <ProFormSwitch name="isActive" label="启用" />
        </Space>
      </ModalForm>

      <ModalForm<AttachmentFormValues>
        title="登记附件"
        open={attachmentModalOpen}
        formRef={attachmentFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setAttachmentModalOpen(false),
        }}
        onOpenChange={setAttachmentModalOpen}
        onFinish={async (values) => {
          if (
            !partner?.id ||
            !values.fileName ||
            !values.mimeType ||
            values.fileSize === undefined ||
            values.fileSize === null ||
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
          message.success('附件已登记');
          setAttachmentModalOpen(false);
          attachmentActionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          message="此处仅登记外部对象存储引用，不上传文件内容。"
          style={{ marginBottom: 16 }}
        />
        <ProFormText
          name="fileName"
          label="文件名"
          rules={[{ required: true, message: '请输入文件名' }]}
        />
        <ProFormText
          name="mimeType"
          label="MIME 类型"
          rules={[{ required: true, message: '请输入 MIME 类型' }]}
        />
        <ProFormDigit
          name="fileSize"
          label="文件大小"
          min={1}
          max={104857600}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入文件大小' }]}
        />
        <ProFormText
          name="objectKey"
          label="对象键"
          rules={[{ required: true, message: '请输入对象键' }]}
        />
        <ProFormText
          name="checksum"
          label="校验和"
        />
        <ProFormText
          name="idempotencyKey"
          label="幂等键"
          rules={[{ required: true, message: '请输入幂等键' }]}
        />
      </ModalForm>
    </>
  );
}
