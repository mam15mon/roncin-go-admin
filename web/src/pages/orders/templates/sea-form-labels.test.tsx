import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, screen } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { expect, it, vi } from 'vitest';
import {
  DIRECT,
  HOUSE,
  props,
  SectionCards,
} from './sea-create-layout.fixtures';

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
    for (const name of ['集运/跨境', '托运类型', '分单签发主体']) {
      expect(
        screen.getByRole('radiogroup', { name, hidden: true }),
      ).toBeInTheDocument();
    }
    if (isDetail) {
      expect(screen.getByLabelText('当前提单模式')).toHaveTextContent(
        '有货代分单（HOUSE）',
      );
      expect(
        screen.queryByRole('radiogroup', { name: '提单模式', hidden: true }),
      ).not.toBeInTheDocument();
    } else {
      expect(
        screen.getByRole('radiogroup', { name: '提单模式', hidden: true }),
      ).toBeInTheDocument();
    }
    for (const name of ['箱型箱量', '货物品类']) {
      expect(
        screen.getAllByRole('group', { name, hidden: true })[0],
      ).toBeInTheDocument();
    }
  },
);

it('详情当前模式随表单数据更新，保留提交值且使用可读标签展示', async () => {
  let form!: FormInstance;
  render(
    <SectionCards
      isDetail
      sectionKeys={['bookingTransport']}
      exposeForm={(instance) => {
        form = instance;
      }}
      initialValues={{ seaDocumentStructure: DIRECT }}
    />,
  );
  const mode = screen.getByLabelText('当前提单模式');
  expect(mode).toHaveTextContent('仅船公司主单（DIRECT）');
  expect(mode.querySelector('.ant-tag')).not.toHaveClass('ant-tag-disabled');
  expect(
    (await form.validateFields(['seaDocumentStructure'])).seaDocumentStructure,
  ).toBe(DIRECT);
  await act(async () => {
    form.setFieldValue('seaDocumentStructure', HOUSE);
  });
  expect(mode).toHaveTextContent('有货代分单（HOUSE）');
});
