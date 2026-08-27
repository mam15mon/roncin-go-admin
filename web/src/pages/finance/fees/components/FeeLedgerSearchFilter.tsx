import React from 'react';
import {
  SearchFilterTemplate,
  type SearchFilterFieldItem,
} from '@/components/ui';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

export interface FeeLedgerFilterParams {
  keyword?: string;
  direction?: string;
  financialProgress?: string;
  status?: string;
  settlementPartyId?: string;
  customerId?: string;
  billNo?: string;
  expenseDateRange?: [any, any];

  // 单据与往来
  orderNo?: string;
  masterNo?: string;
  houseNo?: string;
  feeName?: string;
  invoiceNo?: string;
  consignee?: string;
  shipper?: string;

  // 航次与人员
  businessType?: string;
  currency?: string;
  etdRange?: [any, any];
  etaRange?: [any, any];
  salesName?: string;
  operatorName?: string;
  csName?: string;
  vesselName?: string;
  voyageNo?: string;

  // 账期与审计节点
  invoiceDateRange?: [any, any];
  verificationDateRange?: [any, any];
  orderCreatedAtRange?: [any, any];
  billCreatedAtRange?: [any, any];

  // 合约风控与标签
  isReconciled?: string;
  isLocked?: string;
  contractNo?: string;
  feeCategory?: string;
  serviceType?: string;
  feeTags?: string;
  billTags?: string;
}

export interface FeeLedgerSearchFilterProps {
  onSearch: (values: FeeLedgerFilterParams) => void;
  onReset: () => void;
  loading?: boolean;
}

