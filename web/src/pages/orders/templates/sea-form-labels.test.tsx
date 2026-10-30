import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, screen } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { HOUSE, props, SectionCards } from './sea-create-layout.fixtures';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
}));

vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
}));

vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

props.cargoCategoryOptions = [{ label: '普货', value: 'general' }];

it.each([false, true])(
  '海运表单分组使用无障碍名称且所有标签均关联实际控件（详情：%s）',
  async (isDetail) => {
    const { container } = render(
      <SectionCards
        isDetail={isDetail}
        exposeForm={() => {}}
        initialValues={{
          seaDocumentStructure: HOUSE,
          containerRequests: isDetail
            ? [{ id: 'request-1', containerSpecId: '40HQ', quantity: 2 }]
            : [],
          cargoValue: isDetail ? '100' : undefined,
          cargoCurrency: isDetail ? 'CNY' : undefined,
        }}
      />,
    );
    await act(async () => {});
    const invalidLabels = Array.from(
      container.querySelectorAll<HTMLLabelElement>('label'),
    )
      .filter((label) => !label.control)
      .map((label) => ({ label: label.textContent, target: label.htmlFor }));
    expect(invalidLabels).toEqual([]);
    const unnamedFields = Array.from(
      container.querySelectorAll<HTMLInputElement>('input,select,textarea'),
    )
      .filter((field) => !field.id && !field.name)
      .map((field) => field.outerHTML);
    expect(unnamedFields).toEqual([]);
    for (const name of ['集运/跨境', '托运类型', '提单模式', '分单签发主体']) {
      expect(
        screen.getByRole('radiogroup', { name, hidden: true }),
      ).toBeInTheDocument();
    }
    for (const name of ['箱型箱量', '货物品类']) {
      expect(
        screen.getAllByRole('group', { name, hidden: true })[0],
      ).toBeInTheDocument();
    }
  },
);
