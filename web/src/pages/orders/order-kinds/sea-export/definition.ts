import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import SeaExportDetailFeatures from './SeaExportDetailFeatures';
import { seaExportFormAdapter } from './form-adapter';
import type { OrderKindDefinition } from '../types';

/** 海运出口（SE）注册定义：当前唯一已开放的订单类型。 */
export const seaExportDefinition: OrderKindDefinition = {
  kind: 'sea-export',
  businessType: OrderBusinessType.BUSINESS_TYPE_SE,
  tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
  transportMode: 'sea',
  title: '海运出口订单',
  navigationTitle: '海运出口',
  form: seaExportFormAdapter,
  DetailFeatures: SeaExportDetailFeatures,
};
