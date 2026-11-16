import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import { ORDER_BUSINESS_ROUTES } from '@/features/orders/navigation';
import AirExportDetailFeatures from '../air-export/AirExportDetailFeatures';
import {
  airExportFormAdapter,
  buildAirCreatePayload,
} from '../air-export/form-adapter';
import type { OrderKindDefinition } from '../types';

/** AI 与 AE 共用航空实体和编辑映射，本票进口服务独立维护。 */
export const airImportDefinition: OrderKindDefinition = {
  kind: ORDER_BUSINESS_ROUTES[OrderBusinessType.BUSINESS_TYPE_AI],
  businessType: OrderBusinessType.BUSINESS_TYPE_AI,
  tradeDirection: TradeDirection.TRADE_DIRECTION_IMPORT,
  transportMode: 'air',
  title: '空运进口订单',
  navigationTitle: '空运进口',
  form: {
    ...airExportFormAdapter,
    buildCreatePayload: (values) =>
      buildAirCreatePayload(
        values,
        OrderBusinessType.BUSINESS_TYPE_AI,
        TradeDirection.TRADE_DIRECTION_IMPORT,
      ),
  },
  DetailFeatures: AirExportDetailFeatures,
};
