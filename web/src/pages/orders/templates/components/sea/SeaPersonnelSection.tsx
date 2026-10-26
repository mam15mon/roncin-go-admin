import { Form } from 'antd';
import type { Rule } from 'antd/es/form';
import React from 'react';
import { FormRow, SearchableSelect } from '@/components/ui';
import { formatPersonnelLabel } from '@/features/personnel';
import type { TemplateProps } from '../../types';

// 标记实际搜索 input，避免密码管理器把人员选择识别为登录凭据输入。
const PersonnelSearchInput = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>((props, ref) => (
  <input
    {...props}
    ref={ref}
    autoComplete="off"
    data-1p-ignore="true"
    data-lpignore="true"
    data-bwignore="true"
    data-form-type="other"
  />
));

const personnelSelectComponents = { input: PersonnelSearchInput };

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
  nameOnly?: boolean;
  /** 表单校验规则：新建模式下提成相关三岗由调用方传入必填规则。 */
  rules?: Rule[];
}

export function PersonnelAssignmentFields({
  label,
  userField,
  options,
  disabled = false,
  nameOnly = false,
  rules,
}: PersonnelAssignmentFieldsProps) {
  const userOptions = Array.from(
    new Map(
      options
        .filter((option) => option.userId)
        .map((option) => [
          option.userId as string,
          {
            label: nameOnly
              ? option.displayName || '未命名人员'
              : formatPersonnelLabel(option),
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
        components={personnelSelectComponents}
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
  // 业务/客服由客户档案带入（选择客户时自动填充），界面只读不可手改；
  // 新建模式下两岗必填：客户未配置对应岗位时提交被阻断，需先维护客户档案。
  // 详情模式不施加规则，避免存量订单缺岗时阻塞其它字段保存。
  const customerStaffRules = (message: string): Rule[] =>
    isDetail ? [] : [{ required: true, message }];
  const fields: ReadonlyArray<{
    label: string;
    userField: string;
    rules?: Rule[];
    locked?: boolean;
  }> = [
    {
      label: '操作人员',
      userField: 'operatorUserId',
      rules: isDetail ? [] : [{ required: true, message: '请选择操作人员' }],
    },
    {
      label: '业务人员',
      userField: 'salesUserId',
      rules: customerStaffRules('请先在客户档案配置业务人员'),
      locked: true,
    },
    {
      label: '客服人员',
      userField: 'customerServiceUserId',
      rules: customerStaffRules('请先在客户档案配置客服人员'),
      locked: true,
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
          nameOnly={!isDetail}
        />
        {fields.map(({ label, userField, rules, locked }) => (
          <PersonnelAssignmentFields
            key={userField}
            label={label}
            userField={userField}
            options={personnelOptions}
            rules={rules}
            disabled={locked || props.personnelReadonly}
            nameOnly={!isDetail}
          />
        ))}
      </FormRow>
    ),
  };
}
