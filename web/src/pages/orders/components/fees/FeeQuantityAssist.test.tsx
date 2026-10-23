import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServicePreviewFeeQuantity } from '@/services/roncin/orderFeeService';
import FeeQuantityAssist from './FeeQuantityAssist';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServicePreviewFeeQuantity: vi.fn(),
}));
const preview = vi.mocked(orderFeeServicePreviewFeeQuantity);
type Response = Awaited<ReturnType<typeof orderFeeServicePreviewFeeQuantity>>;
const available = (quantity = '8'): Response => ({
  data: {
    available: true,
    quantity,
    sourceLabel: '本订单实际货物明细',
    formula: 'max(体积, 毛重 / 1000)',
    inputs: [
      { label: '本订单体积', value: '8.0000', unit: 'CBM' },
      { label: '本订单毛重', value: '2500', unit: 'KG' },
    ],
  },
});
const unit = { id: 'unit-rt', code: 'RT', name: '计费吨' };
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe('订单数量确认带入', () => {
  beforeEach(() => {
    preview.mockReset();
    preview.mockResolvedValue(available());
  });

  it('主动查询、展示原始数据和公式，确认才填入', async () => {
    const onConfirm = vi.fn();
    renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={onConfirm}
      />,
    );
    expect(preview).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    expect(
      await screen.findByText('max(体积, 毛重 / 1000)'),
    ).toBeInTheDocument();
    expect(screen.getByText('8 CBM')).toBeInTheDocument();
    expect(screen.getByText('2500 KG')).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
    expect(preview).toHaveBeenCalledWith(
      { orderId: 'order-1', billingUnitId: 'unit-rt', source: 'ACTUAL' },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
    expect(onConfirm).toHaveBeenCalledWith('8');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('取消不填入，关闭后迟到结果不恢复弹窗', async () => {
    const response = deferred<Response>();
    preview.mockReturnValue(response.promise);
    const onConfirm = vi.fn();
    renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={onConfirm}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
    await act(async () => response.resolve(available()));
    expect(onConfirm).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('箱量切换明确来源，迟到的实际箱量不能替代计划箱量', async () => {
    const actual = deferred<Response>();
    preview.mockImplementation(({ source }) =>
      source === 'ACTUAL' ? actual.promise : Promise.resolve(available('3')),
    );
    const onConfirm = vi.fn();
    renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={{
          id: '40hq',
          code: '40HQ',
          name: '40HQ',
          isContainerUnit: true,
        }}
        onConfirm={onConfirm}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    fireEvent.click(screen.getByRole('radio', { name: '订舱计划箱量' }));
    await screen.findByText('3 40HQ');
    await act(async () => actual.resolve(available('2')));
    expect(screen.queryByText('2 40HQ')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
    expect(onConfirm).toHaveBeenCalledWith('3');
  });

  it.each(['order', 'unit'] as const)(
    '切换%s时旧建议立即关闭，迟到数据不能填入',
    async (change) => {
      const response = deferred<Response>();
      preview.mockReturnValue(response.promise);
      const onConfirm = vi.fn();
      const { rerender } = renderWithApp(
        <FeeQuantityAssist
          orderId="order-1"
          billingUnit={unit}
          onConfirm={onConfirm}
        />,
      );
      fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
      rerender(
        <FeeQuantityAssist
          orderId={change === 'order' ? 'order-2' : 'order-1'}
          billingUnit={change === 'unit' ? { ...unit, id: 'unit-cbm' } : unit}
          onConfirm={onConfirm}
        />,
      );
      await act(async () => response.resolve(available()));
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
      expect(onConfirm).not.toHaveBeenCalled();
    },
  );

  it('缺数据时解释原因且不能确认，保留人工填写路径', async () => {
    preview.mockResolvedValue({
      data: {
        available: false,
        sourceLabel: '实际登记箱量',
        reason: '本订单存在共享箱，无法确定独占箱量',
      },
    });
    renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('本订单存在共享箱，无法确定独占箱量');
    expect(screen.getByRole('button', { name: '确认带入' })).toBeDisabled();
  });

  it('冻结或表单关闭时销毁已打开建议，缺少单位时禁用入口', async () => {
    const onConfirm = vi.fn();
    const { rerender } = renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={onConfirm}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('8 计费吨');
    rerender(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        disabled
        onConfirm={onConfirm}
      />,
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.queryByText('从订单带入')).not.toBeInTheDocument();
    rerender(<FeeQuantityAssist orderId="order-1" onConfirm={onConfirm} />);
    expect(screen.getByRole('button', { name: '从订单带入' })).toBeDisabled();
  });

  it('不支持的单位入口置灰，点击不请求或打开弹窗；切换到支持单位恢复可用', async () => {
    const onConfirm = vi.fn();
    const { rerender } = renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={{ id: 'unit-ticket', name: '票', code: 'TICKET' }}
        onConfirm={onConfirm}
      />,
    );
    const trigger = screen.getByRole('button', { name: '从订单带入' });
    expect(trigger).toBeDisabled();
    fireEvent.click(trigger);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(preview).not.toHaveBeenCalled();
    expect(onConfirm).not.toHaveBeenCalled();
    rerender(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={onConfirm}
      />,
    );
    expect(screen.getByRole('button', { name: '从订单带入' })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('8 计费吨');
  });

  it('失败响应不产生可确认数量，重试后可恢复', async () => {
    preview.mockResolvedValueOnce({});
    renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('获取数量建议失败');
    expect(screen.getByRole('button', { name: '确认带入' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '确认带入' }),
      ).not.toBeDisabled(),
    );
  });

  it('已有建议重查失败时不能继续确认缓存数量，重试成功后使用新数量', async () => {
    const onConfirm = vi.fn();
    const { queryClient } = renderWithApp(
      <FeeQuantityAssist
        orderId="order-1"
        billingUnit={unit}
        onConfirm={onConfirm}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('8 计费吨');
    preview.mockResolvedValueOnce({});
    await act(async () => {
      await queryClient.invalidateQueries({ queryKey: ['orders'] });
    });
    await screen.findByText('获取数量建议失败');
    expect(screen.queryByText('8 计费吨')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认带入' })).toBeDisabled();
    expect(onConfirm).not.toHaveBeenCalled();
    preview.mockResolvedValueOnce(available('9'));
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    await screen.findByText('9 计费吨');
    fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
    expect(onConfirm).toHaveBeenCalledWith('9');
  });
});
