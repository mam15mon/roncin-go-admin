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
export async function fillRegistration() {
  const dialog = within(screen.getByRole('dialog'));
  const add = dialog.getByText('继续添加发票').closest('button');
  if (!add) throw new Error('缺少新增发票按钮');
  await waitFor(() => expect(add).toBeVisible());
  fireEvent.click(add);
  fireEvent.mouseDown(dialog.getByLabelText('开票抬头'));
  fireEvent.click(await screen.findByText('测试抬头 · TAX'));
  fireEvent.change(dialog.getByLabelText('已有发票号码'), {
    target: { value: 'NO-1' },
  });
  fireEvent.change(dialog.getByLabelText('发票含税总额'), {
    target: { value: '30' },
  });
  fireEvent.click(dialog.getByText('添加账单分配'));
  const addFee = dialog.getByText('添加费用分配').closest('button');
  if (!addFee) throw new Error('缺少费用分配按钮');
  await waitFor(() => expect(addFee).toBeEnabled());
  fireEvent.click(addFee);
  fireEvent.mouseDown(dialog.getByLabelText('来源费用及可登记余额'));
  fireEvent.click(await screen.findByText('FEE 海运费 · 100.00 · 来源税率 -'));
  fireEvent.change(dialog.getByLabelText('本账单含税分配（CNY）'), {
    target: { value: '30' },
  });
  fireEvent.change(dialog.getByLabelText('本次含税金额'), {
    target: { value: '30' },
  });
  fireEvent.change(dialog.getByLabelText('本次登记税额（财务填写）'), {
    target: { value: '3.00000001' },
  });
  return dialog;
}
export function submitRegistration() {
  const button = screen.getByText('保存登记').closest('button');
  if (!button) throw new Error('缺少保存按钮');
  fireEvent.click(button);
}
