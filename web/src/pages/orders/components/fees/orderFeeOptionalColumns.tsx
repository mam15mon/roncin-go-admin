import type { ProColumns } from '@ant-design/pro-components';
import { Space, Tag, Tooltip } from 'antd';
import Decimal from 'decimal.js';
import { BusinessTagList } from '@/components/business-tag/BusinessTagList';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import { feeLedgerProgressLabels } from '@/features/finance/fee-progress';
import { formatAmount } from '@/utils/format';
import type { FeeBillTrackingView } from './feeBillTracking';
import type { FeeColumnPreference } from './feeColumnPreference';

/** 关联账单查询的整体状态文案：加载中/失败不得显示成未建账。 */
function trackingStateCell(tracking: FeeBillTrackingView) {
  if (tracking.state === 'loading') {
    return <span style={{ color: '#8c8c8c' }}>加载中…</span>;
  }
  return <span style={{ color: '#cf1322' }}>加载失败</span>;
}

function renderBillNoCell(record: API.OrderFee, tracking: FeeBillTrackingView) {
  if (tracking.state !== 'ready') return trackingStateCell(tracking);
  const item = record.id ? tracking.byFeeId[record.id] : undefined;
  if (!item?.billNo) {
    return <span style={{ color: '#8c8c8c' }}>未建账</span>;
  }
  return (
    <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
      {item.billNo}
    </span>
  );
}

function renderFinancialProgressCell(
  record: API.OrderFee,
  tracking: FeeBillTrackingView,
) {
  if (tracking.state !== 'ready') return trackingStateCell(tracking);
  const item = record.id ? tracking.byFeeId[record.id] : undefined;
  const progress = item?.financialProgress;
  if (
    progress === undefined ||
    progress ===
      FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNSPECIFIED ||
    progress ===
      FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED
  ) {
    return <Tag color="gold">未建账</Tag>;
  }
  const label = feeLedgerProgressLabels[progress] ?? {
    text: '未知状态',
    color: 'default',
  };
  return (
    <Tag color={label.color} style={{ margin: 0 }}>
      {label.text}
    </Tag>
  );
}

/**
 * 行内编辑实时预览投影：由 OrderFeeTableTabs 维护，结构与 RowEditPreview 对齐。
 * taxRate 为费用项目默认税率（API 口径：百分数值字符串，如 "6.00" 表示 6%）。
 */
export type FeeRowPreview = {
  feeCode?: string;
  taxRate?: string;
  unitPrice?: string;
  amount?: { total: string; currency: string };
  rate?: { status: string; rate?: string };
};

/** 按行 key 索引的编辑预览集合。 */
export type FeeRowPreviewMap = Record<string, FeeRowPreview>;

/** 未保存或缺少计算输入时的占位：服务端尚未计算，前端暂无实时预览。 */
function pendingSaveCell() {
  return <span style={{ color: '#8c8c8c' }}>待保存</span>;
}

/** 税额实时计算：含税总价、税率（百分数值）推不含税总额与税金，两位小数。 */
function computeTaxBreakdown(
  total: string,
  taxRatePercent: string,
): { netAmount: string; taxAmount: string } | undefined {
  const rate = new Decimal(taxRatePercent).div(100);
  if (rate.isNegative()) return undefined;
  const gross = new Decimal(total);
  const net = gross.div(rate.plus(1)).toDecimalPlaces(2, Decimal.ROUND_HALF_UP);
  const tax = gross.minus(net).toDecimalPlaces(2, Decimal.ROUND_HALF_UP);
  return { netAmount: net.toString(), taxAmount: tax.toString() };
}

/**
 * 不含税单价折算：含税单价 ÷ (1 + 税率/100)，两位小数 ROUND_HALF_UP；
 * 单价或税率缺失、非法时返回 undefined（调用方显示占位）。
 */
function computeNetUnitPrice(
  unitPrice: string,
  taxRatePercent: string | undefined | null,
): string | undefined {
  if (
    unitPrice === '' ||
    taxRatePercent === undefined ||
    taxRatePercent === null ||
    taxRatePercent === '' ||
    !Number.isFinite(Number(taxRatePercent)) ||
    !Number.isFinite(Number(unitPrice))
  ) {
    return undefined;
  }
  const rate = new Decimal(taxRatePercent).div(100);
  if (rate.isNegative()) return undefined;
  return new Decimal(unitPrice)
    .div(rate.plus(1))
    .toDecimalPlaces(2, Decimal.ROUND_HALF_UP)
    .toString();
}

