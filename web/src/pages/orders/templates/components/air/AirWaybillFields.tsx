import {
  ProFormCheckbox,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert, Button, Collapse, Form, Space, Typography } from 'antd';
import type { NamePath } from 'antd/es/form/interface';
import { useEffect, useState } from 'react';
import { FormCell, FormRow, ProFormSearchableSelect } from '@/components/ui';
import { formatAmount } from '@/utils/format';
import { onSeaFormErrorReveal } from '../../sea-form-error-reveal';

type Path = (string | number)[];
type Props = { prefix: Path; disabled?: boolean };
type TextField = readonly [keyof API.AirWaybillContent, string, number?];

const parties: readonly TextField[] = [
  ['shipperAccount', '发货人账号'],
  ['consigneeAccount', '收货人账号'],
  ['shipperText', '发货人名称与地址', 5000],
  ['consigneeText', '收货人名称与地址', 5000],
  ['notifyPartyText', '通知人', 5000],
  ['secondNotifyPartyText', '第二通知人', 5000],
  ['agentText', '代理名称与地址', 5000],
  ['issuedBy', '印发人', 5000],
  ['agentIataCode', '代理 IATA 代码'],
  ['agentAccount', '代理账号'],
];
const routing: readonly TextField[] = [
  ['departureText', '起运机场印字'],
  ['routeTo', '第一程至'],
  ['firstCarrier', '第一承运人'],
  ['routeTo2', '第二程至'],
  ['carrier2', '第二承运人'],
  ['routeTo3', '第三程至'],
  ['carrier3', '第三承运人'],
  ['destinationText', '目的机场印字'],
  ['flightText', '航班印字'],
];
const information: readonly TextField[] = [
  ['accountingInformation', '账务信息', 5000],
  ['referenceNumber', '参考编号'],
  ['information1', '附加信息一'],
  ['information2', '附加信息二'],
  ['sci', '特别海关信息 SCI'],
  ['handlingInformation', '处理信息', 5000],
  ['marks', '唛头', 10000],
  ['goodsDescription', '运单品名', 10000],
  ['clauses', '附加条款', 5000],
  ['transportTerms', '运输条款'],
  ['otherChargesText', '其他费用说明', 5000],
];
const signatures: readonly TextField[] = [
  ['shipperCertification', '发货人声明', 5000],
  ['shipperSignature', '发货人签字'],
  ['carrierExecution', '承运人签发声明', 5000],
  ['issuePlace', '签发地点'],
  ['carrierSignature', '承运人签字'],
  ['remarks', '单证备注', 5000],
];
const paymentOptions = [
  { label: '预付 PPD', value: 'PPD' },
  { label: '到付 COLL', value: 'COLL' },
];
const decimalPattern = /^(0|[1-9]\d{0,15})(\.\d{1,8})?$/;
const commonPartyKeys = new Set<keyof API.AirWaybillContent>([
  'shipperText',
  'consigneeText',
  'notifyPartyText',
  'secondNotifyPartyText',
  'agentText',
]);
const commonCargoKeys = new Set<keyof API.AirWaybillContent>([
  'marks',
  'goodsDescription',
]);

function fieldSection(field: string): string {
  if (parties.some(([key]) => key === field)) return 'parties';
  if (routing.some(([key]) => key === field) || field === 'flightDate')
    return 'routing';
  if (information.some(([key]) => key === field)) return 'information';
  if (signatures.some(([key]) => key === field) || field === 'issueDate')
    return 'signatures';
  if (field === 'rateLines' || field === 'asArranged') return 'rates';
  if (
    [
      'currency',
      'chargeCode',
      'weightPayment',
      'otherPayment',
      'carriageValue',
      'customsValue',
      'insuranceValue',
    ].includes(field)
  )
    return 'declarations';
  return 'charges';
}

function DecimalField({
  name,
  label,
  disabled,
  required,
  integer,
}: {
  name: NamePath;
  label: string;
  disabled?: boolean;
  required?: boolean;
  integer?: boolean;
}) {
  return (
    <ProFormText
      name={name}
      label={label}
      disabled={disabled}
      rules={[
        { required, message: `请输入${label}` },
        {
          pattern: integer ? /^(0|[1-9]\d{0,15})$/ : decimalPattern,
          message: integer ? '请输入非负整数' : '请输入非负数，最多 8 位小数',
        },
      ]}
      fieldProps={{ inputMode: 'decimal', maxLength: 25 }}
    />
  );
}

