import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { expect, it, vi } from 'vitest';
import { buildSeaExportCreateDefaults } from '../order-kinds/sea-export/form-adapter';
import { SectionCards } from './sea-create-layout.fixtures';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));

it('新建统一初始化条款，恢复草稿与用户清空后切换主分单不重新回填', async () => {
  const consoleError = vi.spyOn(console, 'error');
  const consoleWarn = vi.spyOn(console, 'warn');
  let form!: FormInstance;
  const defaults = buildSeaExportCreateDefaults({ cargoCategoryOptions: [] });
  try {
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'cargoCommercial', 'seaDocument']}
        initialValues={defaults}
      />,
    );
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: /分单 \(HBL\)/ })).toBeVisible(),
    );
    const prefixes = [['seaMasterBillContent'], ['seaHouseBill', 'content']];
    expect(form.getFieldValue('paymentTerm')).toBe(1);
    for (const prefix of prefixes) {
      expect(form.getFieldValue(prefix)).toMatchObject({
        freightTerms: 'FREIGHT PREPAID',
        transportTerms: 'CY - CY',
        billForm: 'ORIGINAL',
        releaseType: '电放',
      });
    }

    // 与 OrderFormTemplate 恢复草稿相同：初始化完成后写入草稿完整 store。
    act(() => {
      form.setFieldsValue({
        paymentTerm: 2,
        seaMasterBillContent: {
          freightTerms: 'FREIGHT COLLECT',
          billForm: null,
        },
        seaHouseBill: {
          content: { transportTerms: 'CFS - CFS', releaseType: '' },
        },
      });
      form.setFieldValue(['seaMasterBillContent', 'releaseType'], undefined);
    });
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
      ).not.toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    fireEvent.click(await screen.findByRole('tab', { name: /分单 \(HBL\)/ }));
    expect(form.getFieldValue('paymentTerm')).toBe(2);
    expect(form.getFieldValue(['seaMasterBillContent', 'freightTerms'])).toBe(
      'FREIGHT COLLECT',
    );
    expect(form.getFieldValue(['seaMasterBillContent', 'billForm'])).toBeNull();
    expect(
      form.getFieldValue(['seaMasterBillContent', 'releaseType']),
    ).toBeUndefined();
    expect(
      form.getFieldValue(['seaHouseBill', 'content', 'transportTerms']),
    ).toBe('CFS - CFS');
    expect(form.getFieldValue(['seaHouseBill', 'content', 'releaseType'])).toBe(
      '',
    );

    act(() => form.resetFields());
    expect(form.getFieldValue('paymentTerm')).toBe(1);
    for (const prefix of prefixes)
      expect(form.getFieldValue(prefix)).toEqual(defaults.seaMasterBillContent);
    const warnings = [...consoleError.mock.calls, ...consoleWarn.mock.calls]
      .flat()
      .join('\n');
    expect(warnings).not.toContain("Form already set 'initialValues'");
  } finally {
    consoleError.mockRestore();
    consoleWarn.mockRestore();
  }
});
