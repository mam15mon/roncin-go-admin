import { useQuery } from '@tanstack/react-query';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { adminServiceListOrganizationRoles } from '@/services/roncin/adminService';
import { ensureListResponse, unwrapList } from '@/utils/api';

export default function OrganizationRolesField({
  organizationId,
  label,
  name,
  open,
}: {
  organizationId: string;
  label: string;
  name: string | string[];
  open: boolean;
}) {
  const rolesQuery = useQuery({
    queryKey: ['admin-users', 'organization-roles', organizationId],
    enabled: open && Boolean(organizationId),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListOrganizationRoles({ organizationId }),
          '读取组织角色失败',
        ),
      ),
  });
  return (
    <ProFormSearchableSelect
      name={name}
      label={label}
      mode="multiple"
      preserve={false}
      placeholder="选填，可不分配角色"
      extra={
        rolesQuery.isError
          ? '角色候选读取失败，可保留空角色或重新打开后重试'
          : undefined
      }
      options={(rolesQuery.data ?? [])
        .filter((role) => role.enabled)
        .map((role) => ({
          label: `${role.name} (${role.code})`,
          value: role.id,
          code: role.code,
          name: role.name,
        }))}
      fieldProps={{ loading: rolesQuery.isFetching }}
    />
  );
}