function TextFields({
  prefix,
  disabled,
  fields,
  dense = false,
}: Props & { fields: readonly TextField[]; dense?: boolean }) {
  return (
    <FormRow cols={dense ? 6 : 2} style={dense ? { rowGap: 4 } : undefined}>
      {fields.map(([key, label, maxLength = 200]) => {
        const field =
          maxLength > 200 ? (
            <ProFormTextArea
              key={key}
              name={[...prefix, key]}
              label={label}
              disabled={disabled}
              fieldProps={{
                maxLength,
                ...(dense
                  ? { rows: 2 }
                  : { autoSize: { minRows: 2, maxRows: 8 } }),
              }}
            />
          ) : (
            <ProFormText
              key={key}
              name={[...prefix, key]}
              label={label}
              disabled={disabled}
              fieldProps={{ maxLength }}
            />
          );
        return dense ? (
          <FormCell key={key} span={maxLength > 200 ? 3 : 2}>
            {field}
          </FormCell>
        ) : (
          field
        );
      })}
    </FormRow>
  );
}

function DeclaredValueFields({
  prefix,
  disabled,
  field,
  token,
  label,
}: Props & {
  field: 'carriageValue' | 'customsValue' | 'insuranceValue';
  token: string;
  label: string;
}) {
  const form = Form.useFormInstance();
  const kind = Form.useWatch([...prefix, field, 'kind'], {
    form,
    preserve: true,
  });
  return (
    <div>
      <ProFormSearchableSelect
        name={[...prefix, field, 'kind']}
        label={label}
        disabled={disabled}
        options={[
          { label: '明确数值', value: 'AMOUNT' },
          { label: token, value: token },
        ]}
        fieldProps={{
          onChange: (value) => {
            if (value !== 'AMOUNT')
              form.setFieldValue([...prefix, field, 'amount'], undefined);
          },
        }}
      />
      {kind === 'AMOUNT' && (
        <DecimalField
          name={[...prefix, field, 'amount']}
          label={`${label}数值`}
          disabled={disabled}
          required
        />
      )}
    </div>
  );
}

function RateLines({ prefix, disabled }: Props) {
  return (
    <Form.List name={[...prefix, 'rateLines']}>
      {(fields, { add, remove }) => (
        <Space orientation="vertical" style={{ width: '100%' }} size={12}>
          {fields.map((field, index) => (
            <div key={field.key}>
              <Space style={{ marginBottom: 8 }}>
                <Typography.Text strong>运价行 {index + 1}</Typography.Text>
                {!disabled && (
                  <Button type="link" danger onClick={() => remove(field.name)}>
                    删除运价行 {index + 1}
                  </Button>
                )}
              </Space>
              <FormRow cols={4}>
                <DecimalField
                  name={[field.name, 'pieces']}
                  label="件数"
                  integer
                  disabled={disabled}
                />
                <DecimalField
                  name={[field.name, 'grossWeightKg']}
                  label="毛重 KG"
                  disabled={disabled}
                />
                <ProFormText
                  name={[field.name, 'rateClass']}
                  label="运价等级"
                  disabled={disabled}
                  fieldProps={{ maxLength: 20 }}
                />
                <ProFormText
                  name={[field.name, 'commodityItemNo']}
                  label="商品项目号"
                  disabled={disabled}
                  fieldProps={{ maxLength: 100 }}
                />
                <DecimalField
                  name={[field.name, 'chargeableWeightKg']}
                  label="计费重 KG"
                  disabled={disabled}
                />
                <DecimalField
                  name={[field.name, 'rate']}
                  label="费率"
                  disabled={disabled}
                />
                <DecimalField
                  name={[field.name, 'amount']}
                  label="运价行金额"
                  disabled={disabled}
                />
              </FormRow>
              <ProFormTextArea
                name={[field.name, 'goodsDescription']}
                label="该行品名"
                disabled={disabled}
                fieldProps={{
                  maxLength: 10000,
                  autoSize: { minRows: 2, maxRows: 6 },
                }}
              />
            </div>
          ))}
          {!disabled && (
            <Button onClick={() => add({})} disabled={fields.length >= 200}>
              添加运价行
            </Button>
          )}
        </Space>
      )}
    </Form.List>
  );
}

