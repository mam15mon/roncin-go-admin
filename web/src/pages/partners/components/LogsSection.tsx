import React from 'react';
import { SectionCard } from '@/components/ui';
import AuditLogSection from './AuditLogSection';

type LogsSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  partnerId: string;
  roleLabel: string;
};

/** 操作记录分节：依赖已保存档案与读权限 */
export default function LogsSection({
  collapsed,
  onCollapseChange,
  partnerId,
  roleLabel,
}: LogsSectionProps) {
  return (
    <SectionCard
      key="logs"
      id="section-logs"
      sectionKey="logs"
      title="操作记录"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <AuditLogSection partnerId={partnerId} roleLabel={roleLabel} />
    </SectionCard>
  );
}
