import type { ActionType } from '@ant-design/pro-components';
import { history, useAccess } from '@umijs/max';
import { App, Card, Select, Space } from 'antd';
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
  TableColumnConfigModal,
} from '@/components/ui';
import {
  FeeLedgerFinancialProgress,
  FinanceOrganizationPurpose,
  OrderFeeStatus,
} from '@/enums.generated';
import BillCreationWorkbench, {
  type BillCreationMode,
} from '@/pages/finance/bills/components/BillCreationWorkbench';
import { orderFeeServiceConfirmFee } from '@/services/roncin/orderFeeService';
import {
  settlementServiceBatchAssignFinanceFeeTags,
  settlementServiceBatchRemoveFinanceFeeTags,
  settlementServiceGetFeeLedgerPreference,
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
  buildUserOrderedColumns,
  financialProgressLabels,
  getBaseFeeLedgerColumns,
} from './components/feeLedgerColumns';

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

export default function FinanceFeeLedgerPage() {
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [billWorkbenchMode, setBillWorkbenchMode] =
    useState<BillCreationMode>('NORMAL');
  const [selectedFeeIds, setSelectedFeeIds] = useState<string[]>([]);
  const [selectedBillOrganizationId, setSelectedBillOrganizationId] =
    useState<string>();
  const [columnConfigOpen, setColumnConfigOpen] = useState(false);
  const [preference, setPreference] = useState<API.FeeLedgerPreference>();
  const [filterParams, setFilterParams] = useState<FeeLedgerFilterParams>({});
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);

  useEffect(() => {
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose: FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_FEE_READ,
    })
      .then((response) => {
        if (!cancelled) setOrganizationOptions(response.data ?? []);
      })
      .catch(() => {
        if (!cancelled) message.warning('所属公司候选加载失败');
      });
    return () => {
      cancelled = true;
    };
  }, [message]);

  const handleSearch = (values: FeeLedgerFilterParams) => {
    setFilterParams(values);
    actionRef.current?.reload();
  };

  const handleReset = () => {
    setFilterParams({});
    actionRef.current?.reload();
  };

  const canCreateBill = (row: API.FeeLedgerItem) =>
    row.status === OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED && !row.billNo;

  const confirmDraftRows = async (
    _keys: React.Key[],
    rows: API.FeeLedgerItem[],
  ) => {
    const draftRows = rows.filter(
      (row) =>
        row.status === OrderFeeStatus.ORDER_FEE_STATUS_DRAFT &&
        row.orderId &&
        row.id &&
        row.version,
    );
    if (draftRows.length === 0) {
      message.info('当前勾选中没有可确认的草稿费用');
      return;
    }
    const results = await Promise.allSettled(
      draftRows.map((row) =>
        orderFeeServiceConfirmFee(
          { orderId: row.orderId as string, id: row.id as string },
          {
            orderId: row.orderId as string,
            id: row.id as string,
            expectedVersion: row.version as string,
          },
          { skipErrorHandler: true },
        ),
      ),
    );
    const succeeded = results.filter(
      (item) => item.status === 'fulfilled',
    ).length;
    const failed = results.length - succeeded;
    if (succeeded > 0) {
      message.success(
        `已确认 ${succeeded} 笔费用${failed > 0 ? `，${failed} 笔失败` : ''}`,
      );
      actionRef.current?.reload();
    } else {
      message.error('费用确认失败，请检查费用版本或权限');
    }
  };

  const handleBatchConfirm = (keys: React.Key[], rows: API.FeeLedgerItem[]) => {
    const organizations = [
      ...new Set(rows.map((row) => row.organizationName || '-')),
    ];
    modal.confirm({
      title: '确认勾选费用？',
      content: `所属公司：${organizations.join('、')}。费用确认仍按订单费用权限校验。`,
      onOk: () => confirmDraftRows(keys, rows),
    });
  };

  // 加载当前用户云端表头偏好配置
  useEffect(() => {
    settlementServiceGetFeeLedgerPreference({})
      .then((res) => {
        if (res.data) {
          setPreference(res.data);
        }
      })
      .catch(() => {
        message.warning('费用表格偏好加载失败，当前未应用个人配置');
      });
  }, [message]);

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
  const [tagOptions, setTagOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [tagOptionsLoading, setTagOptionsLoading] = useState(false);
  const [tagFilterIds, setTagFilterIds] = useState<string[]>();
  const tagFilterRequestRef = useRef(0);

  const loadTagFilterOptions = useCallback(
    async (keyword?: string, selectedIds: string[] = []) => {
      if (!organizationId) {
        setTagOptions([]);
        return;
      }
      const requestSequence = ++tagFilterRequestRef.current;
      setTagOptionsLoading(true);
      try {
        const response = await settlementServiceListFinanceFeeTagOptions({
          page: 1,
          pageSize: 50,
          keyword: keyword?.trim() || undefined,
          organizationId,
        });
        if (requestSequence !== tagFilterRequestRef.current) return;
        setTagOptions((current) => {
          const selected = new Set(selectedIds);
          const options = new Map(
            current
              .filter((option) => selected.has(option.value))
              .map((option) => [option.value, option]),
          );
          for (const tag of response.tags ?? []) {
            if (tag.id) {
              options.set(tag.id, { label: tag.name ?? '', value: tag.id });
            }
          }
          return [...options.values()];
        });
      } finally {
        if (requestSequence === tagFilterRequestRef.current) {
          setTagOptionsLoading(false);
        }
      }
    },
    [organizationId],
  );

  useEffect(() => {
    if (organizationId) void loadTagFilterOptions();
  }, [loadTagFilterOptions]);

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

  // 根据当前用户的个性化列偏好动态过滤显示并按用户拖拽顺序重排
  const columns = useMemo(
    () => buildUserOrderedColumns(baseColumns, preference),
    [baseColumns, preference],
  );

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
      <Card
        size="small"
        style={{
          marginBottom: 12,
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
        }}
        styles={{ body: { padding: '10px 16px' } }}
      >
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
                // 使在切换或清空公司前发出的标签请求立即失效，避免迟到结果回填。
                tagFilterRequestRef.current += 1;
                setTagOptionsLoading(false);
                setOrganizationId(value);
                setTagFilterIds(undefined);
                setTagOptions([]);
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
              showSearch
              filterOption={false}
              loading={tagOptionsLoading}
              style={{ minWidth: 280 }}
              placeholder={
                organizationId ? '命中任一标签即返回' : '请先选择所属公司'
              }
              options={tagOptions}
              value={tagFilterIds}
              disabled={!organizationId}
              onSearch={(keyword) =>
                void loadTagFilterOptions(keyword, tagFilterIds)
              }
              onChange={(value) => {
                setTagFilterIds(value.length ? value : undefined);
                actionRef.current?.reload();
              }}
            />
          </Space>
        </Space>
      </Card>

      <FinanceLedgerTemplate<API.FeeLedgerItem>
        pageTitle="集运费用明细"
        pageSubTitle="全维度多币种费用台账，支持按单据、费用状态、结算单位快速对账与生成账单"
        headerTitle="费用明细台账"
        actionRef={actionRef}
        columns={columns}
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
              '所选费用中包含不可建账的记录，请仅选择已确认且未入账单的费用',
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
          {
            key: 'batch-confirm',
            label: '批量确认勾选费用',
            onClick: handleBatchConfirm,
          },
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
                        '所选费用中包含不可建账的记录，请仅选择已确认且未入账单的费用',
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
          ...(access.canManageFinanceFeeTags && organizationId
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
        onOpenColumnConfig={() => setColumnConfigOpen(true)}
        rowColors={preference?.rowColors}
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
            status: filterParams.status || undefined,
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

      {/* 表头排序与 153 字段/颜色配置弹窗 */}
      <TableColumnConfigModal
        open={columnConfigOpen}
        onClose={() => setColumnConfigOpen(false)}
        currentPreference={preference}
        onSaved={(updated) => {
          setPreference(updated);
          actionRef.current?.reload();
        }}
      />
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
