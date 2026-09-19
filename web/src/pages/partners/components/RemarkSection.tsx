import { ProFormTextArea } from '@ant-design/pro-components';
import React from 'react';
import { SectionCard } from '@/components/ui';

type RemarkSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  roleLabel: string;
};

/** 备注分节 */
export default function RemarkSection({
  collapsed,
  onCollapseChange,
  roleLabel,
}: RemarkSectionProps) {
  return (
    <SectionCard
      key="remark"
      id="section-remark"
      sectionKey="remark"
      title={`${roleLabel}备注`}
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <ProFormTextArea
        name="remark"
        placeholder={`可以添加${roleLabel}信息录入时的备注信息`}
        fieldProps={{ rows: 3 }}
      />
    </SectionCard>
  );
}
