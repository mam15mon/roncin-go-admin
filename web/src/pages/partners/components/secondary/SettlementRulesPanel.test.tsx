import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SettlementRulesPanel from './SettlementRulesPanel';

const state = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  row: undefined as API.PartnerSettlementRule | undefined,
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerSettlementRules: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceCreatePartnerSettlementRule: state.create,
  partnerServiceUpdatePartnerSettlementRule: state.update,
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  ResponsiveProTable: (props: {
    toolBarRender: () => React.ReactNode;
    columns: {
      valueType?: string;
      render?: (_: unknown, row: API.PartnerSettlementRule) => React.ReactNode;
    }[];
  }) => (
    <>
      {props.toolBarRender()}
      {state.row
        ? props.columns
            .find((column) => column.valueType === 'option')
            ?.render?.(undefined, {
              ...state.row,
              roleType: 1,
            } as API.PartnerSettlementRule)
        : null}
    </>
  ),
}));

const partner = { id: 'partner', roles: [{ type: 1, enabled: true }] };
function mount(currency: string) {
  renderWithApp(<SettlementRulesPanel partner={partner} canManage />, {
    user: {
      id: 'user',
      currentOrganization: { id: 'company', baseCurrency: currency },
    },
  });
}
async function submit() {
  fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
}
beforeEach(() => {
  vi.clearAllMocks();
  state.row = undefined;
  state.create.mockResolvedValue({ data: { id: 'rule' } });
  state.update.mockResolvedValue({ data: { id: 'rule' } });
});

describe('结算规则信用额度本币表单', () => {
  it('省略 isActive 的已停用规则普通保存仍停用', async () => {
    state.row = {
      id: 'inactive-rule',
      statementMode: 1,
      settlementMethod: 1,
      settlementCurrency: 'USD',
    };
    mount('CNY');
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    await screen.findByLabelText('约定结算币种');
    expect(
      screen.getByRole('switch', { name: '规则启用状态' }),
    ).not.toBeChecked();
    await submit();
    await waitFor(() =>
      expect(state.update).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
          rule: expect.objectContaining({ isActive: false }),
        }),
      ),
    );
  });
  it.each(['CNY', 'USD'])(
    '公司本币为%s时外币结算也按本币提交额度',
    async (baseCurrency) => {
      mount(baseCurrency);
      fireEvent.click(screen.getByRole('button', { name: /新增结算规则/ }));
      fireEvent.change(
        await screen.findByLabelText(`信用额度（${baseCurrency}）`),
        { target: { value: '1000.01' } },
      );
      const settlementCurrency = baseCurrency === 'CNY' ? 'USD' : 'CNY';
      fireEvent.change(screen.getByLabelText('约定结算币种'), {
        target: { value: settlementCurrency },
      });
      await submit();
      await waitFor(() =>
        expect(state.create).toHaveBeenCalledWith(
          expect.anything(),
          expect.objectContaining({
            rule: expect.objectContaining({
              settlementCurrency,
              creditLimitMinor: '100001',
              creditCurrency: baseCurrency,
            }),
          }),
        ),
      );
    },
  );

  it('异常原币额度显示原值、清空输入并要求明确重录，不能只保存其他字段改币种', async () => {
    state.row = {
      id: 'rule',
      statementMode: 1,
      settlementMethod: 1,
      settlementCurrency: 'USD',
      creditLimitMinor: '100000',
      creditCurrency: 'USD',
      isActive: true,
    };
    mount('CNY');
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    expect(
      await screen.findByText('原信用额度为 1,000.00 USD，请重新填写 CNY 额度'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('信用额度（CNY）')).toHaveValue('');
    await submit();
    expect(
      await screen.findByText('请明确重新填写本币额度，填 0 表示不设额度'),
    ).toBeInTheDocument();
    expect(state.update).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('信用额度（CNY）'), {
      target: { value: '2000' },
    });
    await submit();
    await waitFor(() =>
      expect(state.update).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
          rule: expect.objectContaining({
            creditLimitMinor: '200000',
            creditCurrency: 'CNY',
            settlementCurrency: 'USD',
          }),
        }),
      ),
    );
  });
});