export const FeeLedgerSearchFilter: React.FC<FeeLedgerSearchFilterProps> = ({
  onSearch,
  onReset,
  loading = false,
}) => {
  const items: SearchFilterFieldItem[] = [
    // --- 默认首屏展示的前 5 项核心字段 (高密度单行) ---
    {
      name: 'keyword',
      label: '综合搜索',
      placeholder: '输入订单/主单/单位/账单号',
    },
    {
      name: 'direction',
      label: '费用属性',
      type: 'select',
      placeholder: '全部属性',
      options: [
        { label: '应收 (RECEIVABLE)', value: 'RECEIVABLE' },
        { label: '应付 (PAYABLE)', value: 'PAYABLE' },
      ],
    },
    {
      name: 'financialProgress',
      label: '财务进度',
      type: 'select',
      placeholder: '全部财务进度',
      options: [
        { label: '账单未建立', value: 'UNBILLED' },
        { label: '未核销未开票', value: 'UNVERIFIED_UNINVOICED' },
        { label: '已开票未核销', value: 'INVOICED_UNVERIFIED' },
        { label: '已开票部分核销', value: 'INVOICED_PARTIALLY_VERIFIED' },
        { label: '部分核销未开票', value: 'PARTIALLY_VERIFIED_UNINVOICED' },
        { label: '已核销未开票', value: 'VERIFIED_UNINVOICED' },
        { label: '已完成', value: 'COMPLETED' },
      ],
    },
    {
      name: 'status',
      label: '费用状态',
      type: 'select',
      placeholder: '全部状态',
      options: [
        { label: '草稿 (DRAFT)', value: 'DRAFT' },
        { label: '已确认 (CONFIRMED)', value: 'CONFIRMED' },
        { label: '已开账 (BILLED)', value: 'BILLED' },
        { label: '已作废 (CANCELLED)', value: 'CANCELLED' },
      ],
    },
    {
      name: 'settlementPartyId',
      label: '结算单位',
      type: 'searchable-select',
      placeholder: '输入名称/全拼搜索结算单位',
      request: async ({ keyWords }) => {
        const res = await partnerServiceListPartners({
          page: 1,
          pageSize: 200,
          keyword: keyWords,
        });
        return (res.data || []).map((p) => ({
          label: p.legalName || p.code || p.id || '',
          value: p.id || '',
        }));
      },
    },

    // --- 展开后展示的其余 33 项全维高密度业务字段 (一行 6 列) ---
    {
      name: 'expenseDateRange',
      label: '费用时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'customerId',
      label: '委托单位',
      type: 'searchable-select',
      placeholder: '输入名称/全拼搜索委托单位',
      request: async ({ keyWords }) => {
        const res = await partnerServiceListPartners({
          role: 1,
          page: 1,
          pageSize: 200,
          keyword: keyWords,
        });
        return (res.data || []).map((p) => ({
          label: p.legalName || p.code || p.id || '',
          value: p.id || '',
        }));
      },
    },
    {
      name: 'billNo',
      label: '账单编号',
      placeholder: '输入账单编号',
    },
    {
      name: 'orderNo',
      label: '订单编号',
      placeholder: '输入订单编号',
    },
    {
      name: 'masterNo',
      label: '主提单号',
      placeholder: '海运MBL / 空运AWB',
    },
    {
      name: 'houseNo',
      label: '分提单号',
      placeholder: '输入分单号 HBL',
    },
    {
      name: 'feeName',
      label: '费用科目',
      placeholder: '如海运费/报关费/港杂费',
    },
    {
      name: 'invoiceNo',
      label: '发票号码',
      placeholder: '输入发票号码',
    },
    {
      name: 'businessType',
      label: '业务类型',
      type: 'select',
      placeholder: '全部业务类型',
      options: [
        { label: '海运出口 (SE)', value: 'SE' },
        { label: '海运进口 (SI)', value: 'SI' },
        { label: '空运出口 (AE)', value: 'AE' },
        { label: '空运进口 (AI)', value: 'AI' },
        { label: '铁运出口 (RE)', value: 'RE' },
        { label: '散货拼箱 (LCL)', value: 'LCL' },
        { label: '其他业务', value: 'OTHER' },
      ],
    },
    {
      name: 'currency',
      label: '计价币种',
      type: 'select',
      placeholder: '全部币种',
      options: [
        { label: 'CNY - 人民币', value: 'CNY' },
        { label: 'USD - 美元', value: 'USD' },
        { label: 'EUR - 欧元', value: 'EUR' },
        { label: 'HKD - 港币', value: 'HKD' },
      ],
    },
    {
      name: 'etdRange',
      label: '离港时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'etaRange',
      label: '到港时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'salesName',
      label: '业务人员',
      placeholder: '输入业务员姓名',
    },
    {
      name: 'operatorName',
      label: '操作人员',
      placeholder: '输入操作员姓名',
    },
    {
      name: 'csName',
      label: '客服人员',
      placeholder: '输入客服姓名',
    },
    {
      name: 'vesselName',
      label: '船名',
      placeholder: '输入船名',
    },
    {
      name: 'voyageNo',
      label: '航次',
      placeholder: '输入航次编号',
    },
    {
      name: 'consignee',
      label: '收货人',
      placeholder: '输入收货人抬头',
    },
    {
      name: 'shipper',
      label: '发货人',
      placeholder: '输入发货人抬头',
    },
    {
      name: 'invoiceDateRange',
      label: '开票时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'verificationDateRange',
      label: '核销时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'orderCreatedAtRange',
      label: '接单时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'billCreatedAtRange',
      label: '开账时间',
      type: 'date-range',
      placeholder: ['开始时间', '结束时间'],
    },
    {
      name: 'isReconciled',
      label: '是否对账',
      type: 'select',
      placeholder: '全部',
      options: [
        { label: '已对账', value: 'YES' },
        { label: '未对账', value: 'NO' },
      ],
    },
    {
      name: 'isLocked',
      label: '财务锁单',
      type: 'select',
      placeholder: '全部',
      options: [
        { label: '已锁单', value: 'YES' },
        { label: '未锁单', value: 'NO' },
      ],
    },
    {
      name: 'contractNo',
      label: '合约协议',
      placeholder: '输入合约协议号',
    },
    {
      name: 'feeTags',
      label: '费用标签',
      placeholder: '输入费用标签',
    },
    {
      name: 'billTags',
      label: '账单标签',
      placeholder: '输入账单标签',
    },
  ];

  return (
    <SearchFilterTemplate
      layout="grid"
      formLayout="horizontal"
      labelWidth={75}
      colSpan={4}
      collapsible={true}
      defaultCollapsed={true}
      defaultVisibleCount={5}
      items={items}
      onSearch={onSearch}
      onReset={onReset}
      loading={loading}
    />
  );
};

export default FeeLedgerSearchFilter;
