import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import BatchAddressTypeModal from './BatchAddressTypeModal';
import BatchAssigneeModal from './BatchAssigneeModal';
import BatchAssociationModal from './BatchAssociationModal';

afterEach(_clearAllTabCloseGuards);

describe('企业资源批量编辑未保存保护', () => {
  it('地址默认初始化不脏，模式和地址类型真实修改均登记，失败保留且关闭清理', async () => {
    const save = vi.fn().mockResolvedValue(false);
    function Harness() {
      const [open, setOpen] = useState(true);
      const [mode, setMode] = useState<'assign' | 'remove'>('assign');
      const [types, setTypes] = useState<number[]>([1]);
      return (
        <BatchAddressTypeModal
          open={open}
          setOpen={setOpen}
          mode={mode}
          setMode={setMode}
          batchAddressTypes={types}
          setBatchAddressTypes={setTypes}
          onOk={() => void save(types)}
        />
      );
    }
    renderWithApp(<Harness />);
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('移除所选类型'));
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('checkbox', { name: '提货地址' }));
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(save).toHaveBeenCalledWith([1, 2]));
    expect(getAnyDirtyTabKey()).toBeDefined();
    expect(screen.getByRole('checkbox', { name: '提货地址' })).toBeChecked();
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });

  it('人员搜索不脏，仅修改模式即保护，真实选择在切换取消/失败后仍保留', async () => {
    const search = vi.fn().mockResolvedValue(undefined);
    function Harness() {
      const [open, setOpen] = useState(true);
      const [mode, setMode] = useState<'assign' | 'remove'>('assign');
      const [people, setPeople] = useState<string[]>([]);
      return (
        <BatchAssigneeModal
          open={open}
          setOpen={setOpen}
          mode={mode}
          setMode={setMode}
          batchAssignees={people}
          setBatchAssignees={setPeople}
          assigneeOptions={[{ label: '张三', value: 'user-1' }]}
          searchAssignees={search}
          onOk={vi.fn()}
        />
      );
    }
    renderWithApp(<Harness />);
    const inputs = screen.getAllByRole('combobox');
    fireEvent.change(inputs[1], { target: { value: '张' } });
    expect(search).toHaveBeenCalledWith('张');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.mouseDown(inputs[0]);
    fireEvent.click(await screen.findByText('移除所选人员'));
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.mouseDown(inputs[1]);
    fireEvent.click(await screen.findByText('张三'));
    const confirm = vi.fn();
    const switchOrganization = vi.fn().mockRejectedValue(new Error('切换失败'));
    confirmIfAnyTabDirty(switchOrganization, confirm);
    confirm.mock.calls[0][0].onCancel?.();
    expect(switchOrganization).not.toHaveBeenCalled();
    await act(async () => {
      await expect(confirm.mock.calls[0][0].onOk()).rejects.toThrow('切换失败');
    });
    expect(getAnyDirtyTabKey()).toBeDefined();
    expect(screen.getAllByText('张三').length).toBeGreaterThan(0);
    expect(confirmIfAnyTabDirty(switchOrganization, confirm)).toBe(false);
  });

  it('企业搜索不脏，选择关联企业后登记，父流程成功关闭才清理', async () => {
    const search = vi.fn().mockResolvedValue(undefined);
    const submit = vi.fn();
    function Harness() {
      const [open, setOpen] = useState(true);
      const [partners, setPartners] = useState<string[]>([]);
      return (
        <BatchAssociationModal
          open={open}
          setOpen={setOpen}
          mode="link"
          partners={partners}
          setPartners={setPartners}
          partnerOptions={[{ label: '荣勤货代', value: 'partner-1' }]}
          searchPartners={search}
          onOk={() => {
            submit(partners);
            setOpen(false);
          }}
        />
      );
    }
    renderWithApp(<Harness />);
    const input = screen.getByRole('combobox');
    fireEvent.change(input, { target: { value: '荣勤' } });
    expect(search).toHaveBeenCalledWith('荣勤');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.mouseDown(input);
    fireEvent.click(await screen.findByText('荣勤货代'));
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    expect(submit).toHaveBeenCalledWith(['partner-1']);
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });
});
