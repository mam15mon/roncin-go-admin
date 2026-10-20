import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { Button } from 'antd';
import { useRef } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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
}: {
  disabled?: boolean;
  billed?: boolean;
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
    await confirmQuantity();
    expect(screen.getByLabelText('数量')).toHaveValue('8');
    expect(await screen.findByText(/800\.00/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('数量'), { target: { value: '9' } });
    expect(await screen.findByText(/900\.00/)).toBeInTheDocument();
    expect(screen.getByLabelText('数量')).toHaveValue('9');
  });

  it.each([false, true])(
    '补录和更正（correction=%s）确认即时重算金额',
    async (correction) => {
      renderWithApp(
        <FeeSupplementModal
          {...options}
          orderId="order-1"
          open
          onOpenChange={vi.fn()}
          correction={correction}
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
      await confirmQuantity();
      expect(screen.getByLabelText('数量')).toHaveValue('8');
      expect(await screen.findByText(/800\.00/)).toBeInTheDocument();
    },
  );

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
