import type { ActionType, ProFormInstance } from '@ant-design/pro-components';
import { ProFormDependency } from '@ant-design/pro-components';
import { Alert, App, Space, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { GuardedModalForm, ProFormSearchableSelect } from '@/components/ui';
import { AdminOrganizationKind } from '@/enums.generated';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import { adminServiceBatchAuthorizeDingTalkUsers } from '@/services/roncin/adminService';
import { isWriteResponse } from '@/utils/api';
import OrganizationRolesField from './OrganizationRolesField';
import { formatOrganizationHierarchyName } from './userConstants';

type AssignmentValues = {
  organizationIds: string[];
  rolesByOrganization?: Record<string, string[]>;
};
interface Props {
  users: API.AdminUser[];
  organizations: API.AdminOrganization[];
  onClose: () => void;
  onRemaining: (users: API.AdminUser[]) => void;
  actionRef: React.RefObject<ActionType | undefined>;
}

export default function DingTalkAssignmentModal({
  users,
  organizations,
  onClose,
  onRemaining,
  actionRef,
}: Props) {
  const { message } = App.useApp();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [requestFailure, setRequestFailure] = useState<string>();
  const [results, setResults] = useState<API.DingTalkAssignmentResult[]>([]);
  const companyOrganizations = organizations.filter((org) => {
    if (
      !org.enabled ||
      org.kind === AdminOrganizationKind.ORGANIZATION_KIND_SYSTEM
    )
      return false;
    let current: API.AdminOrganization | undefined = org;
    const visited = new Set<string>();
    while (current) {
      if (current.kind === AdminOrganizationKind.ORGANIZATION_KIND_COMPANY)
        return current.enabled === true;
      if (!current.enabled || !current.id || visited.has(current.id))
        return false;
      visited.add(current.id);
      current = organizations.find((item) => item.id === current?.parentId);
    }
    return false;
  });
  const names = users.map(
    (user) => user.displayName || user.username || '未命名人员',
  );
  return (
    <GuardedModalForm<AssignmentValues>
      title={
        users.length === 1 && !users[0].pendingAssignment
          ? '重新开通钉钉人员'
          : '分配组织并开通'
      }
      open={users.length > 0}
      formRef={formRef}
      initialValues={{ organizationIds: [], rolesByOrganization: {} }}
      modalProps={{ width: 600, destroyOnHidden: true }}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      submitter={{
        submitButtonProps: { disabled: Boolean(requestFailure) },
        searchConfig: {
          submitText: results.length ? '重试失败人员' : '分配并开通',
        },
      }}
      onFinish={async (values) => {
        if (requestFailure) return false;
        const items = (values.organizationIds ?? []).map((organizationId) => ({
          organizationId,
          roleIds: values.rolesByOrganization?.[organizationId] ?? [],
        }));
        if (!items.length || !items.some((item) => item.roleIds.length > 0)) {
          formRef.current?.setFields([
            { name: 'organizationIds', errors: ['请至少为一个组织分配角色'] },
          ]);
          return false;
        }
        setRequestFailure(undefined);
        try {
          const response = await adminServiceBatchAuthorizeDingTalkUsers(
            {
              userIds: users.flatMap((user) => (user.id ? [user.id] : [])),
              items,
            },
            { skipErrorHandler: true },
          );
          if (!isWriteResponse(response)) return false;
          if (
            !response?.success ||
            !Array.isArray(response.data) ||
            response.data.length !== users.length ||
            response.data.some(
              (result) =>
                result.success !== true &&
                ((result.success !== false && result.success !== undefined) ||
                  !result.reason?.trim()),
            ) ||
            users.some(
              (user) =>
                !response.data?.some((result) => result.userId === user.id),
            )
          )
            throw new Error('开通未返回结果，请刷新人员状态后确认');
          const failures = response.data.filter((result) => !result.success);
          const remaining = users.filter((user) =>
            failures.some((result) => result.userId === user.id),
          );
          setResults(failures);
          onRemaining(remaining);
          actionRef.current?.reload();
          if (!failures.length) {
            message.success(`已开通 ${response.data.length} 人`);
            return true;
          }
          message.warning(
            `成功 ${response.data.length - failures.length} 人，失败 ${failures.length} 人`,
          );
          return false;
        } catch (error) {
          setRequestFailure(
            getRequestErrorMessage(
              error,
              '开通结果未能确认，请刷新人员状态后处理',
            ),
          );
          actionRef.current?.reload();
          return false;
        }
      }}
    >
      <Alert
        showIcon
        type="info"
        title={`本次处理 ${users.length} 人`}
        description={
          <Space wrap>
            {names.map((name, index) => (
              <Typography.Text key={users[index].id}>{name}</Typography.Text>
            ))}
          </Space>
        }
        style={{ marginBottom: 16 }}
      />
      <Typography.Paragraph type="secondary">
        本批人员使用相同的组织配置，每个组织独立选择角色；至少为一个组织分配角色。每人独立开通，已成功人员保留结果，失败人员可单独重试。
      </Typography.Paragraph>
      {requestFailure && (
        <Alert
          type="error"
          showIcon
          title="请核实开通结果"
          description={requestFailure}
          style={{ marginBottom: 16 }}
        />
      )}
      {results.length > 0 && (
        <Alert
          type="warning"
          showIcon
          title="以下人员未开通"
          description={
            <ul>
              {results.map((result) => (
                <li key={result.userId}>
                  {users.find((user) => user.id === result.userId)
                    ?.displayName || '该人员'}
                  ：{result.message || '请刷新状态后重试'}
                </li>
              ))}
            </ul>
          }
          style={{ marginBottom: 16 }}
        />
      )}
      <ProFormSearchableSelect
        name="organizationIds"
        label="所属组织"
        mode="multiple"
        placeholder="请选择公司、部门或组，可多选"
        options={companyOrganizations.map((org) => ({
          value: org.id,
          label: formatOrganizationHierarchyName(org, organizations),
          code: org.code,
          name: org.name,
        }))}
        validateTrigger={[]}
        rules={[
          { required: true, message: '请选择所属组织' },
          ({ getFieldValue }) => ({
            validator: async (_, organizationIds: string[] = []) => {
              const rolesByOrganization =
                getFieldValue('rolesByOrganization') ?? {};
              if (
                organizationIds.length &&
                !organizationIds.some((id) => rolesByOrganization[id]?.length)
              ) {
                throw new Error('请至少为一个组织分配角色');
              }
            },
          }),
        ]}
        fieldProps={{
          onChange: (organizationIds: string[]) => {
            const previous =
              formRef.current?.getFieldValue('rolesByOrganization') ?? {};
            formRef.current?.setFieldValue(
              'rolesByOrganization',
              Object.fromEntries(
                organizationIds.map((id) => [id, previous[id] ?? []]),
              ),
            );
          },
        }}
      />
      <ProFormDependency name={['organizationIds']}>
        {({ organizationIds }) =>
          (organizationIds ?? []).map((organizationId: string) => (
            <OrganizationRolesField
              key={organizationId}
              organizationId={organizationId}
              label={`${formatOrganizationHierarchyName(organizationId, organizations)} · 角色（选填）`}
              name={['rolesByOrganization', organizationId]}
              open={users.length > 0}
            />
          ))
        }
      </ProFormDependency>
    </GuardedModalForm>
  );
}
