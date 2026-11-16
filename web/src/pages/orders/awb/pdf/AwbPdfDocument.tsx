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
  ['shipperAccount', '发货人账号 / Shipper account'],
  ['consigneeAccount', '收货人账号 / Consignee account'],
  ['shipperText', '发货人 / Shipper name and address'],
  ['consigneeText', '收货人 / Consignee name and address'],
  ['notifyPartyText', '通知人 / Notify party'],
  ['secondNotifyPartyText', '第二通知人 / Second notify party'],
  ['agentText', '代理 / Issuing agent name and address'],
  ['issuedBy', '已存印发人 / Issued by (saved text)'],
  ['agentIataCode', '代理 IATA 代码 / Agent IATA code'],
  ['agentAccount', '代理账号 / Agent account'],
];
const ROUTING: readonly TextField[] = [
  ['departureText', '起运机场 / Airport of departure'],
  ['destinationText', '目的机场 / Airport of destination'],
  ['routeTo', '第一程至 / To'],
  ['firstCarrier', '第一承运人 / First carrier'],
  ['routeTo2', '第二程至 / To 2'],
  ['carrier2', '第二承运人 / Carrier 2'],
  ['routeTo3', '第三程至 / To 3'],
  ['carrier3', '第三承运人 / Carrier 3'],
  ['flightText', '航班印字 / Flight'],
  ['flightDate', '航班日期 / Flight date'],
];
const INFORMATION: readonly TextField[] = [
  ['accountingInformation', '账务信息 / Accounting information'],
  ['referenceNumber', '参考编号 / Reference number'],
  ['information1', '附加信息一 / Information 1'],
  ['information2', '附加信息二 / Information 2'],
  ['sci', '特别海关信息 / SCI'],
  ['handlingInformation', '处理信息 / Handling information'],
];
const DECLARATIONS: readonly TextField[] = [
  ['currency', '运单币种 / Currency'],
  ['chargeCode', '费用代码 / Charge code'],
  ['weightPayment', '重量费用付款 / Weight payment'],
  ['otherPayment', '其他费用付款 / Other payment'],
];
const GOODS: readonly TextField[] = [
  ['marks', '唛头 / Marks'],
  ['goodsDescription', '运单品名 / Nature and quantity of goods'],
];
const TERMS: readonly TextField[] = [
  ['transportTerms', '运输条款 / Transport terms'],
  ['otherChargesText', '其他费用说明 / Other charges text'],
  ['clauses', '附加条款 / Additional clauses'],
];
const SIGNATURES: readonly TextField[] = [
  ['shipperCertification', '发货人声明 / Shipper certification'],
  ['shipperSignature', '发货人签字印字 / Shipper signature text'],
  ['carrierExecution', '承运人签发声明 / Carrier execution'],
  ['carrierSignature', '承运人签字印字 / Carrier signature text'],
  ['issueDate', '签发日期 / Issue date'],
  ['issuePlace', '签发地点 / Place of issue'],
  ['remarks', '单证备注 / Remarks'],
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
  ['weightCharge', '计重运费 / Weight charge'],
  ['valuationCharge', '估价费 / Valuation charge'],
  ['tax', '税额 / Tax'],
  ['otherDueAgent', '代理其他费用 / Other due agent'],
  ['otherDueCarrier', '承运人其他费用 / Other due carrier'],
  ['total', '合计 / Total'],
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
      language="zh-CN"
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
              ? '外部航空运单登记资料（归档）'
              : '航空分运单 / House Air Waybill'}
          </Text>
        </View>
        <Section title="单证身份 / Document identity">
          <Fields
            fields={[
              ['主运单号码 / MAWB number', model.masterNo],
              ['分运单号码 / HAWB number', model.houseNo],
              ['真实签发主体 / Issuer', model.issuerDisplayName],
              ['保存版本 / Saved version', model.version],
            ]}
          />
          {external && (
            <Text style={{ fontSize: 8, marginTop: 5 }}>
              本资料记录已保存的原单信息，不代表本系统办理签发或外部发送。
            </Text>
          )}
        </Section>
        <Section title="收发货人与代理 / Parties and agent">
          <Fields fields={textFields(c, PARTIES)} />
        </Section>
        <Section title="路线与航班印字 / Routing and flight">
          <Fields fields={textFields(c, ROUTING)} />
        </Section>
        <Section title="账务、处理与海关信息 / Accounting and handling">
          <Fields fields={textFields(c, INFORMATION)} />
        </Section>
        <Section title="币种、付款与声明值 / Currency, payment and declared values">
          <Fields
            fields={[
              ...textFields(c, DECLARATIONS),
              [
                '运输声明价值 / Declared value for carriage',
                awbDeclared(c.carriageValue),
              ],
              [
                '海关声明价值 / Declared value for customs',
                awbDeclared(c.customsValue),
              ],
              ['保险声明 / Amount of insurance', awbDeclared(c.insuranceValue)],
            ]}
          />
        </Section>
        <Section title="货物印字 / Goods">
          <Fields fields={textFields(c, GOODS)} />
        </Section>
        <Section title="运价明细 / Rate details">
          {c.asArranged && <Text style={{ marginBottom: 4 }}>AS ARRANGED</Text>}
          <FlowRow
            label
            widths={[10, 15, 10, 17, 17, 15, 16]}
            cells={[
              '件数\nPieces',
              '毛重 KG\nGross weight',
              '等级\nClass',
              '商品项目号\nCommodity',
              '计费重 KG\nChargeable',
              '费率\nRate',
              '金额\nAmount',
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
                    cells={[`该行品名 / Goods: ${line.goodsDescription}`]}
                  />
                )}
              </View>
            ))}
        </Section>
        <Section title="其他费用 / Other charges">
          <FlowRow
            label
            widths={[24, 10, 11, 8, 13, 14, 10, 10]}
            cells={[
              '费用名称\nCharge',
              '计费单位\nUnit',
              '数量\nQuantity',
              '币种\nCurrency',
              '单价\nUnit price',
              '金额\nAmount',
              '归属\nDue to',
              '付款\nPayment',
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
                      ? '代理'
                      : line.dueTo === 'CARRIER'
                        ? '承运人'
                        : line.dueTo,
                    line.payment,
                  ]}
                />
              </View>
            ))}
          <Fields fields={textFields(c, TERMS)} />
        </Section>
        <Section title={`费用汇总 / Charge summary · ${c.currency || '-'}`}>
          <FlowRow
            label
            widths={[50, 25, 25]}
            cells={['费用项目 / Charge', '预付 PPD', '到付 COLL']}
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
        <Section title="目的地币种与费用 / Destination currency and charges">
          <Fields
            fields={[
              ['目的地币种 / Destination currency', c.destinationCurrency],
              [
                '换算汇率 / Currency conversion rate',
                awbDecimal(c.currencyConversionRate),
              ],
              [
                '目的地到付费 / Collect charge',
                awbAmount(c.destinationCollectCharge),
              ],
              [
                '目的地费用 / Charges at destination',
                awbAmount(c.chargesAtDestination),
              ],
              ['到付总额 / Total collect', awbAmount(c.totalCollect)],
            ]}
          />
        </Section>
        <Section title="声明、签发与备注 / Certification, execution and remarks">
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
            `${model.target} ${awbNumberSummary(model)} · 保存版本 ${model.version.slice(0, 20)} · 第 ${pageNumber} / ${totalPages} 页`
          }
        />
      </Page>
    </Document>
  );
}
