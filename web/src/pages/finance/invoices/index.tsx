import {
  CloseCircleOutlined,
  EyeOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { App, Form, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
  type SearchFilterFieldItem,
  SearchFilterTemplate,
} from '@/components/ui';
import {
  FilterSettingsEntry,
  type FilterSettingsField,
  omitHiddenFilterValues,
  useFilterSettings,
} from '@/components/ui/filter-settings';
import { useListQueryState } from '@/components/ui/list-query-state';
import { FinanceInvoiceStatus } from '@/enums.generated';
import { InvoiceRegistrationModal } from '@/features/finance/invoice-registration';
import {
  invoiceLedgerSummaryConfig,
  mapMatchedSummary,
  useLedgerSelectionScope,
} from '@/features/finance/ledger-summary';
import {
  settlementServiceCancelInvoice,
  settlementServiceGetInvoice,
  settlementServiceListInvoices,
  settlementServiceRedFlushInvoice,
} from '@/services/roncin/settlementService';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
  unwrapPage,
} from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount, trimDecimal } from '@/utils/format';
import InvoiceCancelModal from './components/InvoiceCancelModal';
import InvoiceDetailDrawer from './components/InvoiceDetailDrawer';
import { InvoiceRedFlushModal } from './components/InvoiceIssueAndRedFlushModals';
import {
  invoiceCancelSuccessText,
  invoiceStates,
  invoiceStateText,
  invoiceVoidSuccessText,
} from './components/invoiceConstants';

import { invoiceExportColumns } from './ledgerExport';

/** 已提交的发票筛选条件，表格、汇总与导出共用官方采纳快照。 */
type InvoiceLedgerFilterParams = {
  keyword?: string;
  direction?: string;
  status?: string;
};

const invoiceFilterFields: FilterSettingsField[] = [
  { name: 'keyword', label: '综合搜索', defaultVisible: true },
  { name: 'direction', label: '发票方向', defaultVisible: true },
  { name: 'status', label: '发票状态', defaultVisible: true },
];

const invoiceFilterItems: SearchFilterFieldItem[] = [
  {
    name: 'keyword',
    label: '综合搜索',
    placeholder: '记录编号、发票号或结算单位',
  },
  {
    name: 'direction',
    label: '发票方向',
    type: 'select',
    options: [
      { label: '销项', value: 'RECEIVABLE' },
      { label: '进项', value: 'PAYABLE' },
    ],
  },
  {
    name: 'status',
    label: '发票状态',
    type: 'select',
    options: Object.entries(invoiceStates).map(([value, meta]) => ({
      value,
      label:
        Number(value) === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED
          ? '已开具 / 已收票'
          : meta.text,
    })),
  },
];

type RedFlushValues = {
  redInvoiceNo: string;
  redInvoiceDate: Dayjs;
  reason: string;
};
type CancelValues = { reason: string };

