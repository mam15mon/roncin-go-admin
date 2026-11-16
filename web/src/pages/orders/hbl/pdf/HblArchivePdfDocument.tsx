import { Document, Page, Text, View } from '@react-pdf/renderer';
import {
  pdfDate,
  pdfQuantity,
  pdfText,
  validateHblPdfCharacters,
} from './content';
import { HBL_PDF_FONT_FAMILY } from './font';
import type { HblPdfModel } from './types';

/** 外部提单资料页只呈现已登记事实，逐行分页以保留所有长文。 */
export function HblArchivePdfDocument({ model }: { model: HblPdfModel }) {
  const invalid = validateHblPdfCharacters(model);
  if (invalid) throw new Error(invalid);
  // 固定页脚只放有界摘要，完整号码仍在可分页正文中保留。
  const numberCharacters = [...model.houseNo.replace(/\s+/gu, ' ')];
  const numberSummary = `${numberCharacters.slice(0, 32).join('')}${numberCharacters.length > 32 ? '…' : ''}`;
  const content = model.content ?? {};
  const print = model.printDetails ?? {};
  const fields: [string, string | undefined][] = [
    ['HBL 号码', model.houseNo],
    ['外部签发方', model.issuerDisplayName],
    ['保存版本', model.savedVersion],
    ['发货人', content.shipperText],
    ['收货人', content.consigneeText],
    ['通知人', content.notifyPartyText],
    ['第二通知人', content.secondNotifyPartyText],
    ['国外代理', content.foreignAgentText],
    ['唛头', content.marksText],
    ['货物描述', content.goodsDescriptionText],
    [
      '件数',
      content.packageCount == null
        ? undefined
        : pdfQuantity(content.packageCount),
    ],
    ['包装单位', content.packageUnit],
    [
      '毛重 (KG)',
      content.grossWeightKg == null
        ? undefined
        : pdfQuantity(content.grossWeightKg),
    ],
    [
      '体积 (CBM)',
      content.volumeCbm == null ? undefined : pdfQuantity(content.volumeCbm),
    ],
    ['运费条款', content.freightTerms],
    ['运输条款', content.transportTerms],
    ['提单形式（原单登记）', content.billForm],
    ['放单方式（原单登记）', content.releaseType],
    ['特别条款（原单登记）', content.clauses],
    ['联系方式', print.headerContactText],
    ['订舱号', print.bookingNoText],
    ['业务参考号', print.exportReferencesText],
    ['前程运输', print.preCarriageText],
    ['收货地', print.placeOfReceiptText],
    ['境内运输指示', print.domesticRoutingText],
    ['原产地', print.pointOfOriginText],
    ['后续内陆运输', print.onwardRoutingText],
    ['船名航次', print.vesselVoyageText],
    ['装货港', print.portOfLoadingText],
    ['卸货港', print.portOfDischargeText],
    ['交货地', print.placeOfDeliveryText],
    ['箱号与封号', print.containersText],
    [
      '正本份数（原单登记）',
      print.originalCount == null ? undefined : String(print.originalCount),
    ],
    ['装船日期', pdfDate(print.shippedOnBoardDate)],
    ['签发地（原单登记）', print.issuePlaceText],
    ['签发日期（原单登记）', pdfDate(print.issueDate)],
    ['交货联系信息', print.deliveryContactText],
    ['其他参考', print.footerReferenceText],
  ];
  return (
    <Document
      title={`外部提单归档 ${model.houseNo}`}
      author={model.issuerDisplayName}
      language="zh-CN"
    >
      <Page
        size="A4"
        wrap
        style={{
          fontFamily: HBL_PDF_FONT_FAMILY,
          fontSize: 9,
          padding: 30,
          paddingTop: 64,
          paddingBottom: 50,
        }}
      >
        <View
          fixed
          style={{ position: 'absolute', top: 22, left: 30, right: 30 }}
        >
          <Text style={{ fontSize: 14 }}>外部提单登记资料（归档）</Text>
          <Text style={{ fontSize: 8, marginTop: 5 }}>
            本资料记录原单信息，不代表本公司签发提单或办理放货。
          </Text>
        </View>
        {fields
          .filter(([, value]) => value != null && value !== '')
          .map(([label, value]) => (
            <View key={label} style={{ marginBottom: 8 }}>
              <Text
                style={{ fontSize: 9, color: '#475569', marginBottom: 3 }}
                minPresenceAhead={14}
                wrap={false}
              >
                {label}
              </Text>
              {pdfText(value, 52)
                .split('\n')
                .map((value, index) => ({ id: `${label}:${index}`, value }))
                .map((line) => (
                  <Text
                    key={line.id}
                    wrap={false}
                    style={{ height: 14, fontSize: 9 }}
                  >
                    {line.value || ' '}
                  </Text>
                ))}
            </View>
          ))}
        <Text
          fixed
          style={{
            position: 'absolute',
            bottom: 23,
            left: 30,
            right: 30,
            fontSize: 8,
          }}
          render={({ pageNumber, totalPages }) =>
            `HBL ${numberSummary} · 保存版本 ${model.savedVersion ?? '-'} · 第 ${pageNumber} / ${totalPages} 页`
          }
        />
      </Page>
    </Document>
  );
}
