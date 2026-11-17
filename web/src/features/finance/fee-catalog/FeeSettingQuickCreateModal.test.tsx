import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import { FeeSettingQuickCreateModal } from './FeeSettingQuickCreateModal';

const api = vi.hoisted(() => ({ createFee: vi.fn(), createTaxable: vi.fn() }));
vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceCreateFeeSetting: api.createFee,
  feeCatalogServiceCreateTaxableService: api.createTaxable,
  feeCatalogServiceListTaxableServices: vi.fn(),
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
function mount(onSuccess = vi.fn()) {
  renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <FeeSettingQuickCreateModal
        open
        onCancel={vi.fn()}
        onSuccess={onSuccess}
        sessionIdentity="fees:order-1"
        currencies={[{ code: 'CNY', name: '人民币' }]}
        billingUnits={[{ id: 'unit-1', name: '票' }]}
        taxableServices={[
          { id: 'taxable-1', name: '货运代理服务', enabled: true },
        ]}
      />
    </ConfigProvider>,
    { user },
  );
  return onSuccess;
}
async function choose(label: string, text: string) {
  fireEvent.mouseDown(await screen.findByLabelText(label));
  fireEvent.click(await screen.findByText(text));
}
async function fillFee(rate: string) {
  fireEvent.change(await screen.findByLabelText('科目代码'), {
    target: { value: 'OCEAN' },
  });
  fireEvent.change(screen.getByLabelText('科目中文名称'), {
    target: { value: '海运费' },
  });
  await choose('默认币种', 'CNY (人民币)');
  await choose('默认计费单位', '票');
  await choose('默认增值税税率', rate);
  await choose('应税服务类别', '货运代理服务');
}
beforeEach(() => {
  vi.clearAllMocks();
  api.createFee.mockResolvedValue({
    success: true,
    data: { id: 'fee-new', feeCode: 'OCEAN', nameZh: '海运费' },
  });
  api.createTaxable.mockResolvedValue({
    success: true,
    data: {
      id: 'taxable-new',
      name: '新增代理服务',
      defaultTaxRate: '6',
      enabled: true,
    },
  });
});

describe('费用科目快速创建', () => {
  it.each([
    ['0% (零税率/免税)', '0'],
    ['6% (现代服务业/货运代理)', '6'],
    ['9% (基础交通运输)', '9'],
    ['13% (商品贸易/修箱)', '13'],
  ])('按百分数提交税率 %s', async (label, expected) => {
    const onSuccess = mount();
    await fillFee(label);
    fireEvent.click(screen.getByRole('button', { name: '保存并选用' }));
    await waitFor(() =>
      expect(onSuccess).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'fee-new' }),
      ),
    );
    expect(api.createFee).toHaveBeenCalledWith(
      expect.objectContaining({
        feeCode: 'OCEAN',
        taxRate: expected,
        taxableServiceId: 'taxable-1',
      }),
      expect.anything(),
    );
  });

  it('在科目表单中新增应税劳务后直接选用并保留科目输入', async () => {
    mount();
    fireEvent.change(await screen.findByLabelText('科目代码'), {
      target: { value: 'HANDLING' },
    });
    fireEvent.change(screen.getByLabelText('科目中文名称'), {
      target: { value: '操作费' },
    });
    fireEvent.mouseDown(screen.getByLabelText('应税服务类别'));
    fireEvent.click(
      await screen.findByRole('button', { name: /新增货物或应税劳务/ }),
    );
    const dialogs = await screen.findAllByRole('dialog');
    const childDialog = dialogs.find((dialog) =>
      within(dialog).queryByLabelText('货物或应税劳务名称'),
    );
    if (!childDialog) throw new Error('应税劳务创建弹窗未打开');
    const child = within(childDialog);
    fireEvent.change(child.getByLabelText('货物或应税劳务名称'), {
      target: { value: '新增代理服务' },
    });
    fireEvent.change(child.getByLabelText('默认税率（%）'), {
      target: { value: '6' },
    });
    fireEvent.click(child.getByRole('button', { name: '保存并选用' }));
    await waitFor(() =>
      expect(api.createTaxable).toHaveBeenCalledWith(
        expect.objectContaining({ name: '新增代理服务', defaultTaxRate: '6' }),
        expect.anything(),
      ),
    );
    expect(await screen.findByTitle('新增代理服务')).toBeVisible();
    expect(screen.getByLabelText('科目代码')).toHaveValue('HANDLING');
    expect(screen.getByLabelText('科目中文名称')).toHaveValue('操作费');
    expect(api.createFee).not.toHaveBeenCalled();
  });
});
