import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Col, Form, Input, Row, Select, Typography } from 'antd';
import Decimal from 'decimal.js';
import { formatAmount, trimDecimal } from '@/utils/format';
import {
  applyAllocationChange,
  applyBillSelection,
  applyFeeChange,
  applyFeeSelection,
  availableFor,
  recordGaps,
  syncAllocationTotal,
} from './autofill';
import type {
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
} from './types';

type Name = (string | number)[];
export type RecordUpdater = (
  recordIndex: number,
  change: (
    record: InvoiceRegistrationDraft,
    records: InvoiceRegistrationDraft[],
  ) => InvoiceRegistrationDraft,
) => void;

const required = { required: true, message: '请明确填写本项' };
const AMOUNT = /^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/;
const amountRule = {
  validator: async (_: unknown, value: string) => {
    if (!AMOUNT.test(value ?? '') || !new Decimal(value).gt(0))
      throw new Error('请输入大于0的金额，最多8位小数');
  },
};
const listRule = {
  validator: async (_: unknown, value: unknown[]) => {
    if (!value?.length) throw new Error('至少添加一项明确分配');
  },
};

export function rateText(rate?: string) {
  return rate === undefined ? '-' : `${trimDecimal(rate)}%`;
}

export function GapText({ label, gap }: { label: string; gap?: Decimal }) {
  if (!gap)
    return <Typography.Text type="secondary">{label} -</Typography.Text>;
  return (
    <Typography.Text type={gap.isZero() ? 'success' : 'danger'}>
      {label} {formatAmount(gap.toFixed())}
    </Typography.Text>
  );
}

function sourceSetter(
  form: ReturnType<typeof Form.useFormInstance>,
  path: Name,
) {
  return () => form.setFieldValue(path, 'manual');
}

