import { ProForm, ProFormText } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from '@testing-library/react';
import { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { useSeaOrderSave } from './use-sea-order-save';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceUpdateOrder: vi.fn(),
}));
const update = vi.mocked(orderServiceUpdateOrder);
function Harness({ reload }: { reload: () => Promise<void> }) {
  const save = useSeaOrderSave('sea:order-1', reload);
  const [result, setResult] = useState(false);
  return (
    <>
      <ProForm
        submitter={{ searchConfig: { submitText: '保存' } }}
        disabled={save.frozen}
        onFinish={async (values) => {
          try {
            const ok = await save.submit({
              id: 'order-1',
              expectedVersion: '3',
              notes: values.notes,
            });
            setResult(ok);
            return ok;
          } catch {
            return false;
          }
        }}
      >
        <ProFormText name="notes" label="备注" initialValue="草稿" />
      </ProForm>
      <output>{save.status}</output>
      <output>{result ? '完成' : '未完成'}</output>
      <button
        type="button"
        onClick={async () => {
          try {
            setResult(await save.submit());
          } catch {}
        }}
      >
        重试
      </button>
    </>
  );
}
beforeEach(() => vi.clearAllMocks());
describe('整页保存提交生命周期', () => {
  it('真实表单onFinish一次写入后刷新，失败保留输入', async () => {
    update.mockRejectedValueOnce({ response: { status: 409 } });
    const reload = vi.fn().mockResolvedValue(undefined);
    render(<Harness reload={reload} />);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /保\s*存/ })),
    );
    await waitFor(() => expect(update).toHaveBeenCalledTimes(1));
    expect(screen.getByLabelText('备注')).toHaveValue('草稿');
    expect(reload).not.toHaveBeenCalled();
    update.mockResolvedValue({ data: { id: 'order-1' } });
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /保\s*存/ })),
    );
    await screen.findByText('完成');
    expect(update).toHaveBeenCalledTimes(2);
    expect(reload).toHaveBeenCalledTimes(1);
  });
  it('已写入但刷新失败只重试读取', async () => {
    update.mockResolvedValue({ data: { id: 'order-1' } });
    const reload = vi
      .fn()
      .mockRejectedValueOnce(new Error('读取失败'))
      .mockResolvedValue(undefined);
    render(<Harness reload={reload} />);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /保\s*存/ })),
    );
    await screen.findByText('refresh');
    expect(screen.getByLabelText('备注')).toBeDisabled();
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: '重试' })),
    );
    await screen.findByText('完成');
    expect(update).toHaveBeenCalledTimes(1);
    expect(reload).toHaveBeenCalledTimes(2);
  });
  it('网络结果未知重试使用相同幂等键与载荷', async () => {
    update
      .mockRejectedValueOnce(new Error('网络超时'))
      .mockResolvedValue({ data: { id: 'order-1' } });
    render(<Harness reload={vi.fn().mockResolvedValue(undefined)} />);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /保\s*存/ })),
    );
    await screen.findByText('unknown');
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: '重试' })),
    );
    await screen.findByText('完成');
    expect(update.mock.calls[0]).toEqual(update.mock.calls[1]);
  });
  it('结果未知后切换A→B→A不会恢复没有提交快照的冻结状态', async () => {
    update.mockRejectedValueOnce(new Error('网络超时'));
    const reload = vi.fn().mockResolvedValue(undefined);
    const { result, rerender } = renderHook(
      ({ identity }) => useSeaOrderSave(identity, reload),
      { initialProps: { identity: 'A' } },
    );
    await act(async () => {
      await expect(
        result.current.submit({ id: 'order-A', expectedVersion: '3' }),
      ).rejects.toThrow();
    });
    expect(result.current.status).toBe('unknown');
    rerender({ identity: 'B' });
    rerender({ identity: 'A' });
    expect(result.current.status).toBe('idle');
    expect(result.current.frozen).toBe(false);
  });
  it('服务端拒绝保存时保留中文业务原因', async () => {
    update.mockRejectedValueOnce(
      Object.assign(new Error('Request failed with status code 409'), {
        response: {
          status: 409,
          data: { message: '分单已被更新，请刷新后重试' },
        },
      }),
    );
    const { result } = renderHook(() => useSeaOrderSave('A', vi.fn()));
    await act(async () => {
      await expect(
        result.current.submit({ id: 'order-A', expectedVersion: '3' }),
      ).rejects.toThrow('分单已被更新，请刷新后重试');
    });
    expect(result.current.frozen).toBe(false);
  });
});
