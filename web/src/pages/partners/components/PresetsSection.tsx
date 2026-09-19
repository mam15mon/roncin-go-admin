import React from 'react';
import { SectionCard } from '@/components/ui';
import ShippingPresetSection from './ShippingPresetSection';

type PresetsSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  partnerId: string;
  roleLabel: string;
  canCreate: boolean | undefined;
  canUpdate: boolean | undefined;
};

/** 常用信息分节：依赖已保存档案与读权限，新建模式不展示 */
export default function PresetsSection({
  collapsed,
  onCollapseChange,
  partnerId,
  roleLabel,
  canCreate,
  canUpdate,
}: PresetsSectionProps) {
  return (
    <SectionCard
      key="presets"
      id="section-presets"
      sectionKey="presets"
      title="常用信息"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <ShippingPresetSection
        partnerId={partnerId}
        roleLabel={roleLabel}
        canCreate={canCreate}
        canUpdate={canUpdate}
      />
    </SectionCard>
  );
}
