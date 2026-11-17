import { DownloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useMutation } from '@tanstack/react-query';
import { App, Button, Space, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  FilterSettingsEntry,
  type FilterSettingsField,
  FinanceLedgerTemplate,
  omitHiddenFilterValues,
  type SearchFilterFieldItem,
  SearchFilterTemplate,
  useFilterSettings,
} from '@/components/ui';
import type { FinanceLedgerSummaryItem } from '@/components/ui/finance-ledger-template';
import { useListQueryState } from '@/components/ui/list-query-state';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import {
  financeReportServiceListFinanceReportPartyOptions,
  financeReportServiceListFinanceReportPersonnelOptions,
} from '@/services/roncin/financeReportService';
import { type CsvColumn, downloadCsv, serializeCsv } from '@/utils/csv';
import { formatDate } from '@/utils/format';
import {
  agingLabels,
  businessLabels,
  directionLabels,
  makeReportQuery,
  type ReportEnvelope,
  type ReportKind,
  type ReportQuery,
  type ReportSearchValues,
  reportFileName,
  reportPage,
} from './reportModel';

const { Text } = Typography;
type Accepted<T, S> = {
  payload: S;
  query: ReportQuery;
  key: string;
  rows: T[];
};
type Props<T extends FinanceLedgerSummaryItem, S extends ReportEnvelope<T>> = {
  kind: ReportKind;
  pageKey: string;
  title: string;
  subtitle: string;
  columns: ProColumns<T>[];
  exportColumns: CsvColumn<T>[];
  load: (query: ReportQuery) => Promise<S | undefined>;
  loadExport: (query: ReportQuery) => Promise<S | undefined>;
  validateRow: (row: T) => void;
  validateResponse: (response: S) => void;
  renderSummary: (summary: S) => React.ReactNode;
};

// 标签栏宽 80px 只容 5 个汉字，关键字统一叫「综合搜索」（与财务列表一致），检索范围写在占位里。
// 表格标题描述明细内容，不重复页面标题（利润报表与利润分析图共用 profit 明细）。
const TABLE_TITLES: Record<ReportKind, string> = {
  profit: '订单利润明细',
  cashflow: '收付款流水明细',
  aging: '账单账龄明细',
};

const KEYWORD_PLACEHOLDERS: Record<ReportKind, string> = {
  profit: '订单号或客户',
  cashflow: '流水号或参考号',
  aging: '账单号或结算单位',
};

function filterFields(kind: ReportKind): FilterSettingsField[] {
  if (kind === 'profit')
    return [
      { name: 'dateRange', label: '业务日期', defaultVisible: true },
      { name: 'keyword', label: '综合搜索', defaultVisible: true },
      { name: 'businessType', label: '业务类型', defaultVisible: true },
      { name: 'customerId', label: '订单客户', defaultVisible: true },
      { name: 'salesId', label: '销售', defaultVisible: true },
      { name: 'operatorId', label: '操作', defaultVisible: true },
      { name: 'customerServiceId', label: '客服', defaultVisible: false },
    ];
  const fields: FilterSettingsField[] = [
    { name: 'keyword', label: '综合搜索', defaultVisible: true },
    { name: 'direction', label: '收付方向', defaultVisible: true },
    { name: 'settlementPartyId', label: '结算单位', defaultVisible: true },
    { name: 'currency', label: '原币币种', defaultVisible: true },
  ];
  return kind === 'aging'
    ? [...fields, { name: 'bucket', label: '账龄分段', defaultVisible: true }]
    : [
        { name: 'dateRange', label: '交易日期', defaultVisible: true },
        ...fields,
        { name: 'paymentMethod', label: '支付方式', defaultVisible: false },
        { name: 'ourAccount', label: '本方账户', defaultVisible: false },
      ];
}

/** 四个报表入口共用官方列表采纳和导出生命周期，领域金额由各自适配器解释。 */
export default function ReportShell<
  T extends FinanceLedgerSummaryItem,
  S extends ReportEnvelope<T>,
