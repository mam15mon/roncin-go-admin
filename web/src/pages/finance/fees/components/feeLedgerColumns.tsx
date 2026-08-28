import type { ProColumns } from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { Tag } from 'antd';
import React from 'react';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

export const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};

export const feeStatusLabels: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  BILLED: { text: '已进账单', color: 'blue' },
  CANCELLED: { text: '已作废', color: 'default' },
};

export const financialProgressLabels: Record<
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

export function amount(value?: string | number) {
  return Number(value || 0);
}

export function formatMoney(value?: any, decimals = 2): string {
  if (value === undefined || value === null || value === '') return '-';
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return num.toLocaleString('zh-CN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

export function formatRate(value?: any): string {
  if (value === undefined || value === null || value === '') return '-';
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return Number(num.toFixed(4)).toString();
}

export function getBaseFeeLedgerColumns(): ProColumns<API.FeeLedgerItem>[] {
  return   [
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
      title: '财务进度',
      dataIndex: 'financialProgress',
      width: 125,
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
      title: '费用状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      order: 82,
      valueEnum: {
        DRAFT: { text: '草稿' },
        CONFIRMED: { text: '已确认' },
        BILLED: { text: '已开账' },
        CANCELLED: { text: '已作废' },
      },
      render: (_, row) => {
        const statusMap: Record<string, { text: string; color: string }> = {
          DRAFT: { text: '草稿', color: 'default' },
          CONFIRMED: { text: '已确认', color: 'blue' },
          BILLED: { text: '已开账', color: 'green' },
          CANCELLED: { text: '已作废', color: 'error' },
        };
        const item = statusMap[row.status || 'DRAFT'] || {
          text: row.status || '-',
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
      search: false,
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
}

export function buildUserOrderedColumns(
  baseColumns: ProColumns<API.FeeLedgerItem>[],
  preference?: API.FeeLedgerPreference,
): ProColumns<API.FeeLedgerItem>[] {
  if (!preference?.columns || preference.columns.length === 0) {
    return baseColumns;
  }
  const normalizeKey = (k: string) => {
    if (k === 'financial_progress') return 'financialProgress';
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
}
