import { PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { App, Button, Drawer, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  settlementServiceCreateCommissionRule,
  settlementServiceListCommissionRules,
  settlementServiceUpdateCommissionRule,
} from '@/services/roncin/settlementService';
import {
  calculationBasisMeta,
  calculationBasisText,
  decimalText,
  getBusinessReason,
  personnelRoleMeta,
  personnelRoleText,
  type RuleValues,
} from '../types';

type CommissionRulesDrawerProps = {
  open: boolean;
  onClose: () => void;
  canManage: boolean;
};

export default function CommissionRulesDrawer({
  open,
  onClose,
  canManage,
}: CommissionRulesDrawerProps) {
  const { message, modal } = App.useApp();
  const ruleActionRef = useRef<ActionType | undefined>(undefined);
  const [ruleFormOpen, setRuleFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<API.FinanceCommissionRule>();

  const ruleColumns: ProColumns<API.FinanceCommissionRule>[] = [
    {
      title: '规则名称',
      dataIndex: 'name',
      copyable: true,
    },
    {
      title: '适用角色',
      dataIndex: 'personnelRole',
      valueType: 'select',
      valueEnum: personnelRoleMeta,
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      valueType: 'select',
      valueEnum: calculationBasisMeta,
      renderText: (value) => calculationBasisText(value),
    },
    {
      title: '提成比例',
      dataIndex: 'ratePercent',
      align: 'right',
      search: false,
      renderText: (value) => `${decimalText(value)}%`,
    },
    {
      title: '生效区间',
      key: 'effectiveRange',
      search: false,
      render: (_, record) =>
        record.effectiveFrom || record.effectiveTo ? (
          <span>
            {record.effectiveFrom || '-'} ~ {record.effectiveTo || '-'}
          </span>
        ) : (
          '永久有效'
        ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      valueType: 'select',
      valueEnum: {
        true: { text: '已启用', status: 'Success' },
        false: { text: '已停用', status: 'Default' },
      },
      render: (_, record) => (
        <Tag color={record.enabled ? 'success' : 'default'}>
          {record.enabled ? '已启用' : '已停用'}
        </Tag>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      width: 70,
      search: false,
      renderText: (value) => `v${value || 0}`,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      search: false,
      renderText: (value) =>
        value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 100,
      render: (_, record) =>
        canManage ? (
          <a
            key="edit"
            onClick={() => {
              setEditingRule(record);
              setRuleFormOpen(true);
            }}
          >
            编辑
          </a>
        ) : null,
    },
  ];

  return (
    <>
      <Drawer
        title="提成考核规则"
        size={980}
        open={open}
        onClose={onClose}
      >
        <ProTable<API.FinanceCommissionRule>
          actionRef={ruleActionRef}
          rowKey="id"
          columns={ruleColumns}
          bordered
          size="small"
          search={{ defaultCollapsed: false }}
          toolBarRender={() =>
            canManage
              ? [
                  <Button
                    key="new-rule"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingRule(undefined);
                      setRuleFormOpen(true);
                    }}
                  >
                    新建规则
                  </Button>,
                ]
              : []
          }
          request={async (params) => {
            const response = await settlementServiceListCommissionRules({
              page: params.current ?? 1,
              pageSize: params.pageSize ?? 20,
              keyword: params.name,
              personnelRole: params.personnelRole,
              enabled: params.enabled,
            });
            return {
              data: response.data || [],
              total: Number(response.total || 0),
              success: response.success ?? true,
            };
          }}
        />
      </Drawer>

      <ModalForm<RuleValues>
        key={editingRule?.id || 'new-rule'}
        title={editingRule ? '编辑考核规则' : '新建考核规则'}
        open={ruleFormOpen}
        width={620}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setRuleFormOpen(false),
        }}
        initialValues={{
          name: editingRule?.name,
          personnelRole: editingRule?.personnelRole || 'SALES',
          calculationBasis: editingRule?.calculationBasis || 'REALIZED_PROFIT',
          ratePercent: editingRule
            ? Number(editingRule.ratePercent)
            : undefined,
          effectiveRange:
            editingRule?.effectiveFrom && editingRule?.effectiveTo
              ? [
                  dayjs(editingRule.effectiveFrom),
                  dayjs(editingRule.effectiveTo),
                ]
              : undefined,
          enabled: editingRule?.enabled ?? true,
          note: editingRule?.note,
        }}
        onFinish={async (values) => {
          const rule = {
            name: values.name,
            personnelRole: values.personnelRole,
            calculationBasis: values.calculationBasis,
            ratePercent: String(values.ratePercent),
            effectiveFrom: values.effectiveRange?.[0]?.format('YYYY-MM-DD'),
            effectiveTo: values.effectiveRange?.[1]?.format('YYYY-MM-DD'),
            enabled: values.enabled,
            note: values.note,
          };
          try {
            if (editingRule?.id && editingRule.version) {
              await settlementServiceUpdateCommissionRule(
                { id: editingRule.id },
                {
                  id: editingRule.id,
                  expectedVersion: editingRule.version,
                  rule,
                },
              );
              message.success('考核规则已更新');
            } else {
              await settlementServiceCreateCommissionRule({ rule });
              message.success('考核规则已创建');
            }
            setRuleFormOpen(false);
            ruleActionRef.current?.reload();
            return true;
          } catch (error: any) {
            const reason = getBusinessReason(error);
            if (reason === 'FINANCE_COMMISSION_RULE_CONFLICT') {
              modal.warning({
                title: '规则名称冲突或版本已变化',
                content: '请检查规则名称是否重复，或刷新后重试。',
              });
              return false;
            }
            message.error(error.message || '保存规则失败');
            return false;
          }
        }}
      >
        <ProFormText
          name="name"
          label="规则名称"
          rules={[{ required: true, message: '请输入规则名称' }]}
        />
        <ProFormSearchableSelect
          name="personnelRole"
          label="适用客户人员角色"
          rules={[{ required: true, message: '请选择适用角色' }]}
          options={[
            { value: 'SALES', label: '业务人员' },
            { value: 'OPERATOR', label: '操作人员' },
            { value: 'CUSTOMER_SERVICE', label: '客服人员' },
          ]}
        />
        <ProFormSearchableSelect
          name="calculationBasis"
          label="计提口径"
          rules={[{ required: true, message: '请选择计提口径' }]}
          options={[
            {
              value: 'REALIZED_PROFIT',
              label: '已实现毛利（推荐，按收入配比分摊成本）',
            },
            { value: 'REALIZED_REVENUE', label: '已实现收入（按核销收入全额）' },
          ]}
        />
        <ProFormDigit
          name="ratePercent"
          label="提成比例（%）"
          min={0.0001}
          max={100}
          fieldProps={{ precision: 4 }}
          rules={[{ required: true, message: '请输入提成比例' }]}
        />
        <ProFormDateRangePicker name="effectiveRange" label="生效区间" />
        <ProFormSwitch name="enabled" label="启用" />
        <ProFormTextArea
          name="note"
          label="规则说明"
          fieldProps={{ maxLength: 500 }}
        />
      </ModalForm>
    </>
  );
}
