import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import type { ComponentProps } from 'react';
import { beforeEach, vi } from 'vitest';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaOrderReassignment,
  seaOrderChangeServicePreviewSeaOrderReassignment,
} from '@/services/roncin/seaOrderChangeService';
import SeaOrderReassignmentModal from './SeaOrderReassignmentModal';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceMatchSeaMasterBillCandidate: vi.fn(),
}));
vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceExecuteSeaOrderReassignment: vi.fn(),
  seaOrderChangeServicePreviewSeaOrderReassignment: vi.fn(),
}));
vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
}));

export const preview = vi.mocked(
  seaOrderChangeServicePreviewSeaOrderReassignment,
);
export const execute = vi.mocked(
  seaOrderChangeServiceExecuteSeaOrderReassignment,
);
export const matchCandidate = vi.mocked(
  orderServiceMatchSeaMasterBillCandidate,
);

export function validPreview(
  target?: API.SeaOrderReassignmentTargetInput,
  version = '7',
): API.PreviewSeaOrderReassignmentResponse {
  return {
    success: true,
    data: {
      isValid: true,
      orderVersion: version,
      currentLinkVersion: '4',
      differences: [
        {
          fieldName: 'master_no',
          label: '提单号(MBL)',
          currentValue: 'OLDMBL123',
          targetValue: target?.masterNo ?? 'NEWMBL456',
          isDifferent: true,
        },
      ],
      currentMasterBill: { masterNo: 'OLDMBL123', vesselName: 'EVER GIVEN' },
      targetMasterBill: {
        masterNo: target?.masterNo ?? 'NEWMBL456',
        vesselName: target?.vesselName,
      },
    },
  };
}

export const candidate: API.SeaMasterBillCandidate = {
  id: 'candidate-mbl',
  version: '8',
  masterNo: 'EXISTING123',
  shippingLineId: 'carrier-1',
  shippingLineName: '中远海运',
  memberCount: 2,
  transportExecutions: [
    {
      id: 'candidate-te',
      version: '9',
      shippingLineId: 'carrier-1',
      vesselName: 'CANDIDATE VESSEL',
      voyageNo: 'VOYAGE002',
      originLocationId: 'port-origin',
      dischargeLocationId: 'port-discharge',
      etd: '2026-11-10T08:00:00Z',
      eta: '2026-11-15T08:00:00Z',
    },
  ],
};

export function setupReassignmentTests() {
  beforeEach(() => {
    preview.mockReset();
    execute.mockReset();
    matchCandidate.mockReset();
    preview.mockImplementation(async (_params, body) =>
      validPreview(body.target),
    );
    execute.mockResolvedValue({
      success: true,
      data: { orderId: 'order-123', reassignmentEventId: 'event-1' },
    });
    matchCandidate.mockResolvedValue({ success: true, matched: false });
  });
}

export function renderReassignment(
  overrides: Partial<ComponentProps<typeof SeaOrderReassignmentModal>> = {},
) {
  const props: ComponentProps<typeof SeaOrderReassignmentModal> = {
    open: true,
    orderId: 'order-123',
    orderNo: 'SE20261106001',
    initialShippingLineId: 'carrier-1',
    initialShippingLineName: '中远海运',
    onClose: vi.fn(),
    onSuccess: vi.fn(),
    ...overrides,
  };
  const content = (changed: Partial<typeof props> = {}) => (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <SeaOrderReassignmentModal {...props} {...changed} />
    </ConfigProvider>
  );
  const result = renderWithApp(content());
  return {
    ...result,
    props,
    rerenderModal: (changed: Partial<typeof props>) =>
      result.rerender(content(changed)),
  };
}

export function enterDate(label: string, value: string) {
  const input = screen.getByLabelText(label);
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value } });
  fireEvent.keyDown(input, { key: 'Enter' });
  fireEvent.blur(input);
}

export function enterMasterNo(value = 'NEWMBL456') {
  fireEvent.change(screen.getByPlaceholderText('请输入目标提单号'), {
    target: { value },
  });
}

export async function fillConfirmation() {
  await act(async () => {
    fireEvent.change(
      screen.getByPlaceholderText(
        '详细记录改配发生的背景与原因（将记录进不可变事件历史与审计日志）',
      ),
      { target: { value: '客户要求改配' } },
    );
    fireEvent.change(screen.getByPlaceholderText('例如：XX 船代操作部'), {
      target: { value: '测试船代' },
    });
    enterDate('确认时间', '2026-11-06 12:00:00');
    fireEvent.change(
      screen.getByPlaceholderText('记录确认渠道、联系人和确认结论'),
      { target: { value: '船代邮件确认可以变更' } },
    );
  });
}

export function outerConfirmButton() {
  const dialog = screen.getByText(/订单整票改配 - /).closest('[role="dialog"]');
  if (!(dialog instanceof HTMLElement)) throw new Error('未找到改配弹窗');
  return within(dialog).getByRole('button', { name: '确认改配' });
}

export async function openExecutionConfirmation() {
  await waitFor(() => expect(outerConfirmButton()).toBeEnabled());
  fireEvent.click(outerConfirmButton());
  const title = await screen.findByText('确认提交整票改配？');
  const dialog = title.closest('[role="dialog"]');
  if (!(dialog instanceof HTMLElement)) throw new Error('未找到改配确认框');
  await waitFor(() => expect(dialog).toBeVisible());
  return dialog;
}

export function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
