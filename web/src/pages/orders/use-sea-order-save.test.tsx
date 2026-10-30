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
import { orderErrorReasons } from '@/errorReasons.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { useSeaOrderSave } from './use-sea-order-save';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceUpdateOrder: vi.fn(),
}));
const update = vi.mocked(orderServiceUpdateOrder);
function Harness({ reload }: { reload: () => Promise<void> }) {
  const save = useSeaOrderSave('sea:order-1', reload);
  const [result, setResult] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
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
          } catch (error) {
            setErrorMessage(
              error instanceof Error ? error.message : '保存失败',
            );
            return false;
          }
        }}
      >
        <ProFormText name="notes" label="备注" initialValue="草稿" />
      </ProForm>
      <output>{save.status}</output>
      <output>{result ? '完成' : '未完成'}</output>
      <output>{errorMessage}</output>
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
    update.mockRejectedValueOnce({
      response: {
        status: 409,
        data: {
          reason: orderErrorReasons.ORDER_STATUS_CONFLICT,
          message: '订单状态已被其他操作修改',
        },
      },
    });
    const reload = vi.fn().mockResolvedValue(undefined);
    render(<Harness reload={reload} />);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /保\s*存/ })),
    );
    await waitFor(() => expect(update).toHaveBeenCalledTimes(1));
    expect(
      await screen.findByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeInTheDocument();
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
  it.each([
    [orderErrorReasons.SEA_HOUSE_BILL_EXISTS, '海运分单号已存在'],
    [
      orderErrorReasons.SEA_MASTER_BILL_CORRECTION_BLOCKED,
      '主单存在下游事实，不能更正',
    ],
    [
      orderErrorReasons.SEA_MASTER_BILL_VOYAGE_CONFLICT,
      '航程信息与已有主单冲突',
    ],
  ])('非并发拒绝 %s 保留中文业务原因', async (reason, message) => {
    update.mockRejectedValueOnce(
      Object.assign(new Error('Request failed with status code 409'), {
        response: {
          status: 409,
          data: { reason, message },
        },
      }),
    );
    const { result } = renderHook(() => useSeaOrderSave('A', vi.fn()));
    await act(async () => {
      await expect(
        result.current.submit({ id: 'order-A', expectedVersion: '3' }),
      ).rejects.toThrow(message);
    });
    expect(result.current.frozen).toBe(false);
  });

  describe.each(['response', 'data'] as const)('%s 错误信封', (envelope) => {
    it.each([
      orderErrorReasons.ORDER_STATUS_CONFLICT,
      orderErrorReasons.SEA_MASTER_BILL_STATUS_CONFLICT,
      orderErrorReasons.SEA_HOUSE_BILL_STATUS_CONFLICT,
      orderErrorReasons.SEA_DOCUMENT_STRUCTURE_CONFLICT,
      orderErrorReasons.SEA_DOCUMENT_VERSION_CONFLICT,
      orderErrorReasons.SEA_MASTER_BILL_CONFLICT,
      orderErrorReasons.SEA_HOUSE_BILL_CONFLICT,
    ])('%s 统一展示刷新提示且不刷新或重试', async (reason) => {
      const data = { code: 409, reason, message: '服务端内部并发提示' };
      update.mockRejectedValueOnce(
        envelope === 'response'
          ? { response: { status: 409, data } }
          : { data },
      );
      const reload = vi.fn();
      const { result } = renderHook(() => useSeaOrderSave('A', reload));
      const payload = {
        id: 'order-A',
        expectedVersion: '3',
        notes: '未保存输入',
      };
      await act(async () => {
        await expect(result.current.submit(payload)).rejects.toThrow(
          '这个订单已经发生变动，请重新刷新。',
        );
      });
      expect(result.current.status).toBe('idle');
      expect(result.current.frozen).toBe(false);
      expect(update).toHaveBeenCalledOnce();
      expect(reload).not.toHaveBeenCalled();
    });
  });
});
