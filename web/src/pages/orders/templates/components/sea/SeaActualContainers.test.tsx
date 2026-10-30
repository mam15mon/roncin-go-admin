import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Form, Input } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import type { OrderDetailFeaturesContext } from '../../../order-kinds/types';
import { SeaActualContainers } from './SeaActualContainers';

const drawer = vi.hoisted(() => ({
  props: {} as Record<string, unknown>,
  open: vi.fn(),
}));
vi.mock('../../../components/drawers/ContainerDrawer', async () => {
  const React = await import('react');
  return {
    default: React.forwardRef((props: Record<string, unknown>, ref) => {
      drawer.props = props;
      React.useImperativeHandle(ref, () => ({ open: drawer.open }));
      return (
        <button
          type="button"
          disabled={false}
          onClick={() => (props.onChanged as () => void)()}
        >
          模拟箱保存完成
        </button>
      );
    }),
  };
});
vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceListContainers: vi.fn(),
}));
const list = vi.mocked(orderContainerServiceListContainers);
function setup({
  read = true,
  locked = false,
  shipmentType = 1,
}: {
  read?: boolean;
  locked?: boolean;
  shipmentType?: number;
} = {}) {
  const context: OrderDetailFeaturesContext = {
    order: {
      id: 'order-1',
      shipmentType,
      organizationId: 'org-1',
      seaMasterBill: { transportExecutionId: 'te-1' },
    },
    orderId: 'order-1',
    orderFormIdentity: 'sea:order-1',
    kind: 'sea-export',
    businessWritesDisabled: locked,
    canOrder: (operation) => (operation === 'container.read' ? read : true),
    containerSpecOptions: [{ label: '40HQ', value: 'spec-1' }],
    searchLocations: async () => [],
    searchShippingLines: async () => [],
    refreshOrderAndLock: vi.fn(),
    ensureBusinessWriteAllowed: () => true,
  };
  const shared = vi.fn();
  const client = createTestQueryClient();
  render(
    <QueryClientProvider client={client}>
      <Form initialValues={{ note: '已保存' }} disabled={locked}>
        <Form.Item name="note" label="订单备注">
          <Input />
        </Form.Item>
        <SeaActualContainers context={context} onOpenShared={shared} />
      </Form>
    </QueryClientProvider>,
  );
  return { context, shared };
}
beforeEach(() => {
  vi.clearAllMocks();
  list.mockResolvedValue({ data: [] });
});
describe('详情实际箱入口', () => {
  it('独占箱摘要读取成功才显示数量，箱写入刷新摘要保留整页输入和计划', async () => {
    list.mockResolvedValueOnce({
      data: [
        { id: 'box-1', packageCount: 20, grossWeightKg: 200, volumeCbm: 3 },
      ],
    });
    const { context, shared } = setup();
    await screen.findByText('独占箱 1 箱 · 20 件 · 200 KGS · 3 CBM');
    fireEvent.change(screen.getByLabelText('订单备注'), {
      target: { value: '未保存录入' },
    });
    fireEvent.click(screen.getByRole('button', { name: '普通实际箱' }));
    expect(drawer.open).toHaveBeenCalledWith(context.order);
    expect(drawer.props.hideSharedEntry).toBe(true);
    list.mockResolvedValueOnce({ data: [{ id: 'box-1' }, { id: 'box-2' }] });
    await act(async () => fireEvent.click(screen.getByText('模拟箱保存完成')));
    await screen.findByText('独占箱 2 箱 · 0 件 · 0 KGS · 0 CBM');
    expect(screen.getByLabelText('订单备注')).toHaveValue('未保存录入');
    expect(context.refreshOrderAndLock).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '跨订单共享箱' }));
    expect(shared).toHaveBeenCalledOnce();
  });
  it('没有读权限时不读取、不提供入口', async () => {
    setup({ read: false });
    await act(async () => {});
    expect(list).not.toHaveBeenCalled();
    expect(screen.queryByText('普通实际箱')).not.toBeInTheDocument();
  });
  it('审核锁仅允许查看，关闭所有写入能力', async () => {
    setup({ locked: true });
    await screen.findByText(/独占箱 0 箱/);
    expect(drawer.props.canCreate).toBe(false);
    expect(drawer.props.canUpdate).toBe(false);
    expect(drawer.props.canRemove).toBe(false);
    expect(screen.getByRole('button', { name: '普通实际箱' })).toBeEnabled();
  });
  it('拼箱订单不提供普通独占箱写入，仍可进入共享箱', async () => {
    setup({ shipmentType: 2 });
    await screen.findByText(/独占箱 0 箱/);
    expect(drawer.props.canCreate).toBe(false);
    expect(drawer.props.canUpdate).toBe(false);
    expect(drawer.props.canRemove).toBe(false);
    expect(screen.getByRole('button', { name: '跨订单共享箱' })).toBeEnabled();
  });
  it('明确成功但省略空 repeated data 时展示真实零箱', async () => {
    list.mockResolvedValueOnce({ success: true });
    setup();
    await screen.findByText(/独占箱 0 箱/);
    expect(screen.queryByText('实际箱摘要加载失败')).not.toBeInTheDocument();
  });
  it('undefined 失败不显示零，可原地重试', async () => {
    list.mockResolvedValueOnce(undefined as never);
    setup();
    await screen.findByText('实际箱摘要加载失败');
    expect(screen.queryByText(/独占箱 0 箱/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    await waitFor(() =>
      expect(screen.getByText(/独占箱 0 箱/)).toBeInTheDocument(),
    );
  });
});
