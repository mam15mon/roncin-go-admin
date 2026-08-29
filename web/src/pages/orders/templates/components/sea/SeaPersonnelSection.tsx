import { Col, Form, Row } from 'antd';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import type { TemplateProps } from '../../types';

interface PersonnelAssignmentOption {
  userId?: string;
  displayName?: string;
  organizationId?: string;
  organizationName?: string;
}

interface PersonnelAssignmentFieldsProps {
  label: string;
  userField: string;
  organizationField: string;
  options: PersonnelAssignmentOption[];
  disabled?: boolean;
}

export function PersonnelAssignmentFields({
  label,
  userField,
  organizationField,
  options,
  disabled = false,
}: PersonnelAssignmentFieldsProps) {
  const form = Form.useFormInstance();
  const selectedUserID = Form.useWatch(userField);
  const selectedOrganizationID = Form.useWatch(organizationField);
  const organizationOptions = Array.from(
    new Map(
      options
        .filter((option) => option.organizationId)
        .map((option) => [
          option.organizationId as string,
          {
            label: option.organizationName || option.organizationId,
            value: option.organizationId as string,
          },
        ]),
    ).values(),
  );
  const userOptions = Array.from(
    new Map(
      options
        .filter(
          (option) =>
            option.userId &&
            (!selectedOrganizationID ||
              option.organizationId === selectedOrganizationID),
        )
        .map((option) => [
          option.userId as string,
          {
            label: option.displayName || option.userId,
            value: option.userId as string,
          },
        ]),
    ).values(),
  );

  return (
    <Col span={24}>
      <Row gutter={16} align="middle">
        <Col className="col-5">
          <ProFormSearchableSelect
            name={organizationField}
            label={`${label}所属公司`}
            disabled={disabled}
            options={organizationOptions}
            placeholder="请选择所属公司"
            fieldProps={{
              onChange: (value) => {
                const currentMatched = options.some(
                  (item) =>
                    item.userId === selectedUserID &&
                    item.organizationId === value,
                );
                if (!currentMatched) {
                  form?.setFieldValue(userField, undefined);
                }
              },
            }}
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name={userField}
            label={label}
            disabled={disabled}
            options={userOptions}
            placeholder="请选择人员"
            fieldProps={{
              onChange: (value) => {
                if (!value) {
                  return;
                }
                const matched = options.find((item) => item.userId === value);
                if (matched?.organizationId) {
                  form?.setFieldValue(
                    organizationField,
                    matched.organizationId,
                  );
                }
              },
            }}
          />
        </Col>
      </Row>
    </Col>
  );
}

export function buildSeaPersonnelSection(props: TemplateProps) {
  const { personnelOptions, creator } = props;

  return {
    key: 'internalInfo',
    title: '内部信息',
    content: (
      <>
        <PersonnelAssignmentFields
          label="创建人员"
          userField="creatorUserId"
          organizationField="creatorOrganizationId"
          options={
            creator
              ? [
                  {
                    userId: creator.userId,
                    displayName: creator.displayName,
                    organizationId: creator.organizationId,
                    organizationName: creator.organizationName,
                  },
                ]
              : []
          }
          disabled
        />
        <PersonnelAssignmentFields
          label="操作人员"
          userField="operatorUserId"
          organizationField="operatorOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="业务人员"
          userField="salesUserId"
          organizationField="salesOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="客服人员"
          userField="customerServiceUserId"
          organizationField="customerServiceOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="关联人员"
          userField="associateUserId"
          organizationField="associateOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="单证人员"
          userField="documentUserId"
          organizationField="documentOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="商务人员"
          userField="commercialUserId"
          organizationField="commercialOrganizationId"
          options={personnelOptions}
        />
        <PersonnelAssignmentFields
          label="关联人员 2"
          userField="associate2UserId"
          organizationField="associate2OrganizationId"
          options={personnelOptions}
        />
      </>
    ),
  };
}
