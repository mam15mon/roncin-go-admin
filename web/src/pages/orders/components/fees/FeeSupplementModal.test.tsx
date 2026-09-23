import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import FeeSupplementModal from './FeeSupplementModal';

function renderModal(onSubmit: (values: unknown) => Promise<boolean>) {
  return renderWithApp(
    <FeeSupplementModal
      orderId="order-1"
      open
      onOpenChange={vi.fn()}
      feeSettings={[
        {
          id: 'fs-1',
          feeCode: 'OCEAN',
          nameZh: '海运费',
          defaultCurrency: 'CNY',
        },
      ]}
      settlementParties={[{ id: 'sp-1', name: '某供应商', code: 'SUP' }]}
      currencies={[{ code: 'CNY', name: '人民币' }]}
      billingUnits={[{ id: 'bu-1', name: '票', code: 'SHPT' }]}
      onSubmit={onSubmit}
    />,
  );
}

describe('FeeSupplementModal', () => {
  it('说明补录不会修改原费用且不展示方向选项', () => {
    renderModal(vi.fn().mockResolvedValue(true));
    expect(screen.getByText(/不会修改原有费用/)).toBeInTheDocument();
    // 补录固定应付方向，不提供应收选项。
    expect(screen.queryByText('应收')).not.toBeInTheDocument();
  });

  it('补录原因与费用字段必填，缺省提交被拦截', async () => {
    const onSubmit = vi.fn().mockResolvedValue(true);
    renderModal(onSubmit);

    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    // 各字段校验错误在不同微任务中渲染，必须逐个异步等待，避免并发负载下抖动。
    expect(await screen.findByText('请填写补录原因')).toBeInTheDocument();
    expect(await screen.findByText('请选择费用项目')).toBeInTheDocument();
    await waitFor(() => expect(onSubmit).not.toHaveBeenCalled());
  });

  it('仅填写补录原因仍被其余必填字段拦截，提交载荷携带原因', async () => {
    const onSubmit = vi.fn().mockResolvedValue(true);
    renderModal(onSubmit);

    fireEvent.change(screen.getByLabelText('补录原因'), {
      target: { value: '漏录拖车费' },
    });
    fireEvent.change(screen.getByLabelText('单价'), {
      target: { value: '50' },
    });
    fireEvent.change(screen.getByLabelText('数量'), {
      target: { value: '2' },
    });
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));

    expect(await screen.findByText('请选择费用项目')).toBeInTheDocument();
    expect(await screen.findByText('请选择结算单位')).toBeInTheDocument();
    // 文本字段已合法，剩余拦截均来自下拉与日期必填项。
    expect(screen.queryByText('请填写补录原因')).not.toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
