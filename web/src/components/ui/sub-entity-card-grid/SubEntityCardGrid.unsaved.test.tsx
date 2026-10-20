import { ProFormText } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
  isTabDirty,
  registerTabCloseGuard,
} from '@/components/layout/tabCloseGuard';
import ContactCardList from '@/pages/partners/components/ContactCardList';
import { SubEntityCardGrid } from './SubEntityCardGrid';

afterEach(_clearAllTabCloseGuards);

describe('共享联系人子表单未保存保护', () => {
  it('真实联系人输入在提交给父档案前受保护，关闭子表单不会覆盖父守卫', async () => {
    renderWithApp(<ContactCardList contacts={[]} onChange={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: /添加联系人/ }));
    const name = await screen.findByLabelText('姓名');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(name, { target: { value: '未提交联系人' } });
    const key = getAnyDirtyTabKey();
    expect(key).toBeDefined();
    if (!key) throw new Error('联系人未登记守卫');
    const unregisterParent = registerTabCloseGuard(key, {
      isDirty: () => true,
    });
    const confirm = vi.fn();
    const switchOrganization = vi.fn();
    confirmIfAnyTabDirty(switchOrganization, confirm);
    confirm.mock.calls[0][0].onCancel?.();
    expect(name).toHaveValue('未提交联系人');
    expect(switchOrganization).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    expect(isTabDirty(key)).toBe(true);
    unregisterParent();
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });

  it('共享CRUD写入失败和组织切换失败均保留输入，后续成功才清理', async () => {
    const save = vi.fn().mockResolvedValue(false);
    renderWithApp(
      <SubEntityCardGrid
        entityName="联系人"
        items={[]}
        onSave={save}
        renderCard={() => null}
        renderFormItems={() => <ProFormText name="name" label="联系人姓名" />}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: /添加联系人/ }));
    const input = await screen.findByLabelText('联系人姓名');
    fireEvent.change(input, { target: { value: '待保存' } });
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(save).toHaveBeenCalled());
    expect(getAnyDirtyTabKey()).toBeDefined();
    const confirm = vi.fn();
    confirmIfAnyTabDirty(async () => {
      throw new Error('切换失败');
    }, confirm);
    await act(async () => {
      await expect(confirm.mock.calls[0][0].onOk()).rejects.toThrow('切换失败');
    });
    expect(input).toHaveValue('待保存');
    expect(getAnyDirtyTabKey()).toBeDefined();
    save.mockResolvedValue(true);
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });
});
