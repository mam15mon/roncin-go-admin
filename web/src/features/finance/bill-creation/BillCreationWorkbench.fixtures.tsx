import { act, fireEvent, screen } from '@testing-library/react';
import React, { useEffect, useState } from 'react';
import { vi } from 'vitest';
import Workbench from './BillCreationWorkbench';

const mocks = vi.hoisted(() => ({
  organizations: vi.fn(),
  organizationId: 'A' as string | undefined,
  canOperate: true,
  candidates: vi.fn(),
  preview: vi.fn(),
  create: vi.fn(),
  accounts: vi.fn(),
}));
vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: {
    request?: (params: {
      current: number;
      pageSize: number;
    }) => Promise<{ data?: API.FeeLedgerItem[] }>;
    rowSelection?: {
      onChange?: (keys: React.Key[], rows: API.FeeLedgerItem[]) => void;
    };
  }) => {
    const [data, setData] = useState<API.FeeLedgerItem[]>([]);
    useEffect(() => {
      if (props.request)
        void props
          .request({ current: 1, pageSize: 20 })
          .then((r) => setData(r.data || []));
    }, []);
    return (
      <div>
        {data.map((row) => (
          <button
            key={row.id}
            type="button"
            onClick={() =>
              props.rowSelection?.onChange?.([row.id ?? ''], [row])
            }
          >
            {row.feeName}
          </button>
        ))}
      </div>
    );
  },
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    hasAction: () => true,
    canCreateFinanceBills: mocks.canOperate,
    canOperateOrganization: (id?: string) =>
      mocks.canOperate && Boolean(id) && id === mocks.organizationId,
  }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: mocks.organizationId, name: '当前公司' },
      },
    },
  }),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceOrganizationOptions: mocks.organizations,
  settlementServiceListBillCreationCandidates: mocks.candidates,
  settlementServicePreviewBillBatch: mocks.preview,
  settlementServiceCreateBillBatch: mocks.create,
  settlementServiceConfirmBillBatch: vi.fn(),
  settlementServiceListBillSettlementAccountCandidates: mocks.accounts,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([
    { label: 'CNY - 人民币', value: 'CNY' },
    { label: 'USD - 美元', value: 'USD' },
  ]),
}));

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/** 防抖窗口断言仍用真实定时器；包进 act 让窗口期落地的预览/账户回调在 act 内更新状态。 */
export const sleepInAct = async (ms: number) => {
  await act(async () => {
    await sleep(ms);
  });
};

export function resetWorkbenchMocks() {
  mocks.organizationId = 'A';
  mocks.canOperate = true;
  mocks.organizations.mockReset();
  mocks.candidates.mockReset();
  mocks.preview.mockReset();
  mocks.create.mockReset();
  mocks.accounts.mockReset();
  mocks.create.mockResolvedValue({ data: { id: 'batch-1', bills: [] } });
  mocks.organizations.mockResolvedValue({
    data: [
      { id: 'A', name: '公司A' },
      { id: 'B', name: '公司B' },
    ],
  });
  mocks.candidates.mockImplementation(({ organizationId }) =>
    Promise.resolve({
      data: [
        {
          id: `fee-${organizationId}`,
          feeName: `费用${organizationId}`,
          status: 2,
          organizationId,
          canBill: true,
        },
      ],
    }),
  );
  mocks.accounts.mockResolvedValue({
    data: [
      {
        id: 'account-default',
        name: '默认结算账户',
        bankName: '测试银行',
        accountNo: '6222',
        currency: 'CNY',
        isDefault: true,
      },
      {
        id: 'account-second',
        name: '第二账户',
        bankName: '测试银行',
        accountNo: '6223',
        currency: 'CNY',
        isDefault: false,
      },
    ],
  });
}

export const singlePreviewGroup = {
  groupKey: 'g1',
  settlementPartyId: 'partner-a',
  settlementPartyName: '结算单位甲',
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
};

export function pickBillDate(isoDate: string) {
  const input = screen.getByLabelText('账单日期');
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: isoDate } });
  fireEvent.keyDown(input, { key: 'Enter', code: 'Enter', keyCode: 13 });
}

export const BillCreationWorkbench = Workbench;
export { mocks };
