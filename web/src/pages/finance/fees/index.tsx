import { history } from '@umijs/max';
import { App, Badge, Button, Tag } from 'antd';
import { FilterOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { ActionType } from '@ant-design/pro-components';
import {
  FinanceLedgerTemplate,
  TableColumnConfigModal,
  type FinanceLedgerMetricCard,
} from '@/components/ui';
import {
  settlementServiceGetFeeLedgerPreference,
  settlementServiceListFeeLedger,
} from '@/services/roncin/settlementService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import {
  AdvancedFilterDrawer,
  DEFAULT_FEE_FILTER_VALUES,
  type AdvancedFeeFilterValues,
} from './components/AdvancedFilterDrawer';

const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};

const financialProgressLabels: Record<
  string,
  { text: string; color: string; key: keyof API.FeeLedgerRowColors }
> = {
  UNBILLED: { text: '账单未建立', color: 'gold', key: 'unbilled' },
  UNVERIFIED_UNINVOICED: {
    text: '未核销未开票',
    color: 'orange',
    key: 'unverifiedUninvoiced',
  },
  INVOICED_UNVERIFIED: {
    text: '已开票未核销',
    color: 'blue',
    key: 'invoicedUnverified',
  },
  INVOICED_PARTIALLY_VERIFIED: {
    text: '已开票部分核销',
    color: 'cyan',
    key: 'invoicedPartiallyVerified',
  },
  PARTIALLY_VERIFIED_UNINVOICED: {
    text: '部分核销未开票',
    color: 'geekblue',
    key: 'partiallyVerifiedUninvoiced',
  },
  VERIFIED_UNINVOICED: {
    text: '已核销未开票',
    color: 'purple',
    key: 'verifiedUninvoiced',
  },
  COMPLETED: { text: '已完成', color: 'green', key: 'completed' },
};

function amount(value?: string | number) {
  return Number(value || 0);
}

