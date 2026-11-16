import { renderWithApp } from '@root/tests/renderWithApp';
import { cleanup, fireEvent, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { orderDetailPath, orderListPath } from '@/features/orders/navigation';
import { workbenchOrderPath } from './display';
import { QuickEntriesCard, TodosCard } from './WorkbenchSideCards';

const push = vi.hoisted(() => vi.fn());
vi.mock('@/router/history', () => ({ history: { push } }));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('四类工作台导航与真实类型权限', () => {
  it.each([
    ['SE', 'sea-export'],
    ['SI', 'sea-import'],
    ['AE', 'air-export'],
    ['AI', 'air-import'],
  ])('%s详情/费用路径与注册身份一致，未知类型不生成链接', (type, kind) => {
    expect(orderListPath(type)).toBe(`/orders/${kind}`);
    expect(orderDetailPath(type, 'order-id')).toBe(`/orders/${kind}/order-id`);
    expect(workbenchOrderPath(type, 'order-id', true)).toBe(
      `/orders/${kind}/order-id/fees`,
    );
    expect(orderDetailPath('LAND', 'id')).toBeUndefined();
    expect(orderDetailPath(type, undefined)).toBeUndefined();
  });

  it.each([
    ['si', '海运进口', 'sea-import'],
    ['ai', '空运进口', 'air-import'],
  ])(
    '只有%s权限也显示本类待办和快捷入口，不借出口权限',
    (code, title, kind) => {
      const user: API.CurrentUser = {
        id: 'operator',
        currentOrganization: { id: 'company', kind: 2 },
        permissionCapabilities: ['read', 'fee.read', 'abnormal_case.read'].map(
          (operation) => ({
            key: `business.order.${code}.${operation}`,
            dataScope: 'organization',
          }),
        ),
      };
      const onOpen = vi.fn();
      renderWithApp(
        <>
          <QuickEntriesCard />
          <TodosCard
            todos={{ unbilledFeeCount: 2, openAbnormalCount: 3 }}
            onOpen={onOpen}
          />
        </>,
        { user },
      );
      fireEvent.click(screen.getByText(`${title}订单`));
      expect(push).toHaveBeenCalledWith(`/orders/${kind}`);
      expect(screen.queryByText('海运出口订单')).not.toBeInTheDocument();
      expect(screen.queryByText('空运出口订单')).not.toBeInTheDocument();
      expect(screen.getByText('2 笔')).toBeVisible();
      expect(screen.getByText('3 件')).toBeVisible();
      fireEvent.click(screen.getAllByRole('button', { name: '查看明细' })[0]);
      expect(onOpen).toHaveBeenCalledWith('fees');
    },
  );

  it('只有进口订单read而缺费用/异常read不展示对应待办按钮', () => {
    renderWithApp(
      <TodosCard
        todos={{ unbilledFeeCount: 2, openAbnormalCount: 3 }}
        onOpen={vi.fn()}
      />,
      {
        user: {
          id: 'u',
          currentOrganization: { id: 'company', kind: 2 },
          permissionCapabilities: [
            { key: 'business.order.ai.read', dataScope: 'organization' },
          ],
        },
      },
    );
    expect(
      screen.queryByRole('button', { name: '查看明细' }),
    ).not.toBeInTheDocument();
  });
});
