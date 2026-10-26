import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import { history } from '@/router/history';
import { SeaMasterBillFields } from './SeaTransportSection';

const permission = vi.hoisted(() => ({ create: true, organization: true }));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (kind: number, operation: string) =>
      kind === OrderBusinessType.BUSINESS_TYPE_SE &&
      operation === 'create' &&
      permission.create,
    canOperateOrganization: (id?: string) =>
      id === 'org-1' && permission.organization,
  }),
}));
vi.mock('@/router/history', () => ({ history: { push: vi.fn() } }));
const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
const summary = {
  masterBillId: 'mbl-1',
  masterNo: 'MBL001',
  shippingLineId: 'carrier-1',
  transportExecutionId: 'te-1',
  memberCount: 1,
};
function Harness({
  initialValues,
  exposeForm,
}: {
  initialValues?: Record<string, unknown>;
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        disabled
        form={form}
        initialValues={{
          seaMasterBillMasterNo: 'MBL001',
          shippingLineId: 'carrier-1',
          seaDocumentStructure: HOUSE,
          seaMasterBill: summary,
          ...initialValues,
        }}
      >
        <SeaMasterBillFields
          isDetail
          disabled
          seaHouseOrderSourceOrderId="order-source"
          organizationId="org-1"
        />
      </Form>
    </ConfigProvider>
  );
}

describe('从已有主单新增分单入口', () => {
  beforeEach(() => {
    permission.create = true;
    permission.organization = true;
    vi.clearAllMocks();
  });
  it.each([1, 3])(
    '关联 %i 票的锁定来源仍能独立新建分单',
    async (memberCount) => {
      render(
        <Harness
          initialValues={{ seaMasterBill: { ...summary, memberCount } }}
        />,
      );
      const button = screen.getByRole('button', { name: '从此主单新增分单' });
      expect(button).toBeEnabled();
      fireEvent.click(button);
      expect(history.push).toHaveBeenCalledWith(
        '/orders/sea-export/new?sourceOrderId=order-source',
      );
      await act(async () => {});
    },
  );
  it.each(['create', 'organization'] as const)(
    '缺少 %s 能力隐藏入口',
    async (key) => {
      permission[key] = false;
      render(<Harness />);
      expect(
        screen.queryByRole('button', { name: '从此主单新增分单' }),
      ).toBeNull();
      await act(async () => {});
    },
  );
  it.each([
    {
      seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    },
    { seaMasterBill: { ...summary, transportExecutionId: undefined } },
    { seaMasterBill: { ...summary, status: 'VOIDED' } },
  ])('直单、缺航程身份或已作废主单隐藏入口', async (initialValues) => {
    render(<Harness initialValues={initialValues} />);
    expect(
      screen.queryByRole('button', { name: '从此主单新增分单' }),
    ).toBeNull();
    await act(async () => {});
  });
  it('本票未保存的主单身份变化后隐藏入口', async () => {
    let form!: FormInstance;
    render(
      <Harness
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    expect(
      screen.getByRole('button', { name: '从此主单新增分单' }),
    ).toBeVisible();
    await act(async () => {
      form.setFieldValue('seaMasterBillMasterNo', 'MBL002');
    });
    expect(
      screen.queryByRole('button', { name: '从此主单新增分单' }),
    ).toBeNull();
  });
});
