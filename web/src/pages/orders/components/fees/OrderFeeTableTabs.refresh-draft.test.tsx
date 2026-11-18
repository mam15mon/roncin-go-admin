import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { useRef, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  captureFormDraftSession,
  getFormDraft,
  getFormDraftKey,
  getFormDraftScope,
  saveFormDraft,
} from '@/components/layout/formDraft';
import type { OrderFeeDraftActions } from './OrderFeeTableTabs';
import {
  makeProps,
  OrderFeeTableTabs,
  setupFeeColumnsTests,
} from './OrderFeeTableTabs.feeColumns.fixtures';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

function RefreshHarness() {
  const [revision, setRevision] = useState(0);
  const draftActionsRef = useRef<OrderFeeDraftActions | undefined>(undefined);
  return (
    <App>
      <button
        type="button"
        onClick={() => {
          draftActionsRef.current?.discard();
          setRevision((value) => value + 1);
        }}
      >
        明确刷新
      </button>
      <OrderFeeTableTabs
        {...makeProps('order-1')}
        key={revision}
        feeWritesDisabled={false}
        draftActionsRef={draftActionsRef}
        reloadToken={revision}
      />
    </App>
  );
}

async function noteInput() {
  const row = await waitFor(() => {
    const element = document.querySelector('tr[data-row-key="fee-tax-1"]');
    expect(element).not.toBeNull();
    return element as HTMLElement;
  });
  return within(row).findByPlaceholderText('备注（可选）');
}

describe('费用明确刷新与页签草稿', () => {
  setupFeeColumnsTests();
  it('恢复后确认刷新真正清理两侧草稿，旧实例不能回写，也不清无关工作台', async () => {
    window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
    const path = window.location.pathname;
    const scope = getFormDraftScope('user-9', 'org-9');
    const receivableKey = getFormDraftKey(path, `${path}:receivable`, scope);
    const payableKey = getFormDraftKey(path, `${path}:payable`, scope);
    const otherKey = getFormDraftKey(path, `${path}:workbench`, scope);
    saveFormDraft(otherKey, { note: '独立工作台' });
    const first = render(<RefreshHarness />);
    fireEvent.change(await noteInput(), {
      target: { value: '明确放弃的费用输入' },
    });
    first.unmount();
    render(<RefreshHarness />);
    await waitFor(async () =>
      expect(await noteInput()).toHaveValue('明确放弃的费用输入'),
    );
    const oldSession = captureFormDraftSession(receivableKey);
    saveFormDraft(payableKey, { note: '应付草稿也放弃' });
    await act(async () => fireEvent.click(screen.getByText('明确刷新')));
    await waitFor(async () => expect(await noteInput()).toHaveValue(''));
    expect(getFormDraft(receivableKey)).toBeNull();
    expect(getFormDraft(payableKey)).toBeNull();
    expect(
      saveFormDraft(receivableKey, { note: '旧卸载迟到回写' }, oldSession),
    ).toBe(false);
    expect(getFormDraft(otherKey)).toEqual({ note: '独立工作台' });
  });
});