function OtherCharges({ prefix, disabled }: Props) {
  return (
    <Form.List name={[...prefix, 'otherCharges']}>
      {(fields, { add, remove }) => (
        <Space orientation="vertical" style={{ width: '100%' }} size={12}>
          {fields.map((field, index) => (
            <div key={field.key}>
              <Space style={{ marginBottom: 8 }}>
                <Typography.Text strong>其他费用 {index + 1}</Typography.Text>
                {!disabled && (
                  <Button type="link" danger onClick={() => remove(field.name)}>
                    删除其他费用 {index + 1}
                  </Button>
                )}
              </Space>
              <FormRow cols={4}>
                <ProFormText
                  name={[field.name, 'name']}
                  label="费用名称"
                  disabled={disabled}
                  fieldProps={{ maxLength: 200 }}
                />
                <ProFormText
                  name={[field.name, 'billingUnit']}
                  label="计费单位"
                  disabled={disabled}
                  fieldProps={{ maxLength: 100 }}
                />
                <DecimalField
                  name={[field.name, 'quantity']}
                  label="数量"
                  disabled={disabled}
                />
                <DecimalField
                  name={[field.name, 'unitPrice']}
                  label="单价"
                  disabled={disabled}
                />
                <ProFormText
                  name={[field.name, 'currency']}
                  label="费用币种"
                  disabled={disabled}
                  rules={[
                    {
                      required: true,
                      pattern: /^[A-Z]{3}$/,
                      message: '请输入与运单一致的三位大写币种',
                    },
                  ]}
                  fieldProps={{ maxLength: 3 }}
                />
                <DecimalField
                  name={[field.name, 'amount']}
                  label="其他费用金额"
                  disabled={disabled}
                  required
                />
                <ProFormSearchableSelect
                  name={[field.name, 'dueTo']}
                  label="归属"
                  disabled={disabled}
                  rules={[{ required: true, message: '请选择费用归属' }]}
                  options={[
                    { label: '代理', value: 'AGENT' },
                    { label: '承运人', value: 'CARRIER' },
                  ]}
                />
                <ProFormSearchableSelect
                  name={[field.name, 'payment']}
                  label="费用付款"
                  disabled={disabled}
                  rules={[{ required: true, message: '请选择付款方式' }]}
                  options={paymentOptions}
                />
              </FormRow>
            </div>
          ))}
          {!disabled && (
            <Button onClick={() => add({})} disabled={fields.length >= 200}>
              添加其他费用
            </Button>
          )}
        </Space>
      )}
    </Form.List>
  );
}

