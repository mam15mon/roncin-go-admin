import type { ActionType } from '@ant-design/pro-components';
import { App } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
  getDefaultRowColors,
  RowColorSettings,
  type RowColorsConfig,
} from '@/components/ui';
import {
  FilterSettingsEntry,
  omitHiddenFilterValues,
  useFilterSettings,
} from '@/components/ui/filter-settings';
import { useListQueryState } from '@/components/ui/list-query-state';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import {
  type BillCreationMode,
  BillCreationWorkbench,
} from '@/features/finance/bill-creation';
import {
  feeLedgerSummaryConfig,
  mapMatchedSummary,
  useLedgerSelectionScope,
} from '@/features/finance/ledger-summary';
import { history } from '@/router/history';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';
import { ensureListResponse, toTableRequest, unwrapPage } from '@/utils/api';
import { formatAmount } from '@/utils/format';
import {
  type FeeLedgerFilterParams,
  FeeLedgerSearchFilter,
  feeLedgerFilterFields,
} from './components/FeeLedgerSearchFilter';
import {
  amount,
  financialProgressLabels,
  getBaseFeeLedgerColumns,
} from './components/feeLedgerColumns';
import {
  loadFeeLedgerRowColors,
  saveFeeLedgerRowColors,
} from './feeLedgerViewPreference';

import { feeExportColumns } from './ledgerExport';

export function resolveSingleBillCreationOrganization(
  rows: API.FeeLedgerItem[],
  currentOrganizationId?: string,
) {
  return currentOrganizationId &&
    rows.length > 0 &&
    rows.every((row) => row.organizationId === currentOrganizationId)
    ? currentOrganizationId
    : undefined;
}

// 费用台账允许检查双向往来，但普通账单必须由财务按方向分别发起。
export function hasMixedBillDirections(rows: API.FeeLedgerItem[]) {
  return new Set(rows.map((row) => row.direction).filter(Boolean)).size > 1;
}

// 对冲建账要求同一批费用同时包含应收和应付；单方向费用只能走普通账单，
// 页面不自动替用户切换模式。
export function hasBothBillDirections(rows: API.FeeLedgerItem[]) {
  const directions = new Set(rows.map((row) => row.direction).filter(Boolean));
  return directions.has('RECEIVABLE') && directions.has('PAYABLE');
}

