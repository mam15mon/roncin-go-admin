import type { ProColumns } from '@ant-design/pro-components';
import { Space, Tag } from 'antd';
import {
  normalizeOrderFeeStatus,
  orderFeeStatusMeta,
  statusTag,
} from '@/constants/statusMeta';
import { trimDecimal } from '@/utils/format';
import {
  PAYABLE,
  RECEIVABLE,
  feeDirectionCode,
} from './feeConstants';

type FeeBaseColumnsOptions =
  | { variant: 'workbench'; direction: number }
  | { variant: 'panel' };

/** 订单费用表格的公共业务列；入口仅保留标签、权限操作等场景差量。 */
export function feeBaseColumns(
  options: FeeBaseColumnsOptions,
): ProColumns<API.OrderFee>[] {
  const panel = options.variant === 'panel';
  const statusColumn: ProColumns<API.OrderFee> = {
    title: '状态',
    dataIndex: 'status',
    width: 90,
    render: (_, record) =>
      statusTag(
        orderFeeStatusMeta,
        normalizeOrderFeeStatus(record.status),
      ),
  };
  const directionColumn: ProColumns<API.OrderFee> = {
    title: '收付方向',
    dataIndex: 'direction',
    width: 90,
    render: (_, record) =>
      feeDirectionCode(record.direction) === PAYABLE ? (
        <Tag color="volcano">应付</Tag>
      ) : (
        <Tag color="green">应收</Tag>
      ),
  };
  const feeCodeColumn: ProColumns<API.OrderFee> = {
    title: '费用代码',
    dataIndex: 'feeCode',
    width: panel ? 130 : 120,
    copyable: true,
    ...(!panel && {
      render: (_: unknown, record: API.OrderFee) => record.feeCode || '-',
    }),
  };
  const feeNameColumn: ProColumns<API.OrderFee> = {
    title: '费用名称',
    dataIndex: 'feeName',
    width: panel ? 150 : 140,
    ...(panel
      ? { ellipsis: true }
      : {
          render: (_: unknown, record: API.OrderFee) =>
            record.feeName || '-',
        }),
  };
  const settlementPartyColumn: ProColumns<API.OrderFee> = {
    title: '结算单位',
    dataIndex: 'settlementPartyName',
    width: panel ? 190 : 180,
    ellipsis: true,
    ...(!panel && {
      render: (_: unknown, record: API.OrderFee) =>
        record.settlementPartyName || '-',
    }),
  };
  const currencyColumn: ProColumns<API.OrderFee> = {
    title: '币种',
    dataIndex: 'currency',
    width: 80,
    render: (_, record) => <Tag color="blue">{record.currency}</Tag>,
  };
  const unitPriceColumn: ProColumns<API.OrderFee> = {
    title: '单价',
    dataIndex: 'unitPrice',
    width: panel ? 130 : 100,
    align: 'right',
    render: (_, record) => trimDecimal(record.unitPrice),
  };
  const quantityColumn: ProColumns<API.OrderFee> = {
    title: '数量',
    dataIndex: 'quantity',
    width: panel ? 110 : 80,
    align: 'right',
    render: (_, record) => trimDecimal(record.quantity),
  };
  const billingUnitColumn: ProColumns<API.OrderFee> = {
    title: '计费单位',
    dataIndex: 'billingUnit',
    width: 90,
    ...(!panel && {
      render: (_: unknown, record: API.OrderFee) =>
        record.billingUnit || '-',
    }),
  };
  const totalAmountColumn: ProColumns<API.OrderFee> = {
    title: '总金额',
    dataIndex: 'totalAmount',
    width: panel ? 150 : 130,
    align: 'right',
    render: (_, record) =>
      panel ? (
        <strong>
          {trimDecimal(record.totalAmount)} {record.currency}
        </strong>
      ) : (
        <span
          style={{
            fontWeight: 600,
            color:
              options.direction === RECEIVABLE ? '#1677ff' : '#fa8c16',
          }}
        >
          {trimDecimal(record.totalAmount)} {record.currency}
        </span>
      ),
  };
  const exchangeRateColumn: ProColumns<API.OrderFee> = {
    title: '汇率',
    dataIndex: 'exchangeRate',
    width: panel ? 160 : 100,
    align: 'right',
    render: (_, record) => (
      <Space size={4}>
        <span>{trimDecimal(record.exchangeRate)}</span>
        {record.exchangeRateSource === 'MANUAL' && (
          <Tag color="gold">手工</Tag>
        )}
        {record.exchangeRateSource === 'SYSTEM' && (
          <Tag color="blue">系统</Tag>
        )}
        {panel && record.exchangeRateSource === 'BASE_CURRENCY' && (
          <Tag>本币</Tag>
        )}
      </Space>
    ),
  };
  const expenseDateColumn: ProColumns<API.OrderFee> = {
    title: panel ? '费用日期' : '发生日期',
    dataIndex: 'expenseDate',
    width: panel ? 110 : 220,
    ...(!panel && {
      render: (_: unknown, record: API.OrderFee) =>
        record.expenseDate || '-',
    }),
  };
  const noteColumn: ProColumns<API.OrderFee> = {
    title: '备注',
    dataIndex: 'note',
    ...(panel && { width: 180 }),
    ellipsis: true,
    render: (_, record) => record.note || '-',
  };

  if (panel) {
    return [
      statusColumn,
      directionColumn,
      feeCodeColumn,
      feeNameColumn,
      settlementPartyColumn,
      billingUnitColumn,
      quantityColumn,
      unitPriceColumn,
      totalAmountColumn,
      exchangeRateColumn,
      expenseDateColumn,
      noteColumn,
    ];
  }

  return [
    statusColumn,
    feeCodeColumn,
    feeNameColumn,
    settlementPartyColumn,
    currencyColumn,
    unitPriceColumn,
    quantityColumn,
    billingUnitColumn,
    totalAmountColumn,
    exchangeRateColumn,
    expenseDateColumn,
    noteColumn,
  ];
}