/** MAWB/HAWB 共用字段定义，分别绑定自己的正文路径，不读取或改写运输、货量和经营费用。 */
export function AirWaybillFields({
  prefix,
  disabled,
  variant = 'full',
}: Props & { variant?: 'full' | 'order' }) {
  const form = Form.useFormInstance();
  const value = Form.useWatch(prefix, { form, preserve: true }) as
    | API.AirWaybillContent
    | undefined;
  const compact = variant === 'order';
  const textProps = { prefix, disabled, dense: compact };
  const [openSections, setOpenSections] = useState<string[]>([]);
  const prefixKey = JSON.stringify(prefix);
  useEffect(
    () =>
      onSeaFormErrorReveal((fields) => {
        const path = JSON.parse(prefixKey) as Path;
        const name = fields[0]?.name;
        if (!name || !path.every((part, index) => part === name[index])) return;
        const section = fieldSection(String(name[path.length]));
        setOpenSections((current) =>
          current.includes(section) ? current : [...current, section],
        );
      }),
    [prefixKey],
  );
  return (
    <>
      <Form.Item name={prefix} hidden>
        <span />
      </Form.Item>
      {compact && (
        <>
          <TextFields
            {...textProps}
            fields={parties.filter(([key]) => commonPartyKeys.has(key))}
          />
          <TextFields
            {...textProps}
            fields={information.filter(([key]) => commonCargoKeys.has(key))}
          />
        </>
      )}
      <Collapse
        size="small"
        activeKey={openSections}
        onChange={(keys) =>
          setOpenSections(
            Array.isArray(keys) ? keys.map(String) : [String(keys)],
          )
        }
        items={[
          {
            key: 'parties',
            label: '收发货人与代理',
            children: (
              <TextFields
                {...textProps}
                fields={
                  compact
                    ? parties.filter(([key]) => !commonPartyKeys.has(key))
                    : parties
                }
              />
            ),
          },
          {
            key: 'routing',
            label: '路线与航班印字',
            children: (
              <>
                <Typography.Paragraph type="secondary">
                  以下为运单印字，实际运输机场、航班及时间在共同运输资料中维护。
                </Typography.Paragraph>
                <TextFields {...textProps} fields={routing} />
                <ProFormText
                  name={[...prefix, 'flightDate']}
                  label="航班日期"
                  disabled={disabled}
                  fieldProps={{ type: 'date' }}
                />
              </>
            ),
          },
          {
            key: 'information',
            label: '账务、处理与货物印字',
            children: (
              <TextFields
                {...textProps}
                fields={
                  compact
                    ? information.filter(([key]) => !commonCargoKeys.has(key))
                    : information
                }
              />
            ),
          },
          {
            key: 'declarations',
            label: '币种、声明值与付款',
            children: (
              <>
                <FormRow cols={4}>
                  <ProFormText
                    name={[...prefix, 'currency']}
                    label="运单币种"
                    disabled={disabled}
                    rules={[
                      {
                        pattern: /^[A-Z]{3}$/,
                        message: '请输入三位大写币种，如 CNY',
                      },
                    ]}
                    fieldProps={{ maxLength: 3 }}
                  />
                  <ProFormText
                    name={[...prefix, 'chargeCode']}
                    label="费用代码"
                    disabled={disabled}
                    fieldProps={{ maxLength: 200 }}
                  />
                  <ProFormSearchableSelect
                    name={[...prefix, 'weightPayment']}
                    label="重量费用付款"
                    disabled={disabled}
                    options={paymentOptions}
                  />
                  <ProFormSearchableSelect
                    name={[...prefix, 'otherPayment']}
                    label="其他费用付款"
                    disabled={disabled}
                    options={paymentOptions}
                  />
                </FormRow>
                <FormRow cols={3}>
                  <DeclaredValueFields
                    {...textProps}
                    field="carriageValue"
                    token="NVD"
                    label="运输声明价值"
                  />
                  <DeclaredValueFields
                    {...textProps}
                    field="customsValue"
                    token="NCV"
                    label="海关声明价值"
                  />
                  <DeclaredValueFields
                    {...textProps}
                    field="insuranceValue"
                    token="XXX"
                    label="保险声明"
                  />
                </FormRow>
              </>
            ),
          },
          {
            key: 'rates',
            label: '运价明细',
            children: (
              <>
                <ProFormCheckbox
                  name={[...prefix, 'asArranged']}
                  disabled={disabled}
                >
                  AS ARRANGED（运费按约定，不记为零）
                </ProFormCheckbox>
                <Alert
                  type="info"
                  showIcon
                  title="金额按各行明确录入"
                  description="费率与计费重分别保留；系统按各行金额汇总，不自动报价。填写运价行时须明确运单币种和重量费用付款。"
                  style={{ marginBottom: 12 }}
                />
                <RateLines {...textProps} />
              </>
            ),
          },
          {
            key: 'charges',
            label: '其他费用与汇总',
            children: (
              <>
                <OtherCharges {...textProps} />
                <FormRow cols={4}>
                  {(['prepaid', 'collect'] as const).flatMap((payment) => [
                    <DecimalField
                      key={`${payment}-valuation`}
                      name={[...prefix, payment, 'valuationCharge']}
                      label={`${payment === 'prepaid' ? '预付' : '到付'}估价费`}
                      disabled={disabled}
                    />,
                    <DecimalField
                      key={`${payment}-tax`}
                      name={[...prefix, payment, 'tax']}
                      label={`${payment === 'prepaid' ? '预付' : '到付'}税额`}
                      disabled={disabled}
                    />,
                  ])}
                </FormRow>
                <Typography.Paragraph type="secondary">
                  保存时按币种与付款栏计算运费、代理/承运人其他费用及合计；未确定运费与
                  AS ARRANGED 不视为零。上次保存合计：预付{' '}
                  {formatAmount(value?.prepaid?.total || undefined)} / 到付{' '}
                  {formatAmount(value?.collect?.total || undefined)}。
                </Typography.Paragraph>
                <FormRow cols={3}>
                  <ProFormText
                    name={[...prefix, 'destinationCurrency']}
                    label="目的地币种"
                    disabled={disabled}
                    rules={[
                      { pattern: /^[A-Z]{3}$/, message: '请输入三位大写币种' },
                    ]}
                    fieldProps={{ maxLength: 3 }}
                  />
                  {(
                    [
                      'currencyConversionRate',
                      'destinationCollectCharge',
                      'chargesAtDestination',
                      'totalCollect',
                    ] as const
                  ).map((key, index) => (
                    <DecimalField
                      key={key}
                      name={[...prefix, key]}
                      label={
                        [
                          '目的地换算汇率',
                          '目的地到付费',
                          '目的地费用',
                          '到付总额',
                        ][index]
                      }
                      disabled={disabled}
                    />
                  ))}
                </FormRow>
              </>
            ),
          },
          {
            key: 'signatures',
            label: '签发与备注',
            children: (
              <>
                <ProFormText
                  name={[...prefix, 'issueDate']}
                  label="签发日期"
                  disabled={disabled}
                  fieldProps={{ type: 'date' }}
                />
                <TextFields {...textProps} fields={signatures} />
              </>
            ),
          },
        ]}
      />
    </>
  );
}
