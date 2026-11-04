import { formatDate, trimDecimal } from '@/utils/format';
import fontCoverage from './fontCoverage.json';
import fontGlyphWidths from './fontGlyphWidths.json';
import type { HblPdfModel, HblPdfVariant } from './types';

const CONTENT_LABELS: Partial<Record<keyof API.SeaBillContent, string>> = {
  shipperText: '发货人',
  consigneeText: '收货人',
  notifyPartyText: '通知人',
  secondNotifyPartyText: '第二通知人',
  marksText: '唛头',
  goodsDescriptionText: '货物描述',
  packageUnit: '包装单位',
  freightTerms: '运费条款',
  transportTerms: '运输条款',
  billForm: '提单形式',
  releaseType: '放单方式',
  clauses: '特别条款',
  foreignAgentText: '国外代理',
};

const PRINT_LABELS: Partial<
  Record<keyof API.SeaHouseBillPrintDetails, string>
> = {
  headerContactText: '页眉联系方式',
  bookingNoText: '订舱号',
  exportReferencesText: '出口参考号',
  footerReferenceText: '页脚参考号',
  preCarriageText: '前程运输',
  placeOfReceiptText: '收货地',
  domesticRoutingText: '境内运输及出口指示',
  pointOfOriginText: '原产地',
  onwardRoutingText: '后续内陆运输',
  vesselVoyageText: '船名航次',
  portOfLoadingText: '装货港',
  portOfDischargeText: '卸货港',
  placeOfDeliveryText: '交货地',
  containersText: '箱号与封号',
  shippedOnBoardDate: '装船日期',
  issuePlaceText: '签发地',
  issueDate: '签发日期',
  deliveryContactText: '交货联系信息',
};

function hasGlyph(character: string): boolean {
  if (/^[\n\r\t]$/.test(character)) return true;
  const point = character.codePointAt(0);
  if (point === undefined) return true;
  let start = 0;
  let end = fontCoverage.length - 1;
  while (start <= end) {
    const middle = Math.floor((start + end) / 2);
    const [first, last] = fontCoverage[middle];
    if (point < first) end = middle - 1;
    else if (point > last) start = middle + 1;
    else return true;
  }
  return false;
}

/** 与实际嵌入字体的字符表校验，避免 PDF 引擎将未知字符输出为空白。 */
export function validateHblPdfCharacters(model: HblPdfModel): string | null {
  if (!model.issuerNameEn?.trim())
    return '签发公司尚未维护英文名称，请在公司档案中补齐后输出。';
  const fields: [string, unknown][] = [
    ['HBL 号码', model.houseNo],
    ['签发公司英文名称', model.issuerNameEn],
    ...Object.entries(model.content ?? {}).map(
      ([key, value]): [string, unknown] => [
        CONTENT_LABELS[key as keyof API.SeaBillContent] ?? key,
        value,
      ],
    ),
    ...Object.entries(model.printDetails ?? {}).map(
      ([key, value]): [string, unknown] => [
        PRINT_LABELS[key as keyof API.SeaHouseBillPrintDetails] ?? key,
        value,
      ],
    ),
  ];
  for (const [label, value] of fields) {
    if (typeof value !== 'string') continue;
    const unsupported = [
      ...new Set([...value].filter((char) => !hasGlyph(char))),
    ];
    if (unsupported.length > 0)
      return `${label}含暂不支持打印的字符「${unsupported.join('')}」，请调整后重试。`;
  }
  return null;
}

export function pdfText(value?: string | null, maxGlyphs?: number): string {
  const text = value?.replace(/\r\n?/g, '\n').replace(/\t/g, '    ') ?? '';
  if (!maxGlyphs) return text;
  // 按列宽显式拆行；ASCII 与超宽字形使用嵌入字体的 advance，
  // 其余字形按一字一 em 的保守上界，只换行，不插入连字符。
  return text
    .split('\n')
    .map((word) => {
      const lines: string[] = [];
      let line = '';
      let width = 0;
      for (const char of word) {
        const advance =
          fontGlyphWidths[char as keyof typeof fontGlyphWidths] ?? 1;
        if (line && width + advance > maxGlyphs) {
          lines.push(line);
          line = '';
          width = 0;
        }
        line += char;
        width += advance;
      }
      if (line) lines.push(line);
      return lines.join('\n');
    })
    .join('\n');
}

export function pdfQuantity(value?: number, unit?: string): string {
  if (value == null) return '';
  return `${trimDecimal(value)}${unit ? ` ${unit}` : ''}`;
}

export function pdfDate(value?: string): string {
  return value ? formatDate(value, 'date') : '';
}

export function hblPdfFileName(
  houseNo: string,
  variant: HblPdfVariant,
  now: Date,
): string {
  const cleanNo = [...houseNo]
    .map((char) =>
      char.charCodeAt(0) < 32 || '\\/:*?"<>|'.includes(char) ? '_' : char,
    )
    .join('');
  const time = formatDate(now, 'datetime').replaceAll(':', '-');
  return `HBL-${cleanNo}-${variant}-${time}.pdf`;
}
