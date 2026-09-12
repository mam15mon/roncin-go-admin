import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './organization-picker';

const { authServiceSwitchOrganizationMock, messageErrorMock } = vi.hoisted(
  () => ({
    authServiceSwitchOrganizationMock: vi.fn(),
    messageErrorMock: vi.fn(),
  }),
);

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  const MockedApp = Object.assign(actual.App, {
    useApp: () => ({
      message: { error: messageErrorMock },
    }),
  });
  return { ...actual, App: MockedApp };
});

vi.mock('@/services/roncin/authService', () => ({
  authServiceSwitchOrganization: authServiceSwitchOrganizationMock,
}));

const multiOrgOptions = [
  {
    organizationId: 'org-1',
    organizationName: '天津公司',
    organizationCode: 'TJ',
    isDefault: true,
  },
  {
    organizationId: 'org-2',
    organizationName: '北京公司',
    organizationCode: 'BJ',
    isDefault: false,
  },
];

function renderPicker(
  options = multiOrgOptions,
  onEnter: () => void = vi.fn(),
) {
  return render(
    <App>
      <OrganizationPicker options={options} onEnter={onEnter} />
    </App>,
  );
}

describe('OrganizationPicker', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('渲染候选组织卡片：名称、编码与默认标记', () => {
    renderPicker();

    expect(screen.getByText('选择进入的组织')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '进入天津公司' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '进入北京公司' }),
    ).toBeInTheDocument();
    expect(screen.getByText('TJ')).toBeInTheDocument();
    expect(screen.getByText('BJ')).toBeInTheDocument();
    expect(screen.getByText('默认')).toBeInTheDocument();
  });

  it('点击默认组织：不调用切换接口，直接进入', () => {
    const onEnter = vi.fn();
    renderPicker(multiOrgOptions, onEnter);

    fireEvent.click(screen.getByRole('button', { name: '进入天津公司' }));

    expect(authServiceSwitchOrganizationMock).not.toHaveBeenCalled();
    expect(onEnter).toHaveBeenCalledTimes(1);
  });

  it('点击其他组织：先切换会话，成功后进入', async () => {
    const onEnter = vi.fn();
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: { id: 'user-1', currentOrganization: { id: 'org-2' } },
    });
    renderPicker(multiOrgOptions, onEnter);

    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));

    await waitFor(() => {
      expect(onEnter).toHaveBeenCalledTimes(1);
    });
    expect(authServiceSwitchOrganizationMock).toHaveBeenCalledWith(
      { organizationId: 'org-2' },
      { skipErrorHandler: true },
    );
  });

  it('切换失败：提示错误且不进入所选组织', async () => {
    const onEnter = vi.fn();
    authServiceSwitchOrganizationMock.mockRejectedValue(
      new Error('无权进入该组织'),
    );
    renderPicker(multiOrgOptions, onEnter);

    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith('无权进入该组织');
    });
    expect(onEnter).not.toHaveBeenCalled();
    // 失败后可以重试
    authServiceSwitchOrganizationMock.mockResolvedValue({
      data: { id: 'user-1', currentOrganization: { id: 'org-2' } },
    });
    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));
    await waitFor(() => {
      expect(onEnter).toHaveBeenCalledTimes(1);
    });
  });

  it('服务端未返回新会话主体：提示失败且不进入', async () => {
    const onEnter = vi.fn();
    authServiceSwitchOrganizationMock.mockResolvedValue({ data: undefined });
    renderPicker(multiOrgOptions, onEnter);

    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));

    await waitFor(() => {
      expect(messageErrorMock).toHaveBeenCalledWith(
        '进入所选组织失败，请稍后重试',
      );
    });
    expect(onEnter).not.toHaveBeenCalled();
  });
});

describe('resolveLoginOrganizationOptions', () => {
  it('登录响应携带候选列表时优先使用并保留默认标记', () => {
    const choices: API.OrganizationChoice[] = [
      {
        organizationId: 'org-1',
        organizationName: '天津公司',
        organizationCode: 'TJ',
        isDefault: true,
      },
      {
        organizationId: 'org-2',
        organizationName: '北京公司',
        organizationCode: 'BJ',
        isDefault: false,
      },
    ];

    expect(resolveLoginOrganizationOptions(choices)).toEqual([
      {
        organizationId: 'org-1',
        organizationName: '天津公司',
        organizationCode: 'TJ',
        isDefault: true,
      },
      {
        organizationId: 'org-2',
        organizationName: '北京公司',
        organizationCode: 'BJ',
        isDefault: false,
      },
    ]);
  });

  it('无候选列表时回退 principal organizations，会话组织视为默认', () => {
    const currentUser: API.CurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-bj', code: 'BJ', name: '北京公司' },
      organizations: [
        { id: 'org-bj', code: 'BJ', name: '北京公司' },
        { id: 'org-cd', code: 'CD', name: '成都公司' },
      ],
    };

    expect(resolveLoginOrganizationOptions(undefined, currentUser)).toEqual([
      {
        organizationId: 'org-bj',
        organizationName: '北京公司',
        organizationCode: 'BJ',
        isDefault: true,
      },
      {
        organizationId: 'org-cd',
        organizationName: '成都公司',
        organizationCode: 'CD',
        isDefault: false,
      },
    ]);
  });

  it('回退路径过滤缺失 ID 的组织，避免无效候选', () => {
    const currentUser: API.CurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', code: 'TJ', name: '天津公司' },
      organizations: [{ code: 'BJ', name: '北京公司' }],
    };

    expect(resolveLoginOrganizationOptions(undefined, currentUser)).toEqual([]);
  });

  it('单组织候选直接得到长度 1 的列表（调用方据此直进）', () => {
    const currentUser: API.CurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', code: 'TJ', name: '天津公司' },
      organizations: [{ id: 'org-1', code: 'TJ', name: '天津公司' }],
    };

    expect(
      resolveLoginOrganizationOptions(undefined, currentUser),
    ).toHaveLength(1);
  });
});
