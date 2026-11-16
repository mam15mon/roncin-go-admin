import { formatAmount, formatDate, trimDecimal } from '@/utils/format';
import { hasPdfGlyph } from '../../documents/pdf/text';
import { AWB_PDF_TEMPLATE_REVISION, type AwbPdfModel } from './types';

export function awbDecimal(value?: string): string {
  return value ? trimDecimal(value) : '-';
}

export function awbAmount(value?: string): string {
  return formatAmount(value || undefined);
}

export function awbDeclared(value?: API.AirDeclaredValue): string {
  if (!value) return '-';
  return value.kind === 'AMOUNT' ? awbAmount(value.amount) : value.kind || '-';
}

export function awbNumber(model: AwbPdfModel): string {
  return model.target === 'MAWB' ? model.masterNo : model.houseNo || '';
}

/** 固定区域只保留有界摘要；可分页正文始终输出完整原值。 */
export function awbNumberSummary(model: AwbPdfModel): string {
  const characters = [...awbNumber(model).replace(/\s+/gu, ' ')];
  return `${characters.slice(0, 32).join('')}${characters.length > 32 ? '…' : ''}`;
}

export function validateAwbPdf(model: AwbPdfModel): string | null {
  if (
    !model.documentId ||
    !model.version ||
    !model.contentHash ||
    !model.organizationId
  )
    return '缺少已保存单证标识或版本，请重新读取后输出。';
  if (!awbNumber(model).trim()) return '缺少航空运单号码，无法输出。';
  if (!model.issuerDisplayName.trim())
    return '缺少真实签发主体资料，请补齐后输出。';
  if (
    model.outputKind === 'SELF_ISSUED' &&
    (model.target !== 'HAWB' || model.businessType !== 'AE')
  )
    return '此航空单证不适用本公司签发模板。';
  const entries: [string, unknown][] = [
    ['主单号码', model.masterNo],
    ['分单号码', model.houseNo],
    ['签发主体', model.issuerDisplayName],
    ['保存版本', model.version],
    ['正文', model.content],
  ];
  function inspect(label: string, value: unknown): string | null {
    if (typeof value === 'string') {
      const unsupported = [
        ...new Set([...value].filter((char) => !hasPdfGlyph(char))),
      ];
      if (unsupported.length)
        return `${label}含暂不支持打印的字符「${unsupported.join('')}」，请调整后重试。`;
    } else if (Array.isArray(value)) {
      for (const [index, child] of value.entries()) {
        const error = inspect(`${label}第 ${index + 1} 行`, child);
        if (error) return error;
      }
    } else if (value && typeof value === 'object') {
      for (const [key, child] of Object.entries(value)) {
        const error = inspect(
          `${label} · ${FIELD_LABELS[key] ?? '印字内容'}`,
          child,
        );
        if (error) return error;
      }
    }
    return null;
  }
  for (const [label, value] of entries) {
    const error = inspect(label, value);
    if (error) return error;
  }
  return null;
}

const FIELD_LABELS: Record<string, string> = {
  shipperAccount: '发货人账号',
  shipperText: '发货人',
  consigneeAccount: '收货人账号',
  consigneeText: '收货人',
  notifyPartyText: '通知人',
  secondNotifyPartyText: '第二通知人',
  agentText: '代理',
  agentIataCode: '代理 IATA 代码',
  agentAccount: '代理账号',
  issuedBy: '印发人',
  departureText: '起运机场',
  routeTo: '第一程至',
  firstCarrier: '第一承运人',
  routeTo2: '第二程至',
  carrier2: '第二承运人',
  routeTo3: '第三程至',
  carrier3: '第三承运人',
  destinationText: '目的机场',
  flightText: '航班',
  flightDate: '航班日期',
  accountingInformation: '账务信息',
  referenceNumber: '参考编号',
  information1: '附加信息一',
  information2: '附加信息二',
  sci: '特别海关信息',
  handlingInformation: '处理信息',
  marks: '唛头',
  goodsDescription: '品名',
  clauses: '附加条款',
  transportTerms: '运输条款',
  otherChargesText: '其他费用说明',
  shipperCertification: '发货人声明',
  shipperSignature: '发货人签字',
  carrierExecution: '承运人签发声明',
  issuePlace: '签发地点',
  carrierSignature: '承运人签字',
  remarks: '单证备注',
  rateLines: '运价明细',
  otherCharges: '其他费用',
  rateClass: '运价等级',
  commodityItemNo: '商品项目号',
  name: '费用名称',
  billingUnit: '计费单位',
};

export function awbPdfFingerprint(model: AwbPdfModel): string {
  return `${AWB_PDF_TEMPLATE_REVISION}:${JSON.stringify(model)}`;
}

export function awbPdfFileName(model: AwbPdfModel, now: Date): string {
  const number = [...awbNumber(model)]
    .map((char) =>
      char.charCodeAt(0) < 32 || '\\/:*?"<>|'.includes(char) ? '_' : char,
    )
    .join('')
    .slice(0, 100);
  const time = formatDate(now, 'datetime').replaceAll(':', '-');
  return `${model.target}-${number}-${model.outputKind === 'EXTERNAL_ARCHIVE' ? 'archive' : 'waybill'}-${time}.pdf`;
}
