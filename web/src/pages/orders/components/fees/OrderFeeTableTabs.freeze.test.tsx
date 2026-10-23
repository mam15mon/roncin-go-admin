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
  return screen.findByRole('combobox', { name: '费用名称冻结位置' });
}
async function setFreeze(title: string, label: string) {
  await act(async () => {
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: `${title}冻结位置` }),
    );
  });
  await act(async () => {
    fireEvent.click(
      screen.getByText(label, { selector: '.ant-select-item-option-content' }),
    );
  });
}

describe('订单费用自定义多列冻结', () => {
  beforeEach(() => {
    window.localStorage.clear();
    listFees.mockReset();
    listFees.mockResolvedValue({
      data: [{ id: 'fee-1', version: '1', direction: 1, feeName: '海运费' }],
    } as Awaited<ReturnType<typeof listFees>>);
  });
  it('应收同时冻结两列并解除操作冻结，刷新后应付共享且保留显隐', async () => {
    const props = makeProps('order-freeze');
    const rendered = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    await openSettings();
    await setFreeze('费用名称', '左冻结');
    await setFreeze('结算单位', '左冻结');
    await setFreeze('操作', '不冻结');
    await act(async () => {
      fireEvent.click(screen.getByRole('checkbox', { name: '费用代码' }));
    });
    const stored = JSON.parse(
      window.localStorage.getItem(storageKey) as string,
    );
    expect(stored.freeze).toEqual({
      left: ['feeSettingId', 'settlementPartyId'],
      right: [],
    });
    expect(stored.hidden).toContain('feeCode');
    expect(stored.hidden).toContain('tags');
    expect(stored.order.at(-1)).toBe('option');
    rendered.unmount();
    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    await openSettings(1);
    expect(
      document
        .querySelector('[data-column-key="feeSettingId"]')
        ?.closest('[data-freeze-zone]'),
    ).toHaveAttribute('data-freeze-zone', 'left');
    expect(
      document
        .querySelector('[data-column-key="settlementPartyId"]')
        ?.closest('[data-freeze-zone]'),
    ).toHaveAttribute('data-freeze-zone', 'left');
    expect(
      screen.getByRole('checkbox', { name: '费用代码' }),
    ).not.toBeChecked();
    expect(screen.queryByRole('checkbox', { name: '操作' })).toBeNull();
  });
  it('同实例组织切换读取独立配置，恢复默认清除该组织偏好', async () => {
    saveFeeColumnPreference(scope, {
      ...defaultFeeColumnPreference(false),
      order: [
        'note',
        ...defaultFeeColumnPreference(false).order.filter(
          (key) => key !== 'note',
        ),
      ],
      freeze: { left: ['note'], right: [] },
    });
    const props = makeProps('order-scope');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    const titles = () =>
      Array.from(document.querySelector('thead')?.querySelectorAll('th') ?? [])
        .map((header) => header.textContent)
        .filter(Boolean);
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
    await setFreeze('费用名称', '左冻结');
    const key = 'roncin:order-fee-columns:v1:freeze-user:freeze-org-b';
    expect(window.localStorage.getItem(key)).not.toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    });
    expect(window.localStorage.getItem(key)).toBeNull();
    expect(
      JSON.parse(window.localStorage.getItem(storageKey) as string).freeze,
    ).toEqual({ left: ['note'], right: [] });
  });
  it('冻结写入失败保留当前页面效果并明确提示', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-storage-failure')} />
      </App>,
    );
    await screen.findByText('海运费');
    await openSettings();
    const original = window.localStorage.setItem;
    window.localStorage.setItem = vi.fn(() => {
      throw new Error('本地存储已满');
    });
    try {
      await setFreeze('费用名称', '左冻结');
      expect(
        document
          .querySelector('[data-column-key="feeSettingId"]')
          ?.closest('[data-freeze-zone]'),
      ).toHaveAttribute('data-freeze-zone', 'left');
      await waitFor(() =>
        expect(
          screen.getByText(
            '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
          ),
        ).toBeInTheDocument(),
      );
      expect(window.localStorage.getItem(storageKey)).toBeNull();
    } finally {
      window.localStorage.setItem = original;
    }
  });
});

import type { ActionType } from '@ant-design/pro-components';
