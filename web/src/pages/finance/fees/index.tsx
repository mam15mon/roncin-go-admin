import type { ActionType } from '@ant-design/pro-components';
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { App, Select, Space } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
  getDefaultRowColors,
  RowColorSettings,
  type RowColorsConfig,
} from '@/components/ui';
import {
  FeeLedgerFinancialProgress,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import {
  type BillCreationMode,
  BillCreationWorkbench,
} from '@/features/finance/bill-creation';
import { history } from '@/router/history';
import {
  settlementServiceBatchAssignFinanceFeeTags,
  settlementServiceBatchRemoveFinanceFeeTags,
  settlementServiceListFeeLedger,
  settlementServiceListFinanceFeeTagAssignmentOptions,
  settlementServiceListFinanceFeeTagOptions,
  settlementServiceListFinanceOrganizationOptions,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import {
  type FeeLedgerFilterParams,
  FeeLedgerSearchFilter,
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

export function resolveSingleBillCreationOrganization(
  rows: API.FeeLedgerItem[],
) {
  const ids = Array.from(
    new Set(rows.map((row) => row.organizationId).filter(Boolean)),
  );
  return rows.length > 0 &&
    rows.every((row) => Boolean(row.organizationId)) &&
    ids.length === 1
    ? ids[0]
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

// 费用标签写入以当前筛选的单一公司为边界；跨组织选择应在请求候选和提交前拦截。
export function feeRowsBelongToOrganization(
  rows: API.FeeLedgerItem[],
  organizationId: string | undefined,
) {
  return (
    Boolean(organizationId) &&
    rows.length > 0 &&
    rows.every((row) => row.organizationId === organizationId)
  );
}

/** 表头偏好为本浏览器本地配置，按「用户 + 当前组织」隔离。 */

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
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
  const [filterParams, setFilterParams] = useState<FeeLedgerFilterParams>({});
  const [organizationId, setOrganizationId] = useState<string>();

  // 顶部筛选所属公司候选：失败经全局 onError 提示原文案。
  const organizationQuery = useQuery({
    queryKey: [
      'finance',
      'organization-options',
      {
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_FEE_READ,
      },
    ],
    meta: { errorMessage: '所属公司候选加载失败' },
    queryFn: async () => {
      const response = await settlementServiceListFinanceOrganizationOptions({
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_FEE_READ,
      });
      return response.data ?? [];
    },
  });
  const organizationOptions = organizationQuery.data ?? [];

  const handleSearch = (values: FeeLedgerFilterParams) => {
    setFilterParams(values);
    actionRef.current?.reload();
  };

  const handleReset = () => {
    setFilterParams({});
    actionRef.current?.reload();
  };

  // 建账资格以既有账单信息为口径：关联账单号存在（含草稿账单占用）即不可建账；
  // 候选事实由服务端在最终建单事务内复核。
  const canCreateBill = (row: API.FeeLedgerItem) =>
    access.canOperateOrganization(row.organizationId) && !row.billNo;

  const formatAmounts = (
    field: 'receivableBaseAmount' | 'payableBaseAmount' | 'profitBaseAmount',
  ) =>
    summary?.amountsByBaseCurrency
      ?.map((item) => `${amount(item[field])} ${item.baseCurrency || '-'}`)
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
      title: '应收折本币总池',
      value: formatAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'payable-base',
      title: '应付折本币总池',
      value: formatAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'profit-base',
      title: '确认综合毛利',
      value: formatAmounts('profitBaseAmount'),
      valueColor: (summary?.amountsByBaseCurrency ?? []).every(
        (item) => amount(item.profitBaseAmount) >= 0,
      )
        ? '#52c41a'
        : '#ff4d4f',
    },
  ];

  const access = useAccess();
  const [tagModalOpen, setTagModalOpen] = useState(false);
  const [tagFeeIds, setTagFeeIds] = useState<string[]>([]);
  const [tagExisting, setTagExisting] = useState<API.BusinessTagSummary[]>([]);
  const [tagSearchKeyword, setTagSearchKeyword] = useState('');
  const [tagFilterIds, setTagFilterIds] = useState<string[]>();
  // 搜索词规范化与原请求一致：空白关键字按未过滤处理，规范化后的词进 queryKey。
  const normalizedTagKeyword = tagSearchKeyword.trim() || undefined;

  // 标签筛选候选：keyword 进 queryKey，公司切换整体换键重查；搜索期间保留
  // 上一关键词结果，下拉不闪空。历史请求失败静默，声明 silent 避免全局
  // onError 重复提示。
  const tagOptionsQuery = useQuery({
    queryKey: [
      'finance',
      'fee-tag-options',
      { organizationId, keyword: normalizedTagKeyword },
    ],
    enabled: !!organizationId,
    placeholderData: keepPreviousData,
    meta: { silent: true },
    queryFn: async () => {
      const response = await settlementServiceListFinanceFeeTagOptions({
        page: 1,
        pageSize: 50,
        keyword: normalizedTagKeyword,
        organizationId,
      });
      return response.tags ?? [];
    },
  });

  // 渲染侧合并候选（保留原 Map 合并语义）：服务端当前结果作为最终覆盖，
  // 未在当前结果命中的已选标签从历史 keyword 查询缓存回填名称保活，
  // 保证已选项始终以名称展示而非裸 ID。
  const tagOptions = useMemo(() => {
    if (!organizationId) return [];
    const selected = new Set(tagFilterIds ?? []);
    const options = new Map<string, { label: string; value: string }>();
    const cachedEntries = queryClient.getQueriesData<API.BusinessTagSummary[]>({
      queryKey: ['finance', 'fee-tag-options', { organizationId }],
    });
    for (const [, cached] of cachedEntries) {
      for (const tag of cached ?? []) {
        if (tag.id && selected.has(tag.id)) {
          options.set(tag.id, { label: tag.name ?? '', value: tag.id });
        }
      }
    }
    for (const tag of tagOptionsQuery.data ?? []) {
      if (tag.id) {
        options.set(tag.id, { label: tag.name ?? '', value: tag.id });
      }
    }
    return [...options.values()];
  }, [organizationId, tagFilterIds, tagOptionsQuery.data, queryClient]);

  const openTagModal = (_keys: React.Key[], rows: API.FeeLedgerItem[]) => {
    if (!organizationId) {
      message.warning('请先选择所属公司后再维护费用标签');
      return;
    }
    if (!rows.length) return;
    if (!feeRowsBelongToOrganization(rows, organizationId)) {
      message.warning('只能维护当前所属公司的费用标签');
      return;
    }
    setTagFeeIds(rows.map((row) => row.id ?? '').filter(Boolean));
    const seen = new Map<string, API.BusinessTagSummary>();
    for (const row of rows)
      for (const tag of row.tags ?? []) if (tag.id) seen.set(tag.id, tag);
    setTagExisting([...seen.values()]);
    setTagModalOpen(true);
  };

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
      <FinanceLedgerTemplate<API.FeeLedgerItem>
        pageTitle="费用明细台账"
        pageSubTitle="全维度多币种费用台账，支持按单据、财务进度、结算单位快速对账与生成账单"
        topBar={
          <Space size={16} align="center" wrap>
            <Space size={8} align="center">
              <span style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.65)' }}>
                所属公司：
              </span>
              <Select
                allowClear
                placeholder="请选择所属公司"
                style={{ minWidth: 220 }}
                value={organizationId}
                options={organizationOptions.map((item) => ({
                  value: item.id,
                  label: item.name ?? item.code ?? item.id,
                }))}
                onChange={(value) => {
                  // 切换或清空公司即整体换键：旧组织的在途结果落在旧 queryKey，
                  // 不再回填；同步重置标签筛选与搜索词，候选按新公司重查。
                  setOrganizationId(value);
                  setTagSearchKeyword('');
                  setTagFilterIds(undefined);
                  actionRef.current?.reload();
                }}
              />
            </Space>
            <Space size={8} align="center">
              <span style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.65)' }}>
                标签筛选：
              </span>
              <Select
                mode="multiple"
                allowClear
                showSearch={{
                  filterOption: false,
                  onSearch: (keyword) => setTagSearchKeyword(keyword),
                }}
                loading={tagOptionsQuery.isFetching}
                style={{ minWidth: 280 }}
                placeholder={
                  organizationId ? '命中任一标签即返回' : '请先选择所属公司'
                }
                options={tagOptions}
                value={tagFilterIds}
                disabled={!organizationId}
                onChange={(value) => {
                  setTagFilterIds(value.length ? value : undefined);
                  actionRef.current?.reload();
                }}
              />
            </Space>
          </Space>
        }
        headerTitle="费用明细台账"
        actionRef={actionRef}
        columns={baseColumns}
        metricCards={metricCards}
        customSearch={
          <FeeLedgerSearchFilter
            onSearch={handleSearch}
            onReset={handleReset}
          />
        }
        scrollX={3200}
        search={false}
        primaryActionText={
          access.canCreateFinanceBills ? '创建账单' : undefined
        }
        primaryActionRequiresSelection
        onPrimaryAction={(keys, rows) => {
          const invalidRows = rows.filter((row) => !canCreateBill(row));
          if (invalidRows.length > 0) {
            message.warning(
              '所选费用中包含不可建账的记录，请仅选择未建账且未入账单的费用',
            );
            return;
          }
          if (hasMixedBillDirections(rows)) {
            message.warning(
              '普通账单不能同时包含应收和应付，请只保留一个方向后再建账，或改用对冲账单。',
            );
            return;
          }
          const selectedOrganizationID =
            resolveSingleBillCreationOrganization(rows);
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
                        '所选费用中包含不可建账的记录，请仅选择未建账且未入账单的费用',
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
                      resolveSingleBillCreationOrganization(rows);
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
          ...(access.canManageFinanceFeeTags &&
          access.canOperateOrganization(organizationId)
            ? [
                {
                  key: 'manage-tags',
                  label: '添加/移除标签',
                  onClick: (keys: React.Key[], rows: API.FeeLedgerItem[]) =>
                    openTagModal(keys, rows),
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
        request={async (params) => {
          const expenseDateFrom = filterParams.expenseDateRange?.[0]
            ? filterParams.expenseDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const expenseDateTo = filterParams.expenseDateRange?.[1]
            ? filterParams.expenseDateRange[1].format('YYYY-MM-DD')
            : undefined;

          const response = await settlementServiceListFeeLedger({
            page: params.current,
            pageSize: params.pageSize,
            keyword:
              filterParams.orderNo ||
              filterParams.masterNo ||
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
            billNo: filterParams.billNo || undefined,
            businessType: filterParams.businessType || undefined,
            direction: filterParams.direction || undefined,
            financialProgress: filterParams.financialProgress || undefined,
            customerId: filterParams.customerId || undefined,
            settlementPartyId: filterParams.settlementPartyId || undefined,
            currency: filterParams.currency || undefined,
            financeLocked:
              filterParams.financeLocked === 'LOCKED'
                ? true
                : filterParams.financeLocked === 'UNLOCKED'
                  ? false
                  : undefined,
            expenseDateFrom,
            expenseDateTo,
            tagIds: tagFilterIds?.length ? tagFilterIds : undefined,
            organizationId,
          });
          setSummary(response.summary);
          const page = unwrapPage(response);
          return {
            ...toTableRequest(response),
            total: page.total,
            summary: response.summary,
          };
        }}
      />

      {/* 批量转账单工作台 */}
      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={selectedFeeIds}
        initialOrganizationId={selectedBillOrganizationId}
        initialOrganizationName={
          organizationOptions.find(
            (item) => item.id === selectedBillOrganizationId,
          )?.name
        }
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

      {/* 行配色经列设置浮层「更多设置」进入，配色即时生效并持久化 */}
      <BusinessTagModal
        open={tagModalOpen}
        targetCount={tagFeeIds.length}
        existingTags={tagExisting}
        // 跨组织财务标签写入不支持在此快捷新建，避免新标签误归当前工作区。
        canQuickCreate={false}
        loadOptions={(params) => {
          if (!organizationId) {
            return Promise.resolve({ tags: [], total: '0' });
          }
          return settlementServiceListFinanceFeeTagAssignmentOptions({
            ...params,
            organizationId,
          });
        }}
        onSubmit={async (mode, tagIds) => {
          if (!organizationId) {
            message.warning('请先选择所属公司后再维护费用标签');
            return;
          }
          if (mode === 'assign') {
            await settlementServiceBatchAssignFinanceFeeTags({
              feeIds: tagFeeIds,
              tagIds,
              organizationId,
            });
          } else {
            await settlementServiceBatchRemoveFinanceFeeTags({
              feeIds: tagFeeIds,
              tagIds,
              organizationId,
            });
          }
          actionRef.current?.reload();
        }}
        onCancel={() => setTagModalOpen(false)}
      />
    </>
  );
}