/** 单张账单分配：显示可登记余额、本次分配与费用差额，逐费用含税额和税额可编辑。 */
export default function InvoiceAllocationFields({
  listName,
  recordIndex,
  name,
  bills,
  update,
  remove,
}: {
  listName: Name;
  recordIndex: number;
  name: number;
  bills: InvoiceRegistrationBillSource[];
  update: RecordUpdater;
  remove: () => void;
}) {
  const form = Form.useFormInstance();
  const path: Name = [...listName, recordIndex, 'allocations', name];
  const records = Form.useWatch(listName, { form, preserve: true }) as
    | InvoiceRegistrationDraft[]
    | undefined;
  const record = records?.[recordIndex];
  const allocation = record?.allocations?.[name];
  const bill = bills.find((item) => item.id === allocation?.billId);
  const billAvailable = (bill?.fees ?? []).reduce(
    (sum, fee) =>
      sum.plus(availableFor(fee, bill?.id ?? '', records, recordIndex)),
    new Decimal(0),
  );
  const gap = recordGaps(record).billGaps[name];
  const feeSource = (orderFeeId?: string) =>
    bill?.fees.find((fee) => fee.orderFeeId === orderFeeId);
  return (
    <Card
      size="small"
      style={{ marginBottom: 8 }}
      title="账单分配"
      extra={
        <Button type="text" danger onClick={remove}>
          移除账单分配
        </Button>
      }
    >
      <Row gutter={12}>
        <Col xs={24} md={10}>
          <Form.Item
            name={[name, 'billId']}
            label="来源账单"
            rules={[required]}
          >
            <Select
              options={bills.map((item) => ({
                value: item.id,
                label: item.billNo || '本组新账单',
              }))}
              onChange={() =>
                update(recordIndex, (current, all) =>
                  applyBillSelection(current, name, bills, all, recordIndex),
                )
              }
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item
            name={[name, 'amount']}
            label={`本账单含税分配（${bill?.currency ?? ''}）`}
            rules={[required, amountRule]}
          >
            <Input
              onChange={() => {
                sourceSetter(form, [...path, 'amountSource'])();
                update(recordIndex, (current) =>
                  applyAllocationChange(current, name, bills),
                );
              }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={6}>
          <Typography.Paragraph style={{ marginTop: 30, marginBottom: 0 }}>
            <Typography.Text type="secondary">
              可登记余额 {bill ? formatAmount(billAvailable.toFixed()) : '-'}
            </Typography.Text>
            <br />
            <GapText label="费用待分配" gap={gap} />
          </Typography.Paragraph>
        </Col>
      </Row>
      <Form.List name={[name, 'fees']} rules={[listRule]}>
        {(fields, { add, remove: removeFee }, errors) => (
          <>
            {fields.map((field) => {
              const feePath = [...path, 'fees', field.name];
              const draft = allocation?.fees?.[field.name];
              const source = feeSource(draft?.orderFeeId);
              const available = source
                ? availableFor(source, bill?.id ?? '', records, recordIndex)
                : undefined;
              const suggested =
                draft?.taxSource !== 'manual' &&
                source?.sourceTaxRate !== undefined &&
                Boolean(draft?.taxAmount);
              return (
                <Row key={field.key} gutter={8} align="top">
                  <Col xs={24} md={11}>
                    <Form.Item
                      name={[field.name, 'orderFeeId']}
                      label="来源费用及可登记余额"
                      rules={[required]}
                      extra={
                        source &&
                        `本票可用 ${formatAmount(available?.toFixed())} · 来源税率 ${rateText(source.sourceTaxRate)}`
                      }
                    >
                      <Select
                        options={(bill?.fees ?? []).map((fee) => ({
                          value: fee.orderFeeId,
                          label: `${fee.feeCode ?? ''} ${fee.feeName ?? ''} · ${formatAmount(fee.invoiceableAmount ?? fee.totalAmount)} · 来源税率 ${rateText(fee.sourceTaxRate)}`,
                        }))}
                        onChange={() =>
                          update(recordIndex, (current, all) =>
                            applyFeeSelection(
                              current,
                              name,
                              field.name,
                              bills,
                              all,
                              recordIndex,
                            ),
                          )
                        }
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item
                      name={[field.name, 'amount']}
                      label="本次含税金额"
                      rules={[
                        required,
                        amountRule,
                        {
                          validator: async (_, value: string) => {
                            if (
                              available &&
                              AMOUNT.test(value ?? '') &&
                              new Decimal(value).gt(available)
                            )
                              throw new Error(
                                `超过可登记余额 ${formatAmount(available.toFixed())}`,
                              );
                          },
                        },
                      ]}
                    >
                      <Input
                        onChange={() => {
                          sourceSetter(form, [...feePath, 'amountSource'])();
                          update(recordIndex, (current) =>
                            applyFeeChange(current, name, field.name, bills),
                          );
                        }}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item
                      name={[field.name, 'taxAmount']}
                      dependencies={[[...feePath, 'amount']]}
                      label="本次登记税额"
                      extra={
                        suggested
                          ? '建议值，请以实际发票税额为准'
                          : source && source.sourceTaxRate === undefined
                            ? '来源税率未设置，请按发票填写税额'
                            : undefined
                      }
                      rules={[
                        required,
                        {
                          validator: async (_, value: string) => {
                            const amount = form.getFieldValue([
                              ...feePath,
                              'amount',
                            ]);
                            if (
                              !AMOUNT.test(value ?? '') ||
                              !amount ||
                              new Decimal(value).gt(amount)
                            )
                              throw new Error(
                                '税额须在0与本次含税金额之间，最多8位小数',
                              );
                          },
                        },
                      ]}
                    >
                      <Input
                        onChange={sourceSetter(form, [...feePath, 'taxSource'])}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={1}>
                    <Button
                      type="text"
                      danger
                      icon={<MinusCircleOutlined />}
                      aria-label="移除费用分配"
                      style={{ marginTop: 30 }}
                      onClick={() => {
                        removeFee(field.name);
                        update(recordIndex, (current) =>
                          syncAllocationTotal(current, name),
                        );
                      }}
                    />
                  </Col>
                </Row>
              );
            })}
            <Form.ErrorList errors={errors.errors} />
            <Button
              size="small"
              icon={<PlusOutlined />}
              disabled={!bill}
              onClick={() => add({ amountSource: 'auto', taxSource: 'auto' })}
            >
              添加费用分配
            </Button>
          </>
        )}
      </Form.List>
    </Card>
  );
}
