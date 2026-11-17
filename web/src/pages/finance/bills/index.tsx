import { PlusOutlined } from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { App, Form } from 'antd';
import type { Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
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
import { FinanceBillStatus } from '@/enums.generated';
import { BillCreationWorkbench } from '@/features/finance/bill-creation';
import {
  billInvoiceProgressMeta,
  billSettlementProgressMeta,
} from '@/features/finance/bill-status';
import {
  billLedgerSummaryConfig,
  mapMatchedSummary,
  useLedgerSelectionScope,
} from '@/features/finance/ledger-summary';
import { fetchCurrencyOptions } from '@/features/master-data/currencies';
import { searchPartnerOptions } from '@/features/partners';
import {
  settlementServiceCancelBill,
  settlementServiceGetBill,
  settlementServiceListBills,
  settlementServiceUpdateBill,
} from '@/services/roncin/settlementService';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
  unwrapPage,
} from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount, trimDecimal } from '@/utils/format';
import { makeVersionActions } from '@/utils/versionActions';
import BillDetailDrawer from './components/BillDetailDrawer';
import BillEditModal from './components/BillEditModal';
import { getFinanceBillColumns } from './components/billColumns';
import type { BillFormValues } from './components/billConstants';

import { billExportColumns } from './ledgerExport';

/** 搜索筛选表单提交并暂存于页面状态的账单查询参数 */
type BillSearchParams = {
  keyword?: string;
  direction?: string;
  settlementPartyId?: string;
  currency?: string;
  billDateRange?: [Dayjs, Dayjs];
  dueDateRange?: [Dayjs, Dayjs];
  onlyUnsettled?: string;
  onlyOverdue?: string;
  settlementProgress?: number;
  invoiceProgress?: number;
  completed?: string;
};

/** 筛选设置可选字段：默认全部可见。 */
const billFilterFields: FilterSettingsField[] = [
  { name: 'keyword', label: '综合搜索' },
  { name: 'direction', label: '账单属性' },
  { name: 'billDateRange', label: '创建日期' },
  { name: 'dueDateRange', label: '到期日期' },
  { name: 'onlyUnsettled', label: '结清状态' },
  { name: 'onlyOverdue', label: '逾期状态' },
  { name: 'settlementProgress', label: '结算进度' },
  { name: 'invoiceProgress', label: '开票进度' },
  { name: 'completed', label: '完成状态' },
  { name: 'settlementPartyId', label: '结算单位' },
  { name: 'currency', label: '计价币种' },
].map((field) => ({ ...field, defaultVisible: true }));

