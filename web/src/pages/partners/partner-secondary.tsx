import {
  AuditOutlined,
  BankOutlined,
  EditOutlined,
  FileTextOutlined,
  PaperClipOutlined,
  PlusOutlined,
} from '@ant-design/icons';
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
import React, { useRef, useState } from 'react';
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

const { Text } = Typography;

const roleOptions = [
  { label: '客户', value: 1, color: 'blue' },
  { label: '供应商', value: 2, color: 'green' },
  { label: '代理', value: 3, color: 'purple' },
  { label: '承运人', value: 4, color: 'orange' },
];

const roleMap = new Map(roleOptions.map((opt) => [opt.value, opt]));

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
    { title: '发票抬头', dataIndex: 'invoiceTitle', ellipsis: true, render: (t) => <Text strong>{t}</Text> },
    {
      title: '结算币种',
      dataIndex: 'currency',
      width: 100,
      render: (cur) => <Tag color="gold" bordered={false}>{cur}</Tag>,
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
        record.isDefault ? <Tag color="blue">默认</Tag> : <Text type="secondary">-</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) =>
        record.status === 1 ? <Tag color="success">启用</Tag> : <Tag color="default">停用</Tag>,
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
            onClick={() => openAccountForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const contractColumns: ProColumns<API.PartnerContract>[] = [
    {
      title: '合同编号',
      dataIndex: 'contractNo',
      width: 150,
      copyable: true,
      render: (no) => <Text style={{ fontFamily: 'monospace', fontWeight: 500 }}>{no}</Text>,
    },
    { title: '合同名称', dataIndex: 'name', ellipsis: true, render: (name) => <Text strong>{name}</Text> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (_, record) => (
        <Tag color={contractStatusColors[record.status ?? 0]}>
          {contractStatusLabels[record.status ?? 0] ?? '未知'}
        </Tag>
      ),
    },
    {
      title: '生效起止日期',
      dataIndex: 'startDate',
      width: 220,
      render: (_, record) => (
        <span>
          {record.startDate ? dayjs(record.startDate).format('YYYY-MM-DD') : '-'}
          {' ~ '}
          {record.endDate ? dayjs(record.endDate).format('YYYY-MM-DD') : '-'}
        </span>
      ),
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
            onClick={() => openContractForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const accountPanel = hasCustomerRole ? (
    <ProTable<API.PartnerAccount>
      headerTitle={
        <Space size={6}>
          <BankOutlined style={{ color: '#1677ff' }} />
          <span>客户结算账户列表</span>
        </Space>
      }
      rowKey="id"
      actionRef={accountActionRef}
      columns={accountColumns}
      bordered
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
    <Alert
      showIcon
      type="info"
      message="无需结算账户配置"
      description="该往来单位当前未分配客户角色，仅客户身份支持配置发票开票与结算银行账户。"
      style={{ margin: '16px 0' }}
    />
  );

  const contractPanel = (
    <ProTable<API.PartnerContract>
      headerTitle={
        <Space size={6}>
          <FileTextOutlined style={{ color: '#1677ff' }} />
          <span>商务框架合同列表</span>
        </Space>
      }
      rowKey="id"
      actionRef={contractActionRef}
      columns={contractColumns}
      bordered
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
      title: '角色身份',
      dataIndex: 'roleType',
      width: 100,
      render: (_, record) => {
        const item = roleMap.get(record.roleType ?? 0);
        return item ? (
          <Tag color={item.color} bordered={false}>
            {item.label}
          </Tag>
        ) : (
          <Tag>未知</Tag>
        );
      },
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
      title: '约定结算日',
      dataIndex: 'settlementDay',
      width: 100,
      render: (_, record) =>
        record.settlementDay != null ? `${record.settlementDay} 日` : '-',
    },
    {
      title: '结算周期天数',
      dataIndex: 'settlementCycleDays',
      width: 110,
      render: (_, record) =>
        record.settlementCycleDays != null
          ? `${record.settlementCycleDays} 天`
          : '-',
    },
    {
      title: '结算币种',
      dataIndex: 'settlementCurrency',
      width: 90,
      render: (cur) => <Tag color="gold" bordered={false}>{cur}</Tag>,
    },
    {
      title: '启用状态',
      dataIndex: 'isActive',
      width: 90,
      render: (_, record) =>
        record.isActive ? <Tag color="success">启用</Tag> : <Tag color="default">停用</Tag>,
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
            onClick={() => openSettlementRuleForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  const settlementRulePanel = (
    <ProTable<SettlementRuleItem>
      headerTitle={
        <Space size={6}>
          <AuditOutlined style={{ color: '#1677ff' }} />
          <span>财务结算与对账规则</span>
        </Space>
      }
      rowKey="id"
      actionRef={settlementRuleActionRef}
      columns={settlementRuleColumns}
      bordered
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
    { title: '文件名', dataIndex: 'fileName', ellipsis: true, render: (name) => <Text strong>{name}</Text> },
    { title: 'MIME 类型', dataIndex: 'mimeType', width: 140, ellipsis: true },
    { title: '文件大小', dataIndex: 'fileSize', width: 110, render: (s) => `${s} 字节` },
    {
      title: '对象键',
      dataIndex: 'objectKey',
      copyable: true,
      ellipsis: true,
      render: (key) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{key}</Text>,
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
      headerTitle={
        <Space size={6}>
          <PaperClipOutlined style={{ color: '#1677ff' }} />
          <span>往来单位附件与证照</span>
        </Space>
      }
      rowKey="id"
      actionRef={attachmentActionRef}
      columns={attachmentColumns}
      bordered
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
        title={
          <Space size={8}>
            <BankOutlined style={{ color: '#1677ff' }} />
            <span>往来商务档案：{partner?.legalName ?? ''}</span>
            {partner?.code && (
              <Tag bordered={false} style={{ fontFamily: 'monospace' }}>
                {partner.code}
              </Tag>
            )}
          </Space>
        }
        open={open}
        onClose={onClose}
        width={960}
        destroyOnHidden
      >
        <Tabs
          items={[
            {
              key: 'accounts',
              label: (
                <Space size={4}>
                  <BankOutlined />
                  <span>结算账户</span>
                </Space>
              ),
              children: accountPanel,
            },
            {
              key: 'contracts',
              label: (
                <Space size={4}>
                  <FileTextOutlined />
                  <span>商务合同</span>
                </Space>
              ),
              children: contractPanel,
            },
            {
              key: 'settlement-rules',
              label: (
                <Space size={4}>
                  <AuditOutlined />
                  <span>结算规则</span>
                </Space>
              ),
              children: settlementRulePanel,
            },
            {
              key: 'attachments',
              label: (
                <Space size={4}>
                  <PaperClipOutlined />
                  <span>证照附件</span>
                </Space>
              ),
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
            message.success('结算账户已成功更新');
          } else {
            await partnerServiceCreatePartnerAccount(
              { partnerId: partner.id },
              { partnerId: partner.id, account },
            );
            message.success('结算账户已成功创建');
          }
          setAccountModalOpen(false);
          accountActionRef.current?.reload();
          return true;
        }}
      >
        <Space align="start" wrap size={16} style={{ width: '100%', marginBottom: 12 }}>
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
          name="invoiceTitle"
          label="开票发票抬头"
          placeholder="请输入增值税发票抬头"
          rules={[{ required: true, message: '请输入发票抬头' }]}
        />
        <ProFormText name="unifiedSocialCreditCode" label="纳税人识别号 / 统一社会信用代码" placeholder="18位纳税人识别号" />
        <ProFormText name="billingAddress" label="开票法定注册地址" placeholder="请输入开票地址" />
        <ProFormText name="billingPhone" label="开票联系电话" placeholder="请输入开票电话" />
        <ProFormText name="bankName" label="开户银行名称及支行" placeholder="例如：中国工商银行上海自贸试验区分行" />
        <ProFormText name="bankAccount" label="银行开户账号" placeholder="请输入银行结算账号" />
        <ProFormText name="swiftCode" label="SWIFT Code (外币国际结算)" placeholder="例如：ICBKCNBS" />
        <ProFormTextArea
          name="remark"
          label="备注说明"
          placeholder="请输入其他开票说明"
          fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
        />
      </ModalForm>

      <ModalForm<ContractFormValues>
        title={editingContract ? '编辑商务合同' : '新增商务合同'}
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
            message.success('合同已成功更新');
          } else {
            if (!values.contractNo) return false;
            await partnerServiceCreatePartnerContract(
              { partnerId: partner.id },
              {
                partnerId: partner.id,
                contract: { ...common, contractNo: values.contractNo.trim() },
              },
            );
            message.success('合同已成功创建');
          }
          setContractModalOpen(false);
          contractActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="contractNo"
          label="合同唯一编号"
          placeholder="例如：CT-2026-0001"
          disabled={Boolean(editingContract)}
          rules={[{ required: !editingContract, message: '请输入合同编号' }]}
        />
        <ProFormText
          name="name"
          label="合同名称"
          placeholder="例如：2026年度国际海运出口货代服务框架协议"
          rules={[{ required: true, message: '请输入合同名称' }]}
        />
        <ProFormSelect
          name="status"
          label="合同生命周期状态"
          options={availableContractStatuses(editingContract)}
          rules={[{ required: true, message: '请选择合同状态' }]}
        />
        <ProFormDateRangePicker
          name="dateRange"
          label="合同有效起止期限"
          rules={[{ required: true, message: '请选择合同期限' }]}
          fieldProps={{ allowEmpty: [false, false] }}
        />
        <ProFormTextArea
          name="paymentTerms"
          label="付款与账期约定"
          placeholder="请输入结算账期、支付方式与违约金约定"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="disputeResolution"
          label="争议管辖与解决"
          placeholder="例如：提交上海国际仲裁中心仲裁"
          fieldProps={{ rows: 3, maxLength: 2000, showCount: true }}
        />
        <ProFormTextArea
          name="otherNotes"
          label="补充约定事项"
          placeholder="其他补充约定"
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
            message.success('结算规则已成功更新');
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
            message.success('结算规则已成功创建');
          }
          setSettlementRuleModalOpen(false);
          settlementRuleActionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="roleType"
          label="适用业务角色"
          options={(partner?.roles ?? []).map((r) => ({
            label: roleLabels[r.type ?? 0] ?? '未知角色',
            value: r.type,
          }))}
          disabled={Boolean(editingSettlementRule)}
          rules={[{ required: true, message: '请选择角色' }]}
        />
        <ProFormSelect
          name="statementMode"
          label="对账核销模式"
          options={statementModeOptions}
          rules={[{ required: true, message: '请选择对账模式' }]}
        />
        <ProFormSelect
          name="settlementMethod"
          label="结算账期方式"
          options={settlementMethodOptions}
          rules={[{ required: true, message: '请选择结算方式' }]}
        />
        <ProFormSelect
          name="settlementBase"
          label="账期起算基准"
          options={settlementBaseOptions}
        />
        <ProFormDigit
          name="settlementDay"
          label="每月固定结算日"
          min={1}
          max={31}
          placeholder="例如: 25"
          fieldProps={{ precision: 0 }}
        />
        <ProFormDigit
          name="settlementCycleDays"
          label="周期循环天数"
          min={1}
          placeholder="例如: 30"
          fieldProps={{ precision: 0 }}
        />
        <ProFormText
          name="settlementCurrency"
          label="约定结算币种"
          placeholder="例如: CNY / USD"
          rules={[{ required: true, message: '请输入结算币种' }]}
        />
        <ProFormSwitch name="isActive" label="规则启用状态" initialValue />
      </ModalForm>

      <ModalForm<AttachmentFormValues>
        title="登记往来单位证照与附件"
        open={attachmentModalOpen}
        formRef={attachmentFormRef}
        modalProps={{
          destroyOnHidden: true,
          width: 560,
          onCancel: () => setAttachmentModalOpen(false),
        }}
        onOpenChange={setAttachmentModalOpen}
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
          setAttachmentModalOpen(false);
          attachmentActionRef.current?.reload();
          return true;
        }}
      >
        <Alert
          type="info"
          showIcon
          message="此处登记企业营业执照、开户许可证、水运许可证等对象存储引用。"
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
