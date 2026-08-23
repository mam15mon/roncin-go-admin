import type { ReactNode } from 'react';

export interface SelectOption {
  label: string;
  value: string | number;
}

export interface TemplateSection {
  key: string;
  title: string;
  content: ReactNode;
}

export interface TemplateProps {
  statusTemplateOptions: SelectOption[];
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
  locationOptions: SelectOption[];
  searchCustomers: (keyword?: string) => Promise<SelectOption[]>;
  searchCarriers: (keyword?: string) => Promise<SelectOption[]>;
  searchBookingAgents: (keyword?: string) => Promise<SelectOption[]>;
}
