import type { ProColumns } from '@ant-design/pro-components';
import { Alert, Space, Table, Typography } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import {
  financeReportServiceExportProfitReport,
  financeReportServiceListProfitReport,
} from '@/services/roncin/financeReportService';
import { formatAmount } from '@/utils/format';
import ProfitChart from './ProfitChart';
import ReportShell from './ReportShell';
import {
  businessLabels,
  directionLabels,
  profitExportColumns,
  validateProfitResponse,
  validateProfitRow,
} from './reportModel';

const amounts = [
  ['receivableAmount', '含税应收'],
  ['netReceivableAmount', '未税应收'],
  ['payableAmount', '含税应付'],
  ['netPayableAmount', '未税应付'],
  ['profitAmount', '含税利润'],
  ['netProfitAmount', '未税利润'],
] as const;
const columns: ProColumns<API.ProfitReportRow>[] = [
  { title: '订单号', dataIndex: 'orderNo', width: 160, fixed: 'left' },
  {
    title: '业务日期',
    dataIndex: 'businessDate',
    width: 120,
    renderText: (value) => value || '未设置',
  },
  {
    title: '业务类型',
    dataIndex: 'businessType',
    width: 100,
    renderText: (value) => businessLabels[value] ?? value,
  },
  { title: '客户', dataIndex: 'customerName', width: 180 },
  { title: '销售', dataIndex: 'salesName', width: 100 },
  { title: '操作', dataIndex: 'operatorName', width: 100 },
  { title: '客服', dataIndex: 'customerServiceName', width: 100 },
  { title: '本币', dataIndex: 'baseCurrency', width: 70 },
  ...amounts.map(([dataIndex, title]) => ({
    dataIndex,
    title,
    width: 140,
    align: 'right' as const,
    renderText: (value: string | undefined) => formatAmount(value),
  })),
  {
    title: '费用数',
    dataIndex: 'feeCount',
    width: 80,
    renderText: (value) => Number(value ?? 0),
  },
];
function ProfitSummary({
  response,
  analysis,
}: {
  response: API.ListProfitReportResponse;
  analysis: boolean;
}) {
  return (
    <Space
      direction="vertical"
      size={12}
      style={{ display: 'flex', marginBottom: 12 }}
    >
      {Number(response.undatedOrderCount ?? 0) > 0 ? (
        <Alert
          type="warning"
          showIcon
          title={`有 ${Number(response.undatedOrderCount)} 张订单未设置业务日期`}
          description="未设置日期的订单不能归入指定日期范围；清除业务日期筛选可查看未归期费用。"
        />
      ) : null}
      <SectionCard
        title="完整筛选合计"
        extra={
          <Typography.Text type="secondary">
            {Number(response.orderCount ?? 0)} 张订单 ·{' '}
            {Number(response.feeCount ?? 0)} 笔有效费用
          </Typography.Text>
        }
      >
        <Table<API.ProfitReportAmount>
          size="small"
          rowKey="baseCurrency"
          pagination={false}
          scroll={{ x: 'max-content' }}
          dataSource={response.totals ?? []}
          columns={[
            { title: '本币', dataIndex: 'baseCurrency' },
            ...amounts.map(([dataIndex, title]) => ({
              dataIndex,
              title,
              align: 'right' as const,
              render: (value: string | undefined) => formatAmount(value),
            })),
          ]}
        />
        {response.nativeTotals?.length ? (
          <Typography.Paragraph
            type="secondary"
            style={{ marginTop: 8, marginBottom: 0 }}
          >
            原币：
            {response.nativeTotals
              .map(
                (item) =>
                  `${directionLabels[item.direction ?? ''] ?? ''} ${item.currency} 含税 ${formatAmount(item.amount)}／未税 ${formatAmount(item.netAmount)}`,
              )
              .join('；')}
          </Typography.Paragraph>
        ) : null}
      </SectionCard>
      {analysis ? <ProfitChart response={response} /> : null}
    </Space>
  );
}
export function ProfitReportPage({ analysis = false }: { analysis?: boolean }) {
  return (
    <ReportShell<API.ProfitReportRow, API.ListProfitReportResponse>
      kind="profit"
      pageKey={
        analysis
          ? '/finance/reports/profit-analysis'
          : '/finance/reports/profit'
      }
      title={analysis ? '利润分析图' : '利润报表'}
      subtitle="出口按 ETD、进口按 ETA · 当前有效费用 · 含未建账费用"
      columns={columns}
      exportColumns={profitExportColumns}
      validateRow={validateProfitRow}
      validateResponse={validateProfitResponse}
      load={(query) =>
        financeReportServiceListProfitReport(query, { skipErrorHandler: true })
      }
      loadExport={(query) =>
        financeReportServiceExportProfitReport(query, {
          skipErrorHandler: true,
        })
      }
      renderSummary={(response) => (
        <ProfitSummary response={response} analysis={analysis} />
      )}
    />
  );
}
export default ProfitReportPage;
