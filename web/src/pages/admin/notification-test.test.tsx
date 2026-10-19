import {
  catalog,
  chooseType,
  deferred,
  orderId,
  orderUrl,
  renderNotificationPage,
  requestFailure,
  setupNotificationMocks,
  types,
} from '@root/tests/notification-tests/support';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  adminServiceListNotificationTestTypes,
  adminServicePreviewNotificationTest,
  adminServiceSendNotificationTest,
} from '@/services/roncin/adminService';
import { orderServiceListOrders } from '@/services/roncin/orderService';

vi.mock('@/services/roncin/adminService');
vi.mock('@/services/roncin/orderService');

beforeEach(() => {
  vi.resetAllMocks();
  setupNotificationMocks();
});
afterEach(cleanup);

describe('独立 Admin 通知预览', () => {
  it('八类来自后端，加载和切换仅预览；源码与效果一致', async () => {
    renderNotificationPage();
    for (const [index, type] of types.entries()) {
      if (index > 0) await chooseType(type.label as string);
      await screen.findByText(`[测试] ${type.notificationType}`);
      const region = screen.getByRole('region', { name: 'Markdown预览' });
      expect(within(region).getByRole('heading')).toHaveTextContent(
        `测试通知 ${type.notificationType}`,
      );
      expect(within(region).getByText('示例内容').tagName).toBe('STRONG');
    }
    fireEvent.click(screen.getByRole('button', { name: '查看 Markdown 源码' }));
    expect(
      screen.getByText('### 测试通知 PASSWORD_RESET_CODE **示例内容**', {
        selector: 'pre',
      }),
    ).toBeVisible();
    expect(screen.getByText('测试管理员（本人）')).toBeInTheDocument();
    expect(adminServicePreviewNotificationTest).toHaveBeenCalledTimes(8);
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('费用默认没有链接，真实订单按服务端关键字查询并使用最终后端链接', async () => {
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    await chooseType('费用待审核');
    await screen.findByText('[测试] FEE_REVIEW_PENDING');
    expect(
      screen.getByText('未选择真实订单，本次不包含订单链接。'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: '查看订单费用' }),
    ).not.toBeInTheDocument();
    const orderInput = screen.getByRole('combobox', {
      name: '真实订单（选填）',
    });
    fireEvent.change(orderInput, { target: { value: 'SE-TEST' } });
    await waitFor(() =>
      expect(orderServiceListOrders).toHaveBeenCalledWith(
        { page: 1, pageSize: 20, businessType: 1, keyword: 'SE-TEST' },
        { skipErrorHandler: true },
      ),
    );
    fireEvent.mouseDown(orderInput);
    fireEvent.click(
      await screen.findByText('SE-TEST-001', {
        selector: '.ant-select-item-option-content',
      }),
    );
    expect(
      await screen.findByRole('link', { name: '查看订单费用' }),
    ).toHaveAttribute('href', orderUrl);
    expect(adminServicePreviewNotificationTest).toHaveBeenLastCalledWith(
      { notificationType: 'FEE_REVIEW_PENDING', orderId },
      { skipErrorHandler: true },
    );
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('无读取资格明确提示且不查询订单，仍可预览示例', async () => {
    vi.mocked(adminServiceListNotificationTestTypes).mockResolvedValue(
      catalog({
        canSelectOrder: false,
        orderSelectionUnavailableReason: '请切换到有读取权限的公司工作台',
      }),
    );
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    await chooseType('费用审核结果');
    await screen.findByText('[测试] FEE_REVIEW_RESULT');
    expect(
      screen.getByText('请切换到有读取权限的公司工作台'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('combobox', { name: '真实订单（选填）' }),
    ).not.toBeInTheDocument();
    expect(orderServiceListOrders).not.toHaveBeenCalled();
  });

  it('切换通知隔离迟到结果，并禁止原始HTML、图片及非服务端订单链接', async () => {
    const late = deferred<API.PreviewNotificationTestResponse>();
    vi.mocked(adminServicePreviewNotificationTest).mockReturnValueOnce(
      late.promise,
    );
    renderNotificationPage();
    await screen.findByText('测试管理员（本人）');
    await chooseType('账号授权');
    await screen.findByText('[测试] USER_AUTHORIZED');
    await act(async () =>
      late.resolve({
        success: true,
        data: { title: '迟到旧内容', markdown: '迟到旧内容' },
      }),
    );
    expect(screen.queryByText('迟到旧内容')).not.toBeInTheDocument();
    vi.mocked(adminServicePreviewNotificationTest).mockResolvedValueOnce({
      success: true,
      data: {
        title: '受控预览',
        markdown:
          '<img src="x" onerror="alert(1)">\n\n![图片](https://evil.test/x)\n\n[外站](https://evil.test)\n\n[脚本](javascript:alert%281%29)',
      },
    });
    await chooseType('密码验证码');
    await screen.findByText('受控预览');
    const region = screen.getByRole('region', { name: 'Markdown预览' });
    expect(within(region).queryByRole('img')).not.toBeInTheDocument();
    expect(within(region).queryByRole('link')).not.toBeInTheDocument();
    expect(region.querySelector('script')).toBeNull();
  });

  it('目录失败可显式重试；无预览权限时不查询', async () => {
    vi.mocked(adminServiceListNotificationTestTypes).mockRejectedValueOnce(
      requestFailure('目录暂不可用'),
    );
    const { unmount } = renderNotificationPage();
    await screen.findByText('目录暂不可用');
    fireEvent.click(screen.getByRole('button', { name: '重新加载' }));
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    unmount();
    vi.clearAllMocks();
    renderNotificationPage([]);
    expect(screen.getByText('暂无通知预览权限')).toBeInTheDocument();
    expect(adminServiceListNotificationTestTypes).not.toHaveBeenCalled();
    expect(adminServicePreviewNotificationTest).not.toHaveBeenCalled();
  });
});
