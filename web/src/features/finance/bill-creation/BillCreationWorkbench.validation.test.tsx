import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BillCreationWorkbench,
  mocks,
  resetWorkbenchMocks,
  sleepInAct,
} from './BillCreationWorkbench.fixtures';

/** Modal 进场动画在 jsdom 中停在 opacity:0，toBeVisible 不可用；这里只判定 display:none 的叶子隐藏。 */
function expectShown(element: HTMLElement, shown = true) {
  const hidden = Boolean(element.closest('[style*="display: none"]'));
  expect(hidden).toBe(!shown);
}

describe('批量建账配置阻断与定位', () => {
  beforeEach(resetWorkbenchMocks);
  it('任一叶子缺少结算账户时阻断整批创建并自动跳到该叶子', async () => {
    const groups: API.BillBatchPreviewGroup[] = [
      {
        groupKey: 'g-ok',
        billDate: '2026-11-17',
        settlementPartyId: 'partner-ok',
        settlementPartyName: '配置完整单位',
        direction: 'RECEIVABLE',
        currency: 'CNY',
        baseCurrency: 'CNY',
        totalAmount: '100.00',
        baseCurrencyAmount: '100.00',
        estimatedInvoiceCurrency: 'CNY',
        estimatedInvoiceRate: '1',
        estimatedInvoiceAmount: '100.00',
        configurationComplete: true,
        fees: [],
      },
      {
        groupKey: 'g-missing',
        billDate: '2026-11-17',
        settlementPartyId: 'partner-missing',
        settlementPartyName: '缺账户单位',
        direction: 'RECEIVABLE',
        currency: 'EUR',
        baseCurrency: 'CNY',
        totalAmount: '800.00',
        baseCurrencyAmount: '800.00',
        estimatedInvoiceCurrency: 'EUR',
        estimatedInvoiceRate: '1',
        estimatedInvoiceAmount: '800.00',
        configurationComplete: true,
        fees: [],
      },
    ];
    mocks.preview.mockResolvedValue({
      data: groups,
    });
    mocks.accounts.mockImplementation(({ settlementPartyId }) =>
      settlementPartyId === 'partner-missing'
        ? Promise.resolve({ data: [] })
        : Promise.resolve({
            data: [
              {
                id: 'account-default',
                name: '默认结算账户',
                bankName: '测试银行',
                accountNo: '6222',
                currency: 'CNY',
                isDefault: true,
              },
            ],
          }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('配置完整单位')).toBeInTheDocument();
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith(
        expect.objectContaining({ settlementPartyId: 'partner-missing' }),
        expect.anything(),
      ),
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await sleepInAct(700);

    fireEvent.click(screen.getByRole('button', { name: /生成 2 张账单/ }));
    await waitFor(() =>
      expect(
        screen.getByText('请先补齐首个标记叶子的对账抬头和结算账户'),
      ).toBeInTheDocument(),
    );
    // 整批被阻断时自动跳到缺配置叶子，其他叶子不再置前展示。
    await waitFor(() => expectShown(screen.getByDisplayValue('缺账户单位')));
    expectShown(screen.getByDisplayValue('配置完整单位'), false);
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
