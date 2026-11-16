// @vitest-environment node
import { it } from 'vitest';
import { awbPdfExtremeSample, awbPdfSample } from './AwbPdfDocument.fixtures';
import { verifyScale } from './AwbPdfDocument.scale.test-utils';

it('全部5000字收发通知代理印发文本与含换行长原号完整续页', async () => {
  const extreme = awbPdfExtremeSample();
  const c = extreme.content;
  await verifyScale(
    {
      ...extreme,
      content: {
        ...awbPdfSample.content,
        shipperText: c.shipperText,
        consigneeText: c.consigneeText,
        notifyPartyText: c.notifyPartyText,
        secondNotifyPartyText: c.secondNotifyPartyText,
        agentText: c.agentText,
        issuedBy: c.issuedBy,
      },
    },
    'long-parties',
    10,
    [
      'MASTER-NUMBER-END',
      'SHIPPER-END',
      'CONSIGNEE-END',
      'NOTIFY-END',
      'SECOND-NOTIFY-END',
      'AGENT-END',
      'ISSUED-BY-END',
    ],
  );
}, 30_000);
