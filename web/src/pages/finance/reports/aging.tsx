import type { ProColumns } from '@ant-design/pro-components';
import { Alert, Table } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import {
  financeReportServiceExportAgingReport,
  financeReportServiceListAgingReport,
} from '@/services/roncin/financeReportService';
import { formatAmount } from '@/utils/format';
import ReportShell from './ReportShell';
import {
  agingExportColumns,
  agingLabels,
  directionLabels,
  validateAgingResponse,
  validateAgingRow,
} from './reportModel';

const columns: ProColumns<API.AgingReportRow>[] = [
  { title: '账单号', dataIndex: 'billNo', width: 170, fixed: 'left' },
  {
    title: '方向',
    dataIndex: 'direction',
    width: 80,
    renderText: (value) => directionLabels[value] ?? value,
  },
  { title: '结算单位', dataIndex: 'settlementPartyName', width: 190 },
  { title: '创建日期', dataIndex: 'billDate', width: 120 },
  {
    title: '到期日',
    dataIndex: 'dueDate',
    width: 130,
    renderText: (value) => value || '未设置到期日',
  },
  {
    title: '账龄',
    dataIndex: 'bucket',
    width: 130,
    renderText: (value) => agingLabels[value] ?? value,
  },
  { title: '原币', dataIndex: 'currency', width: 70 },
  ...(
    [
      ['totalAmount', '账单金额'],
      ['settledAmount', '已清偿'],
      ['balance', '当前余额'],
    ] as const
  ).map(([dataIndex, title]) => ({
    dataIndex,
    title,
    align: 'right' as const,
    width: 140,
    renderText: (value: string | undefined) => formatAmount(value),
  })),
  { title: '本币', dataIndex: 'baseCurrency', width: 70 },
  {
    title: '本币余额',
    dataIndex: 'baseBalance',
    width: 150,
    align: 'right',
    renderText: (value) => formatAmount(value),
  },
];
const buckets = [
  ['notDue', '未到期'],
  ['days130', '1–30 天'],
  ['days3160', '31–60 天'],
  ['days6190', '61–90 天'],
  ['over90', '90 天以上'],
  ['undated', '未设置到期日'],
  ['total', '合计'],
] as const;
function AgingTotals({
  title,
  items,
}: {
  title: string;
  items: API.AgingReportBucketAmount[];
}) {
  return (
    <SectionCard title={title}>
      <Table<API.AgingReportBucketAmount>
        size="small"
        rowKey={(row) => `${row.direction}:${row.currency}`}
        dataSource={items}
        pagination={false}
        scroll={{ x: 'max-content' }}
        columns={[
          {
            title: '方向',
            dataIndex: 'direction',
            render: (value) => directionLabels[value] ?? value,
          },
          { title: '币种', dataIndex: 'currency' },
          ...buckets.map(([dataIndex, title]) => ({
            title,
            dataIndex,
            align: 'right' as const,
            render: (value: string | undefined) => formatAmount(value),
          })),
        ]}
      />
    </SectionCard>
  );
}
export default function AgingReportPage() {
  return (
    <ReportShell<API.AgingReportRow, API.ListAgingReportResponse>
      kind="aging"
      pageKey="/finance/reports/aging"
      title="账龄表"
      subtitle="按账单到期日 · 当前余额扣除有效核销与对冲"
      columns={columns}
      exportColumns={agingExportColumns}
      validateRow={validateAgingRow}
      validateResponse={validateAgingResponse}
      load={(query) =>
        financeReportServiceListAgingReport(query, { skipErrorHandler: true })
      }
      loadExport={(query) =>
        financeReportServiceExportAgingReport(query, { skipErrorHandler: true })
      }
      renderSummary={(response) => (
        <div style={{ display: 'grid', gap: 12, marginBottom: 12 }}>
          <Alert
            type="info"
            showIcon
            title={`当前业务日：${response.asOfDate ?? ''}`}
            description="本表展示当前余额；到期当天属于未到期。未设置到期日的账单独立展示，不按其他日期推算账龄。"
          />
          <AgingTotals
            title="完整筛选原币账龄合计"
            items={response.nativeTotals ?? []}
          />
          <AgingTotals
            title="完整筛选折本币账龄合计"
            items={response.baseTotals ?? []}
          />
        </div>
      )}
    />
  );
}
