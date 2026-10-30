import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  fillCommissionStaff,
  getSaveButton,
  openDropdown,
  partnerServiceCreatePartner,
  savePartner,
  TestHost,
  testState,
} from './PartnerQuickAddSelect.fixtures';

describe('PartnerQuickAddSelect', () => {
  it('组织切换后本地新建选项被清理', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-1', legalName: '切换前单位' } as never,
    });
    const { rerender } = render(<TestHost key="stable" />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '切换前单位' },
    });
    await fillCommissionStaff();
    await savePartner();
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    testState.organizationId = 'org-2';
    rerender(<TestHost key="stable" />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await screen.findByText('既有单位');
    await waitFor(() =>
      expect(screen.queryByText('切换前单位')).not.toBeInTheDocument(),
    );
  });

  it('组件卸载后创建成功不再回填表单或调用变更回调', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { unmount } = render(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '卸载前新建单位' },
    });
    await fillCommissionStaff();
    await savePartner();
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    unmount();
    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-after-unmount',
          legalName: '卸载前新建单位',
          code: 'PUNMOUNT',
        },
      });
    });

    expect(onPartnerChange).not.toHaveBeenCalled();
  });

  it('组织切换后丢弃迟到的创建结果', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { rerender } = render(
      <TestHost key="stable" onPartnerChange={onPartnerChange} />,
    );
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(
      await screen.findByRole('button', { name: '新增 委托单位' }),
    );
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '旧组织新建单位' },
    });
    await fillCommissionStaff();
    await savePartner();
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    testState.organizationId = 'org-2';
    rerender(<TestHost key="stable" onPartnerChange={onPartnerChange} />);
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
    fireEvent.click(
      await screen.findByRole('button', { name: '新增 委托单位' }),
    );
    expect(await screen.findByLabelText('公司抬头')).toHaveValue('');
    expect(getSaveButton()).not.toBeDisabled();

    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-old',
          legalName: '旧组织新建单位',
          code: 'POLD',
        },
      });
    });

    expect(screen.getByTestId('customer-id')).toHaveTextContent('');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    expect(onPartnerChange).not.toHaveBeenCalled();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('');
  });

  it('保存期间切换只读状态后隔离旧请求与新弹窗', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { rerender } = render(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '锁单前新建单位' },
    });
    await fillCommissionStaff();
    await savePartner();
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    rerender(<TestHost disabled onPartnerChange={onPartnerChange} />);
    rerender(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    const reopenedNameInput = await screen.findByLabelText('公司抬头');
    fireEvent.change(reopenedNameInput, {
      target: { value: '解锁后新输入' },
    });

    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-before-lock',
          legalName: '锁单前新建单位',
          code: 'PLOCK',
        },
      });
    });

    expect(screen.getByTestId('customer-id')).toHaveTextContent('');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    expect(onPartnerChange).not.toHaveBeenCalled();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('解锁后新输入');
  });
});
