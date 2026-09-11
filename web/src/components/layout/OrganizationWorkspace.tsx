import React from 'react';
import { TagsView } from '@/components/layout/TagsView';

export function getOrganizationWorkspaceKey(
  currentUser?: API.CurrentUser,
): string {
  return `${currentUser?.id ?? 'anonymous'}:${currentUser?.currentOrganization?.id ?? 'no-organization'}`;
}

export function OrganizationWorkspace({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="roncin-layout-wrapper roncin-organization-workspace">
      <TagsView />
      <div className="roncin-layout-main">{children}</div>
    </div>
  );
}
