import type { MenuProps } from 'antd';
import type React from 'react';
import type { OrderBusinessType, TradeDirection } from '@/enums.generated';
import type { OrderPermissionOperation } from '@/permissions.generated';
import type { OrderRecordTab } from '../components/detail/OrderAuditTimelineSection';
import type {
  SelectOption,
  TemplateProps,
  TemplateSection,
} from '../templates';
import type {
  CreateOrderFormValues,
  OrderDetailFormValues,
} from './sea-export/form-adapter';

/** 订单类型的稳定路由标识，同时是注册表的唯一 key。 */
export type OrderKind = 'sea-export';

/** 运输方式只描述主数据装载与地点搜索，不承载订单类型分发。 */
export type OrderTransportMode = 'sea' | 'air' | 'land' | 'rail';

/** 新建默认值上下文：默认值只读显式输入，不读取全局 initialState 或表单 ref。 */
export interface OrderCreateDefaultsContext {
  creator?: TemplateProps['creator'];
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
}

/** 品类表单适配入口：页面只通过当前定义取得分节、默认值与请求转换。 */
export interface OrderKindFormAdapter {
  buildSections(props: TemplateProps): TemplateSection[];
  buildCreateDefaults(
    context: OrderCreateDefaultsContext,
  ): Partial<CreateOrderFormValues>;
  buildCreatePayload(values: CreateOrderFormValues): API.CreateOrderRequest;
  buildDetailInitialValues(
    order: API.Order | undefined,
    shippingDocs: API.OrderShippingDocument[],
    personnel: API.OrderPersonnel[],
  ): Partial<OrderDetailFormValues>;
  buildUpdatePayload(
    orderId: string,
    expectedVersion: string,
    values: OrderDetailFormValues,
  ): API.UpdateOrderRequest;
}

/** 订单类型注册项：稳定元数据与表单适配入口的唯一真相。 */
export interface OrderKindDefinition {
  readonly kind: OrderKind;
  readonly businessType: OrderBusinessType;
  readonly tradeDirection: TradeDirection;
  readonly transportMode: OrderTransportMode;
  readonly title: string;
  readonly navigationTitle: string;
  readonly form: OrderKindFormAdapter;
  /** 有状态的类型专属详情扩展；未提供时详情页只渲染通用布局。 */
  readonly DetailFeatures?: React.ComponentType<OrderDetailFeaturesProps>;
}

/** 详情扩展只接收跨边界必需的稳定业务输入与命令，不暴露草稿与显式刷新令牌。 */
export interface OrderDetailFeaturesContext {
  kind: OrderKind;
  orderId: string;
  order: API.Order;
  /** 与通用详情一致的表单身份（`${kind}:${orderId}`），用于扩展内请求门禁。 */
  orderFormIdentity: string;
  businessWritesDisabled: boolean;
  businessWriteBlockedReason?: string;
  /** 已绑定当前业务类型的订单操作权限判断。 */
  canOrder: (operation: OrderPermissionOperation) => boolean;
  searchShippingLines: (keyword?: string) => Promise<SelectOption[]>;
  searchLocations: (keyword?: string) => Promise<SelectOption[]>;
  containerSpecOptions: SelectOption[];
  /** 业务写成功后的普通快照刷新；不清草稿、不触碰显式刷新令牌。 */
  refreshOrderAndLock: () => Promise<void>;
  /** 复用通用锁单提示的写入口校验。 */
  ensureBusinessWriteAllowed: () => boolean;
}

/** 类型扩展向通用详情布局贡献的 UI 描述与刷新命令。 */
export interface OrderDetailFeatureContribution {
  /** 插入「异常情况」与「更多操作」之间的头部动作。 */
  headerActions?: React.ReactNode;
  /** 置于通用更多菜单项之前的菜单项。 */
  moreMenuItems?: MenuProps['items'];
  /** 追加到「关联与记录」卡片操作记录之后的记录页签。 */
  appendTabs?: OrderRecordTab[];
  /** 与通用费用、异常、放货面板并列挂载的覆盖层。 */
  overlays?: React.ReactNode;
  /** 锁状态同步或类型专属成功回调使用的普通刷新命令。 */
  refreshTypeState?: () => Promise<void>;
}

export interface OrderDetailFeaturesProps {
  context: OrderDetailFeaturesContext;
  children: (contribution: OrderDetailFeatureContribution) => React.ReactNode;
}
