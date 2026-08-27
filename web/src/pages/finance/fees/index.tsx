import { history } from '@umijs/max';
import { App, Tag } from 'antd';
import type { ProColumns } from '@ant-design/pro-components';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { ActionType } from '@ant-design/pro-components';
import {
  EllipsisTooltip,
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

const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};
const statusLabels: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  BILLED: { text: '已进账单', color: 'blue' },
  CANCELLED: { text: '已作废', color: 'default' },
};

function amount(value?: string | number) {
  return Number(value || 0);
}

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [selectedFeeIds, setSelectedFeeIds] = useState<string[]>([]);
  const [columnConfigOpen, setColumnConfigOpen] = useState(false);
  const [preference, setPreference] = useState<API.FeeLedgerPreference>();

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

  // 全量预置列定义
  const baseColumns: ProColumns<API.FeeLedgerItem>[] = [
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '属性',
      dataIndex: 'direction',
      width: 75,
      fixed: 'left',
      valueType: 'select',
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
    {
      title: '状态',
      dataIndex: 'status',
      width: 85,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusLabels).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const value = statusLabels[row.status || 'DRAFT'];
        return (
          <Tag color={value.color} style={{ margin: 0 }}>
            {value.text}
          </Tag>
        );
      },
    },
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      width: 155,
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
      title: '业务类型',
      dataIndex: 'businessType',
      width: 95,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(businessLabels).map(([key, text]) => [key, { text }]),
      ),
    },
    {
      title: '费用科目',
      dataIndex: 'feeName',
      width: 130,
      search: false,
      render: (val) => <span style={{ fontWeight: 500 }}>{val}</span>,
    },
    {
      title: '委托单位',
      dataIndex: 'customerId',
      width: 190,
      valueType: 'select',
      request: async () => {
        const response = await partnerServiceListPartners({
          role: 1,
          page: 1,
          pageSize: 200,
        });
        return (response.data || []).map((item) => ({
          label: item.legalName || item.code || item.id,
          value: item.id,
        }));
      },
      fieldProps: { showSearch: true, optionFilterProp: 'label' },
      render: (_, row) => (
        <EllipsisTooltip maxWidth={180}>
          {row.customerName || '-'}
        </EllipsisTooltip>
      ),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 180,
      ellipsis: true,
      search: false,
      render: (val) => (
        <EllipsisTooltip maxWidth={170}>{val || '-'}</EllipsisTooltip>
      ),
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 70,
      align: 'center',
      search: false,
      render: (val) => <Tag style={{ margin: 0 }}>{val}</Tag>,
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 90,
      align: 'right',
      search: false,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 75,
      align: 'right',
      search: false,
    },
    {
      title: '单位',
      dataIndex: 'billingUnit',
      width: 70,
      align: 'center',
      search: false,
    },
    {
      title: '原币金额',
      dataIndex: 'totalAmount',
      width: 125,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong style={{ color: '#262626' }}>
          {row.totalAmount} {row.currency}
        </strong>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 85,
      align: 'right',
      search: false,
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 135,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong
          style={{
            color: row.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
          }}
        >
          {row.baseCurrencyAmount} {row.baseCurrency}
        </strong>
      ),
    },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 75,
      align: 'right',
      search: false,
      render: (val) => (val ? `${Number(val) * 100}%` : '-'),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 95,
      align: 'right',
      search: false,
    },
    {
      title: '不含税金额',
      dataIndex: 'netAmount',
      width: 110,
      align: 'right',
      search: false,
    },
    {
      title: '费用日期',
      dataIndex: 'expenseDate',
      width: 110,
      valueType: 'dateRange',
      search: {
        transform: (value) => ({
          expenseDateFrom: value[0],
          expenseDateTo: value[1],
        }),
      },
    },
    {
      title: '备注说明',
      dataIndex: 'note',
      width: 130,
      ellipsis: true,
      search: false,
      render: (val) =>
        val ? (
          <EllipsisTooltip maxWidth={120}>{val}</EllipsisTooltip>
        ) : (
          '-'
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 125,
      fixed: 'right',
      render: (_, row) => [
        <a
          key="view-detail"
          onClick={() => {
            if (row.orderId) {
              history.push(`/finance/fees/detail/${row.orderId}`);
            }
          }}
        >
          费用详情
        </a>,
        <a
          key="to-bill"
          onClick={() => {
            if (row.id) {
              setSelectedFeeIds([row.id]);
              setBillWorkbenchOpen(true);
            }
          }}
        >
          转账单
        </a>,
      ],
    },
  ];

  // 根据当前用户的个性化列偏好动态过滤显示
  const columns = useMemo(() => {
    if (!preference?.columns || preference.columns.length === 0) {
      return baseColumns;
    }
    const visibleMap = new Map<string, boolean>();
    preference.columns.forEach((c) => {
      if (c.fieldKey) visibleMap.set(c.fieldKey, Boolean(c.visible));
    });

    return baseColumns.filter((col) => {
      // 序号与操作列常驻显示
      if (col.valueType === 'index' || col.valueType === 'option') return true;
      const key = String(col.dataIndex || '');
      if (!key) return true;
      if (visibleMap.has(key)) {
        return visibleMap.get(key);
      }
      return true;
    });
  }, [baseColumns, preference]);

  // 根据费用业务状态映射对应的行高亮背景 key
  const getRowStatusColorKey = (row: API.FeeLedgerItem) => {
    if (row.status === 'DRAFT' || !row.status || row.status === 'CANCELLED') {
      return 'unbilled' as const;
    }
    if (row.status === 'CONFIRMED') {
      return 'unverifiedUninvoiced' as const;
    }
    if (row.status === 'BILLED') {
      return 'invoicedUnverified' as const;
    }
    return undefined;
  };

  return (
    <>
      <FinanceLedgerTemplate<API.FeeLedgerItem>
        headerTitle="集运费用明细台账"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={2300}
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
        request={async (params) => {
          const response = await settlementServiceListFeeLedger({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            businessType: params.businessType,
            direction: params.direction,
            status: params.status,
            customerId: params.customerId,
            expenseDateFrom: params.expenseDateFrom,
            expenseDateTo: params.expenseDateTo,
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
