import dayjs from 'dayjs';

// 1. 单据类型枚举映射 (支持字符串与数字双向兼容)
export interface DocTypeMeta {
  key: string;
  numValue: number;
  label: string;
  shortLabel: string;
  color: string;
  defaultPrefix: string;
  businessCodes?: string[];
}

export const DOC_TYPES: DocTypeMeta[] = [
  {
    key: 'DOCUMENT_TYPE_ORDER',
    numValue: 1,
    label: '订单编号',
    shortLabel: '订单',
    color: 'blue',
    defaultPrefix: '',
    businessCodes: ['SE', 'SI', 'AE', 'AI'],
  },
  {
    key: 'DOCUMENT_TYPE_BILL',
    numValue: 2,
    label: '账单编号',
    shortLabel: '账单',
    color: 'cyan',
    defaultPrefix: 'BI',
  },
  {
    key: 'DOCUMENT_TYPE_BILL_BATCH',
    numValue: 14,
    label: '账单批次号',
    shortLabel: '批次',
    color: 'geekblue',
    defaultPrefix: 'BG',
  },
  {
    key: 'DOCUMENT_TYPE_INVOICE',
    numValue: 11,
    label: '发票记录号',
    shortLabel: '发票',
    color: 'purple',
    defaultPrefix: '',
  },
  {
    key: 'DOCUMENT_TYPE_RECEIPT_PAYMENT',
    numValue: 5,
    label: '收付款流水号',
    shortLabel: '流水',
    color: 'orange',
    defaultPrefix: 'PR',
  },
  {
    key: 'DOCUMENT_TYPE_WRITE_OFF',
    numValue: 4,
    label: '核销单号',
    shortLabel: '核销',
    color: 'green',
    defaultPrefix: 'WO',
  },
  {
    key: 'DOCUMENT_TYPE_COMMISSION',
    numValue: 13,
    label: '提成结算单号',
    shortLabel: '提成',
    color: 'magenta',
    defaultPrefix: 'CM',
  },
  {
    key: 'DOCUMENT_TYPE_HOUSE_BILL',
    numValue: 9,
    label: '分提单号 (HBL)',
    shortLabel: '分单',
    color: 'volcano',
    defaultPrefix: '',
  },
  {
    key: 'DOCUMENT_TYPE_QUOTATION',
    numValue: 3,
    label: '报价单号',
    shortLabel: '报价',
    color: 'gold',
    defaultPrefix: 'QO',
  },
  {
    key: 'DOCUMENT_TYPE_CONTRACT',
    numValue: 6,
    label: '合同协议编号',
    shortLabel: '合同',
    color: 'lime',
    defaultPrefix: 'CT',
  },
  {
    key: 'DOCUMENT_TYPE_FREIGHT_RATE',
    numValue: 12,
    label: '运价方案号',
    shortLabel: '运价',
    color: 'processing',
    defaultPrefix: 'FR',
  },
  {
    key: 'DOCUMENT_TYPE_INTERNAL_REFERENCE',
    numValue: 7,
    label: '内部参考号',
    shortLabel: '内部',
    color: 'default',
    defaultPrefix: '',
  },
  {
    key: 'DOCUMENT_TYPE_CUSTOMER_REFERENCE',
    numValue: 8,
    label: '客户参考号',
    shortLabel: '客户',
    color: 'default',
    defaultPrefix: '',
  },
];

export const docTypeMap = new Map<string | number, DocTypeMeta>();
DOC_TYPES.forEach((t) => {
  docTypeMap.set(t.key, t);
  docTypeMap.set(t.numValue, t);
  docTypeMap.set(String(t.numValue), t);
  docTypeMap.set(t.key.replace('DOCUMENT_TYPE_', ''), t);
});

export function filterVisibleNumberRules(
  rules: API.NumberRule[],
): API.NumberRule[] {
  return rules.filter((rule) => docTypeMap.has(rule.documentType as any));
}

// 2. 日期格式枚举映射 (支持字符串与数字双向兼容)
export const DATE_FORMATS: Record<
  string | number,
  { label: string; formatStr: string; numValue: number }
> = {
  DATE_FORMAT_YYYYMMDD: {
    label: '年月日 (yyyyMMdd)',
    formatStr: 'YYYYMMDD',
    numValue: 1,
  },
  1: { label: '年月日 (yyyyMMdd)', formatStr: 'YYYYMMDD', numValue: 1 },
  DATE_FORMAT_YYYYMM: {
    label: '年月 (yyyyMM)',
    formatStr: 'YYYYMM',
    numValue: 2,
  },
  2: { label: '年月 (yyyyMM)', formatStr: 'YYYYMM', numValue: 2 },
  DATE_FORMAT_YYYY: { label: '年 (yyyy)', formatStr: 'YYYY', numValue: 3 },
  3: { label: '年 (yyyy)', formatStr: 'YYYY', numValue: 3 },
  DATE_FORMAT_NONE: { label: '无日期', formatStr: '', numValue: 4 },
  4: { label: '无日期', formatStr: '', numValue: 4 },
  DATE_FORMAT_UNSPECIFIED: { label: '无日期', formatStr: '', numValue: 4 },
  0: { label: '无日期', formatStr: '', numValue: 4 },
};

// 3. 重置周期枚举映射 (支持字符串与数字双向兼容)
export const RESET_POLICIES: Record<
  string | number,
  { label: string; numValue: number }
> = {
  RESET_POLICY_DAILY: { label: '每日重置', numValue: 1 },
  1: { label: '每日重置', numValue: 1 },
  RESET_POLICY_MONTHLY: { label: '每月重置', numValue: 2 },
  2: { label: '每月重置', numValue: 2 },
  RESET_POLICY_YEARLY: { label: '每年重置', numValue: 3 },
  3: { label: '每年重置', numValue: 3 },
  RESET_POLICY_NEVER: { label: '永不重置', numValue: 4 },
  4: { label: '永不重置', numValue: 4 },
  RESET_POLICY_UNSPECIFIED: { label: '永不重置', numValue: 4 },
  0: { label: '永不重置', numValue: 4 },
};

// 工具函数：解析生成规则示例预览
export function generatePreviewNumber(rule: API.NumberRule): {
  text: string;
  isDynamicType?: boolean;
} {
  const meta = docTypeMap.get(rule.documentType as any);
  const now = dayjs();
  const dateMeta = DATE_FORMATS[rule.dateFormat as any] || DATE_FORMATS[4];
  const dateStr = dateMeta.formatStr ? now.format(dateMeta.formatStr) : '';
  const len = Number(rule.sequenceLength) || 4;
  const seq = '1'.padStart(len, '0');

  // 1. 订单编号：前缀固定为动态业务流向占位 {SE}（若用户配置了额外自定义前缀则拼在前面）
  if (meta?.key === 'DOCUMENT_TYPE_ORDER' || meta?.numValue === 1) {
    const userPrefix = rule.prefix ? `${rule.prefix}-` : '';
    return {
      text: `${userPrefix}{SE}${dateStr}${seq}`,
      isDynamicType: true,
    };
  }

  // 2. 其他单据：严格使用用户当前配置的 prefix；若未配置前缀，则纯无前缀输出
  const prefix = rule.prefix || '';
  return {
    text: `${prefix}${dateStr}${seq}`,
    isDynamicType: false,
  };
}
