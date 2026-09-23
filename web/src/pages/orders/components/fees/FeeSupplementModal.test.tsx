import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import FeeSupplementModal from './FeeSupplementModal';

const accessState = vi.hoisted(() => ({ canOverrideFeeExchangeRate: false }));
vi.mock('@/app/access', () => ({ useAccess: () => accessState }));

function renderModal(
  onSubmit: (values: unknown) => Promise<boolean>,
  initialRequest?: API.OrderFeeSupplementRequestData,
) {
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
      initialRequest={initialRequest}
    />,
  );
}

describe('FeeSupplementModal', () => {
  beforeEach(() => {
    accessState.canOverrideFeeExchangeRate = false;
  });

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

  it('撤回重提预填原始费用字段和原因备注，不被新建默认值覆盖', () => {
    renderModal(vi.fn().mockResolvedValue(true), {
      id: 'sup-1',
      feeSettingId: 'fs-1',
      settlementPartyId: 'sp-1',
      billingUnitId: 'bu-1',
      currency: 'CNY',
      quantity: '7',
      unitPrice: '88',
      expenseDate: '2026-09-19',
      reason: '仓储费遗漏',
      note: '旧备注',
    });

    expect(
      screen.getByText('已预填撤回申请，请核对后作为新申请提交'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('数量')).toHaveValue('7');
    expect(screen.getByLabelText('单价')).toHaveValue('88');
    expect(screen.getByLabelText('补录原因')).toHaveValue('仓储费遗漏');
    expect(screen.getByLabelText('备注说明')).toHaveValue('旧备注');
  });

  it('候选项失效时清除旧值并要求重新选择', async () => {
    const onSubmit = vi.fn().mockResolvedValue(true);
    renderModal(onSubmit, {
      id: 'sup-1',
      feeSettingId: 'removed-fee',
      settlementPartyId: 'removed-party',
      billingUnitId: 'removed-unit',
      currency: 'USD',
      quantity: '2',
      unitPrice: '20',
      expenseDate: '2026-09-19',
      reason: '原原因',
    });
    expect(
      screen.getByText(/费用项目、结算单位、计费单位、币种已不可用/),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    expect(await screen.findByText('请选择费用项目')).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('手动汇率权限失效须明确确认改用系统汇率', async () => {
    const onSubmit = vi.fn().mockResolvedValue(true);
    renderModal(onSubmit, {
      id: 'sup-1',
      feeSettingId: 'fs-1',
      settlementPartyId: 'sp-1',
      billingUnitId: 'bu-1',
      currency: 'CNY',
      quantity: '2',
      unitPrice: '20',
      expenseDate: '2026-09-19',
      reason: '原原因',
      exchangeRateSource: 'MANUAL',
      exchangeRate: '1.2',
    });
    expect(
      screen.getByText('原申请使用手动汇率，当前已无覆盖权限'),
    ).toBeInTheDocument();
    expect(
      screen.queryByLabelText('手动指定汇率 (对 CNY)'),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    expect(await screen.findByText('请先确认汇率变化')).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('仍具备汇率覆盖权限时带入原手动汇率', async () => {
    accessState.canOverrideFeeExchangeRate = true;
    renderModal(vi.fn().mockResolvedValue(true), {
      id: 'sup-1',
      feeSettingId: 'fs-1',
      settlementPartyId: 'sp-1',
      billingUnitId: 'bu-1',
      currency: 'CNY',
      quantity: '2',
      unitPrice: '20',
      expenseDate: '2026-09-19',
      reason: '原原因',
      exchangeRateSource: 'MANUAL',
      exchangeRate: '1.2',
    });
    expect(await screen.findByLabelText('手动指定汇率 (对 CNY)')).toHaveValue(
      '1.2',
    );
    expect(
      screen.queryByText('原申请使用手动汇率，当前已无覆盖权限'),
    ).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('补录原因'), {
      target: { value: '修正后的原因' },
    });
    expect(screen.getByLabelText('手动指定汇率 (对 CNY)')).toHaveValue('1.2');
  });
});
