import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import type { ReactNode } from 'react';
import { expect } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';

export const invoiceUser: API.CurrentUser = {
  id: 'invoice-user',
  currentOrganization: {
    id: 'company',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'system.finance.invoice.create', dataScope: 'organization' },
    { key: 'system.finance.invoice.read', dataScope: 'organization' },
    { key: 'system.finance.invoice.update', dataScope: 'organization' },
  ],
};
export const invoiceBill: API.FinanceBill = {
  id: 'bill',
  version: '2',
  organizationId: 'company',
  billNo: 'BI-001',
  direction: 'RECEIVABLE',
  settlementPartyId: 'party',
  currency: 'CNY',
  baseCurrency: 'CNY',
  invoiceableAmount: '100',
  lines: [
    {
      id: 'line',
      orderFeeId: 'fee',
      active: true,
      feeCode: 'FEE',
      feeName: '海运费',
      currency: 'CNY',
      totalAmount: '100',
      invoiceableAmount: '100',
    },
  ],
};
export function NoMotion({ children }: { children: ReactNode }) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      {children}
    </ConfigProvider>
  );
}
/** 一账单一费用的简单登记：弹窗自动生成发票并选好账单和费用，只填头部四项与税额。 */
export async function fillRegistration() {
  const dialog = within(screen.getByRole('dialog'));
  const profile = await dialog.findByLabelText('开票抬头');
  fireEvent.mouseDown(profile);
  fireEvent.click(await screen.findByText('测试抬头 · TAX'));
  fireEvent.change(dialog.getByLabelText('已有发票号码'), {
    target: { value: 'NO-1' },
  });
  fireEvent.change(dialog.getByLabelText('发票含税总额'), {
    target: { value: '30' },
  });
  await waitFor(() =>
    expect(dialog.getByLabelText('本次含税金额')).toHaveValue('30'),
  );
  expect(dialog.getByLabelText('本账单含税分配（CNY）')).toHaveValue('30');
  expect(
    dialog.getByText('来源税率未设置，请按发票填写税额'),
  ).toBeInTheDocument();
  fireEvent.change(dialog.getByLabelText('本次登记税额'), {
    target: { value: '3.00000001' },
  });
  return dialog;
}
export function submitRegistration() {
  const button = screen.getByText('保存登记').closest('button');
  if (!button) throw new Error('缺少保存按钮');
  fireEvent.click(button);
}
