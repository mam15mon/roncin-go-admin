import { Form } from 'antd';
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
}

export function PersonnelAssignmentFields({
  label,
  userField,
  options,
  disabled = false,
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
    <Form.Item label={label} name={userField} style={{ marginBottom: 0 }}>
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
  const { personnelOptions, creator } = props;
  const fields = [
    ['操作人员', 'operatorUserId'],
    ['业务人员', 'salesUserId'],
    ['客服人员', 'customerServiceUserId'],
    ['关联人员', 'associateUserId'],
    ['单证人员', 'documentUserId'],
    ['商务人员', 'commercialUserId'],
    ['关联人员 2', 'associate2UserId'],
  ] as const;

  return {
    key: 'internalInfo',
    title: '内部信息',
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
        {fields.map(([label, userField]) => (
          <PersonnelAssignmentFields
            key={userField}
            label={label}
            userField={userField}
            options={personnelOptions}
          />
        ))}
      </FormRow>
    ),
  };
}
