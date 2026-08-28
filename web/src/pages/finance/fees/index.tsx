import type { ActionType } from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { App } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  FinanceLedgerTemplate,
  TableColumnConfigModal,
  type FinanceLedgerMetricCard,
} from '@/components/ui';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import { orderFeeServiceConfirmFee } from '@/services/roncin/orderFeeService';
import {
  settlementServiceGetFeeLedgerPreference,
  settlementServiceListFeeLedger,
} from '@/services/roncin/settlementService';
import {
  FeeLedgerSearchFilter,
  type FeeLedgerFilterParams,
} from './components/FeeLedgerSearchFilter';
import {
  amount,
  buildUserOrderedColumns,
  financialProgressLabels,
  getBaseFeeLedgerColumns,
} from './components/feeLedgerColumns';

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [selectedFeeIds, setSelectedFeeIds] = useState<string[]>([]);
  const [columnConfigOpen, setColumnConfigOpen] = useState(false);
  const [preference, setPreference] = useState<API.FeeLedgerPreference>();
  const [filterParams, setFilterParams] = useState<FeeLedgerFilterParams>({});

  const handleSearch = (values: FeeLedgerFilterParams) => {
    setFilterParams(values);
    actionRef.current?.reload();
  };

  const handleReset = () => {
    setFilterParams({});
    actionRef.current?.reload();
  };

  const canCreateBill = (row: API.FeeLedgerItem) =>
    row.status === 'CONFIRMED' && !row.billNo;

  const handleBatchConfirm = async (
    _keys: React.Key[],
    rows: API.FeeLedgerItem[],
  ) => {
    const draftRows = rows.filter(
      (row) =>
        row.status === 'DRAFT' && row.orderId && row.id && row.version,
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

  // 加载当前用户云端表头偏好配置
  useEffect(() => {
    settlementServiceGetFeeLedgerPreference({})
      .then((res) => {
        if (res.data) {
          setPreference(res.data);
        }
      })
      .catch(() => {});
  }, []);

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
      value: amount(summary?.receivableBaseAmount),
      precision: 2,
      suffix: summary?.baseCurrency || 'CNY',
      valueColor: '#1677ff',
    },
    {
      key: 'payable-base',
      title: '应付折本币总池',
      value: amount(summary?.payableBaseAmount),
      precision: 2,
      suffix: summary?.baseCurrency || 'CNY',
      valueColor: '#fa8c16',
    },
    {
      key: 'profit-base',
      title: '确认综合毛利',
      value: amount(summary?.profitBaseAmount),
      precision: 2,
      suffix: summary?.baseCurrency || 'CNY',
      valueColor:
        amount(summary?.profitBaseAmount) >= 0 ? '#52c41a' : '#ff4d4f',
    },
  ];

  const baseColumns = useMemo(() => getBaseFeeLedgerColumns(), []);

  // 根据当前用户的个性化列偏好动态过滤显示并按用户拖拽顺序重排
  const columns = useMemo(
    () => buildUserOrderedColumns(baseColumns, preference),
    [baseColumns, preference],
  );

  // 根据费用财务进度映射对应的行高亮背景 key（支持 7 状态）
  const getRowStatusColorKey = (row: API.FeeLedgerItem) => {
    const progress = row.financialProgress || 'UNBILLED';
    const matched = financialProgressLabels[progress];
    return matched?.key;
  };

  return (
    <>
      <FinanceLedgerTemplate<API.FeeLedgerItem>
        headerTitle="集运费用明细台账"
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
        primaryActionText="创建账单"
        primaryActionRequiresSelection
        rowSelection={{
          getCheckboxProps: (record) => {
            const blocked =
              Boolean(record.billNo) ||
              record.status === 'BILLED' ||
              record.status === 'CANCELLED';
            return {
              disabled: blocked,
              title: blocked
                ? record.billNo
                  ? `已进入账单 ${record.billNo}`
                  : record.status === 'BILLED'
                    ? '已开账费用不可再次操作'
                    : '已作废费用不可操作'
                : undefined,
            };
          },
        }}
        onPrimaryAction={(keys, rows) => {
          const invalidRows = rows.filter((row) => !canCreateBill(row));
          if (invalidRows.length > 0) {
            message.warning(
              '所选费用中包含不可建账的记录，请仅选择已确认且未入账单的费用',
            );
            return;
          }
          setSelectedFeeIds(keys.map(String));
          setBillWorkbenchOpen(true);
        }}
        batchActions={[
          {
            key: 'batch-confirm',
            label: '批量确认勾选费用',
            onClick: handleBatchConfirm,
          },
        ]}
        onImport={() => message.info('可通过 Excel 模板批量导入费用明细')}
        onOpenColumnConfig={() => setColumnConfigOpen(true)}
        rowColors={preference?.rowColors}
        getRowStatusColorKey={getRowStatusColorKey}
        onRowClick={(row) => {
          if (row.orderId) {
            history.push(`/finance/fees/detail/${row.orderId}`);
          }
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
          });
          setSummary(response.summary);
          return {
            data: response.data || [],
            total: Number(response.total || 0),
            success: response.success ?? true,
            summary: response.summary,
          };
        }}
      />

      {/* 批量转账单工作台 */}
      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={selectedFeeIds}
        sourceLabel={`从费用明细勾选的 ${selectedFeeIds.length} 笔费用`}
        onClose={() => setBillWorkbenchOpen(false)}
        onCreated={() => {
          setBillWorkbenchOpen(false);
          setSelectedFeeIds([]);
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
    </>
  );
}