function formatMoney(value?: any, decimals = 2): string {
  if (value === undefined || value === null || value === '') return '-';
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return num.toLocaleString('zh-CN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

function formatRate(value?: any): string {
  if (value === undefined || value === null || value === '') return '-';
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return Number(num.toFixed(4)).toString();
}

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [selectedFeeIds, setSelectedFeeIds] = useState<string[]>([]);
  const [columnConfigOpen, setColumnConfigOpen] = useState(false);
  const [preference, setPreference] = useState<API.FeeLedgerPreference>();
  const [advancedFilterOpen, setAdvancedFilterOpen] = useState(false);
  const [advancedFilters, setAdvancedFilters] =
    useState<AdvancedFeeFilterValues>(DEFAULT_FEE_FILTER_VALUES);

  // 计算当前生效的高级筛选条件数量
  const activeAdvancedFilterCount = useMemo(() => {
    let count = 0;
    if (advancedFilters.direction) count++;
    if (advancedFilters.financialProgress) count++;
    if (advancedFilters.settlementPartyId) count++;
    if (advancedFilters.customerId) count++;
    if (advancedFilters.businessType) count++;
    if (advancedFilters.currency) count++;
    if (advancedFilters.orderOrMasterNo) count++;
    if (advancedFilters.feeName) count++;
    if (advancedFilters.invoiceNo) count++;
    if (advancedFilters.salesName) count++;
    if (advancedFilters.csName) count++;
    if (advancedFilters.operatorName) count++;
    if (advancedFilters.isReconciled) count++;
    if (advancedFilters.isLocked) count++;
    if (advancedFilters.vesselName) count++;
    if (advancedFilters.voyageNo) count++;
    if (advancedFilters.contractNo) count++;
    if (advancedFilters.feeCategory && advancedFilters.feeCategory !== 'ALL') count++;
    if (advancedFilters.serviceType) count++;
    if (advancedFilters.invoiceDateRange) count++;
    if (advancedFilters.verificationDateRange) count++;
    if (advancedFilters.orderCreatedAtRange) count++;
    if (advancedFilters.billCreatedAtRange) count++;
    if (advancedFilters.etdRange) count++;
    if (advancedFilters.etaRange) count++;
    return count;
  }, [advancedFilters]);

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

  // 34 项核心业务字段预置列定义
  const baseColumns: ProColumns<API.FeeLedgerItem>[] = [
    // --- 0. 全局综合关键字搜索（表格内隐藏，固定在搜索栏首位） ---
    {
      title: '综合搜索',
      dataIndex: 'keyword',
      hideInTable: true,
      order: 100,
      fieldProps: {
        placeholder: '输入订单号/费用名/往来单位搜索',
      },
    },

    // 1. 序号与属性（固定左侧）
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 50,
      fixed: 'left',
      search: false,
    },
    {
      title: '属性',
      dataIndex: 'direction',
      width: 65,
      fixed: 'left',
      valueType: 'select',
      order: 90,
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
      render: (_, row) => (
        <Tag
          color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}
          style={{ margin: 0 }}
        >
          {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },

    // 2. 主单号、委托单位、结算单位、业务类型
    {
      title: '主单号',
      dataIndex: 'masterNo',
      width: 140,
      order: 68,
      fieldProps: {
        placeholder: '输入主提单号',
      },
      render: (val) => val || '-',
    },
    {
      title: '委托单位',
      dataIndex: 'customerId',
      width: 180,
      valueType: 'select',
      order: 75,
      request: async ({ keyWords }) => {
        const response = await partnerServiceListPartners({
          role: 1,
          page: 1,
          pageSize: 200,
          keyword: keyWords,
        });
        return (response.data || []).map((item) => ({
          label: item.legalName || item.code || item.id,
          value: item.id,
        }));
      },
      fieldProps: {
        showSearch: true,
        filterOption: false,
        placeholder: '输入名称/全拼/首字母搜索',
      },
      render: (_, row) => row.customerName || '-',
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyId',
      width: 180,
      ellipsis: true,
      valueType: 'select',
      order: 80,
      request: async ({ keyWords }) => {
        const response = await partnerServiceListPartners({
          page: 1,
          pageSize: 200,
          keyword: keyWords,
        });
        return (response.data || []).map((item) => ({
          label: item.legalName || item.code || item.id,
          value: item.id,
        }));
      },
      fieldProps: {
        showSearch: true,
        filterOption: false,
        placeholder: '输入名称/全拼/首字母搜索',
      },
      render: (_, row) => row.settlementPartyName || '-',
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 95,
      valueType: 'select',
      order: 50,
      valueEnum: Object.fromEntries(
        Object.entries(businessLabels).map(([key, text]) => [key, { text }]),
      ),
    },

    // 3. 费用名称、币种、金额、发票号、费用状态、汇率
    {
      title: '费用名称',
      dataIndex: 'feeName',
      width: 120,
      ellipsis: true,
      order: 60,
      fieldProps: {
        placeholder: '输入费用科目 (如海运费)',
      },
      render: (val) => <span style={{ fontWeight: 500 }}>{val}</span>,
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 65,
      align: 'center',
      valueType: 'select',
      order: 45,
      valueEnum: {
        CNY: { text: 'CNY' },
        USD: { text: 'USD' },
        EUR: { text: 'EUR' },
        HKD: { text: 'HKD' },
      },
      render: (val) => <Tag style={{ margin: 0 }}>{val}</Tag>,
    },
    {
      title: '金额',
      dataIndex: 'totalAmount',
      width: 110,
      align: 'right',
      search: false,
      render: (_, row) => (
        <span
          style={{
            fontWeight: 600,
            color: '#262626',
            fontFamily: 'monospace',
            whiteSpace: 'nowrap',
          }}
        >
          {formatMoney(row.totalAmount)}
        </span>
      ),
    },
    {
      title: '发票号',
      dataIndex: 'invoiceNo',
      width: 130,
      order: 25,
      fieldProps: {
        placeholder: '输入发票号',
      },
      render: (val) => val || '-',
    },
    {
      title: '费用状态',
      dataIndex: 'financialProgress',
      width: 120,
      valueType: 'select',
      order: 85,
      valueEnum: Object.fromEntries(
        Object.entries(financialProgressLabels).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const progress = row.financialProgress || 'UNBILLED';
        const item = financialProgressLabels[progress] || {
          text: progress,
          color: 'default',
        };
        return (
          <Tag color={item.color} style={{ margin: 0 }}>
            {item.text}
          </Tag>
        );
      },
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 80,
      align: 'right',
      search: false,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
          {formatRate(val)}
        </span>
      ),
    },

    // 4. 操作人员、业务人员、客服人员、关联人员
    {
      title: '操作人员',
      dataIndex: 'operatorName',
      width: 100,
      ellipsis: true,
      order: 30,
      fieldProps: {
        placeholder: '输入操作员姓名',
      },
      render: (val) => val || '-',
    },
    {
      title: '业务人员',
      dataIndex: 'salesName',
      width: 100,
      ellipsis: true,
      order: 35,
      fieldProps: {
        placeholder: '输入业务员姓名',
      },
      render: (val) => val || '-',
    },
    {
      title: '客服人员',
      dataIndex: 'csName',
      width: 100,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },
    {
      title: '关联人员',
      dataIndex: 'relatedPersonnel',
      width: 100,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },

    // 5. 税率(%)、税金、不含税总价
    {
      title: '税率(%)',
      dataIndex: 'taxRate',
      width: 75,
      align: 'right',
      search: false,
      render: (val) => (
        <span style={{ whiteSpace: 'nowrap' }}>
          {val ? `${Number(val)}%` : '-'}
        </span>
      ),
    },
    {
      title: '税金',
      dataIndex: 'taxAmount',
      width: 90,
      align: 'right',
      search: false,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
          {formatMoney(val)}
        </span>
      ),
    },
    {
      title: '不含税总价',
      dataIndex: 'netAmount',
      width: 100,
      align: 'right',
      search: false,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
          {formatMoney(val)}
        </span>
      ),
    },

    // 6. 分单号、账单编号、订单编号、费用时间、SO号、折本币总价、关联信息
    {
      title: '分单号',
      dataIndex: 'houseNo',
      width: 130,
      order: 66,
      fieldProps: {
        placeholder: '输入分提单号',
      },
      render: (val) => val || '-',
    },
    {
      title: '账单编号',
      dataIndex: 'billNo',
      width: 155,
      order: 64,
      fieldProps: {
        placeholder: '输入账单编号',
      },
      render: (val) => val || '-',
    },
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      width: 160,
      order: 70,
      fieldProps: {
        placeholder: '输入订单编号',
      },
      copyable: true,
      render: (_, row) => (
        <a
          style={{ fontWeight: 500 }}
          onClick={() => history.push(`/finance/fees/detail/${row.orderId}`)}
        >
          {row.orderNo}
        </a>
      ),
    },
    {
      title: '费用时间',
      dataIndex: 'expenseDate',
      width: 110,
      valueType: 'dateRange',
      order: 40,
      search: {
        transform: (value) => ({
          expenseDateFrom: value[0],
          expenseDateTo: value[1],
        }),
      },
    },
    {
      title: 'SO号',
      dataIndex: 'soNo',
      width: 130,
      search: false,
      render: (val) => val || '-',
    },
    {
      title: '折本币总价',
      dataIndex: 'baseCurrencyAmount',
      width: 120,
      align: 'right',
      search: false,
      render: (_, row) => (
        <span
          style={{
            fontWeight: 600,
            fontFamily: 'monospace',
            whiteSpace: 'nowrap',
            color: row.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
          }}
        >
          {formatMoney(row.baseCurrencyAmount)}
        </span>
      ),
    },
    {
      title: '关联信息',
      dataIndex: 'relatedInfo',
      width: 120,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },

    // 7. 收货人简称、发货人简称、通知人简称、已核销金额、未核销金额
    {
      title: '收货人简称',
      dataIndex: 'consignee',
      width: 110,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },
    {
      title: '发货人简称',
      dataIndex: 'shipper',
      width: 110,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },
    {
      title: '通知人简称',
      dataIndex: 'notifyParty',
      width: 110,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },
    {
      title: '已核销金额',
      dataIndex: 'verifiedAmount',
      width: 100,
      align: 'right',
      search: false,
      render: () => <span style={{ color: '#8c8c8c' }}>-</span>,
    },
    {
      title: '未核销金额',
      dataIndex: 'unverifiedAmount',
      width: 100,
      align: 'right',
      search: false,
      render: () => <span style={{ color: '#8c8c8c' }}>-</span>,
    },

    // 8. 实际总毛重(KGS)、实际总体积、备注
    {
      title: '实际总毛重(KGS)',
      dataIndex: 'grossWeightKg',
      width: 115,
      align: 'right',
      search: false,
    },
    {
      title: '实际总体积',
      dataIndex: 'volumeCbm',
      width: 100,
      align: 'right',
      search: false,
    },
    {
      title: '备注',
      dataIndex: 'note',
      width: 120,
      ellipsis: true,
      search: false,
      render: (val) => val || '-',
    },
  ];

  // 根据当前用户的个性化列偏好动态过滤显示并按用户拖拽顺序重排
  const columns = useMemo(() => {
    if (!preference?.columns || preference.columns.length === 0) {
      return baseColumns;
    }
    const normalizeKey = (k: string) => {
      if (k === 'status') return 'financialProgress';
      if (k === 'customerName') return 'customerId';
      return k;
    };

    const orderMap = new Map<string, { visible: boolean; order: number }>();
    preference.columns.forEach((c, idx) => {
      if (c.fieldKey) {
        const normKey = normalizeKey(c.fieldKey);
        orderMap.set(normKey, {
          visible: Boolean(c.visible),
          order: idx + 1,
        });
      }
    });

    const colMap = new Map<string, ProColumns<API.FeeLedgerItem>>();
    let indexCol: ProColumns<API.FeeLedgerItem> | undefined;
    let directionCol: ProColumns<API.FeeLedgerItem> | undefined;

    baseColumns.forEach((col) => {
      if (col.valueType === 'index') {
        indexCol = col;
      } else if (col.dataIndex === 'direction') {
        directionCol = col;
      } else {
        const key = String(col.dataIndex || '');
        if (key) colMap.set(key, col);
      }
    });

    // 1. 全局搜索项、序号与属性列保持稳定
    const result: ProColumns<API.FeeLedgerItem>[] = [];
    const keywordCol = baseColumns.find((c) => c.dataIndex === 'keyword');
    if (keywordCol) result.push(keywordCol);
    if (indexCol) result.push(indexCol);
    if (directionCol) result.push(directionCol);

    // 2. 其余可见列按照用户设置的 order 顺序插入
    const userOrdered: ProColumns<API.FeeLedgerItem>[] = [];
    preference.columns.forEach((c) => {
      if (c.fieldKey) {
        const normKey = normalizeKey(c.fieldKey);
        if (orderMap.get(normKey)?.visible) {
          const matched = colMap.get(normKey);
          if (matched && !userOrdered.includes(matched)) {
            userOrdered.push(matched);
          }
        }
      }
    });

    result.push(...userOrdered);

    // 3. 补充可能在 baseColumns 中存在但在 preference 中未定义的列（如有）
    baseColumns.forEach((col) => {
      if (col !== indexCol && col !== directionCol) {
        const key = String(col.dataIndex || '');
        if (key && !orderMap.has(key) && !result.includes(col)) {
          result.push(col);
        }
      }
    });

    return result;
  }, [baseColumns, preference]);

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
        scrollX={3200}
        primaryActionText="创建账单"
        primaryActionRequiresSelection
        onPrimaryAction={(keys) => {
          setSelectedFeeIds(keys.map(String));
          setBillWorkbenchOpen(true);
        }}
        batchActions={[
          {
            key: 'batch-confirm',
            label: '批量确认勾选费用',
            onClick: (keys) => {
              message.info(
                `已选 ${keys.length} 笔费用，可直接点击【创建账单】自动原子确认并生成批次`,
              );
            },
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
        extraToolBarActions={[
          <Button
            key="advanced-filter-btn"
            icon={<FilterOutlined />}
            onClick={() => setAdvancedFilterOpen(true)}
          >
            高级筛选 (33项)
            {activeAdvancedFilterCount > 0 && (
              <Badge
                count={activeAdvancedFilterCount}
                size="small"
                style={{ marginLeft: 6, backgroundColor: '#1677ff' }}
              />
            )}
          </Button>,
        ]}
        request={async (params) => {
          const expenseDateFrom = advancedFilters.expenseDateRange?.[0]
            ? advancedFilters.expenseDateRange[0].format('YYYY-MM-DD')
            : params.expenseDateFrom;
          const expenseDateTo = advancedFilters.expenseDateRange?.[1]
            ? advancedFilters.expenseDateRange[1].format('YYYY-MM-DD')
            : params.expenseDateTo;

          const response = await settlementServiceListFeeLedger({
            page: params.current,
            pageSize: params.pageSize,
            keyword:
              advancedFilters.orderOrMasterNo ||
              advancedFilters.feeName ||
              params.keyword ||
              params.orderNo ||
              params.masterNo ||
              params.houseNo ||
              params.billNo ||
              params.feeName ||
              params.operatorName ||
              params.salesName ||
              params.invoiceNo ||
              undefined,
            businessType:
              advancedFilters.businessType || params.businessType || undefined,
            direction:
              advancedFilters.direction || params.direction || undefined,
            status:
              advancedFilters.financialProgress ||
              params.financialProgress ||
              params.status ||
              undefined,
            customerId:
              advancedFilters.customerId || params.customerId || undefined,
            settlementPartyId:
              advancedFilters.settlementPartyId ||
              params.settlementPartyId ||
              undefined,
            currency: advancedFilters.currency || params.currency || undefined,
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

      {/* 高级 33 项 6 组全维业务筛选抽屉 */}
      <AdvancedFilterDrawer
        open={advancedFilterOpen}
        onClose={() => setAdvancedFilterOpen(false)}
        initialValues={advancedFilters}
        onApply={(values) => {
          setAdvancedFilters(values);
          actionRef.current?.reload();
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