export default function FinanceBillsPage() {
  const access = useAccess();
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  const organizationId = currentUser?.currentOrganization?.id;
  const hasCompanyScope = Boolean(
    currentUser?.id &&
      organizationId &&
      access.canOperateOrganization(organizationId),
  );
  const getModalOwnerSignal = useModalOwnerSignal();
  const filterSettings = useFilterSettings({
    pageKey: 'finance:bills',
    fields: billFilterFields,
  });
  const queryState = useListQueryState<{
    search: BillSearchParams;
  }>({
    tabKey: '/finance/bills',
    listId: 'finance:bills',
    defaultPageSize: 20,
    defaultFilters: { search: {} },
    // 页签恢复时剔除已隐藏筛选项的值，避免隐藏条件继续生效。
    sanitizeRestored: (filters) => ({
      ...filters,
      search: omitHiddenFilterValues(
        filters.search,
        billFilterFields,
        filterSettings.visibleNames,
      ),
    }),
  });
  const { selectionScope, commitFilters } = useLedgerSelectionScope(
    queryState.commit,
    () => queryState.snapshotRef.current.filters,
  );
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [form] = Form.useForm<BillFormValues>();
  const [workbenchOpen, setWorkbenchOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<API.FinanceBill>();
  const [submitting, setSubmitting] = useState(false);
  const editSession = useRef(0);
  const queryClient = useQueryClient();
  const [detailId, setDetailId] = useState<string>();
  const detailQuery = useQuery({
    queryKey: ['finance', 'bill-detail', { id: detailId }],
    enabled: hasCompanyScope && Boolean(detailId),
    meta: { errorMessage: '加载账单详情失败' },
    queryFn: async ({ signal }) => {
      if (!detailId) throw new Error('缺少账单标识');
      const response = await settlementServiceGetBill(
        { id: detailId },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data?.id || response.success === false)
        throw new Error('加载账单详情失败');
      return response.data;
    },
  });
  const detail = detailQuery.data;
  const detailOpen = Boolean(detailId);
  const detailLoading = detailQuery.isFetching;
  const setDetail = (updated: API.FinanceBill) => {
    queryClient.setQueryData(
      ['finance', 'bill-detail', { id: updated.id }],
      updated,
    );
  };

  // 统计指标
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    amountsByBaseCurrency: [] as API.FinanceBaseCurrencyAmount[],
  });

  const formatBaseCurrencyAmounts = (
    field:
      | 'receivableBaseAmount'
      | 'payableBaseAmount'
      | 'unverifiedBaseAmount'
      | 'overdueReceivableBaseAmount',
  ) =>
    metricStats.amountsByBaseCurrency
      .map(
        (item) =>
          `${formatAmount(item[field] ?? '0')} ${item.baseCurrency ?? '-'}`,
      )
      .join(' / ') || '-';

  const filterItems: SearchFilterFieldItem[] = [
    {
      name: 'keyword',
      label: '综合搜索',
      placeholder: '输入账单编号/对账抬头/结算单位',
    },
    {
      name: 'direction',
      label: '账单属性',
      type: 'select',
      placeholder: '全部属性',
      options: [
        { label: '应收 (RECEIVABLE)', value: 'RECEIVABLE' },
        { label: '应付 (PAYABLE)', value: 'PAYABLE' },
      ],
    },
    {
      name: 'billDateRange',
      label: '创建日期',
      type: 'date-range',
      placeholder: ['开始日期', '结束日期'],
    },
    {
      name: 'dueDateRange',
      label: '到期日期',
      type: 'date-range',
      placeholder: ['到期开始', '到期结束'],
    },
    {
      name: 'onlyUnsettled',
      label: '结清状态',
      type: 'select',
      placeholder: '全部',
      options: [{ label: '仅看未结清', value: 'true' }],
    },
    {
      name: 'onlyOverdue',
      label: '逾期状态',
      type: 'select',
      placeholder: '全部',
      options: [{ label: '仅看已逾期', value: 'true' }],
    },
    {
      name: 'settlementProgress',
      label: '结算进度',
      type: 'select',
      placeholder: '全部',
      // 选项派生自共享进度映射，与列表列、导出保持单一真相。
      options: Object.entries(billSettlementProgressMeta).map(
        ([value, meta]) => ({ label: meta.text, value: Number(value) }),
      ),
    },
    {
      name: 'invoiceProgress',
      label: '开票进度',
      type: 'select',
      placeholder: '全部',
      options: Object.entries(billInvoiceProgressMeta).map(([value, meta]) => ({
        label: meta.text,
        value: Number(value),
      })),
    },
    {
      name: 'completed',
      label: '完成状态',
      type: 'select',
      placeholder: '全部',
      options: [
        { label: '已完成', value: 'true' },
        { label: '未完成', value: 'false' },
      ],
    },
    {
      name: 'settlementPartyId',
      label: '结算单位',
      type: 'searchable-select',
      placeholder: '输入名称/全拼搜索结算单位',
      requestKey: [hasCompanyScope],
      request: ({ keyWords, signal }) =>
        hasCompanyScope
          ? searchPartnerOptions(
              keyWords,
              {},
              { signal, skipErrorHandler: true },
            )
          : Promise.resolve([]),
    },
    {
      name: 'currency',
      label: '计价币种',
      type: 'searchable-select',
      placeholder: '全部币种',
      request: ({ signal }) =>
        fetchCurrencyOptions({ signal, skipErrorHandler: true }),
    },
  ];

  const reload = () => actionRef.current?.reload();
  const billActions = makeVersionActions<API.FinanceBill>(
    { modal, message },
    getModalOwnerSignal,
  );
  const openCreate = () => {
    setWorkbenchOpen(true);
  };

  const openEdit = (bill: API.FinanceBill) => {
    editSession.current += 1;
    setSubmitting(false);
    setEditing(bill);
    form.setFieldsValue({
      statementTitle: bill.statementTitle,
      paymentTermsDays: bill.paymentTermsDays,
      note: bill.note,
      settlementAccountId: bill.settlementAccountId,
      estimatedInvoiceCurrency: bill.estimatedInvoiceCurrency || bill.currency,
      // 服务端 decimal 原串带 8 位尾零，预填输入框前收敛为变长小数
      estimatedInvoiceRate: bill.estimatedInvoiceRate
        ? trimDecimal(bill.estimatedInvoiceRate)
        : undefined,
    });
    setEditOpen(true);
  };

  const openDetail = (bill: API.FinanceBill) => {
    if (!bill.id || !hasCompanyScope) return;
    if (detailId === bill.id) {
      void queryClient.invalidateQueries({
        queryKey: ['finance', 'bill-detail'],
      });
    } else {
      setDetailId(bill.id);
    }
  };

  const closeEdit = () => {
    editSession.current += 1;
    setEditOpen(false);
    setSubmitting(false);
  };
  const refreshEditing = async (): Promise<boolean> => {
    const bill = editing;
    const session = editSession.current;
    if (!bill?.id || !hasCompanyScope) return false;
    try {
      const latest = await queryClient.fetchQuery({
        queryKey: ['finance', 'bill-edit-refresh', { id: bill.id, session }],
        staleTime: 0,
        meta: { silent: true },
        queryFn: async ({ signal }) => {
          const response = await settlementServiceGetBill(
            { id: bill.id as string },
            { signal, skipErrorHandler: true },
          );
          if (
            !response?.data?.version ||
            response.data.id !== bill.id ||
            response.success === false
          )
            throw new Error('刷新账单失败，请重试');
          return response.data;
        },
      });
      if (session !== editSession.current) return false;
      openEdit(latest);
      return true;
    } catch (error) {
      if (session === editSession.current)
        message.error(getErrorMessage(error, '刷新账单失败，请重试'));
      return false;
    }
  };
  const submitBill = async (): Promise<false | undefined> => {
    if (!editing?.id || !editing.version) return false;
    const session = editSession.current;
    const values = await form.validateFields();
    if (session !== editSession.current) return false;
    setSubmitting(true);
    try {
      const response = await settlementServiceUpdateBill(
        { id: editing.id },
        {
          id: editing.id,
          expectedVersion: editing.version,
          statementTitle: values.statementTitle.trim(),
          paymentTermsDays: values.paymentTermsDays,
          note: values.note?.trim() || undefined,
          settlementAccountId: values.settlementAccountId,
          estimatedInvoiceCurrency:
            values.estimatedInvoiceCurrency || undefined,
          estimatedInvoiceRate: values.estimatedInvoiceRate || undefined,
        },
      );
      if (session !== editSession.current || !response) return false;
      if (response.data?.id !== editing.id || response.success === false)
        throw new Error('更新账单失败，请重试');
      message.success('账单已成功更新');
      setEditOpen(false);
      void queryClient.invalidateQueries({ queryKey: ['finance'] });
      reload();
    } catch (error) {
      if (session === editSession.current)
        message.error(getErrorMessage(error, '更新账单失败'));
      return false;
    } finally {
      if (session === editSession.current) setSubmitting(false);
    }
    return undefined;
  };

  const cancelBill = async (bill: API.FinanceBill) => {
    if (!bill.id) return;
    const signal = getModalOwnerSignal();
    let fullBill: API.FinanceBill;
    try {
      const response = await settlementServiceGetBill(
        { id: bill.id },
        { signal },
      );
      if (signal.aborted) return;
      if (
        !response?.data?.id ||
        !response.data.version ||
        !response.data.lines?.length
      ) {
        message.error('账单明细未完整加载，请刷新后重试');
        return;
      }
      fullBill = response.data;
    } catch (error) {
      if (!signal.aborted)
        message.error(getErrorMessage(error, '加载账单取消影响范围失败'));
      return;
    }
    const activeLines = (fullBill.lines ?? []).filter(
      (line) => line.active !== false,
    );
    if (!activeLines.length) {
      message.error('账单没有可释放的有效费用，请刷新后重试');
      return;
    }
    const orderCount = new Set(activeLines.map((line) => line.orderId)).size;
    const orderNos = [
      ...new Set(activeLines.map((line) => line.orderNo).filter(Boolean)),
    ];
    billActions.confirm(
      fullBill,
      `确认取消 ${fullBill.organizationName || '所属公司未标识'} 的整张账单 ${fullBill.billNo || ''}？将释放涉及 ${orderCount} 个订单的 ${activeLines.length} 笔费用${orderNos.length ? `（${orderNos.join('、')}）` : ''}。合并账单内所有关联费用会一并释放；如有发票、核销或对冲，须先按提示逐步回退。`,
      async ({ id, expectedVersion }, reason): Promise<false | undefined> => {
        const writeResponse2 = await settlementServiceCancelBill(
          { id },
          { id, expectedVersion, reason },
        );
        if (!isWriteResponse(writeResponse2, true)) return false;
        message.success('账单已取消，关联明细费用已释放并可重新建单');
        reload();

        return undefined;
      },
      {
        danger: true,
        signal,
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入取消原因',
      },
    );
  };

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-bills',
      title: '筛选账单总记录数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-bills',
      title: '有效应收账单折本币',
      value: formatBaseCurrencyAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'pay-bills',
      title: '有效应付账单折本币',
      value: formatBaseCurrencyAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'unv-bills',
      title: '有效账单未核销折本币',
      value: formatBaseCurrencyAmounts('unverifiedBaseAmount'),
      valueColor: metricStats.amountsByBaseCurrency.some(
        (item) => Number(item.unverifiedBaseAmount ?? 0) > 0,
      )
        ? '#cf1322'
        : '#52c41a',
    },
    {
      key: 'overdue-bills',
      title: '有效账单逾期应收折本币',
      value: formatBaseCurrencyAmounts('overdueReceivableBaseAmount'),
      valueColor: metricStats.amountsByBaseCurrency.some(
        (item) => Number(item.overdueReceivableBaseAmount ?? 0) > 0,
      )
        ? '#cf1322'
        : '#52c41a',
    },
  ];

  const columns = getFinanceBillColumns({
    access,
    onOpenDetail: openDetail,
    onOpenEdit: openEdit,
    onCancelBill: cancelBill,
  });

  return (
    <>
      <FinanceLedgerTemplate<
        API.FinanceBill,
        Record<string, unknown>,
        API.FinanceBillSummary
      >
        pagination={queryState.pagination}
        pageTitle="账单管理"
        pageSubTitle="集中管理与跟踪海运出口及各业务线收付账单、开票与结算流转状态"
        headerTitle="账单列表"
        actionRef={actionRef}
        exportColumns={billExportColumns}
        columns={columns}
        columnSettingsKey="finance:bills"
        metricCards={metricCards}
        search={false}
        customSearch={
          <SearchFilterTemplate<BillSearchParams>
            initialValues={queryState.initial.filters.search}
            initialLabels={queryState.initial.labels}
            onCommit={(values, labels) =>
              commitFilters(
                { ...queryState.snapshotRef.current.filters, search: values },
                labels,
              )
            }
            layout="grid"
            formLayout="horizontal"
            labelWidth={80}
            collapsible={true}
            defaultCollapsed={true}
            defaultVisibleCount={5}
            items={filterItems}
            visibleNames={filterSettings.visibleNames}
            extraRight={
              <FilterSettingsEntry
                fields={billFilterFields}
                visibleNames={filterSettings.visibleNames}
                onChange={filterSettings.setVisible}
                onReset={filterSettings.reset}
              />
            }
            onSearch={(values) => {
              commitFilters({
                ...queryState.snapshotRef.current.filters,
                search: values,
              });
              actionRef.current?.reload(true);
            }}
            onReset={() => {
              commitFilters(
                { ...queryState.snapshotRef.current.filters, search: {} },
                {},
              );
              actionRef.current?.reload(true);
            }}
          />
        }
        primaryActionText={
          access.canCreateFinanceBills ? '批量创建账单' : undefined
        }
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={openCreate}
        summaryConfig={billLedgerSummaryConfig}
        selectionScope={selectionScope}
        onAcceptedSnapshot={(snapshot) => {
          setMetricStats({
            totalCount: snapshot.total,
            amountsByBaseCurrency:
              snapshot.summary?.amountsByBaseCurrency ?? [],
          });
        }}
        request={async (params) => {
          if (!hasCompanyScope) {
            return { data: [], success: false, total: 0 };
          }
          const { search: searchParams } =
            queryState.snapshotRef.current.filters;
          const billDateFrom = searchParams.billDateRange?.[0]
            ? searchParams.billDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const billDateTo = searchParams.billDateRange?.[1]
            ? searchParams.billDateRange[1].format('YYYY-MM-DD')
            : undefined;
          const dueDateFrom = searchParams.dueDateRange?.[0]
            ? searchParams.dueDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const dueDateTo = searchParams.dueDateRange?.[1]
            ? searchParams.dueDateRange[1].format('YYYY-MM-DD')
            : undefined;

          const response = await settlementServiceListBills(
            {
              page: params.current,
              pageSize: params.pageSize,
              keyword: searchParams.keyword || undefined,
              direction: searchParams.direction || undefined,
              // 日常列表只查询有效账单，分页、汇总与导出沿用同一范围。
              status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
              settlementPartyId: searchParams.settlementPartyId || undefined,
              currency: searchParams.currency || undefined,
              billDateFrom,
              billDateTo,
              dueDateFrom,
              dueDateTo,
              onlyUnsettled:
                searchParams.onlyUnsettled === 'true' ? true : undefined,
              onlyOverdue:
                searchParams.onlyOverdue === 'true' ? true : undefined,
              settlementProgress: searchParams.settlementProgress,
              invoiceProgress: searchParams.invoiceProgress,
              completed:
                searchParams.completed === 'true'
                  ? true
                  : searchParams.completed === 'false'
                    ? false
                    : undefined,
              organizationId,
            },
            { skipErrorHandler: true },
          );
          ensureListResponse(response, '财务列表加载失败，请稍后重试');

          const page = unwrapPage(response);
          return {
            ...toTableRequest(response),
            total: page.total,
            summary: response.summary,
            matchedSummary: mapMatchedSummary(response.matchedSummary),
          };
        }}
      />

      <BillEditModal
        key={editing?.id}
        open={editOpen}
        editing={editing}
        form={form}
        submitting={submitting}
        onCancel={closeEdit}
        onOk={submitBill}
        onRefresh={refreshEditing}
      />

      <BillCreationWorkbench
        open={workbenchOpen}
        onClose={() => setWorkbenchOpen(false)}
        onCreated={() => reload()}
      />

      <BillDetailDrawer
        open={detailOpen}
        loading={detailLoading}
        detail={detail}
        onClose={() => setDetailId(undefined)}
        onRegistrationUpdated={() => {
          if (detail) void openDetail(detail);
          void actionRef.current?.reload();
        }}
        invoiceExemptionActions={
          access.canExemptFinanceInvoices &&
          hasCompanyScope &&
          access.canOperateOrganization(detail?.organizationId)
            ? {
                onUpdated: (updated) => {
                  setDetail(updated);
                  reload();
                },
                onRefresh: () => {
                  if (detail) void openDetail(detail);
                },
              }
            : undefined
        }
        feeMaintenance={
          access.canUpdateFinanceBills && hasCompanyScope
            ? {
                onUpdated: (updated) => {
                  setDetail(updated);
                  reload();
                },
                onRefresh: () => {
                  if (detail) void openDetail(detail);
                },
              }
            : undefined
        }
      />
    </>
  );
}
