import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import { BusinessTagModal } from './BusinessTagModal';

const services = vi.hoisted(() => ({ create: vi.fn(), groups: vi.fn() }));
vi.mock('@/services/roncin/enterpriseResourceService', () => ({
  enterpriseResourceServiceCreateEnterpriseResource: services.create,
  enterpriseResourceServiceListEnterpriseTagGroups: services.groups,
}));

const tags = [{ id: 'tag-1', name: '重点客户', enabled: true }];
const loadOptions = vi.fn().mockResolvedValue({ tags });

function Harness({
  submit = vi.fn(),
}: {
  submit?: (mode: 'assign' | 'remove', ids: string[]) => Promise<unknown>;
}) {
  const [open, setOpen] = useState(true);
  return (
    <BusinessTagModal
      open={open}
      targetCount={1}
      canQuickCreate
      loadOptions={loadOptions}
      onSubmit={submit}
      onCancel={() => setOpen(false)}
    />
  );
}

beforeEach(() => {
  services.create.mockReset();
  services.groups.mockResolvedValue({
    data: [{ id: 'group-1', name: '业务分组' }],
  });
  loadOptions.mockClear();
});
afterEach(_clearAllTabCloseGuards);

describe('业务标签待提交输入保护', () => {
  it('搜索候选不提醒，选定待关联标签后取消和切换失败保留选择', async () => {
    const submit = vi.fn().mockResolvedValue(false);
    renderWithApp(<Harness submit={submit} />);
    await waitFor(() => expect(loadOptions).toHaveBeenCalled());
    const search = screen.getByRole('combobox');
    fireEvent.change(search, { target: { value: '重点' } });
    await waitFor(() =>
      expect(loadOptions).toHaveBeenCalledWith(
        expect.objectContaining({ keyword: '重点' }),
      ),
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.mouseDown(search);
    fireEvent.click(await screen.findByText('重点客户'));
    expect(getAnyDirtyTabKey()).toBeDefined();
    const confirm = vi.fn();
    const switchOrganization = vi.fn().mockRejectedValue(new Error('切换失败'));
    confirmIfAnyTabDirty(switchOrganization, confirm);
    confirm.mock.calls[0][0].onCancel?.();
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(screen.getAllByText('重点客户').length).toBeGreaterThan(0);
    await act(async () => {
      await expect(confirm.mock.calls[0][0].onOk()).rejects.toThrow('切换失败');
    });
    expect(confirmIfAnyTabDirty(switchOrganization, confirm)).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: /添加标签/ }));
    await waitFor(() =>
      expect(submit).toHaveBeenCalledWith('assign', ['tag-1']),
    );
    expect(getAnyDirtyTabKey()).toBeDefined();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /添加标签/ })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    submit.mockResolvedValue(true);
    fireEvent.click(screen.getByRole('button', { name: /添加标签/ }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });

  it('快捷新建仅填写名称也登记保护，关闭外层清理', async () => {
    renderWithApp(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: '+ 新增标签' }));
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(await screen.findByPlaceholderText('标签名称'), {
      target: { value: '新标签' },
    });
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });

  it('快捷新建仅选择标签组也登记保护', async () => {
    renderWithApp(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: '+ 新增标签' }));
    await waitFor(() => expect(services.groups).toHaveBeenCalled());
    fireEvent.mouseDown(screen.getAllByRole('combobox')[1]);
    fireEvent.click(await screen.findByText('业务分组'));
    expect(getAnyDirtyTabKey()).toBeDefined();
  });

  it('快捷创建失败保留名称，创建成功后仍保护待关联新标签，关联成功才清理', async () => {
    services.create.mockResolvedValueOnce(undefined).mockResolvedValueOnce({
      data: { id: 'tag-new', shortName: '新标签', enabled: true },
    });
    const submit = vi.fn().mockResolvedValue(false);
    renderWithApp(<Harness submit={submit} />);
    fireEvent.click(screen.getByRole('button', { name: '+ 新增标签' }));
    fireEvent.mouseDown(screen.getAllByRole('combobox')[1]);
    fireEvent.click(await screen.findByText('业务分组'));
    const name = await screen.findByPlaceholderText('标签名称');
    fireEvent.change(name, { target: { value: '新标签' } });
    fireEvent.click(screen.getByRole('button', { name: '创 建' }));
    await waitFor(() => expect(services.create).toHaveBeenCalledTimes(1));
    expect(name).toHaveValue('新标签');
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '创 建' }));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText('标签名称')).not.toBeInTheDocument(),
    );
    expect(screen.getAllByText('新标签').length).toBeGreaterThan(0);
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: /添加标签/ }));
    await waitFor(() =>
      expect(submit).toHaveBeenCalledWith('assign', ['tag-new']),
    );
    expect(getAnyDirtyTabKey()).toBeDefined();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /添加标签/ })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    submit.mockResolvedValue(true);
    fireEvent.click(screen.getByRole('button', { name: /添加标签/ }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });
});
