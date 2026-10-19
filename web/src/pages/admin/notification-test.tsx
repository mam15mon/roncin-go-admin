import { SendOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Descriptions,
  Select,
  Space,
  Spin,
  Tag,
  theme,
} from 'antd';
import { useState } from 'react';
import Markdown from 'react-markdown';
import { useAccess } from '@/app/access';
import { PageHeaderShell, SectionCard } from '@/components/ui';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  adminServiceListNotificationTestTypes,
  adminServicePreviewNotificationTest,
  adminServiceSendNotificationTest,
} from '@/services/roncin/adminService';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import { ensureListResponse, unwrapPage } from '@/utils/api';
import { notificationActionUrl } from './notification-test-action';

const queryPrefix = 'notification-tests';
const sendUnconfirmedMessage =
  '未确认钉钉受理测试通知，请先在客户端确认，再决定是否重试';

function failureMessage(error: unknown, fallback: string) {
  return getRequestErrorMessage(error, fallback);
}

export default function NotificationTestPage() {
  const access = useAccess();
  const { token } = theme.useToken();
  const queryClient = useQueryClient();
  const [selectedType, setSelectedType] = useState<string>();
  const [selectedOrder, setSelectedOrder] = useState<{
    value: string;
    label: string;
  }>();
  const [keyword, setKeyword] = useState('');
  const catalogQuery = useQuery({
    queryKey: [queryPrefix, 'catalog'],
    enabled: access.canPreviewNotifications,
    queryFn: async () => {
      const response = await adminServiceListNotificationTestTypes({
        skipErrorHandler: true,
      });
      if (!response?.success || !response.data) {
        throw new Error('加载通知类型失败');
      }
      return response.data;
    },
    meta: { silent: true },
  });
  const catalog = catalogQuery.data;
  const notificationType =
    selectedType ?? catalog?.types?.[0]?.notificationType;
  const type = catalog?.types?.find(
    (item) => item.notificationType === notificationType,
  );
  const supportsOrder = Boolean(type?.supportsOrder);
  const canSelectOrder = supportsOrder && Boolean(catalog?.canSelectOrder);
  const orderId = canSelectOrder ? selectedOrder?.value : undefined;
  const previewQuery = useQuery({
    queryKey: [queryPrefix, 'preview', { notificationType, orderId }],
    enabled: access.canPreviewNotifications && Boolean(type),
    queryFn: async () => {
      const response = await adminServicePreviewNotificationTest(
        { notificationType: notificationType as string, orderId },
        { skipErrorHandler: true },
      );
      if (!response?.success || !response.data?.markdown) {
        throw new Error('加载通知预览失败');
      }
      return response.data;
    },
    meta: { silent: true },
  });
  const ordersQuery = useQuery({
    queryKey: [queryPrefix, 'order-options', { keyword: keyword.trim() }],
    enabled: canSelectOrder,
    queryFn: async () =>
      unwrapPage(
        ensureListResponse(
          await orderServiceListOrders(
            { page: 1, pageSize: 20, businessType: 1, keyword: keyword.trim() },
            { skipErrorHandler: true },
          ),
          '加载订单候选失败',
        ),
      ),
    meta: { silent: true },
  });
  const sendMutation = useMutation({
    mutationKey: [queryPrefix, 'send'],
    mutationFn: async (input: API.SendNotificationTestRequest) => {
      const response = await adminServiceSendNotificationTest(input, {
        skipErrorHandler: true,
      });
      if (!response?.success || !response.data?.accepted) {
        throw new Error(sendUnconfirmedMessage);
      }
      return response.data;
    },
    meta: { silent: true },
  });
  const pending = sendMutation.isPending;
  const preview = previewQuery.data;
  const actionUrl = notificationActionUrl(
    preview?.actionTitle,
    preview?.actionUrl,
  );
  const invalidAction = Boolean(
    (preview?.actionTitle || preview?.actionUrl) && !actionUrl,
  );
  const canSend = Boolean(
    access.canSendNotificationTests &&
      catalog?.canSend &&
      catalog.recipientBound &&
      catalog.integrationEnabled &&
      preview &&
      !invalidAction &&
      !previewQuery.isFetching &&
      !previewQuery.isError &&
      !pending,
  );
  const orderOptions = (ordersQuery.data?.data ?? []).flatMap((order) =>
    order.id ? [{ value: order.id, label: order.orderNo || order.id }] : [],
  );
  if (
    selectedOrder &&
    !orderOptions.some((item) => item.value === selectedOrder.value)
  ) {
    orderOptions.unshift(selectedOrder);
  }

  return (
    <PageContainer title={false}>
      <PageHeaderShell
        title="通知测试"
        subTitle="预览通知内容，并发送到本人钉钉验收"
        tags={<Tag color="blue">Admin</Tag>}
      />
      {!access.canPreviewNotifications ? (
        <Alert type="warning" title="暂无通知预览权限" showIcon />
      ) : (
        <>
          <SectionCard title="测试设置">
            <Space orientation="vertical" size={16} style={{ width: '100%' }}>
              <Alert
                type="info"
                showIcon
                title="示例通知，仅用于验收"
                description="预览不会发送消息。点击发送后仅发给本人，不触发审批、费用、提成或账号变更；示例验证码无效。排版以钉钉实际显示为准。"
              />
              {catalogQuery.isPending && <Spin aria-label="加载通知类型" />}
              {catalogQuery.isError && (
                <Alert
                  type="error"
                  showIcon
                  title={failureMessage(catalogQuery.error, '加载通知类型失败')}
                  action={
                    <Button onClick={() => void catalogQuery.refetch()}>
                      重新加载
                    </Button>
                  }
                />
              )}
              {catalog && (
                <>
                  <div>
                    <label htmlFor="notification-test-type">通知类型</label>
                    <Select
                      id="notification-test-type"
                      style={{ width: '100%', marginTop: 6 }}
                      value={notificationType}
                      disabled={pending}
                      options={catalog.types?.map((item) => ({
                        value: item.notificationType,
                        label: item.label,
                      }))}
                      onChange={(value) => {
                        setSelectedType(value);
                        setSelectedOrder(undefined);
                        setKeyword('');
                        sendMutation.reset();
                      }}
                    />
                  </div>
                  {supportsOrder && (
                    <div>
                      <label htmlFor="notification-test-order">
                        真实订单（选填）
                      </label>
                      {canSelectOrder ? (
                        <>
                          <Select
                            id="notification-test-order"
                            allowClear
                            showSearch={{
                              filterOption: false,
                              onSearch: setKeyword,
                            }}
                            labelInValue
                            value={selectedOrder}
                            options={orderOptions}
                            loading={ordersQuery.isFetching}
                            disabled={pending}
                            style={{ width: '100%', marginTop: 6 }}
                            placeholder="搜索当前工作台可读的海运出口订单"
                            onChange={(value) => {
                              setSelectedOrder(value);
                              sendMutation.reset();
                            }}
                          />
                          <p>
                            候选最多显示 20
                            条，可输入关键字查找。仅使用订单编号与费用入口，其余内容仍为示例。
                          </p>
                          {ordersQuery.isError && (
                            <Alert
                              type="error"
                              title={failureMessage(
                                ordersQuery.error,
                                '加载订单候选失败',
                              )}
                            />
                          )}
                        </>
                      ) : (
                        <p>
                          {catalog.orderSelectionUnavailableReason ||
                            '系统管理工作台无经营数据读取权限，本页仅使用示例测试'}
                        </p>
                      )}
                      {!orderId && (
                        <p>
                          本次使用示例数据，按钮仅演示打开系统首页，不指向真实订单。
                        </p>
                      )}
                    </div>
                  )}
                  <Descriptions
                    size="small"
                    column={1}
                    items={[
                      {
                        key: 'recipient',
                        label: '测试接收人',
                        children: `${catalog.recipientName || '当前登录人'}（本人）`,
                      },
                      {
                        key: 'binding',
                        label: '钉钉绑定',
                        children: catalog.recipientBound
                          ? '已绑定'
                          : '未绑定，无法发送测试',
                      },
                      {
                        key: 'integration',
                        label: '钉钉集成',
                        children: catalog.integrationEnabled
                          ? '已启用'
                          : '不可用，仍可查看示例预览',
                      },
                    ]}
                  />
                  {!access.canSendNotificationTests ? (
                    <Alert
                      type="warning"
                      title="当前账号只有预览权限，无法发送测试"
                      showIcon
                    />
                  ) : catalog.sendUnavailableReason ? (
                    <Alert
                      type="warning"
                      title={catalog.sendUnavailableReason}
                      showIcon
                    />
                  ) : null}
                </>
              )}
            </Space>
          </SectionCard>
          <SectionCard title="通知预览" extra={<Tag>测试通知</Tag>}>
            {previewQuery.isFetching ? (
              <Spin aria-label="加载通知预览" />
            ) : previewQuery.isError ? (
              <Alert
                type="error"
                title={failureMessage(previewQuery.error, '加载通知预览失败')}
                action={
                  <Button onClick={() => void previewQuery.refetch()}>
                    重试预览
                  </Button>
                }
                showIcon
              />
            ) : preview ? (
              <>
                <p
                  style={{
                    margin: '0 0 12px',
                    color: token.colorTextSecondary,
                  }}
                >
                  网页为样式示意，以钉钉实际显示为准。有按钮的消息使用
                  ActionCard；验证码仅展示正文。
                </p>
                <section
                  aria-label="消息卡片预览"
                  style={{
                    width: '100%',
                    maxWidth: 420,
                    overflowWrap: 'anywhere',
                    border: `1px solid ${token.colorBorderSecondary}`,
                    borderRadius: token.borderRadiusLG,
                    background: token.colorBgContainer,
                    marginBottom: 16,
                  }}
                >
                  <div style={{ padding: '14px 16px 6px' }}>
                    <div style={{ marginBottom: 10 }}>
                      <Tag color="blue">测试通知</Tag>
                    </div>
                    <section aria-label="Markdown预览">
                      <Markdown
                        skipHtml
                        disallowedElements={['img', 'a']}
                        unwrapDisallowed
                        urlTransform={() => ''}
                        components={{
                          h1: ({ children }) => (
                            <h1
                              style={{
                                fontSize: 16,
                                margin: '0 0 10px',
                                fontWeight: 600,
                              }}
                            >
                              {children}
                            </h1>
                          ),
                          h2: ({ children }) => (
                            <h2
                              style={{
                                fontSize: 16,
                                margin: '0 0 10px',
                                fontWeight: 600,
                              }}
                            >
                              {children}
                            </h2>
                          ),
                          h3: ({ children }) => (
                            <h3
                              style={{
                                fontSize: 16,
                                margin: '0 0 10px',
                                fontWeight: 600,
                              }}
                            >
                              {children}
                            </h3>
                          ),
                          p: ({ children }) => (
                            <p
                              style={{
                                fontSize: 14,
                                lineHeight: 1.65,
                                margin: '0 0 10px',
                              }}
                            >
                              {children}
                            </p>
                          ),
                          strong: ({ children }) => (
                            <strong style={{ fontWeight: 600 }}>
                              {children}
                            </strong>
                          ),
                          blockquote: ({ children }) => (
                            <blockquote
                              style={{
                                margin: '0 0 10px',
                                paddingLeft: 10,
                                borderLeft: `2px solid ${token.colorBorderSecondary}`,
                                color: token.colorTextSecondary,
                              }}
                            >
                              {children}
                            </blockquote>
                          ),
                        }}
                      >
                        {preview.markdown}
                      </Markdown>
                    </section>
                  </div>
                  {actionUrl && (
                    <div
                      style={{
                        borderTop: `1px solid ${token.colorBorderSecondary}`,
                        padding: '10px 16px',
                      }}
                    >
                      <Button
                        type="primary"
                        block
                        href={actionUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {preview.actionTitle}
                      </Button>
                    </div>
                  )}
                </section>
                {actionUrl && !orderId && (
                  <p style={{ color: token.colorTextSecondary }}>
                    示例按钮仅打开系统首页，不指向真实订单。
                  </p>
                )}
                {invalidAction && (
                  <Alert
                    type="error"
                    title="通知按钮目标无效，无法发送测试，请重新加载预览"
                    action={
                      <Button onClick={() => void previewQuery.refetch()}>
                        重新加载预览
                      </Button>
                    }
                    showIcon
                  />
                )}
                <SectionCard title="查看消息内容" collapsible defaultCollapsed>
                  <Descriptions
                    size="small"
                    column={1}
                    items={[
                      {
                        key: 'title',
                        label: '会话列表标题',
                        children: preview.title,
                      },
                      {
                        key: 'action-title',
                        label: '按钮文案',
                        children: preview.actionTitle || '无按钮',
                      },
                      {
                        key: 'action-url',
                        label: '按钮目标',
                        children: (
                          <span style={{ overflowWrap: 'anywhere' }}>
                            {preview.actionUrl || '无跳转'}
                          </span>
                        ),
                      },
                    ]}
                  />
                  <p style={{ margin: '12px 0 6px' }}>Markdown 正文</p>
                  <pre
                    style={{
                      whiteSpace: 'pre-wrap',
                      overflowWrap: 'anywhere',
                      margin: 0,
                    }}
                  >
                    {preview.markdown}
                  </pre>
                </SectionCard>
              </>
            ) : null}
          </SectionCard>
          <SectionCard title="发送测试">
            <Space orientation="vertical" size={12} style={{ width: '100%' }}>
              <Button
                type="primary"
                icon={<SendOutlined aria-hidden="true" />}
                loading={pending}
                disabled={!canSend}
                onClick={() => {
                  if (
                    canSend &&
                    notificationType &&
                    queryClient.isMutating({
                      mutationKey: [queryPrefix, 'send'],
                    }) === 0
                  ) {
                    sendMutation.mutate({ notificationType, orderId });
                  }
                }}
              >
                发送给我测试
              </Button>
              {sendMutation.isError && (
                <Alert
                  type="error"
                  showIcon
                  title={failureMessage(
                    sendMutation.error,
                    sendUnconfirmedMessage,
                  )}
                  description="发送结果尚未确认，请先在钉钉客户端核对，再决定是否重试。"
                />
              )}
              {sendMutation.isSuccess && (
                <Alert
                  type="success"
                  showIcon
                  title="钉钉已受理，请在客户端确认"
                />
              )}
            </Space>
          </SectionCard>
        </>
      )}
    </PageContainer>
  );
}
