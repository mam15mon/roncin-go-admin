import type { ProColumns } from '@ant-design/pro-components';
import { Space, Tag, Tooltip } from 'antd';
import { BusinessTagList } from '@/components/business-tag/BusinessTagList';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import { feeLedgerProgressLabels } from '@/features/finance/fee-progress';
import { formatAmount } from '@/utils/format';
import type { FeeBillTrackingView } from './feeBillTracking';
import type { FeeColumnPreference } from './feeColumnPreference';
import { FEE_CANCELLED, feeStatusCode } from './feeConstants';

/** 未保存新行的派生金额提示：服务端尚未计算，前端不猜算。 */
function pendingSaveCell() {
  return <span style={{ color: '#8c8c8c' }}>待保存</span>;
}

/** 关联账单查询的整体状态文案：加载中/失败不得显示成未建账。 */
function trackingStateCell(tracking: FeeBillTrackingView) {
  if (tracking.state === 'loading') {
    return <span style={{ color: '#8c8c8c' }}>加载中…</span>;
  }
  return <span style={{ color: '#cf1322' }}>加载失败</span>;
}

function renderBillNoCell(record: API.OrderFee, tracking: FeeBillTrackingView) {
  if (tracking.state !== 'ready') return trackingStateCell(tracking);
  if (feeStatusCode(record.status) === FEE_CANCELLED) {
    return <span style={{ color: '#8c8c8c' }}>已作废</span>;
  }
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
  if (feeStatusCode(record.status) === FEE_CANCELLED) {
    return <span style={{ color: '#8c8c8c' }}>已作废</span>;
  }
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
 * 可选只读列：税率、税金、不含税总额、折本币金额与费用标签。
 * 金额与税率均为保存后服务端快照，前端不重算；未保存新行显示“待保存”。
 * 传入关联账单投影时（即具备财务费用读取权限）追加账单号与关联账单财务
 * 进度列；进度属于整张关联账单，多笔费用共用同一账单时显示相同进度。
 */
export function buildOptionalFeeColumns(
  feeBillTracking?: FeeBillTrackingView,
): ProColumns<API.OrderFee>[] {
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
      title: '税率(%)',
      dataIndex: 'taxRate',
      width: 100,
      align: 'right',
      editable: false,
      render: (_, record) => {
        if (!record.version) return pendingSaveCell();
        if (
          record.taxRate === undefined ||
          record.taxRate === null ||
          record.taxRate === ''
        ) {
          return '-';
        }
        return (
          <Space size={4}>
            <span
              style={{ whiteSpace: 'nowrap' }}
            >{`${Number(record.taxRate)}%`}</span>
            {record.taxInclusive === false && (
              <Tooltip title="该行按不含税单价口径录入，单价不含税金">
                <Tag>未税单价</Tag>
              </Tooltip>
            )}
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
      render: (_, record) =>
        record.version ? (
          <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
            {formatAmount(record.taxAmount)}
          </span>
        ) : (
          pendingSaveCell()
        ),
    },
    {
      title: '不含税总额',
      dataIndex: 'netAmount',
      width: 110,
      align: 'right',
      editable: false,
      render: (_, record) =>
        record.version ? (
          <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
            {formatAmount(record.netAmount)}
          </span>
        ) : (
          pendingSaveCell()
        ),
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 140,
      align: 'right',
      editable: false,
      render: (_, record) => {
        if (!record.version) return pendingSaveCell();
        const amount = formatAmount(record.baseCurrencyAmount);
        if (amount === '-') return '-';
        return (
          <Space size={4}>
            <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
              {amount}
            </span>
            <Tag>{record.baseCurrency || '-'}</Tag>
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
