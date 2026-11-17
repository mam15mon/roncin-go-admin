import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind, PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { settlementServiceListFinanceSettlementPartyOptions } from '@/services/roncin/settlementService';
import FinanceCashflowsPage from './index';

vi.mock('@/components/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: ({
    onPrimaryAction,
  }: {
    onPrimaryAction: () => void;
  }) => (
    <button type="button" onClick={onPrimaryAction}>
      登记流水
    </button>
  ),
}));
vi.mock('@/features/finance/credit-control', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/features/finance/credit-control')
  >()),
  useCreditLimitIntervention: () => false,
}));
vi.mock('@/features/master-data/currencies', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/features/master-data/currencies')
  >()),
  getCurrencyOptions: vi
    .fn()
    .mockResolvedValue([{ label: '人民币', value: 'CNY' }]),
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceSettlementPartyOptions: vi.fn(),
}));

const user: API.CurrentUser = {
  id: 'user-1',
  username: 'operator',
  currentOrganization: {
    id: 'company-1',
    name: '测试公司',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    'system.finance.cashflow.read',
    'system.finance.cashflow.create',
    'business.partner.create',
  ].map((key) => ({ key, dataScope: 'organization' })),
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
    success: true,
    data: { id: 'created-partner', legalName: '新结算单位', isCasual: true },
  } as never);
  vi.mocked(
    settlementServiceListFinanceSettlementPartyOptions,
  ).mockImplementation(
    async (params) =>
      ({
        success: true,
        data:
          params.keyword === '新结算单位'
            ? [{ id: 'created-partner', name: '新结算单位', isCasual: true }]
            : [],
      }) as never,
  );
});

describe('收付登记内创建结算单位', () => {
  it.each(['RECEIVABLE', 'PAYABLE'])(
    '%s按方向预选角色，保留流水备注并经收付用途重新校验后选用',
    async (direction) => {
      renderWithApp(
        <ConfigProvider theme={{ token: { motion: false } }}>
          <FinanceCashflowsPage />
        </ConfigProvider>,
        { user },
      );
      fireEvent.click(screen.getByText('登记流水'));
      await screen.findByLabelText('往来结算单位');
      if (direction === 'PAYABLE') {
        fireEvent.mouseDown(screen.getByLabelText('流水方向'));
        const payable = await screen.findByText('付款（流出）');
        await act(async () => fireEvent.click(payable));
      }
      fireEvent.change(screen.getByLabelText('备注'), {
        target: { value: '保留登记输入' },
      });
      fireEvent.mouseDown(screen.getByLabelText('往来结算单位'));
      const createButton = (
        await screen.findByText('快捷新增往来单位')
      ).closest('button');
      if (!createButton) throw new Error('缺少新增入口');
      fireEvent.click(createButton);
      const input = await screen.findByLabelText('单位全称');
      fireEvent.change(input, { target: { value: '新结算单位' } });
      const dialog = input.closest('[role="dialog"]') as HTMLElement;
      const save = within(dialog).getByRole('button', { name: '保存并选用' });
      await waitFor(() => expect(save).toBeVisible());
      fireEvent.click(save);
      await waitFor(() =>
        expect(screen.getByTitle('新结算单位')).toBeVisible(),
      );
      expect(screen.getByLabelText('备注')).toHaveValue('保留登记输入');
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        expect.objectContaining({
          roles: [
            {
              type:
                direction === 'PAYABLE'
                  ? PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER
                  : PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
        }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
      expect(
        settlementServiceListFinanceSettlementPartyOptions,
      ).toHaveBeenCalledWith(
        expect.objectContaining({
          purpose: 4,
          organizationId: 'company-1',
          keyword: '新结算单位',
        }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      );
    },
  );
});
