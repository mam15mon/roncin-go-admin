/** 输出版式、字体或分页变化时推进；不静态导入 PDF 渲染器。 */
export const AWB_PDF_TEMPLATE_REVISION = '2026-11-16.2';

/** 仅消费已保存正文的结构类型，不传入运输实体或运行时协议对象。 */
export type AwbPdfContent = API.AirWaybillContent;

export type AwbPdfModel = {
  target: 'MAWB' | 'HAWB';
  businessType: 'AE' | 'AI';
  documentId: string;
  version: string;
  contentHash: string;
  masterNo: string;
  houseNo?: string;
  issuerDisplayName: string;
  outputKind: 'SELF_ISSUED' | 'EXTERNAL_ARCHIVE';
  organizationId: string;
  content: AwbPdfContent;
};
