import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import FeeSupplementModal from './FeeSupplementModal';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceResolveFeeExchangeRate: vi
    .fn()
    .mockResolvedValue({ exchangeRate: '7', exchangeRateSource: 'SYSTEM' }),
}));
vi.mock('@/features/finance/fee-catalog', () => ({
  FeeSettingQuickCreateModal: ({
    open,
    onSuccess,
  }: {
    open: boolean;
    onSuccess: (setting: API.FeeSetting) => void;
  }) =>
    open ? (
      <button
        type="button"
        onClick={() =>
          onSuccess({
            id: 'fee-new',
            feeCode: 'NEW',
            nameZh: '新建操作费',
            defaultCurrency: 'USD',
            billingUnitId: 'unit-new',
            taxRate: '6',
          })
        }
      >
        完成科目创建
      </button>
    ) : null,
}));
const user: API.CurrentUser = {
  id: 'user-1',
  currentOrganization: {
    id: 'company-1',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'system.finance.fee_setting.create', dataScope: 'organization' },
  ],
};

describe('费用补录快捷创建回填', () => {
  it('创建后选中科目并带入币种与单位，同时保留数量、单价和补录原因', async () => {
    const submit = vi.fn().mockResolvedValue(false);
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <FeeSupplementModal
          open
          orderId="order-1"
          onOpenChange={vi.fn()}
          onSubmit={submit}
          feeSettings={[]}
          settlementParties={[{ id: 'party-1', name: '供应商' }]}
          currencies={[
            { code: 'CNY', name: '人民币' },
            { code: 'USD', name: '美元' },
          ]}
          billingUnits={[
            { id: 'unit-old', name: '票', code: 'SHPT' },
            { id: 'unit-new', name: '箱', code: 'BOX' },
          ]}
          initialRequest={{
            id: 'correction-1',
            settlementPartyId: 'party-1',
            quantity: '3',
            unitPrice: '20',
            currency: 'CNY',
            billingUnitId: 'unit-old',
            reason: '漏录操作费',
            note: '保留备注',
          }}
        />
      </ConfigProvider>,
      { user },
    );
    fireEvent.mouseDown(await screen.findByLabelText('费用项目'));
    fireEvent.click(
      await screen.findByRole('button', { name: /快捷新增费用科目/ }),
    );
    fireEvent.click(screen.getByRole('button', { name: '完成科目创建' }));
    expect(await screen.findByTitle('新建操作费 (NEW)')).toBeVisible();
    expect(screen.getByLabelText('数量')).toHaveValue('3');
    expect(screen.getByLabelText('单价')).toHaveValue('20');
    expect(screen.getByLabelText('补录原因')).toHaveValue('漏录操作费');
    expect(screen.getByLabelText('备注说明')).toHaveValue('保留备注');
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    await waitFor(() =>
      expect(submit).toHaveBeenCalledWith(
        expect.objectContaining({
          feeSettingId: 'fee-new',
          currency: 'USD',
          billingUnitId: 'unit-new',
          quantity: '3',
          unitPrice: '20',
          reason: '漏录操作费',
        }),
      ),
    );
  });
});
