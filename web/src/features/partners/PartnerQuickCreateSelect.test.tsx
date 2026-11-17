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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind, PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import {
  PartnerQuickCreateSelect,
  type PartnerQuickCreateSelectProps,
} from './PartnerQuickCreateSelect';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

const user: API.CurrentUser = {
  id: 'user-1',
  username: 'operator',
  currentOrganization: {
    id: 'company-1',
    name: '测试公司',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'business.partner.create', dataScope: 'organization' },
    { key: 'business.partner.read', dataScope: 'organization' },
  ],
};

function Host({
  sessionIdentity = 'single',
  ...props
}: Partial<PartnerQuickCreateSelectProps>) {
  const [value, setValue] = useState('');
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <PartnerQuickCreateSelect
        aria-label="往来单位"
        sessionIdentity={sessionIdentity}
        role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
        value={value}
        onChange={setValue}
        {...props}
      />
      <output data-testid="selected">{value}</output>
    </ConfigProvider>
  );
}

async function openCreate() {
  fireEvent.mouseDown(screen.getByLabelText('往来单位'));
  const create = await screen.findByText('快捷新增往来单位');
  const createButton = create.closest('button');
  if (!createButton) throw new Error('新增按钮不存在');
  fireEvent.click(createButton);
  const input = await screen.findByLabelText('单位全称');
  fireEvent.change(input, { target: { value: '  新供应商  ' } });
  const dialog = input.closest('[role="dialog"]') as HTMLElement;
  const save = within(dialog).getByRole('button', { name: '保存并选用' });
  await waitFor(() => expect(save).toBeVisible());
  return { input, save, dialog };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('往来单位快捷创建', () => {
  it('保存固定供应商角色并回填ID与名称', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      success: true,
      data: { id: 'new-partner', legalName: '新供应商' },
    } as never);
    renderWithApp(<Host />, { user });
    const { save } = await openCreate();
    fireEvent.click(save);
    await waitFor(() =>
      expect(screen.getByTestId('selected')).toHaveTextContent('new-partner'),
    );
    expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
      {
        legalName: '新供应商',
        roles: [
          { type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER, enabled: true },
        ],
        isCasual: true,
      },
      { signal: expect.any(AbortSignal) },
    );
    expect(screen.getByTitle('新供应商')).toBeVisible();
  });

  it('多选追加新档案并通过标准onChange返回全部选中项', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      success: true,
      data: { id: 'new-partner', legalName: '新供应商' },
    } as never);
    const changed = vi.fn();
    function Multiple() {
      const [value, setValue] = useState(['existing']);
      return (
        <ConfigProvider theme={{ token: { motion: false } }}>
          <PartnerQuickCreateSelect<string[]>
            aria-label="往来单位"
            mode="multiple"
            value={value}
            options={[{ value: 'existing', label: '既有单位' }]}
            role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
            sessionIdentity="multiple"
            onChange={(next, options) => {
              setValue(next);
              changed(next, options);
            }}
          />
        </ConfigProvider>
      );
    }
    renderWithApp(<Multiple />, { user });
    fireEvent.click((await openCreate()).save);
    await waitFor(() =>
      expect(changed).toHaveBeenCalledWith(
        ['existing', 'new-partner'],
        [
          expect.objectContaining({ value: 'existing' }),
          expect.objectContaining({ value: 'new-partner' }),
        ],
      ),
    );
  });

  it('创建失败保留输入，重试成功才回填', async () => {
    vi.mocked(partnerServiceCreatePartner)
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({
        success: true,
        data: { id: 'retry-partner' },
      } as never);
    renderWithApp(<Host />, { user });
    const { input, save } = await openCreate();
    fireEvent.click(save);
    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(1),
    );
    await waitFor(() => expect(save).not.toHaveClass('ant-btn-loading'));
    expect(input).toHaveValue('  新供应商  ');
    expect(screen.getByTestId('selected')).toBeEmptyDOMElement();
    fireEvent.click(save);
    await waitFor(() =>
      expect(screen.getByTestId('selected')).toHaveTextContent('retry-partner'),
    );
  });

  it('关闭或切换父会话后拒绝迟到创建结果', async () => {
    let resolve!: (value: never) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((complete) => {
          resolve = complete;
        }),
    );
    const view = renderWithApp(<Host sessionIdentity="first" />, { user });
    fireEvent.click((await openCreate()).save);
    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(1),
    );
    const signal = vi.mocked(partnerServiceCreatePartner).mock.calls[0][1]
      ?.signal;
    view.rerender(<Host sessionIdentity="second" />);
    await waitFor(() => expect(signal?.aborted).toBe(true));
    await act(async () =>
      resolve({ success: true, data: { id: 'stale' } } as never),
    );
    expect(screen.getByTestId('selected')).toBeEmptyDOMElement();
  });

  it('财务候选确认失败后只重试确认，不重复创建档案', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      success: true,
      data: { id: 'new-partner', legalName: '新供应商' },
    } as never);
    const resolveOption = vi
      .fn()
      .mockResolvedValueOnce(undefined)
      .mockResolvedValueOnce({ value: 'new-partner', label: '新供应商' });
    renderWithApp(<Host resolveCreatedOption={resolveOption} />, { user });
    const { save } = await openCreate();
    fireEvent.click(save);
    await screen.findByText(
      '档案已创建，但不符合当前候选条件，请核对后重新选择',
    );
    await waitFor(() => expect(save).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(save);
    await waitFor(() =>
      expect(screen.getByTestId('selected')).toHaveTextContent('new-partner'),
    );
    expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(1);
    expect(resolveOption).toHaveBeenCalledTimes(2);
  });

  it('无建档权限时仍可选择已有单位，但没有新增入口', async () => {
    renderWithApp(
      <Host options={[{ value: 'existing', label: '既有单位' }]} />,
      { user: { ...user, permissionCapabilities: [] } },
    );
    fireEvent.mouseDown(screen.getByLabelText('往来单位'));
    await screen.findByText('既有单位');
    expect(screen.queryByText('快捷新增往来单位')).not.toBeInTheDocument();
    await act(async () => {});
  });

  it('字段显式disabled=false仍继承父表单禁用状态，隐藏新增入口且不查询候选', async () => {
    const request = vi.fn().mockResolvedValue([]);
    renderWithApp(
      <Form disabled>
        <Host disabled={false} request={request} />
      </Form>,
      { user },
    );
    expect(screen.getByLabelText('往来单位')).toBeDisabled();
    expect(screen.queryByText('快捷新增往来单位')).not.toBeInTheDocument();
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
    await act(async () => {});
    expect(request).not.toHaveBeenCalled();
  });

  it('身份未就绪或非公司工作台不请求远程候选', async () => {
    const request = vi.fn().mockResolvedValue([]);
    const missing = renderWithApp(<Host request={request} />);
    await act(async () => {});
    expect(request).not.toHaveBeenCalled();
    missing.unmount();
    const system = renderWithApp(<Host request={request} />, {
      user: {
        ...user,
        currentOrganization: {
          id: 'system',
          kind: AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
        },
      },
    });
    await act(async () => {});
    expect(request).not.toHaveBeenCalled();
    system.unmount();
    renderWithApp(<Host request={request} />, { user });
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
  });

  it('成功响应缺少必填ID时明确提示且保留输入', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      success: true,
      data: {},
    } as never);
    renderWithApp(<Host />, { user });
    const { save, input } = await openCreate();
    fireEvent.click(save);
    await screen.findByText('创建结果缺少伙伴 ID，请核对后重试');
    expect(input).toHaveValue('  新供应商  ');
    expect(screen.getByTestId('selected')).toBeEmptyDOMElement();
  });
});
