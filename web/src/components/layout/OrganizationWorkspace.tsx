import { QueryClientProvider } from '@tanstack/react-query';
import { App } from 'antd';
import React, { useLayoutEffect, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { TagsView } from '@/components/layout/TagsView';
import { ListQueryStateProvider } from '@/components/ui/list-query-state';
import { createQueryClient } from '@/utils/queryClient';
import { activateWorkspaceRequestScope } from '@/utils/requestClient';

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
  // 身份不完整时不挂业务查询，也不借用入口的公开登录缓存。
  if (!user?.id || !user.currentOrganization?.id) return null;

  return (
    <WorkspaceQueryProvider key={getOrganizationWorkspaceKey(user)}>
      {/* 业务确认框随工作区销毁；头部切换确认仍由根 App 持有。 */}
      <App component={false}>
        <ListQueryStateProvider enabled>
          <div className="roncin-layout-wrapper roncin-organization-workspace">
            <TagsView />
            <div className="roncin-layout-main">{children}</div>
          </div>
        </ListQueryStateProvider>
      </App>
    </WorkspaceQueryProvider>
  );
}

function WorkspaceQueryProvider({ children }: { children: React.ReactNode }) {
  const [client] = useState(createQueryClient);

  useLayoutEffect(() => {
    const releaseRequests = activateWorkspaceRequestScope();
    return () => {
      releaseRequests();
      // QueryCache.clear 会同步取消查询；忽略 AbortSignal 的迟到结果也无法回写。
      // 布局阶段释放旧实例，下一工作区挂载前已清除全部业务缓存。
      client.clear();
    };
  }, [client]);

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
