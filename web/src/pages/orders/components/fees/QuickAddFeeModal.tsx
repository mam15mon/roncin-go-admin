import { FeeSettingQuickCreateModal } from '@/features/finance/fee-catalog';

type QuickAddFeeModalProps = {
  open: boolean;
  onCancel: () => void;
  onSuccess: (newFeeSetting: API.FeeSetting) => void;
  currencies: API.Currency[];
  billingUnits: API.BillingUnit[];
  taxableServices: API.TaxableService[];
};

/** 订单费用入口沿用同一公司费用科目快建表单。 */
export default function QuickAddFeeModal(props: QuickAddFeeModalProps) {
  return <FeeSettingQuickCreateModal {...props} />;
}
