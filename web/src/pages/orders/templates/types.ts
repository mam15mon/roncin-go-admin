import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';

export interface SelectOption {
  label: string;
  value: string | number;
  code?: string;
}

/** 区块结构由订单表单模板统一定义，页面模板只负责组装字段内容。 */
export type TemplateSection = OrderFormTemplateSection;

export interface TemplateProps {
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
  locationOptions: SelectOption[];
  searchLocations: (keyword?: string) => Promise<SelectOption[]>;
  currencyOptions: SelectOption[];
  containerSpecOptions: SelectOption[];
  searchCustomers: (keyword?: string) => Promise<SelectOption[]>;
  searchCarriers: (keyword?: string) => Promise<SelectOption[]>;
  searchBookingAgents: (keyword?: string) => Promise<SelectOption[]>;
  searchForeignAgents: (keyword?: string) => Promise<SelectOption[]>;
  searchShippingAgents: (keyword?: string) => Promise<SelectOption[]>;
  searchIssuers?: (keyword?: string) => Promise<SelectOption[]>;
  setCustomerCode: (code?: string) => void;
  checkCustomerReferenceNo: () => Promise<void>;
  checkInternalReferenceNo: () => Promise<void>;
  personnelOptions: API.OrderPersonnelOption[];
  creator?: {
    userId: string;
    displayName: string;
    organizationId: string;
    organizationName: string;
  };
  /** 是否为详情查看/编辑模式：详情页中不展示订单编号时间选择框，且创建人永久锁定不可变 */
  isDetail?: boolean;
}
