import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import QuickAddPartnerModal from './QuickAddPartnerModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

describe('QuickAddPartnerModal', () => {
  beforeEach(() => {
    vi.mocked(partnerServiceCreatePartner).mockReset();
  });

  it('以单选角色和默认散客创建伙伴并回填，不强制统一社会信用代码', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-1',
        legalName: '费用测试单位',
        code: 'P00000001',
        isCasual: true,
      } as never,
    });
    const onSuccess = vi.fn();

    render(
      <App>
        <QuickAddPartnerModal open onCancel={vi.fn()} onSuccess={onSuccess} />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('单位全称'), {
      target: { value: '  费用测试单位  ' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保存并选用/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith({
        legalName: '费用测试单位',
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            enabled: true,
          },
        ],
        isCasual: true,
      }),
    );
    expect(onSuccess).toHaveBeenCalledWith({
      id: 'partner-1',
      name: '费用测试单位',
      code: 'P00000001',
      isCasual: true,
    });
  });

  it('支持 defaultRole 预选供应商且可取消单次合作', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-2',
        legalName: '车队供应商',
        code: 'P00000002',
        isCasual: false,
      } as never,
    });
    const onSuccess = vi.fn();

    render(
      <App>
        <QuickAddPartnerModal
          open
          defaultRole={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
          onCancel={vi.fn()}
          onSuccess={onSuccess}
        />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('单位全称'), {
      target: { value: '车队供应商' },
    });
    const checkbox = screen.getByLabelText('单次合作往来单位（散客）');
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();

    fireEvent.click(screen.getByRole('button', { name: /保存并选用/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith({
        legalName: '车队供应商',
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
            enabled: true,
          },
        ],
        isCasual: false,
      }),
    );
  });
  it('请求层已处理的失败不重复提示，保留输入后可重试成功', async () => {
    vi.mocked(partnerServiceCreatePartner)
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({
        data: { id: 'partner-retry', legalName: '重试单位' },
      });
    const onSuccess = vi.fn();
    render(
      <App>
        <QuickAddPartnerModal open onCancel={vi.fn()} onSuccess={onSuccess} />
      </App>,
    );
    fireEvent.change(screen.getByLabelText('单位全称'), {
      target: { value: '重试单位' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保存并选用/ }));
    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /取\s*消/ }),
      ).not.toBeDisabled(),
    );
    expect(screen.getByLabelText('单位全称')).toHaveValue('重试单位');
    expect(onSuccess).not.toHaveBeenCalled();
    expect(document.querySelector('.ant-message-notice')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /保存并选用/ }));
    await waitFor(() =>
      expect(onSuccess).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'partner-retry' }),
      ),
    );
    expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(2);
  });

  it('非空响应缺少结果时仍提示缺少伙伴 ID 并保留输入', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({});
    const onSuccess = vi.fn();
    render(
      <App>
        <QuickAddPartnerModal open onCancel={vi.fn()} onSuccess={onSuccess} />
      </App>,
    );
    fireEvent.change(screen.getByLabelText('单位全称'), {
      target: { value: '缺结果单位' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保存并选用/ }));
    expect(
      await screen.findByText('创建结果缺少伙伴 ID，请重试'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('单位全称')).toHaveValue('缺结果单位');
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
