import type { OrderBusinessType, TradeDirection } from '@/enums.generated';
import type { SelectOption, TemplateProps, TemplateSection } from '../templates';
import type { CreateOrderFormValues, OrderDetailFormValues } from './sea-export/form-adapter';

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
}
