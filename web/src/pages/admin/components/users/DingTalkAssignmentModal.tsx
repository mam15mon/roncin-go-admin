import type { ActionType, ProFormInstance } from '@ant-design/pro-components';
import { ModalForm } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Alert, App, Space, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { AdminOrganizationKind } from '@/enums.generated';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  adminServiceBatchAuthorizeDingTalkUsers,
  adminServiceListOrganizationRoles,
} from '@/services/roncin/adminService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import { formatOrganizationHierarchyName } from './userConstants';

type AssignmentValues = { organizationId: string; roleIds: string[] };
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
  const [organizationId, setOrganizationId] = useState<string>();
  const [requestFailure, setRequestFailure] = useState<string>();
  const [results, setResults] = useState<API.DingTalkAssignmentResult[]>([]);
  const rolesQuery = useQuery({
    queryKey: ['admin-users', 'organization-roles', organizationId],
    enabled: Boolean(organizationId && users.length),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListOrganizationRoles({
            organizationId: organizationId ?? '',
          }),
          '读取公司角色失败',
        ),
      ),
  });
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
    <ModalForm<AssignmentValues>
      title={
        users.length === 1 && !users[0].pendingAssignment
          ? '重新开通钉钉人员'
          : '分配组织并开通'
      }
      open={users.length > 0}
      formRef={formRef}
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
        setRequestFailure(undefined);
        try {
          const response = await adminServiceBatchAuthorizeDingTalkUsers(
            {
              userIds: users.flatMap((user) => (user.id ? [user.id] : [])),
              organizationId: values.organizationId,
              roleIds: values.roleIds,
            },
            { skipErrorHandler: true },
          );
          if (
            !response?.data ||
            response.data.length !== users.length ||
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
        本批人员分配相同组织和角色。每人独立开通，已成功人员保留结果，失败人员可单独重试。
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
        name="organizationId"
        label="所属组织"
        placeholder="请选择公司、部门或组"
        options={companyOrganizations.map((org) => ({
          value: org.id,
          label: formatOrganizationHierarchyName(org, organizations),
          code: org.code,
          name: org.name,
        }))}
        rules={[{ required: true, message: '请选择所属组织' }]}
        fieldProps={{
          onChange: (id: string) => {
            formRef.current?.setFieldValue('roleIds', []);
            setOrganizationId(id);
          },
        }}
      />
      <ProFormSearchableSelect
        name="roleIds"
        label="公司角色"
        mode="multiple"
        placeholder="请至少选择一个角色"
        disabled={!organizationId || rolesQuery.isFetching}
        options={(rolesQuery.data ?? [])
          .filter((role) => role.enabled)
          .map((role) => ({
            value: role.id,
            label: role.name,
            code: role.code,
            name: role.name,
          }))}
        rules={[{ required: true, message: '请至少分配一个角色' }]}
      />
    </ModalForm>
  );
}
