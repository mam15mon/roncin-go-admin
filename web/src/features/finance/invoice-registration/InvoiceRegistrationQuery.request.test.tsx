import { renderWithApp } from '@root/tests/renderWithApp';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import type { AxiosAdapter } from 'axios';
import { CanceledError } from 'axios';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from '@/utils/queryClient';
import InvoiceRegistrationEditor from './InvoiceRegistrationEditor';
import InvoiceRegistrationModal from './InvoiceRegistrationModal';
import { invoiceUser, NoMotion } from './InvoiceRegistrationModal.fixtures';

const { transport, notify, message } = vi.hoisted(() => ({
  transport: vi.fn<AxiosAdapter>(),
  notify: vi.fn(),
  message: vi.fn(),
}));
vi.mock('axios', async (original) => {
  const actual = await original<typeof import('axios')>();
  return {
    ...actual,
    default: {
      ...actual.default,
      create: (config: Parameters<typeof actual.default.create>[0]) =>
        actual.default.create({ ...config, adapter: transport }),
    },
  };
});
vi.mock('@sentry/react', () => ({ captureException: vi.fn() }));
vi.mock('@/utils/appFeedback', () => ({
  showErrorMessage: message,
  showErrorNotification: notify,
}));

const clients: ReturnType<typeof createQueryClient>[] = [];
beforeEach(() => vi.clearAllMocks());
afterEach(() => {
  for (const client of clients.splice(0)) client.clear();
});
const scenes = [
  {
    name: '新登记抬头候选',
    path: '/api/v1/finance/invoice-creation/party-profiles',
    errorMessage: '加载登记抬头失败',
    content: () => (
      <Form initialValues={{ records: [] }}>
        <InvoiceRegistrationEditor
          name={['records']}
          organizationId="company"
          settlementPartyId="party"
          bills={[]}
        />
      </Form>
    ),
  },
  {
    name: '修订初始完整影响',
    path: '/api/v1/finance/invoices/invoice/revision-impact',
    errorMessage: '加载全部发票分配与结算影响失败',
    content: () => (
      <InvoiceRegistrationModal
        open
        reviseInvoice={{
          id: 'invoice',
          organizationId: 'company',
          version: '1',
        }}
        onOpenChange={vi.fn()}
        onSuccess={vi.fn()}
      />
    ),
  },
];
function mount(scene: (typeof scenes)[number]) {
  const client = createQueryClient();
  clients.push(client);
  const view = renderWithApp(
    <QueryClientProvider client={client}>
      <NoMotion>{scene.content()}</NoMotion>
    </QueryClientProvider>,
    { user: invoiceUser },
  );
  return { ...view, client };
}

describe('发票查询真实请求的取消与错误归属', () => {
  for (const scene of scenes) {
    it(`${scene.name}卸载确实取消传输，取消不显示请求失败通知或查询错误`, async () => {
      let aborted = false;
      transport.mockImplementation(
        (config) =>
          new Promise((_resolve, reject) => {
            config.signal?.addEventListener?.('abort', () => {
              aborted = true;
              reject(new CanceledError('canceled', config));
            });
          }),
      );
      const view = mount(scene);
      await waitFor(() => {
        expect(transport).toHaveBeenCalledOnce();
        expect(transport.mock.calls[0][0].url).toBe(scene.path);
      });
      await act(async () => view.unmount());
      await waitFor(() => expect(aborted).toBe(true));
      expect(notify).not.toHaveBeenCalled();
      expect(message).not.toHaveBeenCalled();
    });

    it(`${scene.name}真实请求失败仍由查询缓存显示明确业务提示一次`, async () => {
      transport.mockImplementation(async (config) => {
        throw {
          config,
          response: { status: 500, data: { message: '服务端异常' } },
        };
      });
      mount(scene);
      await waitFor(() =>
        expect(message).toHaveBeenCalledWith(scene.errorMessage),
      );
      expect(message).toHaveBeenCalledOnce();
      expect(notify).not.toHaveBeenCalled();
    });
  }
});
