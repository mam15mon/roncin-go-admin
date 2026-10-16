import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { adminServiceListPermissions } from '@/services/roncin/adminService';
import PermissionsPanel from './permissions';

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListPermissions: vi.fn(),
}));

const permissions: API.AdminPermission[] = [
  {
    key: 'finance.bill.read',
    name: '查看账单',
    group: '费用管理 · 账单',
    description: '核对往来余额',
  },
  {
    key: 'order.se.read',
    name: '查看订单',
    group: '订单管理 · 海运出口 · 订单',
  },
  {
    key: 'new.report.read',
    name: '查看账单报表',
    group: '新增业务 · 报表',
  },
];

async function search(keyword: string) {
  const calls = vi.mocked(adminServiceListPermissions).mock.calls.length;
  fireEvent.change(
    screen.getByPlaceholderText('搜索功能名称、权限码、业务分组或说明'),
    {
      target: { value: keyword },
    },
  );
  fireEvent.click(screen.getByRole('button', { name: /查\s*询/ }));
  await waitFor(() => {
    expect(adminServiceListPermissions).toHaveBeenCalledTimes(calls + 1);
  });
  await waitFor(() => {
    expect(
      document.querySelector('.ant-spin-spinning'),
    ).not.toBeInTheDocument();
  });
}

describe('权限字典查找', () => {
  beforeEach(() => {
    vi.mocked(adminServiceListPermissions).mockReset();
    vi.mocked(adminServiceListPermissions).mockResolvedValue({
      data: permissions,
    });
  });

  it('按模块与关键词组合查询，保留完整候选并可重置、恢复空结果', async () => {
    renderWithApp(<PermissionsPanel />);
    expect(await screen.findByText('查看订单')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(
      await screen.findByText('费用管理', {
        selector: '.ant-select-item-option-content',
      }),
    );
    await search('账单');
    await waitFor(() =>
      expect(screen.queryByText('查看账单报表')).not.toBeInTheDocument(),
    );
    expect(screen.getByText('查看账单')).toBeInTheDocument();
    expect(screen.queryByText('查看订单')).not.toBeInTheDocument();

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(
      await screen.findByText('新增业务', {
        selector: '.ant-select-item-option-content',
      }),
    ).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'Escape' });

    await search('不存在的功能');
    expect(
      await screen.findByText('未找到匹配的权限，请调整关键词或业务模块'),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /重\s*置/ }));
    expect(await screen.findByText('查看订单')).toBeInTheDocument();
    expect(screen.getByText('查看账单报表')).toBeInTheDocument();
  });

  it('支持权限码、说明、完整路径检索，刷新发现新增模块', async () => {
    renderWithApp(<PermissionsPanel />);
    await screen.findByText('查看订单');

    for (const keyword of [
      ' FINANCE.BILL.READ ',
      '往来余额',
      '费用管理 → 账单',
    ]) {
      await search(keyword);
      await waitFor(() =>
        expect(screen.queryByText('查看订单')).not.toBeInTheDocument(),
      );
      expect(screen.getByText('查看账单')).toBeInTheDocument();
      expect(screen.queryByText('查看账单报表')).not.toBeInTheDocument();
    }

    vi.mocked(adminServiceListPermissions).mockResolvedValue({
      data: [
        ...permissions,
        { key: 'warehouse.read', name: '查看仓库', group: '仓储管理 · 仓库' },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    await waitFor(() =>
      expect(adminServiceListPermissions).toHaveBeenCalledTimes(5),
    );
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(
      await screen.findByText('仓储管理', {
        selector: '.ant-select-item-option-content',
      }),
    ).toBeInTheDocument();
    expect(screen.getByText('查看账单')).toBeInTheDocument();
  });

  it('全局处理后的空响应保持请求失败，刷新成功后恢复权限与模块', async () => {
    // 统一请求层处理错误后会返回 undefined，生成客户端尚未声明该运行时结果。
    vi.mocked(adminServiceListPermissions).mockImplementationOnce(
      vi.fn().mockResolvedValue(undefined),
    );
    renderWithApp(<PermissionsPanel />);
    await waitFor(() => {
      expect(adminServiceListPermissions).toHaveBeenCalledTimes(1);
      expect(
        document.querySelector('.ant-spin-spinning'),
      ).not.toBeInTheDocument();
    });
    expect(screen.queryByText('查看账单')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    expect(await screen.findByText('查看账单')).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(
      await screen.findByText('新增业务', {
        selector: '.ant-select-item-option-content',
      }),
    ).toBeInTheDocument();
  });
});
