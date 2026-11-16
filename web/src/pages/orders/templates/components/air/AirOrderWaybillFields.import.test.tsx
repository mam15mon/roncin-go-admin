import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import { renderAirOrderContent } from './AirOrderWaybillFields.fixtures';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
}));
beforeEach(() => vi.clearAllMocks());

describe('航空正文显式带入', () => {
  it('客户和代理带入英文抬头并经父动作标脏，不改其他正文', async () => {
    vi.mocked(partnerServiceGetPartner).mockImplementation(async ({ id }) => ({
      success: true,
      data: {
        id,
        organizationId: 'company-1',
        legalName: '中文单位',
        profile: {
          nameEn: id === 'customer-1' ? 'CUSTOMER LTD' : 'AGENT LTD',
          addressEn: 'ADDRESS',
        },
      },
    }));
    const view = renderAirOrderContent({
      consigneeText: '手填收货人',
      remarks: '低频备注',
    });
    fireEvent.click(screen.getByRole('button', { name: /从委托客户带入/ }));
    await waitFor(() =>
      expect(
        screen.getByRole('textbox', { name: '发货人名称与地址' }),
      ).toHaveValue('CUSTOMER LTD\nADDRESS'),
    );
    fireEvent.click(screen.getByRole('button', { name: /从国外代理带入/ }));
    await waitFor(() =>
      expect(
        screen.getByRole('textbox', { name: '代理名称与地址' }),
      ).toHaveValue('AGENT LTD\nADDRESS'),
    );
    expect(view.imported).toHaveBeenCalledTimes(2);
    expect(view.readForm().isFieldTouched(['content', 'agentText'])).toBe(true);
    expect(view.readForm().getFieldValue(['content', 'consigneeText'])).toBe(
      '手填收货人',
    );
    expect(view.readForm().getFieldValue(['content', 'remarks'])).toBe(
      '低频备注',
    );
  });

  it('订单货描覆盖须确认，取消保留原文，确认只替换品名', async () => {
    const view = renderAirOrderContent({
      goodsDescription: '原单证品名',
      marks: '原唛头',
    });
    fireEvent.click(screen.getByRole('button', { name: /从订单货物信息带入/ }));
    const first = await screen.findByRole('dialog');
    fireEvent.click(within(first).getByRole('button', { name: '保留原内容' }));
    await waitFor(() => expect(first).not.toBeVisible());
    expect(view.imported).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /从订单货物信息带入/ }),
      ).not.toHaveClass('ant-btn-loading'),
    );
    fireEvent.click(screen.getByRole('button', { name: /从订单货物信息带入/ }));
    const second = await screen.findByRole('dialog');
    fireEvent.click(within(second).getByRole('button', { name: '确认覆盖' }));
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: '运单品名' })).toHaveValue(
        '订单原始品名',
      ),
    );
    expect(view.readForm().getFieldValue(['content', 'marks'])).toBe('原唛头');
  });
});
