import { fireEvent, screen } from '@testing-library/react';
import { vi } from 'vitest';
import NotificationTestPage from '@/pages/admin/notification-test';
import {
  adminServiceListNotificationTestTypes,
  adminServicePreviewNotificationTest,
  adminServiceSendNotificationTest,
} from '@/services/roncin/adminService';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import { renderWithApp } from '../renderWithApp';

export const orderId = '12345678-1234-4234-8234-123456789abc';
export const orderUrl = `https://configured.example.test/orders/sea-export/${orderId}/fees`;

export function requestFailure(message: string) {
  return Object.assign(new Error('Request failed with status code 503'), {
    response: { data: { message } },
  });
}
export const types: API.NotificationTestType[] = [
  { notificationType: 'ORDER_PERSONNEL_ASSIGNED', label: '订单人员分配' },
  { notificationType: 'USER_AUTHORIZED', label: '账号授权' },
  { notificationType: 'EXCHANGE_RATE_WEEKLY_REMINDER', label: '周汇率提醒' },
  {
    notificationType: 'FEE_REVIEW_PENDING',
    label: '费用待审核',
    supportsOrder: true,
  },
  {
    notificationType: 'FEE_REVIEW_RESULT',
    label: '费用审核结果',
    supportsOrder: true,
  },
  { notificationType: 'FEE_SUPPLEMENT_APPROVAL_PENDING', label: '补录待审批' },
  { notificationType: 'COMMISSION_DECREASE_SUGGESTED', label: '提成冲减' },
  { notificationType: 'PASSWORD_RESET_CODE', label: '密码验证码' },
];

export function catalog(overrides: Partial<API.NotificationTestCatalog> = {}) {
  return {
    success: true,
    data: {
      types,
      recipientName: '测试管理员',
      recipientBound: true,
      integrationEnabled: true,
      canSend: true,
      canSelectOrder: true,
      ...overrides,
    },
  };
}

export function setupNotificationMocks() {
  vi.mocked(adminServiceListNotificationTestTypes).mockResolvedValue(catalog());
  vi.mocked(adminServicePreviewNotificationTest).mockImplementation(
    async (params) => ({
      success: true,
      data: {
        notificationType: params.notificationType,
        title: `[测试] ${params.notificationType}`,
        markdown: `### 测试通知 ${params.notificationType}\n\n**示例内容**${params.orderId ? `\n\n[查看订单费用](${orderUrl})` : ''}`,
        orderUrl: params.orderId ? orderUrl : '',
      },
    }),
  );
  vi.mocked(adminServiceSendNotificationTest).mockResolvedValue({
    success: true,
    data: { accepted: true },
  });
  vi.mocked(orderServiceListOrders).mockResolvedValue({
    success: true,
    data: [{ id: orderId, orderNo: 'SE-TEST-001' }],
    total: 1,
  });
}

export function renderNotificationPage(
  permissions = [
    'system.notification.preview',
    'system.notification.test_send',
  ],
) {
  return renderWithApp(<NotificationTestPage />, {
    user: {
      id: 'admin-test',
      displayName: '测试管理员',
      currentOrganization: { id: 'workspace-test', kind: 1 },
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'organization',
      })),
      roleScopes: [{ roleCode: 'arbitrary-role' }],
    },
    routerProps: { initialEntries: ['/admin/notification-test'] },
  });
}

export async function chooseType(label: string) {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: '通知类型' }));
  fireEvent.click(
    await screen.findByText(label, {
      selector: '.ant-select-item-option-content',
    }),
  );
}

export function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}
