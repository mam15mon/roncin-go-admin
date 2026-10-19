import {
  catalog,
  chooseType,
  deferred,
  renderNotificationPage,
  requestFailure,
  setupNotificationMocks,
} from '@root/tests/notification-tests/support';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  adminServiceListNotificationTestTypes,
  adminServicePreviewNotificationTest,
  adminServiceSendNotificationTest,
} from '@/services/roncin/adminService';

vi.mock('@/services/roncin/adminService');
vi.mock('@/services/roncin/orderService');

beforeEach(() => {
  vi.resetAllMocks();
  setupNotificationMocks();
});
afterEach(cleanup);

describe('本人钉钉测试发送', () => {
  it('只显式点击才发，连续点击仅一次；在途禁止换类型和重复发送', async () => {
    const send = deferred<API.SendNotificationTestResponse>();
    vi.mocked(adminServiceSendNotificationTest).mockReturnValueOnce(
      send.promise,
    );
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    await chooseType('费用待审核');
    await screen.findByText('[测试] FEE_REVIEW_PENDING');
    const button = screen.getByRole('button', { name: '发送给我测试' });
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
    act(() => {
      button.click();
      button.click();
    });
    await waitFor(() =>
      expect(adminServiceSendNotificationTest).toHaveBeenCalledTimes(1),
    );
    expect(adminServiceSendNotificationTest).toHaveBeenCalledWith(
      { notificationType: 'FEE_REVIEW_PENDING', orderId: undefined },
      { skipErrorHandler: true },
    );
    expect(button).toBeDisabled();
    expect(screen.getByRole('combobox', { name: '通知类型' })).toBeDisabled();
    await act(async () =>
      send.resolve({ success: true, data: { accepted: true } }),
    );
    expect(
      await screen.findByText('钉钉已受理，请在客户端确认'),
    ).toBeInTheDocument();
  });

  it('无绑定或集成不可用都只能预览，发送权限也不能绕过', async () => {
    vi.mocked(adminServiceListNotificationTestTypes).mockResolvedValueOnce(
      catalog({
        recipientBound: false,
        canSend: false,
        sendUnavailableReason: '当前账号未绑定钉钉',
      }),
    );
    const first = renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    expect(screen.getByText('当前账号未绑定钉钉')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    first.unmount();
    vi.mocked(adminServiceListNotificationTestTypes).mockResolvedValueOnce(
      catalog({
        integrationEnabled: false,
        canSend: false,
        sendUnavailableReason: '钉钉集成尚未启用',
      }),
    );
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    expect(screen.getByText('钉钉集成尚未启用')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('缺发送权限或预览失败时禁止发消息', async () => {
    const first = renderNotificationPage(['system.notification.preview']);
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    expect(
      screen.getByText('当前账号只有预览权限，无法发送测试'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    first.unmount();
    vi.mocked(adminServicePreviewNotificationTest).mockRejectedValueOnce(
      requestFailure('预览暂不可用'),
    );
    renderNotificationPage();
    await screen.findByText('预览暂不可用');
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('上游失败不报成功、不自动重试，显式重试后仅报告受理', async () => {
    vi.mocked(adminServiceSendNotificationTest).mockRejectedValueOnce(
      requestFailure('钉钉暂时不可用'),
    );
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    fireEvent.click(screen.getByRole('button', { name: '发送给我测试' }));
    await screen.findByText('钉钉暂时不可用');
    expect(
      screen.getByText(
        '发送结果尚未确认，请先在钉钉客户端核对，再决定是否重试。',
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByText('钉钉已受理，请在客户端确认'),
    ).not.toBeInTheDocument();
    expect(adminServiceSendNotificationTest).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: '发送给我测试' }));
    await screen.findByText('钉钉已受理，请在客户端确认');
    expect(adminServiceSendNotificationTest).toHaveBeenCalledTimes(2);
    await chooseType('账号授权');
    await screen.findByText('[测试] USER_AUTHORIZED');
    expect(
      screen.queryByText('钉钉已受理，请在客户端确认'),
    ).not.toBeInTheDocument();
  });

  it('全局消费的undefined不伪报成功，离开页面后的迟到结果不写入新页', async () => {
    vi.mocked(adminServiceSendNotificationTest).mockResolvedValueOnce(
      undefined as unknown as API.SendNotificationTestResponse,
    );
    const first = renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    fireEvent.click(screen.getByRole('button', { name: '发送给我测试' }));
    await screen.findByText(
      '未确认钉钉受理测试通知，请先在客户端确认，再决定是否重试',
    );
    const late = deferred<API.SendNotificationTestResponse>();
    vi.mocked(adminServiceSendNotificationTest).mockReturnValueOnce(
      late.promise,
    );
    fireEvent.click(screen.getByRole('button', { name: '发送给我测试' }));
    await waitFor(() =>
      expect(adminServiceSendNotificationTest).toHaveBeenCalledTimes(2),
    );
    first.unmount();
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    await act(async () =>
      late.resolve({ success: true, data: { accepted: true } }),
    );
    expect(
      screen.queryByText('钉钉已受理，请在客户端确认'),
    ).not.toBeInTheDocument();
  });

  it.each([
    [
      '网络超时',
      () =>
        Promise.reject(
          Object.assign(new Error('timeout'), { code: 'ECONNABORTED' }),
        ),
    ],
    ['缺少受理标识', () => Promise.resolve({ success: true, data: {} })],
  ])('%s时提示先确认客户端，不断言未发送或自动重试', async (_name, result) => {
    vi.mocked(adminServiceSendNotificationTest).mockImplementationOnce(result);
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    fireEvent.click(screen.getByRole('button', { name: '发送给我测试' }));
    await screen.findByText(
      '未确认钉钉受理测试通知，请先在客户端确认，再决定是否重试',
    );
    expect(
      screen.queryByText('钉钉已受理，请在客户端确认'),
    ).not.toBeInTheDocument();
    expect(adminServiceSendNotificationTest).toHaveBeenCalledTimes(1);
  });
});
