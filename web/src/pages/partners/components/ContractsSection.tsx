import React from 'react';
import { SectionCard } from '@/components/ui';
import ContractCardList from './ContractCardList';

type ContractsSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  partnerId: string;
  roleLabel: string;
  canCreate: boolean | undefined;
  canUpdate: boolean | undefined;
};

/** 合同管理分节：依赖已保存档案与读权限，新建模式不展示 */
export default function ContractsSection({
  collapsed,
  onCollapseChange,
  partnerId,
  roleLabel,
  canCreate,
  canUpdate,
}: ContractsSectionProps) {
  return (
    <SectionCard
      key="contracts"
      id="section-contracts"
      sectionKey="contracts"
      title="合同管理"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <ContractCardList
        partnerId={partnerId}
        roleLabel={roleLabel}
        canCreate={canCreate}
        canUpdate={canUpdate}
      />
    </SectionCard>
  );
}
