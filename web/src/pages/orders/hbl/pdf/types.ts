/** 输出版式、字体或分页变化时同步推进，参与预览与变动检查指纹。 */
export const HBL_PDF_TEMPLATE_REVISION = '2026-11-16.2';

export type HblPdfVariant = 'draft' | 'original' | 'surrendered' | 'copy';

export type HblPdfModel = {
  houseNo: string;
  /** 进口外部提单仅输出登记归档资料，不套用本公司签发模板。 */
  outputKind?: 'external-archive';
  issuerDisplayName?: string;
  savedVersion?: string;
  issuerNameEn: string;
  content?: API.SeaBillContent;
  printDetails?: API.SeaHouseBillPrintDetails;
};

export const HBL_PDF_VARIANTS: { label: string; value: HblPdfVariant }[] = [
  { label: '草稿', value: 'draft' },
  { label: '正本', value: 'original' },
  { label: '电放', value: 'surrendered' },
  { label: '副本', value: 'copy' },
];

export const HBL_PDF_VARIANT_MARKS: Record<HblPdfVariant, string> = {
  draft: 'DRAFT',
  original: 'ORIGINAL',
  surrendered: 'SURRENDERED / TELEX RELEASE',
  copy: 'NON-NEGOTIABLE COPY',
};
