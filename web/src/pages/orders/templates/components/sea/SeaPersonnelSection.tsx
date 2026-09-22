import { Form } from 'antd';
import type { Rule } from 'antd/es/form';
import React from 'react';
import { FormRow, SearchableSelect } from '@/components/ui';
import { formatPersonnelLabel } from '@/features/personnel';
import type { TemplateProps } from '../../types';

interface PersonnelAssignmentOption {
  userId?: string;
  displayName?: string;
  departmentNames?: string[];
}

interface PersonnelAssignmentFieldsProps {
  label: string;
  userField: string;
  options: PersonnelAssignmentOption[];
  disabled?: boolean;
  /** 表单校验规则：新建模式下提成相关三岗由调用方传入必填规则。 */
  rules?: Rule[];
}

export function PersonnelAssignmentFields({
  label,
  userField,
  options,
  disabled = false,
  rules,
}: PersonnelAssignmentFieldsProps) {
  const userOptions = Array.from(
    new Map(
      options
        .filter((option) => option.userId)
        .map((option) => [
          option.userId as string,
          {
            label: formatPersonnelLabel(option),
            value: option.userId as string,
          },
        ]),
    ).values(),
  );

  return (
    <Form.Item
      label={label}
      name={userField}
      rules={rules}
      style={{ marginBottom: 0 }}
    >
      <SearchableSelect
        disabled={disabled}
        options={userOptions}
        placeholder="选择人员"
        popupMatchSelectWidth={false}
        allowClear={!disabled}
      />
    </Form.Item>
  );
}

export function buildSeaPersonnelSection(props: TemplateProps) {
  const { personnelOptions, creator, isDetail } = props;
  // 提成归属以订单人员为唯一真相：新建模式下操作/业务/客服三岗必填；
  // 详情模式不施加规则，避免存量订单缺岗时阻塞其它字段保存。
  const commissionStaffRules = (message: string): Rule[] =>
    isDetail ? [] : [{ required: true, message }];
  const fields: ReadonlyArray<{
    label: string;
    userField: string;
    rules?: Rule[];
  }> = [
    {
      label: '操作人员',
      userField: 'operatorUserId',
      rules: commissionStaffRules('请选择操作人员'),
    },
    {
      label: '业务人员',
      userField: 'salesUserId',
      rules: commissionStaffRules('请选择业务人员'),
    },
    {
      label: '客服人员',
      userField: 'customerServiceUserId',
      rules: commissionStaffRules('请选择客服人员'),
    },
    { label: '关联人员', userField: 'associateUserId' },
    { label: '单证人员', userField: 'documentUserId' },
    { label: '商务人员', userField: 'commercialUserId' },
    { label: '关联人员 2', userField: 'associate2UserId' },
  ];

  return {
    key: 'internalPersonnel',
    title: '内部人员',
    content: (
      <FormRow cols={4}>
        <PersonnelAssignmentFields
          label="创建人员"
          userField="creatorUserId"
          options={
            creator
              ? [{ userId: creator.userId, displayName: creator.displayName }]
              : []
          }
          disabled
        />
        {fields.map(({ label, userField, rules }) => (
          <PersonnelAssignmentFields
            key={userField}
            label={label}
            userField={userField}
            options={personnelOptions}
            rules={rules}
          />
        ))}
      </FormRow>
    ),
  };
}