export default function FinanceInvoicesPage() {
  const access = useAccess();
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  const organizationId = currentUser?.currentOrganization?.id;
  const hasCompanyScope = Boolean(
    currentUser?.id &&
      organizationId &&
      access.canOperateOrganization(organizationId),
  );
  const filterSettings = useFilterSettings({
    pageKey: 'finance:invoices',
    fields: invoiceFilterFields,
  });
  const queryState = useListQueryState<InvoiceLedgerFilterParams>({
    tabKey: '/finance/invoices',
    listId: 'finance:invoices',
    defaultPageSize: 40,
    defaultFilters: {},
    sanitizeRestored: (filters) =>
      omitHiddenFilterValues(
        filters,
        invoiceFilterFields,
        filterSettings.visibleNames,
      ),
  });
  const { selectionScope, commitFilters } = useLedgerSelectionScope(
    queryState.commit,
    () => queryState.snapshotRef.current.filters,
  );
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [redFlushForm] = Form.useForm<RedFlushValues>();
  const [cancelForm] = Form.useForm<CancelValues>();
  const [createOpen, setCreateOpen] = useState(false);
  const [reviseTarget, setReviseTarget] = useState<API.FinanceInvoice>();
  const [redFlushTarget, setRedFlushTarget] = useState<API.FinanceInvoice>();
  const [cancelTarget, setCancelTarget] = useState<API.FinanceInvoice>();
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<API.FinanceInvoice>();
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    issuedCount: 0,
    amountsByBaseCurrency: [] as API.FinanceBaseCurrencyAmount[],
  });
  const formatBaseCurrencyAmounts = (
    field: 'receivableBaseAmount' | 'payableBaseAmount',
  ) =>
    metricStats.amountsByBaseCurrency
      .map(
        (item) =>
          `${formatAmount(item[field] ?? '0')} ${item.baseCurrency ?? '-'}`,
      )
      .join(' / ') || '-';
  const reload = () => actionRef.current?.reload();

  const showDetail = async (row: API.FinanceInvoice) => {
    if (!row.id) return;
    try {
      setDetail((await settlementServiceGetInvoice({ id: row.id })).data);
    } catch (error) {
      message.error(getErrorMessage(error, '加载开票详情失败'));
    }
  };

  const cancelInvoice = async (): Promise<false | undefined> => {
    if (!cancelTarget?.id || !cancelTarget.version) return;
    const issued =
      cancelTarget.status ===
      FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED;
    const values = await cancelForm.validateFields();
    setSubmitting(true);
    try {
      const writeResponse3 = await settlementServiceCancelInvoice(
        { id: cancelTarget.id },
        {
          id: cancelTarget.id,
          expectedVersion: cancelTarget.version,
          reason: values.reason,
        },
      );
      if (!isWriteResponse(writeResponse3, true)) return false;
      message.success(
        issued
          ? invoiceVoidSuccessText(cancelTarget.direction)
          : invoiceCancelSuccessText(cancelTarget.direction),
      );
      setCancelTarget(undefined);
      reload();
    } catch (error) {
      message.error(
        getErrorMessage(error, issued ? '作废发票失败' : '取消开票记录失败'),
      );
    } finally {
      setSubmitting(false);
    }

    return undefined;
  };

  const redFlushInvoice = async (): Promise<false | undefined> => {
    if (!redFlushTarget?.id || !redFlushTarget.version) return;
    const values = await redFlushForm.validateFields();
    setSubmitting(true);
    try {
      const writeResponse4 = await settlementServiceRedFlushInvoice(
        { id: redFlushTarget.id },
        {
          id: redFlushTarget.id,
          expectedVersion: redFlushTarget.version,
          redInvoiceNo: values.redInvoiceNo,
          redInvoiceDate: values.redInvoiceDate.format('YYYY-MM-DD'),
          reason: values.reason,
        },
      );
      if (!isWriteResponse(writeResponse4, true)) return false;
      message.success('发票已红冲，原账单开票占用已释放');
      setRedFlushTarget(undefined);
      reload();
    } catch (error) {
      message.error(getErrorMessage(error, '发票红冲失败'));
    } finally {
      setSubmitting(false);
    }

    return undefined;
  };

  const columns: ProColumns<API.FinanceInvoice>[] = [
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 150,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '记录编号、发票号或结算单位' },
    },
    {
      title: '记录编号',
      dataIndex: 'recordNo',
      width: 160,
      copyable: true,
      search: false,
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '销项' }, PAYABLE: { text: '进项' } },
      render: (_, r) => (
        <Tag color={r.direction === 'RECEIVABLE' ? 'blue' : 'purple'}>
          {r.direction === 'RECEIVABLE' ? '销项' : '进项'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(invoiceStates).map(([k, v]) => [
          k,
          {
            text:
              Number(k) === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED
                ? '已开具 / 已收票'
                : v.text,
          },
        ]),
      ),
      render: (_, r) => {
        const v =
          invoiceStates[
            r.status ?? FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT
          ];
        return (
          <Tag color={v?.color}>{invoiceStateText(r.status, r.direction)}</Tag>
        );
      },
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '发票类型',
      dataIndex: 'invoiceType',
      width: 100,
      search: false,
      renderText: (v) => (v === 'SPECIAL' ? '专用发票' : '普通发票'),
    },
    {
      title: '金额',
      dataIndex: 'totalAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong style={{ color: '#262626' }}>
          {formatAmount(r.totalAmount)} {r.currency}
        </strong>
      ),
    },
    {
      title: '开票汇率',
      dataIndex: 'exchangeRate',
      width: 135,
      align: 'right',
      search: false,
      render: (_, r) => {
        if (!r.exchangeRate) {
          return r.status ===
            FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT ? (
            <span style={{ color: '#8c8c8c' }}>开票时确定</span>
          ) : (
            '-'
          );
        }
        const sourceLabel =
          r.exchangeRateSource === 'MANUAL'
            ? '手工'
            : r.exchangeRateSource === 'BASE_CURRENCY'
              ? '本币'
              : '系统';
        const sourceColor =
          r.exchangeRateSource === 'MANUAL' ? 'purple' : 'default';
        return (
          <Space size={4}>
            <span>{trimDecimal(r.exchangeRate)}</span>
            <Tag color={sourceColor} style={{ margin: 0, fontSize: 10 }}>
              {sourceLabel}
            </Tag>
          </Space>
        );
      },
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 145,
      align: 'right',
      search: false,
      render: (_, r) =>
        r.baseCurrencyAmount ? (
          <strong
            style={{
              color: r.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
            }}
          >
            {formatAmount(r.baseCurrencyAmount)} {r.baseCurrency}
          </strong>
        ) : (
          '-'
        ),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 120,
      align: 'right',
      search: false,
      render: (_, r) => formatAmount(r.taxAmount),
    },
    {
      title: '税务发票号',
      dataIndex: 'taxInvoiceNo',
      width: 160,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '开票日期',
      dataIndex: 'invoiceDate',
      width: 110,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 210,
      render: (_, r) => [
        <a key="detail" onClick={() => void showDetail(r)}>
          <EyeOutlined /> 详情
        </a>,
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED ? (
          <a key="revise" onClick={() => setReviseTarget(r)}>
            修订 / 影响检查
          </a>
        ) : null,
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        (r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT ||
          r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED) ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => {
              cancelForm.resetFields();
              setCancelTarget(r);
            }}
          >
            <CloseCircleOutlined />{' '}
            {r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED
              ? '作废'
              : '取消'}
          </a>
        ) : null,
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED ? (
          <a
            key="red-flush"
            style={{ color: '#cf1322' }}
            onClick={() => {
              setRedFlushTarget(r);
              redFlushForm.setFieldsValue({ redInvoiceDate: dayjs() });
            }}
          >
            <SwapOutlined /> 红冲
          </a>
        ) : null,
      ],
    },
  ];

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-invoices',
      title: '发票总记录数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-invoices',
      title: '已开票销项金额',
      value: formatBaseCurrencyAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'pay-invoices',
      title: '已开票进项金额',
      value: formatBaseCurrencyAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'issued-count',
      title: '已开具 / 已收票',
      value: metricStats.issuedCount,
      suffix: '笔',
      valueColor: '#52c41a',
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<
        API.FinanceInvoice,
        InvoiceLedgerFilterParams,
        API.FinanceInvoiceSummary
      >
        pageTitle="发票管理"
        pageSubTitle="进项与销项发票开具、收票核验与发票状态跟踪"
        headerTitle="发票明细列表"
        actionRef={actionRef}
        exportColumns={invoiceExportColumns}
        columns={columns}
        columnSettingsKey="finance:invoices"
        metricCards={metricCards}
        search={false}
        pagination={queryState.pagination}
        customSearch={
          <SearchFilterTemplate<InvoiceLedgerFilterParams>
            items={invoiceFilterItems}
            initialValues={queryState.initial.filters}
            initialLabels={queryState.initial.labels}
            visibleNames={filterSettings.visibleNames}
            extraRight={
              <FilterSettingsEntry
                fields={invoiceFilterFields}
                visibleNames={filterSettings.visibleNames}
                onChange={filterSettings.setVisible}
                onReset={filterSettings.reset}
              />
            }
            onCommit={(values, labels) => commitFilters(values, labels)}
            onSearch={(values) => {
              commitFilters(values);
              actionRef.current?.reload(true);
            }}
            onReset={() => {
              commitFilters({}, {});
              actionRef.current?.reload(true);
            }}
          />
        }
        primaryActionText={
          access.canCreateFinanceInvoices ? '从账单登记发票' : undefined
        }
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={() => {
          setCreateOpen(true);
        }}
        summaryConfig={invoiceLedgerSummaryConfig}
        selectionScope={selectionScope}
        onAcceptedSnapshot={(snapshot) => {
          setMetricStats({
            totalCount: snapshot.total,
            issuedCount: Number(snapshot.summary?.issuedCount ?? 0),
            amountsByBaseCurrency:
              snapshot.summary?.amountsByBaseCurrency ?? [],
          });
        }}
        request={async (p) => {
          if (!hasCompanyScope) return { data: [], success: false, total: 0 };
          const r = await settlementServiceListInvoices(
            {
              page: p.current,
              pageSize: p.pageSize,
              keyword: queryState.snapshotRef.current.filters.keyword,
              direction: queryState.snapshotRef.current.filters.direction,
              status: queryState.snapshotRef.current.filters.status
                ? Number(queryState.snapshotRef.current.filters.status)
                : undefined,
              organizationId,
            },
            { skipErrorHandler: true },
          );
          ensureListResponse(r, '财务列表加载失败，请稍后重试');
          const page = unwrapPage(r);
          return {
            ...toTableRequest(r),
            total: page.total,
            summary: r.summary,
            matchedSummary: mapMatchedSummary(r.matchedSummary),
          };
        }}
      />

      <InvoiceRegistrationModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => void actionRef.current?.reload()}
      />
      <InvoiceRegistrationModal
        open={Boolean(reviseTarget)}
        reviseInvoice={reviseTarget}
        onOpenChange={(open) => {
          if (!open) setReviseTarget(undefined);
        }}
        onSuccess={() => void actionRef.current?.reload()}
      />

      <InvoiceRedFlushModal
        open={Boolean(redFlushTarget)}
        submitting={submitting}
        redFlushTarget={redFlushTarget}
        redFlushForm={redFlushForm}
        onCancel={() => setRedFlushTarget(undefined)}
        onOk={redFlushInvoice}
      />

      <InvoiceCancelModal
        open={Boolean(cancelTarget)}
        submitting={submitting}
        cancelForm={cancelForm}
        cancelTarget={cancelTarget}
        onCancel={() => setCancelTarget(undefined)}
        onOk={cancelInvoice}
      />

      <InvoiceDetailDrawer
        detail={detail}
        onClose={() => setDetail(undefined)}
      />
    </>
  );
}
