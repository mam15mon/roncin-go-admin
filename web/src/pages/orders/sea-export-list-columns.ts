import type { ProColumns } from '@ant-design/pro-components';
import type { OrderListItem } from '@/components/ui/order-list-template/types';
import { formatAmount, formatDate } from '@/utils/format';

const timeColumn = (
  title: string,
  dataIndex: keyof OrderListItem,
): ProColumns<OrderListItem> => ({
  title,
  dataIndex,
  width: 144,
  renderText: (value) => formatDate(value, 'minute'),
});

const textColumn = (
  title: string,
  dataIndex: keyof OrderListItem,
): ProColumns<OrderListItem> => ({
  title,
  dataIndex,
  width: 120,
  ellipsis: true,
  renderText: (value) => value || '-',
});

/** 海运领域专属展示列；只消费已映射的稳定行模型。 */
export const seaExportListColumns: ProColumns<OrderListItem>[] = [
  textColumn('船公司', 'shippingLineName'),
  { ...timeColumn('ETD', 'etd'), sorter: true },
  { ...timeColumn('ETA', 'eta'), sorter: true },
  timeColumn('截 SI 时间', 'siCutoff'),
  timeColumn('截文件时间', 'docCutoff'),
  timeColumn('截关时间', 'customsCutoff'),
  timeColumn('截 VGM 时间', 'vgmCutoff'),
  timeColumn('截申报时间', 'declarationCutoffAt'),
  textColumn('订舱编号', 'bookingNo'),
  textColumn('共同段卸货港', 'dischargePortName'),
  textColumn('中转港', 'transitPortName'),
  textColumn('国外代理', 'foreignAgentName'),
  textColumn('船代', 'shippingAgentName'),
  textColumn('企业内部编号', 'internalReferenceNo'),
  {
    ...timeColumn('业务日期', 'orderDate'),
    renderText: (value) => formatDate(value, 'date'),
  },
  timeColumn('接单时间', 'receivedAt'),
  timeColumn('货好时间', 'cargoReadyAt'),
  textColumn('工厂', 'factoryName'),
  textColumn('品名', 'goodsDescription'),
  {
    title: '货值',
    dataIndex: 'cargoValue',
    width: 120,
    align: 'right',
    renderText: (value) => formatAmount(value),
  },
  textColumn('货值币种', 'cargoCurrency'),
  textColumn('UN NO.', 'unNumber'),
  textColumn('危险品类别', 'hazardClass'),
];

/** 初始仅展开最常用的预计离港时间，其余信息按需勾选。 */
export const SEA_EXPORT_HIDDEN_COLUMNS = seaExportListColumns
  .map((column) => String(column.dataIndex))
  .filter((key) => key !== 'etd');
