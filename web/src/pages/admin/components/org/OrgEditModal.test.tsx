import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import OrgEditModal from './OrgEditModal';

const updateOrganization = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceUpdateOrganization: updateOrganization,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: async () => [{ label: 'CNY', value: 'CNY' }],
}));

const company: API.AdminOrganization = {
  id: 'company-1',
  code: 'TEST',
  kind: 2,
  name: '测试公司',
  nameEn: 'Test Company Co., Ltd',
  enabled: true,
  baseCurrency: 'CNY',
};

async function mount(organization = company) {
  await act(async () => {
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <OrgEditModal
          open
          onOpenChange={vi.fn()}
          editingOrg={organization}
          onSuccess={vi.fn()}
        />
      </ConfigProvider>,
    );
  });
}

async function submit() {
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
  });
  await waitFor(() => expect(updateOrganization).toHaveBeenCalled());
}

beforeEach(() => {
  vi.clearAllMocks();
  updateOrganization.mockResolvedValue({ success: true, data: company });
});

describe('公司英文名称维护', () => {
  it('回显已存英文名，修改其他资料时完整保留', async () => {
    await mount();
    expect(screen.getByLabelText('公司英文名称')).toHaveValue(company.nameEn);
    fireEvent.change(screen.getByLabelText('组织名称'), {
      target: { value: '更新中文名称' },
    });
    await submit();
    expect(updateOrganization).toHaveBeenCalledWith(
      { id: company.id },
      expect.objectContaining({
        name: '更新中文名称',
        nameEn: 'Test Company Co., Ltd',
      }),
    );
  });

  it('可明确清空英文名而不复用旧值', async () => {
    await mount();
    fireEvent.change(screen.getByLabelText('公司英文名称'), {
      target: { value: '   ' },
    });
    await submit();
    expect(updateOrganization).toHaveBeenCalledWith(
      { id: company.id },
      expect.objectContaining({ nameEn: '' }),
    );
  });

  it('超长和内部控制字符均明确拒绝，不产生写请求', async () => {
    await mount();
    const input = screen.getByLabelText('公司英文名称');
    fireEvent.change(input, { target: { value: 'A'.repeat(201) } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
    });
    await waitFor(() =>
      expect(screen.getByText('公司英文名称不能超过200个字符')).toBeVisible(),
    );
    expect(updateOrganization).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { value: 'Company\u2028Limited' } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
    });
    await waitFor(() =>
      expect(
        screen.getByText('公司英文名称不能包含换行或控制字符'),
      ).toBeVisible(),
    );
    expect(updateOrganization).not.toHaveBeenCalled();
  });

  it.each([1, 3, 4])('非公司类型%s不提供英文名称字段', async (kind) => {
    await mount({ ...company, kind });
    expect(screen.queryByLabelText('公司英文名称')).not.toBeInTheDocument();
  });
});

it.each([undefined, false, true])(
  '组织 enabled=%s 修改名称后保持原启停语义',
  async (enabled) => {
    const { enabled: _enabled, ...withoutEnabled } = company;
    await mount({
      ...withoutEnabled,
      ...(enabled === undefined ? {} : { enabled }),
    });
    expect(screen.getByRole('switch')).toHaveAttribute(
      'aria-checked',
      String(enabled === true),
    );
    fireEvent.change(screen.getByLabelText('组织名称'), {
      target: { value: '只修改名称' },
    });
    await submit();
    expect(updateOrganization).toHaveBeenCalledWith(
      { id: company.id },
      expect.objectContaining({
        name: '只修改名称',
        enabled: enabled === true,
      }),
    );
  },
);
