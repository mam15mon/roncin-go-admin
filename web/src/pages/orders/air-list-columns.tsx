import type { ProColumns } from '@ant-design/pro-components';
import { Space, Tag } from 'antd';
import type { OrderListItem } from '@/components/ui/order-list-template/types';
import { formatDate, trimDecimal } from '@/utils/format';

/** 航空列表只展示本票确认，两个货量来源各自保留，不替用户选择费用来源。 */
export function AirWeightSummary({
  summaries,
}: {
  summaries: OrderListItem['airWeightSummaries'];
}) {
  if (!summaries?.length) return <span>未确认</span>;
  return (
    <Space orientation="vertical" size={2}>
      {summaries.map((summary) => (
        <Space key={summary.source} size={4} wrap>
          <span>
            {summary.source === 'PLANNED' ? '委托' : '实际'}：
            {trimDecimal(summary.confirmedWeightKg)} KG
          </span>
          {summary.confirmationStale && <Tag color="warning">需重确认</Tag>}
        </Space>
      ))}
    </Space>
  );
}

export const airListColumns: ProColumns<OrderListItem>[] = [
  {
    title: '航空公司',
    dataIndex: 'shippingLineName',
    width: 136,
    ellipsis: true,
    renderText: (value) => value || '-',
  },
  {
    title: 'ETD',
    dataIndex: 'etd',
    width: 144,
    sorter: true,
    renderText: (value) => formatDate(value, 'minute'),
  },
  {
    title: 'ETA',
    dataIndex: 'eta',
    width: 144,
    sorter: true,
    renderText: (value) => formatDate(value, 'minute'),
  },
  {
    title: '本票确认计费重',
    dataIndex: 'airWeightSummaries',
    width: 200,
    render: (_, row) => <AirWeightSummary summaries={row.airWeightSummaries} />,
  },
  {
    title: '主单单证状态',
    dataIndex: 'airMasterStatus',
    width: 116,
    renderText: (value) => value || '-',
  },
  {
    title: '分单单证状态',
    dataIndex: 'airHouseStatus',
    width: 116,
    renderText: (value) => value || '-',
  },
];

export const AIR_VISIBLE_CARGO_COLUMNS = [
  'totalPackages',
  'grossWeightKg',
  'volumeCbm',
];