/** 等宽字体单元格样式：与税金、不含税总额列一致。 */
function monoAmountCell(value: string) {
  return (
    <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
      {value}
    </span>
  );
}

/** 已保存行的不含税单价数值口径：不含税行直接取单价，含税行按税率反算。 */
function netUnitPriceNumberOf(record: API.OrderFee): number | undefined {
  if (!record.unitPrice || !Number.isFinite(Number(record.unitPrice))) {
    return undefined;
  }
  // protojson 省略零值：wire 上非 true（含 undefined）即历史不含税行。
  if (record.taxInclusive !== true) return Number(record.unitPrice);
  const netUnit = computeNetUnitPrice(record.unitPrice, record.taxRate);
  return netUnit === undefined ? undefined : Number(netUnit);
}

/**
 * 可选只读列：不含税单价、税率、税金、不含税总额、折本币金额与费用标签。
 * 保存后金额与税率为服务端快照；行内编辑进行中（传入该行预览时）按
 * 单价×数量与所选费用项目默认税率实时折算，折本币金额乘实时解析汇率，
 * 不含税单价按预览单价与默认税率反算（编辑始终提交含税口径）。
 * 传入关联账单投影时（即具备财务费用读取权限）追加账单号与关联账单财务
 * 进度列；进度属于整张关联账单，多笔费用共用同一账单时显示相同进度。
 */
export function buildOptionalFeeColumns(
  feeBillTracking?: FeeBillTrackingView,
  rowPreviews?: FeeRowPreviewMap,
): ProColumns<API.OrderFee>[] {
  const previewOf = (record: API.OrderFee): FeeRowPreview | undefined =>
    rowPreviews?.[String(record.id ?? '')];
  const financeColumns: ProColumns<API.OrderFee>[] = feeBillTracking
    ? [
        {
          title: '账单号',
          dataIndex: 'billNo',
          width: 150,
          editable: false,
          render: (_, record) => renderBillNoCell(record, feeBillTracking),
        },
        {
          title: (
            <Tooltip title="整张关联账单的开票与核销综合进度；同一账单下多笔费用显示相同进度">
              <span>关联账单财务进度</span>
            </Tooltip>
          ),
          dataIndex: 'financialProgress',
          width: 150,
          editable: false,
          render: (_, record) =>
            renderFinancialProgressCell(record, feeBillTracking),
        },
      ]
    : [];

  return [
    {
      title: '不含税单价',
      dataIndex: 'netUnitPrice',
      width: 110,
      align: 'right',
      editable: false,
      shouldCellUpdate: () => true,
      sorter: (a, b) => {
        const valueA = netUnitPriceNumberOf(a);
        const valueB = netUnitPriceNumberOf(b);
        if (valueA === undefined && valueB === undefined) return 0;
        if (valueA === undefined) return 1;
        if (valueB === undefined) return -1;
        return valueA - valueB;
      },
      render: (_, record) => {
        const preview = previewOf(record);
        if (preview?.unitPrice && preview.taxRate) {
          const netUnit = computeNetUnitPrice(
            preview.unitPrice,
            preview.taxRate,
          );
          if (netUnit !== undefined)
            return monoAmountCell(formatAmount(netUnit));
        }
        if (!record.version) return pendingSaveCell();
        // 历史不含税行单价即不含税口径，直接展示；protojson 省略零值，
        // wire 上非 true（含 undefined）即不含税行，禁止 === false 死分支。
        if (record.taxInclusive !== true) {
          const unit = record.unitPrice
            ? new Decimal(record.unitPrice)
                .toDecimalPlaces(2, Decimal.ROUND_HALF_UP)
                .toString()
            : undefined;
          return unit === undefined ? '-' : monoAmountCell(formatAmount(unit));
        }
        const netUnit = record.unitPrice
          ? computeNetUnitPrice(record.unitPrice, record.taxRate)
          : undefined;
        return netUnit === undefined
          ? '-'
          : monoAmountCell(formatAmount(netUnit));
      },
    },
    {
      title: '税率(%)',
      dataIndex: 'taxRate',
      width: 100,
      align: 'right',
      editable: false,
      shouldCellUpdate: () => true,
      render: (_, record) => {
        const previewTaxRate = previewOf(record)?.taxRate;
        const taxRate = previewTaxRate ?? record.taxRate;
        if (!previewTaxRate && !record.version) return pendingSaveCell();
        if (taxRate === undefined || taxRate === null || taxRate === '') {
          return '-';
        }
        return (
          <Space size={4}>
            <span
              style={{ whiteSpace: 'nowrap' }}
            >{`${Number(taxRate)}%`}</span>
            {previewTaxRate ? <Tag color="processing">预览</Tag> : null}
          </Space>
        );
      },
    },
    {
      title: '税金',
      dataIndex: 'taxAmount',
      width: 100,
      align: 'right',
      editable: false,
      shouldCellUpdate: () => true,
      render: (_, record) => {
        const preview = previewOf(record);
        if (preview?.amount && preview.taxRate) {
          const breakdown = computeTaxBreakdown(
            preview.amount.total,
            preview.taxRate,
          );
          if (breakdown) {
            return (
              <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                {formatAmount(breakdown.taxAmount)}
              </span>
            );
          }
        }
        if (!record.version) return pendingSaveCell();
        return (
          <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
            {formatAmount(record.taxAmount)}
          </span>
        );
      },
    },
    {
      title: '不含税总额',
      dataIndex: 'netAmount',
      width: 110,
      align: 'right',
      editable: false,
      shouldCellUpdate: () => true,
      render: (_, record) => {
        const preview = previewOf(record);
        if (preview?.amount && preview.taxRate) {
          const breakdown = computeTaxBreakdown(
            preview.amount.total,
            preview.taxRate,
          );
          if (breakdown) {
            return (
              <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                {formatAmount(breakdown.netAmount)}
              </span>
            );
          }
        }
        if (!record.version) return pendingSaveCell();
        return (
          <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
            {formatAmount(record.netAmount)}
          </span>
        );
      },
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 140,
      align: 'right',
      editable: false,
      shouldCellUpdate: () => true,
      render: (_, record) => {
        const preview = previewOf(record);
        const previewBase =
          preview?.amount && preview.rate?.status === 'resolved'
            ? new Decimal(preview.amount.total)
                .mul(new Decimal(preview.rate.rate ?? '1'))
                .toDecimalPlaces(2, Decimal.ROUND_HALF_UP)
                .toString()
            : undefined;
        const amount = formatAmount(previewBase ?? record.baseCurrencyAmount);
        if (amount === '-') {
          return !record.version && !previewBase ? pendingSaveCell() : '-';
        }
        return (
          <Space size={4}>
            <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
              {amount}
            </span>
            <Tag>{record.baseCurrency || 'CNY'}</Tag>
          </Space>
        );
      },
    },
    {
      title: '费用标签',
      dataIndex: 'tags',
      width: 140,
      editable: false,
      render: (_, record) => <BusinessTagList tags={record.tags} />,
    },
    ...financeColumns,
  ];
}

