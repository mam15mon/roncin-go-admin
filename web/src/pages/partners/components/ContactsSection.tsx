import React from 'react';
import { SectionCard } from '@/components/ui';
import ContactCardList, { type ContactItem } from './ContactCardList';

type ContactsSectionProps = {
  collapsed: boolean;
  onCollapseChange: (collapsed: boolean) => void;
  contacts: ContactItem[];
  onChange: React.Dispatch<React.SetStateAction<ContactItem[]>>;
};

/** 联系方式分节 */
export default function ContactsSection({
  collapsed,
  onCollapseChange,
  contacts,
  onChange,
}: ContactsSectionProps) {
  return (
    <SectionCard
      key="contacts"
      id="section-contacts"
      sectionKey="contacts"
      title="联系方式"
      collapsible
      collapsed={collapsed}
      onCollapseChange={onCollapseChange}
    >
      <ContactCardList contacts={contacts} onChange={onChange} />
    </SectionCard>
  );
}
