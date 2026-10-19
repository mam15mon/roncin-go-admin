import {
  chooseType,
  deferred,
  homepageUrl,
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
      if (type.notificationType === 'PASSWORD_RESET_CODE') {
        expect(within(region).getByText('012345').tagName).toBe('STRONG');
        expect(
          within(region).getByText('5分钟内有效，请勿向他人提供。'),
        ).toBeVisible();
        expect(screen.queryByRole('link')).not.toBeInTheDocument();
      } else {
        expect(within(region).getByText('示例内容').tagName).toBe('STRONG');
        expect(
          screen.getByRole('link', { name: '打开系统首页' }),
        ).toHaveAttribute('href', homepageUrl);
      }
    }
    fireEvent.click(screen.getByRole('button', { name: '查看消息内容' }));
    expect(
      screen.getByText(
        '### 🔐 测试通知 PASSWORD_RESET_CODE **012345** 5分钟内有效，请勿向他人提供。 > 示例验证码无效，不可用于重置密码。',
        {
          selector: 'pre',
        },
      ),
    ).toBeVisible();
    expect(screen.getByText('测试管理员（本人）')).toBeInTheDocument();
    expect(adminServicePreviewNotificationTest).toHaveBeenCalledTimes(8);
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('系统管理员仅用示例测试，费用没有真实订单候选，按钮明确指向首页', async () => {
    renderNotificationPage();
    await screen.findByText('[测试] ORDER_PERSONNEL_ASSIGNED');
    for (const label of ['费用待审核', '费用审核结果']) {
      await chooseType(label);
      await screen.findByText(
        label === '费用待审核'
          ? '[测试] FEE_REVIEW_PENDING'
          : '[测试] FEE_REVIEW_RESULT',
      );
      expect(
        screen.getByText(
          '系统管理工作台无经营数据读取权限，本页仅使用示例测试',
        ),
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          '本次使用示例数据，按钮仅演示打开系统首页，不指向真实订单。',
        ),
      ).toBeInTheDocument();
      expect(
        screen.queryByRole('combobox', { name: '真实订单（选填）' }),
      ).not.toBeInTheDocument();
      expect(
        screen.getByRole('link', { name: '打开系统首页' }),
      ).toHaveAttribute('href', homepageUrl);
      expect(
        screen.queryByRole('link', { name: '查看订单费用' }),
      ).not.toBeInTheDocument();
    }
    expect(orderServiceListOrders).not.toHaveBeenCalled();
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('切换通知隔离迟到结果，并禁止原始HTML、图片和正文任意链接', async () => {
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
          '<img src="x" onerror="alert(1)">\n\n![图片](https://evil.test/x)\n\n[外站](https://evil.test)\n\n[脚本](javascript:alert%281%29)\n\n[本站正文链接](https://configured.example.test/welcome)',
      },
    });
    await chooseType('密码验证码');
    await screen.findByText('受控预览');
    const region = screen.getByRole('region', { name: 'Markdown预览' });
    expect(within(region).queryByRole('img')).not.toBeInTheDocument();
    expect(within(region).queryByRole('link')).not.toBeInTheDocument();
    expect(region.querySelector('script')).toBeNull();
  });

  it('按钮文字按文本输出，消息内容可核对正文、标题和目标', async () => {
    vi.mocked(adminServicePreviewNotificationTest).mockResolvedValueOnce({
      success: true,
      data: {
        title: '固定会话标题',
        markdown:
          '### 🔵 示例通知\n\n**编号001**\n\n[正文链接](https://configured.example.test/welcome)',
        actionTitle: '<b>打开系统首页</b>',
        actionUrl: homepageUrl,
      },
    });
    renderNotificationPage();
    const button = await screen.findByRole('link', {
      name: '<b>打开系统首页</b>',
    });
    expect(button.querySelector('b')).toBeNull();
    expect(button).toHaveAttribute('rel', 'noopener noreferrer');
    expect(button).toHaveAttribute('target', '_blank');
    expect(
      within(screen.getByRole('region', { name: 'Markdown预览' })).queryByRole(
        'link',
      ),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '查看消息内容' }));
    expect(screen.getByText('固定会话标题')).toBeVisible();
    expect(screen.getByText(homepageUrl)).toBeVisible();
    expect(
      screen.getByText(
        '### 🔵 示例通知 **编号001** [正文链接](https://configured.example.test/welcome)',
        { selector: 'pre' },
      ),
    ).toBeVisible();
  });

  it('非法动作不生成按钮且禁止发送，原始目标只作为核对文本', async () => {
    vi.mocked(adminServicePreviewNotificationTest).mockResolvedValueOnce({
      success: true,
      data: {
        title: '非法动作示例',
        markdown: '### 示例正文',
        actionTitle: '打开系统首页',
        actionUrl: 'javascript:alert(1)',
      },
    });
    renderNotificationPage();
    await screen.findByText('通知按钮目标无效，无法发送测试，请重新加载预览');
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '查看消息内容' }));
    expect(screen.getByText('javascript:alert(1)')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '重新加载预览' }));
    expect(
      await screen.findByRole('link', { name: '打开系统首页' }),
    ).toHaveAttribute('href', homepageUrl);
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeEnabled();
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
  });

  it('旧预览迟到失败不影响当前内容，当前预览失败则移除旧卡片并禁止发送', async () => {
    const late = deferred<API.PreviewNotificationTestResponse>();
    vi.mocked(adminServicePreviewNotificationTest).mockReturnValueOnce(
      late.promise,
    );
    renderNotificationPage();
    await screen.findByText('测试管理员（本人）');
    await chooseType('账号授权');
    await screen.findByText('[测试] USER_AUTHORIZED');
    await act(async () => late.reject(requestFailure('旧预览错误')));
    expect(screen.queryByText('旧预览错误')).not.toBeInTheDocument();
    expect(
      within(screen.getByRole('region', { name: 'Markdown预览' })).getByRole(
        'heading',
      ),
    ).toHaveTextContent('USER_AUTHORIZED');
    vi.mocked(adminServicePreviewNotificationTest).mockRejectedValueOnce(
      requestFailure('新预览暂不可用'),
    );
    await chooseType('密码验证码');
    await screen.findByText('新预览暂不可用');
    expect(
      screen.queryByRole('region', { name: '消息卡片预览' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发送给我测试' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '重试预览' }));
    expect(
      await screen.findByRole('heading', { name: /PASSWORD_RESET_CODE/ }),
    ).toBeVisible();
    expect(adminServiceSendNotificationTest).not.toHaveBeenCalled();
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