/** 提取列稳定 key：优先 dataIndex，操作列（valueType=option）固定为 option。 */
function columnKeyOf(column: ProColumns<API.OrderFee>): string | undefined {
  if (column.valueType === 'option') return 'option';
  const dataIndex = column.dataIndex;
  if (typeof dataIndex === 'string') return dataIndex;
  if (Array.isArray(dataIndex) && typeof dataIndex[0] === 'string') {
    return dataIndex[0];
  }
  return column.key ? String(column.key) : undefined;
}

/** 按列偏好重排并过滤列；偏好未覆盖或外部注入的列保持原顺序追加在末尾。 */
export function orderColumnsByPreference(
  columns: ProColumns<API.OrderFee>[],
  preference: FeeColumnPreference,
): ProColumns<API.OrderFee>[] {
  const byKey = new Map<string, ProColumns<API.OrderFee>>();
  for (const column of columns) {
    const key = columnKeyOf(column);
    if (key && !byKey.has(key)) byKey.set(key, column);
  }
  const hidden = new Set<string>(preference.hidden);
  const ordered: ProColumns<API.OrderFee>[] = [];
  const placed = new Set<string>();
  for (const key of preference.order) {
    const column = byKey.get(key);
    if (column && !hidden.has(key)) {
      ordered.push(column);
      placed.add(key);
    }
  }
  for (const column of columns) {
    const key = columnKeyOf(column);
    if (key && !placed.has(key) && !hidden.has(key)) {
      ordered.push(column);
      placed.add(key);
    }
  }
  return ordered;
}
