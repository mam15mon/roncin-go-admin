import {
  Document,
  Image,
  Page,
  StyleSheet,
  Text,
  View,
} from '@react-pdf/renderer';
import { pdfDate, pdfQuantity, pdfText } from './content';
import { HBL_PDF_FONT_FAMILY } from './font';
import {
  HBL_PDF_VARIANT_MARKS,
  type HblPdfModel,
  type HblPdfVariant,
} from './types';

const COMPANY = 'RONCIN (BEIJING) SUPPLY CHAIN MANAGEMENT CO., LTD';
const BODY_WIDTH = 559.28;
const BODY_FONT_SIZE = 8.1;
const BODY_LINE_HEIGHT = 11;

const styles = StyleSheet.create({
  page: {
    paddingTop: 98,
    paddingHorizontal: 18,
    paddingBottom: 46,
    fontFamily: HBL_PDF_FONT_FAMILY,
    fontSize: BODY_FONT_SIZE,
    color: '#111111',
  },
  header: { position: 'absolute', top: 15, left: 18, right: 18, height: 76 },
  company: {
    fontFamily: 'Helvetica-Bold',
    fontSize: 14.1,
    color: '#3d409c',
    letterSpacing: -0.2,
  },
  subtitle: {
    fontFamily: 'Helvetica',
    fontSize: 9,
    textAlign: 'center',
    marginTop: 4,
    marginRight: 65,
  },
  logo: { position: 'absolute', width: 70, height: 47.2, right: 0, top: 12 },
  headerLine: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: 'row',
    justifyContent: 'space-between',
    borderBottomWidth: 0.7,
    paddingBottom: 3,
    fontSize: 7,
  },
  variant: { fontFamily: 'Helvetica-Bold', color: '#30358b' },
  row: { flexDirection: 'row', flexShrink: 0 },
  column: {
    paddingHorizontal: 4,
    borderRightWidth: 0.6,
    borderColor: '#333333',
  },
  finalColumn: { borderRightWidth: 0 },
  label: { fontFamily: 'Helvetica', fontSize: 6.2, paddingTop: 3 },
  bottom: { borderBottomWidth: 0.6, borderColor: '#333333' },
  cargoTitle: {
    fontFamily: 'Helvetica',
    fontSize: 6.4,
    textAlign: 'center',
    borderBottomWidth: 0.6,
    paddingVertical: 3,
  },
  clause: {
    fontFamily: 'Helvetica',
    fontSize: 6.1,
    marginTop: 5,
    marginHorizontal: 5,
  },
  footer: {
    position: 'absolute',
    left: 18,
    right: 18,
    bottom: 10,
    borderTopWidth: 0.6,
    paddingTop: 4,
    fontSize: 6.4,
  },
});

type Column = { title: string; value?: string; width: number };

/**
 * 每次只向分页引擎交付一行短表格，避免超高并排单元格在跨页重排时互相压缩。
 * 字段先按嵌入字体的列宽拆行；只有单行不可拆分，整个长字段仍正常跨页。
 */
function FlowTable({
  columns,
  minRows = 1,
}: {
  columns: Column[];
  minRows?: number;
}) {
  const prepared = columns.map((column) => ({
    ...column,
    lines: pdfText(
      column.value,
      Math.max(1, (BODY_WIDTH * column.width - 9) / BODY_FONT_SIZE),
    ).split('\n'),
  }));
  const lineCount = Math.max(
    minRows,
    ...prepared.map((column) => column.lines.length),
  );
  const rows = Array.from({ length: lineCount }, (_, index) => ({
    id: `line-${index}`,
    index,
  }));
  return (
    <>
      <View style={[styles.row, { height: 16 }]} wrap={false}>
        {prepared.map((column, index) => (
          <View
            key={column.title}
            style={[
              styles.column,
              { width: BODY_WIDTH * column.width, height: 16 },
              ...(index === prepared.length - 1 ? [styles.finalColumn] : []),
            ]}
          >
            <Text style={styles.label}>{column.title}</Text>
          </View>
        ))}
      </View>
      {rows.map((row) => (
        <View
          key={row.id}
          style={[
            styles.row,
            { height: BODY_LINE_HEIGHT },
            ...(row.index === lineCount - 1 ? [styles.bottom] : []),
          ]}
          wrap={false}
        >
          {prepared.map((column, index) => (
            <View
              key={column.title}
              style={[
                styles.column,
                { width: BODY_WIDTH * column.width, height: BODY_LINE_HEIGHT },
                ...(index === prepared.length - 1 ? [styles.finalColumn] : []),
              ]}
            >
              <Text>{column.lines[row.index] || ' '}</Text>
            </View>
          ))}
        </View>
      ))}
    </>
  );
}

