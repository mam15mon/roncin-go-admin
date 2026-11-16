import { Document, Page, Text, View } from '@react-pdf/renderer';
import {
  hblPdfContent,
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
  const content = hblPdfContent(model);
  const print = model.printDetails ?? {};
  const fields: [string, string | undefined][] = [
    ['HBL number', model.houseNo],
    ['External issuer', model.issuerDisplayName],
    ['Saved version', model.savedVersion],
    ['Shipper', content.shipperText],
    ['Consignee', content.consigneeText],
    ['Notify party', content.notifyPartyText],
    ['Second notify party', content.secondNotifyPartyText],
    ['Forwarding agent', content.foreignAgentText],
    ['Marks and numbers', content.marksText],
    ['Description of goods', content.goodsDescriptionText],
    [
      'Number of packages',
      content.packageCount == null
        ? undefined
        : pdfQuantity(content.packageCount),
    ],
    ['Package unit', content.packageUnit],
    [
      'Gross weight (KG)',
      content.grossWeightKg == null
        ? undefined
        : pdfQuantity(content.grossWeightKg),
    ],
    [
      'Volume (CBM)',
      content.volumeCbm == null ? undefined : pdfQuantity(content.volumeCbm),
    ],
    ['Freight terms', content.freightTerms],
    ['Transport terms', content.transportTerms],
    ['Bill form (as registered)', content.billForm],
    ['Release type (as registered)', content.releaseType],
    ['Additional clauses (as registered)', content.clauses],
    ['Contact details', print.headerContactText],
    ['Booking number', print.bookingNoText],
    ['Reference number', print.exportReferencesText],
    ['Pre-carriage', print.preCarriageText],
    ['Place of receipt', print.placeOfReceiptText],
    ['Domestic routing instructions', print.domesticRoutingText],
    ['Point of origin', print.pointOfOriginText],
    ['Onward inland routing', print.onwardRoutingText],
    ['Vessel / voyage', print.vesselVoyageText],
    ['Port of loading', print.portOfLoadingText],
    ['Port of discharge', print.portOfDischargeText],
    ['Place of delivery', print.placeOfDeliveryText],
    ['Container / seal numbers', print.containersText],
    [
      'Number of originals (as registered)',
      print.originalCount == null ? undefined : String(print.originalCount),
    ],
    ['Shipped on board date', pdfDate(print.shippedOnBoardDate)],
    ['Place of issue (as registered)', print.issuePlaceText],
    ['Date of issue (as registered)', pdfDate(print.issueDate)],
    ['Delivery contact details', print.deliveryContactText],
    ['Other references', print.footerReferenceText],
  ];
  return (
    <Document
      title={`External bill of lading archive ${model.houseNo}`}
      author={model.issuerDisplayName}
      language="en"
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
          <Text style={{ fontSize: 14 }}>
            EXTERNAL BILL OF LADING RECORD (ARCHIVE)
          </Text>
          <Text style={{ fontSize: 8, marginTop: 5 }}>
            This record reflects the original document. It does not constitute
            issuance or cargo release by this company.
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
            `HBL ${numberSummary} · Saved version ${model.savedVersion ?? '-'} · Page ${pageNumber} / ${totalPages}`
          }
        />
      </Page>
    </Document>
  );
}
