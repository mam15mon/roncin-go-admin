import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Row,
  Select,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import Decimal from 'decimal.js';
import { useAccess } from '@/app/access';
import { settlementServiceListInvoiceProfilesForParty } from '@/services/roncin/settlementService';
import { formatAmount, trimDecimal } from '@/utils/format';
import type {
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
} from './types';

type Name = (string | number)[];
type Props = {
  name: Name;
  bills: InvoiceRegistrationBillSource[];
  organizationId: string;
  settlementPartyId: string;
  profiles?: API.FinanceInvoiceProfileOption[];
  allowAdd?: boolean;
};
const required = { required: true, message: '请明确填写本项' };
const amountRule = {
  validator: async (_: unknown, value: string) => {
    if (
      !/^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/.test(value ?? '') ||
      !new Decimal(value).gt(0)
    )
      throw new Error('请输入大于0的金额，最多8位小数');
  },
};
const listRule = {
  validator: async (_: unknown, value: unknown[]) => {
    if (!value?.length) throw new Error('至少添加一项明确分配');
  },
};

function AllocationFields({
  path,
  name,
  bills,
  remove,
}: {
  path: Name;
  name: number;
  bills: InvoiceRegistrationBillSource[];
  remove: () => void;
}) {
  const form = Form.useFormInstance();
  const billID = Form.useWatch([...path, name, 'billId'], form);
  const bill = bills.find((item) => item.id === billID);
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
        <Col xs={24} md={16}>
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
              onChange={() => {
                form.setFieldValue([...path, name, 'fees'], []);
              }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item
            name={[name, 'amount']}
            label={`本账单含税分配（${bill?.currency ?? ''}）`}
            rules={[required, amountRule]}
          >
            <Input />
          </Form.Item>
        </Col>
      </Row>
      <Form.List name={[name, 'fees']} rules={[listRule]}>
        {(fields, { add, remove: removeFee }, errors) => (
          <>
            {fields.map((field) => (
              <Row key={field.key} gutter={8} align="middle">
                <Col xs={24} md={11}>
                  <Form.Item
                    name={[field.name, 'orderFeeId']}
                    label="来源费用及可登记余额"
                    rules={[required]}
                  >
                    <Select
                      options={(bill?.fees ?? []).map((fee) => ({
                        value: fee.orderFeeId,
                        label: `${fee.feeCode ?? ''} ${fee.feeName ?? ''} · ${formatAmount(fee.invoiceableAmount ?? fee.totalAmount)} · 来源税率 ${fee.sourceTaxRate === undefined ? '-' : `${trimDecimal(fee.sourceTaxRate)}%`}`,
                      }))}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={6}>
                  <Form.Item
                    name={[field.name, 'amount']}
                    label="本次含税金额"
                    rules={[required, amountRule]}
                  >
                    <Input />
                  </Form.Item>
                </Col>
                <Col xs={24} md={6}>
                  <Form.Item
                    name={[field.name, 'taxAmount']}
                    dependencies={[
                      [...path, name, 'fees', field.name, 'amount'],
                    ]}
                    label="本次登记税额（财务填写）"
                    rules={[
                      required,
                      {
                        validator: async (_, value: string) => {
                          const amount = form.getFieldValue([
                            ...path,
                            name,
                            'fees',
                            field.name,
                            'amount',
                          ]);
                          if (
                            !/^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/.test(
                              value ?? '',
                            ) ||
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
                    <Input />
                  </Form.Item>
                </Col>
                <Col xs={24} md={1}>
                  <Button
                    type="text"
                    danger
                    icon={<MinusCircleOutlined />}
                    aria-label="移除费用分配"
                    onClick={() => removeFee(field.name)}
                  />
                </Col>
              </Row>
            ))}
            <Form.ErrorList errors={errors.errors} />
            <Button
              size="small"
              icon={<PlusOutlined />}
              disabled={!bill}
              onClick={() => add({})}
            >
              添加费用分配
            </Button>
          </>
        )}
      </Form.List>
    </Card>
  );
}

/** 已有票据多条登记编辑器：独立登记与建账组共用，提交由所属事务入口处理。 */
export default function InvoiceRegistrationEditor({
  name,
  bills,
  organizationId,
  settlementPartyId,
  profiles,
  allowAdd = true,
}: Props) {
  const form = Form.useFormInstance();
  const records = Form.useWatch(name, form) as
    | InvoiceRegistrationDraft[]
    | undefined;
  const access = useAccess();
  const canQuery =
    access.canCreateFinanceInvoices &&
    access.canOperateOrganization(organizationId);
  const query = useQuery({
    queryKey: [
      'finance',
      'invoice-registration-profiles',
      organizationId,
      settlementPartyId,
    ],
    enabled:
      canQuery &&
      (profiles === undefined ||
        Boolean(records?.some((record) => !record.sourceInvoiceId))) &&
      Boolean(settlementPartyId),
    meta: { errorMessage: '加载登记抬头失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceListInvoiceProfilesForParty(
        { settlementPartyId },
        { signal, skipErrorHandler: true },
      );
      if (
        !response?.data ||
        response.success === false ||
        response.data.organizationId !== organizationId ||
        response.data.settlementPartyId !== settlementPartyId
      )
        throw new Error('登记抬头来源不正确');
      return response.data.data ?? [];
    },
  });
  const profileOptions = (items: API.FinanceInvoiceProfileOption[]) =>
    items.map((profile) => ({
      value: profile.id,
      label: `${profile.invoiceTitle ?? ''} · ${profile.taxpayerIdentificationNo ?? ''}`,
    }));
  return (
    <>
      <Alert
        type="info"
        showIcon
        title="登记已有发票。逐费用手填含税额和税额，费用合计须等于账单分配，账单合计须等于发票总额。"
        style={{ marginBottom: 12 }}
      />
      <Form.List
        name={name}
        rules={[
          {
            validator: async (_, rows: InvoiceRegistrationDraft[]) => {
              if ((rows?.length ?? 0) > 100)
                throw new Error('一次最多登记100张发票');
            },
          },
        ]}
      >
        {(fields, { add, remove }, errors) => (
          <>
            {fields.map((field) => (
              <Card
                key={field.key}
                size="small"
                title={`第${field.name + 1}张发票`}
                style={{ marginBottom: 12 }}
                extra={
                  <Button type="text" danger onClick={() => remove(field.name)}>
                    移除发票记录
                  </Button>
                }
              >
                <Form.Item name={[field.name, 'sourceInvoiceId']} hidden>
                  <Input />
                </Form.Item>
                <Row gutter={12}>
                  <Col xs={24} md={12}>
                    <Form.Item
                      name={[field.name, 'invoiceProfileId']}
                      label="开票抬头"
                      rules={[required]}
                    >
                      <Select
                        options={profileOptions(
                          records?.[field.name]?.sourceInvoiceId
                            ? (profiles ?? [])
                            : (query.data ?? []),
                        )}
                        loading={query.isFetching}
                        disabled={Boolean(
                          records?.[field.name]?.sourceInvoiceId,
                        )}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item
                      name={[field.name, 'invoiceType']}
                      label="发票类型"
                      rules={[required]}
                    >
                      <Select
                        options={[
                          { value: 'NORMAL', label: '普通发票' },
                          { value: 'SPECIAL', label: '专用发票' },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'taxInvoiceNo']}
                      label="已有发票号码"
                      rules={[required, { max: 100, whitespace: true }]}
                    >
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'invoiceDate']}
                      label="发票日期"
                      rules={[required]}
                    >
                      <DatePicker style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'totalAmount']}
                      label="发票含税总额"
                      rules={[required, amountRule]}
                    >
                      <Input />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.List
                  name={[field.name, 'allocations']}
                  rules={[listRule]}
                >
                  {(allocations, actions, allocationErrors) => (
                    <>
                      {allocations.map((allocation) => (
                        <AllocationFields
                          key={allocation.key}
                          name={allocation.name}
                          path={[...name, field.name, 'allocations']}
                          bills={bills}
                          remove={() => actions.remove(allocation.name)}
                        />
                      ))}
                      <Form.ErrorList errors={allocationErrors.errors} />
                      <Button
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() =>
                          actions.add({
                            billId:
                              bills.length === 1 ? bills[0].id : undefined,
                            fees: [],
                          })
                        }
                      >
                        添加账单分配
                      </Button>
                    </>
                  )}
                </Form.List>
                <Form.Item
                  name={[field.name, 'note']}
                  label="备注"
                  rules={[{ max: 500 }]}
                  style={{ marginTop: 12 }}
                >
                  <Input />
                </Form.Item>
              </Card>
            ))}
            <Form.ErrorList errors={errors.errors} />
            {allowAdd && (
              <Button
                icon={<PlusOutlined />}
                onClick={() =>
                  add({
                    invoiceDate: dayjs(),
                    invoiceType: 'NORMAL',
                    allocations: [],
                  })
                }
              >
                继续添加发票
              </Button>
            )}
          </>
        )}
      </Form.List>
      <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
        来源税率仅供追溯，登记税额不改写原费用。任一登记失败，当前提交全部回滚并保留输入。
      </Typography.Paragraph>
    </>
  );
}