function FlowText({
  value,
  fontSize = BODY_FONT_SIZE,
  color = '#111111',
}: {
  value?: string;
  fontSize?: number;
  color?: string;
}) {
  return pdfText(value, (BODY_WIDTH - 10) / fontSize)
    .split('\n')
    .map((line, index) => ({ value: line, id: `line-${index}` }))
    .map((line) => (
      <Text
        key={line.id}
        style={{ fontSize, color, height: fontSize * 1.4 }}
        wrap={false}
      >
        {line.value || ' '}
      </Text>
    ));
}

function freightText(value?: string): string {
  if (value === '预付') return 'FREIGHT PREPAID';
  if (value === '到付') return 'FREIGHT COLLECT';
  return value ?? '';
}

export type HblPdfDocumentProps = {
  model: HblPdfModel;
  variant: HblPdfVariant;
  logoSrc?: string;
};

/** 业务内容只读投影；预览、下载和打印共用这一份 A4 文档。 */
export function HblPdfDocument({
  model,
  variant,
  logoSrc = '/images/roncin-hbl-logo.png',
}: HblPdfDocumentProps) {
  const content = model.content ?? {};
  const details = model.printDetails ?? {};
  const variantMark = HBL_PDF_VARIANT_MARKS[variant];
  // 页脚号码摘要折叠空白，避免内部换行把固定区域撑满整页；正文仍保留原文。
  const footerHouseNo = pdfText(model.houseNo.replace(/\s+/gu, ' '), 75);
  const paddingBottom = Math.max(
    46,
    26 + (footerHouseNo.split('\n').length + 1) * 8,
  );
  const cargoDescription = [
    content.goodsDescriptionText,
    details.containersText
      ? `\nCONTAINER / SEAL / PARTICULARS\n${details.containersText}`
      : '',
    content.clauses ? `\n${content.clauses}` : '',
    content.freightTerms ? `\n${freightText(content.freightTerms)}` : '',
  ]
    .filter(Boolean)
    .join('\n');

  return (
    <Document
      title={`HBL ${model.houseNo} - ${variantMark}`}
      author={COMPANY}
      language="zh-CN"
    >
      <Page size="A4" style={[styles.page, { paddingBottom }]} wrap>
        <View style={styles.header} fixed>
          <Text style={styles.company}>{COMPANY}</Text>
          <Text style={styles.subtitle}>COMBINED TRANSPORT BILL OF LADING</Text>
          <Image style={styles.logo} src={logoSrc} />
          <View style={styles.headerLine}>
            <Text>BILL OF LADING</Text>
            <Text style={styles.variant}>{variantMark}</Text>
          </View>
        </View>
        {details.headerContactText && (
          <FlowText
            value={details.headerContactText}
            fontSize={7.2}
            color="#3d409c"
          />
        )}
        <FlowTable
          columns={[
            {
              title: 'SHIPPER / EXPORTER',
              value: content.shipperText,
              width: 0.5,
            },
            {
              title: 'BOOKING NO. / B/L NO. / EXPORT REFERENCES',
              value: `${details.bookingNoText ?? ''}\nB/L NO. ${model.houseNo}\n${details.exportReferencesText ?? ''}`,
              width: 0.5,
            },
          ]}
          minRows={4}
        />
        <FlowTable
          columns={[
            { title: 'CONSIGNEE', value: content.consigneeText, width: 0.5 },
            {
              title: 'DOMESTIC ROUTING / POINT AND COUNTRY OF ORIGIN',
              value: `${details.domesticRoutingText ?? ''}\n${details.pointOfOriginText ?? ''}`,
              width: 0.5,
            },
          ]}
          minRows={4}
        />
        <FlowTable
          columns={[
            {
              title: 'NOTIFY PARTY',
              value: content.notifyPartyText,
              width: 0.5,
            },
            {
              title: 'FORWARDING AGENT / REFERENCES',
              value: content.foreignAgentText,
              width: 0.5,
            },
          ]}
          minRows={4}
        />
        {content.secondNotifyPartyText && (
          <FlowTable
            columns={[
              {
                title: 'ALSO NOTIFY PARTY',
                value: content.secondNotifyPartyText,
                width: 1,
              },
            ]}
          />
        )}
        <FlowTable
          columns={[
            {
              title: 'PRE-CARRIAGE BY',
              value: details.preCarriageText,
              width: 0.5,
            },
            {
              title: 'PLACE OF RECEIPT',
              value: details.placeOfReceiptText,
              width: 0.5,
            },
          ]}
        />
        <FlowTable
          columns={[
            {
              title: 'OCEAN VESSEL / VOY. NO.',
              value: details.vesselVoyageText,
              width: 0.34,
            },
            {
              title: 'PORT OF LOADING',
              value: details.portOfLoadingText,
              width: 0.33,
            },
            {
              title: 'ONWARD INLAND ROUTING',
              value: details.onwardRoutingText,
              width: 0.33,
            },
          ]}
        />
        <FlowTable
          columns={[
            {
              title: 'PORT OF DISCHARGE',
              value: details.portOfDischargeText,
              width: 0.28,
            },
            {
              title: 'PLACE OF DELIVERY',
              value: details.placeOfDeliveryText,
              width: 0.28,
            },
            {
              title: 'TYPE OF MOVE',
              value: content.transportTerms,
              width: 0.22,
            },
            {
              title: 'NO. OF ORIGINAL B/L',
              value: pdfQuantity(details.originalCount),
              width: 0.22,
            },
          ]}
        />
        <Text style={styles.cargoTitle}>PARTICULARS FURNISHED BY SHIPPER</Text>
        <FlowTable
          columns={[
            {
              title: 'MARKS AND NUMBERS',
              value: content.marksText,
              width: 0.16,
            },
            {
              title: 'NO. OF PKGS',
              value: pdfQuantity(content.packageCount, content.packageUnit),
              width: 0.13,
            },
            {
              title: 'DESCRIPTION OF PACKAGES AND GOODS',
              value: cargoDescription,
              width: 0.45,
            },
            {
              title: 'GROSS WEIGHT',
              value: pdfQuantity(content.grossWeightKg, 'KGS'),
              width: 0.14,
            },
            {
              title: 'MEASUREMENT',
              value: pdfQuantity(content.volumeCbm, 'CBM'),
              width: 0.12,
            },
          ]}
          minRows={12}
        />
        <FlowTable
          columns={[
            {
              title: 'FREIGHT AND CHARGES',
              value: `SHIPPED ON BOARD\nDate: ${pdfDate(details.shippedOnBoardDate)}`,
              width: 0.4,
            },
            {
              title: 'TO OBTAIN DELIVERY CONTACT',
              value: details.deliveryContactText,
              width: 0.6,
            },
          ]}
          minRows={3}
        />
        <Text style={styles.clause}>
          The surrender of the original order bill of lading properly endorsed
          shall be required before delivery of the property. Inspection of
          property covered by this bill of lading will not be permitted unless
          provided by law or unless permission is endorsed on this original bill
          of lading or given in writing by the shipper.
        </Text>
        <Text style={styles.clause}>
          IN WITNESS WHEREOF, THE UNDERSIGNED, SIGNING ON BEHALF OF {COMPANY},
          HAS SIGNED{' '}
          {details.originalCount == null
            ? 'THE NUMBER OF ORIGINAL BILLS OF LADING STATED ABOVE'
            : `${details.originalCount} ORIGINAL BILL(S) OF LADING`}
          , ALL OF THE SAME TENOR AND DATE, ONE OF WHICH BEING ACCOMPLISHED, THE
          OTHERS TO STAND VOID.
        </Text>
        <Text style={{ marginTop: 8, fontSize: 7.5 }}>
          Place and date of issue:
        </Text>
        <FlowText
          value={`${details.issuePlaceText ?? ''} ${pdfDate(details.issueDate)}`}
          fontSize={7.5}
        />
        <Text style={{ marginTop: 5, fontSize: 7.5 }}>
          For and on behalf of
        </Text>
        <Text
          style={{ fontFamily: 'Helvetica-Bold', fontSize: 8, marginTop: 3 }}
        >
          {COMPANY}
        </Text>
        <Text style={[styles.clause, { marginTop: 8, marginBottom: 5 }]}>
          ATTENTION OF SHIPPER: The terms and conditions of the order bill of
          lading under which this shipment is accepted form part of this bill.
          Unless otherwise specified, charges do not include customs duties,
          taxes, customs clearance charges and similar non-transportation
          charges for the account of the cargo.
        </Text>
        {details.footerReferenceText && (
          <FlowTable
            columns={[
              {
                title: 'REFERENCE',
                value: details.footerReferenceText,
                width: 1,
              },
            ]}
          />
        )}
        <View style={styles.footer} fixed>
          <Text>B/L NO. {footerHouseNo}</Text>
          <Text
            render={({ pageNumber, totalPages }) =>
              `${variantMark}  |  ${pageNumber} / ${totalPages}`
            }
          />
        </View>
      </Page>
    </Document>
  );
}
