import React, { type MutableRefObject, type ReactNode } from 'react';
import { vi } from 'vitest';
import ReportPage from './profit';
import type { ReportQuery } from './reportModel';

type Summary = {
  payload: API.ListProfitReportResponse;
  query: ReportQuery;
  key: string;
  rows: API.ProfitReportRow[];
};
type Page = {
  data: API.ProfitReportRow[];
  total: number;
  success?: boolean;
  summary?: Summary;
};
export type CapturedReport = {
  request: (p: { current?: number; pageSize?: number }) => Promise<Page>;
  onAcceptedSnapshot: (p: {
    rows: API.ProfitReportRow[];
    total: number;
    summary?: Summary;
  }) => void;
  actionRef: MutableRefObject<{ reload: () => Promise<void> } | undefined>;
  customSearch?: ReactNode;
  extraToolBarActions?: ReactNode[];
  headerExtra?: ReactNode;
};
const reportMocks = vi.hoisted(() => ({
  template: undefined as CapturedReport | undefined,
  list: vi.fn(),
  export: vi.fn(),
  parties: vi.fn(),
  personnel: vi.fn(),
  reload: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: (props: CapturedReport) => {
    reportMocks.template = props;
    props.actionRef.current = { reload: reportMocks.reload };
    return (
      <div>
        {props.headerExtra}
        {props.customSearch}
        {props.extraToolBarActions}
      </div>
    );
  },
}));
vi.mock('@/services/roncin/financeReportService', () => ({
  financeReportServiceListProfitReport: reportMocks.list,
  financeReportServiceExportProfitReport: reportMocks.export,
  financeReportServiceListFinanceReportPartyOptions: reportMocks.parties,
  financeReportServiceListFinanceReportPersonnelOptions: reportMocks.personnel,
}));
export const profitAmount: API.ProfitReportAmount = {
  baseCurrency: 'CNY',
  receivableAmount: '10600',
  payableAmount: '6360',
  profitAmount: '4240',
  netReceivableAmount: '10000',
  netPayableAmount: '6000',
  netProfitAmount: '4000',
};
export const profitRow: API.ProfitReportRow = {
  ...profitAmount,
  id: 'row-1',
  orderId: 'order-1',
  orderNo: 'ORDER-1',
  businessDate: '2026-11-14',
};
export const profitResponse: API.ListProfitReportResponse = {
  success: true,
  generatedAt: '2026-11-14T00:00:00Z',
  total: '2',
  data: [profitRow],
  totals: [profitAmount],
  nativeTotals: [
    {
      direction: 'RECEIVABLE',
      currency: 'CNY',
      amount: '10600',
      netAmount: '10000',
    },
  ],
  months: [{ month: '2026-11', amounts: [profitAmount] }],
  orderCount: '2',
  feeCount: '4',
};
export function resetReportMocks() {
  reportMocks.template = undefined;
  reportMocks.list.mockReset().mockResolvedValue(profitResponse);
  reportMocks.export.mockReset().mockResolvedValue(profitResponse);
  reportMocks.parties
    .mockReset()
    .mockResolvedValue({ success: true, data: [] });
  reportMocks.personnel
    .mockReset()
    .mockResolvedValue({ success: true, data: [] });
  reportMocks.reload.mockClear();
}

export const ProfitReportPage = ReportPage;
export { reportMocks };
