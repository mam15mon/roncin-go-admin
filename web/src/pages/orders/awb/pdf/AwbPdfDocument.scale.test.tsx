// @vitest-environment node
import { it } from 'vitest';
import { awbPdfExtremeSample, awbPdfSample } from './AwbPdfDocument.fixtures';
import { verifyScale } from './AwbPdfDocument.scale.test-utils';

it('两表各200行完整续页，末行、未知金额及声明令牌保留', async () => {
  const extreme = awbPdfExtremeSample();
  await verifyScale(
    {
      ...extreme,
      masterNo: '999-12345675',
      content: {
        ...awbPdfSample.content,
        rateLines: extreme.content.rateLines,
        otherCharges: extreme.content.otherCharges,
        asArranged: true,
        prepaid: extreme.content.prepaid,
        carriageValue: { kind: 'NVD' },
      },
    },
    'many-rows',
    10,
    [
      'RATE-ROW-200-END',
      'OTHER-ROW-200-END',
      '6.7183',
      'AS ARRANGED',
      'NVD',
      'NCV',
      'XXX',
    ],
  );
}, 30_000);
