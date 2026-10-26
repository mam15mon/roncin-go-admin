import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import type { FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { SectionCards } from './sea-create-layout.fixtures';
import { revealSeaFormErrors } from './sea-form-error-reveal';

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

describe('SE 就近折叠备注', () => {
  it('空备注初始收起、有内容初始展开，收起显示有备注且值保留', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
        initialValues={{ bookingNotes: '已有订舱备注', operationNotes: '' }}
      />,
    );

    const bookingNotes = screen.getByTestId('sea-notes-bookingNotes');
    expect(
      within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
    ).toBeVisible();
    // 空备注初始收起：输入框挂载但不可见，值不丢失、可展开编辑。
    const operationNotes = screen.getByTestId('sea-notes-operationNotes');
    expect(
      within(operationNotes).getByPlaceholderText('请输入操作备注'),
    ).not.toBeVisible();

    fireEvent.click(within(bookingNotes).getByRole('button', { name: '收起' }));
    expect(within(bookingNotes).getByText('有备注')).toBeVisible();
    expect(
      within(bookingNotes).queryByPlaceholderText('请输入订舱备注'),
    ).not.toBeVisible();
    expect(form.getFieldValue('bookingNotes')).toBe('已有订舱备注');

    fireEvent.click(within(bookingNotes).getByRole('button', { name: '展开' }));
    expect(
      within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
    ).toBeVisible();
    // 手动展开空备注后可直接录入。
    fireEvent.click(
      within(operationNotes).getByRole('button', { name: '展开' }),
    );
    fireEvent.change(
      within(operationNotes).getByPlaceholderText('请输入操作备注'),
      { target: { value: '新操作备注' } },
    );
    expect(form.getFieldValue('operationNotes')).toBe('新操作备注');
    await act(async () => {});
  });

  it('异步回填到达后自动展开；用户手动收起后数据更新不强制重新展开', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
      />,
    );
    const allocationNotes = screen.getByTestId('sea-notes-allocationNotes');
    expect(
      within(allocationNotes).getByPlaceholderText('请输入配舱备注'),
    ).not.toBeVisible();

    // 模拟详情回填/草稿恢复：数据到达后有内容自动展开。
    await act(async () => {
      form.setFieldsValue({ allocationNotes: '回填的配舱备注' });
    });
    await waitFor(
      () =>
        expect(
          within(allocationNotes).getByPlaceholderText('请输入配舱备注'),
        ).toBeVisible(),
      { timeout: 5000 },
    );

    // 用户手动收起后，普通重绘（再次回填同值）不强制重新展开。
    fireEvent.click(
      within(allocationNotes).getByRole('button', { name: '收起' }),
    );
    await act(async () => {
      form.setFieldsValue({ allocationNotes: '回填的配舱备注-新' });
    });
    expect(
      within(allocationNotes).queryByPlaceholderText('请输入配舱备注'),
    ).not.toBeVisible();
    expect(within(allocationNotes).getByText('有备注')).toBeVisible();
    expect(form.getFieldValue('allocationNotes')).toBe('回填的配舱备注-新');
  });

  it('校验失败定位链路按首个错误展开折叠备注', async () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        sectionKeys={['bookingTransport']}
        initialValues={{ bookingNotes: '已有订舱备注' }}
      />,
    );
    const bookingNotes = screen.getByTestId('sea-notes-bookingNotes');
    fireEvent.click(within(bookingNotes).getByRole('button', { name: '收起' }));
    expect(
      within(bookingNotes).queryByPlaceholderText('请输入订舱备注'),
    ).not.toBeVisible();

    act(() => {
      revealSeaFormErrors([{ name: ['bookingNotes'], errors: ['校验失败'] }]);
    });
    await waitFor(() =>
      expect(
        within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
      ).toBeVisible(),
    );
  });
});
