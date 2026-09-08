import { cleanup, render, screen } from '@testing-library/react';
import React, { useEffect } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

const lifecycle = vi.hoisted(() => [] as string[]);

vi.mock('@/components/layout/TagsView', () => ({
  TagsView: () => {
    useEffect(() => {
      lifecycle.push('tabs:mount');
      return () => {
        lifecycle.push('tabs:unmount');
      };
    }, []);
    return <div data-testid="tags-view" />;
  },
}));

vi.mock('@umijs/max', () => ({
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
  history: {
    location: { hash: '', pathname: '/welcome', search: '' },
    replace: vi.fn(),
  },
  useModel: vi.fn(),
}));

vi.mock('@/components/layout/HeaderMenus', () => ({
  HeaderMenus: () => null,
}));

vi.mock('@/components/layout/HeaderTitle', () => ({
  HeaderTitle: () => null,
}));

vi.mock('@/components/OrganizationSwitcher', () => ({
  default: () => null,
}));

vi.mock('@/components/RightContent/AvatarDropdown', () => ({
  AvatarDropdown: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('@/utils/appFeedback', () => ({
  AppFeedbackBridge: () => null,
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceMe: vi.fn(),
}));

vi.mock('antd', () => ({
  Result: () => null,
}));

import { getOrganizationWorkspaceKey, OrganizationWorkspace } from './app';

function Page({ label }: { label: string }) {
  useEffect(() => {
    lifecycle.push(`page:${label}:mount`);
    return () => {
      lifecycle.push(`page:${label}:unmount`);
    };
  }, [label]);
  return <div>{label}</div>;
}

describe('organization workspace', () => {
  afterEach(() => {
    lifecycle.length = 0;
    cleanup();
  });

  it('用户或组织身份变化时，同时重建页签和页面', () => {
    const user = {
      id: 'user-1',
      currentOrganization: { id: 'org-1' },
    } as API.CurrentUser;
    const { rerender } = render(
      <OrganizationWorkspace key={getOrganizationWorkspaceKey(user)}>
        <Page label="天津页面" />
      </OrganizationWorkspace>,
    );

    expect(screen.getByTestId('tags-view')).toBeInTheDocument();
    expect(lifecycle).toEqual(['tabs:mount', 'page:天津页面:mount']);

    const switchedUser = {
      ...user,
      currentOrganization: { id: 'org-2' },
    };
    rerender(
      <OrganizationWorkspace key={getOrganizationWorkspaceKey(switchedUser)}>
        <Page label="北京页面" />
      </OrganizationWorkspace>,
    );

    expect(lifecycle).toEqual([
      'tabs:mount',
      'page:天津页面:mount',
      'tabs:unmount',
      'page:天津页面:unmount',
      'tabs:mount',
      'page:北京页面:mount',
    ]);
    expect(screen.getByText('北京页面')).toBeInTheDocument();
  });

  it('workspace key 同时包含用户与组织', () => {
    expect(
      getOrganizationWorkspaceKey({
        id: 'user-1',
        currentOrganization: { id: 'org-1' },
      }),
    ).toBe('user-1:org-1');
    expect(
      getOrganizationWorkspaceKey({
        id: 'user-2',
        currentOrganization: { id: 'org-1' },
      }),
    ).toBe('user-2:org-1');
  });
});
