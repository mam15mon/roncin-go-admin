import { describe, expect, it, vi } from 'vitest';
import { makeVersionActions } from './versionActions';

describe('makeVersionActions', () => {
  const app = {
    modal: { confirm: vi.fn() },
    message: { warning: vi.fn() },
  } as never;

  it('向动作传递记录 ID 与乐观锁版本', async () => {
    const submit = vi.fn().mockResolvedValue(undefined);
    const actions = makeVersionActions<{ id?: string; version?: string }>(app);

    await actions.run({ id: 'record-1', version: '3' }, submit);

    expect(submit).toHaveBeenCalledWith(
      { id: 'record-1', expectedVersion: '3' },
      { id: 'record-1', version: '3' },
    );
  });

  it('缺少 ID 或版本时不执行动作', async () => {
    const submit = vi.fn().mockResolvedValue(undefined);
    const actions = makeVersionActions<{ id?: string; version?: number }>(app);

    await actions.run({ version: 1 }, submit);
    await actions.run({ id: 'record-1' }, submit);

    expect(submit).not.toHaveBeenCalled();
  });

  it('仅为完整记录打开原因确认框', () => {
    const modalConfirm = vi.fn();
    const actions = makeVersionActions<{ id?: string; version?: number }>({
      modal: { confirm: modalConfirm },
      message: { warning: vi.fn() },
    } as never);

    actions.confirm({ id: 'record-1' }, '取消记录？', vi.fn());
    actions.confirm({ id: 'record-1', version: 2 }, '取消记录？', vi.fn());

    expect(modalConfirm).toHaveBeenCalledTimes(1);
    expect(modalConfirm.mock.calls[0][0].title).toBe('取消记录？');
  });
});
