import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import type { RuleValues } from '../types';
import type { EmployeeOption } from './commissionRuleShared';

type CommissionRuleFormModalProps = {
  open: boolean;
  editingRule: API.FinanceCommissionRule | undefined;
  formRef: React.RefObject<ProFormInstance | undefined>;
  organizationOptions: API.FinanceOrganizationOption[];
  employeeOptions: EmployeeOption[];
  searchEmployees: (
    keyword: string,
    targetOrganizationId: string | undefined,
    apply: (options: EmployeeOption[]) => void,
  ) => Promise<void>;
  onEmployeeOptionsChange: React.Dispatch<
    React.SetStateAction<EmployeeOption[]>
  >;
  onCancel: () => void;
  onFinish: (values: RuleValues) => Promise<boolean>;
};

/** 提成方案新建 / 编辑表单弹窗 */
export default function CommissionRuleFormModal({
  open,
  editingRule,
  formRef,
  organizationOptions,
  employeeOptions,
  searchEmployees,
  onEmployeeOptionsChange,
  onCancel,
  onFinish,
}: CommissionRuleFormModalProps) {
  const editingReached =
    Boolean(editingRule?.enabled) &&
    Boolean(editingRule?.effectiveFrom) &&
    (editingRule?.effectiveFrom ?? '') <= dayjs().format('YYYY-MM-DD');

  return (
    <ModalForm<RuleValues>
      formRef={formRef}
      key={editingRule?.id || 'new-rule'}
      title={editingRule ? '编辑提成方案' : '新建提成方案'}
      open={open}
      width={620}
      modalProps={{
        destroyOnHidden: true,
        onCancel,
      }}
      initialValues={{
        name: editingRule?.name,
        personnelRole: editingRule?.personnelRole || 'SALES',
        calculationBasis: editingRule?.calculationBasis || 'REALIZED_PROFIT',
        ratePercent: editingRule ? Number(editingRule.ratePercent) : undefined,
        effectiveRange:
          editingRule?.effectiveFrom && editingRule?.effectiveTo
            ? [dayjs(editingRule.effectiveFrom), dayjs(editingRule.effectiveTo)]
            : undefined,
        enabled: editingRule?.enabled ?? true,
        note: editingRule?.note,
        organizationId: editingRule?.organizationId,
      }}
      onFinish={onFinish}
    >
      {editingRule?.legacyReadonly && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="迁移前历史旧规则为只读"
          description="该规则没有任何员工分配且已停用，不能编辑或重新启用；请复制为新方案并重新分配员工。"
        />
      )}
      {!editingRule && (
        <ProFormSearchableSelect
          name="organizationId"
          label="所属公司"
          rules={[{ required: true, message: '请选择所属公司' }]}
          options={organizationOptions.map((item) => ({
            value: item.id ?? '',
            label: item.name ?? item.code ?? item.id ?? '',
          }))}
          fieldProps={{
            onChange: (value: string) => {
              // 组织变化后清空员工候选并按新组织重载。
              onEmployeeOptionsChange([]);
              if (value) {
                void searchEmployees('', value, onEmployeeOptionsChange);
              }
            },
          }}
        />
      )}
      <ProFormText
        name="name"
        label="方案名称"
        disabled={Boolean(editingRule?.legacyReadonly)}
        rules={[{ required: true, message: '请输入方案名称' }]}
      />
      <ProFormSearchableSelect
        name="personnelRole"
        label="提成人员身份"
        disabled={Boolean(editingRule)}
        rules={[{ required: true, message: '请选择人员身份' }]}
        options={[
          { value: 'SALES', label: '业务人员' },
          { value: 'OPERATOR', label: '操作人员' },
          { value: 'CUSTOMER_SERVICE', label: '客服人员' },
        ]}
        extra="人员身份表示适用员工按哪一种订单提成归属参与计算；方案生效后不可修改。"
      />
      <ProFormSearchableSelect
        name="calculationBasis"
        label="计提口径"
        disabled={Boolean(editingRule && editingReached)}
        rules={[{ required: true, message: '请选择计提口径' }]}
        options={[
          {
            value: 'REALIZED_PROFIT',
            label: '已实现毛利（推荐，按收入配比分摊成本）',
          },
          {
            value: 'REALIZED_REVENUE',
            label: '已实现收入（按来源单分摊收入全额）',
          },
        ]}
      />
      <ProFormDigit
        name="ratePercent"
        label="提成比例（%）"
        min={0.0001}
        max={100}
        fieldProps={{ precision: 4 }}
        disabled={Boolean(editingRule && editingReached)}
        rules={[{ required: true, message: '请输入提成比例' }]}
      />
      <ProFormDateRangePicker
        name="effectiveRange"
        label="生效区间"
        fieldProps={{
          disabled: [Boolean(editingRule && editingReached), false],
        }}
        extra={
          editingRule && editingReached
            ? '方案已生效：起始日不可修改，终止日只能选择当天或未来日期。'
            : '新启用方案的起始日必须是当天或未来日期。'
        }
      />
      <ProFormSwitch
        name="enabled"
        label="启用"
        disabled={Boolean(editingRule && editingReached)}
      />
      <ProFormTextArea
        name="note"
        label="方案说明"
        fieldProps={{ maxLength: 500 }}
        disabled={Boolean(editingRule?.legacyReadonly)}
      />
      {!editingRule && (
        <ProFormSelect
          name="employeeIds"
          label="适用员工"
          extra="同一批员工共享方案的身份、口径、比例与有效期；个别员工比例不同请另建方案。启用方案必须至少分配一名员工。"
          fieldProps={{
            mode: 'multiple',
            filterOption: false,
            onSearch: (value: string) => {
              void searchEmployees(
                value,
                formRef.current?.getFieldValue('organizationId') as
                  | string
                  | undefined,
                onEmployeeOptionsChange,
              );
            },
            onDropdownVisibleChange: (visible: boolean) => {
              if (visible) {
                void searchEmployees(
                  '',
                  formRef.current?.getFieldValue('organizationId') as
                    | string
                    | undefined,
                  onEmployeeOptionsChange,
                );
              }
            },
          }}
          options={employeeOptions}
          placeholder="搜索当前组织员工"
        />
      )}
    </ModalForm>
  );
}
