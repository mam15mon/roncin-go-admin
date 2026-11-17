import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form } from 'antd';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import { PartnerQuickCreateSelect } from './PartnerQuickCreateSelect';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

const user: API.CurrentUser = {
  id: 'operator',
  currentOrganization: {
    id: 'company',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'business.partner.create', dataScope: 'organization' },
  ],
};

function Host({ disabled = false, blocked = false } = {}) {
  const [value, setValue] = useState('');
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form disabled={disabled}>
        <PartnerQuickCreateSelect
          aria-label="结算单位"
          sessionIdentity="policy"
          value={value}
          onChange={setValue}
          resolveCreatedOption={async () => ({
            value: 'partner',
            label: '新增结算单位',
            creditExceeded: true,
          })}
          isOptionDisabled={(option) =>
            blocked && option.creditExceeded === true
          }
        />
      </Form>
    </ConfigProvider>
  );
}

async function openCreate() {
  fireEvent.mouseDown(screen.getByLabelText('结算单位'));
  fireEvent.click(await screen.findByText('快捷新增往来单位'));
  const input = await screen.findByLabelText('单位全称');
  fireEvent.change(input, { target: { value: '新增结算单位' } });
  const dialog = input.closest('[role="dialog"]') as HTMLElement;
  await waitFor(() => expect(dialog).toBeVisible());
  return within(dialog);
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
    success: true,
    data: { id: 'partner', legalName: '新增结算单位' },
  } as never);
});
afterEach(() => vi.useRealTimers());

describe('快捷创建的动态权限与候选策略', () => {
  it('父表单冻结后关闭创建会话，恢复编辑不会自动重新打开', async () => {
    const view = renderWithApp(<Host />, { user });
    await openCreate();
    view.rerender(<Host disabled />);
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    view.rerender(<Host />);
    await act(async () => {});
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
  });

  it('信用策略迟到后，新建的本地候选同步禁选', async () => {
    const view = renderWithApp(<Host />, { user });
    const dialog = await openCreate();
    fireEvent.click(dialog.getByRole('button', { name: '保存并选用' }));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    view.rerender(<Host blocked />);
    fireEvent.mouseDown(screen.getByLabelText('结算单位'));
    await waitFor(() => {
      const option = screen
        .getAllByText('新增结算单位')
        .find((element) => element.closest('[aria-disabled="true"]'));
      expect(option).toBeDefined();
    });
  });

  it('远程关键词去首尾空白并防抖，连续输入只请求最终词', async () => {
    const request = vi.fn().mockResolvedValue([]);
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <PartnerQuickCreateSelect
          aria-label="结算单位"
          sessionIdentity="search"
          request={request}
        />
      </ConfigProvider>,
      { user },
    );
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    vi.useFakeTimers();
    fireEvent.change(screen.getByLabelText('结算单位'), {
      target: { value: '  新' },
    });
    fireEvent.change(screen.getByLabelText('结算单位'), {
      target: { value: '  新公司  ' },
    });
    await act(async () => vi.advanceTimersByTime(299));
    expect(request).toHaveBeenCalledOnce();
    await act(async () => vi.advanceTimersByTime(1));
    expect(request).toHaveBeenCalledTimes(2);
    expect(request).toHaveBeenLastCalledWith('新公司', expect.any(AbortSignal));
  });
});
