import { SectionCard } from '@/components/ui';
import PaymentTermsFields from './PaymentTermsFields';

type SettlementSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  baseCurrency?: string;
  requireCreditLimit?: boolean;
  disabled?: boolean;
};

export default function SettlementSection({
  collapsed,
  onCollapseChange,
  ...fieldProps
}: SettlementSectionProps) {
  return (
    <SectionCard
      id="section-settlement"
      sectionKey="settlement"
      title="收款约定"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <PaymentTermsFields {...fieldProps} />
    </SectionCard>
  );
}
