import type { OrderBusinessType, TradeDirection } from '@/enums.generated';
import type { TemplateProps, TemplateSection } from '../templates';

/** 订单类型的稳定路由标识，同时是注册表的唯一 key。 */
export type OrderKind = 'sea-export';

/** 运输方式只描述主数据装载与地点搜索，不承载订单类型分发。 */
export type OrderTransportMode = 'sea' | 'air' | 'land' | 'rail';

/** 品类表单适配入口：页面只通过当前定义取得分节、默认值与请求转换。 */
export interface OrderKindFormAdapter {
  buildSections(props: TemplateProps): TemplateSection[];
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
