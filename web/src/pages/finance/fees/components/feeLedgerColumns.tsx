import type { ProColumns } from '@ant-design/pro-components';
import { Tag } from 'antd';
import { BusinessTagList } from '@/components/business-tag/BusinessTagList';
import type { RowColorsConfig } from '@/components/ui/finance-ledger-template';
import {
  businessTypeMeta,
  normalizeBusinessType,
  orderFeeStatusMeta,
  statusTag,
  statusText,
} from '@/constants/statusMeta';
import { FeeLedgerFinancialProgress, OrderFeeStatus } from '@/enums.generated';
import { feeLedgerProgressLabels } from '@/features/finance/fee-progress';
import { searchPartnerOptions } from '@/features/partners';
import { history } from '@/router/history';
import { formatAmount, trimDecimal } from '@/utils/format';

const feeLedgerBusinessTypeValueEnum = Object.fromEntries(
  ['SE', 'SI', 'AE', 'AI', 'LAND', 'RAIL'].map((code) => [
    code,
    { text: statusText(businessTypeMeta, normalizeBusinessType(code), code) },
  ]),
);

const feeLedgerStatusValueEnum = Object.fromEntries(
  [
    OrderFeeStatus.ORDER_FEE_STATUS_UNBILLED,
    OrderFeeStatus.ORDER_FEE_STATUS_BILLED,
    OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED,
  ].map((status) => [
    status,
    {
      text: statusText(orderFeeStatusMeta, status, String(status)),
    },
  ]),
);

/** 行配色 key 与财务进度枚举一一对应；文案与颜色以共享映射为唯一真相源。 */
const progressRowColorKeys: Record<number, keyof RowColorsConfig> = {
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED]:
    'unbilled',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED]:
    'unverifiedUninvoiced',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_UNVERIFIED]:
    'invoicedUnverified',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED]:
    'invoicedPartiallyVerified',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_PARTIALLY_VERIFIED_UNINVOICED]:
    'partiallyVerifiedUninvoiced',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_VERIFIED_UNINVOICED]:
    'verifiedUninvoiced',
  [FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED]:
    'completed',
};

export const financialProgressLabels: Record<
  number,
  { text: string; color: string; key: keyof RowColorsConfig }
> = Object.fromEntries(
  Object.entries(feeLedgerProgressLabels).map(([progress, label]) => [
    Number(progress),
    { ...label, key: progressRowColorKeys[Number(progress)] },
  ]),
);

export function amount(value?: string | number) {
  return Number(value || 0);
}

/** 汇率展示统一 trimDecimal 口径：去尾零变长小数（业务口径 ≤4 位）。 */
export function formatRate(value?: string | number | null): string {
  return trimDecimal(value);
}

export function getBaseFeeLedgerColumns(): ProColumns<API.FeeLedgerItem>[] {
  return [
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
      title: '标签',
      dataIndex: 'tags',
      width: 140,
      search: false,
      render: (_, row) => <BusinessTagList tags={row.tags} />,
    },
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 150,
      search: false,
      renderText: (value) => value || '-',
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
      request: ({ keyWords }) => searchPartnerOptions(keyWords, { role: 1 }),
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
      request: ({ keyWords }) => searchPartnerOptions(keyWords),
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
      valueEnum: feeLedgerBusinessTypeValueEnum,
      render: (_, row) =>
        statusText(
          businessTypeMeta,
          normalizeBusinessType(row.businessType),
          row.businessType || '-',
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
          {formatAmount(row.totalAmount)}
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
        const progress =
          row.financialProgress ??
          FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED;
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
      valueEnum: feeLedgerStatusValueEnum,
      render: (_, row) =>
        statusTag(
          orderFeeStatusMeta,
          row.status ?? OrderFeeStatus.ORDER_FEE_STATUS_UNSPECIFIED,
          row.status == null ? '-' : String(row.status),
        ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 80,
      align: 'right',
      search: false,
      render: (_, row) => (
        <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
          {formatRate(row.exchangeRate)}
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
          {formatAmount(val as string | number | undefined)}
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
          {formatAmount(val as string | number | undefined)}
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
          {formatAmount(row.baseCurrencyAmount)}
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

/** 台账本地列配置：完整顺序（含隐藏列）+ 隐藏 key 集合。 */
export interface FeeLedgerColumnSetting {
  order: string[];
  hidden: string[];
}

/** 台账钉住列：综合搜索、序号与属性固定在表头前部，不参与列设置。 */
function isPinnedLedgerColumn(col: ProColumns<API.FeeLedgerItem>): boolean {
  return (
    col.dataIndex === 'keyword' ||
    col.valueType === 'index' ||
    col.dataIndex === 'direction'
  );
}

export function buildUserOrderedColumns(
  baseColumns: ProColumns<API.FeeLedgerItem>[],
  setting?: FeeLedgerColumnSetting,
): ProColumns<API.FeeLedgerItem>[] {
  if (!setting || setting.order.length === 0) {
    return baseColumns;
  }

  const managed = new Map<string, ProColumns<API.FeeLedgerItem>>();
  const pinned: ProColumns<API.FeeLedgerItem>[] = [];
  baseColumns.forEach((col) => {
    const key = String(col.dataIndex || col.key || '');
    if (isPinnedLedgerColumn(col)) {
      pinned.push(col);
      return;
    }
    if (key && !managed.has(key)) managed.set(key, col);
  });

  const hidden = new Set(setting.hidden);
  const orderedKeys = setting.order.filter((key) => managed.has(key));
  for (const key of managed.keys()) {
    if (!orderedKeys.includes(key)) orderedKeys.push(key);
  }

  const result: ProColumns<API.FeeLedgerItem>[] = [...pinned];
  for (const key of orderedKeys) {
    const column = managed.get(key);
    if (column && !hidden.has(key)) result.push(column);
  }
  return result;
}
