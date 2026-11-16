import { describe, expect, it } from 'vitest';
import { DangerousPackingGroup } from '@/enums.generated';
import { airImportDefinition } from '../air-import/definition';
import { airExportDefinition } from './definition';

describe('航空危险品取消的真实wire清空', () => {
  it.each([airExportDefinition, airImportDefinition])(
    '$kind更新清空UN/CLASS/截申报/包装等级，创建未填保持省略',
    (definition) => {
      for (const cleared of [undefined, null]) {
        const values = {
          customerId: 'customer',
          paymentTerm: 1,
          cargoCategoryIds: ['general'],
          unNumber: '',
          hazardClass: '',
          dangerousPackingGroup: cleared,
          declarationCutoffAt: cleared,
          goodsDescription: '取消危险品不清货描',
        };
        const updated = JSON.parse(
          JSON.stringify(
            definition.form.buildUpdatePayload('order', '7', values),
          ),
        );
        expect(updated).toMatchObject({
          expectedVersion: '7',
          unNumber: '',
          hazardClass: '',
          declarationCutoffAt: '',
          dangerousPackingGroup: 0,
          goodsDescription: '取消危险品不清货描',
          cargoCategoryIds: ['general'],
        });
        const created = JSON.parse(
          JSON.stringify(definition.form.buildCreatePayload(values)),
        );
        for (const key of [
          'unNumber',
          'hazardClass',
          'declarationCutoffAt',
          'dangerousPackingGroup',
        ])
          expect(created).not.toHaveProperty(key);
      }
    },
  );

  it.each([airExportDefinition, airImportDefinition])(
    '$kind已选择字段继续原值提交，显式UNSPECIFIED更新仍清空',
    (definition) => {
      const original = {
        customerId: 'customer',
        paymentTerm: 1,
        unNumber: '1263',
        hazardClass: '3',
        dangerousPackingGroup: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_II,
        declarationCutoffAt: '2026-11-16T12:30:00.000Z',
      };
      const payload = definition.form.buildUpdatePayload(
        'order',
        '4',
        original,
      );
      expect(payload).toMatchObject(original);
      const clear = definition.form.buildUpdatePayload('order', '4', {
        ...original,
        dangerousPackingGroup:
          DangerousPackingGroup.DANGEROUS_PACKING_GROUP_UNSPECIFIED,
      });
      expect(JSON.parse(JSON.stringify(clear)).dangerousPackingGroup).toBe(0);
    },
  );
});
