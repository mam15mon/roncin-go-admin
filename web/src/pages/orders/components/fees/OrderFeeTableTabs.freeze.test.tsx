import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import {
  defaultFeeColumnPreference,
  saveFeeColumnPreference,
} from './feeColumnPreference';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceAddFee: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
  orderFeeServiceBulkUpdateOrderFees: vi.fn(),
  orderFeeServiceBulkRemoveOrderFees: vi.fn(),
  orderFeeServiceListOrderFeeTagOptions: vi.fn(),
  orderFeeServiceBatchAssignOrderFeeTags: vi.fn(),
  orderFeeServiceBatchRemoveOrderFeeTags: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: { totalAmount: 0, count: 0 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: [],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: false,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    columnSettingScope: scope,
  };
}

const scope = { userId: 'freeze-user', organizationId: 'freeze-org-a' };
const storageKey = 'roncin:order-fee-columns:v1:freeze-user:freeze-org-a';

async function openSettings(tableIndex = 0) {
  await act(async () => {
    fireEvent.click(
      screen.getAllByRole('button', { name: '列设置' })[tableIndex],
    );
  });
  return screen.findByRole('switch', { name: '冻结首列' });
}

describe('订单费用首尾冻结即时设置', () => {
  beforeEach(() => {
    window.localStorage.clear();
    listFees.mockReset();
    listFees.mockResolvedValue({
      data: [{ id: 'fee-1', version: '1', direction: 1, feeName: '海运费' }],
    } as Awaited<ReturnType<typeof listFees>>);
  });

  it('应收改开关后即时保存，刷新与应付入口均共享冻结和原显隐', async () => {
    const props = {
      ...makeProps('order-freeze'),
      columnSettingScope: scope,
    };
    const rendered = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    const first = await openSettings();
    expect(first).toHaveAttribute('aria-checked', 'false');
    const last = screen.getByRole('switch', { name: '冻结尾列' });
    expect(last).toHaveAttribute('aria-checked', 'true');
    await act(async () => {
      fireEvent.click(first);
    });
    await act(async () => {
      fireEvent.click(last);
    });
    expect(first).toHaveAttribute('aria-checked', 'true');
    expect(last).toHaveAttribute('aria-checked', 'false');
    const feeCodeLabel = screen
      .getAllByText('费用代码')
      .find((element) => element.closest('label'));
    const feeCodeCheckbox = feeCodeLabel
      ?.closest('label')
      ?.querySelector('input');
    expect(feeCodeCheckbox).not.toBeNull();
    await act(async () => {
      fireEvent.click(feeCodeCheckbox as HTMLInputElement);
    });
    expect(screen.getByText('当前：费用名称')).toBeInTheDocument();
    expect(screen.getByText('当前：操作')).toBeInTheDocument();
    expect(
      screen.queryAllByRole('columnheader', { name: '费用代码' }),
    ).toHaveLength(0);
    const stored = JSON.parse(
      window.localStorage.getItem(storageKey) as string,
    );
    expect(stored.freeze).toEqual({ first: true, last: false });
    expect(stored.hidden).toContain('tags');
    expect(stored.hidden).toContain('feeCode');
    expect(stored.order.at(-1)).toBe('option');
    rendered.unmount();

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    await openSettings(1);
    expect(screen.getByRole('switch', { name: '冻结首列' })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    expect(screen.getByRole('switch', { name: '冻结尾列' })).toHaveAttribute(
      'aria-checked',
      'false',
    );
    expect(screen.queryByRole('checkbox', { name: /操作/ })).toBeNull();
    expect(screen.getByText('当前：费用名称')).toBeInTheDocument();
  });

  it('同实例切换组织立即应用对应顺序和开关，恢复默认同时恢复冻结并清除偏好', async () => {
    saveFeeColumnPreference(scope, {
      ...defaultFeeColumnPreference(false),
      order: [
        'note',
        ...defaultFeeColumnPreference(false).order.filter(
          (key) => key !== 'note',
        ),
      ],
      freeze: { first: true, last: false },
    });
    const props = makeProps('order-scope');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} columnSettingScope={scope} />
      </App>,
    );
    await screen.findByText('海运费');
    const titles = () =>
      Array.from(document.querySelector('thead')?.querySelectorAll('th') ?? [])
        .map((header) => header.textContent)
        .filter((title) => title);
    expect(titles()[0]).toBe('备注');
    rerender(
      <App>
        <OrderFeeTableTabs
          {...props}
          columnSettingScope={{ ...scope, organizationId: 'freeze-org-b' }}
        />
      </App>,
    );
    expect(titles()[0]).toBe('费用代码');
    await openSettings();
    expect(screen.getByRole('switch', { name: '冻结首列' })).toHaveAttribute(
      'aria-checked',
      'false',
    );
    expect(screen.getByRole('switch', { name: '冻结尾列' })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    await act(async () => {
      fireEvent.click(screen.getByRole('switch', { name: '冻结首列' }));
    });
    const organizationBKey =
      'roncin:order-fee-columns:v1:freeze-user:freeze-org-b';
    expect(window.localStorage.getItem(organizationBKey)).not.toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /恢复默认/ }));
    });
    expect(screen.getByRole('switch', { name: '冻结首列' })).toHaveAttribute(
      'aria-checked',
      'false',
    );
    expect(window.localStorage.getItem(organizationBKey)).toBeNull();
    expect(
      JSON.parse(window.localStorage.getItem(storageKey) as string).freeze,
    ).toEqual({
      first: true,
      last: false,
    });
  });

  it('冻结写入失败时保留当前开关效果并明确提示未持久保存', async () => {
    render(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-storage-failure')}
          columnSettingScope={scope}
        />
      </App>,
    );
    await screen.findByText('海运费');
    const first = await openSettings();
    const original = window.localStorage.setItem;
    window.localStorage.setItem = vi.fn(() => {
      throw new Error('本地存储已满');
    });
    try {
      await act(async () => {
        fireEvent.click(first);
      });
      expect(first).toHaveAttribute('aria-checked', 'true');
      await waitFor(() => {
        expect(
          screen.getByText(
            '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
          ),
        ).toBeInTheDocument();
      });
      expect(window.localStorage.getItem(storageKey)).toBeNull();
    } finally {
      window.localStorage.setItem = original;
    }
  });
});

import type { ActionType } from '@ant-design/pro-components';
