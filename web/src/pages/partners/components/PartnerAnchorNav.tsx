import React from 'react';
import { FormAnchorNav, locateSectionError } from '@/components/ui';

type PartnerAnchorNavProps = {
  sectionErrors: Record<string, number>;
  isCreate: boolean;
  partnerId: string | undefined;
  canReadSettlementRules: boolean;
  canReadAccounts: boolean;
  canReadShippingPresets: boolean;
  canReadContracts: boolean;
  canReadAudit: boolean;
  roleLabel: string;
  onActiveCollapseKeysChange: React.Dispatch<React.SetStateAction<string[]>>;
  onCollapsedChange: (collapsed: boolean) => void;
};

/** 楼层大纲与错误定位导航 */
export default function PartnerAnchorNav({
  sectionErrors,
  isCreate,
  partnerId,
  canReadSettlementRules,
  canReadAccounts,
  canReadShippingPresets,
  canReadContracts,
  canReadAudit,
  roleLabel,
  onActiveCollapseKeysChange,
  onCollapsedChange,
}: PartnerAnchorNavProps) {
  return (
    <FormAnchorNav
      sectionErrors={sectionErrors}
      defaultCollapsed={true}
      onCollapsedChange={onCollapsedChange}
      items={[
        { key: 'basic', title: '基础信息' },
        ...(isCreate || canReadSettlementRules
          ? [{ key: 'settlement', title: '财务结算' }]
          : []),
        ...(partnerId && canReadAccounts
          ? [{ key: 'accounts', title: '账户信息' }]
          : []),
        { key: 'contacts', title: '联系方式' },
        ...(partnerId && canReadShippingPresets
          ? [{ key: 'presets', title: '常用信息' }]
          : []),
        ...(partnerId && canReadContracts
          ? [{ key: 'contracts', title: '合同管理' }]
          : []),
        { key: 'remark', title: `${roleLabel}备注` },
        ...(partnerId && canReadAudit
          ? [{ key: 'logs', title: '操作记录' }]
          : []),
      ]}
      onSelect={(key) => {
        onActiveCollapseKeysChange((prev) =>
          Array.from(new Set([...prev, key])),
        );
      }}
      onErrorClick={(sectionKey) => {
        // 先展开折叠分节，再等布局稳定后定位错误项（居中滚动）或分节标题
        // （按实测吸顶高度落位，避免被吸顶按钮栏遮挡）。
        onActiveCollapseKeysChange((prev) =>
          Array.from(new Set([...prev, sectionKey])),
        );
        void locateSectionError(sectionKey);
      }}
    />
  );
}
