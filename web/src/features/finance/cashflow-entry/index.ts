/** 收付登记与核销差额登记共用的支付方式候选和展示名称。 */
export const paymentMethodOptions = [
  { label: '银行转账', value: 'BANK_TRANSFER' },
  { label: '支票', value: 'CHECK' },
  { label: '现金', value: 'CASH' },
  { label: '第三方支付', value: 'THIRD_PARTY' },
];

export function getPaymentMethodLabel(paymentMethod: string | undefined) {
  return (
    paymentMethodOptions.find((option) => option.value === paymentMethod)
      ?.label ?? paymentMethod
  );
}
