import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ShippingAgentsPanel from './ShippingAgentsPanel';

const state = vi.hoisted(() => ({
  access: {
    isSystemWorkspace: true,
    canCreateMasterDataItems: true,
    canUpdateMasterDataItems: true,
  },
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
}));
vi.mock('@/app/access', () => ({ useAccess: () => state.access }));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListAllItems: state.list,
  masterDataServiceCreateItem: state.create,
  masterDataServiceUpdateItem: state.update,
}));

const agent: API.MasterDataItem = {
  id: 'agent-1',
  code: 'HGJ-1',
  name: '上海船代',
  nameEn: 'Shanghai Agent',
  enabled: true,
  source: 'seed',
  sortOrder: 10,
};

function renderPanel() {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <ShippingAgentsPanel />
    </ConfigProvider>,
  );
}

describe('船代公共主数据维护', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    state.access = {
      isSystemWorkspace: true,
      canCreateMasterDataItems: true,
      canUpdateMasterDataItems: true,
    };
    state.list.mockResolvedValue({ data: [agent] });
    state.create.mockResolvedValue({ data: { ...agent, id: 'agent-new' } });
    state.update.mockResolvedValue({ data: agent });
  });

  it.each([
    { system: false, write: true, notice: true },
    { system: true, write: false, notice: false },
  ])('工作台与权限共同限制写入口：%s', async ({ system, write, notice }) => {
    state.access = {
      isSystemWorkspace: system,
      canCreateMasterDataItems: write,
      canUpdateMasterDataItems: write,
    };
    renderPanel();
    expect(await screen.findByText('HGJ-1')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /新增船代/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
    expect(screen.queryByText('停用')).toBeNull();
    expect(Boolean(screen.queryByText('由系统管理员统一维护与共享'))).toBe(
      notice,
    );
  });

  it('全量包含停用记录，本地翻页与搜索不再次请求', async () => {
    state.list.mockResolvedValue({
      data: Array.from({ length: 12 }, (_, index) => ({
        ...agent,
        id: `agent-${index}`,
        code: `HGJ-${index + 1}`,
        name: `船代${index + 1}`,
        enabled: index !== 11,
      })),
    });
    renderPanel();
    expect(await screen.findByText('HGJ-10')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('2'));
    expect(await screen.findByText('HGJ-12')).toBeInTheDocument();
    expect(screen.getByText('船代12')).toBeInTheDocument();
    fireEvent.change(
      screen.getByPlaceholderText('搜索船代编码 / 中文名称 / 英文名称...'),
      {
        target: { value: 'HGJ-12' },
      },
    );
    expect(await screen.findByText('HGJ-12')).toBeInTheDocument();
    expect(state.list).toHaveBeenCalledExactlyOnceWith({ kind: 11 });
  });

  it('新增失败保留弹窗与输入，成功后刷新全量列表', async () => {
    state.create.mockResolvedValueOnce(undefined);
    renderPanel();
    await screen.findByText('HGJ-1');
    fireEvent.click(screen.getByRole('button', { name: /新增船代/ }));
    const modal = await screen.findByRole('dialog');
    fireEvent.change(within(modal).getByLabelText('船代编码'), {
      target: { value: 'NEW-1' },
    });
    fireEvent.change(within(modal).getByLabelText('中文名称'), {
      target: { value: '新船代' },
    });
    fireEvent.click(within(modal).getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(state.create).toHaveBeenCalledOnce());
    expect(within(modal).getByLabelText('中文名称')).toHaveValue('新船代');
    expect(state.list).toHaveBeenCalledOnce();
    fireEvent.click(within(modal).getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(state.list).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(state.create).toHaveBeenLastCalledWith({
      kind: 11,
      code: 'NEW-1',
      name: '新船代',
      nameEn: '',
      source: 'manual',
      sortOrder: 100,
    });
    expect(state.list).toHaveBeenCalledTimes(2);
  });

  it('编辑与启停保持来源和原编码，刷新后显示更新值', async () => {
    renderPanel();
    await screen.findByText('HGJ-1');
    fireEvent.click(screen.getByText('编辑'));
    const modal = await screen.findByRole('dialog');
    expect(within(modal).getByLabelText('船代编码')).toBeDisabled();
    fireEvent.change(within(modal).getByLabelText('中文名称'), {
      target: { value: '修改船代' },
    });
    state.list.mockResolvedValue({ data: [{ ...agent, name: '修改船代' }] });
    fireEvent.click(within(modal).getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(state.list).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(await screen.findByText('修改船代')).toBeInTheDocument();
    expect(state.update).toHaveBeenCalledWith(
      { id: 'agent-1' },
      {
        id: 'agent-1',
        kind: 11,
        name: '修改船代',
        nameEn: 'Shanghai Agent',
        source: 'seed',
        sortOrder: 10,
        enabled: true,
      },
    );
    fireEvent.click(screen.getByText('停用'));
    state.list.mockResolvedValue({
      data: [{ ...agent, name: '修改船代', enabled: false }],
    });
    fireEvent.click(await screen.findByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(state.update).toHaveBeenCalledTimes(2));
    expect(state.update).toHaveBeenLastCalledWith(
      { id: 'agent-1' },
      expect.objectContaining({
        name: '修改船代',
        enabled: false,
        source: 'seed',
      }),
    );
    expect(await screen.findByText('启用')).toBeInTheDocument();
  });
});
