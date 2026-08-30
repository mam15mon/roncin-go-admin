import { AuditOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Space, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  PartnerRoleType,
  PartnerSettlementBase,
  PartnerSettlementMethod,
  PartnerStatementMode,
} from '@/enums.generated';
import {
  partnerServiceCreatePartnerSettlementRule,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartnerSettlementRule,
} from '@/services/roncin/partnerService';
import { unwrapList } from '@/utils/api';

const roleOptions = [
  { label: '客户', value: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER, color: 'blue' },
  { label: '供应商', value: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER, color: 'green' },
  { label: '国外代理', value: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT, color: 'purple' },
  { label: '承运人', value: PartnerRoleType.PARTNER_ROLE_TYPE_CARRIER, color: 'orange' },
];

const roleMap = new Map<number, (typeof roleOptions)[number]>(
  roleOptions.map((option) => [option.value, option]),
);

const roleLabels: Record<number, string> = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
);

const statementModeOptions = [
  { label: '单票对账', value: PartnerStatementMode.PARTNER_STATEMENT_MODE_SINGLE },
  { label: '汇总对账', value: PartnerStatementMode.PARTNER_STATEMENT_MODE_MULTI },
];

const statementModeLabels: Record<number, string> = Object.fromEntries(
  statementModeOptions.map((option) => [option.value, option.label]),
);

const settlementMethodOptions = [
  { label: '单票结算', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BY_TICKET },
  { label: '月结', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_MONTHLY },
  { label: '周结', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_WEEKLY },
  { label: '半月结', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY },
  { label: '双月结', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BI_MONTHLY },
  { label: '季结', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_QUARTERLY },
  { label: '45天', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_DAYS_45 },
  { label: '预付', value: PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_PREPAID },
];

const settlementMethodLabels: Record<number, string> = Object.fromEntries(
  settlementMethodOptions.map((option) => [option.value, option.label]),
);

const settlementBaseOptions = [
  { label: '账单日', value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_BILL_DATE },
  { label: '开航日', value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_SAILING_DATE },
  { label: '到港日', value: PartnerSettlementBase.PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE },
];

const settlementBaseLabels: Record<number, string> = Object.fromEntries(
  settlementBaseOptions.map((option) => [option.value, option.label]),
);

export type SettlementRuleItem = API.PartnerSettlementRule & {
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

type SettlementRulesPanelProps = {
  partner?: API.Partner;
  canManage: boolean;
};

export default function SettlementRulesPanel({
  partner,
  canManage,
}: SettlementRulesPanelProps) {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<SettlementRuleItem>();

  const openForm = (rule?: SettlementRuleItem) => {
    setEditingRule(rule);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const columns: ProColumns<SettlementRuleItem>[] = [
    {
      title: '角色身份',
      dataIndex: 'roleType',
      width: 100,
      render: (_, record) => {
        const item = roleMap.get(record.roleType ?? 0);
        return item ? (
          <Tag color={item.color} variant="filled">
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
      render: (cur) => (
        <Tag color="gold" variant="filled">
          {cur}
        </Tag>
      ),
    },
    {
      title: '启用状态',
      dataIndex: 'isActive',
      width: 90,
      render: (_, record) =>
        record.isActive ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
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
            onClick={() => openForm(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <ProTable<SettlementRuleItem>
        headerTitle={
          <Space size={6}>
            <AuditOutlined style={{ color: '#1677ff' }} />
            <span>财务结算与对账规则</span>
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
              return unwrapList(response).map((item) => ({
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
                  onClick={() => openForm()}
                >
                  新增结算规则
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<SettlementRuleFormValues>
        title={editingRule ? '编辑结算规则' : '新增结算规则'}
        open={modalOpen}
        formRef={formRef}
        initialValues={
          editingRule ?? {
            roleType: partner?.roles?.[0]?.type ?? 1,
            statementMode: PartnerStatementMode.PARTNER_STATEMENT_MODE_SINGLE,
            settlementMethod:
              PartnerSettlementMethod.PARTNER_SETTLEMENT_METHOD_BY_TICKET,
            settlementCurrency: 'CNY',
            isActive: true,
          }
        }
        modalProps={{
          destroyOnHidden: true,
          width: 720,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
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
          if (editingRule?.id) {
            await partnerServiceUpdatePartnerSettlementRule(
              {
                partnerId: partner.id,
                roleType: values.roleType,
                id: editingRule.id,
              },
              {
                partnerId: partner.id,
                roleType: values.roleType,
                id: editingRule.id,
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
          setModalOpen(false);
          actionRef.current?.reload();
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
          disabled={Boolean(editingRule)}
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
    </>
  );
}
