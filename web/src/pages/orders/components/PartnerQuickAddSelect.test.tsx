import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { PartnerSelectOption } from './PartnerQuickAddSelect';
import {
  openDropdown,
  partnerServiceCreatePartner,
  pushMock,
  searchPartners,
  TestHost,
  testState,
} from './PartnerQuickAddSelect.fixtures';

describe('PartnerQuickAddSelect', () => {
  it('有权限且可编辑时下拉底部显示新增入口；无权限时不显示', async () => {
    const { unmount } = render(<TestHost />);
    await openDropdown();
    const addButton = screen.getByRole('button', { name: '新增 委托单位' });
    expect(addButton).toHaveAttribute('type', 'button');
    unmount();

    testState.accessOverrides = { canCreatePartners: false };
    render(<TestHost />);
    await openDropdown();
    expect(screen.queryByText('新增 委托单位')).not.toBeInTheDocument();
  });

  it('disabled 时不显示新增入口', async () => {
    render(<TestHost disabled />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(() =>
      expect(screen.queryByText('新增 委托单位')).not.toBeInTheDocument(),
    );
  });

  it('添加公司详情不触发保存，携带编码后的公司抬头跳转', async () => {
    vi.mocked(partnerServiceCreatePartner);
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: 'AB C公司' } });
    fireEvent.click(screen.getByRole('button', { name: '添加公司详情' }));

    expect(pushMock).toHaveBeenCalledWith(
      '/partners/customers/create?legalName=AB%20C%E5%85%AC%E5%8F%B8',
    );
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
    // 弹窗关闭的 motion 起步更新由宏任务/帧回调驱动，
    // 在 act 作用域内让出事件循环，确保用例结束前全部落地。
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it('组织切换后丢弃迟到的搜索结果', async () => {
    let resolveOldSearch!: (options: PartnerSelectOption[]) => void;
    searchPartners
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveOldSearch = resolve;
          }),
      )
      .mockResolvedValue([{ label: '新组织单位', value: 'partner-new' }]);

    const { rerender } = render(<TestHost key="stable" />);
    await waitFor(() => expect(searchPartners).toHaveBeenCalledTimes(1));

    testState.organizationId = 'org-2';
    rerender(<TestHost key="stable" />);
    await act(async () => {
      resolveOldSearch([{ label: '旧组织单位', value: 'partner-old' }]);
    });

    // 组织切换会按交互保护关闭下拉；重新展开后验证新组织结果已写入受控 options。
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(
      () => {
        expect(searchPartners).toHaveBeenCalledTimes(2);
        expect(screen.getByText('新组织单位')).toBeInTheDocument();
      },
      { timeout: 10000 },
    );
    expect(screen.queryByText('旧组织单位')).not.toBeInTheDocument();
  });

  it('连续搜索时只显示最后一次请求的候选项', async () => {
    let resolveFirst!: (options: PartnerSelectOption[]) => void;
    let resolveSecond!: (options: PartnerSelectOption[]) => void;
    searchPartners
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveSecond = resolve;
          }),
      );

    render(<TestHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '最新关键字' },
    });
    await waitFor(() => expect(searchPartners).toHaveBeenCalledTimes(2));

    await act(async () => {
      resolveSecond([{ label: '最新关键字单位', value: 'partner-latest' }]);
    });
    expect(await screen.findByText('最新关键字单位')).toBeInTheDocument();

    await act(async () => {
      resolveFirst([{ label: '旧单位', value: 'partner-stale' }]);
    });
    expect(screen.queryByText('旧单位')).not.toBeInTheDocument();
  });

  it('普通选择与清空同步更新 customerCode', async () => {
    render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByText('既有单位'));

    await waitFor(() => {
      expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-0');
      expect(screen.getByTestId('customer-code')).toHaveTextContent('P0');
    });

    const selector = screen.getByRole('combobox').closest('.ant-select');
    expect(selector).not.toBeNull();
    fireEvent.mouseEnter(selector as Element);
    const clearButton = (selector as Element).querySelector(
      '.ant-select-clear',
    );
    expect(clearButton).not.toBeNull();
    fireEvent.mouseDown(clearButton as Element);
    fireEvent.click(clearButton as Element);

    await waitFor(() => {
      expect(screen.getByTestId('customer-id')).toHaveTextContent('');
      expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    });
  });
});
