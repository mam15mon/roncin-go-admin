import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HOUSE, SectionCards } from './sea-create-layout.fixtures';
import { revealSeaFormErrors } from './sea-form-error-reveal';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
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

describe('分单错误按实际位置揭示', () => {
  it('新建号码错误保留 MBL 页签，签发方错误切到 HBL', async () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    await act(async () => {
      revealSeaFormErrors([
        { name: ['seaHouseBill', 'houseNo'], errors: ['分单号不能为空'] },
      ]);
    });
    expect(screen.getByRole('tab', { name: /主单 \(MBL\)/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getAllByPlaceholderText('请输入分单号')).toHaveLength(1);
    expect(screen.getByPlaceholderText('请输入分单号')).toBeVisible();
    await act(async () => {
      revealSeaFormErrors([
        {
          name: ['seaHouseBill', 'issuerSource'],
          errors: ['请选择分单签发主体'],
        },
      ]);
    });
    expect(screen.getByRole('tab', { name: /分单 \(HBL\)/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
  });
  it('详情号码错误保留当前提单页签并定位主单旁输入', async () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        isDetail
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    await act(async () => {
      revealSeaFormErrors([
        { name: ['seaHouseBill', 'houseNo'], errors: ['分单号不能为空'] },
      ]);
    });
    expect(screen.getByRole('tab', { name: /主单 \(MBL\)/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getByPlaceholderText('请输入分单号')).toBeVisible();
  });
});
