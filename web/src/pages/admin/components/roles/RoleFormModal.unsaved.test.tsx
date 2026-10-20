import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { useRef, useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import RoleFormModal from './RoleFormModal';

const services = vi.hoisted(() => ({ create: vi.fn() }));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateRole: services.create,
  adminServiceUpdateRole: vi.fn(),
}));
afterEach(_clearAllTabCloseGuards);

function Harness() {
  const [open, setOpen] = useState(true);
  const [keys, setKeys] = useState<string[]>([]);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  return (
    <RoleFormModal
      open={open}
      onOpenChange={setOpen}
      formRef={formRef}
      selectedPermissionKeys={keys}
      setSelectedPermissionKeys={setKeys}
      permissions={[
        { key: 'business.order.read', name: '查看订单', group: '订单管理' },
      ]}
      onSuccess={vi.fn()}
    />
  );
}

describe('真实角色输入的组织切换保护', () => {
  it('仅编辑名称即可提示；取消和切换失败保留名称，下次仍提醒', async () => {
    renderWithApp(<Harness />);
    const name = await screen.findByLabelText('角色名称');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(name, { target: { value: '北京管理员' } });
    const switchOrganization = vi.fn().mockRejectedValue(new Error('切换失败'));
    const confirm = vi.fn();
    confirmIfAnyTabDirty(switchOrganization, confirm);
    confirm.mock.calls[0][0].onCancel?.();
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(name).toHaveValue('北京管理员');
    await act(async () => {
      await expect(confirm.mock.calls[0][0].onOk()).rejects.toThrow('切换失败');
    });
    expect(name).toHaveValue('北京管理员');
    expect(confirmIfAnyTabDirty(switchOrganization, confirm)).toBe(false);
    expect(confirm).toHaveBeenCalledTimes(2);
  });

  it('搜索和展开仅浏览不提醒，只勾权限也必须保护，关闭后清理', async () => {
    renderWithApp(<Harness />);
    fireEvent.change(await screen.findByLabelText('搜索角色权限'), {
      target: { value: '订单' },
    });
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    expect(getAnyDirtyTabKey()).toBeDefined();
    expect(screen.getByText('已选 1 / 1 项')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });

  it('保存失败不清理名称和矩阵的两个守卫，成功保存后都清理', async () => {
    services.create
      .mockResolvedValueOnce(undefined)
      .mockResolvedValueOnce({ data: { id: 'role-new' } });
    renderWithApp(<Harness />);
    fireEvent.change(await screen.findByLabelText('角色名称'), {
      target: { value: '管理员' },
    });
    fireEvent.click(screen.getByRole('button', { name: /全选读写/ }));
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(services.create).toHaveBeenCalledTimes(1));
    expect(screen.getByLabelText('角色名称')).toHaveValue('管理员');
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });
});
