import { render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RegisterRedirect from './index';

const { navigateMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
}));

let mockSearch = '';

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    Navigate: (props: { to: string; replace?: boolean }) => {
      navigateMock(props);
      return null;
    },
    useLocation: () => ({ search: mockSearch }),
  };
});

describe('RegisterRedirect', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('访问 /user/register 时自动重定向到 /user/login', () => {
    mockSearch = '';
    render(<RegisterRedirect />);
    expect(navigateMock).toHaveBeenCalledWith({
      to: '/user/login',
      replace: true,
    });
  });

  it('访问带 ?invite= 参数时完整透传至 /user/login', () => {
    mockSearch = '?invite=test-token-123';
    render(<RegisterRedirect />);
    expect(navigateMock).toHaveBeenCalledWith({
      to: '/user/login?invite=test-token-123',
      replace: true,
    });
  });
});
