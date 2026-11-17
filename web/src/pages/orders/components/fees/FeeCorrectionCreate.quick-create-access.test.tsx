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
import { AuthOrganizationKind } from '@/enums.generated';
import FeeCorrectionCreate from './FeeCorrectionCreate';

const api = vi.hoisted(() => ({ createFee: vi.fn(), correction: vi.fn() }));
vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceCreateFeeSetting: api.createFee,
  feeCatalogServiceListTaxableServices: vi.fn().mockResolvedValue({
    success: true,
    data: [{ id: 'taxable-1', name: '货运代理服务', enabled: true }],
  }),
}));
vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceCreateOrderFeeCorrection: api.correction,
  orderFeeServiceResolveFeeExchangeRate: vi
    .fn()
    .mockResolvedValue({ exchangeRate: '1', exchangeRateSource: 'SYSTEM' }),
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
const options = {
  orderId: 'order-1',
  fees: [],
  onCreated: vi.fn(),
  feeSettings: [],
  settlementParties: [],
  currencies: [
    { code: 'CNY', name: '人民币' },
    { code: 'USD', name: '美元' },
  ],
  billingUnits: [{ id: 'unit-1', name: '票' }],
};
function Entry({ disabled = false, canCreate = true }) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <FeeCorrectionCreate
        {...options}
        disabled={disabled}
        canCreate={canCreate}
        canUpdate
        canDelete
      />
    </ConfigProvider>
  );
}
async function select(label: string, text: string) {
  const dialogs = screen.getAllByRole('dialog');
  const child = dialogs.find((dialog) =>
    within(dialog).queryByLabelText('科目代码'),
  );
  if (!child) throw new Error('科目创建弹窗未打开');
  fireEvent.mouseDown(await within(child).findByLabelText(label));
  fireEvent.click(await screen.findByText(text));
}
beforeEach(() => vi.clearAllMocks());

describe('补录已打开后的快建权限收敛', () => {
  it.each(['disabled', 'lost-create'] as const)(
    '%s时冻结补录并拒绝科目创建迟到回填，保留原输入',
    async (mode) => {
      let resolve!: (value: unknown) => void;
      api.createFee.mockImplementationOnce(
        () =>
          new Promise((done) => {
            resolve = done;
          }),
      );
      const view = renderWithApp(<Entry />, { user });
      fireEvent.click(screen.getByRole('button', { name: '申请费用补录' }));
      fireEvent.click(await screen.findByRole('button', { name: /确\s*定/ }));
      fireEvent.change(await screen.findByLabelText('补录原因'), {
        target: { value: '保留这次漏录原因' },
      });
      fireEvent.change(screen.getByLabelText('单价'), {
        target: { value: '88' },
      });
      fireEvent.mouseDown(screen.getByLabelText('费用项目'));
      fireEvent.click(
        await screen.findByRole('button', { name: /快捷新增费用科目/ }),
      );
      fireEvent.change(await screen.findByLabelText('科目代码'), {
        target: { value: 'LATE' },
      });
      fireEvent.change(screen.getByLabelText('科目中文名称'), {
        target: { value: '迟到科目' },
      });
      await select('默认币种', 'USD (美元)');
      await select('默认计费单位', '票');
      await select('默认增值税税率', '6% (现代服务业/货运代理)');
      await select('应税服务类别', '货运代理服务');
      fireEvent.click(screen.getByRole('button', { name: '保存并选用' }));
      await waitFor(() => expect(api.createFee).toHaveBeenCalledOnce());
      view.rerender(
        mode === 'disabled' ? <Entry disabled /> : <Entry canCreate={false} />,
      );
      expect(screen.getByLabelText('补录原因')).toHaveValue('保留这次漏录原因');
      expect(screen.getByLabelText('补录原因')).toBeDisabled();
      expect(screen.getByLabelText('单价')).toHaveValue('88');
      expect(screen.getByLabelText('费用项目')).toBeDisabled();
      await waitFor(() =>
        expect(screen.queryByLabelText('科目代码')).not.toBeInTheDocument(),
      );
      await act(async () => {
        resolve({
          success: true,
          data: {
            id: 'late-fee',
            feeCode: 'LATE',
            nameZh: '迟到科目',
            defaultCurrency: 'USD',
            billingUnitId: 'unit-1',
          },
        });
      });
      expect(screen.queryByTitle('迟到科目 (LATE)')).not.toBeInTheDocument();
      expect(screen.getByLabelText('补录原因')).toHaveValue('保留这次漏录原因');
      const parentDialog = screen.getByRole('dialog');
      expect(
        within(parentDialog).getByRole('button', { name: /确\s*定/ }),
      ).toBeDisabled();
      expect(api.correction).not.toHaveBeenCalled();
    },
  );
});