/** 表头偏好为本浏览器本地配置，按「用户 + 当前组织」隔离。 */

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  const organizationId = currentUser?.currentOrganization?.id;
  const rowColorScope = useMemo(
    () => ({
      userId: currentUser?.id,
      organizationId: currentUser?.currentOrganization?.id,
    }),
    [currentUser?.id, currentUser?.currentOrganization?.id],
  );
  const [rowColors, setRowColors] = useState<RowColorsConfig>(() =>
    getDefaultRowColors(),
  );
  // 登录用户与组织就绪后重新解析对应隔离范围的本地行配色偏好。
  useEffect(() => {
    setRowColors(
      loadFeeLedgerRowColors(rowColorScope) ?? getDefaultRowColors(),
    );
  }, [rowColorScope]);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [billWorkbenchMode, setBillWorkbenchMode] =
    useState<BillCreationMode>('NORMAL');
  const [selectedFeeIds, setSelectedFeeIds] = useState<string[]>([]);
  const [selectedBillOrganizationId, setSelectedBillOrganizationId] =
    useState<string>();
  const filterSettings = useFilterSettings({
    pageKey: 'finance:fees',
    fields: feeLedgerFilterFields,
  });
  const queryState = useListQueryState<{
    search: FeeLedgerFilterParams;
  }>({
    tabKey: '/finance/fees',
    listId: 'finance:fees',
    defaultPageSize: 40,
    defaultFilters: { search: {} },
    // 页签恢复时剔除已隐藏筛选项的值，避免隐藏条件继续生效。
    sanitizeRestored: (filters) => ({
      ...filters,
      search: omitHiddenFilterValues(
        filters.search,
        feeLedgerFilterFields,
        filterSettings.visibleNames,
      ),
    }),
  });
  const { selectionScope, commitFilters } = useLedgerSelectionScope(
    queryState.commit,
    () => queryState.snapshotRef.current.filters,
  );
  const handleSearch = (values: FeeLedgerFilterParams) => {
    commitFilters({
      ...queryState.snapshotRef.current.filters,
      search: values,
    });
    actionRef.current?.reload(true);
  };

  const handleReset = () => {
    commitFilters(
      { ...queryState.snapshotRef.current.filters, search: {} },
      {},
    );
    actionRef.current?.reload(true);
  };

  // 建账资格以既有账单信息为口径：关联账单号存在（含草稿账单占用）即不可建账；
  // 候选事实由服务端在最终建单事务内复核。
  const canCreateBill = (row: API.FeeLedgerItem) =>
    access.canOperateOrganization(row.organizationId) && row.canBill === true;

  const formatAmounts = (
    field: 'receivableBaseAmount' | 'payableBaseAmount' | 'profitBaseAmount',
  ) =>
    summary?.amountsByBaseCurrency
      ?.map(
        (item) =>
          `${formatAmount(item[field] ?? '0')} ${item.baseCurrency || '-'}`,
      )
      .join(' / ') || '-';
  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'active-count',
      title: '有效费用总笔数',
      value: Number(summary?.activeCount || 0),
      suffix: '笔',
    },
    {
      key: 'receivable-base',
      title: '有效费用应收折本币',
      value: formatAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'payable-base',
      title: '有效费用应付折本币',
      value: formatAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'profit-base',
      title: '有效费用利润',
      value: formatAmounts('profitBaseAmount'),
      valueColor: (summary?.amountsByBaseCurrency ?? []).every(
        (item) => amount(item.profitBaseAmount) >= 0,
      )
        ? '#52c41a'
        : '#ff4d4f',
    },
  ];

  const access = useAccess();
  const hasCompanyScope = Boolean(
    currentUser?.id &&
      organizationId &&
      access.canOperateOrganization(organizationId),
  );
  const baseColumns = useMemo(() => getBaseFeeLedgerColumns(), []);

  // 列显隐与排序由统一列设置浮层按 columnSettingsKey 即时应用与持久化。

  // 行配色即时生效并持久化到浏览器本地；存储失败明确提示且不声称已保存。
  const handleRowColorChange = (key: keyof RowColorsConfig, color: string) => {
    const next = { ...rowColors, [key]: color };
    setRowColors(next);
    if (!saveFeeLedgerRowColors(rowColorScope, next)) {
      message.error('行配色保存到本地浏览器失败，本次设置仅当前页面生效');
    }
  };

  const handleResetRowColors = () => {
    const next = getDefaultRowColors();
    setRowColors(next);
    if (!saveFeeLedgerRowColors(rowColorScope, next)) {
      message.error('行配色保存到本地浏览器失败，本次设置仅当前页面生效');
      return;
    }
    message.success('行背景高亮颜色已重置为默认值');
  };

  // 根据费用财务进度映射对应的行高亮背景 key（支持 7 状态）
  const getRowStatusColorKey = (row: API.FeeLedgerItem) => {
    const progress =
      row.financialProgress ??
      FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED;
    const matched = financialProgressLabels[progress];
    return matched?.key;
  };

  return (
    <>
      <FinanceLedgerTemplate<
        API.FeeLedgerItem,
        Record<string, unknown>,
        API.FeeLedgerSummary
      >
        pagination={queryState.pagination}
        pageTitle="费用明细台账"
        pageSubTitle="全维度多币种费用台账，支持按单据、财务进度、结算单位快速对账与生成账单"
        headerTitle="费用明细台账"
        actionRef={actionRef}
        exportColumns={feeExportColumns}
        columns={baseColumns}
        metricCards={metricCards}
        customSearch={
          <FeeLedgerSearchFilter
            enabled={hasCompanyScope}
            initialValues={queryState.initial.filters.search}
            initialLabels={queryState.initial.labels}
            onCommit={(values, labels) =>
              commitFilters(
                { ...queryState.snapshotRef.current.filters, search: values },
                labels,
              )
            }
            onSearch={handleSearch}
            onReset={handleReset}
            visibleNames={filterSettings.visibleNames}
            extraRight={
              <FilterSettingsEntry
                fields={feeLedgerFilterFields}
                visibleNames={filterSettings.visibleNames}
                onChange={filterSettings.setVisible}
                onReset={filterSettings.reset}
              />
            }
          />
        }
        search={false}
        primaryActionText={
          access.canCreateFinanceBills ? '创建账单' : undefined
        }
        primaryActionRequiresSelection
        onPrimaryAction={(keys, rows) => {
          const invalidRows = rows.filter((row) => !canCreateBill(row));
          if (invalidRows.length > 0) {
            message.warning(
              '请选择可建账的未建账费用；待审补录仅限制对应原费用，具体原因见建账限制列',
            );
            return;
          }
          if (hasMixedBillDirections(rows)) {
            message.warning(
              '普通账单不能同时包含应收和应付，请只保留一个方向后再建账，或改用对冲账单。',
            );
            return;
          }
          const selectedOrganizationID = resolveSingleBillCreationOrganization(
            rows,
            organizationId,
          );
          if (!selectedOrganizationID) {
            message.warning('所选费用缺少所属公司或跨公司，不能创建账单');
            return;
          }
          setSelectedFeeIds(keys.map(String));
          setSelectedBillOrganizationId(selectedOrganizationID);
          setBillWorkbenchMode('NORMAL');
          setBillWorkbenchOpen(true);
        }}
        batchActions={[
          ...(access.canCreateFinanceNettings
            ? [
                {
                  key: 'create-netting-bill',
                  label: '创建对冲账单',
                  onClick: (_keys: React.Key[], rows: API.FeeLedgerItem[]) => {
                    const invalidRows = rows.filter(
                      (row) => !canCreateBill(row),
                    );
                    if (invalidRows.length > 0) {
                      message.warning(
                        '请选择可建账的未建账费用；待审补录仅限制对应原费用，具体原因见建账限制列',
                      );
                      return;
                    }
                    if (!hasBothBillDirections(rows)) {
                      message.warning(
                        '对冲账单需同时包含应收和应付费用，请各保留至少一笔后再对冲。',
                      );
                      return;
                    }
                    const selectedOrganizationID =
                      resolveSingleBillCreationOrganization(
                        rows,
                        organizationId,
                      );
                    if (!selectedOrganizationID) {
                      message.warning(
                        '所选费用缺少所属公司或跨公司，不能创建对冲账单',
                      );
                      return;
                    }
                    setSelectedFeeIds(
                      rows.map((row) => row.id || '').filter(Boolean),
                    );
                    setSelectedBillOrganizationId(selectedOrganizationID);
                    setBillWorkbenchMode('NETTING');
                    setBillWorkbenchOpen(true);
                  },
                },
              ]
            : []),
        ]}
        onImport={() => message.info('可通过 Excel 模板批量导入费用明细')}
        columnSettingsKey="finance-fees:ledger"
        columnSettingsStructuralKeys={['direction']}
        advancedSettings={
          <RowColorSettings
            rowColors={rowColors}
            onColorChange={handleRowColorChange}
            onResetColors={handleResetRowColors}
          />
        }
        advancedSettingsTitle="行背景配色设置"
        rowColors={rowColors}
        getRowStatusColorKey={getRowStatusColorKey}
        onRowClick={(row) => {
          if (row.orderId) history.push(`/finance/fees/detail/${row.orderId}`);
        }}
        summaryConfig={feeLedgerSummaryConfig}
        selectionScope={selectionScope}
        onAcceptedSnapshot={(snapshot) => {
          setSummary(snapshot.summary);
        }}
        request={async (params) => {
          if (!hasCompanyScope) return { data: [], success: false, total: 0 };
          const { search: filterParams } =
            queryState.snapshotRef.current.filters;
          const createdAtFrom = filterParams.createdAtRange?.[0]
            ? filterParams.createdAtRange[0].format('YYYY-MM-DD')
            : undefined;
          const createdAtTo = filterParams.createdAtRange?.[1]
            ? filterParams.createdAtRange[1].format('YYYY-MM-DD')
            : undefined;

          const response = await settlementServiceListFeeLedger(
            {
              page: params.current,
              pageSize: params.pageSize,
              keyword:
                filterParams.orderNo ||
                filterParams.houseNo ||
                filterParams.feeName ||
                filterParams.operatorName ||
                filterParams.salesName ||
                filterParams.invoiceNo ||
                filterParams.consignee ||
                filterParams.shipper ||
                filterParams.vesselName ||
                filterParams.voyageNo ||
                filterParams.keyword ||
                undefined,
              masterNo: filterParams.masterNo || undefined,
              billNo: filterParams.billNo || undefined,
              businessType: filterParams.businessType || undefined,
              direction: filterParams.direction || undefined,
              financialProgress: filterParams.financialProgress || undefined,
              orderLocked:
                filterParams.orderLocked === 'LOCKED'
                  ? true
                  : filterParams.orderLocked === 'UNLOCKED'
                    ? false
                    : undefined,
              hasPendingFeeSupplement:
                filterParams.hasPendingFeeSupplement === 'yes'
                  ? true
                  : filterParams.hasPendingFeeSupplement === 'no'
                    ? false
                    : undefined,
              customerId: filterParams.customerId || undefined,
              settlementPartyId: filterParams.settlementPartyId || undefined,
              currency: filterParams.currency || undefined,

              createdAtFrom,
              createdAtTo,
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

      {/* 批量转账单工作台 */}
      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={selectedFeeIds}
        initialOrganizationId={selectedBillOrganizationId}
        initialOrganizationName={currentUser?.currentOrganization?.name}
        sourceLabel={`从费用明细勾选的 ${selectedFeeIds.length} 笔费用`}
        mode={billWorkbenchMode}
        onClose={() => setBillWorkbenchOpen(false)}
        onCreated={() => {
          setBillWorkbenchOpen(false);
          setSelectedFeeIds([]);
          setSelectedBillOrganizationId(undefined);
          actionRef.current?.reload();
        }}
      />
    </>
  );
}
