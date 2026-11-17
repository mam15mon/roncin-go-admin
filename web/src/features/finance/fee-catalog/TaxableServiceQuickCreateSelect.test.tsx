import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import { TaxableServiceQuickCreateSelect } from './TaxableServiceQuickCreateSelect';

const api = vi.hoisted(() => ({ create: vi.fn() }));
vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceCreateTaxableService: api.create,
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
function Field({
  identity = 'fee-new',
  active = true,
  disabled = false,
  onChange = vi.fn(),
}) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form disabled={disabled}>
        <Form.Item label="应税劳务" name="taxableId">
          <TaxableServiceQuickCreateSelect
            taxableServices={[]}
            sessionIdentity={identity}
            active={active}
            onChange={onChange}
          />
        </Form.Item>
      </Form>
    </ConfigProvider>
  );
}
async function openCreate() {
  fireEvent.mouseDown(await screen.findByLabelText('应税劳务'));
  fireEvent.click(
    await screen.findByRole('button', { name: /新增货物或应税劳务/ }),
  );
  return within(await screen.findByRole('dialog'));
}
beforeEach(() => {
  vi.clearAllMocks();
  api.create.mockResolvedValue({
    success: true,
    data: { id: 'taxable-1', name: '代理服务', enabled: true },
  });
});

describe('应税劳务快速创建的会话', () => {
  it('失败保留输入，成功选用且下次打开清空上次输入', async () => {
    const onChange = vi.fn();
    renderWithApp(<Field onChange={onChange} />, { user });
    const form = await openCreate();
    fireEvent.change(form.getByLabelText('货物或应税劳务名称'), {
      target: { value: '代理服务' },
    });
    api.create.mockResolvedValueOnce(undefined);
    fireEvent.click(form.getByRole('button', { name: '保存并选用' }));
    await waitFor(() => expect(api.create).toHaveBeenCalledOnce());
    await waitFor(() =>
      expect(form.getByRole('button', { name: '保存并选用' })).toBeEnabled(),
    );
    expect(form.getByLabelText('货物或应税劳务名称')).toHaveValue('代理服务');
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.click(form.getByRole('button', { name: '保存并选用' }));
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith(
        'taxable-1',
        expect.objectContaining({ label: '代理服务' }),
      ),
    );
    const next = await openCreate();
    expect(next.getByLabelText('货物或应税劳务名称')).toHaveValue('');
  });

  it('关闭所属表单后拒绝迟到结果', async () => {
    let resolve!: (result: unknown) => void;
    api.create.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const onChange = vi.fn();
    const view = renderWithApp(<Field onChange={onChange} />, { user });
    const form = await openCreate();
    fireEvent.change(form.getByLabelText('货物或应税劳务名称'), {
      target: { value: '代理服务' },
    });
    fireEvent.click(form.getByRole('button', { name: '保存并选用' }));
    await waitFor(() => expect(api.create).toHaveBeenCalledOnce());
    view.rerender(<Field active={false} onChange={onChange} />);
    await act(async () => {
      resolve({ success: true, data: { id: 'late', name: '迟到服务' } });
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('父表单禁用时不提供新增入口', async () => {
    renderWithApp(<Field disabled />, { user });
    const field = await screen.findByLabelText('应税劳务');
    expect(field).toBeDisabled();
    fireEvent.mouseDown(field);
    expect(
      screen.queryByRole('button', { name: /新增货物或应税劳务/ }),
    ).not.toBeInTheDocument();
  });
  it('无新增权限时仍可使用下拉且不提供新增入口', async () => {
    renderWithApp(<Field />, { user: { ...user, permissionCapabilities: [] } });
    const field = await screen.findByLabelText('应税劳务');
    expect(field).toBeEnabled();
    fireEvent.mouseDown(field);
    expect(
      screen.queryByRole('button', { name: /新增货物或应税劳务/ }),
    ).not.toBeInTheDocument();
    expect(api.create).not.toHaveBeenCalled();
  });
});
