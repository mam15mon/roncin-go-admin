import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { Button } from 'antd';
import { useRef } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import {
  orderFeeServicePreviewFeeQuantity,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { useFeeExchangePreview } from '../../use-fee-exchange-preview';
import FeeFormModal, { type FeeFormValues } from './FeeFormModal';
import FeeSupplementModal from './FeeSupplementModal';

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOverrideFeeExchangeRate: false }),
}));
vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServicePreviewFeeQuantity: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const options = {
  feeSettings: [
    {
      id: 'fs',
      feeCode: 'LCL',
      nameZh: '拼箱运费',
      defaultBillingUnitId: 'unit',
      defaultCurrency: 'CNY',
    },
  ],
  settlementParties: [{ id: 'sp', name: '结算客户' }],
  currencies: [{ code: 'CNY', name: '人民币' }],
  billingUnits: [{ id: 'unit', name: '计费吨', code: 'RT' }],
};

/** 普通新增和复制共用表单，按真实父组件使用金额预览 Hook。 */
function OrdinaryForm({
  disabled = false,
  billed = false,
  directSource = false,
}: {
  disabled?: boolean;
  billed?: boolean;
  directSource?: boolean;
}) {
  const formRef = useRef<ProFormInstance<FeeFormValues>>(undefined);
  const amount = useFeeExchangePreview('order-1', formRef);

  return (
    <>
      <Button
        onClick={() =>
          formRef.current?.setFieldsValue({
            quantity: '1',
            unitPrice: '100',
            billingUnitId: 'unit',
            currency: 'CNY',
            quantitySource: directSource
              ? {
                  mode: 'DIRECT',
                  confirmationId: 'confirmation',
                  confirmationVersion: '1',
                  referenceQuantity: '1',
                }
              : undefined,
          })
        }
      >
        载入复制费用
      </Button>
      <FeeFormModal
        {...options}
        orderId="order-1"
        open
        onOpenChange={vi.fn()}
        formRef={formRef}
        quantityAssistDisabled={disabled}
        modalDirection={1}
        isFeeBilled={billed}
        totalPreview={amount.totalPreview}
        exchangeRateStatus={amount.exchangeRateStatus}
        manualExchangeRate={false}
        setManualExchangeRate={vi.fn()}
        onOpenQuickAddFee={vi.fn()}
        onOpenQuickAddPartner={vi.fn()}
        onValuesChange={amount.handleValuesChange}
        onQuantityChange={() => amount.handleValuesChange({ amountOnly: true })}
        onFeeSettingSelect={vi.fn()}
        onSubmit={vi.fn()}
      />
    </>
  );
}

async function confirmQuantity() {
  fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
  await screen.findByText('8 计费吨');
  fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
}

describe('费用数量带入表单入口', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
    vi.mocked(orderFeeServicePreviewFeeQuantity).mockResolvedValue({
      data: {
        available: true,
        quantity: '8',
        sourceLabel: '本订单实际货物明细',
        formula: 'max(8, 2500 / 1000)',
      },
    });
    vi.mocked(orderFeeServiceResolveFeeExchangeRate).mockResolvedValue({
      exchangeRate: '1',
      exchangeRateSource: 'SYSTEM',
    });
  });

  it('普通及复制新增表单：确认即时重算金额，允许继续手改', async () => {
    renderWithApp(<OrdinaryForm />);
    fireEvent.click(screen.getByRole('button', { name: '载入复制费用' }));
    expect(screen.getByLabelText('数量')).toHaveValue('1');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('8 计费吨');
    fireEvent.click(
      within(
        screen
          .getAllByRole('dialog')
          .find((dialog) =>
            within(dialog).queryByText('从订单带入数量'),
          ) as HTMLElement,
      ).getByRole('button', { name: /取\s*消/ }),
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();
    await confirmQuantity();
    const switchOrganization = vi.fn();
    const confirmation = vi.fn();
    expect(confirmIfAnyTabDirty(switchOrganization, confirmation)).toBe(false);
    expect(confirmation).toHaveBeenCalledOnce();
    confirmation.mock.calls[0][0].onCancel?.();
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
    expect(screen.getByLabelText('数量')).toHaveValue('8');
    expect(await screen.findByText(/800\.00/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('数量'), { target: { value: '9' } });
    expect(await screen.findByText(/900\.00/)).toBeInTheDocument();
    expect(screen.getByLabelText('数量')).toHaveValue('9');
  });

  it('统一费用补录确认即时重算金额', async () => {
    renderWithApp(
      <FeeSupplementModal
        {...options}
        orderId="order-1"
        open
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        initialRequest={{
          id: 'request',
          billingUnitId: 'unit',
          quantity: '1',
          unitPrice: '100',
          currency: 'CNY',
        }}
      />,
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();
    await confirmQuantity();
    const switchOrganization = vi.fn();
    expect(confirmIfAnyTabDirty(switchOrganization, vi.fn())).toBe(false);
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
    expect(screen.getByLabelText('数量')).toHaveValue('8');
    expect(await screen.findByText(/800\.00/)).toBeInTheDocument();
  });

  it('仅将确认重量来源改为纯手工也立即登记未保存保护', async () => {
    renderWithApp(<OrdinaryForm directSource />);
    fireEvent.click(screen.getByRole('button', { name: '载入复制费用' }));
    const changeSource = await screen.findByRole('button', {
      name: '改为纯手工数量',
    });
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.click(changeSource);
    expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
    expect(
      screen.queryByRole('button', { name: '改为纯手工数量' }),
    ).not.toBeInTheDocument();
  });

  it('已建账费用的来源切换同数量与单位一起禁用', async () => {
    renderWithApp(<OrdinaryForm directSource billed />);
    fireEvent.click(screen.getByRole('button', { name: '载入复制费用' }));
    expect(
      await screen.findByRole('button', { name: '改为纯手工数量' }),
    ).toBeDisabled();
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });

  it.each([{ disabled: true }, { billed: true }])(
    '普通费用冻结/已建账不显示带入入口 %j',
    async (props) => {
      renderWithApp(<OrdinaryForm {...props} />);
      fireEvent.click(screen.getByRole('button', { name: '载入复制费用' }));
      await waitFor(() =>
        expect(screen.getByLabelText('数量')).toHaveValue('1'),
      );
      expect(
        screen.queryByRole('button', { name: '从订单带入' }),
      ).not.toBeInTheDocument();
    },
  );
});
