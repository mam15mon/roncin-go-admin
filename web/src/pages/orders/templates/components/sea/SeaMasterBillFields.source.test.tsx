import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import type { SeaHouseOrderCreateSource } from '../../types';
import { SeaMasterBillFields } from './SeaTransportSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
  }),
}));
const source: SeaHouseOrderCreateSource = {
  orderId: 'source-1',
  masterBillId: 'mbl-1',
  masterNo: 'MBL001',
  shippingLineId: 'carrier-1',
  transportExecutionId: 'te-source',
};
const first = {
  id: 'te-first',
  version: '4',
  vesselName: 'FIRST',
  voyageNo: '001W',
};
const current = {
  id: 'te-source',
  version: '9',
  vesselName: 'LATEST',
  voyageNo: '009W',
};
function response(executions = [first, current], id = 'mbl-1') {
  return {
    matched: true,
    candidate: {
      id,
      version: '7',
      masterNo: 'MBL001',
      shippingLineId: 'carrier-1',
      memberCount: 2,
      members: [],
      transportExecutions: executions,
    },
  };
}
function Harness({
  exposeForm,
  createSource = source,
}: {
  exposeForm: (form: FormInstance) => void;
  createSource?: SeaHouseOrderCreateSource;
}) {
  const [form] = Form.useForm();
  exposeForm(form);
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        form={form}
        initialValues={{
          seaMasterBillMasterNo: 'MBL001',
          shippingLineId: 'carrier-1',
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
        }}
      >
        <SeaMasterBillFields seaHouseOrderCreateSource={createSource} />
      </Form>
    </ConfigProvider>
  );
}

describe('来源主单最新航程确认', () => {
  beforeEach(() => vi.restoreAllMocks());
  it('多航程精确选择来源 TE 并用最新版本与航程回填', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(response());
    let form!: FormInstance;
    render(
      <Harness
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe(
        'te-source',
      ),
    );
    expect(form.getFieldValue('vesselVoyage')).toBe('LATEST / 009W');
    expect(form.getFieldValue('seaMasterBillExpectedCandidateTeVersion')).toBe(
      '9',
    );
    expect(form.getFieldValue('seaMasterBillExpectedCandidateVersion')).toBe(
      '7',
    );
    expect(screen.getByText(/本票保存后将加入该主单，共/)).toHaveTextContent(
      '3 票',
    );
  });
  it('来源 TE 消失且只剩另一条时阻断，用户明确选择后才关联', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(response([first]));
    let form!: FormInstance;
    render(
      <Harness
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    await screen.findByText('来源主单或航程不可用');
    expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBeUndefined();
    expect(form.getFieldValue('vesselVoyage')).toBeUndefined();
    await act(async () => {
      await expect(
        form.validateFields(['seaMasterBillMasterNo']),
      ).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({
            errors: [expect.stringContaining('来源航程已失效')],
          }),
        ],
      });
    });
    fireEvent.mouseDown(screen.getByRole('combobox', { name: '本票共用航程' }));
    fireEvent.click(await screen.findByText('FIRST / 001W / ETD未填写'));
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-first'),
    );
  });
  it.each([false, true])(
    '来源主单未匹配或被替换时不得创建其他主单（替换：%s）',
    async (replaced) => {
      vi.spyOn(
        orderService,
        'orderServiceMatchSeaMasterBillCandidate',
      ).mockResolvedValue(
        replaced ? response([current], 'mbl-other') : { matched: false },
      );
      let form!: FormInstance;
      render(
        <Harness
          exposeForm={(value) => {
            form = value;
          }}
        />,
      );
      await screen.findByText('来源主单或航程不可用');
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
      await act(async () => {
        await expect(
          form.validateFields(['seaMasterBillMasterNo']),
        ).rejects.toMatchObject({
          errorFields: [
            expect.objectContaining({
              errors: [expect.stringContaining('来源主单已失效')],
            }),
          ],
        });
      });
    },
  );
  it('主动改主单身份后按当前候选流程确认新身份', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(response([current], 'mbl-other'));
    let form!: FormInstance;
    render(
      <Harness
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    await screen.findByText('来源主单或航程不可用');
    await act(async () => {
      form.setFieldValue('seaMasterBillMasterNo', 'MBL002');
    });
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-other'),
    );
  });
});
