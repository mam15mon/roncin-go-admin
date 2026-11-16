// @vitest-environment node
import { it } from 'vitest';
import { awbPdfExtremeSample, awbPdfSample } from './AwbPdfDocument.fixtures';
import { verifyScale } from './AwbPdfDocument.scale.test-utils';

it('各10000字唛头品名、5000字备注和10000字单行货描完整续页', async () => {
  const extreme = awbPdfExtremeSample();
  const c = extreme.content;
  await verifyScale(
    {
      ...extreme,
      masterNo: '999-12345675',
      content: {
        ...awbPdfSample.content,
        marks: c.marks,
        goodsDescription: c.goodsDescription,
        remarks: c.remarks,
        rateLines: [
          {
            ...awbPdfSample.content.rateLines?.[0],
            goodsDescription: `${'LLLL'.repeat(2495)}RATE-LONG-GOODS-END`,
          },
        ],
      },
    },
    'long-cargo',
    10,
    [
      'MARKS-LONG-END',
      'GOODS-LONG-END',
      'REMARKS-LONG-END',
      'RATE-LONG-GOODS-END',
    ],
  );
}, 30_000);
