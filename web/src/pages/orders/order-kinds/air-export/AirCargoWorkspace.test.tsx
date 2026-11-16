import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, InputNumber } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import AirCargoWorkspace, { summarizeAirCargo } from './AirCargoWorkspace';

vi.mock('@/services/roncin/orderCargoItemService', () => ({
  orderCargoItemServiceListCargoItems: vi.fn(),
}));
vi.mock('@/services/roncin/airWeightService', () => ({
  airWeightServicePreviewAirWeight: vi.fn().mockResolvedValue({
    success: true,
    data: {
      available: true,
      inputFingerprint: 'current',
      suggestedChargeableWeightKg: '200',
    },
  }),
  airWeightServiceConfirmAirWeight: vi.fn(),
  airWeightServiceListAirWeightConfirmations: vi
    .fn()
    .mockResolvedValue({ success: true }),
}));
const cargo = vi.mocked(orderCargoItemServiceListCargoItems);
const order: API.Order = {
  id: 'order',
  organizationId: 'company',
  businessType: OrderBusinessType.BUSINESS_TYPE_AE,
  version: '1',
  totalPackages: 2,
  totalGrossWeightKg: 100,
  totalVolumeCbm: 1.2,
};

function mount(locked = false, onOpenCargo = () => {}) {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form initialValues={order} disabled={locked}>
        <Form.Item name="totalGrossWeightKg" label="委托毛重">
          <InputNumber />
        </Form.Item>
        <AirCargoWorkspace
          order={order}
          canReadCargo
          canUpdate={!locked}
          onOpenCargo={onOpenCargo}
        />
      </Form>
    </ConfigProvider>,
  );
}

describe('空运订单就近货量核对', () => {
  beforeEach(() => {
    cargo.mockReset();
  });

  it('父表单审核只读时可查看实际货物、切换来源与重新核对，但不能确认重量', async () => {
    cargo.mockResolvedValue({ success: true });
    const open = vi.fn();
    mount(true, open);
    await screen.findByText('暂无货物明细');
    const button = screen.getByRole('button', { name: '维护实际货物' });
    expect(button).toBeEnabled();
    fireEvent.click(button);
    expect(open).toHaveBeenCalledOnce();
    expect(screen.getByRole('radio', { name: '订单委托货量' })).toBeEnabled();
    expect(screen.getByRole('button', { name: '重新核对' })).toBeEnabled();
    expect(
      screen.queryByRole('button', { name: '确认或调整计费重' }),
    ).not.toBeInTheDocument();
  });

  it('全量实际明细汇总超过200行，非optional零值省略仍为零，错票显式失败', () => {
    const items = Array.from({ length: 205 }, (_, i) => ({
      id: `cargo-${i}`,
      orderId: 'order',
      packageCount: 1,
      grossWeightKg: 0.1,
      volumeCbm: 0.001,
    }));
    expect(summarizeAirCargo(items, 'order')).toEqual({
      packages: 205,
      gross: '20.5',
      volume: '0.205',
      count: 205,
    });
    expect(
      summarizeAirCargo([{ id: 'zero', orderId: 'order' }], 'order'),
    ).toEqual({ packages: 0, gross: '0', volume: '0', count: 1 });
    expect(() =>
      summarizeAirCargo([{ id: 'wrong', orderId: 'other' }], 'order'),
    ).toThrow('货物明细与当前订单不一致');
  });

  it('未保存委托货量明确提示且禁止重量确认，恢复原值后可确认；实际货量重查更新', async () => {
    cargo.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'cargo',
          orderId: 'order',
          packageCount: 3,
          grossWeightKg: 120,
          volumeCbm: 1.5,
        },
      ],
    });
    const view = mount();
    expect(await screen.findByText('120')).toBeVisible();
    const button = await screen.findByRole('button', {
      name: '确认或调整计费重',
    });
    expect(button).toBeEnabled();
    fireEvent.change(screen.getByLabelText('委托毛重'), {
      target: { value: '110' },
    });
    expect(await screen.findByText('委托货量有未保存修改')).toBeVisible();
    expect(button).toBeDisabled();
    fireEvent.change(screen.getByLabelText('委托毛重'), {
      target: { value: '100' },
    });
    await waitFor(() => expect(button).toBeEnabled());
    cargo.mockResolvedValue({
      success: true,
      data: [{ id: 'cargo', orderId: 'order', grossWeightKg: 130 }],
    });
    await act(async () => {
      await view.queryClient.invalidateQueries({ queryKey: ['air-weight'] });
    });
    expect(await screen.findByText('130')).toBeVisible();
  });

  it('读取失败不伪装零货量，成功省略repeated才展示无明细', async () => {
    cargo.mockResolvedValueOnce(
      undefined as unknown as API.ListCargoItemsResponse,
    );
    mount();
    expect(await screen.findByText('实际货量加载失败')).toBeVisible();
    expect(screen.queryByText('暂无货物明细')).not.toBeInTheDocument();
    cargo.mockResolvedValue({ success: true });
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    expect(await screen.findByText('暂无货物明细')).toBeVisible();
  });
});