>({
  kind,
  pageKey,
  title,
  subtitle,
  columns,
  exportColumns,
  load,
  loadExport,
  validateRow,
  validateResponse,
  renderSummary,
}: Props<T, S>) {
  const access = useAccess();
  const { initialState } = useInitialState();
  const user = initialState?.currentUser;
  const org = user?.currentOrganization;
  const enabled = Boolean(
    user?.id &&
      org?.id &&
      access.canReadFinanceReports &&
      access.canOperateOrganization(org.id),
  );
  const { message } = App.useApp();
  const exportGuard = useAsyncGuard();
  const fields = filterFields(kind);
  const settings = useFilterSettings({ pageKey, fields });
  const queryState = useListQueryState<ReportSearchValues>({
    tabKey: pageKey,
    listId: pageKey,
    defaultPageSize: 20,
    defaultFilters:
      kind !== 'aging' && settings.visibleNames.includes('dateRange')
        ? { dateRange: [dayjs().startOf('month'), dayjs().endOf('month')] }
        : {},
    sanitizeRestored: (value) =>
      omitHiddenFilterValues(value, fields, settings.visibleNames),
  });
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [scope, setScope] = useState(() =>
    JSON.stringify([
      org?.id,
      makeReportQuery(
        kind,
        queryState.snapshotRef.current.filters,
        org?.id ?? '',
      ),
    ]),
  );
  const [accepted, setAccepted] = useState<Accepted<T, S>>();
  const identity = JSON.stringify([org?.id, scope, enabled]);
  const identityRef = useRef(identity);
  identityRef.current = identity;
  const organizationRef = useRef(org?.id);
  organizationRef.current = org?.id;
  const current = accepted?.key === identity ? accepted : undefined;
  const commit = (
    values: ReportSearchValues,
    labels?: Record<string, string>,
  ) => {
    queryState.commit(values, labels);
    setScope(
      JSON.stringify([org?.id, makeReportQuery(kind, values, org?.id ?? '')]),
    );
  };
  const candidates = async (
    keyword: string | undefined,
    personnel: boolean,
    signal: AbortSignal,
  ) => {
    const organizationId = org?.id;
    if (!enabled || !organizationId) return [];
    const fn = personnel
      ? financeReportServiceListFinanceReportPersonnelOptions
      : financeReportServiceListFinanceReportPartyOptions;
    const response = await fn(
      {
        organizationId,
        keyword: keyword?.trim() || undefined,
        page: 1,
        pageSize: 50,
      },
      { signal, skipErrorHandler: true },
    );
    if (organizationRef.current !== organizationId) return [];
    if (response?.success !== true) throw new Error('报表候选加载失败');
    return (response.data ?? []).map((item) => {
      if (!item.id || !item.name) throw new Error('报表候选缺少标识或名称');
      return { value: item.id, label: item.name };
    });
  };
  const items: SearchFilterFieldItem[] = fields.map((field) => {
    const base: SearchFilterFieldItem = {
      name: field.name,
      label: field.label,
    };
    if (field.name === 'dateRange')
      return {
        ...base,
        type: 'date-range',
        fieldProps: { allowEmpty: [true, true] },
      };
    if (field.name === 'businessType')
      return {
        ...base,
        type: 'select',
        options: Object.entries(businessLabels).map(([value, label]) => ({
          value,
          label,
        })),
      };
    if (field.name === 'direction')
      return {
        ...base,
        type: 'select',
        options: Object.entries(directionLabels).map(([value, label]) => ({
          value,
          label,
        })),
      };
    if (field.name === 'bucket')
      return {
        ...base,
        type: 'select',
        options: Object.entries(agingLabels).map(([value, label]) => ({
          value,
          label,
        })),
      };
    if (
      [
        'salesId',
        'operatorId',
        'customerServiceId',
        'customerId',
        'settlementPartyId',
      ].includes(field.name)
    ) {
      return {
        ...base,
        type: 'searchable-select',
        placeholder: '名称 / 拼音搜索',
        requestKey: [enabled, kind],
        request: ({ keyWords, signal }) =>
          candidates(
            keyWords,
            ['salesId', 'operatorId', 'customerServiceId'].includes(field.name),
            signal,
          ),
      };
    }
    return {
      ...base,
      placeholder:
        field.name === 'currency'
          ? '例如 CNY、USD'
          : field.name === 'keyword'
            ? KEYWORD_PLACEHOLDERS[kind]
            : undefined,
    };
  });
  const exporting = useMutation({
    mutationFn: async () => {
      if (!enabled || !current) return;
      const requestIdentity = identity;
      await exportGuard.run(
        () => loadExport(current.query),
        (response) => {
          if (requestIdentity !== identityRef.current) return;
          const page = reportPage(response, validateRow);
          if (response) validateResponse(response);
          if (!page.data.length) {
            message.info('当前筛选无可导出数据');
            return;
          }
          const content = serializeCsv(page.data, exportColumns);
          if (content) {
            downloadCsv(content, reportFileName(title, response?.generatedAt));
            message.success(`已导出 ${page.data.length} 条完整筛选记录`);
          }
        },
      );
    },
    meta: { errorMessage: '报表导出失败' },
  });
  return (
    <FinanceLedgerTemplate<T, Record<string, never>, Accepted<T, S>>
      pageTitle={title}
      pageSubTitle={`${org?.name ?? ''} · ${subtitle}`}
      headerTitle={TABLE_TITLES[kind]}
      columns={columns}
      exportColumns={exportColumns}
      actionRef={actionRef}
      search={false}
      rowSelection={false}
      showSummaryBoard={false}
      columnSettingsKey={pageKey}
      selectionScope={identity}
      pagination={queryState.pagination}
      headerExtra={
        <Space>
          <Text type="secondary">
            {current
              ? `生成于 ${formatDate(current.payload.generatedAt, 'minute')}`
              : '等待报表查询'}
          </Text>
        </Space>
      }
      customSearch={
        <>
          <SearchFilterTemplate<ReportSearchValues>
            items={items}
            initialValues={queryState.initial.filters}
            initialLabels={queryState.initial.labels}
            visibleNames={settings.visibleNames}
            extraRight={
              <FilterSettingsEntry
                fields={fields}
                visibleNames={settings.visibleNames}
                onChange={settings.setVisible}
                onReset={settings.reset}
              />
            }
            onCommit={commit}
            onSearch={(value) => {
              commit(value);
              actionRef.current?.reload(true);
            }}
            onReset={() => {
              commit({}, {});
              actionRef.current?.reload(true);
            }}
          />
          {current ? renderSummary(current.payload) : null}
        </>
      }
      extraToolBarActions={[
        <ReportExportButton
          key="full-export"
          disabled={!enabled || !current}
          loading={exporting.isPending}
          onClick={() => exporting.mutate()}
        />,
      ]}
      request={async (p) => {
        if (!enabled || !org?.id) return { data: [], total: 0, success: false };
        const query = makeReportQuery(
          kind,
          queryState.snapshotRef.current.filters,
          org.id,
        );
        const response = await load({
          ...query,
          page: p.current ?? 1,
          pageSize: p.pageSize ?? 20,
        });
        const page = reportPage(response, validateRow);
        if (response) validateResponse(response);
        if (!response) throw new Error('报表查询失败');
        return {
          ...page,
          summary: { payload: response, query, key: identity, rows: page.data },
        };
      }}
      onAcceptedSnapshot={(snapshot) => {
        if (snapshot.summary?.key === identityRef.current)
          setAccepted(snapshot.summary);
      }}
    />
  );
}

function ReportExportButton(props: {
  disabled: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  // 与模板的“导出当前页”并列，完整筛选导出走服务端独立接口。
  return (
    <Button icon={<DownloadOutlined />} {...props}>
      导出全部筛选结果
    </Button>
  );
}
