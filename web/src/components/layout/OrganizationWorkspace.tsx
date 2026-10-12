import React from 'react';
import { useInitialState } from '@/app/AppProvider';
import { TagsView } from '@/components/layout/TagsView';
import { ListQueryStateProvider } from '@/components/ui/list-query-state';

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
  const { initialState } = useInitialState();
  const user = initialState?.currentUser;
  return (
    <ListQueryStateProvider
      enabled={Boolean(user?.id && user.currentOrganization?.id)}
    >
      <div className="roncin-layout-wrapper roncin-organization-workspace">
        <TagsView />
        <div className="roncin-layout-main">{children}</div>
      </div>
    </ListQueryStateProvider>
  );
}
