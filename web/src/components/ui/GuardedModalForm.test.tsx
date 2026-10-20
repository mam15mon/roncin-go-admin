import { ProFormText } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { Button, Form } from 'antd';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import { GuardedModalForm } from './GuardedModalForm';

afterEach(_clearAllTabCloseGuards);

describe('GuardedModalForm 官方生命周期', () => {
  it('trigger打开、自动回填不标脏；用户输入和失败保留，成功清理', async () => {
    const submit = vi.fn().mockResolvedValue(false);
    function Harness() {
      const [form] = Form.useForm();
      return (
        <GuardedModalForm
          trigger={<Button>新建联系人</Button>}
          form={form}
          initialValues={{ name: '默认姓名' }}
          modalProps={{ destroyOnHidden: true }}
          onFinish={submit}
        >
          <ProFormText name="name" label="联系人姓名" />
          <Button onClick={() => form.setFieldsValue({ name: '自动带入' })}>
            自动回填
          </Button>
        </GuardedModalForm>
      );
    }
    renderWithApp(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: '新建联系人' }));
    const input = await screen.findByLabelText('联系人姓名');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.click(screen.getByRole('button', { name: '自动回填' }));
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(input, { target: { value: '未保存联系人' } });
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(submit).toHaveBeenCalled());
    expect(input).toHaveValue('未保存联系人');
    expect(getAnyDirtyTabKey()).toBeDefined();
    submit.mockResolvedValue(true);
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });

  it('组织切换取消、确认后失败均保留原输入，再次切换仍提示', async () => {
    function Harness() {
      const [open, setOpen] = useState(true);
      return (
        <GuardedModalForm
          open={open}
          onOpenChange={setOpen}
          onFinish={async () => false}
        >
          <ProFormText name="name" label="姓名" />
        </GuardedModalForm>
      );
    }
    renderWithApp(<Harness />);
    const input = await screen.findByLabelText('姓名');
    fireEvent.change(input, { target: { value: '需要保留' } });
    const switchOrganization = vi.fn().mockRejectedValue(new Error('切换失败'));
    const modal = vi.fn();
    confirmIfAnyTabDirty(switchOrganization, modal);
    modal.mock.calls[0][0].onCancel?.();
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(input).toHaveValue('需要保留');
    await act(async () => {
      await expect(modal.mock.calls[0][0].onOk()).rejects.toThrow('切换失败');
    });
    expect(input).toHaveValue('需要保留');
    expect(confirmIfAnyTabDirty(switchOrganization, modal)).toBe(false);
    expect(modal).toHaveBeenCalledTimes(2);
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
  });
});
