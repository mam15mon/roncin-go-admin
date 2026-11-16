import { Document, Page, Text, View } from '@react-pdf/renderer';
import type { ReactNode } from 'react';
import { ORDER_PDF_FONT_FAMILY } from '../../documents/pdf/font';
import { pdfText } from '../../documents/pdf/text';
import {
  awbAmount,
  awbDecimal,
  awbDeclared,
  awbNumberSummary,
  validateAwbPdf,
} from './content';
import type { AwbPdfContent, AwbPdfModel } from './types';

type Field = readonly [label: string, value: string | undefined];
type TextKey = {
  [K in keyof AwbPdfContent]: AwbPdfContent[K] extends string | undefined
    ? K
    : never;
}[keyof AwbPdfContent] &
  string;
type TextField = readonly [key: TextKey, label: string];

const PARTIES: readonly TextField[] = [
  ['shipperAccount', 'Shipper account'],
  ['consigneeAccount', 'Consignee account'],
  ['shipperText', 'Shipper name and address'],
  ['consigneeText', 'Consignee name and address'],
  ['notifyPartyText', 'Notify party'],
  ['secondNotifyPartyText', 'Second notify party'],
  ['agentText', 'Issuing agent name and address'],
  ['issuedBy', 'Issued by (saved text)'],
  ['agentIataCode', 'Agent IATA code'],
  ['agentAccount', 'Agent account'],
];
const ROUTING: readonly TextField[] = [
  ['departureText', 'Airport of departure'],
  ['destinationText', 'Airport of destination'],
  ['routeTo', 'To'],
  ['firstCarrier', 'First carrier'],
  ['routeTo2', 'To 2'],
  ['carrier2', 'Carrier 2'],
  ['routeTo3', 'To 3'],
  ['carrier3', 'Carrier 3'],
  ['flightText', 'Flight'],
  ['flightDate', 'Flight date'],
];
const INFORMATION: readonly TextField[] = [
  ['accountingInformation', 'Accounting information'],
  ['referenceNumber', 'Reference number'],
  ['information1', 'Information 1'],
  ['information2', 'Information 2'],
  ['sci', 'SCI'],
  ['handlingInformation', 'Handling information'],
];
const DECLARATIONS: readonly TextField[] = [
  ['currency', 'Currency'],
  ['chargeCode', 'Charge code'],
  ['weightPayment', 'Weight payment'],
  ['otherPayment', 'Other payment'],
];
const GOODS: readonly TextField[] = [
  ['marks', 'Marks'],
  ['goodsDescription', 'Nature and quantity of goods'],
];
const TERMS: readonly TextField[] = [
  ['transportTerms', 'Transport terms'],
  ['otherChargesText', 'Other charges text'],
  ['clauses', 'Additional clauses'],
];
const SIGNATURES: readonly TextField[] = [
  ['shipperCertification', 'Shipper certification'],
  ['shipperSignature', 'Shipper signature text'],
  ['carrierExecution', 'Carrier execution'],
  ['carrierSignature', 'Carrier signature text'],
  ['issueDate', 'Issue date'],
  ['issuePlace', 'Place of issue'],
  ['remarks', 'Remarks'],
];

function textFields(
  content: AwbPdfContent,
  fields: readonly TextField[],
): Field[] {
  return fields.map(([key, label]) => [label, content[key]]);
}

/** 以嵌入字体实际字宽保守拆行，每行独立分页；不让超高并排文本被引擎裁切。 */
function FlowRow({
  cells,
  widths,
  label = false,
}: {
  cells: readonly (string | undefined)[];
  widths: readonly number[];
  label?: boolean;
}) {
  const lines = cells.map((cell, index) =>
    pdfText(
      cell || (label ? '' : '-'),
      (widths[index] * 535.28) / 100 / 8.5 - 1.5,
    ).split('\n'),
  );
  return Array.from(
    { length: Math.max(...lines.map((cell) => cell.length)) },
    (_, line) => ({ id: `line-${line}`, line }),
  ).map(({ id, line }) => (
    <View
      key={id}
      wrap={false}
      style={{
        flexDirection: 'row',
        borderLeftWidth: 0.5,
        borderBottomWidth: 0.5,
        borderColor: '#b7c0ca',
        minHeight: 14,
      }}
    >
      {lines
        .map((cell, column) => ({ id: `column-${column}`, cell, column }))
        .map(({ id, cell, column }) => (
          <View
            key={id}
            style={{
              width: `${widths[column]}%`,
              borderRightWidth: 0.5,
              borderColor: '#b7c0ca',
              paddingHorizontal: 4,
              paddingTop: 1,
              backgroundColor: label ? '#f2f5f8' : '#ffffff',
            }}
          >
            <Text style={{ fontSize: label ? 7 : 8.5, height: 13 }}>
              {cell[line] || ' '}
            </Text>
          </View>
        ))}
    </View>
  ));
}

