import type { ProColumns } from '@ant-design/pro-components';
import { Table, Typography } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import {
  financeReportServiceExportCashflowReport,
  financeReportServiceListCashflowReport,
} from '@/services/roncin/financeReportService';
import { formatAmount } from '@/utils/format';
import ReportShell from './ReportShell';
import {
  cashflowExportColumns,
  validateCashflowResponse,
  validateCashflowRow,
} from './reportModel';

const columns: ProColumns<API.CashflowReportRow>[] = [
  { title: '流水号', dataIndex: 'flowNo', width: 170, fixed: 'left' },
  { title: '交易日期', dataIndex: 'transactionDate', width: 120 },
  {
    title: '方向',
    dataIndex: 'direction',
    width: 80,
    renderText: (value) => (value === 'RECEIVABLE' ? '收款' : '付款'),
  },
  { title: '结算单位', dataIndex: 'settlementPartyName', width: 200 },
  { title: '原币', dataIndex: 'currency', width: 70 },
  {
    title: '实际金额',
    dataIndex: 'amount',
    width: 150,
    align: 'right',
    renderText: (value) => formatAmount(value),
  },
  { title: '本币', dataIndex: 'baseCurrency', width: 70 },
  {
    title: '折本币金额',
    dataIndex: 'baseAmount',
    width: 150,
    align: 'right',
    renderText: (value) => formatAmount(value),
  },
  { title: '本方账户', dataIndex: 'ourAccount', width: 180 },
  { title: '支付方式', dataIndex: 'paymentMethod', width: 110 },
  { title: '银行参考号', dataIndex: 'bankReferenceNo', width: 160 },
];
export function CashReportTotals({
  items,
  label,
}: {
  items: API.FinanceReportCashAmount[];
  label: string;
}) {
  return (
    <SectionCard title={label}>
      <Table<API.FinanceReportCashAmount>
        size="small"
        rowKey="currency"
        dataSource={items}
        pagination={false}
        columns={[
          { title: '币种', dataIndex: 'currency' },
          {
            title: '实际收款',
            dataIndex: 'receivableAmount',
            align: 'right',
            render: (value) => formatAmount(value),
          },
          {
            title: '实际付款',
            dataIndex: 'payableAmount',
            align: 'right',
            render: (value) => formatAmount(value),
          },
          {
            title: '收支净额',
            dataIndex: 'netAmount',
            align: 'right',
            render: (value) => formatAmount(value),
          },
        ]}
      />
    </SectionCard>
  );
}
export default function CashflowReportPage() {
  return (
    <ReportShell<API.CashflowReportRow, API.ListCashflowReportResponse>
      kind="cashflow"
      pageKey="/finance/reports/cashflows"
      title="实际收付款报表"
      subtitle="按已确认流水交易日期 · 含未核销收付 · 取消流水不计"
      columns={columns}
      exportColumns={cashflowExportColumns}
      validateRow={validateCashflowRow}
      validateResponse={validateCashflowResponse}
      load={(query) =>
        financeReportServiceListCashflowReport(query, {
          skipErrorHandler: true,
        })
      }
      loadExport={(query) =>
        financeReportServiceExportCashflowReport(query, {
          skipErrorHandler: true,
        })
      }
      renderSummary={(response) => (
        <div style={{ display: 'grid', gap: 12, marginBottom: 12 }}>
          <CashReportTotals
            label="完整筛选原币合计"
            items={response.nativeTotals ?? []}
          />
          <CashReportTotals
            label="完整筛选折本币合计"
            items={response.baseTotals ?? []}
          />
          <Typography.Text type="secondary">
            收付款反核销仅改变分配，不减少真实收付；对冲不计现金。
          </Typography.Text>
        </div>
      )}
    />
  );
}
