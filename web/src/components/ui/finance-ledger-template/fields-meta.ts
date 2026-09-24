export interface FinanceFieldMeta {
  id: number;
  key: string;
  name: string;
  defaultVisible: boolean;
  category?: string;
  align?: 'left' | 'center' | 'right';
  width?: number;
}

/**
 * 费用台账可配置字段元数据：与 `pages/finance/fees` 的
 * `getBaseFeeLedgerColumns` 中受管数据列一一对应（key、名称、宽度）。
 * 综合搜索（keyword）、序号（index）与属性（direction）为钉住列，
 * 不参与列设置；新增表格列时必须同步维护本清单，否则设置与表头脱节。
 */
export const FEE_LEDGER_FIELDS: FinanceFieldMeta[] = [
  { id: 1, key: 'tags', name: '标签', defaultVisible: true, width: 140 },
  {
    id: 2,
    key: 'organizationName',
    name: '所属公司',
    defaultVisible: true,
    width: 150,
  },
  { id: 3, key: 'masterNo', name: '主单号', defaultVisible: true, width: 140 },
  {
    id: 4,
    key: 'customerId',
    name: '委托单位',
    defaultVisible: true,
    width: 180,
  },
  {
    id: 5,
    key: 'settlementPartyId',
    name: '结算单位',
    defaultVisible: true,
    width: 180,
  },
  {
    id: 6,
    key: 'businessType',
    name: '业务类型',
    defaultVisible: true,
    width: 95,
  },
  { id: 7, key: 'feeName', name: '费用名称', defaultVisible: true, width: 120 },
  {
    id: 8,
    key: 'currency',
    name: '币种',
    defaultVisible: true,
    align: 'center',
    width: 65,
  },
  {
    id: 9,
    key: 'totalAmount',
    name: '金额',
    defaultVisible: true,
    align: 'right',
    width: 110,
  },
  {
    id: 10,
    key: 'invoiceNo',
    name: '发票号',
    defaultVisible: true,
    width: 130,
  },
  {
    id: 11,
    key: 'financialProgress',
    name: '财务进度',
    defaultVisible: true,
    width: 125,
  },
  {
    id: 12,
    key: 'status',
    name: '费用状态',
    defaultVisible: true,
    width: 90,
  },
  {
    id: 13,
    key: 'exchangeRate',
    name: '汇率',
    defaultVisible: true,
    align: 'right',
    width: 80,
  },
  {
    id: 14,
    key: 'operatorName',
    name: '操作人员',
    defaultVisible: true,
    width: 100,
  },
  {
    id: 15,
    key: 'salesName',
    name: '业务人员',
    defaultVisible: true,
    width: 100,
  },
  { id: 16, key: 'csName', name: '客服人员', defaultVisible: true, width: 100 },
  {
    id: 17,
    key: 'relatedPersonnel',
    name: '关联人员',
    defaultVisible: true,
    width: 100,
  },
  {
    id: 18,
    key: 'taxRate',
    name: '税率(%)',
    defaultVisible: true,
    align: 'right',
    width: 75,
  },
  {
    id: 19,
    key: 'taxAmount',
    name: '税金',
    defaultVisible: true,
    align: 'right',
    width: 90,
  },
  {
    id: 20,
    key: 'netAmount',
    name: '不含税总价',
    defaultVisible: true,
    align: 'right',
    width: 100,
  },
  { id: 21, key: 'houseNo', name: '分单号', defaultVisible: true, width: 130 },
  { id: 22, key: 'billNo', name: '账单编号', defaultVisible: true, width: 155 },
  {
    id: 23,
    key: 'orderNo',
    name: '订单编号',
    defaultVisible: true,
    width: 160,
  },
  {
    id: 24,
    key: 'expenseDate',
    name: '费用时间',
    defaultVisible: true,
    width: 110,
  },
  { id: 25, key: 'soNo', name: 'SO号', defaultVisible: true, width: 130 },
  {
    id: 26,
    key: 'baseCurrencyAmount',
    name: '折本币总价',
    defaultVisible: true,
    align: 'right',
    width: 120,
  },
  {
    id: 27,
    key: 'relatedInfo',
    name: '关联信息',
    defaultVisible: true,
    width: 120,
  },
  {
    id: 28,
    key: 'consignee',
    name: '收货人简称',
    defaultVisible: true,
    width: 110,
  },
  {
    id: 29,
    key: 'shipper',
    name: '发货人简称',
    defaultVisible: true,
    width: 110,
  },
  {
    id: 30,
    key: 'notifyParty',
    name: '通知人简称',
    defaultVisible: true,
    width: 110,
  },
  {
    id: 31,
    key: 'verifiedAmount',
    name: '已核销金额',
    defaultVisible: true,
    align: 'right',
    width: 100,
  },
  {
    id: 32,
    key: 'unverifiedAmount',
    name: '未核销金额',
    defaultVisible: true,
    align: 'right',
    width: 100,
  },
  {
    id: 33,
    key: 'grossWeightKg',
    name: '实际总毛重(KGS)',
    defaultVisible: true,
    align: 'right',
    width: 115,
  },
  {
    id: 34,
    key: 'volumeCbm',
    name: '实际总体积',
    defaultVisible: true,
    align: 'right',
    width: 100,
  },
  { id: 35, key: 'note', name: '备注', defaultVisible: true, width: 120 },
];

export function getDefaultRowColors(): {
  unbilled: string;
  unverifiedUninvoiced: string;
  invoicedUnverified: string;
  verifiedUninvoiced: string;
  completed: string;
  invoicedPartiallyVerified: string;
  partiallyVerifiedUninvoiced: string;
} {
  return {
    unbilled: '#FFF7E6',
    unverifiedUninvoiced: '#FFFBE6',
    invoicedUnverified: '#E6F4FF',
    verifiedUninvoiced: '#F9F0FF',
    completed: '#F6FFED',
    invoicedPartiallyVerified: '#E6F4FF',
    partiallyVerifiedUninvoiced: '#F9F0FF',
  };
}
