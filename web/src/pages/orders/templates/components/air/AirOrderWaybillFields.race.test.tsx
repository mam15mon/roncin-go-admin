import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import { renderAirOrderContent } from './AirOrderWaybillFields.fixtures';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
}));

describe('航空带入在途保护', () => {
  it('请求期间继续输入后迟到客户资料不覆盖新稿', async () => {
    let resolve!: (value: API.GetPartnerResponse) => void;
    vi.mocked(partnerServiceGetPartner).mockReturnValue(
      new Promise((done) => {
        resolve = done;
      }),
    );
    const view = renderAirOrderContent();
    fireEvent.click(screen.getByRole('button', { name: /从委托客户带入/ }));
    await waitFor(() => expect(partnerServiceGetPartner).toHaveBeenCalled());
    fireEvent.change(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
      { target: { value: '在途手填内容' } },
    );
    await act(async () =>
      resolve({
        success: true,
        data: {
          id: 'customer-1',
          organizationId: 'company-1',
          legalName: '迟到客户',
        },
      }),
    );
    expect(view.imported).not.toHaveBeenCalled();
    expect(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
    ).toHaveValue('在途手填内容');
  });

  it('客户A→B→A与覆盖确认期间编辑均不借回原目标覆盖', async () => {
    let resolve!: (value: API.GetPartnerResponse) => void;
    vi.mocked(partnerServiceGetPartner).mockReturnValue(
      new Promise((done) => {
        resolve = done;
      }),
    );
    const view = renderAirOrderContent({ goodsDescription: '原稿' });
    fireEvent.click(screen.getByRole('button', { name: /从委托客户带入/ }));
    await waitFor(() => expect(partnerServiceGetPartner).toHaveBeenCalled());
    await act(async () =>
      view.readForm().setFieldValue('customerId', 'customer-2'),
    );
    await act(async () =>
      view.readForm().setFieldValue('customerId', 'customer-1'),
    );
    await act(async () =>
      resolve({
        success: true,
        data: {
          id: 'customer-1',
          organizationId: 'company-1',
          legalName: '迟到客户',
        },
      }),
    );
    expect(view.imported).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /从订单货物信息带入/ }));
    const modal = await screen.findByRole('dialog');
    await act(async () =>
      view
        .readForm()
        .setFieldValue(['content', 'goodsDescription'], '确认期间的新稿'),
    );
    fireEvent.click(within(modal).getByRole('button', { name: '确认覆盖' }));
    await waitFor(() => expect(modal).not.toBeVisible());
    expect(view.imported).not.toHaveBeenCalled();
    expect(view.readForm().getFieldValue(['content', 'goodsDescription'])).toBe(
      '确认期间的新稿',
    );
  });
});
