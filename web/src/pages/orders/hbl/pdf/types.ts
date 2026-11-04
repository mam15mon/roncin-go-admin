export type HblPdfVariant = 'draft' | 'original' | 'surrendered' | 'copy';

export type HblPdfModel = {
  houseNo: string;
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