function Fields({ fields }: { fields: readonly Field[] }) {
  const present = fields.filter(([, value]) => value != null && value !== '');
  return Array.from({ length: Math.ceil(present.length / 2) }, (_, row) => {
    const pair = present.slice(row * 2, row * 2 + 2);
    const widths = pair.length === 2 ? [50, 50] : [100];
    return (
      <View key={pair[0][0]}>
        <View minPresenceAhead={28}>
          <FlowRow cells={pair.map(([label]) => label)} widths={widths} label />
        </View>
        <FlowRow cells={pair.map(([, value]) => value)} widths={widths} />
      </View>
    );
  });
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <View style={{ marginTop: 7 }}>
      <Text
        wrap={false}
        minPresenceAhead={36}
        style={{ fontSize: 10, marginBottom: 4 }}
      >
        {title}
      </Text>
      {children}
    </View>
  );
}

const SUMMARY: readonly (readonly [keyof API.AirChargeSummary, string])[] = [
  ['weightCharge', 'Weight charge'],
  ['valuationCharge', 'Valuation charge'],
  ['tax', 'Tax'],
  ['otherDueAgent', 'Other due agent'],
  ['otherDueCarrier', 'Other due carrier'],
  ['total', 'Total'],
];

/** 一套航空 A4 分区，归档与自签只表达真实主体，不增加签章、放货或业务状态事实。 */
export function AwbPdfDocument({ model }: { model: AwbPdfModel }) {
  const invalid = validateAwbPdf(model);
  if (invalid) throw new Error(invalid);
  const c = model.content;
  const external = model.outputKind === 'EXTERNAL_ARCHIVE';
  return (
    <Document
      title={`${model.target} ${awbNumberSummary(model)}`}
      author={model.issuerDisplayName}
      language="en"
    >
      <Page
        size="A4"
        wrap
        style={{
          fontFamily: ORDER_PDF_FONT_FAMILY,
          fontSize: 8.5,
          paddingHorizontal: 30,
          paddingTop: 71,
          paddingBottom: 48,
        }}
      >
        <View
          fixed
          style={{ position: 'absolute', top: 23, left: 30, right: 30 }}
        >
          <Text style={{ fontSize: 16 }}>AIR WAYBILL · {model.target}</Text>
          <Text style={{ fontSize: 8, marginTop: 4 }}>
            {external
              ? 'EXTERNAL AIR WAYBILL RECORD (ARCHIVE)'
              : 'House Air Waybill'}
          </Text>
        </View>
        <Section title="Document identity">
          <Fields
            fields={[
              ['MAWB number', model.masterNo],
              ['HAWB number', model.houseNo],
              ['Issuer', model.issuerDisplayName],
              ['Saved version', model.version],
            ]}
          />
          {external && (
            <Text style={{ fontSize: 8, marginTop: 5 }}>
              This record reflects the saved original document. It does not
              constitute issuance or external transmission.
            </Text>
          )}
        </Section>
        <Section title="Parties and agent">
          <Fields fields={textFields(c, PARTIES)} />
        </Section>
        <Section title="Routing and flight">
          <Fields fields={textFields(c, ROUTING)} />
        </Section>
        <Section title="Accounting and handling">
          <Fields fields={textFields(c, INFORMATION)} />
        </Section>
        <Section title="Currency, payment and declared values">
          <Fields
            fields={[
              ...textFields(c, DECLARATIONS),
              ['Declared value for carriage', awbDeclared(c.carriageValue)],
              ['Declared value for customs', awbDeclared(c.customsValue)],
              ['Amount of insurance', awbDeclared(c.insuranceValue)],
            ]}
          />
        </Section>
        <Section title="Goods">
          <Fields fields={textFields(c, GOODS)} />
        </Section>
        <Section title="Rate details">
          {c.asArranged && <Text style={{ marginBottom: 4 }}>AS ARRANGED</Text>}
          <FlowRow
            label
            widths={[10, 15, 10, 17, 17, 15, 16]}
            cells={[
              'Pieces',
              'Gross weight',
              'Class',
              'Commodity',
              'Chargeable',
              'Rate',
              'Amount',
            ]}
          />
          {c.rateLines
            ?.map((line, index) => ({ id: `rate-${index}`, line }))
            .map(({ id, line }) => (
              <View key={id}>
                <FlowRow
                  widths={[10, 15, 10, 17, 17, 15, 16]}
                  cells={[
                    awbDecimal(line.pieces),
                    awbDecimal(line.grossWeightKg),
                    line.rateClass,
                    line.commodityItemNo,
                    awbDecimal(line.chargeableWeightKg),
                    awbDecimal(line.rate),
                    awbAmount(line.amount),
                  ]}
                />
                {line.goodsDescription && (
                  <FlowRow
                    widths={[100]}
                    cells={[`Goods: ${line.goodsDescription}`]}
                  />
                )}
              </View>
            ))}
        </Section>
        <Section title="Other charges">
          <FlowRow
            label
            widths={[24, 10, 11, 8, 13, 14, 10, 10]}
            cells={[
              'Charge',
              'Unit',
              'Quantity',
              'Currency',
              'Unit price',
              'Amount',
              'Due to',
              'Payment',
            ]}
          />
          {c.otherCharges
            ?.map((line, index) => ({ id: `other-${index}`, line }))
            .map(({ id, line }) => (
              <View key={id}>
                <FlowRow
                  widths={[24, 10, 11, 8, 13, 14, 10, 10]}
                  cells={[
                    line.name,
                    line.billingUnit,
                    awbDecimal(line.quantity),
                    line.currency,
                    awbDecimal(line.unitPrice),
                    awbAmount(line.amount),
                    line.dueTo === 'AGENT'
                      ? 'Agent'
                      : line.dueTo === 'CARRIER'
                        ? 'Carrier'
                        : line.dueTo,
                    line.payment,
                  ]}
                />
              </View>
            ))}
          <Fields fields={textFields(c, TERMS)} />
        </Section>
        <Section title={`Charge summary · ${c.currency || '-'}`}>
          <FlowRow
            label
            widths={[50, 25, 25]}
            cells={['Charge', 'Prepaid PPD', 'Collect COLL']}
          />
          {SUMMARY.map(([key, label]) => (
            <FlowRow
              key={key}
              widths={[50, 25, 25]}
              cells={[
                label,
                c.asArranged &&
                key === 'weightCharge' &&
                c.weightPayment === 'PPD' &&
                !c.prepaid?.[key]
                  ? 'AS ARRANGED'
                  : awbAmount(c.prepaid?.[key]),
                c.asArranged &&
                key === 'weightCharge' &&
                c.weightPayment === 'COLL' &&
                !c.collect?.[key]
                  ? 'AS ARRANGED'
                  : awbAmount(c.collect?.[key]),
              ]}
            />
          ))}
        </Section>
        <Section title="Destination currency and charges">
          <Fields
            fields={[
              ['Destination currency', c.destinationCurrency],
              [
                'Currency conversion rate',
                awbDecimal(c.currencyConversionRate),
              ],
              ['Collect charge', awbAmount(c.destinationCollectCharge)],
              ['Charges at destination', awbAmount(c.chargesAtDestination)],
              ['Total collect', awbAmount(c.totalCollect)],
            ]}
          />
        </Section>
        <Section title="Certification, execution and remarks">
          <Fields fields={textFields(c, SIGNATURES)} />
        </Section>
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
            `${model.target} ${awbNumberSummary(model)} · Saved version ${model.version.slice(0, 20)} · Page ${pageNumber} / ${totalPages}`
          }
        />
      </Page>
    </Document>
  );
}
