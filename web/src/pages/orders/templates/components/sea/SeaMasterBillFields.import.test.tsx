import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import { history } from '@/router/history';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import { SeaMasterBillFields } from './SeaTransportSection';

const permission = vi.hoisted(() => ({ allowedType: 2 }));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (type: number) => type === permission.allowedType,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('@/router/history', () => ({ history: { push: vi.fn() } }));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceMatchSeaMasterBillCandidate: vi
    .fn()
    .mockResolvedValue({ matched: false }),
}));
const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
function Harness({ detail = false }: { detail?: boolean }) {
  return (
    <Form
      initialValues={{
        shippingLineId: 'carrier',
        seaMasterBillMasterNo: 'MBLSAMENUMBER',
        seaDocumentStructure: HOUSE,
        ...(detail
          ? {
              seaMasterBill: {
                masterBillId: 'import-master',
                masterNo: 'MBLSAMENUMBER',
                shippingLineId: 'carrier',
                transportExecutionId: 'import-voyage',
              },
            }
          : {}),
      }}
    >
      <SeaMasterBillFields
        businessType={OrderBusinessType.BUSINESS_TYPE_SI}
        orderKind="sea-import"
        isDetail={detail}
        organizationId="org"
        seaHouseOrderSourceOrderId={detail ? 'import-order' : undefined}
      />
    </Form>
  );
}
describe('SI 真实主单身份与新增分单入口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    permission.allowedType = OrderBusinessType.BUSINESS_TYPE_SI;
  });
  it('候选明确查询SI，同号不查询出口候选', async () => {
    render(<Harness />);
    await waitFor(() =>
      expect(orderServiceMatchSeaMasterBillCandidate).toHaveBeenCalledWith(
        expect.objectContaining({
          businessType: 'SI',
          masterNo: 'MBLSAMENUMBER',
        }),
        expect.anything(),
      ),
    );
    expect(
      vi
        .mocked(orderServiceMatchSeaMasterBillCandidate)
        .mock.calls.every(([request]) => request.businessType === 'SI'),
    ).toBe(true);
  });
  it('SI权限打开同方向创建地址，只有SE权限不显示SI入口', async () => {
    const view = render(<Harness detail />);
    fireEvent.click(screen.getByRole('button', { name: '从此主单新增分单' }));
    expect(history.push).toHaveBeenCalledWith(
      '/orders/sea-import/new?sourceOrderId=import-order',
    );
    view.unmount();
    permission.allowedType = OrderBusinessType.BUSINESS_TYPE_SE;
    render(<Harness detail />);
    expect(
      screen.queryByRole('button', { name: '从此主单新增分单' }),
    ).toBeNull();
    await act(async () => {});
  });
});
