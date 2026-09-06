import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AvatarDropdown } from './AvatarDropdown';

let mockCurrentUser: any = null;

vi.mock('@umijs/max', () => ({
  history: {
    replace: vi.fn(),
  },
  useModel: (model: string) => {
    if (model === '@@initialState') {
      return {
        initialState: { currentUser: mockCurrentUser },
        setInitialState: vi.fn(),
      };
    }
    return {};
  },
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceLogout: vi.fn().mockResolvedValue({}),
}));

describe('AvatarDropdown Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = null;
  });

  afterEach(() => {
    cleanup();
  });

  it('未登录时展示 Spin 加载态', () => {
    const { container } = render(<AvatarDropdown />);
    expect(container.querySelector('.ant-spin')).toBeInTheDocument();
  });

  it('已登录时展示用户头像首字母与带 roncin-avatar-name 类的用户名', () => {
    mockCurrentUser = {
      username: 'testadmin',
      displayName: '测试管理员',
      currentOrganization: { id: 'org-1', name: '总部' },
      roleScopes: [{ roleCode: 'admin' }],
    };

    const { container } = render(<AvatarDropdown />);
    const nameEl = container.querySelector('.roncin-avatar-name');
    expect(nameEl).toBeInTheDocument();
    expect(nameEl).toHaveTextContent('测试管理员');
    expect(screen.getByText('测')).toBeInTheDocument();
  });
});
