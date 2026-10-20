import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import type { OrderKindDefinition } from '../types';
import AirExportDetailFeatures from './AirExportDetailFeatures';
import { airExportFormAdapter } from './form-adapter';

export const airExportDefinition: OrderKindDefinition = {
  kind: 'air-export',
  businessType: OrderBusinessType.BUSINESS_TYPE_AE,
  tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
  transportMode: 'air',
  title: '空运出口订单',
  navigationTitle: '空运出口',
  form: airExportFormAdapter,
  DetailFeatures: AirExportDetailFeatures,
};
