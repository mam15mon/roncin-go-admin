import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getCargoAllocationRows,
  makeSplitContext,
  pageReady,
  renderSplit,
  setupSplitCargoTests,
  splitServices,
  splitState,
} from './split-cargo.fixtures';

setupSplitCargoTests();

beforeEach(() => {
  vi.spyOn(document, 'hasFocus').mockReturnValue(false);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
}

async function startForegroundChecks() {
  splitState.context = makeSplitContext(true);
  renderSplit();
  await pageReady();
  const note = screen.getByPlaceholderText(
    '可填写本次拆票说明（将永久记录在不可变拆票事件历史中）',
  );
  fireEvent.change(note, { target: { value: '检查后仍保留的拆票输入' } });
  vi.useFakeTimers();
  vi.mocked(document.hasFocus).mockReturnValue(true);
  await act(async () => {
    document.dispatchEvent(new Event('visibilitychange'));
  });
  await advance(0);
  return note;
}

describe('拆票前台变动检查', () => {
  it('回焦与15秒检查发现独立货物版本变化，订单版本不变时仍保留分配基线', async () => {
    const note = await startForegroundChecks();
    expect(splitServices.context).toHaveBeenCalledTimes(2);
    await advance(14_999);
    expect(splitServices.context).toHaveBeenCalledTimes(2);
    splitState.context = {
      ...splitState.context,
      cargoItems: [
        {
          ...splitState.context.cargoItems?.[0],
          version: '2',
          packageCount: 120,
        },
      ],
    };
    expect(splitState.context.orderVersion).toBe('1');
    await advance(1);
    expect(splitServices.context).toHaveBeenCalledTimes(3);
    // 周期请求完成后的 Query 通知在下一轮计时器落到组件。
    await advance(1);
    expect(screen.getByText(/这个订单已经发生变动，请重新刷新/)).toBeVisible();
    expect(note).toHaveValue('检查后仍保留的拆票输入');
    const { original } = getCargoAllocationRows();
    expect(original.querySelector('input')).toHaveValue('100');
    expect(
      splitServices.preview.mock.calls.at(-1)?.[1].expectedVersions
        .cargoItemVersions,
    ).toEqual({ 'cargo-1': '1' });
    expect(screen.getByText('确认执行拆票').closest('button')).toBeDisabled();
    // 提醒保持到显式刷新，即使后续读请求又返回旧版本，也不能自动恢复提交。
    splitState.context = makeSplitContext(true);
    await advance(15_000);
    expect(screen.getByText('拆票数据需要刷新')).toBeVisible();
    expect(note).toHaveValue('检查后仍保留的拆票输入');
  });

  it('窗口失焦或页面隐藏时停止，恢复前台立即检查且不采纳最新基线', async () => {
    const note = await startForegroundChecks();
    await act(async () => {
      vi.mocked(document.hasFocus).mockReturnValue(false);
      document.dispatchEvent(new Event('visibilitychange'));
    });
    await advance(45_000);
    expect(splitServices.context).toHaveBeenCalledTimes(2);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    vi.mocked(document.hasFocus).mockReturnValue(true);
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
      fireEvent(document, new Event('visibilitychange'));
    });
    await advance(45_000);
    expect(splitServices.context).toHaveBeenCalledTimes(2);
    splitState.context = { ...splitState.context, orderVersion: '2' };
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    await act(async () => {
      fireEvent(document, new Event('visibilitychange'));
    });
    await advance(0);
    expect(splitServices.context).toHaveBeenCalledTimes(3);
    expect(screen.getByText(/这个订单已经发生变动，请重新刷新/)).toBeVisible();
    expect(note).toHaveValue('检查后仍保留的拆票输入');
    expect(splitServices.context).toHaveBeenCalledTimes(3);
  });
});
