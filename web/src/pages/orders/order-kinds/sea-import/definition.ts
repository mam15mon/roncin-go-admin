import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import { getSeaTemplateSections } from '../../templates';
import {
  buildSeaCreatePayload,
  buildSeaExportCreateDefaults,
  buildSeaExportDetailInitialValues,
  buildSeaExportUpdatePayload,
} from '../sea-export/form-adapter';
import SeaExportDetailFeatures from '../sea-export/SeaExportDetailFeatures';
import type { OrderKindDefinition } from '../types';

/** SI 复用真实海运实体与字段；进口服务进度由独立业务区维护。 */
export const seaImportDefinition: OrderKindDefinition = {
  kind: 'sea-import',
  businessType: OrderBusinessType.BUSINESS_TYPE_SI,
  tradeDirection: TradeDirection.TRADE_DIRECTION_IMPORT,
  transportMode: 'sea',
  title: '海运进口订单',
  navigationTitle: '海运进口',
  form: {
    buildSections: getSeaTemplateSections,
    buildCreateDefaults: (context) => ({
      ...buildSeaExportCreateDefaults(context),
      // 进口外部分单的主体必须明确选择，不能自动以当前公司签发。
      seaHouseBill: undefined,
    }),
    buildCreatePayload: (values) =>
      buildSeaCreatePayload(
        values,
        OrderBusinessType.BUSINESS_TYPE_SI,
        TradeDirection.TRADE_DIRECTION_IMPORT,
      ),
    buildDetailInitialValues: buildSeaExportDetailInitialValues,
    buildUpdatePayload: buildSeaExportUpdatePayload,
  },
  DetailFeatures: SeaExportDetailFeatures,
};
