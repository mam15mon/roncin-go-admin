import {
  ModalForm,
  ProFormDatePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import type { EmployeeOption } from './commissionRuleShared';

/** CopyValues 复制为新方案表单值。 */
export type CopyValues = {
  name: string;
  personnelRole: 'SALES' | 'OPERATOR' | 'CUSTOMER_SERVICE';
  calculationBasis: 'REALIZED_PROFIT' | 'REALIZED_REVENUE';
  ratePercent: number;
  effectiveFrom?: string;
  effectiveTo?: string;
  employeeIds?: string[];
  note?: string;
};

type CommissionRuleCopyModalProps = {
  copySourceRule: API.FinanceCommissionRule | undefined;
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
  onFinish: (values: CopyValues) => Promise<boolean>;
};

/** 复制为新方案弹窗：原方案在新方案生效日前一日自然结束 */
export default function CommissionRuleCopyModal({
  copySourceRule,
  employeeOptions,
  searchEmployees,
  onEmployeeOptionsChange,
  onCancel,
  onFinish,
}: CommissionRuleCopyModalProps) {
  return (
    <ModalForm<CopyValues>
      key={copySourceRule?.id || 'copy-rule'}
      title="复制为新方案"
      open={Boolean(copySourceRule)}
      width={620}
      modalProps={{
        destroyOnHidden: true,
        onCancel,
      }}
      initialValues={{
        name: copySourceRule ? `${copySourceRule.name}-新方案` : undefined,
        personnelRole:
          (copySourceRule?.personnelRole as CopyValues['personnelRole']) ||
          'SALES',
        calculationBasis:
          (copySourceRule?.calculationBasis as CopyValues['calculationBasis']) ||
          'REALIZED_PROFIT',
        ratePercent: copySourceRule
          ? Number(copySourceRule.ratePercent)
          : undefined,
        note: copySourceRule?.note,
        employeeIds: (copySourceRule?.assignments ?? [])
          .filter(
            (item) =>
              !item.effectiveTo ||
              item.effectiveTo >= dayjs().format('YYYY-MM-DD'),
          )
          .map((item) => item.employeeId ?? ''),
      }}
      onFinish={onFinish}
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title="原方案将在新方案生效日前一日自然结束"
        description="新方案必须以当天或未来日期生效；原方案的历史计算依据保持不变，来源日期在原方案期间内的提成继续按原方案计算。"
      />
      <ProFormText
        name="name"
        label="新方案名称"
        rules={[{ required: true, message: '请输入新方案名称' }]}
      />
      <ProFormSearchableSelect
        name="personnelRole"
        label="提成人员身份"
        rules={[{ required: true, message: '请选择人员身份' }]}
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
          { value: 'REALIZED_PROFIT', label: '已实现毛利（推荐）' },
          { value: 'REALIZED_REVENUE', label: '已实现收入' },
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
      <ProFormDatePicker
        name="effectiveFrom"
        label="新方案生效起始日"
        fieldProps={{
          disabledDate: (current: Dayjs) =>
            Boolean(current?.isBefore(dayjs().startOf('day'))),
        }}
        rules={[{ required: true, message: '请选择生效起始日' }]}
        extra="必须为当天或未来日期。"
      />
      <ProFormDatePicker name="effectiveTo" label="新方案生效终止日（可选）" />
      <ProFormSelect
        name="employeeIds"
        label="适用员工"
        extra="默认带出原方案当前/未来名单，可调整。"
        fieldProps={{
          mode: 'multiple',
          filterOption: false,
          onSearch: (value: string) => {
            void searchEmployees(
              value,
              copySourceRule?.organizationId,
              onEmployeeOptionsChange,
            );
          },
          onDropdownVisibleChange: (visible: boolean) => {
            if (visible) {
              void searchEmployees(
                '',
                copySourceRule?.organizationId,
                onEmployeeOptionsChange,
              );
            }
          },
        }}
        options={employeeOptions}
        placeholder="搜索当前组织员工"
      />
      <ProFormTextArea
        name="note"
        label="方案说明"
        fieldProps={{ maxLength: 500 }}
      />
    </ModalForm>
  );
}
