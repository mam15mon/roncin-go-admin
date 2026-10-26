import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';

export interface SelectOption {
  label: string;
  value: string | number;
  code?: string;
}

/** 区块结构由订单表单模板统一定义，页面模板只负责组装字段内容。 */
export type TemplateSection = OrderFormTemplateSection;

/** 连续新增分单只携带来源共享身份，航程及版本由最新候选确认。 */
export interface SeaHouseOrderCreateSource {
  orderId: string;
  orderNo?: string;
  masterBillId: string;
  masterNo: string;
  shippingLineId: string;
  shippingLineName?: string;
  transportExecutionId: string;
}

export interface TemplateProps {
  cargoCategoryOptions: SelectOption[];
  locationOptions: SelectOption[];
  searchLocations: (keyword?: string) => Promise<SelectOption[]>;
  currencyOptions: SelectOption[];
  containerSpecOptions: SelectOption[];
  searchCustomers: (keyword?: string) => Promise<SelectOption[]>;
  searchShippingLines: (keyword?: string) => Promise<SelectOption[]>;
  searchBookingAgents: (keyword?: string) => Promise<SelectOption[]>;
  searchForeignAgents: (keyword?: string) => Promise<SelectOption[]>;
  searchShippingAgents: (keyword?: string) => Promise<SelectOption[]>;
  setCustomerCode: (code?: string) => void;
  personnelOptions: API.OrderPersonnelOption[];
  /** 无人员候选读取能力时，详情中的人员字段保持只读。 */
  personnelReadonly?: boolean;
  creator?: {
    userId: string;
    displayName: string;
  };
  /** 是否为详情查看/编辑模式：详情页中不展示订单编号时间选择框，且创建人永久锁定不可变 */
  isDetail?: boolean;
  /** 当前订单归属，供独立单证命令校验工作台能力。 */
  organizationId?: string;
  /** 新建分单的主单及指定航程来源。 */
  seaHouseOrderCreateSource?: SeaHouseOrderCreateSource;
  /** 详情页独立新建分单入口的来源订单身份。 */
  seaHouseOrderSourceOrderId?: string;
  /** 详情页业务写入口是否因锁单等状态关闭。 */
  readonly?: boolean;
  /** 单证命令成功后刷新订单聚合，确保 expectedOrderVersion 使用最新值。 */
  onOrderDataChanged?: () => Promise<void> | void;
}
