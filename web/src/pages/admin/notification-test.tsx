import { SendOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Descriptions, Select, Space, Spin, Tag } from 'antd';
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

const queryPrefix = 'notification-tests';
const sendUnconfirmedMessage =
  '未确认钉钉受理测试通知，请先在客户端确认，再决定是否重试';

function failureMessage(error: unknown, fallback: string) {
  return getRequestErrorMessage(error, fallback);
}

/** 只允许当前预览中由服务端返回的本站费用入口；不开放其他 Markdown 链接或图片。 */
function allowedOrderLink(href: string, orderUrl?: string) {
  if (!orderUrl || href !== orderUrl) return '';
  try {
    const url = new URL(href);
    return /^https?:$/.test(url.protocol) &&
      /^\/orders\/sea-export\/[0-9a-f-]+\/fees$/.test(url.pathname) &&
      !url.search &&
      !url.hash &&
      !url.username &&
      !url.password
      ? href
      : '';
  } catch {
    return '';
  }
}

export default function NotificationTestPage() {
  const access = useAccess();
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
  const canSend = Boolean(
    access.canSendNotificationTests &&
      catalog?.canSend &&
      catalog.recipientBound &&
      catalog.integrationEnabled &&
      preview &&
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
                      {!orderId && <p>本次使用示例数据，不包含订单链接。</p>}
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
                <p>
                  <strong>消息标题：</strong>
                  {preview.title}
                </p>
                <section
                  aria-label="Markdown预览"
                  style={{ overflowWrap: 'anywhere' }}
                >
                  <Markdown
                    skipHtml
                    disallowedElements={['img']}
                    urlTransform={(url) =>
                      allowedOrderLink(url, preview.orderUrl)
                    }
                    components={{
                      h3: ({ children }) => (
                        <h3
                          style={{
                            fontSize: 17,
                            fontWeight: 600,
                            margin: '0 0 16px',
                          }}
                        >
                          {children}
                        </h3>
                      ),
                      p: ({ children }) => (
                        <p style={{ lineHeight: 1.8, margin: '0 0 12px' }}>
                          {children}
                        </p>
                      ),
                      strong: ({ children }) => (
                        <strong style={{ fontWeight: 600 }}>{children}</strong>
                      ),
                      a: ({ href, children }) =>
                        href ? (
                          <a
                            href={href}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            {children}
                          </a>
                        ) : (
                          <span>{children}</span>
                        ),
                    }}
                  >
                    {preview.markdown}
                  </Markdown>
                </section>
                <SectionCard
                  title="查看 Markdown 源码"
                  collapsible
                  defaultCollapsed
                >
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
