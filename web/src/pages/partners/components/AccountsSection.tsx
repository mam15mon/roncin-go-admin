import React from 'react';
import { SectionCard } from '@/components/ui';
import AccountsPanel from './secondary/AccountsPanel';

type AccountsSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  partner: API.Partner | undefined;
  canRead: boolean;
  canCreate: boolean;
  canUpdate: boolean;
};

/** 账户信息分节：依赖已保存档案与读权限，新建模式不展示 */
export default function AccountsSection({
  collapsed,
  onCollapseChange,
  partner,
  canRead,
  canCreate,
  canUpdate,
}: AccountsSectionProps) {
  return (
    <SectionCard
      key="accounts"
      id="section-accounts"
      sectionKey="accounts"
      title="账户信息"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <AccountsPanel
        partner={partner}
        canRead={canRead}
        canCreate={canCreate}
        canUpdate={canUpdate}
      />
    </SectionCard>
  );
}
