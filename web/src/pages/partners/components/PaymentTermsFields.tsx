import { ProFormDependency, ProFormDigit } from '@ant-design/pro-components';
import { Form, Radio } from 'antd';
import { creditLimitFields, hasInvalidCreditCurrency } from '../creditLimit';

export type PaymentTermsFormValues = {
  paymentTermsMode: 'none' | 'cash' | 'days';
  paymentTermsDays?: number;
  creditLimit?: number | string | null;
};

/** optional 账期保留未约定、现结和天数的存在性。 */
export function paymentTermsToForm(
  rule?: API.PartnerSettlementRule | null,
  baseCurrency?: string,
): PaymentTermsFormValues {
  const days = rule?.paymentTermsDays;
  return {
    paymentTermsMode: days == null ? 'none' : days === 0 ? 'cash' : 'days',
    paymentTermsDays: days != null && days > 0 ? days : undefined,
    creditLimit:
      rule?.creditLimitMinor != null &&
      !hasInvalidCreditCurrency(rule, baseCurrency)
        ? Number(rule.creditLimitMinor) / 100
        : undefined,
  };
}

export function paymentTermsFromForm(
  values: PaymentTermsFormValues,
  baseCurrency?: string,
): API.PartnerSettlementRuleInput {
  const credit = creditLimitFields(values.creditLimit, baseCurrency);
  if (values.paymentTermsMode === 'none') return credit;
  if (values.paymentTermsMode === 'cash') {
    return { ...credit, paymentTermsDays: 0 };
  }
  const days = values.paymentTermsDays;
  if (
    values.paymentTermsMode !== 'days' ||
    days == null ||
    !Number.isInteger(days) ||
    days < 1 ||
    days > 3650
  ) {
    throw new Error('请填写 1–3650 的整数天数');
  }
  return { ...credit, paymentTermsDays: days };
}

type PaymentTermsFieldsProps = {
  baseCurrency?: string;
  requireCreditLimit?: boolean;
  disabled?: boolean;
};

export default function PaymentTermsFields({
  baseCurrency,
  requireCreditLimit,
  disabled,
}: PaymentTermsFieldsProps) {
  return (
    <>
      <Form.Item
        name="paymentTermsMode"
        label="收款期限"
        rules={
          disabled ? undefined : [{ required: true, message: '请选择收款期限' }]
        }
      >
        <Radio.Group
          disabled={disabled}
          options={[
            { label: '未约定', value: 'none' },
            { label: '现结', value: 'cash' },
            { label: '建账后 N 天', value: 'days' },
          ]}
        />
      </Form.Item>
      <ProFormDependency name={['paymentTermsMode']}>
        {({ paymentTermsMode }) =>
          paymentTermsMode === 'days' ? (
            <ProFormDigit
              name="paymentTermsDays"
              label="天数"
              disabled={disabled}
              min={1}
              max={3650}
              fieldProps={{ precision: 0 }}
              rules={
                disabled
                  ? undefined
                  : [{ required: true, message: '请填写收款天数' }]
              }
            />
          ) : null
        }
      </ProFormDependency>
      <ProFormDigit
        name="creditLimit"
        label={`信用额度（${baseCurrency || '本币'}）`}
        disabled={disabled}
        min={0}
        fieldProps={{ precision: 2 }}
        placeholder="未约定额度"
        tooltip="公司本币的最大未核销应收额度；留空或填 0 表示不设额度"
        rules={
          requireCreditLimit && !disabled
            ? [
                {
                  required: true,
                  message: '请明确重新填写本币额度，填 0 表示不设额度',
                },
              ]
            : undefined
        }
      />
    </>
  );
}
