import type { CreateOrderFormValues } from '../order-kinds/form-values';
import type { TemplateProps } from './types';

/** 仅映射订单内的两个可编辑正文目标；深合并由模板 applyValues 完成。 */
export function airDocumentImportPatch(
  ...[prefix, field, value]: Parameters<
    NonNullable<TemplateProps['onAirDocumentImport']>
  >
): Partial<CreateOrderFormValues> {
  if (prefix.length === 1 && prefix[0] === 'airHouseContent')
    return { airHouseContent: { [field]: value } };
  if (
    prefix.length === 2 &&
    prefix[0] === 'airMaster' &&
    prefix[1] === 'waybillContent'
  )
    return { airMaster: { waybillContent: { [field]: value } } };
  throw new Error('单证带入目标无效');
}
